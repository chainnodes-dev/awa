package designer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/pkg/asmtypes"
	"gopkg.in/yaml.v3"
)

// PipelineRequest is the shared input for every step of the multi-step generator.
type PipelineRequest struct {
	// ProcessDescription is the original plain-language business process description.
	// Carried through every step so the LLM always has full context.
	ProcessDescription string `json:"process_description"`
	// CurrentYAML is the YAML produced by the previous step.
	// Empty only for Step 1 (Decompose).
	CurrentYAML string `json:"current_yaml,omitempty"`
	// Provider optionally overrides the default LLM provider.
	Provider string `json:"provider,omitempty"`
	// MCPServers optionally restricts which MCP servers are injected into the prompt.
	MCPServers []string `json:"mcp_servers,omitempty"`
}

// PipelineStepResult is the output of any pipeline step.
type PipelineStepResult struct {
	YAML         string                `json:"yaml"`
	Definition   *asmtypes.WorkflowDef `json:"definition"`
	Interactions []llm.Message         `json:"interactions"`
	// ValidationError is populated when YAML parses but fails semantic validation.
	// Steps 1–3 tolerate this because the workflow is not yet runnable by design.
	ValidationError string `json:"validation_error,omitempty"`
}

// MarshalPipelineStepResult serialises a PipelineStepResult to the standard API shape.
func MarshalPipelineStepResult(r *PipelineStepResult) (map[string]interface{}, error) {
	defBytes, err := json.Marshal(r.Definition)
	if err != nil {
		return nil, err
	}
	var defMap interface{}
	_ = json.Unmarshal(defBytes, &defMap)
	out := map[string]interface{}{
		"yaml":         r.YAML,
		"definition":   defMap,
		"interactions": r.Interactions,
	}
	if r.ValidationError != "" {
		out["validation_error"] = r.ValidationError
	}
	return out, nil
}

// codeJobResult holds the outcome of a parallel Codegen call.
type codeJobResult struct {
	stateName string
	result    *CodegenResult
	err       error
}

// scriptJobResult holds the outcome of a parallel Scaffold call.
type scriptJobResult struct {
	stateName string
	result    *ScaffoldResult
	err       error
}

// ── Step 1: Decompose ────────────────────────────────────────────────────────

// DecomposeWorkflow identifies the states/nodes of a process from a plain-language
// description.  All states are typed as 'intermediate' so the canvas can render
// them immediately.  No transitions, code, or expressions are generated yet.
func (g *Generator) DecomposeWorkflow(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	system := g.prompts.Get(PromptIDDecompose, DefaultDecomposePrompt())
	system += g.buildMCPSection(req.MCPServers)
	system += pipelineOutputFormat()

	userMsg := fmt.Sprintf("Process description:\n\n%s", req.ProcessDescription)
	return g.runPipelineStep(ctx, req, system, userMsg, true)
}

// ── Step 2: Categorise ───────────────────────────────────────────────────────

// CategoriseNodes assigns the correct Phaxa node type to every state produced by
// DecomposeWorkflow.  Placeholder implementations are injected for code/script
// nodes so the YAML remains loadable throughout the pipeline.
func (g *Generator) CategoriseNodes(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	if req.CurrentYAML == "" {
		return nil, fmt.Errorf("current_yaml is required for the categorise step")
	}
	system := g.prompts.Get(PromptIDCategorise, DefaultCategorisePrompt())
	system += g.buildMCPSection(req.MCPServers)
	system += pipelineOutputFormat()

	userMsg := fmt.Sprintf(
		"Process description:\n\n%s\n\nSkeleton workflow YAML (all nodes currently typed as 'intermediate'):\n\n```yaml\n%s\n```",
		req.ProcessDescription, req.CurrentYAML,
	)
	return g.runPipelineStep(ctx, req, system, userMsg, true)
}

// ── Step 3: Wire ─────────────────────────────────────────────────────────────

// WireWorkflow defines all transitions and the blackboard schema.  Node
// implementations remain as placeholders — this step produces the shared data
// contract that Step 4's code generators depend on.
func (g *Generator) WireWorkflow(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	if req.CurrentYAML == "" {
		return nil, fmt.Errorf("current_yaml is required for the wire step")
	}
	system := g.prompts.Get(PromptIDWire, DefaultWirePrompt())
	system += g.buildMCPSection(req.MCPServers)
	system += pipelineOutputFormat()

	userMsg := fmt.Sprintf(
		"Process description:\n\n%s\n\nCategorised workflow YAML (types assigned, needs transitions and blackboard schema):\n\n```yaml\n%s\n```",
		req.ProcessDescription, req.CurrentYAML,
	)
	return g.runPipelineStep(ctx, req, system, userMsg, true)
}

