package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/asm-platform/asm/internal/mcp/server"
)

func main() {
	s := server.New("mcp-gleif", "1.0.0")

	s.Register(server.Tool{
		Name:        "gleif_search",
		Description: "Search for a company's Legal Entity Identifier (LEI) and official details using the GLEIF database.",
		InputSchema: server.InputSchema{
			Type: "object",
			Properties: map[string]server.ToolParam{
				"query": {Type: "string", Description: "The company name to search for (e.g. 'Apple Inc')"},
			},
			Required: []string{"query"},
		},
		Handler: gleifSearch,
	})

	if err := s.Start("8091"); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func gleifSearch(args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	encodedQuery := url.QueryEscape(query)
	reqUrl := fmt.Sprintf("https://api.gleif.org/api/v1/lei-records?filter[entity.legalName]=%s", encodedQuery)

	req, _ := http.NewRequest(http.MethodGet, reqUrl, nil)
	req.Header.Set("Accept", "application/vnd.api+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call GLEIF: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		Data []struct {
			Attributes struct {
				LEI    string `json:"lei"`
				Entity struct {
					LegalName struct {
						Name string `json:"name"`
					} `json:"legalName"`
					LegalAddress struct {
						City    string `json:"city"`
						Country string `json:"country"`
					} `json:"legalAddress"`
					EntityStatus string `json:"entityStatus"`
				} `json:"entity"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil || len(apiResp.Data) == 0 {
		if len(apiResp.Data) == 0 && err == nil {
			return "No company found for query.", nil
		}
		return string(body), nil
	}

	first := apiResp.Data[0].Attributes
	summary := fmt.Sprintf("Legal Name: %s\nLEI: %s\nStatus: %s\nCity: %s\nCountry: %s",
		first.Entity.LegalName.Name, first.LEI, first.Entity.EntityStatus, first.Entity.LegalAddress.City, first.Entity.LegalAddress.Country)

	return summary, nil
}
