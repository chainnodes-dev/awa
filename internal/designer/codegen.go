package designer

import (
	"context"
	"fmt"
	"strings"

	"github.com/asm-platform/asm/internal/executor/llm"
)

// CodegenRequest is the input to the JavaScript code generator.
type CodegenRequest struct {
	// Instructions is a plain-language description of what the code should do.
	Instructions string `json:"instructions" binding:"required"`
	// StateName is the name of the workflow state (used for context only).
	StateName string `json:"state_name,omitempty"`
	// ValidTriggers lists the transition trigger names the code may fire.
	ValidTriggers []string `json:"valid_triggers,omitempty"`
	// BBSchema maps blackboard field names to their types.
	BBSchema map[string]string `json:"bb_schema,omitempty"`
	// ExistingCode is the current editor content. When non-empty the LLM
	// refines it rather than generating from scratch.
	ExistingCode string `json:"existing_code,omitempty"`
}

// CodegenResult holds the LLM-generated JavaScript code.
type CodegenResult struct {
	// Code is ready-to-run JavaScript for the goja sandbox.
	Code string `json:"code"`
	// Explanation is a one-line human-readable summary of what was generated.
	Explanation string `json:"explanation,omitempty"`
}

// Codegen calls the LLM and returns JavaScript code for a Code node.
func (g *Generator) Codegen(ctx context.Context, req CodegenRequest) (*CodegenResult, error) {
	system := buildCodegenSystemPrompt()
	userMsg := buildCodegenUserMsg(req)

	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages:     []llm.Message{{Role: "user", Content: userMsg}},
		MaxTokens:    4096,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	code, err := extractJSBlock(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("no JavaScript code block in LLM response: %w", err)
	}

	// Extract optional explanation from a line starting with "// Explanation:"
	explanation := ""
	for _, line := range strings.Split(resp.Content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Explanation:") {
			explanation = strings.TrimSpace(strings.TrimPrefix(line, "Explanation:"))
			break
		}
	}

	return &CodegenResult{Code: code, Explanation: explanation}, nil
}

// buildCodegenSystemPrompt returns the system prompt that teaches the LLM
// the goja sandbox environment and expected output format.
func buildCodegenSystemPrompt() string {
	return `You are a JavaScript code generator for a state-machine automation engine.
Your task is to write JavaScript code that runs inside a deterministic sandboxed Code node based on plain-language instructions or technical requirements.

## Source of Truth
- **Technical Requirements**: If provided, these are your primary, machine-readable instructions.
- **Instructions**: If technical requirements are missing or ambiguous, use the human-readable instructions as a fallback.

## Sandbox Environment

The code runs in a goja JavaScript engine (ES5.1+ with some ES6). The following are available:

### Blackboard (bb)
` + "`bb`" + ` is a mutable object containing the current workflow blackboard.
Read and write fields directly:
` + "```js" + `
const amount = bb.amount;   // read
bb.result = amount * 2;     // write — change is captured automatically
` + "```" + `

### Triggering Transitions
Fire a transition in one of two ways:

**Early exit** (stops execution immediately):
` + "```js" + `
trigger('trigger_name');
` + "```" + `

**Return value** (after all logic has run):
` + "```js" + `
return {
  trigger: 'trigger_name',
  reasoning: 'optional explanation logged in the event log',
};
` + "```" + `

You can also pass explicit blackboard updates in the return value (these override any ` + "`bb`" + ` mutations):
` + "```js" + `
return {
  trigger: 'done',
  blackboard_updates: { result: 42 },
};
` + "```" + `

### console
` + "`console.log()`" + `, ` + "`console.warn()`" + `, ` + "`console.error()`" + ` are available (output captured in server logs).

## Constraints
- No ` + "`fetch`" + `, ` + "`XMLHttpRequest`" + `, or any network access — the sandbox is fully isolated.
- No ` + "`require`" + ` / ` + "`import`" + ` — no module system.
- No ` + "`setTimeout`" + ` / ` + "`setInterval`" + `.
- The trigger name in ` + "`return { trigger }`" + ` or ` + "`trigger()`" + ` MUST match one of the allowed trigger names.
- Prefer mutating ` + "`bb`" + ` directly over ` + "`blackboard_updates`" + ` unless you need to conditionally replace all updates atomically.

## Output Format
Respond with:
1. A single ` + "```js" + ` code block containing the complete JavaScript code.
2. A single line starting with ` + "`Explanation:`" + ` summarising what the code does (one sentence, plain English).

Do NOT include any other prose.
`
}

// buildCodegenUserMsg constructs the user message from the codegen request.
func buildCodegenUserMsg(req CodegenRequest) string {
	var sb strings.Builder

	if req.ExistingCode != "" {
		sb.WriteString(fmt.Sprintf("Refine the JavaScript code for the state %q.\n\n", req.StateName))
	} else {
		sb.WriteString(fmt.Sprintf("Generate JavaScript code for the state %q.\n\n", req.StateName))
	}

	sb.WriteString("## Instructions\n\n")
	sb.WriteString(req.Instructions)
	sb.WriteString("\n\n")

	if len(req.ValidTriggers) > 0 {
		sb.WriteString("## Valid Trigger Names\n\n")
		sb.WriteString("The code MUST fire exactly one of these triggers:\n")
		for _, t := range req.ValidTriggers {
			sb.WriteString(fmt.Sprintf("- %q\n", t))
		}
		sb.WriteString("\n")
	}

	if len(req.BBSchema) > 0 {
		sb.WriteString("## Blackboard Schema\n\n")
		sb.WriteString("These fields are available on `bb`:\n")
		for field, typ := range req.BBSchema {
			sb.WriteString(fmt.Sprintf("- `bb.%s` (%s)\n", field, typ))
		}
		sb.WriteString("\n")
	}

	if req.ExistingCode != "" {
		sb.WriteString("## Existing Code (refine this)\n\n")
		sb.WriteString("```js\n")
		sb.WriteString(req.ExistingCode)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("Return a ```js code block followed by an Explanation: line.\n")
	return sb.String()
}

// extractJSBlock finds the first ```js or ```javascript block in the LLM response,
// falling back to any plain ``` block that looks like JavaScript.
func extractJSBlock(content string) (string, error) {
	for _, marker := range []string{"```js\n", "```javascript\n", "```js", "```javascript"} {
		start := strings.Index(content, marker)
		if start == -1 {
			continue
		}
		inner := content[start+len(marker):]
		end := strings.Index(inner, "```")
		if end != -1 {
			return strings.TrimSpace(inner[:end]), nil
		}
	}

	// Plain ``` block fallback — accept if it contains JS-ish keywords.
	start := strings.Index(content, "```\n")
	if start != -1 {
		inner := content[start+4:]
		end := strings.Index(inner, "```")
		if end != -1 {
			candidate := strings.TrimSpace(inner[:end])
			if strings.ContainsAny(candidate, "{}();") {
				return candidate, nil
			}
		}
	}

	// Last resort: if the whole response looks like JS, use it directly.
	trimmed := strings.TrimSpace(content)
	if strings.ContainsAny(trimmed, "{}();") && !strings.HasPrefix(trimmed, "{") {
		return trimmed, nil
	}

	return "", fmt.Errorf("response does not contain a JavaScript code block")
}