// ── Step 4: Implement ────────────────────────────────────────────────────────

// ImplementNodes generates the final implementation for every node that needs it.
//
// Strategy:
//  1. Parse the wired YAML to extract BB schema and per-state trigger lists.
//  2. Fan out parallel Codegen() calls for code nodes and Scaffold() calls for
//     script nodes (both have dedicated specialised prompts).
//  3. Patch the generated code back into the YAML.
//  4. Run one final LLM pass to implement intermediate, hitl, wait, and
//     emit_event nodes and produce a complete, runnable workflow.
func (g *Generator) ImplementNodes(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	if req.CurrentYAML == "" {
		return nil, fmt.Errorf("current_yaml is required for the implement step")
	}

	def, _, parseErr := asmtypes.LoadFromYAML([]byte(req.CurrentYAML))
	if parseErr != nil || def == nil {
		return g.implementFallback(ctx, req)
	}

	bbSchema := buildBBSchemaMap(def)
	triggersPerState := buildTriggersPerState(def)

	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		codeJobs   []codeJobResult
		scriptJobs []scriptJobResult
	)

	for _, s := range def.States {
		switch s.Type {
		case asmtypes.StateCode:
			if isPlaceholderCode(s) {
				wg.Add(1)
				go func(state asmtypes.StateDef) {
					defer wg.Done()
					res, err := g.Codegen(ctx, CodegenRequest{
						Instructions:  state.Instructions,
						StateName:     state.Name,
						ValidTriggers: triggersPerState[state.Name],
						BBSchema:      bbSchema,
					})
					mu.Lock()
					codeJobs = append(codeJobs, codeJobResult{state.Name, res, err})
					mu.Unlock()
				}(s)
			}
		case asmtypes.StateScript:
			if isPlaceholderScript(s) {
				wg.Add(1)
				go func(state asmtypes.StateDef) {
					defer wg.Done()
					res, err := g.Scaffold(ctx, ScaffoldRequest{
						StateName:   state.Name,
						Description: state.Instructions,
						Triggers:    triggersPerState[state.Name],
						BBSchema:    bbSchema,
					})
					mu.Lock()
					scriptJobs = append(scriptJobs, scriptJobResult{state.Name, res, err})
					mu.Unlock()
				}(s)
			}
		}
	}
	wg.Wait()

	patchedYAML, patchErr := patchImplementations(req.CurrentYAML, codeJobs, scriptJobs)
	if patchErr != nil {
		patchedYAML = req.CurrentYAML
	}

	finishSystem := g.prompts.Get(PromptIDImplFinish, DefaultImplementFinishPrompt())
	finishSystem += g.buildMCPSection(req.MCPServers)
	finishSystem += pipelineOutputFormat()
	finishMsg := fmt.Sprintf(
		"Original process description:\n\n%s\n\nWorkflow with code and script nodes already implemented (remaining nodes need finalising):\n\n```yaml\n%s\n```",
		req.ProcessDescription, patchedYAML,
	)
	return g.runPipelineStep(ctx, req, finishSystem, finishMsg, false)
}

// ── Internal helpers ─────────────────────────────────────────────────────────

