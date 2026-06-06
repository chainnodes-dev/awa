package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/asm-platform/asm/internal/mcp/server"
)

func main() {
	s := server.New("mcp-exchange", "1.0.0")

	s.Register(server.Tool{
		Name:        "get_exchange_rate",
		Description: "Convert a financial amount from one currency to another using current ECB exchange rates (Frankfurter).",
		InputSchema: server.InputSchema{
			Type: "object",
			Properties: map[string]server.ToolParam{
				"amount": {Type: "number", Description: "The numerical amount to convert"},
				"from":   {Type: "string", Description: "The 3-letter currency code to convert from (e.g. 'GBP', 'USD')"},
				"to":     {Type: "string", Description: "The 3-letter currency code to convert to (e.g. 'EUR')"},
			},
			Required: []string{"amount", "from", "to"},
		},
		Handler: getExchangeRate,
	})

	if err := s.Start("8092"); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func getExchangeRate(args map[string]interface{}) (interface{}, error) {
	amount, okAm := args["amount"].(float64)
	from, okFr := args["from"].(string)
	to, okTo := args["to"].(string)

	if !okAm || !okFr || !okTo {
		return nil, fmt.Errorf("missing or invalid arguments: amount(number), from(string), to(string)")
	}

	reqUrl := fmt.Sprintf("https://api.frankfurter.app/latest?amount=%v&from=%s&to=%s", amount, url.QueryEscape(from), url.QueryEscape(to))

	resp, err := http.Get(reqUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to call Frankfurter API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
