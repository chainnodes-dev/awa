package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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

// FetchMarketplace retrieves the curated list of MCP servers from a remote URL.
// In an enterprise environment, this URL would point to the company's internal registry.
func FetchMarketplace(ctx context.Context, url string) ([]MarketEntry, error) {
	if url == "" {
		// Fallback to internal defaults if no URL is provided
		return GetDefaultMarketplace(), nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

	var entries []MarketEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode marketplace data: %w", err)
	}

	return entries, nil
}

// GetDefaultMarketplace returns the Phaxa-curated list of reliable MCP servers.
func GetDefaultMarketplace() []MarketEntry {
	return []MarketEntry{
		{
			ID:          "sec-edgar",
			Name:        "SEC Edgar",
			Description: "Search and retrieve official SEC filings (10-K, 10-Q, 8-K) for US public companies.",
			Icon:        "trending-up",
			Category:    "Finance",
			Transport:   "stdio",
			Command:     "python3",
			Args:        []string{"-m", "sec_edgar_mcp"},
			DocURL:      "https://github.com/stefanoamorelli/sec-edgar-mcp",
			IsVerified:  true,
		},
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
	}
}
