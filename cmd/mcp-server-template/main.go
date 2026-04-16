// cmd/mcp-server-template is a ready-to-copy starter for building standalone MCP servers.
//
// # Dockerfile (copy this):
//
//	FROM golang:1.23-alpine AS builder
//	WORKDIR /app
//	COPY . .
//	RUN go build -o mcp-server .
//
//	FROM alpine:latest
//	COPY --from=builder /app/mcp-server .
//	EXPOSE 8090
//	CMD ["./mcp-server"]
//
// # Usage
//
//  1. Copy this directory to your own repository.
//  2. Update the module path in go.mod.
//  3. Replace the example tool with your own logic.
//  4. Run: PORT=8090 go run .
//
// # Environment variables
//
//	PORT  HTTP port to listen on (default: 8090)
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// ── JSON-RPC 2.0 types ────────────────────────────────────────────────────────

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func errResp(id interface{}, code int, msg string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
}

// ── Tool descriptors ──────────────────────────────────────────────────────────

type toolDescriptor struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string                `json:"type"`
	Properties map[string]schemaProp `json:"properties"`
	Required   []string              `json:"required,omitempty"`
}

type schemaProp struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// tools is the static list of tools this server exposes.
// Add or remove entries to match your handlers below.
var tools = []toolDescriptor{
	{
		Name:        "greet",
		Description: "Returns a greeting for the given name.",
		InputSchema: inputSchema{
			Type: "object",
			Properties: map[string]schemaProp{
				"name": {Type: "string", Description: "The name to greet"},
			},
			Required: []string{"name"},
		},
	},
}

// ── Tool call dispatcher ──────────────────────────────────────────────────────

type callParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type callResult struct {
	Content []contentItem `json:"content"`
}

func dispatchTool(params callParams) (callResult, error) {
	switch params.Name {
	case "greet":
		name, _ := params.Arguments["name"].(string)
		if name == "" {
			return callResult{}, fmt.Errorf("name is required")
		}
		return callResult{Content: []contentItem{
			{Type: "text", Text: fmt.Sprintf("Hello, %s! 👋", name)},
		}}, nil

	default:
		return callResult{}, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

// ── HTTP handler ──────────────────────────────────────────────────────────────

func handler(w http.ResponseWriter, r *http.Request) {
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, errResp(nil, -32700, "parse error: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var resp rpcResponse
	switch req.Method {
	case "initialize":
		resp = rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{},
				"serverInfo": map[string]string{
					"name":    "mcp-server",
					"version": "1.0.0",
				},
			},
		}
	case "notifications/initialized":
		return // notifications don't get a response

	case "tools/list":
		resp = rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{"tools": tools}}

	case "tools/call":
		var p callParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			resp = errResp(req.ID, -32602, "invalid params: "+err.Error())
			break
		}
		result, err := dispatchTool(p)
		if err != nil {
			resp = errResp(req.ID, -32603, err.Error())
		} else {
			resp = rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
		}

	default:
		resp = errResp(req.ID, -32601, "method not found: "+req.Method)
	}

	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	_ = json.NewEncoder(w).Encode(v)
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("Shutting down")
		_ = srv.Close()
	}()

	slog.Info("MCP server listening", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
