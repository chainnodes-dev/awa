package designer

import (
	"context"
	"fmt"
	"strings"

	"github.com/asm-platform/asm/internal/executor/llm"
)

// MCPToolDef describes a single tool that the generated MCP server should expose.
type MCPToolDef struct {
	// Name is the tool identifier (snake_case).
	Name        string `json:"name"`
	Description string `json:"description"`
	// InputSchema maps parameter names to their JSON types ("string", "number", "boolean", "object", "array").
	InputSchema map[string]string `json:"input_schema,omitempty"`
}

// MCPScaffoldRequest is the input to the MCP server code generator.
type MCPScaffoldRequest struct {
	// ServerName is the logical name for the server binary (kebab-case).
	ServerName string `json:"server_name" binding:"required"`
	// Description explains the server's overall purpose.
	Description string `json:"description"`
	// Tools is the list of tools to expose.
	Tools []MCPToolDef `json:"tools" binding:"required,min=1"`
}

// MCPScaffoldResult holds the generated Go MCP server source code.
type MCPScaffoldResult struct {
	// Code is a complete, self-contained Go source file ready to run with `go run .`.
	Code string `json:"code"`
}

// ScaffoldMCP calls the LLM and returns a complete Go MCP server implementation
// for the described tools. The server exposes JSON-RPC 2.0 tools/list and tools/call
// endpoints and is ready to run as a single binary.
func (g *Generator) ScaffoldMCP(ctx context.Context, req MCPScaffoldRequest) (*MCPScaffoldResult, error) {
	system := buildMCPScaffoldSystemPrompt()
	userMsg := buildMCPScaffoldUserMsg(req)

	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages:     []llm.Message{{Role: "user", Content: userMsg}},
		MaxTokens:    8192,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	code, err := extractGoBlock(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("no Go code block in LLM response: %w", err)
	}

	return &MCPScaffoldResult{Code: code}, nil
}

func buildMCPScaffoldSystemPrompt() string {
	return `You are an expert Go developer who builds standalone MCP (Model Context Protocol) servers.

Your task is to generate a complete, production-ready Go MCP server that:
1. Exposes a JSON-RPC 2.0 HTTP endpoint (POST /)
2. Implements the ` + "`tools/list`" + ` method — returns a list of available tools
3. Implements the ` + "`tools/call`" + ` method — dispatches to the correct handler

## JSON-RPC 2.0 Protocol

Request format:
` + "```json" + `
{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}}
{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": "tool_name", "arguments": {...}}}
` + "```" + `

Response format (success):
` + "```json" + `
{"jsonrpc": "2.0", "id": 1, "result": {...}}
` + "```" + `

Response format (error):
` + "```json" + `
{"jsonrpc": "2.0", "id": 1, "error": {"code": -32601, "message": "Method not found"}}
` + "```" + `

## Required JSON-RPC Error Codes

- ` + "`-32700`" + ` Parse error
- ` + "`-32600`" + ` Invalid Request
- ` + "`-32601`" + ` Method not found
- ` + "`-32602`" + ` Invalid params
- ` + "`-32603`" + ` Internal error

## tools/list Response Structure

` + "```json" + `
{
  "tools": [
    {
      "name": "tool_name",
      "description": "What this tool does",
      "inputSchema": {
        "type": "object",
        "properties": {
          "param1": {"type": "string", "description": "..."}
        },
        "required": ["param1"]
      }
    }
  ]
}
` + "```" + `

## tools/call Response Structure

` + "```json" + `
{
  "content": [
    {"type": "text", "text": "Result of the tool call"}
  ]
}
` + "```" + `

## Go Implementation Requirements

1. Single file, package main, standard library only (no external deps).
2. HTTP server on port from ` + "`PORT`" + ` env var (default: 8090).
3. Each tool has its own handler function.
4. Graceful shutdown on SIGINT/SIGTERM.
5. Structured log output to stdout.
6. CORS headers: ` + "`Access-Control-Allow-Origin: *`" + `.
7. Health check endpoint: ` + "`GET /health`" + ` → 200 OK.

## Rules

1. The tool logic should be deterministic and idiomatic Go.
2. Tool handlers should return descriptive text content.
3. Include brief comments explaining the tool logic.
4. Use standard library ` + "`encoding/json`" + `, ` + "`net/http`" + `, ` + "`log/slog`" + `.
5. Include a Dockerfile comment at the top showing how to containerise.

## Output Format

Respond with ONLY the Go source code wrapped in a ` + "```go" + ` code block.
`
}

func buildMCPScaffoldUserMsg(req MCPScaffoldRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Generate a Go MCP server named %q.\n\n", req.ServerName))

	if req.Description != "" {
		sb.WriteString("## Server Purpose\n\n")
		sb.WriteString(req.Description)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Tools to Expose\n\n")
	for _, tool := range req.Tools {
		sb.WriteString(fmt.Sprintf("### %s\n", tool.Name))
		sb.WriteString(fmt.Sprintf("Description: %s\n", tool.Description))
		if len(tool.InputSchema) > 0 {
			sb.WriteString("Input parameters:\n")
			for param, typ := range tool.InputSchema {
				sb.WriteString(fmt.Sprintf("  - `%s` (%s)\n", param, typ))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Generate the complete Go source file. Implement realistic stub logic for each tool that demonstrates the intended behaviour.\n")

	return sb.String()
}
