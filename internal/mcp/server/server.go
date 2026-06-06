package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// ToolParam describes a parameter in the inputSchema.
type ToolParam struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// InputSchema describes JSON Schema parameters.
type InputSchema struct {
	Type       string               `json:"type"`
	Properties map[string]ToolParam `json:"properties"`
	Required   []string             `json:"required,omitempty"`
}

// Tool represents an MCP tool definition and its handler.
type Tool struct {
	Name        string
	Description string
	InputSchema InputSchema
	Handler     func(arguments map[string]interface{}) (interface{}, error)
}

// Server implements a lightweight MCP Server.
type Server struct {
	Name    string
	Version string
	tools   []Tool
}

// New creates a new Server instance.
func New(name, version string) *Server {
	return &Server{
		Name:    name,
		Version: version,
		tools:   []Tool{},
	}
}

// Register registers a tool with the server.
func (s *Server) Register(t Tool) {
	s.tools = append(s.tools, t)
}

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

type toolDescriptor struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

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

// Start runs the HTTP server and blocks until terminated by signal.
func (s *Server) Start(defaultPort string) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRPC)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("Shutting down MCP server", "name", s.Name)
		_ = srv.Close()
	}()

	slog.Info("MCP server listening", "name", s.Name, "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
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
		s.writeJSON(w, errResp(nil, -32700, "parse error: "+err.Error()))
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
					"name":    s.Name,
					"version": s.Version,
				},
			},
		}
	case "notifications/initialized":
		return

	case "tools/list":
		var list []toolDescriptor
		for _, t := range s.tools {
			list = append(list, toolDescriptor{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.InputSchema,
			})
		}
		resp = rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{"tools": list},
		}

	case "tools/call":
		var p callParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			resp = errResp(req.ID, -32602, "invalid params: "+err.Error())
			break
		}

		var targetTool *Tool
		for i := range s.tools {
			if s.tools[i].Name == p.Name {
				targetTool = &s.tools[i]
				break
			}
		}

		if targetTool == nil {
			resp = errResp(req.ID, -32601, "method not found: "+p.Name)
			break
		}

		res, err := targetTool.Handler(p.Arguments)
		if err != nil {
			resp = errResp(req.ID, -32603, err.Error())
		} else {
			var textResult string
			switch v := res.(type) {
			case string:
				textResult = v
			case []byte:
				textResult = string(v)
			default:
				bytesVal, marshalErr := json.Marshal(res)
				if marshalErr != nil {
					textResult = fmt.Sprintf("%v", res)
				} else {
					textResult = string(bytesVal)
				}
			}
			resp = rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: callResult{
					Content: []contentItem{
						{Type: "text", Text: textResult},
					},
				},
			}
		}

	default:
		resp = errResp(req.ID, -32601, "method not found: "+req.Method)
	}

	s.writeJSON(w, resp)
}

func (s *Server) writeJSON(w http.ResponseWriter, v interface{}) {
	_ = json.NewEncoder(w).Encode(v)
}
