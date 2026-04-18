package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ── EventSubscriptionStore ────────────────────────────────────────────────────

func (s *PostgresStore) RegisterEventSubscription(ctx context.Context, sub *EventSubscription) error {
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = time.Now()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO event_subscriptions
		     (id, tenant_id, run_id, temporal_id, event_name, on_match_trigger, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (id) DO UPDATE SET
		     on_match_trigger = EXCLUDED.on_match_trigger`,
		sub.ID, sub.TenantID, sub.RunID, sub.TemporalID,
		sub.EventName, sub.OnMatchTrigger, sub.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("register event subscription: %w", err)
	}
	return nil
}

func (s *PostgresStore) UnregisterEventSubscription(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM event_subscriptions WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("unregister event subscription: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListEventSubscriptions(ctx context.Context, tenantID, eventName string) ([]*EventSubscription, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, run_id, temporal_id, event_name, on_match_trigger, created_at
		 FROM event_subscriptions
		 WHERE tenant_id=$1 AND event_name=$2
		 ORDER BY created_at`,
		tenantID, eventName,
	)
	if err != nil {
		return nil, fmt.Errorf("list event subscriptions: %w", err)
	}
	defer rows.Close()
	var out []*EventSubscription
	for rows.Next() {
		var sub EventSubscription
		if err := rows.Scan(
			&sub.ID, &sub.TenantID, &sub.RunID, &sub.TemporalID,
			&sub.EventName, &sub.OnMatchTrigger, &sub.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event subscription: %w", err)
		}
		out = append(out, &sub)
	}
	return out, rows.Err()
}

// ── APIKeyStore ───────────────────────────────────────────────────────────────

func (s *PostgresStore) CreateAPIKey(ctx context.Context, key *APIKey) error {
	if key.ID == "" {
		key.ID = uuid.NewString()
	}
	if key.CreatedAt.IsZero() {
		key.CreatedAt = time.Now()
	}
	var createdBy *string
	if key.CreatedBy != "" {
		createdBy = &key.CreatedBy
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO api_keys (id, tenant_id, name, key_hash, created_by, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		key.ID, key.TenantID, key.Name, key.KeyHash, createdBy, key.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create api key: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetAPIKeyByHash(ctx context.Context, hash string) (*APIKey, error) {
	var key APIKey
	var createdBy *string
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, key_hash, created_by, created_at, last_used_at, revoked_at
		 FROM api_keys WHERE key_hash=$1 AND revoked_at IS NULL`,
		hash,
	).Scan(
		&key.ID, &key.TenantID, &key.Name, &key.KeyHash,
		&createdBy, &key.CreatedAt, &key.LastUsedAt, &key.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("api key not found")
	}
	if createdBy != nil {
		key.CreatedBy = *createdBy
	}
	return &key, nil
}

func (s *PostgresStore) ListAPIKeys(ctx context.Context) ([]*APIKey, error) {
	tenantID := TenantIDFromContext(ctx)
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, name, key_hash, created_by, created_at, last_used_at, revoked_at
		 FROM api_keys WHERE tenant_id=$1 ORDER BY created_at`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()
	var out []*APIKey
	for rows.Next() {
		var key APIKey
		var createdBy *string
		if err := rows.Scan(
			&key.ID, &key.TenantID, &key.Name, &key.KeyHash,
			&createdBy, &key.CreatedAt, &key.LastUsedAt, &key.RevokedAt,
		); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		if createdBy != nil {
			key.CreatedBy = *createdBy
		}
		out = append(out, &key)
	}
	return out, rows.Err()
}

func (s *PostgresStore) RevokeAPIKey(ctx context.Context, id string) error {
	tenantID := TenantIDFromContext(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE api_keys SET revoked_at=NOW() WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
	)
	return err
}

func (s *PostgresStore) TouchAPIKey(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE api_keys SET last_used_at=NOW() WHERE id=$1`, id,
	)
	return err
}
