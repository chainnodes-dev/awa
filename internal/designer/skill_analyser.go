package designer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// AnalyseSkillRequest is the input to the skill → workflow decomposer.
type AnalyseSkillRequest struct {
	// Description is the natural-language description of what the skill does.
	Description string `json:"description" binding:"required"`
	// Inputs is the typed I/O contract the caller declares.
	// When provided, generated blackboard fields MUST include all input ports.
	Inputs []asmtypes.SkillPortDef `json:"inputs,omitempty"`
	// Outputs is the typed output contract.
	// Generated terminal states MUST write all output ports to the blackboard.
	Outputs []asmtypes.SkillPortDef `json:"outputs,omitempty"`
	// MCPServers lists MCP server names available to the generated workflow.
	MCPServers []string `json:"mcp_servers,omitempty"`
	// ExistingYAML when set causes the LLM to refine rather than generate from scratch.
	ExistingYAML string `json:"existing_yaml,omitempty"`
}

// AnalyseSkillResult holds the generated workflow.
// The shape is identical to GenerateResult so the same frontend handler works.
type AnalyseSkillResult struct {
	YAML       string                `json:"yaml"`
	Definition *asmtypes.WorkflowDef `json:"definition"`
}

// AnalyseSkill decomposes a skill description into a WorkflowDef YAML.
// It classifies each step as:
//   - intermediate (LLM reasoning required)
//   - code         (programmatic transformation)
//   - script       (simple deterministic expression)
//   - hitl         (requires human input)
//   - wait         (event-driven pause)
//   - skill_call   (delegates to a registered skill in the catalog)
//
// On YAML validation failure the LLM is retried once with the error.
func (g *Generator) AnalyseSkill(ctx context.Context, req AnalyseSkillRequest) (*AnalyseSkillResult, error) {
	// Fetch the registered skill catalog so the LLM can reuse existing skills
	// instead of reimplementing their logic inline.
	var registeredSkills []*store.Skill
	if g.skillStore != nil {
		var err error
		registeredSkills, err = g.skillStore.ListSkills(ctx)
		if err != nil {
			// Non-fatal: log and proceed without registry context.
			slog.Warn("skill analyser: failed to list registered skills", "error", err)
		}
	}

	system := g.buildSkillAnalyserPrompt(req.MCPServers, registeredSkills)
	if req.ExistingYAML != "" {
		system += g.refinementAddendum()
	}

	userMsg := g.buildSkillUserMsg(req)
	messages := []llm.Message{{Role: "user", Content: userMsg}}

	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages:     messages,
		MaxTokens:    4096,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	yamlStr, def, err := g.extractAndValidate(resp.Content)
	if err != nil {
		// Retry once with error feedback.
		retryMessages := append(messages,
			llm.Message{Role: "assistant", Content: resp.Content},
			llm.Message{Role: "user", Content: fmt.Sprintf(
				"The YAML you produced is invalid:\n\n%s\n\nPlease fix all issues and return the corrected, complete YAML.",
				err.Error(),
			)},
		)
		resp2, err2 := g.provider.Complete(ctx, llm.CompletionRequest{
			SystemPrompt: system,
			Messages:     retryMessages,
			MaxTokens:    4096,
		})
		if err2 != nil {
			return nil, fmt.Errorf("LLM retry failed: %w (original: %w)", err2, err)
		}
		yamlStr, def, err = g.extractAndValidate(resp2.Content)
		if err != nil {
			return nil, fmt.Errorf("generated workflow still invalid after retry: %w", err)
		}
	}

	return &AnalyseSkillResult{YAML: yamlStr, Definition: def}, nil
}

