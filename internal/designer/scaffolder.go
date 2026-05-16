package designer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/expr-lang/expr"
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

// ScaffoldResult holds the LLM-generated expr-lang script definition.
type ScaffoldResult struct {
	// Trigger is an expr-lang expression that returns a transition trigger name.
	Trigger string `json:"trigger"`
	// Updates maps blackboard field names to expr-lang expression strings.
	Updates map[string]string `json:"updates,omitempty"`
}

// Scaffold calls the LLM and returns a Go HandlerFunc implementation for the
// described agent. The generated code is ready to paste into a specialist
// worker built with the internal/specialist SDK.
func (g *Generator) Scaffold(ctx context.Context, req ScaffoldRequest) (*ScaffoldResult, error) {
	system := buildScaffoldSystemPrompt()
	
	messages := []llm.Message{
		{Role: "user", Content: buildScaffoldUserMsg(req)},
	}

	maxRetries := 3
	var lastErr error
	var lastRes *ScaffoldResult

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
			SystemPrompt: system,
			Messages:     messages,
			MaxTokens:    4096,
		})
		if err != nil {
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		// Add assistant response to history
		messages = append(messages, llm.Message{Role: "assistant", Content: resp.Content})

		// Extract JSON block
		jsonStr, err := extractJSONBlock(resp.Content)
		if err != nil {
			lastErr = fmt.Errorf("no JSON block in LLM response: %w", err)
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: fmt.Sprintf("Your response did not contain a valid JSON block. Error: %v. Please return ONLY a JSON block.", err),
			})
			continue
		}

		var res ScaffoldResult
		if err := json.Unmarshal([]byte(jsonStr), &res); err != nil {
			lastErr = fmt.Errorf("failed to parse generated script JSON: %w", err)
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: fmt.Sprintf("Failed to parse JSON: %v. Please ensure it is valid JSON and matches the required format.", err),
			})
			continue
		}
		
		lastRes = &res

		if res.Trigger == "" {
			lastErr = fmt.Errorf("LLM failed to generate a trigger expression")
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: "You must include a 'trigger' expression in the JSON.",
			})
			continue
		}

		// Compile trigger
		if _, err := expr.Compile(res.Trigger); err != nil {
			lastErr = fmt.Errorf("invalid trigger expression: %w", err)
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: fmt.Sprintf("The 'trigger' expression failed to compile with expr-lang. Error: %v\nPlease fix the syntax and remember the rules (no if/then/else, no ctx. prefix).", err),
			})
			continue
		}

		// Compile updates
		validUpdates := true
		for k, v := range res.Updates {
			if _, err := expr.Compile(v); err != nil {
				lastErr = fmt.Errorf("invalid update expression for %q: %w", k, err)
				messages = append(messages, llm.Message{
					Role:    "user",
					Content: fmt.Sprintf("The 'updates' expression for field %q failed to compile. Error: %v\nPlease fix the syntax.", k, err),
				})
				validUpdates = false
				break
			}
		}

		if !validUpdates {
			continue
		}

		// Success!
		return &res, nil
	}

	// If we exhausted retries, return the last parsed result anyway if we have one, 
	// so the user can see what was generated (it can be manually fixed in the UI).
	if lastRes != nil {
		return lastRes, nil
	}

	return nil, fmt.Errorf("failed to generate valid script after %d attempts. Last error: %w", maxRetries, lastErr)
}

