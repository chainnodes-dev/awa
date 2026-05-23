package store

import (
	"context"
	"time"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// DefaultTenantID is the fixed UUID of the built-in "default" tenant.
// Used for first-boot seeding and single-tenant deployments.
const DefaultTenantID = "00000000-0000-0000-0000-000000000001"

// contextKey is an unexported type for context keys in this package.
type contextKey string

const tenantIDKey contextKey = "tenant_id"

// WithTenantID returns a new context carrying the given tenant ID.
// Store methods extract it to scope queries to the correct tenant.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// TenantIDFromContext extracts the tenant ID set by WithTenantID.
// Returns "" if no tenant ID has been set (e.g. internal engine goroutines).
func TenantIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(tenantIDKey).(string)
	return v
}

// DefinitionFilter controls which workflow definitions are returned by ListDefinitions.
type DefinitionFilter struct {
	Limit        int // 0 = no limit
	Offset       int
	ReusableOnly bool // If true, only return workflows marked as reusable
}

// VersionSummary is a lightweight record returned by ListDefinitionVersions.
type VersionSummary struct {
	VersionNumber int       `json:"version_number"`
	Version       string    `json:"version"`       // original text version tag
	CreatedAt     time.Time `json:"created_at"`
}

// WorkflowStore manages workflow definitions.
type WorkflowStore interface {
	// SaveDefinition creates a new version (auto-incremented version_number).
	SaveDefinition(ctx context.Context, def *asmtypes.WorkflowDef, yamlSource string) error
	// UpdateDefinition overwrites an existing version in place (no new version created).
	UpdateDefinition(ctx context.Context, def *asmtypes.WorkflowDef, yamlSource string) error
	GetDefinition(ctx context.Context, name, version string) (*asmtypes.WorkflowDef, string, error)
	GetDefinitionByVersion(ctx context.Context, name string, versionNumber int) (*asmtypes.WorkflowDef, string, error)
	GetLatestDefinition(ctx context.Context, name string) (*asmtypes.WorkflowDef, string, error)
	ListDefinitions(ctx context.Context, filter DefinitionFilter) ([]*asmtypes.WorkflowDef, error)
	ListDefinitionVersions(ctx context.Context, name string) ([]VersionSummary, error)
	DeleteDefinition(ctx context.Context, name, version string) error
	CountDefinitions(ctx context.Context) (int, error)
}

// RunStore manages workflow run instances.
type RunStore interface {
	CreateRun(ctx context.Context, run *asmtypes.WorkflowRun) error
	GetRun(ctx context.Context, id string) (*asmtypes.WorkflowRun, error)
	UpdateRun(ctx context.Context, run *asmtypes.WorkflowRun) error
	DeleteRun(ctx context.Context, id string) error
	ListRuns(ctx context.Context, filter RunFilter) ([]*asmtypes.WorkflowRun, error)

	RecordTransition(ctx context.Context, record *asmtypes.TransitionRecord) error
	ListTransitions(ctx context.Context, runID string) ([]*asmtypes.TransitionRecord, error)
	CountRuns(ctx context.Context, filter RunFilter) (int, error)
	GetRunStats(ctx context.Context) (map[string]int, error)
}

// HITLFilter controls which HITL requests are returned.
type HITLFilter struct {
	TenantID string
	Assignee string
	Resolved *bool // nil = any, true = only resolved, false = only pending
}

// HITLStore manages human-in-the-loop requests.
type HITLStore interface {
	CreateHITL(ctx context.Context, req *asmtypes.HITLRequest) error
	GetHITL(ctx context.Context, runID string) (*asmtypes.HITLRequest, error)
	ResolveHITL(ctx context.Context, runID, resolution, resolver string) error
	ListHITLs(ctx context.Context, filter HITLFilter) ([]*asmtypes.HITLRequest, error)
}

// SystemSettingsStore manages platform-wide configuration.
type SystemSettingsStore interface {
	GetSystemSetting(ctx context.Context, key string) (string, error)
	SetSystemSetting(ctx context.Context, key, value string) error
}

// Store is the combined store interface.
type Store interface {
	WorkflowStore
	RunStore
	HITLStore
	UserStore
	RefreshTokenStore
	TenantStore
	MCPServerStore
	LLMConfigStore
	EventSubscriptionStore
	APIKeyStore
	SystemSettingsStore
	Close() error
}

// RunFilter controls which runs are returned by ListRuns.
// TenantID is required for tenant-scoped queries; leave empty only for
// internal/administrative operations that need cross-tenant access.
type RunFilter struct {
	TenantID     string
	WorkflowName string
	Status       asmtypes.RunStatus
	CurrentState string
	StartedFrom  *time.Time // inclusive lower bound on started_at
	StartedTo    *time.Time // inclusive upper bound on started_at
	Limit        int
	Offset       int
}