// runPipelineStep performs a single LLM call and extracts a YAML block from the response.
// When softValidate is true, validation failures populate ValidationError but are not fatal.
func (g *Generator) runPipelineStep(ctx context.Context, req PipelineRequest, system, userMsg string, softValidate bool) (*PipelineStepResult, error) {
	interaction := []llm.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: userMsg},
	}

	messages := []llm.Message{
		{Role: "user", Content: userMsg},
	}

	maxRetries := 3
	var lastYamlStr string
	var lastDef *asmtypes.WorkflowDef
	var lastValErr error
	var lastExtractErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
			SystemPrompt: system,
			Messages:     messages,
			MaxTokens:    g.provider.MaxOutputTokens(),
		})
		if err != nil {
			return &PipelineStepResult{Interactions: interaction}, fmt.Errorf("LLM call failed: %w", err)
		}

		interaction = append(interaction, llm.Message{Role: "assistant", Content: resp.Content})
		messages = append(messages, llm.Message{Role: "assistant", Content: resp.Content})

		yamlStr, err := extractYAMLBlock(resp.Content)
		if err != nil {
			lastExtractErr = err
			failMsg := fmt.Sprintf("No YAML block found in your response: %v. Please output ONLY a valid YAML block.", err)
			interaction = append(interaction, llm.Message{Role: "user", Content: failMsg})
			messages = append(messages, llm.Message{Role: "user", Content: failMsg})
			continue
		}

		// SELF-HEALING: AI often produces 'script: "string"' instead of 'script: {trigger: "string"}'
		// or 'code: "string"' instead of 'code: {code: "string"}'. We fix these common shortcuts here.
		yamlStr = repairNodeStructures(yamlStr)
		lastYamlStr = yamlStr

		def, _, valErr := asmtypes.LoadFromYAML([]byte(yamlStr))
		if valErr != nil {
			lastValErr = valErr
			errorDetail := valErr.Error()
			// Provide specific guidance for common formatting hallucinations
			if strings.Contains(errorDetail, "cannot start any token") || strings.Contains(errorDetail, "found character") {
				errorDetail = "You likely used TAB characters for indentation. YAML REQUIRES SPACES. Please rewrite using ONLY SPACES (2 per level)."
			}

			failMsg := fmt.Sprintf("The generated YAML is invalid and failed to parse: %v. Please fix the formatting errors and ensure you strictly follow the schema rules.", errorDetail)
			interaction = append(interaction, llm.Message{Role: "user", Content: failMsg})
			messages = append(messages, llm.Message{Role: "user", Content: failMsg})
			continue
		}

		lastDef = def
		lastValErr = nil
		break // Success!
	}

	// If we still have an extraction error after all retries and no YAML at all
	if lastExtractErr != nil && lastYamlStr == "" {
		return &PipelineStepResult{Interactions: interaction}, fmt.Errorf("no YAML block in response after %d attempts: %w", maxRetries, lastExtractErr)
	}

	validationError := ""
	if lastValErr != nil {
		if !softValidate {
			return &PipelineStepResult{YAML: lastYamlStr, Interactions: interaction},
				fmt.Errorf("YAML validation failed after %d attempts: %w", maxRetries, lastValErr)
		}
		validationError = lastValErr.Error()
	}

	// PERSISTENCE: Ensure the process description is carried through every step.
	if lastDef != nil {
		lastDef.Metadata.ProcessDescription = req.ProcessDescription
		// Update the YAML string to include the persisted description
		if out, err := yaml.Marshal(lastDef); err == nil {
			lastYamlStr = string(out)
		}
	}

	return &PipelineStepResult{
		YAML:            lastYamlStr,
		Definition:      lastDef,
		Interactions:    interaction,
		ValidationError: validationError,
	}, nil
}

