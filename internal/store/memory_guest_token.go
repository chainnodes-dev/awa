package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CreateGuestToken stores a new guest token in memory.
func (s *MemoryStore) CreateGuestToken(_ context.Context, token *GuestToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if token.ID == "" {
		token.ID = uuid.NewString()
	}
	if token.Status == "" {
		token.Status = "active"
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}
	if token.TokenHash == "" {
		return fmt.Errorf("token_hash is required")
	}

	// Make a shallow copy to prevent external mutation
	copied := *token
	s.guestTokens[token.ID] = &copied
	s.guestTokensByHash[token.TokenHash] = token.ID

	return nil
}

// GetGuestTokenByHash retrieves a guest token by its SHA-256 hash.
func (s *MemoryStore) GetGuestTokenByHash(_ context.Context, tokenHash string) (*GuestToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.guestTokensByHash[tokenHash]
	if !ok {
		return nil, fmt.Errorf("guest token not found")
	}

	tok, ok := s.guestTokens[id]
	if !ok {
		return nil, fmt.Errorf("guest token not found")
	}

	copied := *tok
	return &copied, nil
}

// GetActiveGuestToken finds an active, non-expired guest token for a run and state.
func (s *MemoryStore) GetActiveGuestToken(_ context.Context, runID, stateName string) (*GuestToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	for _, tok := range s.guestTokens {
		if tok.RunID == runID && tok.StateName == stateName && tok.Status == "active" && now.Before(tok.ExpiresAt) {
			copied := *tok
			return &copied, nil
		}
	}

	return nil, fmt.Errorf("active guest token not found for run %s and state %s", runID, stateName)
}

// MarkGuestTokenCompleted marks the guest token as completed with the submitter.
func (s *MemoryStore) MarkGuestTokenCompleted(_ context.Context, id, completedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tok, ok := s.guestTokens[id]
	if !ok {
		return fmt.Errorf("guest token %s not found", id)
	}

	now := time.Now()
	tok.Status = "completed"
	tok.CompletedAt = &now
	tok.CompletedBy = completedBy

	return nil
}

// RevokeGuestToken revokes an active guest token.
func (s *MemoryStore) RevokeGuestToken(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tok, ok := s.guestTokens[id]
	if !ok {
		return fmt.Errorf("guest token %s not found", id)
	}

	tok.Status = "revoked"
	return nil
}
