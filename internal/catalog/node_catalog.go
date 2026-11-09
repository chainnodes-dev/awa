package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type NodeCatalogEntry struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Category     string                 `json:"category,omitempty"`
	Icon         string                 `json:"icon,omitempty"`
	Description  string                 `json:"description,omitempty"`
	InputsSchema map[string]interface{} `json:"inputs_schema,omitempty"`
	CodeTemplate string                 `json:"code_template"`
	Author       string                 `json:"author,omitempty"`
	CreatedAt    string                 `json:"created_at,omitempty"`
}

func FetchNodeCatalog(ctx context.Context, source string) ([]NodeCatalogEntry, error) {
	if source == "" {
		return GetDefaultCatalog(), nil
	}

	// Try parsing as raw JSON if it looks like a JSON array
	if strings.HasPrefix(strings.TrimSpace(source), "[") {
		var entries []NodeCatalogEntry
		if err := json.Unmarshal([]byte(source), &entries); err == nil {
			return entries, nil
		}
	}

	// Remote HTTP/HTTPS
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, err
		}
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var entries []NodeCatalogEntry
		if err := json.Unmarshal(body, &entries); err != nil {
			return nil, fmt.Errorf("failed to parse catalog JSON: %w", err)
		}
		return entries, nil
	}

	// Local file
	path := source
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, source)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// Fallback check parent directories if running from test/subfolder
		if !os.IsNotExist(err) {
			return nil, err
		}
		for i := 1; i <= 3; i++ {
			up := strings.Repeat("../", i) + source
			if d, e := os.ReadFile(up); e == nil {
				data = d
				err = nil
				break
			}
		}
		if err != nil {
			return GetDefaultCatalog(), nil
		}
	}

	var entries []NodeCatalogEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse local catalog JSON: %w", err)
	}

	return entries, nil
}

func GetDefaultCatalog() []NodeCatalogEntry {
	return []NodeCatalogEntry{
		{
			ID:          "eu-vat-calculator",
			Name:        "EU VAT & Tax Calculator",
			Version:     "1.0.0",
			Category:    "Finance",
			Icon:        "calculator",
			Description: "Calculates standard VAT rates for European transactions based on country code.",
			InputsSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"amount", "country_code"},
				"properties": map[string]interface{}{
					"amount": map[string]interface{}{
						"type":        "string",
						"title":       "Net Amount",
						"description": "Blackboard expression or literal number",
						"default":     "blackboard.amount",
					},
					"country_code": map[string]interface{}{
						"type":    "string",
						"title":   "Country Code",
						"enum":    []string{"DE", "AT", "FR", "IT", "ES", "NL"},
						"default": "DE",
					},
					"output_field": map[string]interface{}{
						"type":    "string",
						"title":   "Result Blackboard Field",
						"default": "tax_calculation",
					},
				},
			},
			CodeTemplate: "// EU VAT & Tax Calculator\nconst rates = { DE: 0.19, AT: 0.20, FR: 0.20, IT: 0.22, ES: 0.21, NL: 0.21 };\nconst rate = rates[inputs.country_code] || 0.19;\nconst net = Number(inputs.amount);\nconst vat = Math.round(net * rate * 100) / 100;\nbb[inputs.output_field || 'tax_calculation'] = {\n  net: net,\n  vat_rate: rate,\n  vat_amount: vat,\n  gross: Math.round((net + vat) * 100) / 100\n};\ntrigger('done');",
		},
	}
}

