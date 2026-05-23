package designer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/pkg/asmtypes"
	"gopkg.in/yaml.v3"
)

// PipelineRequest is the shared input for every step of the multi-step generator.
type PipelineRequest struct {
	ProcessDescription string   `json:"process_description"`
	CurrentYAML        string   `json:"current_yaml,omitempty"`
	Provider           string   `json:"provider,omitempty"`
	MCPServers         []string `json:"mcp_servers,omitempty"`
}

// PipelineStepResult is the output of any pipeline step.
type PipelineStepResult struct {
	YAML            string                `json:"yaml"`
	Definition      *asmtypes.WorkflowDef `json:"definition"`
	Interactions    []llm.Message         `json:"interactions"`
	ValidationError string                `json:"validation_error,omitempty"`
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

type codeJobResult struct {
	stateName string
	result    *CodegenResult
	err       error
}

type scriptJobResult struct {
	stateName string
	result    *ScaffoldResult
	err       error
}

// ── Step 1: Decompose ────────────────────────────────────────────────────────

func (g *Generator) DecomposeWorkflow(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	system := g.assemblePipelineSystemPrompt(ctx, PromptIDDecompose, DefaultDecomposePrompt(), req.MCPServers)
	userMsg := fmt.Sprintf("Process description:\n\n%s", req.ProcessDescription)

	type step1Response struct {
		Name                string `json:"name"`
		Abstract            string `json:"abstract"`
		DetailedDescription string `json:"detailed_description"`
		States              []struct {
			Name                  string `json:"name"`
			Instructions          string `json:"instructions"`
			TechnicalRequirements string `json:"technical_requirements"`
			DataContract          struct {
				Reads []struct {
					Name            string `json:"name"`
					Type            string `json:"type"`
					Description     string `json:"description"`
					IsWorkflowInput bool   `json:"is_workflow_input"`
				} `json:"reads"`
				Writes []struct {
					Name             string `json:"name"`
					Type             string `json:"type"`
					Description      string `json:"description"`
					IsWorkflowOutput bool   `json:"is_workflow_output"`
				} `json:"writes"`
			} `json:"data_contract"`
		} `json:"states"`
		Transitions []struct {
			From    string `json:"from"`
			To      string `json:"to"`
			Trigger string `json:"trigger"`
		} `json:"transitions"`
	}

	return g.runJSONPipelineStep(ctx, req, system, userMsg, func(e *WorkflowEditor, b []byte) error {
		var resp step1Response
		if err := json.Unmarshal(b, &resp); err != nil {
			return err
		}

		e.SetMetadata(resp.Name, resp.Abstract, resp.DetailedDescription)

		bbSchema := make(map[string]asmtypes.FieldDef)
		inputsMap := make(map[string]asmtypes.PortDef)
		outputsMap := make(map[string]asmtypes.PortDef)

		for i, s := range resp.States {
			stype := asmtypes.StatePrompt
			if i == 0 {
				stype = asmtypes.StateInitial
			} else if i == len(resp.States)-1 && len(resp.States) > 1 {
				stype = asmtypes.StateTerminal
			}
			e.AddState(s.Name, stype, s.Instructions, s.TechnicalRequirements)

			// Process Data Contract
			for _, r := range s.DataContract.Reads {
				if _, exists := bbSchema[r.Name]; !exists {
					bbSchema[r.Name] = asmtypes.FieldDef{Type: r.Type, Required: r.IsWorkflowInput}
				}
				if r.IsWorkflowInput {
					inputsMap[r.Name] = asmtypes.PortDef{Name: r.Name, Type: r.Type, Description: r.Description, Required: true}
				}
			}
			for _, w := range s.DataContract.Writes {
				if _, exists := bbSchema[w.Name]; !exists {
					bbSchema[w.Name] = asmtypes.FieldDef{Type: w.Type, IsOutput: w.IsWorkflowOutput}
				}
				if w.IsWorkflowOutput {
					outputsMap[w.Name] = asmtypes.PortDef{Name: w.Name, Type: w.Type, Description: w.Description}
				}
			}
		}

		var inputs []asmtypes.PortDef
		for _, p := range inputsMap {
			inputs = append(inputs, p)
		}
		var outputs []asmtypes.PortDef
		for _, p := range outputsMap {
			outputs = append(outputs, p)
		}

		e.SetBlackboardSchema(bbSchema)
		e.SetPorts(inputs, outputs)

		for _, t := range resp.Transitions {
			e.AddTransition(t.From, t.To, t.Trigger, "")
		}
		return nil
	})
}

// ── Step 2: Categorise ───────────────────────────────────────────────────────

func (g *Generator) CategoriseNodes(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	if req.CurrentYAML == "" {
		return nil, fmt.Errorf("current_yaml is required for the categorise step")
	}
	system := g.assemblePipelineSystemPrompt(ctx, PromptIDCategorise, DefaultCategorisePrompt(), req.MCPServers)
	userMsg := fmt.Sprintf(
		"Process description:\n\n%s\n\nSkeleton workflow YAML:\n\n```yaml\n%s\n```",
		req.ProcessDescription, req.CurrentYAML,
	)

	type step2Response struct {
		States []struct {
			Name       string   `json:"name"`
			Type       string   `json:"type"`
			AgentName  string   `json:"agent_name"`
			MCPServers []string `json:"mcp_servers"`
		} `json:"states"`
	}

	res, err := g.runJSONPipelineStep(ctx, req, system, userMsg, func(e *WorkflowEditor, b []byte) error {
		var resp step2Response
		if err := json.Unmarshal(b, &resp); err != nil {
			return err
		}
		for _, s := range resp.States {
			if s.Type == "" {
				continue
			}
			e.UpdateState(s.Name, asmtypes.StateType(s.Type), "", "")
			switch s.Type {
			case "prompt":
				agentName := s.AgentName
				if agentName == "" {
					agentName = "primary-agent"
				}
				
				// Ensure MCP servers from the request are assigned if the LLM didn't provide specific ones
				mcpServers := s.MCPServers
				if len(mcpServers) == 0 && len(req.MCPServers) > 0 {
					mcpServers = req.MCPServers
				}
				
				e.SetStateAgent(s.Name, agentName, mcpServers)
			case "code":
				e.SetStateCode(s.Name, "// TODO: implemented in background")
			case "script":
				e.SetStateScript(s.Name, "'done'", nil)
			}
		}
		hasInitial := false
		hasTerminal := false
		for _, s := range e.Def.States {
			if s.Type == asmtypes.StateInitial {
				hasInitial = true
			}
			if s.Type == asmtypes.StateTerminal {
				hasTerminal = true
			}
		}
		if !hasInitial && len(e.Def.States) > 0 {
			e.Def.States[0].Type = asmtypes.StateInitial
		}
		if !hasTerminal && len(e.Def.States) > 1 {
			e.Def.States[len(e.Def.States)-1].Type = asmtypes.StateTerminal
		}

		return nil
	})

	if err != nil {
		return res, err
	}

	// Post-processing: Auto-generate code/scripts for algorithmic nodes
	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		codeJobs   []codeJobResult
		scriptJobs []scriptJobResult
	)

	for _, s := range res.Definition.States {
		switch s.Type {
		case asmtypes.StateCode:
			wg.Add(1)
			go func(state asmtypes.StateDef) {
				defer wg.Done()
				triggers := getValidTriggers(res.Definition, state.Name)
				codegenRes, cErr := g.Codegen(ctx, CodegenRequest{
					Instructions:  state.TechnicalRequirements + "\n" + state.Instructions,
					StateName:     state.Name,
					ValidTriggers: triggers,
					BBSchema:      buildBBSchemaMap(res.Definition),
				})
				mu.Lock()
				codeJobs = append(codeJobs, codeJobResult{stateName: state.Name, result: codegenRes, err: cErr})
				mu.Unlock()
			}(s)
		case asmtypes.StateScript:
			wg.Add(1)
			go func(state asmtypes.StateDef) {
				defer wg.Done()
				triggers := getValidTriggers(res.Definition, state.Name)
				scaffoldRes, sErr := g.Scaffold(ctx, ScaffoldRequest{
					AgentName:   "system",
					StateName:   state.Name,
					Description: state.TechnicalRequirements + "\n" + state.Instructions,
					Triggers:    triggers,
					BBSchema:    buildBBSchemaMap(res.Definition),
				})
				mu.Lock()
				scriptJobs = append(scriptJobs, scriptJobResult{stateName: state.Name, result: scaffoldRes, err: sErr})
				mu.Unlock()
			}(s)
		}
	}

	wg.Wait()

	// Apply generated code
	editor := NewWorkflowEditor(res.Definition)
	for _, job := range codeJobs {
		if job.err == nil && job.result != nil {
			editor.SetStateCode(job.stateName, job.result.Code)
		}
	}
	for _, job := range scriptJobs {
		if job.err == nil && job.result != nil {
			editor.SetStateScript(job.stateName, job.result.Trigger, job.result.Updates)
		}
	}

	outYAML, _ := yaml.Marshal(editor.Def)
	res.YAML = string(outYAML)
	return res, nil
}

