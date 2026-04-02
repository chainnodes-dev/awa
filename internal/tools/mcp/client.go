// Package mcp provides a client for the Model Context Protocol (MCP).
// MCP servers expose tools over JSON-RPC 2.0 via HTTP POST.
// Server URLs must come from environment variables — never hardcode them.
// Reference them in workflow YAML agent config as: {{ env.MY_MCP_SERVER_URL }}
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/asm-platform/asm/internal/tools"
)

// Client talks to a single MCP server over HTTP JSON-RPC 2.0.
type Client struct {
	serverURL  string
	httpClient *http.Client
	idCounter  atomic.Int64
}

// NewClient creates a new MCP client for the given server URL.
// The URL must come from environment variables, not from source code.
func NewClient(serverURL string) *Client {
	return &Client{
		serverURL:  serverURL,
		httpClient: &http.Client{},
	}
}

// ListTools queries the MCP server for available tools and returns them as
// Tool instances ready to be added to an agent's tool registry.
func (c *Client) ListTools(ctx context.Context) ([]tools.Tool, error) {
	var result struct {
		Tools []struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"inputSchema"`
		} `json:"tools"`
	}

	if err := c.call(ctx, "tools/list", nil, &result); err != nil {
		return nil, fmt.Errorf("mcp tools/list from %s: %w", c.serverURL, err)
	}

	out := make([]tools.Tool, len(result.Tools))
	for i, t := range result.Tools {
		schema := t.InputSchema
		if len(schema) == 0 {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out[i] = &Tool{
			client:      c,
			name:        t.Name,
			description: t.Description,
			inputSchema: schema,
		}
	}
	return out, nil
}

// CallTool invokes a named tool on the MCP server and returns its JSON result.
func (c *Client) CallTool(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	var rawInput interface{}
	if err := json.Unmarshal(args, &rawInput); err != nil {
		// If args isn't valid JSON, send it as a string.
		rawInput = string(args)
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError,omitempty"`
	}

	if err := c.call(ctx, "tools/call", map[string]interface{}{
		"name":      name,
		"arguments": rawInput,
	}, &result); err != nil {
		return nil, fmt.Errorf("mcp tools/call %s: %w", name, err)
	}

	if result.IsError && len(result.Content) > 0 {
		return nil, fmt.Errorf("tool '%s' returned error: %s", name, result.Content[0].Text)
	}

	// Concatenate all text content blocks.
	var text string
	for _, block := range result.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}

	// Return as JSON: if the text is already valid JSON, pass it through;
	// otherwise wrap it as a JSON string.
	if json.Valid([]byte(text)) {
		return json.RawMessage(text), nil
	}
	wrapped, _ := json.Marshal(text)
	return wrapped, nil
}

// --- JSON-RPC internals ---

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *Client) call(ctx context.Context, method string, params interface{}, out interface{}) error {
	id := c.idCounter.Add(1)

	reqBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return fmt.Errorf("marshal rpc request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http post to %s: %w", c.serverURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mcp server %s returned status %d: %s", c.serverURL, resp.StatusCode, string(body))
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return fmt.Errorf("parse rpc response from %s: %w", c.serverURL, err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	if out != nil && rpcResp.Result != nil {
		if err := json.Unmarshal(rpcResp.Result, out); err != nil {
			return fmt.Errorf("unmarshal rpc result: %w", err)
		}
	}
	return nil
}
