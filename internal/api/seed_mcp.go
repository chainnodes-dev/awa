package api

import (
	"context"
	"log/slog"
	"os"

	"github.com/asm-platform/asm/internal/designer"
	"github.com/asm-platform/asm/internal/store"
)

// SeedMCPServers ensures that a core set of MCP servers is registered in the DB.
func SeedMCPServers(s store.Store, tenantID string, registryEntries []designer.MCPServerEntry) {
	ctx := store.WithTenantID(context.Background(), tenantID)

	// 1. Built-in servers
	existing, err := s.ListMCPServers(ctx)
	if err != nil {
		slog.Error("Failed to list existing MCP servers during seeding", "error", err)
		return
	}

	servers := []store.MCPServer{
		// ── Search & Knowledge ──────────────────────────────────────────────
		{
			Name:        "brave-search",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-brave-search"},
			EnvVars:     map[string]string{"BRAVE_API_KEY": ""},
			Description: "Real-time web search and content discovery.",
		},
		{
			Name:        "wikipedia",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-wikipedia"},
			Description: "Search and retrieve full-text articles from Wikipedia.",
		},

		// ── Productivity (Google & Microsoft) ──────────────────────────────
		{
			Name:        "google-sheets",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-google-sheets"},
			Description: "Read, write, and manage Google Sheets spreadsheets.",
		},
		{
			Name:        "google-drive",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-google-drive"},
			Description: "Access and manage files in Google Drive.",
		},
		{
			Name:        "google-calendar",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-google-calendar"},
			Description: "Full access to Google Calendar events and scheduling.",
		},
		{
			Name:        "microsoft-graph",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-microsoft-graph"},
			Description: "Access Microsoft 365 services: Outlook, OneDrive, and Teams.",
		},

		// ── Infrastructure & Cloud ──────────────────────────────────────────
		{
			Name:        "aws-s3",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-aws-s3"},
			Description: "Manage objects and buckets in Amazon S3.",
		},
		{
			Name:        "github",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-github"},
			EnvVars:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": ""},
			Description: "Automate GitHub tasks: PRs, issues, and code search.",
		},

		// ── Databases ──────────────────────────────────────────────────────
		{
			Name:        "postgres",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-postgres"},
			EnvVars:     map[string]string{"DATABASE_URL": ""},
			Description: "Connect to and query PostgreSQL databases.",
		},
		{
			Name:        "mysql",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-mysql"},
			Description: "Connect to and query MySQL or MariaDB databases.",
		},

		// ── Communication & CRM ─────────────────────────────────────────────
		{
			Name:        "slack",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-slack"},
			EnvVars:     map[string]string{"SLACK_BOT_TOKEN": ""},
			Description: "Send messages, list channels, and interact with Slack workspaces.",
		},
		{
			Name:        "jira",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@aashari/mcp-server-atlassian-jira"},
			EnvVars:     map[string]string{"JIRA_URL": "", "JIRA_API_TOKEN": "", "JIRA_EMAIL": ""},
			Description: "Manage Jira projects, issues, and sprints.",
		},
		{
			Name:        "salesforce",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@advanced-communities/salesforce-mcp-server"},
			Description: "Interact with Salesforce CRM: Leads, Accounts, and Opportunities.",
		},
		{
			Name:        "send-mail",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-resend"},
			EnvVars:     map[string]string{"RESEND_API_KEY": ""},
			Description: "Sends transactional emails via Resend.",
		},

		// ── Utilities ──────────────────────────────────────────────────────
		{
			Name:        "fetch",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-fetch"},
			Description: "High-performance web content scraper and Markdown converter for LLMs.",
		},
		{
			Name:        "filesystem",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-filesystem", "./data"},
			Description: "Restricted local file system access (./data).",
		},
	}

	for _, srv := range servers {
		toSeed := srv
		toSeed.TenantID = tenantID
		
		found := false
		for _, ex := range existing {
			if ex.Name == toSeed.Name {
				found = true
				break
			}
		}
		if found {
			continue
		}

		if err := s.CreateMCPServer(ctx, &toSeed); err != nil {
			slog.Error("Failed to seed MCP server", "name", toSeed.Name, "error", err)
		} else {
			slog.Info("Seeded internal MCP server", "name", toSeed.Name)
		}
	}

	// 2. Registry-based servers (from mcp_registry.yaml)
	for _, e := range registryEntries {
		found := false
		for _, ex := range existing {
			if ex.Name == e.Name {
				found = true
				break
			}
		}
		if found {
			continue
		}

		srv := &store.MCPServer{
			TenantID:    tenantID,
			Name:        e.Name,
			EnvVars:     map[string]string{e.EnvVar: ""},
			Description: e.Description,
			CreatedBy:   "seed-registry",
		}

		if e.Transport != "" {
			srv.Transport = e.Transport
		} else if e.EnvVar != "" && (len(e.EnvVar) > 4 && e.EnvVar[len(e.EnvVar)-4:] == "_URL") {
			srv.Transport = "sse"
		} else {
			srv.Transport = "stdio"
		}

		if srv.Transport == "stdio" {
			if e.Command != "" {
				srv.Command = e.Command
			} else {
				srv.Command = "npx"
			}
			if len(e.Args) > 0 {
				srv.Args = e.Args
			} else if srv.Command == "npx" {
				srv.Args = []string{"-y", e.Name}
			}
		} else {
			if e.URL != "" {
				srv.URL = e.URL
			} else if e.EnvVar != "" {
				srv.URL = os.Getenv(e.EnvVar)
			}
		}

		if err := s.CreateMCPServer(ctx, srv); err != nil {
			slog.Warn("MCP registry seed: failed to insert entry", "name", e.Name, "error", err)
		} else {
			slog.Info("Seeded registry MCP server", "name", e.Name)
		}
	}
}
