package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// MemoryStore is an in-memory implementation for development and testing.
type MemoryStore struct {
	mu            sync.RWMutex
	tenants       map[string]*Tenant  // key: id
	tenantsBySlug map[string]string   // slug → id
	idpConfigs    map[string]*IDPConfig
	definitions   map[string]*defEntry       // key: tenantID:name@version
	runs          map[string]*asmtypes.WorkflowRun
	transitions   map[string][]*asmtypes.TransitionRecord // key: runID
	hitl          map[string]*asmtypes.HITLRequest        // key: runID
	users         map[string]*User         // key: id
	usersByName   map[string]string        // tenantID:username → id
	refreshTokens map[string]*RefreshToken // key: TokenHash
	mcpServers    map[string]*MCPServer    // key: id
	llmConfigs    map[string]*LLMConfig    // key: tenantID:provider
	// Phase-4 stores.
	eventSubs    map[string]*EventSubscription // key: id
	apiKeys      map[string]*APIKey            // key: id
	apiKeysByHash map[string]string            // hash → id

	mcpLogs       map[string][]*asmtypes.MCPAuditLog // key: runID
	auditLogs     []*AuditLog

	systemSettings map[string]string
}

type defEntry struct {
	def  *asmtypes.WorkflowDef
	yaml string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tenants:       make(map[string]*Tenant),
		tenantsBySlug: make(map[string]string),
		idpConfigs:    make(map[string]*IDPConfig),
		definitions:   make(map[string]*defEntry),
		runs:          make(map[string]*asmtypes.WorkflowRun),
		transitions:   make(map[string][]*asmtypes.TransitionRecord),
		hitl:          make(map[string]*asmtypes.HITLRequest),
		users:         make(map[string]*User),
		usersByName:   make(map[string]string),
		refreshTokens: make(map[string]*RefreshToken),
		mcpServers:    make(map[string]*MCPServer),
		llmConfigs:    make(map[string]*LLMConfig),
		eventSubs:     make(map[string]*EventSubscription),
		apiKeys:       make(map[string]*APIKey),
		apiKeysByHash: make(map[string]string),
		mcpLogs:       make(map[string][]*asmtypes.MCPAuditLog),
		systemSettings: make(map[string]string),
	}
}

// -- TenantStore --

func (s *MemoryStore) CreateTenant(_ context.Context, t *Tenant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	if _, exists := s.tenantsBySlug[t.Slug]; exists {
		return fmt.Errorf("tenant with slug '%s' already exists", t.Slug)
	}
	cp := *t
	s.tenants[t.ID] = &cp
	s.tenantsBySlug[t.Slug] = t.ID
	return nil
}

func (s *MemoryStore) GetTenant(_ context.Context, id string) (*Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tenants[id]
	if !ok {
		return nil, fmt.Errorf("tenant '%s' not found", id)
	}
	cp := *t
	return &cp, nil
}

func (s *MemoryStore) GetTenantBySlug(_ context.Context, slug string) (*Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.tenantsBySlug[slug]
	if !ok {
		return nil, fmt.Errorf("tenant '%s' not found", slug)
	}
	cp := *s.tenants[id]
	return &cp, nil
}

func (s *MemoryStore) UpdateTenantLicense(ctx context.Context, id, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[id]
	if !ok {
		return fmt.Errorf("tenant '%s' not found", id)
	}
	t.LicenseToken = token
	return nil
}

func (s *MemoryStore) GetTenantIDPConfig(_ context.Context, tenantID string) (*IDPConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.idpConfigs[tenantID]
	if !ok {
		return nil, fmt.Errorf("idp config for tenant '%s' not found", tenantID)
	}
	cp := *cfg
	return &cp, nil
}

func (s *MemoryStore) UpsertTenantIDPConfig(_ context.Context, cfg *IDPConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *cfg
	s.idpConfigs[cfg.TenantID] = &cp
	return nil
}

