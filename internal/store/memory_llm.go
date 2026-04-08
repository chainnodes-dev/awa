package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (m *MemoryStore) UpsertLLMConfig(_ context.Context, cfg *LLMConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cfg.ID == "" {
		cfg.ID = uuid.NewString()
	}
	cfg.UpdatedAt = time.Now()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = cfg.UpdatedAt
	}
	key := cfg.TenantID + ":" + cfg.Provider
	cp := *cfg
	m.llmConfigs[key] = &cp
	return nil
}

func (m *MemoryStore) GetLLMConfig(ctx context.Context, provider string) (*LLMConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantID := TenantIDFromContext(ctx)
	key := tenantID + ":" + provider
	cfg, ok := m.llmConfigs[key]
	if !ok {
		return nil, fmt.Errorf("llm config not found: %s", provider)
	}
	cp := *cfg
	return &cp, nil
}

func (m *MemoryStore) ListLLMConfigs(ctx context.Context) ([]*LLMConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantID := TenantIDFromContext(ctx)
	prefix := tenantID + ":"
	var out []*LLMConfig
	for key, cfg := range m.llmConfigs {
		if strings.HasPrefix(key, prefix) {
			cp := *cfg
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *MemoryStore) DeleteLLMConfig(ctx context.Context, provider string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantID := TenantIDFromContext(ctx)
	delete(m.llmConfigs, tenantID+":"+provider)
	return nil
}

func (m *MemoryStore) SetDefaultProvider(ctx context.Context, provider string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantID := TenantIDFromContext(ctx)
	prefix := tenantID + ":"
	for key, cfg := range m.llmConfigs {
		if strings.HasPrefix(key, prefix) {
			cfg.IsDefault = (cfg.Provider == provider)
		}
	}
	return nil
}

func (m *MemoryStore) HasAnyLLMConfig(ctx context.Context) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantID := TenantIDFromContext(ctx)
	prefix := tenantID + ":"
	for key := range m.llmConfigs {
		if strings.HasPrefix(key, prefix) {
			return true, nil
		}
	}
	return false, nil
}
