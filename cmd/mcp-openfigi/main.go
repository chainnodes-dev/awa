package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/asm-platform/asm/internal/mcp/server"
)

func main() {
	s := server.New("mcp-openfigi", "1.0.0")

	s.Register(server.Tool{
		Name:        "figi_lookup",
		Description: "Lookup the Financial Instrument Global Identifier (FIGI) for a given stock ticker using OpenFIGI. Provides the FIGI code and official company name.",
		InputSchema: server.InputSchema{
			Type: "object",
			Properties: map[string]server.ToolParam{
				"ticker": {Type: "string", Description: "The stock ticker symbol (e.g. 'AAPL')"},
			},
			Required: []string{"ticker"},
		},
		Handler: figiLookup,
	})

	if err := s.Start("8093"); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func figiLookup(args map[string]interface{}) (interface{}, error) {
	ticker, ok := args["ticker"].(string)
	if !ok || ticker == "" {
		return nil, fmt.Errorf("ticker is required")
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
		return nil, fmt.Errorf("failed to call OpenFIGI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("API Error (%d): %s", resp.StatusCode, string(body)), nil
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
		return string(body), nil
	}

	if len(apiResp[0].Data) == 0 {
		return "No FIGI mapping found for the given ticker.", nil
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

	return resultText, nil
}
