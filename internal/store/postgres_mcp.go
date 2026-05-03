package store

import (
	"context"
	"fmt"
	"time"

	"github.com/asm-platform/asm/pkg/asmtypes"
	"github.com/google/uuid"
)

func (s *PostgresStore) CreateMCPServer(ctx context.Context, srv *MCPServer) error {
	if srv.ID == "" {
		srv.ID = uuid.NewString()
	}
	if srv.CreatedAt.IsZero() {
		srv.CreatedAt = time.Now()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO mcp_servers (id, tenant_id, name, transport, url, command, args, env_var, env_value, description, created_by, created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		srv.ID, srv.TenantID, srv.Name, srv.Transport, srv.URL, srv.Command, srv.Args, srv.EnvVar, srv.EnvValue, srv.Description, srv.CreatedBy, srv.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create MCP server: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetMCPServer(ctx context.Context, id string) (*MCPServer, error) {
	var srv MCPServer
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, transport, COALESCE(url, ''), COALESCE(command, ''), COALESCE(args, '{}'), COALESCE(env_var, ''), COALESCE(env_value, ''), COALESCE(description, ''), created_by, created_at
         FROM mcp_servers WHERE id=$1`, id,
	).Scan(&srv.ID, &srv.TenantID, &srv.Name, &srv.Transport, &srv.URL, &srv.Command, &srv.Args, &srv.EnvVar, &srv.EnvValue, &srv.Description, &srv.CreatedBy, &srv.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("MCP server %q not found: %w", id, err)
	}
	return &srv, nil
}

func (s *PostgresStore) ListMCPServers(ctx context.Context) ([]*MCPServer, error) {
	tenantID := TenantIDFromContext(ctx)
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, name, transport, COALESCE(url, ''), COALESCE(command, ''), COALESCE(args, '{}'), COALESCE(env_var, ''), COALESCE(env_value, ''), COALESCE(description, ''), created_by, created_at
         FROM mcp_servers WHERE tenant_id=$1 ORDER BY name`, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list MCP servers: %w", err)
	}
	defer rows.Close()
	var out []*MCPServer
	for rows.Next() {
		var srv MCPServer
		if err := rows.Scan(&srv.ID, &srv.TenantID, &srv.Name, &srv.Transport, &srv.URL, &srv.Command, &srv.Args, &srv.EnvVar, &srv.EnvValue,
			&srv.Description, &srv.CreatedBy, &srv.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &srv)
	}
	return out, rows.Err()
}

func (s *PostgresStore) UpdateMCPServer(ctx context.Context, srv *MCPServer) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE mcp_servers SET name=$1, transport=$2, url=$3, command=$4, args=$5, env_var=$6, env_value=$7, description=$8 WHERE id=$9`,
		srv.Name, srv.Transport, srv.URL, srv.Command, srv.Args, srv.EnvVar, srv.EnvValue, srv.Description, srv.ID,
	)
	if err != nil {
		return fmt.Errorf("update MCP server: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteMCPServer(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM mcp_servers WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete MCP server: %w", err)
	}
	return nil
}

func (s *PostgresStore) HasAnyMCPServer(ctx context.Context) (bool, error) {
	tenantID := TenantIDFromContext(ctx)
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM mcp_servers WHERE tenant_id=$1 LIMIT 1`, tenantID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("has any MCP server: %w", err)
	}
	return count > 0, nil
}

func (s *PostgresStore) RecordMCPCall(ctx context.Context, log *asmtypes.MCPAuditLog) error {
	if log.ID == "" {
		log.ID = uuid.NewString()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO mcp_audit_logs (
			id, run_id, state_name, agent_name, server_url, method, tool_name, 
			input, output, is_error, error_msg, duration_ms, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		log.ID, log.RunID, log.StateName, log.AgentName, log.ServerURL, log.Method, log.ToolName,
		log.Input, log.Output, log.IsError, log.ErrorMsg, log.DurationMs, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("record MCP call: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListMCPCalls(ctx context.Context, runID string) ([]*asmtypes.MCPAuditLog, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, run_id, state_name, agent_name, server_url, method, tool_name, 
		        input, output, is_error, error_msg, duration_ms, created_at
		 FROM mcp_audit_logs WHERE run_id=$1 ORDER BY created_at ASC`, runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list MCP calls: %w", err)
	}
	defer rows.Close()

	var out []*asmtypes.MCPAuditLog
	for rows.Next() {
		log := &asmtypes.MCPAuditLog{}
		err := rows.Scan(
			&log.ID, &log.RunID, &log.StateName, &log.AgentName, &log.ServerURL, &log.Method, &log.ToolName,
			&log.Input, &log.Output, &log.IsError, &log.ErrorMsg, &log.DurationMs, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, log)
	}
	return out, rows.Err()
}