// ── Step 3: Wire ─────────────────────────────────────────────────────────────

func (g *Generator) WireWorkflow(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	if req.CurrentYAML == "" {
		return nil, fmt.Errorf("current_yaml is required for the wire step")
	}
	system := g.assemblePipelineSystemPrompt(ctx, PromptIDWire, DefaultWirePrompt(), req.MCPServers)
	userMsg := fmt.Sprintf(
		"Process description:\n\n%s\n\nCategorised workflow YAML:\n\n```yaml\n%s\n```",
		req.ProcessDescription, req.CurrentYAML,
	)

	type step3Response struct {
		Transitions []struct {
			From    string `json:"from"`
			To      string `json:"to"`
			Trigger string `json:"trigger"`
			Guard   string `json:"guard"`
		} `json:"transitions"`
	}

	return g.runJSONPipelineStep(ctx, req, system, userMsg, func(e *WorkflowEditor, b []byte) error {
		var resp step3Response
		if err := json.Unmarshal(b, &resp); err != nil {
			return err
		}
		for _, t := range resp.Transitions {
			e.AddTransition(t.From, t.To, t.Trigger, t.Guard)
		}
		return nil
	})
}

// ── Step 4: Implement ────────────────────────────────────────────────────────

func (g *Generator) ImplementNodes(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	if req.CurrentYAML == "" {
		return nil, fmt.Errorf("current_yaml is required for the implement step")
	}
	system := g.assemblePipelineSystemPrompt(ctx, PromptIDImplFinish, DefaultImplementFinishPrompt(), req.MCPServers)
	userMsg := fmt.Sprintf(
		"Process description:\n\n%s\n\nWired workflow YAML:\n\n```yaml\n%s\n```",
		req.ProcessDescription, req.CurrentYAML,
	)

	type step4Response struct {
		Metadata struct {
			Name               string `json:"name"`
			Abstract           string `json:"abstract"`
			DetailedDescription string `json:"detailed_description"`
		} `json:"metadata"`
		BlackboardSchema map[string]asmtypes.FieldDef `json:"blackboard_schema"`
		Inputs               []asmtypes.PortDef           `json:"inputs"`
		Outputs              []asmtypes.PortDef           `json:"outputs"`
		WorkflowCapabilities []string                    `json:"workflow_capabilities"`
	}

	return g.runJSONPipelineStep(ctx, req, system, userMsg, func(e *WorkflowEditor, b []byte) error {
		var resp step4Response
		if err := json.Unmarshal(b, &resp); err != nil {
			return err
		}
		e.SetMetadata(resp.Metadata.Name, resp.Metadata.Abstract, resp.Metadata.DetailedDescription)
		e.SetBlackboardSchema(resp.BlackboardSchema)
		e.SetPorts(resp.Inputs, resp.Outputs)
		if len(resp.WorkflowCapabilities) > 0 {
			e.SetCapabilities(resp.WorkflowCapabilities)
		} else if len(req.MCPServers) > 0 {
			// Fallback: if user requested specific servers but LLM didn't put them in capabilities,
			// add them now to ensure they are available to agents.
			e.SetCapabilities(req.MCPServers)
		}
		return nil
	})
}

