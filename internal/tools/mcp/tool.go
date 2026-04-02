package mcp

import (
	"context"
	"encoding/json"

	"github.com/asm-platform/asm/internal/tools"
)

// Tool is a single MCP tool exposed as the tools.Tool interface.
// It delegates execution to the MCP client that owns it.
type Tool struct {
	client      *Client
	name        string
	description string
	inputSchema json.RawMessage
}

func (t *Tool) Name() string                 { return t.name }
func (t *Tool) Description() string          { return t.description }
func (t *Tool) InputSchema() json.RawMessage { return t.inputSchema }

func (t *Tool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	return t.client.CallTool(ctx, t.name, input)
}

// LoadRegistry connects to an MCP server, discovers its tools, and returns a
// tools.Registry pre-populated with them.
// serverURL must be a resolved value from an environment variable — never a
// hardcoded string.
func LoadRegistry(ctx context.Context, serverURL string) (*tools.Registry, error) {
	c := NewClient(serverURL)
	toolList, err := c.ListTools(ctx)
	if err != nil {
		return nil, err
	}
	reg := tools.NewRegistry()
	for _, t := range toolList {
		reg.Register(t)
	}
	return reg, nil
}
