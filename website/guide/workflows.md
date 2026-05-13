# Workflow DSL

Phaxa workflows are defined using a declarative YAML schema. This allows for version control, portability, and readability.

## Basic Structure

```yaml
name: "Financial Analysis"
description: "Extracts metrics from a report"
metadata:
  reusable: true

blackboard:
  schema:
    ticker: "string"
    revenue: "number"
    report: "file"

states:
  START:
    type: "prompt"
    agent: "analyst"
    prompt: "Read the report and extract revenue for {{ blackboard.ticker }}."
    transitions:
      success: "REVIEW"
      
  REVIEW:
    type: "hitl"
    assignee: "finance_team"
    form_schema:
      approved: "boolean"
    transitions:
      success: "TERMINAL"
```

## Node Types

| Type | Description |
| :--- | :--- |
| `prompt` | Executes an LLM agent with optional tools (MCP). |
| `hitl` | Pauses execution for human input or judgment. |
| `script` | Executes a small JS expression for data manipulation. |
| `wait` | Pauses for a specific duration or external signal. |
| `terminal` | The end state of a workflow. |

## Expressions
Phaxa uses the `expr` engine. You can access the blackboard data using `{{ blackboard.key }}` in prompts or `blackboard.key` directly in scripts.
