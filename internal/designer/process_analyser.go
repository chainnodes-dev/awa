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

// ProcessAnalyseRequest is the input to the process → workflow decomposer.
type ProcessAnalyseRequest struct {
	// Description is the natural-language description of what the process does.
	Description string `json:"description" binding:"required"`
	// Inputs is the typed I/O contract the caller declares.
	Inputs []asmtypes.SkillPortDef `json:"inputs,omitempty"`
	// Outputs is the typed output contract.
	Outputs []asmtypes.SkillPortDef `json:"outputs,omitempty"`
	// MCPServers lists MCP server names available to the generated workflow.
	MCPServers []string `json:"mcp_servers,omitempty"`
	// ExistingYAML when set causes the LLM to refine rather than generate from scratch.
	ExistingYAML string `json:"existing_yaml,omitempty"`
	// History is the prior conversation turns (chat).
	History []ChatMessage `json:"history,omitempty"`
	// Provider is an optional LLM provider name to use for this request, overriding
	// the default provider.
	Provider string `json:"provider,omitempty"`
}

// ProcessAnalyseResult holds the generated workflow.
type ProcessAnalyseResult struct {
	YAML         string                `json:"yaml"`
	Definition   *asmtypes.WorkflowDef `json:"definition"`
	Interactions []llm.Message         `json:"interactions"`
}

// ProcessSummariseRequest is the input to the workflow → summary generator.
type ProcessSummariseRequest struct {
	// WorkflowYAML is the raw YAML to be summarised.
	WorkflowYAML string `json:"workflow_yaml" binding:"required"`
}

// ProcessSummariseResult holds the generated summary.
type ProcessSummariseResult struct {
	Summary      string        `json:"summary"`
	Interactions []llm.Message `json:"interactions"`
}

// AnalyseProcess decomposes a process description into a WorkflowDef YAML.
// It classifies each step as:
//   - intermediate (LLM reasoning required)
//   - code         (programmatic transformation)
//   - script       (simple deterministic expression)
//   - hitl         (requires human input)
//   - wait         (event-driven pause)
//   - skill_call   (delegates to another reusable process)
//
// On YAML validation failure the LLM is retried once with the error.
func (g *Generator) AnalyseProcess(ctx context.Context, req ProcessAnalyseRequest) (*ProcessAnalyseResult, error) {
	// Fetch the catalog of reusable workflows so the LLM can leverage them
	// via skill_call nodes instead of reimplementing logic inline.
	var reusableWorkflows []*asmtypes.WorkflowDef
	if g.wfStore != nil {
		var err error
		reusableWorkflows, err = g.wfStore.ListDefinitions(ctx, store.DefinitionFilter{ReusableOnly: true})
		if err != nil {
			slog.Warn("process analyser: failed to list reusable workflows", "error", err)
		}
	}

	system := g.buildProcessAnalyserPrompt(req.MCPServers, reusableWorkflows)
	if req.ExistingYAML != "" {
		system += g.refinementAddendum()
	}

	userMsg := g.buildProcessUserMsg(req)
	messages := make([]llm.Message, 0, len(req.History)+1)
	for _, m := range req.History {
		messages = append(messages, llm.Message{Role: m.Role, Content: m.Content})
	}
	messages = append(messages, llm.Message{Role: "user", Content: userMsg})

	interaction := make([]llm.Message, 0)
	interaction = append(interaction, llm.Message{Role: "system", Content: system})
	interaction = append(interaction, messages...)

	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages:     messages,
		MaxTokens:    g.provider.MaxOutputTokens(),
	})
	if err != nil {
		return &ProcessAnalyseResult{Interactions: interaction}, fmt.Errorf("LLM call failed: %w", err)
	}

	interaction = append(interaction, llm.Message{Role: "assistant", Content: resp.Content})

	yamlStr, def, err := g.extractAndValidate(resp.Content)
	if err != nil {
		// No automatic retry.
		return &ProcessAnalyseResult{
			YAML:         yamlStr,
			Definition:   def,
			Interactions: interaction,
		}, fmt.Errorf("generated workflow is invalid: %w", err)
	}

	return &ProcessAnalyseResult{
		YAML:         yamlStr,
		Definition:   def,
		Interactions: interaction,
	}, nil
}

