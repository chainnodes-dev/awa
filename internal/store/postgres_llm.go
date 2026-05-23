package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *PostgresStore) UpsertLLMConfig(ctx context.Context, cfg *LLMConfig) error {
	if cfg.ID == "" {
		cfg.ID = uuid.NewString()
	}
	if cfg.TenantID == "" {
		cfg.TenantID = TenantIDFromContext(ctx)
	}
	cfg.UpdatedAt = time.Now()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = cfg.UpdatedAt
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO llm_configs
		  (id, tenant_id, provider, api_key, base_url, default_model, max_output_tokens, enabled, is_default, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (tenant_id, provider) DO UPDATE SET
		  api_key=$4, base_url=$5, default_model=$6, max_output_tokens=$7, enabled=$8, updated_at=$11`,
		cfg.ID, cfg.TenantID, cfg.Provider, cfg.APIKey, cfg.BaseURL,
		cfg.DefaultModel, cfg.MaxOutputTokens, cfg.Enabled, cfg.IsDefault, cfg.CreatedAt, cfg.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) GetLLMConfig(ctx context.Context, provider string) (*LLMConfig, error) {
	tenantID := TenantIDFromContext(ctx)
	var c LLMConfig
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, provider, api_key, base_url, default_model, max_output_tokens, enabled, is_default, created_at, updated_at
		FROM llm_configs WHERE tenant_id=$1 AND provider=$2`,
		tenantID, provider,
	).Scan(&c.ID, &c.TenantID, &c.Provider, &c.APIKey, &c.BaseURL,
		&c.DefaultModel, &c.MaxOutputTokens, &c.Enabled, &c.IsDefault, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get llm config %q: %w", provider, err)
	}
	return &c, nil
}

func (s *PostgresStore) ListLLMConfigs(ctx context.Context) ([]*LLMConfig, error) {
	tenantID := TenantIDFromContext(ctx)
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, provider, api_key, base_url, default_model, max_output_tokens, enabled, is_default, created_at, updated_at
		FROM llm_configs WHERE tenant_id=$1 ORDER BY provider ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*LLMConfig
	for rows.Next() {
		var c LLMConfig
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Provider, &c.APIKey, &c.BaseURL,
			&c.DefaultModel, &c.MaxOutputTokens, &c.Enabled, &c.IsDefault, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (s *PostgresStore) DeleteLLMConfig(ctx context.Context, provider string) error {
	tenantID := TenantIDFromContext(ctx)
	_, err := s.pool.Exec(ctx,
		`DELETE FROM llm_configs WHERE tenant_id=$1 AND provider=$2`,
		tenantID, provider,
	)
	return err
}

func (s *PostgresStore) SetDefaultProvider(ctx context.Context, provider string) error {
	tenantID := TenantIDFromContext(ctx)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`UPDATE llm_configs SET is_default=FALSE WHERE tenant_id=$1`, tenantID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE llm_configs SET is_default=TRUE WHERE tenant_id=$1 AND provider=$2`,
		tenantID, provider); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) HasAnyLLMConfig(ctx context.Context) (bool, error) {
	tenantID := TenantIDFromContext(ctx)
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM llm_configs WHERE tenant_id=$1 LIMIT 1`, tenantID,
	).Scan(&count)
	return count > 0, err
}
