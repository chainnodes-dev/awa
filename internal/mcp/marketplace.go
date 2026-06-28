package mcp

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

	"gopkg.in/yaml.v3"
)

// MarketEntry represents a curated MCP server in the marketplace.
type MarketEntry struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Icon        string            `json:"icon,omitempty"`
	Author      string            `json:"author,omitempty"`
	Category    string            `json:"category,omitempty"`
	Transport   string            `json:"transport"`
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	URL         string            `json:"url,omitempty"`
	EnvVars     []MarketEnvVar    `json:"env_vars,omitempty"`
	IsVerified  bool              `json:"is_verified"`
	DocURL      string            `json:"doc_url,omitempty"`
}

type MarketEnvVar struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// FetchMarketplace retrieves the curated list of MCP servers from a remote URL or local file.
func FetchMarketplace(ctx context.Context, source string) ([]MarketEntry, error) {
	if source == "" {
		return GetDefaultMarketplace(), nil
	}

	// Try parsing as raw JSON if it looks like a JSON array
	if strings.HasPrefix(strings.TrimSpace(source), "[") {
		var entries []MarketEntry
		if err := json.Unmarshal([]byte(source), &entries); err == nil {
			return entries, nil
		}
	}

	var data []byte
	var err error

	// Try local file first if it doesn't look like a URL
	if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		data, err = os.ReadFile(source)
	}

	if err != nil || data == nil {
		// Try as URL
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch marketplace: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("marketplace returned status: %d", resp.StatusCode)
		}
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	var entries []MarketEntry
	if filepath.Ext(source) == ".yaml" || filepath.Ext(source) == ".yml" {
		var wrapper struct {
			MCPServers []struct {
				Name        string   `yaml:"name"`
				Description string   `yaml:"description"`
				EnvVar      string   `yaml:"env_var"`
				EnvVars     []string `yaml:"env_vars"`
				Transport   string   `yaml:"transport"`
				Command     string   `yaml:"command"`
				Args        []string `yaml:"args"`
				URL         string   `yaml:"url"`
				Icon        string   `yaml:"icon"`
				Category    string   `yaml:"category"`
				DocURL      string   `yaml:"doc_url"`
				Author      string   `yaml:"author"`
			} `yaml:"mcp_servers"`
		}
		if err := yaml.Unmarshal(data, &wrapper); err != nil {
			return nil, fmt.Errorf("failed to decode YAML marketplace data: %w", err)
		}
		for _, e := range wrapper.MCPServers {
			id := strings.ToLower(strings.ReplaceAll(e.Name, " ", "-"))
			entry := MarketEntry{
				ID:          id,
				Name:        e.Name,
				Description: e.Description,
				Transport:   e.Transport,
				Command:     e.Command,
				Args:        e.Args,
				URL:         e.URL,
				Icon:        e.Icon,
				Category:    e.Category,
				DocURL:      e.DocURL,
				Author:      e.Author,
				IsVerified:  true,
			}
			if entry.Transport == "" {
				entry.Transport = "stdio"
			}
			if entry.Transport == "stdio" {
				if entry.Command == "" {
					entry.Command = "npx"
				}
				if len(entry.Args) == 0 && entry.Command == "npx" {
					entry.Args = []string{"-y", "@modelcontextprotocol/server-" + id}
				}
			}
			if e.EnvVar != "" {
				entry.EnvVars = append(entry.EnvVars, MarketEnvVar{Name: e.EnvVar, Required: true})
			}
			for _, ev := range e.EnvVars {
				entry.EnvVars = append(entry.EnvVars, MarketEnvVar{Name: ev, Required: true})
			}
			entries = append(entries, entry)
		}
	} else {
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil, fmt.Errorf("failed to decode marketplace data: %w", err)
		}
		// Apply defaults to JSON entries too
		for i := range entries {
			if entries[i].Transport == "" {
				entries[i].Transport = "stdio"
			}
			if entries[i].Transport == "stdio" {
				if entries[i].Command == "" {
					entries[i].Command = "npx"
				}
				if len(entries[i].Args) == 0 && entries[i].Command == "npx" {
					entries[i].Args = []string{"-y", "@modelcontextprotocol/server-" + entries[i].ID}
				}
			}
		}
	}

	return entries, nil
}

// GetDefaultMarketplace returns the Chain Nodes-curated list of reliable MCP servers.
func GetDefaultMarketplace() []MarketEntry {
	// Try loading dynamically from configs/mcp-market.json first!
	if data, err := os.ReadFile("configs/mcp-market.json"); err == nil {
		var entries []MarketEntry
		if err := json.Unmarshal(data, &entries); err == nil {
			return entries
		}
	}
	return nil
}
