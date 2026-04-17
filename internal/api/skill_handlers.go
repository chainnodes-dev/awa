package api

import (
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

	// Return the updated record.
	updated, err := h.store.GetSkill(ctx, name)
	if err != nil {
		c.JSON(http.StatusOK, sk)
		return
	}
	c.JSON(http.StatusOK, updated)
}

// DeleteSkill removes a skill from the registry.
func (h *Handlers) DeleteSkill(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")
	if err := h.store.DeleteSkill(ctx, name); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
