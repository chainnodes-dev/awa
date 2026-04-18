package store

import (
	"context"
	"time"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// Skill is a registered, reusable capability backed by a workflow implementation.
// Skills have a typed I/O contract (inputs / outputs) and declare the MCP server
// capabilities they require.
type Skill struct {
	ID             string                    `json:"id"`
	TenantID       string                    `json:"tenant_id"`
	Name           string                    `json:"name"`
	Description    string                    `json:"description"`
	WorkflowRef    string                    `json:"workflow_ref"`
	Inputs         []asmtypes.SkillPortDef   `json:"inputs"`
	Outputs        []asmtypes.SkillPortDef   `json:"outputs"`
	Capabilities   []asmtypes.CapabilityDecl `json:"capabilities"`
	// DependentsCount is populated by ListSkills and GetSkill — it is NOT stored in
	// the skills table. It counts how many other skills invoke this skill via skill_call.
	DependentsCount int                      `json:"dependents_count"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
}

// SkillStore manages the skill registry.
type SkillStore interface {
	// CreateSkill persists a new skill entry. ID is auto-assigned if empty.
	CreateSkill(ctx context.Context, s *Skill) error
	// GetSkill retrieves a skill by name for the tenant in ctx.
	// DependentsCount is populated on the returned skill.
	GetSkill(ctx context.Context, name string) (*Skill, error)
	// ListSkills returns all skills for the tenant in ctx, ordered by name.
	// DependentsCount is populated on every returned entry.
	ListSkills(ctx context.Context) ([]*Skill, error)
	// UpdateSkill replaces the mutable fields of an existing skill.
	UpdateSkill(ctx context.Context, s *Skill) error
	// DeleteSkill removes a skill by name for the tenant in ctx.
	DeleteSkill(ctx context.Context, name string) error

	// UpsertSkillDependencies replaces all dependency edges for the given skill.
	// dependsOnIDs is the complete set of skill IDs that skillID calls via skill_call.
	// Passing an empty or nil slice removes all edges.
	UpsertSkillDependencies(ctx context.Context, skillID string, dependsOnIDs []string) error
	// ListSkillDependents returns all skills that invoke the named skill via skill_call.
	ListSkillDependents(ctx context.Context, name string) ([]*Skill, error)
}