// ── Internal helpers ─────────────────────────────────────────────────────────

func (g *Generator) runJSONPipelineStep(ctx context.Context, req PipelineRequest, system, userMsg string, applyFunc func(e *WorkflowEditor, b []byte) error) (*PipelineStepResult, error) {
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
		// Initialize editor on every attempt to ensure clean slate if previous attempt partially modified it
		var def *asmtypes.WorkflowDef
		if req.CurrentYAML != "" {
			parsedDef, _, err := asmtypes.LoadFromYAML([]byte(req.CurrentYAML))
			if err == nil && parsedDef != nil {
				def = parsedDef
			} else {
				slog.Error("Pipeline step failed to load current YAML", "error", err, "yaml_snippet", req.CurrentYAML[:minVal(len(req.CurrentYAML), 100)])
				def = &asmtypes.WorkflowDef{}
			}
		} else {
			def = &asmtypes.WorkflowDef{}
		}
		editor := NewWorkflowEditor(def)

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

		jsonStr, err := extractJSONBlock(resp.Content)
		if err != nil {
			lastExtractErr = err
			failMsg := fmt.Sprintf("No JSON block found in your response: %v. Please output ONLY a valid JSON block matching the schema.", err)
			interaction = append(interaction, llm.Message{Role: "user", Content: failMsg})
			messages = append(messages, llm.Message{Role: "user", Content: failMsg})
			continue
		}

		// Apply the JSON changes via Editor
		if applyErr := applyFunc(editor, []byte(jsonStr)); applyErr != nil {
			lastValErr = applyErr
			failMsg := fmt.Sprintf("Failed to apply JSON parameters: %v. Please correct the parameters.", applyErr)
			interaction = append(interaction, llm.Message{Role: "user", Content: failMsg})
			messages = append(messages, llm.Message{Role: "user", Content: failMsg})
			continue
		}

		// Success! Serialize to YAML
		outYAML, err := yaml.Marshal(editor.Def)
		if err != nil {
			lastValErr = err
			continue
		}

		lastYamlStr = string(outYAML)
		lastDef = editor.Def
		lastValErr = nil
		break 
	}

	if lastExtractErr != nil && lastYamlStr == "" {
		return &PipelineStepResult{Interactions: interaction}, fmt.Errorf("no JSON block in response after %d attempts: %w", maxRetries, lastExtractErr)
	}

	if lastValErr != nil {
		return &PipelineStepResult{YAML: lastYamlStr, Interactions: interaction},
			fmt.Errorf("JSON application failed after %d attempts: %w", maxRetries, lastValErr)
	}

	if lastDef != nil {
		lastDef.Metadata.ProcessDescription = req.ProcessDescription
		if out, err := yaml.Marshal(lastDef); err == nil {
			lastYamlStr = string(out)
		}
	}

	return &PipelineStepResult{
		YAML:            lastYamlStr,
		Definition:      lastDef,
		Interactions:    interaction,
	}, nil
}

