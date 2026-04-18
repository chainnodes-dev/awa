package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// ListSkills returns all skills for the caller's tenant, ordered by name.
func (h *Handlers) ListSkills(c *gin.Context) {
	ctx := c.Request.Context()
	skills, err := h.store.ListSkills(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if skills == nil {
		skills = []*store.Skill{}
	}
	c.JSON(http.StatusOK, skills)
}

// GetSkill returns a single skill by name.
func (h *Handlers) GetSkill(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")
	sk, err := h.store.GetSkill(ctx, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sk)
}

// CreateSkill registers a new skill in the registry.
func (h *Handlers) CreateSkill(c *gin.Context) {
	ctx := c.Request.Context()
	claims := auth.ClaimsFrom(c)

	var body struct {
		Name         string                    `json:"name"         binding:"required"`
		Description  string                    `json:"description"`
		WorkflowRef  string                    `json:"workflow_ref"`
		Inputs       []asmtypes.SkillPortDef   `json:"inputs"`
		Outputs      []asmtypes.SkillPortDef   `json:"outputs"`
		Capabilities []asmtypes.CapabilityDecl `json:"capabilities"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sk := &store.Skill{
		TenantID:     claims.TenantID,
		Name:         body.Name,
		Description:  body.Description,
		WorkflowRef:  body.WorkflowRef,
		Inputs:       body.Inputs,
		Outputs:      body.Outputs,
		Capabilities: body.Capabilities,
	}
	if sk.Inputs == nil {
		sk.Inputs = []asmtypes.SkillPortDef{}
	}
	if sk.Outputs == nil {
		sk.Outputs = []asmtypes.SkillPortDef{}
	}
	if sk.Capabilities == nil {
		sk.Capabilities = []asmtypes.CapabilityDecl{}
	}

	if err := h.store.CreateSkill(ctx, sk); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	h.syncSkillDependencies(ctx, sk)
	c.JSON(http.StatusCreated, sk)
}

// UpdateSkill replaces the mutable fields of an existing skill.
func (h *Handlers) UpdateSkill(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")

	var body struct {
		Description  string                    `json:"description"`
		WorkflowRef  string                    `json:"workflow_ref"`
		Inputs       []asmtypes.SkillPortDef   `json:"inputs"`
		Outputs      []asmtypes.SkillPortDef   `json:"outputs"`
		Capabilities []asmtypes.CapabilityDecl `json:"capabilities"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sk := &store.Skill{
		Name:         name,
		Description:  body.Description,
		WorkflowRef:  body.WorkflowRef,
		Inputs:       body.Inputs,
		Outputs:      body.Outputs,
		Capabilities: body.Capabilities,
	}
	if sk.Inputs == nil {
		sk.Inputs = []asmtypes.SkillPortDef{}
	}
	if sk.Outputs == nil {
		sk.Outputs = []asmtypes.SkillPortDef{}
	}
	if sk.Capabilities == nil {
		sk.Capabilities = []asmtypes.CapabilityDecl{}
	}

	if err := h.store.UpdateSkill(ctx, sk); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Return the updated record (also populates DependentsCount).
	updated, err := h.store.GetSkill(ctx, name)
	if err != nil {
		c.JSON(http.StatusOK, sk)
		return
	}
	h.syncSkillDependencies(ctx, updated)
	// Re-fetch to pick up any DependentsCount change from the sync.
	if fresh, err := h.store.GetSkill(ctx, name); err == nil {
		updated = fresh
	}
	c.JSON(http.StatusOK, updated)
}

// DeleteSkill removes a skill from the registry.
// Returns 409 with a list of dependent skills when callers exist, unless
// the request includes ?force=true.
func (h *Handlers) DeleteSkill(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")

	if c.Query("force") != "true" {
		deps, err := h.store.ListSkillDependents(ctx, name)
		if err == nil && len(deps) > 0 {
			names := make([]string, len(deps))
			for i, d := range deps {
				names[i] = d.Name
			}
			c.JSON(http.StatusConflict, gin.H{
				"error":      "skill is used by other skills; use ?force=true to delete anyway",
				"dependents": names,
			})
			return
		}
	}

	if err := h.store.DeleteSkill(ctx, name); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ListSkillDependents returns skills that invoke the named skill via skill_call.
// GET /api/v1/skills/:name/dependents
func (h *Handlers) ListSkillDependents(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")
	deps, err := h.store.ListSkillDependents(ctx, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if deps == nil {
		deps = []*store.Skill{}
	}
	c.JSON(http.StatusOK, deps)
}

// syncSkillDependencies scans the skill's workflow for skill_call states and records
// the resulting dependency edges. Non-fatal: logs warnings and returns on any error.
func (h *Handlers) syncSkillDependencies(ctx context.Context, skill *store.Skill) {
	if skill.ID == "" {
		return
	}
	if skill.WorkflowRef == "" {
		_ = h.store.UpsertSkillDependencies(ctx, skill.ID, nil)
		return
	}

	def, _, err := h.store.GetLatestDefinition(ctx, skill.WorkflowRef)
	if err != nil {
		slog.Warn("skill deps: workflow not found", "skill", skill.Name, "workflow_ref", skill.WorkflowRef, "error", err)
		return
	}

	var depIDs []string
	for _, state := range def.States {
		if state.Type != asmtypes.StateSkillCall || state.SkillCall == nil {
			continue
		}
		dep, err := h.store.GetSkill(ctx, state.SkillCall.SkillRef)
		if err != nil {
			slog.Warn("skill deps: referenced skill not found", "skill", skill.Name, "ref", state.SkillCall.SkillRef)
			continue
		}
		depIDs = append(depIDs, dep.ID)
	}

	if err := h.store.UpsertSkillDependencies(ctx, skill.ID, depIDs); err != nil {
		slog.Warn("skill deps: upsert failed", "skill", skill.Name, "error", err)
	}
}