// SummariseProcess converts a WorkflowDef YAML into a human-readable process summary.
func (g *Generator) SummariseProcess(ctx context.Context, req ProcessSummariseRequest) (*ProcessSummariseResult, error) {
	system := "You are a process summarisation engine. Convert the provided ASM workflow YAML into a professional, human-readable summary that explains the business logic, roles involved, and expected outcomes."
	userMsg := fmt.Sprintf("Summarise this workflow:\n\n```yaml\n%s\n```", req.WorkflowYAML)

	messages := []llm.Message{
		{Role: "user", Content: userMsg},
	}

	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages:     messages,
		MaxTokens:    1024,
	})
	if err != nil {
		return nil, fmt.Errorf("summarisation failed: %w", err)
	}

	return &ProcessSummariseResult{
		Summary:      resp.Content,
		Interactions: append([]llm.Message{{Role: "system", Content: system}, {Role: "user", Content: userMsg}}, llm.Message{Role: "assistant", Content: resp.Content}),
	}, nil
}

// buildProcessAnalyserPrompt produces the system prompt for the workflow decomposer.
func (g *Generator) buildProcessAnalyserPrompt(mcpServers []string, registeredProcesses []*asmtypes.WorkflowDef) string {
	// Start from the standard schema prompt.
	base := g.buildSystemPrompt(mcpServers)

	// Prepend decomposition instructions.
	preamble := g.prompts.Get(PromptIDSkill, `You are a process decomposition engine for the ASM (Agentic State Machine) platform.

Your task is to analyse a process description and convert it into an optimal ASM workflow.
A "process" is a workflow that may be reusable. Your job is to:

1. **Decompose** the process into atomic, sequential (or branching) steps.
2. **Classify** each step into the most efficient state type:
   - Use **skill_call** FIRST when a reusable sub-process in the catalog (see below) already
     implements the required capability. This is always preferred over reimplementing
     the same logic inline.
   - Use **intermediate** (LLM agent) ONLY when genuine reasoning, judgment, or language
     understanding is required that cannot be expressed deterministically and no
     reusable process covers it.
   - Use **code** (JavaScript) when the step requires computation, transformation, loops,
     or complex conditional logic.
   - Use **script** (expr-lang) for simple single-expression routing or field updates.
   - Use **hitl** when the step explicitly requires a human decision.
   - Use **wait** when the step must pause until external data arrives.
3. **Identify data dependencies** — what each step reads and writes on the blackboard.
4. **Honour the declared I/O contract** — all declared input ports MUST appear as
   blackboard fields; all declared output ports MUST be written before terminal states.
5. **Abstract Generation** — Populate "metadata.description" with a concise 1-sentence abstract. PRESERVE existing descriptions if present.
6. **Minimise LLM usage** — prefer code, script, or skill_call nodes where possible.

Ask yourself for each step:
"Does a reusable process already do this?" If yes, use skill_call.
"Can this be expressed without an LLM?" If yes, use code or script.

`) + base

	if len(registeredProcesses) > 0 {
		preamble += g.buildProcessCatalogSection(registeredProcesses)
	}

	return preamble
}

// buildProcessCatalogSection renders the reusable process catalog as a prompt section.
func (g *Generator) buildProcessCatalogSection(processes []*asmtypes.WorkflowDef) string {
	var sb strings.Builder

	sb.WriteString(`
## Reusable Process Catalog

The following processes are already implemented and marked as reusable. When a step
in the workflow you are designing matches one of these, you MUST emit a **skill_call**
state that delegates to it — do NOT reimplement the same logic inline.

For each skill_call state:
- Set "skill_ref" to the process name shown below.
- Map the parent blackboard fields to the sub-process's input ports via "input_mappings".
- Map the sub-process's output ports back to parent blackboard fields via "output_mappings".

`)

	for _, s := range processes {
		sb.WriteString(fmt.Sprintf("### %s\n", s.Metadata.Name))
		if s.Metadata.Description != "" {
			sb.WriteString(s.Metadata.Description + "\n\n")
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

// buildProcessUserMsg constructs the user message with the full context.
func (g *Generator) buildProcessUserMsg(req ProcessAnalyseRequest) string {
	var sb strings.Builder

	if req.ExistingYAML != "" {
		sb.WriteString("Refine the following workflow to better implement the process described below.\n\n")
		sb.WriteString("## Current Workflow\n\n```yaml\n")
		sb.WriteString(req.ExistingYAML)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("## Process Description\n\n")
	sb.WriteString(req.Description)
	sb.WriteString("\n\n")

	if len(req.Inputs) > 0 {
		sb.WriteString("## Input Contract\n\n")
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
		for _, p := range req.Outputs {
			sb.WriteString(fmt.Sprintf("- `%s` (%s)", p.Name, p.Type))
			if p.Description != "" {
				sb.WriteString(": " + p.Description)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Decompose this into the most efficient ASM workflow. ")
	sb.WriteString("Return only the YAML.\n")

	return sb.String()
}
