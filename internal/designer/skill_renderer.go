package designer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// RenderSkillRequest is the input to the workflow → skill description converter.
type RenderSkillRequest struct {
	// WorkflowYAML is the raw YAML of the workflow to describe.
	WorkflowYAML string `json:"workflow_yaml" binding:"required"`
	// ExistingSkill is the current SkillDef when updating an existing description.
	// When provided, the LLM preserves declared ports and updates only the prose.
	ExistingSkill *asmtypes.SkillDef `json:"existing_skill,omitempty"`
}

// RenderSkillResult holds the rendered human-readable description plus
// inferred I/O ports extracted from the workflow's blackboard schema.
type RenderSkillResult struct {
	// Description is the generated plain-English skill description.
	Description string `json:"description"`
	// InferredInputs are ports the LLM inferred from the initial-state blackboard fields.
	InferredInputs []asmtypes.SkillPortDef `json:"inferred_inputs,omitempty"`
	// InferredOutputs are ports inferred from fields written before terminal states.
	InferredOutputs []asmtypes.SkillPortDef `json:"inferred_outputs,omitempty"`
}

// RenderSkill produces a human-readable description of a workflow as a skill.
// It also infers the I/O port contract from the blackboard schema.
func (g *Generator) RenderSkill(ctx context.Context, req RenderSkillRequest) (*RenderSkillResult, error) {
	system := buildRenderSkillSystemPrompt()
	userMsg := buildRenderSkillUserMsg(req)

	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages:     []llm.Message{{Role: "user", Content: userMsg}},
		MaxTokens:    2048,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	jsonStr, err := extractJSONBlock(resp.Content)
	if err != nil {
		// Fallback: try to use the raw response as the description.
		return &RenderSkillResult{Description: strings.TrimSpace(resp.Content)}, nil
	}

	var result RenderSkillResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse render result JSON: %w", err)
	}

	return &result, nil
}

func buildRenderSkillSystemPrompt() string {
	return `You are a technical writer for the ASM (Agentic State Machine) platform.

Your task is to read an ASM workflow YAML definition and produce:
1. A clear, concise natural-language description of what the skill does — written for
   a developer or business analyst who has not seen the YAML.
2. An inferred list of input ports (data the skill requires at invocation time).
3. An inferred list of output ports (data the skill produces when complete).

## Rules

- The description should be 2-5 sentences. Describe the WHAT and WHY, not the HOW.
  Do not mention internal state names, node types, or technical implementation details.
- Infer inputs from blackboard fields that are required at run start (required: true)
  or consumed by the initial state without being written first.
- Infer outputs from blackboard fields written by states that transition to terminal nodes.
- For each port include: name (the blackboard field name), type (string/number/bool/object),
  description (one short sentence), required (true/false).

## Output Format

Respond with ONLY a JSON block:

` + "```json" + `
{
  "description": "Plain-English description of the skill...",
  "inferred_inputs": [
    { "name": "field_name", "type": "string", "description": "What this field represents", "required": true }
  ],
  "inferred_outputs": [
    { "name": "result_field", "type": "number", "description": "What this field contains", "required": false }
  ]
}
` + "```" + `
`
}

func buildRenderSkillUserMsg(req RenderSkillRequest) string {
	var sb strings.Builder

	sb.WriteString("## Workflow YAML\n\n```yaml\n")
	sb.WriteString(req.WorkflowYAML)
	sb.WriteString("\n```\n\n")

	if req.ExistingSkill != nil {
		sb.WriteString("## Existing Skill Description (update it, preserve port names)\n\n")
		sb.WriteString(req.ExistingSkill.Metadata.Description)
		sb.WriteString("\n\n")
		if len(req.ExistingSkill.Inputs) > 0 {
			sb.WriteString("Current declared inputs: ")
			names := make([]string, 0, len(req.ExistingSkill.Inputs))
			for _, p := range req.ExistingSkill.Inputs {
				names = append(names, p.Name)
			}
			sb.WriteString(strings.Join(names, ", "))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("Describe this workflow as a skill and infer its input/output ports. Return only the JSON block.\n")
	return sb.String()
}
