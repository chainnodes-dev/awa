package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ── EventSubscriptionStore ────────────────────────────────────────────────────

func (s *MemoryStore) RegisterEventSubscription(_ context.Context, sub *EventSubscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = time.Now()
	}
	cp := *sub
	s.eventSubs[sub.ID] = &cp
	return nil
}

func (s *MemoryStore) UnregisterEventSubscription(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.eventSubs, id)
	return nil
}

func (s *MemoryStore) ListEventSubscriptions(_ context.Context, tenantID, eventName string) ([]*EventSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*EventSubscription
	for _, sub := range s.eventSubs {
		if sub.TenantID == tenantID && sub.EventName == eventName {
			cp := *sub
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

// ── APIKeyStore ───────────────────────────────────────────────────────────────

func (s *MemoryStore) CreateAPIKey(_ context.Context, key *APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key.ID == "" {
		key.ID = uuid.NewString()
	}
	if key.CreatedAt.IsZero() {
		key.CreatedAt = time.Now()
	}
	if _, exists := s.apiKeysByHash[key.KeyHash]; exists {
		return fmt.Errorf("api key with this hash already exists")
	}
	cp := *key
	s.apiKeys[key.ID] = &cp
	s.apiKeysByHash[key.KeyHash] = key.ID
	return nil
}

func (s *MemoryStore) GetAPIKeyByHash(_ context.Context, hash string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.apiKeysByHash[hash]
	if !ok {
		return nil, fmt.Errorf("api key not found")
	}
	key := s.apiKeys[id]
	if key.IsRevoked() {
		return nil, fmt.Errorf("api key is revoked")
	}
	cp := *key
	return &cp, nil
}

func (s *MemoryStore) ListAPIKeys(ctx context.Context) ([]*APIKey, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*APIKey
	for _, key := range s.apiKeys {
		if tenantID == "" || key.TenantID == tenantID {
			cp := *key
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) RevokeAPIKey(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.apiKeys[id]
	if !ok {
		return fmt.Errorf("api key %q not found", id)
	}
	now := time.Now()
	key.RevokedAt = &now
	return nil
}

func (s *MemoryStore) TouchAPIKey(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.apiKeys[id]
	if !ok {
		return nil // non-fatal
	}
	now := time.Now()
	key.LastUsedAt = &now
	return nil
}