// repairNodeStructures uses simple string/regex replacement to fix common AI YAML shortcuts
// that violate the strict Phaxa schema.
func repairNodeStructures(y string) string {
	y = sanitizeYAML(y)
	lines := strings.Split(y, "\n")
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Fix 'script: "expr"' -> 'script: { trigger: "expr" }'
		if strings.HasPrefix(trimmed, "script:") && !strings.Contains(trimmed, "{") && len(trimmed) > 8 {
			val := strings.TrimSpace(trimmed[7:])
			if val != "" && val != "{}" {
				indent := line[:strings.Index(line, "script:")]
				out = append(out, fmt.Sprintf("%sscript: { trigger: %s }", indent, val))
				continue
			}
		}
		// Fix 'code: "body"' -> 'code: { code: "body" }'
		if strings.HasPrefix(trimmed, "code:") && !strings.Contains(trimmed, "{") && len(trimmed) > 6 {
			val := strings.TrimSpace(trimmed[5:])
			if val != "" && val != "{}" {
				indent := line[:strings.Index(line, "code:")]
				out = append(out, fmt.Sprintf("%scode: { code: %s }", indent, val))
				continue
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// implementFallback runs a single refinement LLM call when the wired YAML cannot
// be structurally parsed for the parallel code-generation fanout.
func (g *Generator) implementFallback(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	system := g.buildSystemPrompt(req.MCPServers) + g.refinementAddendum()
	userMsg := fmt.Sprintf(
		"Here is the current workflow YAML:\n\n```yaml\n%s\n```\n\nProcess context: %s\n\nPlease apply the following change:\n\nImplement all placeholder nodes. Replace any TODO code with working JavaScript. Replace placeholder script triggers with real expr-lang expressions. Finalise hitl assignees, wait conditions, and emit_event definitions. Produce a complete runnable workflow.",
		req.CurrentYAML, req.ProcessDescription,
	)
	return g.runPipelineStep(ctx, req, system, userMsg, false)
}

// buildMCPSection builds the MCP server context block for a pipeline step prompt.
func (g *Generator) buildMCPSection(requestedServers []string) string {
	relevantServers := g.registry
	if len(requestedServers) > 0 {
		relevantServers = filterServers(g.registry, requestedServers)
	}
	if len(relevantServers) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n## Available MCP Servers\n\n")
	for _, s := range relevantServers {
		sb.WriteString(fmt.Sprintf("- **%s** (`{{ env.%s }}`): %s\n", s.Name, s.EnvVar, s.Description))
	}
	return sb.String()
}

// buildBBSchemaMap extracts a flat fieldName→type map from the workflow definition.
func buildBBSchemaMap(def *asmtypes.WorkflowDef) map[string]string {
	out := make(map[string]string, len(def.Blackboard.Schema))
	for name, fd := range def.Blackboard.Schema {
		out[name] = fd.Type
	}
	return out
}

// buildTriggersPerState returns outgoing trigger names indexed by state name.
func buildTriggersPerState(def *asmtypes.WorkflowDef) map[string][]string {
	out := make(map[string][]string)
	for _, t := range def.Transitions {
		out[t.From] = append(out[t.From], t.Trigger)
	}
	return out
}

// isPlaceholderCode returns true when a code node still has a placeholder body.
func isPlaceholderCode(s asmtypes.StateDef) bool {
	if s.Code == nil {
		return true
	}
	code := strings.TrimSpace(s.Code.Code)
	return code == "" ||
		strings.Contains(code, "// TODO: implement") ||
		code == "return {trigger: 'done'}" ||
		code == "return { trigger: 'done' }"
}

// isPlaceholderScript returns true when a script node still has a placeholder trigger.
func isPlaceholderScript(s asmtypes.StateDef) bool {
	if s.Script == nil {
		return true
	}
	t := strings.TrimSpace(s.Script.Trigger)
	return t == "" || t == `"done"` || t == "'done'"
}

// patchImplementations merges parallel code/script generation results into the YAML
// by round-tripping through a generic yaml map.
func patchImplementations(yamlStr string, codeJobs []codeJobResult, scriptJobs []scriptJobResult) (string, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &doc); err != nil {
		return yamlStr, fmt.Errorf("parse YAML for patching: %w", err)
	}

	statesRaw, ok := doc["states"]
	if !ok {
		return yamlStr, nil
	}
	states, ok := statesRaw.([]interface{})
	if !ok {
		return yamlStr, nil
	}

	codeByName := make(map[string]*CodegenResult)
	for _, j := range codeJobs {
		if j.err == nil && j.result != nil {
			codeByName[j.stateName] = j.result
		}
	}
	scriptByName := make(map[string]*ScaffoldResult)
	for _, j := range scriptJobs {
		if j.err == nil && j.result != nil {
			scriptByName[j.stateName] = j.result
		}
	}

	for _, raw := range states {
		state, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := state["name"].(string)
		if res, ok := codeByName[name]; ok {
			// Ensure we always produce the nested {code: {code: "..."}} structure
			state["code"] = map[string]interface{}{"code": res.Code}
		}
		if res, ok := scriptByName[name]; ok {
			// Ensure we always produce the nested {script: {trigger: "...", updates: [...]}} structure
			script := map[string]interface{}{"trigger": res.Trigger}
			if len(res.Updates) > 0 {
				script["updates"] = res.Updates
			}
			state["script"] = script
		}
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return yamlStr, fmt.Errorf("re-marshal patched YAML: %w", err)
	}
	return string(out), nil
}

func pipelineOutputFormat() string {
	return `
## Output Format

Respond with ONLY the workflow YAML, wrapped in a ` + "```yaml" + ` code block.
Do not include any explanation before or after the code block.
The YAML must be complete and valid according to the Phaxa schema.
`
}

