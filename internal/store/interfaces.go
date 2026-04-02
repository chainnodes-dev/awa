package store

import (
	"context"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// WorkflowStore manages workflow definitions.
type WorkflowStore interface {
	SaveDefinition(ctx context.Context, def *asmtypes.WorkflowDef, yamlSource string) error
	GetDefinition(ctx context.Context, name, version string) (*asmtypes.WorkflowDef, string, error)
	ListDefinitions(ctx context.Context) ([]*asmtypes.WorkflowDef, error)
	DeleteDefinition(ctx context.Context, name, version string) error
}

// RunStore manages workflow run instances.
type RunStore interface {
	CreateRun(ctx context.Context, run *asmtypes.WorkflowRun) error
	GetRun(ctx context.Context, id string) (*asmtypes.WorkflowRun, error)
	UpdateRun(ctx context.Context, run *asmtypes.WorkflowRun) error
	ListRuns(ctx context.Context, filter RunFilter) ([]*asmtypes.WorkflowRun, error)

	RecordTransition(ctx context.Context, record *asmtypes.TransitionRecord) error
	ListTransitions(ctx context.Context, runID string) ([]*asmtypes.TransitionRecord, error)
}

// HITLStore manages human-in-the-loop requests.
type HITLStore interface {
	CreateHITL(ctx context.Context, req *asmtypes.HITLRequest) error
	GetHITL(ctx context.Context, runID string) (*asmtypes.HITLRequest, error)
	ResolveHITL(ctx context.Context, runID, resolution, resolver string) error
	ListPendingHITL(ctx context.Context) ([]*asmtypes.HITLRequest, error)
}

// Store is the combined store interface.
type Store interface {
	WorkflowStore
	RunStore
	HITLStore
	UserStore
	RefreshTokenStore
	Close() error
}

type RunFilter struct {
	WorkflowName string
	Status       asmtypes.RunStatus
	Limit        int
	Offset       int
}