func (s *MemoryStore) UpdateTenantBranding(_ context.Context, id, name, logoURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[id]
	if !ok {
		return fmt.Errorf("tenant %s not found", id)
	}
	t.Name = name
	t.LogoURL = logoURL
	return nil
}

func (s *MemoryStore) RecordAuditLog(_ context.Context, entry *AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *entry
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	s.auditLogs = append(s.auditLogs, &cp)
	return nil
}

func (s *MemoryStore) ListAuditLogs(_ context.Context, tenantID string, limit, offset int) ([]*AuditLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*AuditLog
	for _, l := range s.auditLogs {
		if l.TenantID == tenantID {
			out = append(out, l)
		}
	}
	// Simple slice logic for limit/offset
	start := offset
	if start > len(out) {
		return []*AuditLog{}, nil
	}
	end := start + limit
	if end > len(out) {
		end = len(out)
	}
	return out[start:end], nil
}

func (s *MemoryStore) ListTenants(_ context.Context) ([]*Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		cp := *t
		out = append(out, &cp)
	}
	return out, nil
}

func (s *MemoryStore) DeleteTenant(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[id]
	if !ok {
		return fmt.Errorf("tenant '%s' not found", id)
	}
	delete(s.tenantsBySlug, t.Slug)
	delete(s.tenants, id)
	return nil
}

func (s *MemoryStore) HasAnyTenant(_ context.Context) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tenants) > 0, nil
}

// -- WorkflowStore --

func (s *MemoryStore) SaveDefinition(ctx context.Context, def *asmtypes.WorkflowDef, yamlSource string) error {
	tenantID := TenantIDFromContext(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()

	// Auto-increment: find max version_number for this workflow.
	maxVer := 0
	prefix := tenantID + ":" + def.Metadata.Name + "@"
	for k, e := range s.definitions {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			if e.def.Metadata.VersionNumber > maxVer {
				maxVer = e.def.Metadata.VersionNumber
			}
		}
	}
	nextVer := maxVer + 1
	def.Metadata.VersionNumber = nextVer
	def.Metadata.Version = fmt.Sprintf("v%d", nextVer)

	s.definitions[tenantDefKey(tenantID, def.Metadata.Name, def.Metadata.Version)] = &defEntry{def: def, yaml: yamlSource}
	return nil
}

func (s *MemoryStore) UpdateDefinition(ctx context.Context, def *asmtypes.WorkflowDef, yamlSource string) error {
	tenantID := TenantIDFromContext(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()

	key := tenantDefKey(tenantID, def.Metadata.Name, def.Metadata.Version)
	if _, ok := s.definitions[key]; !ok {
		return fmt.Errorf("workflow definition '%s@%s' not found", def.Metadata.Name, def.Metadata.Version)
	}
	s.definitions[key] = &defEntry{def: def, yaml: yamlSource}
	return nil
}

func (s *MemoryStore) GetDefinition(ctx context.Context, name, version string) (*asmtypes.WorkflowDef, string, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.definitions[tenantDefKey(tenantID, name, version)]
	if !ok {
		return nil, "", fmt.Errorf("workflow definition '%s@%s' not found", name, version)
	}
	return e.def, e.yaml, nil
}

func (s *MemoryStore) GetDefinitionByVersion(ctx context.Context, name string, versionNumber int) (*asmtypes.WorkflowDef, string, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := tenantID + ":" + name + "@"
	for k, e := range s.definitions {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix && e.def.Metadata.VersionNumber == versionNumber {
			return e.def, e.yaml, nil
		}
	}
	return nil, "", fmt.Errorf("workflow '%s' version %d not found", name, versionNumber)
}

func (s *MemoryStore) GetLatestDefinition(ctx context.Context, name string) (*asmtypes.WorkflowDef, string, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := tenantID + ":" + name + "@"
	var best *defEntry
	for k, e := range s.definitions {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			if best == nil || e.def.Metadata.VersionNumber > best.def.Metadata.VersionNumber {
				best = e
			}
		}
	}
	if best == nil {
		return nil, "", fmt.Errorf("workflow '%s' not found", name)
	}
	return best.def, best.yaml, nil
}

