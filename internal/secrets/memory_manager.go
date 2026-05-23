package secrets

import (
	"context"
	"sync"
)

// MemorySecretManager provides an in-memory implementation for tests and local dev.
type MemorySecretManager struct {
	mu      sync.RWMutex
	secrets map[string]map[string]string // tenantID -> key -> value
}

func NewMemorySecretManager() *MemorySecretManager {
	return &MemorySecretManager{
		secrets: make(map[string]map[string]string),
	}
}

func (m *MemorySecretManager) GetSecret(_ context.Context, tenantID, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if t, ok := m.secrets[tenantID]; ok {
		if val, ok := t[key]; ok {
			return val, nil
		}
	}
	return "", ErrSecretNotFound
}

func (m *MemorySecretManager) SetSecret(_ context.Context, tenantID, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.secrets[tenantID]; !ok {
		m.secrets[tenantID] = make(map[string]string)
	}
	m.secrets[tenantID][key] = value
	return nil
}

func (m *MemorySecretManager) DeleteSecret(_ context.Context, tenantID, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.secrets[tenantID]; ok {
		delete(t, key)
	}
	return nil
}

func (m *MemorySecretManager) ListSecrets(_ context.Context, tenantID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var keys []string
	if t, ok := m.secrets[tenantID]; ok {
		for k := range t {
			keys = append(keys, k)
		}
	}
	return keys, nil
}
