package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *PostgresStore) CreateGuestToken(ctx context.Context, token *GuestToken) error {
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

	fs, err := json.Marshal(token.FormSchema)
	if err != nil {
		fs = []byte("{}")
	}
	meta, err := json.Marshal(token.Metadata)
	if err != nil {
		meta = []byte("{}")
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO guest_tokens (id, tenant_id, run_id, state_name, token_hash, expires_at, status, title, description, form_schema, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		token.ID, token.TenantID, token.RunID, token.StateName, token.TokenHash, token.ExpiresAt, token.Status,
		token.Title, token.Description, fs, meta, token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert guest token: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetGuestTokenByHash(ctx context.Context, tokenHash string) (*GuestToken, error) {
	var token GuestToken
	var fs, meta []byte
	var title, desc, compBy *string

	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, run_id, state_name, token_hash, expires_at, status,
		       title, description, form_schema, metadata, created_at, completed_at, completed_by
		FROM guest_tokens
		WHERE token_hash = $1`,
		tokenHash,
	).Scan(
		&token.ID, &token.TenantID, &token.RunID, &token.StateName, &token.TokenHash,
		&token.ExpiresAt, &token.Status, &title, &desc, &fs, &meta,
		&token.CreatedAt, &token.CompletedAt, &compBy,
	)
	if err != nil {
		return nil, fmt.Errorf("get guest token by hash: %w", err)
	}

	if title != nil {
		token.Title = *title
	}
	if desc != nil {
		token.Description = *desc
	}
	if compBy != nil {
		token.CompletedBy = *compBy
	}
	if len(fs) > 0 {
		_ = json.Unmarshal(fs, &token.FormSchema)
	}
	if len(meta) > 0 {
		_ = json.Unmarshal(meta, &token.Metadata)
	}
	return &token, nil
}

func (s *PostgresStore) GetActiveGuestToken(ctx context.Context, runID, stateName string) (*GuestToken, error) {
	var token GuestToken
	var fs, meta []byte
	var title, desc, compBy *string

	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, run_id, state_name, token_hash, expires_at, status,
		       title, description, form_schema, metadata, created_at, completed_at, completed_by
		FROM guest_tokens
		WHERE run_id = $1 AND state_name = $2 AND status = 'active' AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1`,
		runID, stateName,
	).Scan(
		&token.ID, &token.TenantID, &token.RunID, &token.StateName, &token.TokenHash,
		&token.ExpiresAt, &token.Status, &title, &desc, &fs, &meta,
		&token.CreatedAt, &token.CompletedAt, &compBy,
	)
	if err != nil {
		return nil, fmt.Errorf("active guest token not found: %w", err)
	}

	if title != nil {
		token.Title = *title
	}
	if desc != nil {
		token.Description = *desc
	}
	if compBy != nil {
		token.CompletedBy = *compBy
	}
	if len(fs) > 0 {
		_ = json.Unmarshal(fs, &token.FormSchema)
	}
	if len(meta) > 0 {
		_ = json.Unmarshal(meta, &token.Metadata)
	}
	return &token, nil
}

func (s *PostgresStore) MarkGuestTokenCompleted(ctx context.Context, id, completedBy string) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx, `
		UPDATE guest_tokens
		SET status = 'completed', completed_at = $2, completed_by = $3
		WHERE id = $1`,
		id, now, completedBy,
	)
	if err != nil {
		return fmt.Errorf("mark guest token completed: %w", err)
	}
	return nil
}

func (s *PostgresStore) RevokeGuestToken(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE guest_tokens
		SET status = 'revoked'
		WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("revoke guest token: %w", err)
	}
	return nil
}