// buildMCPSection builds the MCP server context block for a pipeline step prompt.
func (g *Generator) buildMCPSection(requestedServers []string, registry []MCPServerEntry) string {
	relevantServers := registry
	if len(requestedServers) > 0 {
		relevantServers = filterServers(registry, requestedServers)
	}
	if len(relevantServers) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n## Available MCP Servers\n\n")
	for _, s := range relevantServers {
		toolsStr := ""
		if len(s.Tools) > 0 {
			toolsStr = fmt.Sprintf(" (Tools: %s)", strings.Join(s.Tools, ", "))
		}
		sb.WriteString(fmt.Sprintf("- **%s** (`{{ env.%s }}`): %s%s\n", s.Name, s.EnvVar, s.Description, toolsStr))
	}
	return sb.String()
}

// buildBBSchemaMap returns a simple map of bb field names to their types.
func buildBBSchemaMap(def *asmtypes.WorkflowDef) map[string]string {
	m := make(map[string]string)
	if def == nil || def.Blackboard.Schema == nil {
		return m
	}
	for k, v := range def.Blackboard.Schema {
		m[k] = v.Type
	}
	return m
}

// getValidTriggers returns a list of trigger names defined in transitions leaving the state.
func getValidTriggers(def *asmtypes.WorkflowDef, stateName string) []string {
	var triggers []string
	if def == nil {
		return triggers
	}
	for _, t := range def.Transitions {
		if t.From == stateName {
			triggers = append(triggers, t.Trigger)
		}
	}
	return triggers
}

func (g *Generator) assemblePipelineSystemPrompt(ctx context.Context, promptID, defaultContent string, mcpServers []string) string {
	return g.AssembleSystemPrompt(ctx, mcpServers, nil, g.prompts.Get(promptID, defaultContent))
}
func minVal(a, b int) int {
	if a < b {
		return a
	}
	return b
}
