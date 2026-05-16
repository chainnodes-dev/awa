package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// MCPTool represents a tool definition provided by an MCP server.
type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// MCPServer is a registered MCP server entry managed via the platform API.
// When the mcp_servers table has entries for a tenant, they take precedence
// over the static mcp_registry.yaml file.
type MCPServer struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	// Name is the logical identifier used in workflow YAML agent configs.
	Name string `json:"name"`
	// Transport defines how to connect: "sse" (default) or "stdio" (local command).
	Transport string `json:"transport"`
	// URL is the HTTP endpoint (used if Transport == "sse").
	URL string `json:"url,omitempty"`
	// Command is the local executable to run (used if Transport == "stdio").
	Command string `json:"command,omitempty"`
	// Args are the CLI arguments for the Command (used if Transport == "stdio").
	Args []string `json:"args,omitempty"`
	// EnvVars is a map of environment variable keys and values.
	// These will be injected into the MCP server process on startup.
	EnvVars     map[string]string `json:"env_vars,omitempty"`
	Description string            `json:"description"`
	// DocURL is an optional link to the server's documentation.
	DocURL string `json:"doc_url,omitempty"`
	// Tools is the list of discovered tools cached from the server.
	Tools     []MCPTool `json:"tools,omitempty"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// MCPServerStore manages the dynamic MCP server registry.
type MCPServerStore interface {
	// CreateMCPServer persists a new entry. ID is populated if empty.
	CreateMCPServer(ctx context.Context, srv *MCPServer) error
	// GetMCPServer retrieves an entry by its UUID.
	GetMCPServer(ctx context.Context, id string) (*MCPServer, error)
	// ListMCPServers returns all entries for the tenant in ctx.
	ListMCPServers(ctx context.Context) ([]*MCPServer, error)
	// UpdateMCPServer replaces mutable fields (URL, EnvVar, Description).
	UpdateMCPServer(ctx context.Context, srv *MCPServer) error
	// DeleteMCPServer removes an entry by UUID.
	DeleteMCPServer(ctx context.Context, id string) error
	// HasAnyMCPServer returns true if at least one MCP server is registered for the tenant.
	HasAnyMCPServer(ctx context.Context) (bool, error)

	// RecordMCPCall persists an audit log of a communication event.
	RecordMCPCall(ctx context.Context, log *asmtypes.MCPAuditLog) error
	// ListMCPCalls returns the audit trail for a specific run.
	ListMCPCalls(ctx context.Context, runID string) ([]*asmtypes.MCPAuditLog, error)
}
