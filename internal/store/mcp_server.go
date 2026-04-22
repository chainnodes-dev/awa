package store

import (
	"context"
	"time"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// MCPServer is a registered MCP server entry managed via the platform API.
// When the mcp_servers table has entries for a tenant, they take precedence
// over the static mcp_registry.yaml file.
type MCPServer struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	// Name is the logical identifier used in workflow YAML agent configs.
	Name string `json:"name"`
	// URL is the HTTP endpoint of the MCP server.
	// May be left empty if EnvVar is used instead (for parity with the YAML registry).
	URL string `json:"url,omitempty"`
	// EnvVar is an environment variable key whose value is the server URL.
	// Referenced in workflow YAML as {{ env.ENV_VAR }}.
	EnvVar      string    `json:"env_var,omitempty"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
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
