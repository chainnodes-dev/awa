package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
)

// OIDCLogin handles the initial redirect to the external identity provider.
func (h *Handlers) OIDCLogin(c *gin.Context) {
	tenantSlug := c.Query("tenant")
	if tenantSlug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant query parameter required"})
		return
	}

	tenant, err := h.store.GetTenantBySlug(c.Request.Context(), tenantSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	cfg, err := h.store.GetTenantIDPConfig(c.Request.Context(), tenant.ID)
	if err != nil || !cfg.Active {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OIDC not configured for this tenant"})
		return
	}

	// Enterprise Gating: SSO requires a valid license.
	if err := h.RequireFeature(c.Request.Context(), "sso"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	provider := auth.NewOIDCProvider((*auth.IDPConfig)(cfg))
	disc, err := provider.Discover(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("discovery failed: %v", err)})
		return
	}

	// Redirect to IdP
	redirectURI := fmt.Sprintf("%s/api/v1/auth/oidc/callback", c.Request.Host) // Need proper public URL
	authURL := fmt.Sprintf("%s?client_id=%s&response_type=code&scope=openid email profile&redirect_uri=%s&state=%s",
		disc.AuthorizationEndpoint, cfg.ClientID, redirectURI, tenantSlug)

	c.Redirect(http.StatusFound, authURL)
}

// OIDCCallback handles the return from the external identity provider.
func (h *Handlers) OIDCCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state") // we use state as the tenant slug

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code or state"})
		return
	}

	tenant, err := h.store.GetTenantBySlug(c.Request.Context(), state)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	cfg, err := h.store.GetTenantIDPConfig(c.Request.Context(), tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load IDP config"})
		return
	}

	// Enterprise Gating: SSO requires a valid license.
	if err := h.RequireFeature(c.Request.Context(), "sso"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	provider := auth.NewOIDCProvider((*auth.IDPConfig)(cfg))
	tokens, err := provider.ExchangeToken(c.Request.Context(), code, fmt.Sprintf("%s/api/v1/auth/oidc/callback", c.Request.Host))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("token exchange failed: %v", err)})
		return
	}

	idToken, ok := tokens["id_token"].(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no id_token in response"})
		return
	}

	claims, err := provider.VerifyIDToken(idToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to verify ID token"})
		return
	}

	// Map OIDC user to Phaxa user.
	externalEmail, _ := claims["email"].(string)
	if externalEmail == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "OIDC token missing email claim"})
		return
	}

	// 1. Find or create user in the database.
	// 2. Issue Phaxa JWT access token.
	
	// Stub for demo: generate a token.
	token, err := h.jwtSvc.GenerateAccessToken("oidc-"+externalEmail, tenant.ID, externalEmail, string(auth.RoleOperator))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"tenant":       tenant.Slug,
	})
}