// buildSkillAnalyserPrompt produces the system prompt for the skill decomposer.
// It reuses the standard YAML schema prompt but adds skill-specific decomposition rules
// and, when the skill catalog is non-empty, a section listing registered skills so the
// LLM can delegate to them via skill_call nodes instead of reimplementing logic inline.
func (g *Generator) buildSkillAnalyserPrompt(mcpServers []string, registeredSkills []*store.Skill) string {
	// Start from the standard schema prompt (which already documents all state types,
	// transitions, guards, MCP server syntax, and validation rules).
	base := g.buildSystemPrompt(mcpServers)

	// Prepend skill-specific decomposition instructions.
	preamble := `You are a skill decomposition engine for the ASM (Agentic State Machine) platform.

Your task is to analyse a skill description and convert it into an optimal ASM workflow.
A "skill" is a reusable capability described in natural language or code. Your job is to:

1. **Decompose** the skill into atomic, sequential (or branching) steps.
2. **Classify** each step into the most efficient state type:
   - Use **skill_call** FIRST when a registered skill in the catalog (see below) already
     implements the required capability. This is always preferred over reimplementing
     the same logic inline.
   - Use **intermediate** (LLM agent) ONLY when genuine reasoning, judgment, or language
     understanding is required that cannot be expressed deterministically and no
     registered skill covers it.
   - Use **code** (JavaScript) when the step requires computation, transformation, loops,
     or complex conditional logic that the LLM would otherwise do wastefully.
   - Use **script** (expr-lang) for simple single-expression routing or field updates
     (e.g. "if amount > 1000 route to review").
   - Use **hitl** when the step explicitly requires a human decision.
   - Use **wait** when the step must pause until external data arrives.
3. **Identify data dependencies** — what each step reads and writes on the blackboard.
   These become the blackboard schema fields.
4. **Honour the declared I/O contract** — all declared input ports MUST appear as
   blackboard fields; all declared output ports MUST be written before terminal states.
5. **Minimise LLM usage** — prefer code, script, or skill_call nodes over intermediate
   nodes wherever possible. This reduces cost, latency, and non-determinism.

Think step by step before writing the YAML. Ask yourself for each step:
"Does a registered skill already do this?" If yes, use skill_call.
"Can this be expressed without an LLM?" If yes, use code or script.

` + base

	// Inject the registered skill catalog when the registry is non-empty.
	if len(registeredSkills) > 0 {
		preamble += g.buildSkillCatalogSection(registeredSkills)
	}

	return preamble
}

// buildSkillCatalogSection renders the registered skill catalog as a prompt section.
// Each entry shows the skill name, description, and typed I/O ports so the LLM can
// decide whether to delegate to it via a skill_call node.
func (g *Generator) buildSkillCatalogSection(skills []*store.Skill) string {
	var sb strings.Builder

	sb.WriteString(`
## Registered Skills (Skill Catalog)

The following skills are already implemented and registered in the platform. When a step
in the workflow you are designing matches one of these skills, you MUST emit a **skill_call**
state that delegates to it — do NOT reimplement the same logic inline.

For each skill_call state:
- Set "skill_ref" to the skill name shown below.
- Map the parent blackboard fields to the skill's input ports via "input_mappings".
- Map the skill's output ports back to parent blackboard fields via "output_mappings".
- Choose a "completion_trigger" and optionally a "failure_trigger".

`)

	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("### %s\n", s.Name))
		if s.Description != "" {
			sb.WriteString(s.Description + "\n\n")
		}

		if len(s.Inputs) > 0 {
			sb.WriteString("**Input ports:**\n")
			for _, p := range s.Inputs {
				req := ""
				if p.Required {
					req = " *(required)*"
				}
				sb.WriteString(fmt.Sprintf("- `%s` (%s)%s", p.Name, p.Type, req))
				if p.Description != "" {
					sb.WriteString(": " + p.Description)
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}

		if len(s.Outputs) > 0 {
			sb.WriteString("**Output ports:**\n")
			for _, p := range s.Outputs {
				sb.WriteString(fmt.Sprintf("- `%s` (%s)", p.Name, p.Type))
				if p.Description != "" {
					sb.WriteString(": " + p.Description)
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// buildSkillUserMsg constructs the user message with the full skill context.
func (g *Generator) buildSkillUserMsg(req AnalyseSkillRequest) string {
	var sb strings.Builder

	if req.ExistingYAML != "" {
		sb.WriteString("Refine the following workflow to better implement the skill described below.\n\n")
		sb.WriteString("## Current Workflow\n\n```yaml\n")
		sb.WriteString(req.ExistingYAML)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("## Skill Description\n\n")
	sb.WriteString(req.Description)
	sb.WriteString("\n\n")

	if len(req.Inputs) > 0 {
		sb.WriteString("## Input Contract\n\n")
		sb.WriteString("The workflow MUST accept these inputs via the blackboard (required fields at run start):\n\n")
		for _, p := range req.Inputs {
			req := ""
			if p.Required {
				req = " (required)"
			}
			sb.WriteString(fmt.Sprintf("- `%s` (%s)%s", p.Name, p.Type, req))
			if p.Description != "" {
				sb.WriteString(": " + p.Description)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(req.Outputs) > 0 {
		sb.WriteString("## Output Contract\n\n")
		sb.WriteString("The workflow MUST write these fields to the blackboard before reaching any terminal state:\n\n")
		for _, p := range req.Outputs {
			sb.WriteString(fmt.Sprintf("- `%s` (%s)", p.Name, p.Type))
			if p.Description != "" {
				sb.WriteString(": " + p.Description)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Decompose this skill into the most efficient ASM workflow. ")
	sb.WriteString("Prefer code and script nodes over LLM agents wherever the logic is deterministic. ")
	sb.WriteString("Return only the YAML.\n")

	return sb.String()
}