func (s *MemoryStore) ListDefinitionVersions(ctx context.Context, name string) ([]VersionSummary, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := tenantID + ":" + name + "@"
	var out []VersionSummary
	for k, e := range s.definitions {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, VersionSummary{
				VersionNumber: e.def.Metadata.VersionNumber,
				Version:       e.def.Metadata.Version,
			})
		}
	}
	// Sort descending by version_number.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].VersionNumber > out[i].VersionNumber {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (s *MemoryStore) ListDefinitions(ctx context.Context, filter DefinitionFilter) ([]*asmtypes.WorkflowDef, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect only the latest version of each workflow.
	latest := make(map[string]*asmtypes.WorkflowDef) // key: workflow name
	prefix := tenantID + ":"
	for k, e := range s.definitions {
		if tenantID == "" || (len(k) >= len(prefix) && k[:len(prefix)] == prefix) {
			if filter.ReusableOnly && !e.def.Metadata.Reusable {
				continue
			}
			existing, ok := latest[e.def.Metadata.Name]
			if !ok || e.def.Metadata.VersionNumber > existing.Metadata.VersionNumber {
				latest[e.def.Metadata.Name] = e.def
			}
		}
	}
	out := make([]*asmtypes.WorkflowDef, 0, len(latest))
	for _, def := range latest {
		out = append(out, def)
	}

	if filter.Offset >= len(out) {
		return []*asmtypes.WorkflowDef{}, nil
	}
	if filter.Offset > 0 {
		out = out[filter.Offset:]
	}
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

func (s *MemoryStore) DeleteDefinition(ctx context.Context, name, version string) error {
	tenantID := TenantIDFromContext(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.definitions, tenantDefKey(tenantID, name, version))
	return nil
}

func (s *MemoryStore) CountDefinitions(ctx context.Context) (int, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Only count unique workflow names for the limit
	names := make(map[string]bool)
	prefix := tenantID + ":"
	for k := range s.definitions {
		if strings.HasPrefix(k, prefix) {
			// key format is tid:name@version
			parts := strings.Split(strings.TrimPrefix(k, prefix), "@")
			if len(parts) > 0 {
				names[parts[0]] = true
			}
		}
	}
	return len(names), nil
}

// -- RunStore --

func (s *MemoryStore) CreateRun(_ context.Context, run *asmtypes.WorkflowRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	cp := deepCopyRun(run)
	s.runs[run.ID] = cp
	return nil
}

