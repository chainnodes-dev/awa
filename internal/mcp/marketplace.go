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

// GetDefaultMarketplace returns the Phaxa-curated list of reliable MCP servers.
func GetDefaultMarketplace() []MarketEntry {
	return []MarketEntry{
		{
			ID:          "brave-search",
			Name:        "Brave Search",
			Description: "High-quality, private web search for real-time information.",
			Icon:        "search",
			Category:    "Web",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-brave-search"},
			EnvVars: []MarketEnvVar{
				{Name: "BRAVE_API_KEY", Description: "Brave Search API Key", Required: true},
			},
			IsVerified: true,
		},
		{
			ID:          "wikipedia",
			Name:        "Wikipedia",
			Description: "Search and retrieve Wikipedia articles.",
			Icon:        "book",
			Category:    "Knowledge",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-wikipedia"},
			IsVerified:  true,
		},
		{
			ID:          "google-drive",
			Name:        "Google Drive",
			Description: "Access and manage files in Google Drive.",
			Icon:        "hard-drive",
			Category:    "Productivity",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-google-drive"},
			IsVerified:  true,
		},
		{
			ID:          "google-sheets",
			Name:        "Google Sheets",
			Description: "Manage Google Sheets spreadsheets.",
			Icon:        "table",
			Category:    "Productivity",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-google-sheets"},
			IsVerified:  true,
		},
		{
			ID:          "github",
			Name:        "GitHub",
			Description: "Manage issues, PRs, and search code across repositories.",
			Icon:        "github",
			Category:    "Development",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-github"},
			EnvVars: []MarketEnvVar{
				{Name: "GITHUB_PERSONAL_ACCESS_TOKEN", Description: "GitHub PAT with repo scope", Required: true},
			},
			IsVerified: true,
		},
		{
			ID:          "postgres",
			Name:        "Postgres",
			Description: "Query and manage PostgreSQL databases.",
			Icon:        "database",
			Category:    "Databases",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-postgres"},
			EnvVars: []MarketEnvVar{
				{Name: "DATABASE_URL", Description: "Postgres Connection String", Required: true},
			},
			IsVerified: true,
		},
		{
			ID:          "slack",
			Name:        "Slack",
			Description: "Read and post messages to your team's Slack channels.",
			Icon:        "message-square",
			Category:    "Communication",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-slack"},
			EnvVars: []MarketEnvVar{
				{Name: "SLACK_BOT_TOKEN", Description: "Slack Bot User OAuth Token", Required: true},
			},
			IsVerified: true,
		},
		{
			ID:          "sec-edgar",
			Name:        "SEC Edgar",
			Description: "Search and retrieve official SEC filings for US public companies.",
			Icon:        "trending-up",
			Category:    "Finance",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-sec-edgar"},
			IsVerified:  true,
		},
		{
			ID:          "weather",
			Name:        "Weather",
			Description: "Real-time weather data and forecasts.",
			Icon:        "cloud-sun",
			Category:    "Utilities",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-weather"},
			EnvVars: []MarketEnvVar{
				{Name: "OPENWEATHERMAP_API_KEY", Description: "OpenWeatherMap API Key", Required: true},
			},
			IsVerified: true,
		},
	}
}
