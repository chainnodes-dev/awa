package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/store"
)

// ListUsers returns all users. Admin only.
// GET /api/v1/users?limit=100&offset=0
func (h *Handlers) ListUsers(c *gin.Context) {
	filter := store.UserFilter{
		Limit:  queryInt(c, "limit", 100),
		Offset: queryInt(c, "offset", 0),
	}
	users, err := h.store.ListUsers(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

// CreateUser adds a new user. Admin only.
// POST /api/v1/users
func (h *Handlers) CreateUser(c *gin.Context) {
	if err := h.RequireFeature(c.Request.Context(), "user_management"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"     binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Prevent non-super_admin callers from creating super_admin accounts.
	if auth.Role(body.Role) == auth.RoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot create super_admin via user API"})
		return
	}
	if !auth.ValidRole(body.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin, editor, runner, operator, or viewer"})
		return
	}

	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
		return
	}

	claims := auth.ClaimsFrom(c)
	u := &store.User{
		TenantID:     claims.TenantID,
		Username:     body.Username,
		PasswordHash: hash,
		Role:         body.Role,
	}
	if err := h.store.CreateUser(c.Request.Context(), u); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, u)
}

// UpdateUserRole changes a user's role. Admin only.
// PUT /api/v1/users/:id/role
func (h *Handlers) UpdateUserRole(c *gin.Context) {
	if err := h.RequireFeature(c.Request.Context(), "user_management"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	var body struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !auth.ValidRole(body.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin, editor, runner, operator, or viewer"})
		return
	}

	id := c.Param("id")
	if err := h.store.UpdateUserRole(c.Request.Context(), id, body.Role); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// DeleteUser removes a user. Admin only.
// DELETE /api/v1/users/:id
func (h *Handlers) DeleteUser(c *gin.Context) {
	if err := h.RequireFeature(c.Request.Context(), "user_management"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")
	// Prevent admins from deleting themselves.
	if claims := auth.ClaimsFrom(c); claims != nil && claims.UserID == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete your own account"})
		return
	}
	if err := h.store.DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetMe returns the currently authenticated user's profile.
// GET /api/v1/users/me
func (h *Handlers) GetMe(c *gin.Context) {
	claims := auth.ClaimsFrom(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	user, err := h.store.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}
