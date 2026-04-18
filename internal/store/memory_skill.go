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
	cp.DependentsCount = s.countDependentsLocked(sk.ID)
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
			cp.DependentsCount = s.countDependentsLocked(sk.ID)
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
	sk, ok := s.skills[key]
	if !ok {
		return fmt.Errorf("skill %q not found", name)
	}
	delete(s.skillDeps, sk.ID)
	delete(s.skills, key)
	return nil
}

func (s *MemoryStore) UpsertSkillDependencies(ctx context.Context, skillID string, dependsOnIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(dependsOnIDs) == 0 {
		delete(s.skillDeps, skillID)
		return nil
	}
	set := make(map[string]bool, len(dependsOnIDs))
	for _, id := range dependsOnIDs {
		set[id] = true
	}
	s.skillDeps[skillID] = set
	return nil
}

func (s *MemoryStore) ListSkillDependents(ctx context.Context, name string) ([]*Skill, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()

	target, ok := s.skills[tenantID+":"+name]
	if !ok {
		return nil, fmt.Errorf("skill %q not found", name)
	}

	var out []*Skill
	for callerID, deps := range s.skillDeps {
		if !deps[target.ID] {
			continue
		}
		// Find the skill with this ID in the tenant.
		for _, sk := range s.skills {
			if sk.ID == callerID && (tenantID == "" || sk.TenantID == tenantID) {
				cp := *sk
				cp.DependentsCount = s.countDependentsLocked(sk.ID)
				out = append(out, &cp)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// countDependentsLocked counts how many skills depend on the given skill ID.
// Must be called with s.mu held (read or write).
func (s *MemoryStore) countDependentsLocked(skillID string) int {
	count := 0
	for _, deps := range s.skillDeps {
		if deps[skillID] {
			count++
		}
	}
	return count
}
