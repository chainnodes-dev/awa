package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// MemoryStore is an in-memory implementation for development and testing.
type MemoryStore struct {
	mu            sync.RWMutex
	definitions   map[string]*defEntry       // key: name@version
	runs          map[string]*asmtypes.WorkflowRun
	transitions   map[string][]*asmtypes.TransitionRecord // key: runID
	hitl          map[string]*asmtypes.HITLRequest        // key: runID
	users         map[string]*User         // key: id
	usersByName   map[string]string        // username → id
	refreshTokens map[string]*RefreshToken // key: TokenHash
}

type defEntry struct {
	def  *asmtypes.WorkflowDef
	yaml string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		definitions:   make(map[string]*defEntry),
		runs:          make(map[string]*asmtypes.WorkflowRun),
		transitions:   make(map[string][]*asmtypes.TransitionRecord),
		hitl:          make(map[string]*asmtypes.HITLRequest),
		users:         make(map[string]*User),
		usersByName:   make(map[string]string),
		refreshTokens: make(map[string]*RefreshToken),
	}
}

// -- WorkflowStore --

func (s *MemoryStore) SaveDefinition(_ context.Context, def *asmtypes.WorkflowDef, yamlSource string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.definitions[defKey(def.Metadata.Name, def.Metadata.Version)] = &defEntry{def: def, yaml: yamlSource}
	return nil
}

func (s *MemoryStore) GetDefinition(_ context.Context, name, version string) (*asmtypes.WorkflowDef, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.definitions[defKey(name, version)]
	if !ok {
		return nil, "", fmt.Errorf("workflow definition '%s@%s' not found", name, version)
	}
	return e.def, e.yaml, nil
}

func (s *MemoryStore) ListDefinitions(_ context.Context) ([]*asmtypes.WorkflowDef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*asmtypes.WorkflowDef, 0, len(s.definitions))
	for _, e := range s.definitions {
		out = append(out, e.def)
	}
	return out, nil
}

func (s *MemoryStore) DeleteDefinition(_ context.Context, name, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.definitions, defKey(name, version))
	return nil
}

// -- RunStore --

func (s *MemoryStore) CreateRun(_ context.Context, run *asmtypes.WorkflowRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	// deep copy via JSON to avoid sharing maps
	cp := deepCopyRun(run)
	s.runs[run.ID] = cp
	return nil
}

func (s *MemoryStore) GetRun(_ context.Context, id string) (*asmtypes.WorkflowRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[id]
	if !ok {
		return nil, fmt.Errorf("run '%s' not found", id)
	}
	return deepCopyRun(r), nil
}

func (s *MemoryStore) UpdateRun(_ context.Context, run *asmtypes.WorkflowRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[run.ID]; !ok {
		return fmt.Errorf("run '%s' not found", run.ID)
	}
	s.runs[run.ID] = deepCopyRun(run)
	return nil
}

func (s *MemoryStore) ListRuns(_ context.Context, filter RunFilter) ([]*asmtypes.WorkflowRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*asmtypes.WorkflowRun
	for _, r := range s.runs {
		if filter.WorkflowName != "" && r.WorkflowName != filter.WorkflowName {
			continue
		}
		if filter.Status != "" && r.Status != filter.Status {
			continue
		}
		out = append(out, deepCopyRun(r))
	}
	return out, nil
}

func (s *MemoryStore) RecordTransition(_ context.Context, rec *asmtypes.TransitionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec.ID == "" {
		rec.ID = uuid.NewString()
	}
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now()
	}
	s.transitions[rec.RunID] = append(s.transitions[rec.RunID], rec)
	return nil
}

func (s *MemoryStore) ListTransitions(_ context.Context, runID string) ([]*asmtypes.TransitionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.transitions[runID], nil
}

// -- HITLStore --

func (s *MemoryStore) CreateHITL(_ context.Context, req *asmtypes.HITLRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	s.hitl[req.RunID] = req
	return nil
}

func (s *MemoryStore) GetHITL(_ context.Context, runID string) (*asmtypes.HITLRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.hitl[runID]
	if !ok {
		return nil, fmt.Errorf("no HITL request for run '%s'", runID)
	}
	return req, nil
}

func (s *MemoryStore) ResolveHITL(_ context.Context, runID, resolution, resolver string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.hitl[runID]
	if !ok {
		return fmt.Errorf("no HITL request for run '%s'", runID)
	}
	now := time.Now()
	req.Resolved = true
	req.ResolvedAt = &now
	req.Resolution = resolution
	req.Resolver = resolver
	return nil
}

func (s *MemoryStore) ListPendingHITL(_ context.Context) ([]*asmtypes.HITLRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*asmtypes.HITLRequest
	for _, r := range s.hitl {
		if !r.Resolved {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *MemoryStore) Close() error { return nil }

// -- UserStore --

func (s *MemoryStore) CreateUser(_ context.Context, u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	if _, exists := s.usersByName[u.Username]; exists {
		return fmt.Errorf("user '%s' already exists", u.Username)
	}
	cp := *u
	s.users[u.ID] = &cp
	s.usersByName[u.Username] = u.ID
	return nil
}

func (s *MemoryStore) GetUserByID(_ context.Context, id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("user '%s' not found", id)
	}
	cp := *u
	return &cp, nil
}

func (s *MemoryStore) GetUserByUsername(_ context.Context, username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.usersByName[username]
	if !ok {
		return nil, fmt.Errorf("user '%s' not found", username)
	}
	cp := *s.users[id]
	return &cp, nil
}

func (s *MemoryStore) UpdateUserRole(_ context.Context, id, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return fmt.Errorf("user '%s' not found", id)
	}
	u.Role = role
	return nil
}

func (s *MemoryStore) DeleteUser(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return fmt.Errorf("user '%s' not found", id)
	}
	delete(s.usersByName, u.Username)
	delete(s.users, id)
	return nil
}

func (s *MemoryStore) ListUsers(_ context.Context) ([]*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		cp := *u
		out = append(out, &cp)
	}
	return out, nil
}

func (s *MemoryStore) HasAnyUser(_ context.Context) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users) > 0, nil
}

// -- RefreshTokenStore --

func (s *MemoryStore) CreateRefreshToken(_ context.Context, t *RefreshToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	cp := *t
	s.refreshTokens[t.TokenHash] = &cp
	return nil
}

func (s *MemoryStore) GetRefreshTokenByHash(_ context.Context, hash string) (*RefreshToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.refreshTokens[hash]
	if !ok {
		return nil, fmt.Errorf("refresh token not found")
	}
	cp := *t
	return &cp, nil
}

func (s *MemoryStore) RevokeRefreshToken(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.refreshTokens {
		if t.ID == id {
			t.Revoked = true
			return nil
		}
	}
	return fmt.Errorf("refresh token '%s' not found", id)
}

func (s *MemoryStore) RevokeAllUserTokens(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.refreshTokens {
		if t.UserID == userID {
			t.Revoked = true
		}
	}
	return nil
}

// helpers

func defKey(name, version string) string { return name + "@" + version }

func deepCopyRun(r *asmtypes.WorkflowRun) *asmtypes.WorkflowRun {
	b, _ := json.Marshal(r)
	var cp asmtypes.WorkflowRun
	_ = json.Unmarshal(b, &cp)
	return &cp
}