// buildScaffoldSystemPrompt returns the system prompt that teaches the LLM
// the specialist-worker SDK conventions.
func buildScaffoldSystemPrompt() string {
	return `You are a deterministic logic generator for a state-machine automation engine.
Your task is to convert plain-language instructions or technical requirements into deterministic "expr-lang" expressions.

## Source of Truth
- **Technical Requirements**: If provided, these are your primary, machine-readable instructions.
- **Instructions**: If technical requirements are missing or ambiguous, use the human-readable instructions as a fallback.

## expr-lang Syntax

We use the "expr-lang/expr" Go library. Expressions have access to the "blackboard" as a map of variables.

### Accessing Data
- Variables: Just use the field name directly, e.g. "amount", "status", "invoice_id".
- CRITICAL: DO NOT use prefixes like "ctx.", "bb.", "blackboard.", or "data.".
- Maps: "metadata.author" or "nested['key']".
- Existence: "field != nil".

### Logic & Math
- Standard operators: "+", "-", "*", "/", "==", "!=", ">", "<", ">=", "<=".
- Boolean: "&&", "||", "!".
- Ternary: "condition ? true_val : false_val".
- Chain ternaries for multiple conditions: "cond1 ? val1 : cond2 ? val2 : default_val".
- CRITICAL: "expr-lang" DOES NOT support "if / then / else" syntax. You MUST use ternaries instead.

### Built-ins
- Strings: "contains(str, substr)", "startsWith(str, prefix)", "upper(str)", "lower(str)".
- Collections: "len(list)", "any(list, # > 0)", "all(list, # != nil)".

## Your Task

Generate a JSON object containing:
1. "trigger": A single expression string that returns a string (the name of the transition trigger to fire).
2. "updates": A map of blackboard field names to expression strings evaluating to the new value for that field.

## Rules
1. Be 100% deterministic. No stochastic behavior.
2. The Trigger expression must return one of the allowed trigger names.
3. Access blackboard variables directly by their name, with no prefix.
4. If a field might be missing, use a safe check like "status ?? 'default'".

## Output Format
Respond with ONLY a JSON block:
` + "```json" + `
{
  "trigger": "priority == 'high' ? 'escalate' : priority == 'low' ? 'ignore' : 'process'",
  "updates": {
    "priority": "amount > 100 ? 'high' : 'low'",
    "processed": "true"
  }
}
` + "```" + `
`
}

// buildScaffoldUserMsg constructs the user message from the scaffold request.
func buildScaffoldUserMsg(req ScaffoldRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Generate expr-lang logic for the state %q handled by agent %q", req.StateName, req.AgentName))
	sb.WriteString(".\n\n")

	if req.Description != "" {
		sb.WriteString("## State Description / Instructions\n\n")
		sb.WriteString(req.Description)
		sb.WriteString("\n\n")
	}

	if len(req.Triggers) > 0 {
		sb.WriteString("## Allowed Trigger Names\n\n")
		sb.WriteString("The 'trigger' expression must evaluate to one of these strings:\n")
		for _, t := range req.Triggers {
			sb.WriteString(fmt.Sprintf("- %q\n", t))
		}
		sb.WriteString("\n")
	}

	if len(req.BBSchema) > 0 {
		sb.WriteString("## Blackboard Variable Names\n\n")
		sb.WriteString("Use these field names in your expressions:\n")
		for field, typ := range req.BBSchema {
			sb.WriteString(fmt.Sprintf("- `%s` (%s)\n", field, typ))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Return only a JSON block with 'trigger' and 'updates'.\n")
	return sb.String()
}

// extractGoBlock finds the first ```go ... ``` block in the LLM response.
func extractGoBlock(content string) (string, error) {
	const marker = "```go"
	const endMarker = "```"

	start := strings.Index(content, marker)
	if start != -1 {
		inner := content[start+len(marker):]
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

// extractJSONBlock finds the first ```json ... ``` block in the LLM response.
func extractJSONBlock(content string) (string, error) {
	const marker = "```json"
	const endMarker = "```"

	start := strings.Index(content, marker)
	if start != -1 {
		inner := content[start+len(marker):]
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
			if strings.HasPrefix(candidate, "{") {
				return candidate, nil
			}
		}
	}

	// Last resort: if it looks like JSON, return as-is.
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") {
		return trimmed, nil
	}

	return "", fmt.Errorf("response does not contain a JSON block")
}
