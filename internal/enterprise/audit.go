package enterprise

import (
	"context"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/store"
)

// Actions defines common loggable actions.
const (
	ActionWorkflowCreated = "workflow.created"
	ActionWorkflowDeleted = "workflow.deleted"
	ActionLicenseUpdated  = "license.updated"
	ActionAPIKeyCreated   = "apikey.created"
	ActionAPIKeyRevoked   = "apikey.revoked"
	ActionLoginOIDC       = "auth.oidc_login"
	ActionBrandingUpdated = "branding.updated"
	ActionSecretsUpdated  = "secrets.updated"
	ActionAnalyticsViewed = "analytics.viewed"
	ActionSettingsUpdated  = "settings.updated"
)

// RecordAction is a helper to record a tenant-scoped audit log.
func RecordAction(ctx context.Context, s store.Store, action string, details map[string]interface{}) error {
	tenantID := store.TenantIDFromContext(ctx)
	if tenantID == "" {
		return nil // skip if no tenant context (e.g. system jobs)
	}

	claims := auth.ClaimsFromContext(ctx)
	userID := ""
	if claims != nil {
		userID = claims.Subject
	}

	entry := &store.AuditLog{
		TenantID: tenantID,
		UserID:   userID,
		Action:   action,
		Details:  details,
	}

	return s.RecordAuditLog(ctx, entry)
}
