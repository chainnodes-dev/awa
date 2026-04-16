package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

var tools = []toolDescriptor{
	{
		Name:        "figi_lookup",
		Description: "Lookup the Financial Instrument Global Identifier (FIGI) for a given stock ticker using OpenFIGI. Provides the FIGI code and official company name.",
		InputSchema: inputSchema{
			Type: "object",
			Properties: map[string]schemaProp{
				"ticker": {Type: "string", Description: "The stock ticker symbol (e.g. 'AAPL')"},
			},
			Required: []string{"ticker"},
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
	if params.Name != "figi_lookup" {
		return callResult{}, fmt.Errorf("unknown tool: %s", params.Name)
	}

	ticker, ok := params.Arguments["ticker"].(string)
	if !ok || ticker == "" {
		return callResult{}, fmt.Errorf("ticker is required")
	}

	// OpenFIGI POST payload
	payload := []map[string]string{
		{
			"idType":  "TICKER",
			"idValue": ticker,
		},
	}
	bodyData, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "https://api.openfigi.com/v3/mapping", bytes.NewReader(bodyData))
	req.Header.Set("Content-Type", "application/json")
	
	// Add optional API key
	apiKey := os.Getenv("OPENFIGI_API_KEY")
	if apiKey != "" {
		req.Header.Set("X-OPENFIGI-APIKEY", apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return callResult{}, fmt.Errorf("failed to call OpenFIGI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return callResult{}, err
	}

    if resp.StatusCode != http.StatusOK {
        return callResult{Content: []contentItem{{Type: "text", Text: fmt.Sprintf("API Error (%d): %s", resp.StatusCode, string(body))}}}, nil
    }

	var apiResp []struct {
		Data []struct {
			Figi        string `json:"figi"`
			Name        string `json:"name"`
			Ticker      string `json:"ticker"`
			ExchCode    string `json:"exchCode"`
			MarketSector string `json:"marketSector"`
            SecurityType string `json:"securityType"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil || len(apiResp) == 0 {
		return callResult{Content: []contentItem{{Type: "text", Text: string(body)}}}, nil
	}

	if len(apiResp[0].Data) == 0 {
		return callResult{Content: []contentItem{{Type: "text", Text: "No FIGI mapping found for the given ticker."}}}, nil
	}

	// Extract the most relevant entries
    resultText := fmt.Sprintf("Found %d mappings for ticker %s:\n", len(apiResp[0].Data), ticker)
    for i, item := range apiResp[0].Data {
        if i >= 3 {
             resultText += "... (truncated)"
             break
        }
        resultText += fmt.Sprintf("- Name: %s | FIGI: %s | Exchange: %s | Type: %s\n", item.Name, item.Figi, item.ExchCode, item.SecurityType)
    }

	return callResult{Content: []contentItem{{Type: "text", Text: resultText}}}, nil
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
		port = "8093"
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

	slog.Info("MCP OpenFIGI server listening", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
