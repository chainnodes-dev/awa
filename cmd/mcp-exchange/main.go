package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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

var tools = []toolDescriptor{
	{
		Name:        "get_exchange_rate",
		Description: "Convert a financial amount from one currency to another using current ECB exchange rates (Frankfurter).",
		InputSchema: inputSchema{
			Type: "object",
			Properties: map[string]schemaProp{
				"amount": {Type: "number", Description: "The numerical amount to convert"},
				"from":   {Type: "string", Description: "The 3-letter currency code to convert from (e.g. 'GBP', 'USD')"},
				"to":     {Type: "string", Description: "The 3-letter currency code to convert to (e.g. 'EUR')"},
			},
			Required: []string{"amount", "from", "to"},
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
	if params.Name != "get_exchange_rate" {
		return callResult{}, fmt.Errorf("unknown tool: %s", params.Name)
	}

	amount, okAm := params.Arguments["amount"].(float64)
	from, okFr := params.Arguments["from"].(string)
	to, okTo := params.Arguments["to"].(string)

	if !okAm || !okFr || !okTo {
		return callResult{}, fmt.Errorf("missing or invalid arguments: amount(number), from(string), to(string)")
	}

	reqUrl := fmt.Sprintf("https://api.frankfurter.app/latest?amount=%v&from=%s&to=%s", amount, url.QueryEscape(from), url.QueryEscape(to))

	resp, err := http.Get(reqUrl)
	if err != nil {
		return callResult{}, fmt.Errorf("failed to call Frankfurter API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return callResult{}, err
	}

	// Just return the raw JSON for the LLM. 
	// Frankfurter returns e.g. {"amount":10.0,"base":"GBP","date":"2023-01-01","rates":{"EUR":11.5}}
	return callResult{Content: []contentItem{{Type: "text", Text: string(body)}}}, nil
}

// ── HTTP handler ──────────────────────────────────────────────────────────────

func handler(w http.ResponseWriter, r *http.Request) {
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
		port = "8092"
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

	slog.Info("MCP Exchange Rates server listening", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
