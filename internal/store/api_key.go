package store

import (
	"context"
	"time"
)

// APIKey is a long-lived token used by external callers to authenticate without JWT.
// The raw key is shown only once at creation; only its SHA-256 hash is persisted.
type APIKey struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"-"`               // never serialised to JSON
	CreatedBy   string     `json:"created_by"`      // user ID
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
}

// IsRevoked reports whether the key has been revoked.
func (k *APIKey) IsRevoked() bool { return k.RevokedAt != nil }

// APIKeyStore manages API keys for external callers.
type APIKeyStore interface {
	// CreateAPIKey persists a new API key. ID is auto-assigned if empty.
	CreateAPIKey(ctx context.Context, key *APIKey) error
	// GetAPIKeyByHash retrieves an active (non-revoked) key by its SHA-256 hash.
	// Returns an error if not found or revoked.
	GetAPIKeyByHash(ctx context.Context, hash string) (*APIKey, error)
	// ListAPIKeys returns all API keys for the tenant in ctx, ordered by created_at.
	ListAPIKeys(ctx context.Context) ([]*APIKey, error)
	// RevokeAPIKey sets revoked_at to now for the given key ID.
	RevokeAPIKey(ctx context.Context, id string) error
	// TouchAPIKey updates last_used_at to now for the given key ID.
	TouchAPIKey(ctx context.Context, id string) error
}