func (s *MemoryStore) GetRun(ctx context.Context, id string) (*asmtypes.WorkflowRun, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[id]
	if !ok {
		return nil, fmt.Errorf("run '%s' not found", id)
	}
	if tenantID != "" && r.TenantID != tenantID {
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

func (s *MemoryStore) DeleteRun(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[id]; !ok {
		return fmt.Errorf("run '%s' not found", id)
	}
	delete(s.runs, id)
	delete(s.transitions, id)
	delete(s.hitl, id)
	return nil
}

func (s *MemoryStore) ListRuns(_ context.Context, filter RunFilter) ([]*asmtypes.WorkflowRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*asmtypes.WorkflowRun
	for _, r := range s.runs {
		if filter.TenantID != "" && r.TenantID != filter.TenantID {
			continue
		}
		if filter.WorkflowName != "" && r.WorkflowName != filter.WorkflowName {
			continue
		}
		if filter.Status != "" && r.Status != filter.Status {
			continue
		}
		if filter.CurrentState != "" && r.CurrentState != filter.CurrentState {
			continue
		}
		if filter.StartedFrom != nil && r.StartedAt.Before(*filter.StartedFrom) {
			continue
		}
		if filter.StartedTo != nil && r.StartedAt.After(*filter.StartedTo) {
			continue
		}
		out = append(out, deepCopyRun(r))
	}
	if filter.Offset >= len(out) {
		return []*asmtypes.WorkflowRun{}, nil
	}
	if filter.Offset > 0 {
		out = out[filter.Offset:]
	}
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
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

func (s *MemoryStore) CountRuns(ctx context.Context, filter RunFilter) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, r := range s.runs {
		if filter.TenantID != "" && r.TenantID != filter.TenantID {
			continue
		}
		if filter.WorkflowName != "" && r.WorkflowName != filter.WorkflowName {
			continue
		}
		if filter.Status != "" && r.Status != filter.Status {
			continue
		}
		if filter.StartedFrom != nil && r.StartedAt.Before(*filter.StartedFrom) {
			continue
		}
		if filter.StartedTo != nil && r.StartedAt.After(*filter.StartedTo) {
			continue
		}
		count++
	}
	return count, nil
}

func (s *MemoryStore) GetRunStats(ctx context.Context) (map[string]int, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]int)
	for _, r := range s.runs {
		if tenantID != "" && r.TenantID != tenantID {
			continue
		}
		out[string(r.Status)]++
	}
	return out, nil
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

func (s *MemoryStore) ListHITLs(ctx context.Context, filter HITLFilter) ([]*asmtypes.HITLRequest, error) {
	tenantID := filter.TenantID
	if tenantID == "" {
		tenantID = TenantIDFromContext(ctx)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*asmtypes.HITLRequest
	for _, r := range s.hitl {
		if filter.Resolved != nil && r.Resolved != *filter.Resolved {
			continue
		}
		if filter.Assignee != "" && r.Assignee != filter.Assignee {
			continue
		}
		// Filter by tenant if set: check the owning run's tenant_id.
		if tenantID != "" {
			run, ok := s.runs[r.RunID]
			if !ok || run.TenantID != tenantID {
				continue
			}
		}
		out = append(out, r)
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
	nameKey := tenantUserKey(u.TenantID, u.Username)
	if _, exists := s.usersByName[nameKey]; exists {
		return fmt.Errorf("user '%s' already exists in this tenant", u.Username)
	}
	cp := *u
	s.users[u.ID] = &cp
	s.usersByName[nameKey] = u.ID
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

func (s *MemoryStore) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.usersByName[tenantUserKey(tenantID, username)]
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
	delete(s.usersByName, tenantUserKey(u.TenantID, u.Username))
	delete(s.users, id)
	return nil
}

func (s *MemoryStore) ListUsers(ctx context.Context, filter UserFilter) ([]*User, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		if tenantID != "" && u.TenantID != tenantID {
			continue
		}
		cp := *u
		out = append(out, &cp)
	}
	if filter.Offset >= len(out) {
		return []*User{}, nil
	}
	if filter.Offset > 0 {
		out = out[filter.Offset:]
	}
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
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

// -- SystemSettingsStore --

func (s *MemoryStore) GetSystemSetting(_ context.Context, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.systemSettings[key]
	if !ok {
		return "", fmt.Errorf("system setting '%s' not found", key)
	}
	return val, nil
}

func (s *MemoryStore) SetSystemSetting(_ context.Context, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.systemSettings[key] = value
	return nil
}

// -- helpers --

// tenantDefKey builds the map key for a workflow definition.
func tenantDefKey(tenantID, name, version string) string {
	return tenantID + ":" + name + "@" + version
}

// tenantUserKey builds the map key for a username within a tenant.
func tenantUserKey(tenantID, username string) string {
	return tenantID + ":" + username
}

func deepCopyRun(r *asmtypes.WorkflowRun) *asmtypes.WorkflowRun {
	b, _ := json.Marshal(r)
	var cp asmtypes.WorkflowRun
	_ = json.Unmarshal(b, &cp)
	return &cp
}
