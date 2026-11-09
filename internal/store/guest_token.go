package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// GuestToken tracks public, token-secured external web forms allowing external users
// (e.g. candidates, patients, vendors) to view forms, upload documents, and resolve
// pending HITL pauses without an account.
type GuestToken struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	RunID       string                 `json:"run_id"`
	StateName   string                 `json:"state_name"`
	TokenHash   string                 `json:"token_hash"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Status      string                 `json:"status"` // "active", "completed", "expired", "revoked"
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	FormSchema  map[string]interface{} `json:"form_schema,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	CompletedBy string                 `json:"completed_by,omitempty"`
}

// IsValid checks if the token is active and has not expired.
func (t *GuestToken) IsValid() bool {
	if t.Status != "active" {
		return false
	}
	return time.Now().Before(t.ExpiresAt)
}

// GuestTokenStore provides persistence and lookup operations for guest tokens.
type GuestTokenStore interface {
	CreateGuestToken(ctx context.Context, token *GuestToken) error
	GetGuestTokenByHash(ctx context.Context, tokenHash string) (*GuestToken, error)
	GetActiveGuestToken(ctx context.Context, runID, stateName string) (*GuestToken, error)
	MarkGuestTokenCompleted(ctx context.Context, id, completedBy string) error
	RevokeGuestToken(ctx context.Context, id string) error
}

// HashGuestToken computes the SHA-256 hex string of a raw guest token.
func HashGuestToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// GenerateGuestToken produces a 32-byte (64 hex char) cryptographically secure random token
// and its corresponding SHA-256 hash.
func GenerateGuestToken() (rawToken string, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate random bytes: %w", err)
	}
	rawToken = hex.EncodeToString(b)
	tokenHash = HashGuestToken(rawToken)
	return rawToken, tokenHash, nil
}

