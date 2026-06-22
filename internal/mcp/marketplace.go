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

// GetDefaultMarketplace returns the Chain Nodes-curated list of reliable MCP servers.
func GetDefaultMarketplace() []MarketEntry {
	return []MarketEntry{
		{
			ID:          "brave-search",
			Name:        "Brave Search",
			Description: "High-quality, private web search for real-time information.",
			Icon:        "search",
			Category:    "Search",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-brave-search"},
			EnvVars: []MarketEnvVar{
				{Name: "BRAVE_API_KEY", Description: "Brave Search API Key", Required: true},
			},
			IsVerified: true,
			DocURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/brave-search",
		},
		{
			ID:          "duckduckgo",
			Name:        "DuckDuckGo Search",
			Description: "DuckDuckGo web search and content retrieval (no API key required).",
			Icon:        "search",
			Category:    "Search",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "duckduckgo-mcp-server"},
			IsVerified:  true,
			DocURL:     "https://github.com/zhsama/duckduckgo-mcp-server",
		},
		{
			ID:          "open-websearch",
			Name:        "OpenWebSearch",
			Description: "Multi-engine web search (Bing, DuckDuckGo, Brave, etc.) and scraping (no API key required).",
			Icon:        "search",
			Category:    "Search",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "open-websearch"},
			IsVerified:  true,
			DocURL:     "https://github.com/aas-ee/open-websearch",
		},
		{
			ID:          "wikipedia",
			Name:        "Wikipedia",
			Description: "Search and retrieve full-text articles from Wikipedia.",
			Icon:        "book",
			Category:    "Knowledge",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@shelm/wikipedia-mcp-server"},
			IsVerified:  true,
			DocURL:     "https://github.com/shelm/wikipedia-mcp-server",
		},
		{
			ID:          "notion",
			Name:        "Notion",
			Description: "Access and manage Notion pages, databases, and blocks.",
			Icon:        "book-open",
			Category:    "Productivity",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@notionhq/notion-mcp-server"},
			EnvVars: []MarketEnvVar{
				{Name: "NOTION_API_KEY", Description: "Notion API Key", Required: true},
			},
			IsVerified: true,
			DocURL:     "https://github.com/notionhq/notion-mcp-server",
		},
		{
			ID:          "linear",
			Name:        "Linear",
			Description: "Manage issues and projects in Linear.",
			Icon:        "check-square",
			Category:    "Productivity",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "mcp-linear"},
			EnvVars: []MarketEnvVar{
				{Name: "LINEAR_API_KEY", Description: "Linear API Key", Required: true},
			},
			IsVerified: true,
			DocURL:     "https://github.com/linear/mcp-linear",
		},
		{
			ID:          "airtable",
			Name:        "Airtable",
			Description: "Read and write data to Airtable bases.",
			Icon:        "grid",
			Category:    "Productivity",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "airtable-mcp-server"},
			EnvVars: []MarketEnvVar{
				{Name: "AIRTABLE_API_KEY", Description: "Airtable API Key", Required: true},
			},
			IsVerified: true,
			DocURL:     "https://github.com/domdomegg/airtable-mcp-server",
		},
		{
			ID:          "github",
			Name:        "GitHub",
			Description: "Automate GitHub tasks: PRs, issues, and code search.",
			Icon:        "github",
			Category:    "Development",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-github"},
			EnvVars: []MarketEnvVar{
				{Name: "GITHUB_PERSONAL_ACCESS_TOKEN", Description: "GitHub Personal Access Token", Required: true},
			},
			IsVerified: true,
			DocURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/github",
		},
		{
			ID:          "everything",
			Name:        "Everything",
			Description: "Anthropic's official capability test server showcasing all MCP features.",
			Icon:        "cpu",
			Category:    "Development",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-everything"},
			IsVerified:  true,
			DocURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/everything",
		},
		{
			ID:          "postgres",
			Name:        "Postgres",
			Description: "Connect to and query PostgreSQL databases.",
			Icon:        "database",
			Category:    "Databases",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-postgres"},
			EnvVars: []MarketEnvVar{
				{Name: "DATABASE_URL", Description: "PostgreSQL Database URL", Required: true},
			},
			IsVerified: true,
			DocURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/postgres",
		},
		{
			ID:          "redis",
			Name:        "Redis",
			Description: "Interact with Redis key-value stores.",
			Icon:        "database",
			Category:    "Databases",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-redis"},
			EnvVars: []MarketEnvVar{
				{Name: "REDIS_URL", Description: "Redis Connection URL", Required: false},
			},
			IsVerified: true,
			DocURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/redis",
		},
		{
			ID:          "slack",
			Name:        "Slack",
			Description: "Send messages, list channels, and interact with Slack workspaces.",
			Icon:        "message-square",
			Category:    "Communication",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-slack"},
			EnvVars: []MarketEnvVar{
				{Name: "SLACK_BOT_TOKEN", Description: "Slack Bot Token", Required: true},
			},
			IsVerified: true,
			DocURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/slack",
		},
		{
			ID:          "fetch",
			Name:        "Fetch",
			Description: "Download web content safely and convert it to Markdown.",
			Icon:        "globe",
			Category:    "Utilities",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@infoinlet/mcp-fetch"},
			IsVerified:  true,
			DocURL:     "https://github.com/infoinlet/mcp-fetch",
		},
		{
			ID:          "filesystem",
			Name:        "Filesystem",
			Description: "Access local files (restricted to ./data).",
			Icon:        "file-text",
			Category:    "Utilities",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-filesystem", "./data"},
			IsVerified:  true,
			DocURL:     "https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem",
		},
		{
			ID:          "fast-filesystem",
			Name:        "Fast Filesystem",
			Description: "High-performance filesystem access (restricted to ./data).",
			Icon:        "file-text",
			Category:    "Utilities",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "fast-filesystem-mcp", "./data"},
			IsVerified:  true,
			DocURL:     "https://github.com/efforthye/fast-filesystem-mcp",
		},
		{
			ID:          "firecrawl",
			Name:        "Firecrawl",
			Description: "Web scraping, deep crawling, and search with structured data extraction.",
			Icon:        "globe",
			Category:    "Utilities",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "firecrawl-mcp"},
			EnvVars: []MarketEnvVar{
				{Name: "FIRECRAWL_API_KEY", Description: "Firecrawl API Key", Required: true},
			},
			IsVerified:  true,
			DocURL:     "https://github.com/firecrawl/firecrawl",
		},
		{
			ID:          "sap-fiori",
			Name:        "SAP Fiori",
			Description: "Tooling for developing and modifying SAP Fiori applications.",
			Icon:        "layout",
			Category:    "Enterprise",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@sap-ux/fiori-mcp-server"},
			IsVerified:  true,
			DocURL:     "https://github.com/SAP/fiori-tools",
		},
		{
			ID:          "openfigi",
			Name:        "OpenFigi",
			Description: "Lookup the Financial Instrument Global Identifier (FIGI) for a stock ticker.",
			Icon:        "trending-up",
			Category:    "Finance",
			Transport:   "sse",
			URL:         "http://localhost:8093",
			EnvVars: []MarketEnvVar{
				{Name: "OPENFIGI_API_KEY", Description: "OpenFIGI API Key (Optional)", Required: false},
			},
			IsVerified:  true,
		},
		{
			ID:          "lei",
			Name:        "LEI (GLEIF)",
			Description: "Search for a company's Legal Entity Identifier (LEI) and official details.",
			Icon:        "award",
			Category:    "Finance",
			Transport:   "sse",
			URL:         "http://localhost:8091",
			IsVerified:  true,
		},
		{
			ID:          "sec-edgar",
			Name:        "SEC EDGAR",
			Description: "Search and retrieve official SEC filings, company facts, and financial facts from SEC EDGAR.",
			Icon:        "file-search",
			Category:    "Finance",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@cyanheads/secedgar-mcp-server"},
			EnvVars: []MarketEnvVar{
				{Name: "EDGAR_USER_AGENT", Description: "User agent string (e.g. 'AppName contact@email.com') required by SEC", Required: true},
			},
			IsVerified:  true,
			DocURL:     "https://www.npmjs.com/package/@cyanheads/secedgar-mcp-server",
		},
	}
}
