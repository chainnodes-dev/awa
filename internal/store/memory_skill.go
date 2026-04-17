package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// SkillStore methods on MemoryStore.

func (s *MemoryStore) CreateSkill(ctx context.Context, sk *Skill) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sk.ID == "" {
		sk.ID = uuid.NewString()
	}
	now := time.Now()
	if sk.CreatedAt.IsZero() {
		sk.CreatedAt = now
	}
	sk.UpdatedAt = now
	key := sk.TenantID + ":" + sk.Name
	if _, exists := s.skills[key]; exists {
		return fmt.Errorf("skill %q already exists", sk.Name)
	}
	cp := *sk
	s.skills[key] = &cp
	return nil
}

func (s *MemoryStore) GetSkill(ctx context.Context, name string) (*Skill, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := tenantID + ":" + name
	sk, ok := s.skills[key]
	if !ok {
		return nil, fmt.Errorf("skill %q not found", name)
	}
	cp := *sk
	return &cp, nil
}

func (s *MemoryStore) ListSkills(ctx context.Context) ([]*Skill, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Skill
	for _, sk := range s.skills {
		if tenantID == "" || sk.TenantID == tenantID {
			cp := *sk
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *MemoryStore) UpdateSkill(ctx context.Context, sk *Skill) error {
	tenantID := TenantIDFromContext(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tenantID + ":" + sk.Name
	existing, ok := s.skills[key]
	if !ok {
		return fmt.Errorf("skill %q not found", sk.Name)
	}
	existing.Description = sk.Description
	existing.WorkflowRef = sk.WorkflowRef
	existing.Inputs = sk.Inputs
	existing.Outputs = sk.Outputs
	existing.Capabilities = sk.Capabilities
	existing.UpdatedAt = time.Now()
	return nil
}

func (s *MemoryStore) DeleteSkill(ctx context.Context, name string) error {
	tenantID := TenantIDFromContext(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tenantID + ":" + name
	if _, ok := s.skills[key]; !ok {
		return fmt.Errorf("skill %q not found", name)
	}
	delete(s.skills, key)
	return nil
}
