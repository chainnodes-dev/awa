package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/enterprise"
	"github.com/asm-platform/asm/internal/store"
)

// SetLicenseRequest holds the incoming license token.
type SetLicenseRequest struct {
	Token string `json:"token" binding:"required"`
}

// extractLicenseToken removes PEM wrappers, spaces, and newlines from the token.
func extractLicenseToken(content []byte) string {
	str := strings.TrimSpace(string(content))
	if strings.Contains(str, "-----BEGIN") {
		lines := strings.Split(str, "\n")
		var tokenParts []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "-----BEGIN") || strings.HasPrefix(line, "-----END") || line == "" {
				continue
			}
			tokenParts = append(tokenParts, line)
		}
		return strings.Join(tokenParts, "")
	}
	return str
}

// SetLicense updates the license token for the active tenant.
func (h *Handlers) SetLicense(c *gin.Context) {
	var token string

	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		// Handle file upload
		file, err := c.FormFile("file")
		if err != nil {
			// Fallback to "license" field
			file, err = c.FormFile("license")
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded (use field 'file' or 'license')"})
			return
		}

		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to open file: %v", err)})
			return
		}
		defer src.Close()

		content, err := io.ReadAll(src)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read file: %v", err)})
			return
		}

		token = extractLicenseToken(content)
	} else {
		// Default to JSON body
		var req SetLicenseRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		token = extractLicenseToken([]byte(req.Token))
	}

	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant context missing"})
		return
	}

	// 1. Verify the token first.
	claims, err := h.licenseVerifier.Verify(token)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("invalid license: %v", err)})
		return
	}

	// Ensure the license is actually for THIS tenant.
	if claims.TenantID != "" && claims.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "license tenant mismatch"})
		return
	}

	// 2. Persist to DB.
	if err := h.store.UpdateTenantLicense(c.Request.Context(), tenantID, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 3. Audit Log.
	_ = enterprise.RecordAction(c.Request.Context(), h.store, enterprise.ActionLicenseUpdated, map[string]interface{}{
		"token_received": true,
	})

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "license updated"})
}

// GetLicenseStatus returns the current license tier and metadata for the active tenant.
func (h *Handlers) GetLicenseStatus(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant context missing"})
		return
	}

	tenant, err := h.store.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If no token, return default free tier status.
	if tenant.LicenseToken == "" {
		c.JSON(http.StatusOK, gin.H{
			"tier":      "free",
			"managed":   false,
			"features":  []string{"basic_execution"},
		})
		return
	}

	// If we have a token, decode it to show status.
	if h.licenseVerifier == nil {
		c.JSON(http.StatusOK, gin.H{
			"tier":    "unknown",
			"error":   "license verifier not configured on server",
			"token_present": true,
		})
		return
	}

	claims, err := h.licenseVerifier.Verify(tenant.LicenseToken)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"tier":  "invalid",
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tier":              claims.Tier,
		"issued_at":         claims.IssuedAt,
		"expires_at":        claims.ExpiresAt,
		"features":          claims.Features,
		"managed":           true,
	})
}

