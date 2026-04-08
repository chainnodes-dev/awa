package designer

import (
	"context"
	"fmt"
	"strings"

	"github.com/asm-platform/asm/internal/executor/llm"
)

// ScaffoldRequest is the input to the specialist-worker code generator.
type ScaffoldRequest struct {
	// AgentName is the value of the "name" field in the workflow YAML agent definition.
	AgentName string `json:"agent_name" binding:"required"`
	// StateName is the name of the workflow state this agent handles.
	StateName string `json:"state_name"`
	// Description is a plain-language explanation of what the handler should do.
	Description string `json:"description"`
	// Triggers lists the trigger strings this handler may return.
	// When empty the LLM infers sensible names from the description.
	Triggers []string `json:"triggers,omitempty"`
	// BBSchema maps blackboard field names to their types ("string", "number", "bool", "object").
	BBSchema map[string]string `json:"bb_schema,omitempty"`
}

// ScaffoldResult holds the LLM-generated Go handler code.
type ScaffoldResult struct {
	// Code is a self-contained Go source snippet that can be pasted into a
	// specialist worker binary. It includes the package declaration, imports,
	// and a single HandlerFunc implementation.
	Code string `json:"code"`
}

// Scaffold calls the LLM and returns a Go HandlerFunc implementation for the
// described agent. The generated code is ready to paste into a specialist
// worker built with the internal/specialist SDK.
func (g *Generator) Scaffold(ctx context.Context, req ScaffoldRequest) (*ScaffoldResult, error) {
	system := buildScaffoldSystemPrompt()
	userMsg := buildScaffoldUserMsg(req)

	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages:     []llm.Message{{Role: "user", Content: userMsg}},
		MaxTokens:    4096,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	code, err := extractGoBlock(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("no Go code block in LLM response: %w", err)
	}

	return &ScaffoldResult{Code: code}, nil
}

// buildScaffoldSystemPrompt returns the system prompt that teaches the LLM
// the specialist-worker SDK conventions.
func buildScaffoldSystemPrompt() string {
	return `You are an expert Go developer who writes specialist worker handlers for the ASM (Agentic State Machine) platform.

Your task is to generate a self-contained Go source file containing a single handler function that implements deterministic business logic as an alternative to an LLM agent.

## SDK Contract

Every handler must satisfy the HandlerFunc type:

` + "```go" + `
type HandlerFunc func(ctx context.Context, bb map[string]interface{}) (*asmtypes.AgentOutput, error)
` + "```" + `

Where AgentOutput is:

` + "```go" + `
// AgentOutput is the value returned by every agent (LLM or specialist).
type AgentOutput struct {
    // Trigger is the transition trigger to fire (must be non-empty).
    // It must match one of the "trigger" values defined in the workflow transitions.
    Trigger string ` + "`json:\"trigger\"`" + `

    // Reasoning is a human-readable explanation for the decision (optional).
    Reasoning string ` + "`json:\"reasoning,omitempty\"`" + `

    // BlackboardUpdates is a map of field name → new value to merge into
    // the workflow's shared blackboard (optional).
    BlackboardUpdates map[string]interface{} ` + "`json:\"blackboard_updates,omitempty\"`" + `
}
` + "```" + `

## Imports

Use these exact import paths:

` + "```go" + `
import (
    "context"
    "fmt"

    "github.com/asm-platform/asm/pkg/asmtypes"
)
` + "```" + `

## Blackboard Access Pattern

Blackboard values are always ` + "`map[string]interface{}`" + `. Access them with type assertions:

` + "```go" + `
// String field
val, _ := bb["field_name"].(string)

// Number field (JSON numbers decode as float64)
amount, _ := bb["amount"].(float64)

// Optional with default
status, ok := bb["status"].(string)
if !ok {
    status = "pending"
}
` + "```" + `

## Rules

1. The function must be deterministic — no LLM calls, no HTTP requests to external services.
2. Never return a nil *asmtypes.AgentOutput with a nil error. Always return one or the other.
3. The Trigger field in AgentOutput must be non-empty.
4. Handle type assertion failures gracefully — fields may be absent from the blackboard.
5. Return a descriptive Reasoning string so operators understand the decision.
6. Include a package declaration (` + "`package main`" + `) and all necessary imports.
7. Write idiomatic, production-quality Go code with brief comments.

## Output Format

Respond with ONLY the Go source code, wrapped in a ` + "```go" + ` code block.
Do not include any explanation before or after the code block.
`
}

// buildScaffoldUserMsg constructs the user message from the scaffold request.
func buildScaffoldUserMsg(req ScaffoldRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Generate a specialist worker handler for the agent named %q", req.AgentName))
	if req.StateName != "" {
		sb.WriteString(fmt.Sprintf(" (used in state %q)", req.StateName))
	}
	sb.WriteString(".\n\n")

	if req.Description != "" {
		sb.WriteString("## Handler Description\n\n")
		sb.WriteString(req.Description)
		sb.WriteString("\n\n")
	}

	if len(req.Triggers) > 0 {
		sb.WriteString("## Triggers to Return\n\n")
		sb.WriteString("The handler must return exactly one of these trigger strings:\n")
		for _, t := range req.Triggers {
			sb.WriteString(fmt.Sprintf("- %q\n", t))
		}
		sb.WriteString("\n")
	}

	if len(req.BBSchema) > 0 {
		sb.WriteString("## Blackboard Schema\n\n")
		sb.WriteString("The following fields are available in the blackboard (`bb` parameter):\n")
		for field, typ := range req.BBSchema {
			sb.WriteString(fmt.Sprintf("- `%s` (%s)\n", field, typ))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Generate the complete Go source file including package declaration, imports, and the handler function.\n")
	sb.WriteString("Name the function after the agent: ")
	funcName := agentNameToFuncName(req.AgentName)
	sb.WriteString(fmt.Sprintf("`%s`\n", funcName))

	return sb.String()
}

// extractGoBlock finds the first ```go ... ``` block in the LLM response.
func extractGoBlock(content string) (string, error) {
	const goMarker = "```go"
	const endMarker = "```"

	start := strings.Index(content, goMarker)
	if start != -1 {
		inner := content[start+len(goMarker):]
		end := strings.Index(inner, endMarker)
		if end != -1 {
			return strings.TrimSpace(inner[:end]), nil
		}
	}

	// Plain ``` block as fallback.
	const plainMarker = "```"
	start = strings.Index(content, plainMarker)
	if start != -1 {
		inner := content[start+len(plainMarker):]
		end := strings.Index(inner, plainMarker)
		if end != -1 {
			candidate := strings.TrimSpace(inner[:end])
			if strings.Contains(candidate, "func ") {
				return candidate, nil
			}
		}
	}

	// Last resort: if it looks like Go code, return as-is.
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "package ") {
		return trimmed, nil
	}

	return "", fmt.Errorf("response does not contain a Go code block")
}

// agentNameToFuncName converts a kebab-case or snake_case agent name to a
// camelCase Go function name, e.g. "invoice-validator" → "invoiceValidator".
func agentNameToFuncName(name string) string {
	// Normalise separators to underscore first.
	name = strings.ReplaceAll(name, "-", "_")
	parts := strings.Split(name, "_")
	if len(parts) == 0 {
		return "handle"
	}
	var sb strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			sb.WriteString(strings.ToLower(p))
		} else {
			sb.WriteString(strings.ToUpper(p[:1]))
			sb.WriteString(strings.ToLower(p[1:]))
		}
	}
	return sb.String()
}
