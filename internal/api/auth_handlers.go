package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/store"
)

// Login authenticates a user and returns an access token + refresh token.
// POST /api/v1/auth/login
// Body: { "username": "...", "password": "...", "tenant_slug": "..." }
// tenant_slug defaults to "default" if omitted, for backward compatibility.
func (h *Handlers) Login(c *gin.Context) {
	var body struct {
		Username   string `json:"username" binding:"required"`
		Password   string `json:"password" binding:"required"`
		TenantSlug string `json:"tenant_slug"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.TenantSlug == "" {
		body.TenantSlug = "default"
	}

	tenant, err := h.store.GetTenantBySlug(c.Request.Context(), body.TenantSlug)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Scope the user lookup to this tenant.
	tenantCtx := store.WithTenantID(c.Request.Context(), tenant.ID)
	user, err := h.store.GetUserByUsername(tenantCtx, body.Username)
	if err != nil {
		// Keep the error message generic to prevent username enumeration.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, body.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	accessToken, err := h.jwtSvc.GenerateAccessToken(user.ID, user.TenantID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
		return
	}

	rawRefresh, tokenHash, err := auth.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate refresh token"})
		return
	}

	rt := &store.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(auth.RefreshTokenTTL()),
	}
	if err := h.store.CreateRefreshToken(c.Request.Context(), rt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not store refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": rawRefresh,
		"token_type":    "Bearer",
		"expires_in":    int(auth.AccessTokenTTL().Seconds()),
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"role":      user.Role,
			"tenant_id": user.TenantID,
		},
	})
}

// RefreshToken issues a new access token given a valid refresh token.
// POST /api/v1/auth/refresh
func (h *Handlers) RefreshToken(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash := auth.HashToken(body.RefreshToken)
	rt, err := h.store.GetRefreshTokenByHash(c.Request.Context(), hash)
	if err != nil || rt.Revoked || rt.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	user, err := h.store.GetUserByID(c.Request.Context(), rt.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Revoke the used refresh token (rotation: issue a new one).
	_ = h.store.RevokeRefreshToken(c.Request.Context(), rt.ID)

	accessToken, err := h.jwtSvc.GenerateAccessToken(user.ID, user.TenantID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
		return
	}

	rawRefresh, tokenHash, err := auth.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate refresh token"})
		return
	}
	newRT := &store.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(auth.RefreshTokenTTL()),
	}
	if err := h.store.CreateRefreshToken(c.Request.Context(), newRT); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not store refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": rawRefresh,
		"token_type":    "Bearer",
		"expires_in":    int(auth.AccessTokenTTL().Seconds()),
	})
}

// Logout revokes all refresh tokens for the current user.
// POST /api/v1/auth/logout
func (h *Handlers) Logout(c *gin.Context) {
	claims := auth.ClaimsFrom(c)
	if claims != nil {
		_ = h.store.RevokeAllUserTokens(c.Request.Context(), claims.UserID)
	}
	c.Status(http.StatusNoContent)
}

// GetAuthStatus checks if the platform has been initialized (at least one user exists).
// GET /api/v1/auth/status
func (h *Handlers) GetAuthStatus(c *gin.Context) {
	initialized, err := h.store.HasAnyUser(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check auth status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"initialized": initialized})
}

// SetupAdmin creates the first tenant and admin user.
// POST /api/v1/auth/setup
func (h *Handlers) SetupAdmin(c *gin.Context) {
	// 1. Ensure no users exist.
	initialized, err := h.store.HasAnyUser(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check auth status"})
		return
	}
	if initialized {
		c.JSON(http.StatusForbidden, gin.H{"error": "platform already initialized"})
		return
	}

	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. Ensure default tenant exists.
	hasTenant, err := h.store.HasAnyTenant(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check tenant status"})
		return
	}

	var tenantID string
	if !hasTenant {
		t := &store.Tenant{
			Name: "Default Tenant",
			Slug: "default",
		}
		if err := h.store.CreateTenant(c.Request.Context(), t); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create default tenant"})
			return
		}
		tenantID = t.ID
	} else {
		t, err := h.store.GetTenantBySlug(c.Request.Context(), "default")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get default tenant"})
			return
		}
		tenantID = t.ID
	}

	// 3. Create the super admin user.
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
		return
	}

	u := &store.User{
		TenantID:     tenantID,
		Username:     body.Username,
		PasswordHash: hash,
		Role:         string(auth.RoleSuperAdmin),
	}
	if err := h.store.CreateUser(c.Request.Context(), u); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create admin user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "platform initialized successfully"})
}