// UpdateBranding handles the configuration of tenant name and logo.
func (h *Handlers) UpdateBranding(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant context missing"})
		return
	}

	if err := h.RequireFeature(c.Request.Context(), "branding"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		Name    string `json:"name" binding:"required"`
		LogoURL string `json:"logo_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.UpdateTenantBranding(c.Request.Context(), tenantID, req.Name, req.LogoURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = enterprise.RecordAction(c.Request.Context(), h.store, enterprise.ActionBrandingUpdated, map[string]interface{}{
		"name": req.Name,
	})

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "branding updated"})
}

// GetBranding returns the tenant's custom name and logo.
func (h *Handlers) GetBranding(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant context missing"})
		return
	}

	if err := h.RequireFeature(c.Request.Context(), "branding"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	tenant, err := h.store.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":     tenant.Name,
		"logo_url": tenant.LogoURL,
	})
}

// UpdateSecrets updates tenant-scoped environment variables/secrets.
func (h *Handlers) UpdateSecrets(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant context missing"})
		return
	}

	if err := h.RequireFeature(c.Request.Context(), "secrets"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	var secrets map[string]string
	if err := c.ShouldBindJSON(&secrets); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Delete all existing secrets and set the new ones
	// Or maybe just upsert all keys passed. 
	// The frontend usually sends the entire set of secrets for UpdateSecrets.
	existingKeys, err := h.secretMgr.ListSecrets(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list existing secrets"})
		return
	}
	
	// Delete any keys that are no longer in the provided map
	for _, key := range existingKeys {
		if _, ok := secrets[key]; !ok {
			_ = h.secretMgr.DeleteSecret(c.Request.Context(), tenantID, key)
		}
	}
	
	// Set/Update provided keys
	for k, v := range secrets {
		if err := h.secretMgr.SetSecret(c.Request.Context(), tenantID, k, v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update secrets"})
			return
		}
	}

	_ = enterprise.RecordAction(c.Request.Context(), h.store, enterprise.ActionSecretsUpdated, map[string]interface{}{
		"keys":   len(secrets),
	})

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "secrets updated"})
}

// GetSecrets returns the keys of the tenant's secrets (values are masked).
func (h *Handlers) GetSecrets(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant context missing"})
		return
	}

	if err := h.RequireFeature(c.Request.Context(), "secrets"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	keys, err := h.secretMgr.ListSecrets(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load secrets"})
		return
	}

	masked := make(map[string]string)
	for _, k := range keys {
		masked[k] = "********"
	}

	c.JSON(http.StatusOK, masked)
}

// ListAuditLogs returns the audit trail for the tenant.
func (h *Handlers) ListAuditLogs(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant context missing"})
		return
	}

	if err := h.RequireFeature(c.Request.Context(), "audit_logs"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	logs, err := h.store.ListAuditLogs(c.Request.Context(), tenantID, 50, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetAnalytics returns performance metrics for the tenant.
func (h *Handlers) GetAnalytics(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant context missing"})
		return
	}

	stats, err := h.store.GetRunStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = enterprise.RecordAction(c.Request.Context(), h.store, enterprise.ActionAnalyticsViewed, nil)

	c.JSON(http.StatusOK, stats)
}

// UpdateMarketSource updates the URL or file path for the MCP Marketplace.
// PUT /api/v1/enterprise/mcp-market-source
func (h *Handlers) UpdateMarketSource(c *gin.Context) {
	if err := h.RequireFeature(c.Request.Context(), "audit_logs"); err != nil { // Reuse audit_logs as a proxy for admin-level config
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		Source string `json:"source" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.SetSystemSetting(c.Request.Context(), "mcp_market_source", req.Source); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = enterprise.RecordAction(c.Request.Context(), h.store, enterprise.ActionSettingsUpdated, map[string]interface{}{
		"setting": "mcp_market_source",
		"value":   req.Source,
	})

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Marketplace source updated"})
}

// GetMarketSource returns the currently configured MCP Marketplace source.
// GET /api/v1/enterprise/mcp-market-source
func (h *Handlers) GetMarketSource(c *gin.Context) {
	ctx := c.Request.Context()
	source := h.cfg.MCPMarketSource

	if dbSource, err := h.store.GetSystemSetting(ctx, "mcp_market_source"); err == nil && dbSource != "" {
		source = dbSource
	}

	c.JSON(http.StatusOK, gin.H{"source": source})
}

// SetIDPConfig handles the configuration of external OIDC providers.
func (h *Handlers) SetIDPConfig(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant context missing"})
		return
	}

	if err := h.RequireFeature(c.Request.Context(), "sso"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	var req store.IDPConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TenantID = tenantID

	if err := h.store.UpsertTenantIDPConfig(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "IDP configuration updated"})
}

// GenerateLicenseToken generates a signed license token for a specific tenant.
// POST /api/v1/tenants/:id/license/generate
func (h *Handlers) GenerateLicenseToken(c *gin.Context) {
	if h.licenseSigner == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "license signing not configured on server (set LICENSE_PRIVATE_KEY)"})
		return
	}

	targetTenantID := c.Param("id")
	var req struct {
		Tier         string   `json:"tier" binding:"required"`
		ExpiresInDays int      `json:"expires_in_days" binding:"required"`
		Features     []string `json:"features"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims := &enterprise.LicenseClaims{
		Tier:         enterprise.Tier(req.Tier),
		TenantID:     targetTenantID,
		IssuedAt:     time.Now().Unix(),
		ExpiresAt:    time.Now().AddDate(0, 0, req.ExpiresInDays).Unix(),
		Features:     req.Features,
	}

	token, err := h.licenseSigner.Sign(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to sign token: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"claims": claims,
	})
}
