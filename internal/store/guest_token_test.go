package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGuestToken_MemoryStore(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	rawToken, tokenHash, err := GenerateGuestToken()
	if err != nil {
		t.Fatalf("GenerateGuestToken failed: %v", err)
	}
	if len(rawToken) != 64 {
		t.Errorf("expected 64 char hex token, got %d", len(rawToken))
	}
	if tokenHash != HashGuestToken(rawToken) {
		t.Errorf("token hash mismatch")
	}

	runID := uuid.NewString()
	token := &GuestToken{
		ID:          uuid.NewString(),
		TenantID:    "t-default",
		RunID:       runID,
		StateName:   "candidate_upload",
		TokenHash:   tokenHash,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		Status:      "active",
		Title:       "Candidate Onboarding",
		Description: "Please upload your resume and fill in your details.",
		FormSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"fullName": map[string]interface{}{"type": "string"},
			},
		},
	}

	if err := s.CreateGuestToken(ctx, token); err != nil {
		t.Fatalf("CreateGuestToken failed: %v", err)
	}

	// Retrieve by hash
	retrieved, err := s.GetGuestTokenByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetGuestTokenByHash failed: %v", err)
	}
	if retrieved.ID != token.ID || retrieved.Title != "Candidate Onboarding" {
		t.Errorf("retrieved token mismatch: %+v", retrieved)
	}
	if !retrieved.IsValid() {
		t.Errorf("expected token to be valid")
	}

	// Retrieve active
	active, err := s.GetActiveGuestToken(ctx, runID, "candidate_upload")
	if err != nil {
		t.Fatalf("GetActiveGuestToken failed: %v", err)
	}
	if active.ID != token.ID {
		t.Errorf("active token ID mismatch")
	}

	// Mark completed
	if err := s.MarkGuestTokenCompleted(ctx, token.ID, "john.doe@example.com"); err != nil {
		t.Fatalf("MarkGuestTokenCompleted failed: %v", err)
	}

	updated, err := s.GetGuestTokenByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetGuestTokenByHash failed: %v", err)
	}
	if updated.Status != "completed" || updated.CompletedBy != "john.doe@example.com" || updated.CompletedAt == nil {
		t.Errorf("expected completed token, got: %+v", updated)
	}
	if updated.IsValid() {
		t.Errorf("expected completed token to not be valid")
	}

	// GetActive should now fail
	if _, err := s.GetActiveGuestToken(ctx, runID, "candidate_upload"); err == nil {
		t.Errorf("expected error getting active token after completion")
	}
}

func TestGuestToken_Expiry(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_, tokenHash, _ := GenerateGuestToken()
	runID := uuid.NewString()

	expiredToken := &GuestToken{
		ID:        uuid.NewString(),
		TenantID:  "t-default",
		RunID:     runID,
		StateName: "review",
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // expired 1h ago
		Status:    "active",
	}

	_ = s.CreateGuestToken(ctx, expiredToken)

	retrieved, err := s.GetGuestTokenByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetGuestTokenByHash failed: %v", err)
	}
	if retrieved.IsValid() {
		t.Errorf("expected expired token to be invalid")
	}

	// GetActive should not return expired tokens
	if _, err := s.GetActiveGuestToken(ctx, runID, "review"); err == nil {
		t.Errorf("expected error getting active token when expired")
	}
}

