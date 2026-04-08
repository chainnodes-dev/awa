package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/store"
)

// CreateTenant creates a new tenant.
// POST /api/v1/tenants  (super_admin only)
func (h *Handlers) CreateTenant(c *gin.Context) {
	var body struct {
		Name string `json:"name" binding:"required"`
		Slug string `json:"slug" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t := &store.Tenant{
		Name: body.Name,
		Slug: body.Slug,
	}
	if err := h.store.CreateTenant(c.Request.Context(), t); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

// ListTenants returns all tenants.
// GET /api/v1/tenants  (super_admin only)
func (h *Handlers) ListTenants(c *gin.Context) {
	tenants, err := h.store.ListTenants(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tenants)
}

// GetTenant returns a single tenant by ID.
// GET /api/v1/tenants/:id  (super_admin only)
func (h *Handlers) GetTenant(c *gin.Context) {
	id := c.Param("id")
	t, err := h.store.GetTenant(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, t)
}

// DeleteTenant removes a tenant by ID.
// DELETE /api/v1/tenants/:id  (super_admin only)
func (h *Handlers) DeleteTenant(c *gin.Context) {
	id := c.Param("id")
	if id == store.DefaultTenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete the default tenant"})
		return
	}
	if err := h.store.DeleteTenant(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
