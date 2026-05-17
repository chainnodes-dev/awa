package store

import (
	"context"
	"fmt"
	"time"

	"github.com/asm-platform/asm/pkg/asmtypes"
	"github.com/google/uuid"
)

// MCPServerStore methods on MemoryStore.

func (s *MemoryStore) CreateMCPServer(_ context.Context, srv *MCPServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if srv.ID == "" {
		srv.ID = uuid.NewString()
	}
	if srv.CreatedAt.IsZero() {
		srv.CreatedAt = time.Now()
	}
	// Check unique name within tenant and upsert if exists
	var existingID string
	for id, existing := range s.mcpServers {
		if existing.TenantID == srv.TenantID && existing.Name == srv.Name {
			existingID = id
			break
		}
	}
	if existingID != "" {
		srv.ID = existingID
		cp := *srv
		s.mcpServers[existingID] = &cp
		return nil
	}
	cp := *srv
	s.mcpServers[srv.ID] = &cp
	return nil
}

func (s *MemoryStore) GetMCPServer(_ context.Context, id string) (*MCPServer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	srv, ok := s.mcpServers[id]
	if !ok {
		return nil, fmt.Errorf("MCP server %q not found", id)
	}
	cp := *srv
	return &cp, nil
}

func (s *MemoryStore) ListMCPServers(ctx context.Context) ([]*MCPServer, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*MCPServer
	for _, srv := range s.mcpServers {
		if tenantID == "" || srv.TenantID == tenantID {
			cp := *srv
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (s *MemoryStore) UpdateMCPServer(_ context.Context, srv *MCPServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.mcpServers[srv.ID]
	if !ok {
		return fmt.Errorf("MCP server %q not found", srv.ID)
	}
	existing.Name = srv.Name
	existing.Transport = srv.Transport
	existing.URL = srv.URL
	existing.Command = srv.Command
	existing.Args = srv.Args
	existing.EnvVars = srv.EnvVars
	existing.Description = srv.Description
	existing.DocURL = srv.DocURL
	return nil
}

func (s *MemoryStore) DeleteMCPServer(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.mcpServers[id]; !ok {
		return fmt.Errorf("MCP server %q not found", id)
	}
	delete(s.mcpServers, id)
	return nil
}

func (s *MemoryStore) HasAnyMCPServer(ctx context.Context) (bool, error) {
	tenantID := TenantIDFromContext(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, srv := range s.mcpServers {
		if tenantID == "" || srv.TenantID == tenantID {
			return true, nil
		}
	}
	return false, nil
}

func (s *MemoryStore) RecordMCPCall(_ context.Context, log *asmtypes.MCPAuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if log.ID == "" {
		log.ID = uuid.NewString()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	s.mcpLogs[log.RunID] = append(s.mcpLogs[log.RunID], log)
	return nil
}

func (s *MemoryStore) ListMCPCalls(_ context.Context, runID string) ([]*asmtypes.MCPAuditLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mcpLogs[runID], nil
}
