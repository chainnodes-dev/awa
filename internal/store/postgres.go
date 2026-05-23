package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// PostgresStore implements Store backed by PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}

func (s *PostgresStore) Pool() *pgxpool.Pool {
	return s.pool
}

// -- TenantStore --

func (s *PostgresStore) CreateTenant(ctx context.Context, t *Tenant) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO tenants (id, name, slug, license_token, created_at) VALUES ($1,$2,$3,$4,$5)`,
		t.ID, t.Name, t.Slug, t.LicenseToken, t.CreatedAt,
	)
	return err
}

func (s *PostgresStore) GetTenant(ctx context.Context, id string) (*Tenant, error) {
	var t Tenant
	var logoURL, licenseToken *string
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, logo_url, license_token, created_at FROM tenants WHERE id=$1`, id,
	).Scan(&t.ID, &t.Name, &t.Slug, &logoURL, &licenseToken, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("tenant '%s' not found: %w", id, err)
	}
	if logoURL != nil {
		t.LogoURL = *logoURL
	}
	if licenseToken != nil {
		t.LicenseToken = *licenseToken
	}
	return &t, nil
}

func (s *PostgresStore) GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error) {
	var t Tenant
	var logoURL, licenseToken *string
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, logo_url, license_token, created_at FROM tenants WHERE slug=$1`, slug,
	).Scan(&t.ID, &t.Name, &t.Slug, &logoURL, &licenseToken, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("tenant '%s' not found: %w", slug, err)
	}
	if logoURL != nil {
		t.LogoURL = *logoURL
	}
	if licenseToken != nil {
		t.LicenseToken = *licenseToken
	}
	return &t, nil
}

func (s *PostgresStore) ListTenants(ctx context.Context) ([]*Tenant, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, slug, logo_url, license_token, created_at FROM tenants ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Tenant
	for rows.Next() {
		var t Tenant
		var logoURL, licenseToken *string
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &logoURL, &licenseToken, &t.CreatedAt); err != nil {
			return nil, err
		}
		if logoURL != nil {
			t.LogoURL = *logoURL
		}
		if licenseToken != nil {
			t.LicenseToken = *licenseToken
		}
		out = append(out, &t)
	}
	return out, nil
}


func (s *PostgresStore) DeleteTenant(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1`, id)
	return err
}

func (s *PostgresStore) UpdateTenantBranding(ctx context.Context, id, name, logoURL string) error {
	_, err := s.pool.Exec(ctx, `UPDATE tenants SET name=$2, logo_url=$3 WHERE id=$1`, id, name, logoURL)
	return err
}


func (s *PostgresStore) UpdateTenantLicense(ctx context.Context, id, token string) error {
	_, err := s.pool.Exec(ctx, `UPDATE tenants SET license_token=$2 WHERE id=$1`, id, token)
	return err
}

func (s *PostgresStore) GetTenantIDPConfig(ctx context.Context, tenantID string) (*IDPConfig, error) {
	var cfg IDPConfig
	var roleMapping []byte
	err := s.pool.QueryRow(ctx,
		`SELECT tenant_id, issuer_url, client_id, client_secret, role_mapping, active FROM tenant_idp_configs WHERE tenant_id=$1`,
		tenantID,
	).Scan(&cfg.TenantID, &cfg.IssuerURL, &cfg.ClientID, &cfg.ClientSecret, &roleMapping, &cfg.Active)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(roleMapping, &cfg.RoleMapping); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *PostgresStore) UpsertTenantIDPConfig(ctx context.Context, cfg *IDPConfig) error {
	roleMapping, _ := json.Marshal(cfg.RoleMapping)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO tenant_idp_configs (tenant_id, issuer_url, client_id, client_secret, role_mapping, active)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (tenant_id) DO UPDATE SET
		 issuer_url = EXCLUDED.issuer_url,
		 client_id = EXCLUDED.client_id,
		 client_secret = EXCLUDED.client_secret,
		 role_mapping = EXCLUDED.role_mapping,
		 active = EXCLUDED.active,
		 updated_at = CURRENT_TIMESTAMP`,
		cfg.TenantID, cfg.IssuerURL, cfg.ClientID, cfg.ClientSecret, roleMapping, cfg.Active,
	)
	return err
}

func (s *PostgresStore) RecordAuditLog(ctx context.Context, entry *AuditLog) error {
	details, _ := json.Marshal(entry.Details)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_logs (tenant_id, user_id, action, details, ip_address) VALUES ($1,$2,$3,$4,$5)`,
		entry.TenantID, entry.UserID, entry.Action, details, entry.IPAddress,
	)
	return err
}

func (s *PostgresStore) ListAuditLogs(ctx context.Context, tenantID string, limit, offset int) ([]*AuditLog, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, user_id, action, details, ip_address, created_at
		 FROM audit_logs WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*AuditLog
	for rows.Next() {
		var l AuditLog
		var details []byte
		err := rows.Scan(&l.ID, &l.TenantID, &l.UserID, &l.Action, &details, &l.IPAddress, &l.CreatedAt)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal(details, &l.Details)
		out = append(out, &l)
	}
	return out, nil
}

func (s *PostgresStore) HasAnyTenant(ctx context.Context) (bool, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM tenants`).Scan(&count)
	return count > 0, err
}

// -- WorkflowStore --
// tenant_id is read from context when set; omitted for internal operations.

func (s *PostgresStore) SaveDefinition(ctx context.Context, def *asmtypes.WorkflowDef, yamlSource string) error {
	tenantID := TenantIDFromContext(ctx)

	// Auto-increment: SELECT MAX(version_number) for this workflow, then +1.
	var maxVer *int
	err := s.pool.QueryRow(ctx,
		`SELECT MAX(version_number) FROM workflow_definitions WHERE tenant_id=$1 AND name=$2`,
		tenantID, def.Metadata.Name,
	).Scan(&maxVer)
	if err != nil {
		return fmt.Errorf("query max version: %w", err)
	}
	nextVer := 1
	if maxVer != nil {
		nextVer = *maxVer + 1
	}

	def.Metadata.VersionNumber = nextVer
	// Keep the text version field in sync for human readability.
	def.Metadata.Version = fmt.Sprintf("v%d", nextVer)

	// Marshal AFTER updating version so the JSONB definition column is consistent
	// with the version/version_number columns.
	defJSON, err := json.Marshal(def)
	if err != nil {
		return err
	}

	inputsJSON, _ := json.Marshal(def.Inputs)
	outputsJSON, _ := json.Marshal(def.Outputs)
	capsJSON, _ := json.Marshal(def.Capabilities)

	_, err = s.pool.Exec(ctx, `
		INSERT INTO workflow_definitions (name, version, version_number, definition, yaml_source, process_description, inputs, outputs, capabilities, reusable, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		def.Metadata.Name, def.Metadata.Version, nextVer, defJSON, yamlSource, def.Metadata.ProcessDescription,
		inputsJSON, outputsJSON, capsJSON, def.Metadata.Reusable, tenantID,
	)
	return err
}

func (s *PostgresStore) UpdateDefinition(ctx context.Context, def *asmtypes.WorkflowDef, yamlSource string) error {
	tenantID := TenantIDFromContext(ctx)

	defJSON, err := json.Marshal(def)
	if err != nil {
		return err
	}

	inputsJSON, _ := json.Marshal(def.Inputs)
	outputsJSON, _ := json.Marshal(def.Outputs)
	capsJSON, _ := json.Marshal(def.Capabilities)

	query := `
		UPDATE workflow_definitions 
		SET definition = $1, yaml_source = $2, process_description = $3, inputs = $4, outputs = $5, capabilities = $6, reusable = $7
		WHERE name = $8 AND version = $9`
	args := []interface{}{
		defJSON, yamlSource, def.Metadata.ProcessDescription,
		inputsJSON, outputsJSON, capsJSON, def.Metadata.Reusable,
		def.Metadata.Name, def.Metadata.Version,
	}
	if tenantID != "" {
		query += " AND tenant_id = $10"
		args = append(args, tenantID)
	}

	res, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("workflow definition '%s@%s' not found", def.Metadata.Name, def.Metadata.Version)
	}
	return nil
}

func (s *PostgresStore) GetDefinition(ctx context.Context, name, version string) (*asmtypes.WorkflowDef, string, error) {
	tenantID := TenantIDFromContext(ctx)
	query := `SELECT definition, yaml_source, version_number, version FROM workflow_definitions WHERE name=$1 AND version=$2`
	args := []interface{}{name, version}
	if tenantID != "" {
		query += " AND tenant_id=$3"
		args = append(args, tenantID)
	}
	var defJSON []byte
	var yamlSource string
	var versionNumber int
	err := s.pool.QueryRow(ctx, query, args...).Scan(&defJSON, &yamlSource, &versionNumber, &version)
	if err != nil {
		return nil, "", fmt.Errorf("workflow '%s@%s' not found: %w", name, version, err)
	}
	var def asmtypes.WorkflowDef
	if err := json.Unmarshal(defJSON, &def); err != nil {
		return nil, "", err
	}
	def.Metadata.VersionNumber = versionNumber
	def.Metadata.Version = version
	return &def, yamlSource, nil
}

func (s *PostgresStore) GetDefinitionByVersion(ctx context.Context, name string, versionNumber int) (*asmtypes.WorkflowDef, string, error) {
	tenantID := TenantIDFromContext(ctx)
	query := `SELECT definition, yaml_source, version_number, version FROM workflow_definitions WHERE name=$1 AND version_number=$2`
	args := []interface{}{name, versionNumber}
	if tenantID != "" {
		query += " AND tenant_id=$3"
		args = append(args, tenantID)
	}
	var defJSON []byte
	var yamlSource string
	var vnum int
	var ver string
	err := s.pool.QueryRow(ctx, query, args...).Scan(&defJSON, &yamlSource, &vnum, &ver)
	if err != nil {
		return nil, "", fmt.Errorf("workflow '%s' version %d not found: %w", name, versionNumber, err)
	}
	var def asmtypes.WorkflowDef
	if err := json.Unmarshal(defJSON, &def); err != nil {
		return nil, "", err
	}
	def.Metadata.VersionNumber = vnum
	def.Metadata.Version = ver
	return &def, yamlSource, nil
}

func (s *PostgresStore) GetLatestDefinition(ctx context.Context, name string) (*asmtypes.WorkflowDef, string, error) {
	tenantID := TenantIDFromContext(ctx)
	query := `SELECT definition, yaml_source, version_number, version FROM workflow_definitions WHERE name=$1`
	args := []interface{}{name}
	if tenantID != "" {
		query += " AND tenant_id=$2"
		args = append(args, tenantID)
	}
	query += " ORDER BY version_number DESC LIMIT 1"
	var defJSON []byte
	var yamlSource string
	var versionNumber int
	var ver string
	err := s.pool.QueryRow(ctx, query, args...).Scan(&defJSON, &yamlSource, &versionNumber, &ver)
	if err != nil {
		return nil, "", fmt.Errorf("workflow '%s' not found: %w", name, err)
	}
	var def asmtypes.WorkflowDef
	if err := json.Unmarshal(defJSON, &def); err != nil {
		return nil, "", err
	}
	def.Metadata.VersionNumber = versionNumber
	def.Metadata.Version = ver
	return &def, yamlSource, nil
}

func (s *PostgresStore) ListDefinitions(ctx context.Context, filter DefinitionFilter) ([]*asmtypes.WorkflowDef, error) {
	tenantID := TenantIDFromContext(ctx)
	query := `SELECT DISTINCT ON (name) definition, version_number, version FROM workflow_definitions`
	var args []interface{}
	i := 1
	var where []string
	if tenantID != "" {
		where = append(where, fmt.Sprintf("tenant_id=$%d", i))
		args = append(args, tenantID)
		i++
	}
	if filter.ReusableOnly {
		where = append(where, "reusable=TRUE")
	}

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	query += " ORDER BY name, version_number DESC"
	// Wrap in a subquery so LIMIT/OFFSET apply after DISTINCT ON.
	if filter.Limit > 0 || filter.Offset > 0 {
		outer := fmt.Sprintf("SELECT definition, version_number, version FROM (%s) sub", query)
		if filter.Limit > 0 {
			outer += fmt.Sprintf(" LIMIT $%d", i)
			args = append(args, filter.Limit)
			i++
		}
		if filter.Offset > 0 {
			outer += fmt.Sprintf(" OFFSET $%d", i)
			args = append(args, filter.Offset)
		}
		query = outer
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*asmtypes.WorkflowDef
	for rows.Next() {
		var defJSON []byte
		var vnum int
		var ver string
		if err := rows.Scan(&defJSON, &vnum, &ver); err != nil {
			return nil, err
		}
		var def asmtypes.WorkflowDef
		if err := json.Unmarshal(defJSON, &def); err != nil {
			return nil, err
		}
		def.Metadata.VersionNumber = vnum
		def.Metadata.Version = ver // override JSONB value with SQL column (may differ for pre-migration rows)
		out = append(out, &def)
	}
	return out, nil
}

func (s *PostgresStore) ListDefinitionVersions(ctx context.Context, name string) ([]VersionSummary, error) {
	tenantID := TenantIDFromContext(ctx)
	query := `SELECT version_number, version, created_at FROM workflow_definitions WHERE name=$1`
	args := []interface{}{name}
	if tenantID != "" {
		query += " AND tenant_id=$2"
		args = append(args, tenantID)
	}
	query += " ORDER BY version_number DESC"
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []VersionSummary
	for rows.Next() {
		var v VersionSummary
		if err := rows.Scan(&v.VersionNumber, &v.Version, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *PostgresStore) DeleteDefinition(ctx context.Context, name, version string) error {
	tenantID := TenantIDFromContext(ctx)
	query := `DELETE FROM workflow_definitions WHERE name=$1 AND version=$2`
	args := []interface{}{name, version}
	if tenantID != "" {
		query += " AND tenant_id=$3"
		args = append(args, tenantID)
	}
	_, err := s.pool.Exec(ctx, query, args...)
	return err
}

func (s *PostgresStore) CountDefinitions(ctx context.Context) (int, error) {
	tenantID := TenantIDFromContext(ctx)
	query := `SELECT COUNT(DISTINCT name) FROM workflow_definitions`
	var args []interface{}
	if tenantID != "" {
		query += " WHERE tenant_id=$1"
		args = append(args, tenantID)
	}
	var count int
	err := s.pool.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

// -- RunStore --

func (s *PostgresStore) CreateRun(ctx context.Context, run *asmtypes.WorkflowRun) error {
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	bb, _ := json.Marshal(run.Blackboard)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO workflow_runs
		  (id, tenant_id, workflow_name, workflow_version, status, current_state, blackboard, temporal_workflow_id, started_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		run.ID, run.TenantID, run.WorkflowName, run.WorkflowVersion, string(run.Status),
		run.CurrentState, bb, run.TemporalID, run.StartedAt, run.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) GetRun(ctx context.Context, id string) (*asmtypes.WorkflowRun, error) {
	tenantID := TenantIDFromContext(ctx)
	query := `
		SELECT id, tenant_id, workflow_name, workflow_version, status, current_state,
		       blackboard, coalesce(temporal_workflow_id,''), coalesce(failure_reason,''),
		       started_at, updated_at, completed_at
		FROM workflow_runs WHERE id=$1`
	args := []interface{}{id}
	if tenantID != "" {
		query += " AND tenant_id=$2"
		args = append(args, tenantID)
	}
	var r asmtypes.WorkflowRun
	var status string
	var bb []byte
	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&r.ID, &r.TenantID, &r.WorkflowName, &r.WorkflowVersion, &status, &r.CurrentState,
		&bb, &r.TemporalID, &r.FailureReason, &r.StartedAt, &r.UpdatedAt, &r.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("run '%s' not found: %w", id, err)
	}
	r.Status = asmtypes.RunStatus(status)
	_ = json.Unmarshal(bb, &r.Blackboard)
	return &r, nil
}

func (s *PostgresStore) UpdateRun(ctx context.Context, run *asmtypes.WorkflowRun) error {
	bb, _ := json.Marshal(run.Blackboard)
	run.UpdatedAt = time.Now()
	// COALESCE(NULLIF($8,''), temporal_workflow_id) preserves the existing
	// temporal_workflow_id when the caller passes an empty string (most activity
	// updates), while still persisting it on the first UpdateRun call after
	// StartWorkflow returns the ID.
	_, err := s.pool.Exec(ctx, `
		UPDATE workflow_runs
		SET status=$2, current_state=$3, blackboard=$4, updated_at=$5, completed_at=$6, failure_reason=$7,
		    temporal_workflow_id = COALESCE(NULLIF($8, ''), temporal_workflow_id)
		WHERE id=$1`,
		run.ID, string(run.Status), run.CurrentState, bb, run.UpdatedAt, run.CompletedAt, run.FailureReason,
		run.TemporalID,
	)
	return err
}

func (s *PostgresStore) DeleteRun(ctx context.Context, id string) error {
	tenantID := TenantIDFromContext(ctx)
	query := `DELETE FROM workflow_runs WHERE id=$1`
	args := []interface{}{id}
	if tenantID != "" {
		query += " AND tenant_id=$2"
		args = append(args, tenantID)
	}
	_, err := s.pool.Exec(ctx, query, args...)
	return err
}

func (s *PostgresStore) ListRuns(ctx context.Context, filter RunFilter) ([]*asmtypes.WorkflowRun, error) {
	query := `SELECT id, tenant_id, workflow_name, workflow_version, status, current_state,
	                 blackboard, coalesce(temporal_workflow_id,''), coalesce(failure_reason,''),
	                 started_at, updated_at, completed_at
	          FROM workflow_runs WHERE TRUE`
	args := []interface{}{}
	i := 1
	if filter.TenantID != "" {
		query += fmt.Sprintf(" AND tenant_id=$%d", i)
		args = append(args, filter.TenantID)
		i++
	}
	if filter.WorkflowName != "" {
		query += fmt.Sprintf(" AND workflow_name=$%d", i)
		args = append(args, filter.WorkflowName)
		i++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status=$%d", i)
		args = append(args, string(filter.Status))
		i++
	}
	if filter.CurrentState != "" {
		query += fmt.Sprintf(" AND current_state=$%d", i)
		args = append(args, filter.CurrentState)
		i++
	}
	if filter.StartedFrom != nil {
		query += fmt.Sprintf(" AND started_at >= $%d", i)
		args = append(args, *filter.StartedFrom)
		i++
	}
	if filter.StartedTo != nil {
		query += fmt.Sprintf(" AND started_at <= $%d", i)
		args = append(args, *filter.StartedTo)
		i++
	}
	query += " ORDER BY started_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", i)
		args = append(args, filter.Limit)
		i++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", i)
		args = append(args, filter.Offset)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*asmtypes.WorkflowRun
	for rows.Next() {
		var r asmtypes.WorkflowRun
		var status string
		var bb []byte
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.WorkflowName, &r.WorkflowVersion, &status, &r.CurrentState,
			&bb, &r.TemporalID, &r.FailureReason, &r.StartedAt, &r.UpdatedAt, &r.CompletedAt,
		); err != nil {
			return nil, err
		}
		r.Status = asmtypes.RunStatus(status)
		_ = json.Unmarshal(bb, &r.Blackboard)
		out = append(out, &r)
	}
	return out, nil
}

func (s *PostgresStore) RecordTransition(ctx context.Context, rec *asmtypes.TransitionRecord) error {
	if rec.ID == "" {
		rec.ID = uuid.NewString()
	}
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now()
	}
	snap, _ := json.Marshal(rec.BlackboardSnapshot)
	ao, _ := json.Marshal(rec.AgentOutput)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO state_transitions (id, run_id, from_state, to_state, trigger, blackboard_snapshot, agent_output, timestamp)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		rec.ID, rec.RunID, rec.FromState, rec.ToState, rec.Trigger, snap, ao, rec.Timestamp,
	)
	return err
}

func (s *PostgresStore) ListTransitions(ctx context.Context, runID string) ([]*asmtypes.TransitionRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, run_id, from_state, to_state, trigger, blackboard_snapshot, agent_output, timestamp
		FROM state_transitions WHERE run_id=$1 ORDER BY timestamp ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*asmtypes.TransitionRecord
	for rows.Next() {
		var r asmtypes.TransitionRecord
		var snap, ao []byte
		if err := rows.Scan(&r.ID, &r.RunID, &r.FromState, &r.ToState, &r.Trigger, &snap, &ao, &r.Timestamp); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(snap, &r.BlackboardSnapshot)
		if ao != nil {
			var output asmtypes.AgentOutput
			_ = json.Unmarshal(ao, &output)
			r.AgentOutput = &output
		}
		out = append(out, &r)
	}
	return out, nil
}

func (s *PostgresStore) CountRuns(ctx context.Context, filter RunFilter) (int, error) {
	query := `SELECT COUNT(*) FROM workflow_runs WHERE TRUE`
	var args []interface{}
	i := 1
	if filter.TenantID != "" {
		query += fmt.Sprintf(" AND tenant_id=$%d", i)
		args = append(args, filter.TenantID)
		i++
	}
	if filter.WorkflowName != "" {
		query += fmt.Sprintf(" AND workflow_name=$%d", i)
		args = append(args, filter.WorkflowName)
		i++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status=$%d", i)
		args = append(args, string(filter.Status))
		i++
	}
	if filter.StartedFrom != nil {
		query += fmt.Sprintf(" AND started_at >= $%d", i)
		args = append(args, *filter.StartedFrom)
		i++
	}
	if filter.StartedTo != nil {
		query += fmt.Sprintf(" AND started_at <= $%d", i)
		args = append(args, *filter.StartedTo)
		i++
	}

	var count int
	err := s.pool.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

func (s *PostgresStore) GetRunStats(ctx context.Context) (map[string]int, error) {
	tenantID := TenantIDFromContext(ctx)
	rows, err := s.pool.Query(ctx,
		`SELECT status, count(*) FROM workflow_runs WHERE tenant_id=$1 GROUP BY status`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		out[status] = count
	}
	return out, nil
}

// -- HITLStore --

func (s *PostgresStore) CreateHITL(ctx context.Context, req *asmtypes.HITLRequest) error {
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	meta, _ := json.Marshal(req.Metadata)
	fs, _ := json.Marshal(req.FormSchema)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO hitl_requests (id, run_id, state_name, assignee, timeout_at, metadata, form_schema, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		req.ID, req.RunID, req.StateName, req.Assignee, req.TimeoutAt, meta, fs, req.CreatedAt,
	)
	return err
}

func (s *PostgresStore) GetHITL(ctx context.Context, runID string) (*asmtypes.HITLRequest, error) {
	var req asmtypes.HITLRequest
	var meta, fs, bb, bbs []byte
	err := s.pool.QueryRow(ctx, `
		SELECT hr.id, hr.run_id, hr.state_name, coalesce(hr.assignee,''), hr.timeout_at, hr.resolved,
		       hr.resolved_at, coalesce(hr.resolution,''), coalesce(hr.resolver,''), hr.metadata, hr.form_schema, hr.created_at,
		       wr.workflow_name, wr.blackboard,
		       w.definition->'blackboard'->'schema'
		FROM hitl_requests hr
		JOIN workflow_runs wr ON wr.id = hr.run_id
		JOIN workflow_definitions w ON w.name = wr.workflow_name AND w.version = wr.workflow_version
		WHERE hr.run_id=$1 AND hr.resolved=FALSE ORDER BY hr.created_at DESC LIMIT 1`, runID,
	).Scan(&req.ID, &req.RunID, &req.StateName, &req.Assignee, &req.TimeoutAt, &req.Resolved,
		&req.ResolvedAt, &req.Resolution, &req.Resolver, &meta, &fs, &req.CreatedAt,
		&req.WorkflowName, &bb, &bbs)
	if err != nil {
		return nil, fmt.Errorf("no HITL request for run '%s': %w", runID, err)
	}
	_ = json.Unmarshal(meta, &req.Metadata)
	_ = json.Unmarshal(fs, &req.FormSchema)
	_ = json.Unmarshal(bb, &req.Blackboard)
	_ = json.Unmarshal(bbs, &req.BlackboardSchema)
	return &req, nil
}

func (s *PostgresStore) ResolveHITL(ctx context.Context, runID, resolution, resolver string) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx, `
		UPDATE hitl_requests
		SET resolved=TRUE, resolved_at=$2, resolution=$3, resolver=$4
		WHERE run_id=$1 AND resolved=FALSE`,
		runID, now, resolution, resolver,
	)
	return err
}

func (s *PostgresStore) ListHITLs(ctx context.Context, filter HITLFilter) ([]*asmtypes.HITLRequest, error) {
	tenantID := filter.TenantID
	if tenantID == "" {
		tenantID = TenantIDFromContext(ctx)
	}

	query := `
		SELECT hr.id, hr.run_id, hr.state_name, coalesce(hr.assignee,''), hr.timeout_at, hr.metadata, hr.form_schema, hr.created_at,
		       wr.workflow_name, wr.blackboard,
		       w.definition->'blackboard'->'schema'
		FROM hitl_requests hr
		JOIN workflow_runs wr ON wr.id = hr.run_id
		JOIN workflow_definitions w ON w.name = wr.workflow_name AND w.version = wr.workflow_version
		WHERE TRUE`
	var args []interface{}
	i := 1

	if tenantID != "" {
		query += fmt.Sprintf(" AND wr.tenant_id=$%d", i)
		args = append(args, tenantID)
		i++
	}
	if filter.Assignee != "" {
		query += fmt.Sprintf(" AND hr.assignee=$%d", i)
		args = append(args, filter.Assignee)
		i++
	}
	if filter.Resolved != nil {
		query += fmt.Sprintf(" AND hr.resolved=$%d", i)
		args = append(args, *filter.Resolved)
		i++
	}

	query += " ORDER BY hr.created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*asmtypes.HITLRequest
	for rows.Next() {
		var req asmtypes.HITLRequest
		var meta, fs, bb, bbs []byte
		err := rows.Scan(&req.ID, &req.RunID, &req.StateName, &req.Assignee, &req.TimeoutAt, &meta, &fs, &req.CreatedAt,
			&req.WorkflowName, &bb, &bbs)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal(meta, &req.Metadata)
		_ = json.Unmarshal(fs, &req.FormSchema)
		_ = json.Unmarshal(bb, &req.Blackboard)
		_ = json.Unmarshal(bbs, &req.BlackboardSchema)
		out = append(out, &req)
	}
	return out, nil
}


// -- UserStore --

func (s *PostgresStore) CreateUser(ctx context.Context, u *User) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO users (id, tenant_id, username, password_hash, role, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		u.ID, u.TenantID, u.Username, u.PasswordHash, u.Role, u.CreatedAt,
	)
	return err
}

func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, username, password_hash, role, created_at FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("user '%s' not found: %w", id, err)
	}
	return &u, nil
}

func (s *PostgresStore) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	tenantID := TenantIDFromContext(ctx)
	var u User
	var err error
	if tenantID != "" {
		err = s.pool.QueryRow(ctx,
			`SELECT id, tenant_id, username, password_hash, role, created_at
			 FROM users WHERE tenant_id=$1 AND username=$2`,
			tenantID, username,
		).Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	} else {
		err = s.pool.QueryRow(ctx,
			`SELECT id, tenant_id, username, password_hash, role, created_at
			 FROM users WHERE username=$1`,
			username,
		).Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	}
	if err != nil {
		return nil, fmt.Errorf("user '%s' not found: %w", username, err)
	}
	return &u, nil
}

func (s *PostgresStore) UpdateUserRole(ctx context.Context, id, role string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET role=$2 WHERE id=$1`, id, role)
	return err
}

func (s *PostgresStore) DeleteUser(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	return err
}

func (s *PostgresStore) ListUsers(ctx context.Context, filter UserFilter) ([]*User, error) {
	tenantID := TenantIDFromContext(ctx)
	query := `SELECT id, tenant_id, username, role, created_at FROM users`
	var args []interface{}
	i := 1
	if tenantID != "" {
		query += fmt.Sprintf(" WHERE tenant_id=$%d", i)
		args = append(args, tenantID)
		i++
	}
	query += " ORDER BY created_at ASC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", i)
		args = append(args, filter.Limit)
		i++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", i)
		args = append(args, filter.Offset)
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, nil
}

func (s *PostgresStore) HasAnyUser(ctx context.Context) (bool, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&count)
	return count > 0, err
}

// -- RefreshTokenStore --

func (s *PostgresStore) CreateRefreshToken(ctx context.Context, t *RefreshToken) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		t.ID, t.UserID, t.TokenHash, t.ExpiresAt, t.CreatedAt,
	)
	return err
}

func (s *PostgresStore) GetRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	var t RefreshToken
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked, created_at
		 FROM refresh_tokens WHERE token_hash=$1`, hash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Revoked, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found: %w", err)
	}
	return &t, nil
}

func (s *PostgresStore) RevokeRefreshToken(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked=TRUE WHERE id=$1`, id)
	return err
}

func (s *PostgresStore) RevokeAllUserTokens(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked=TRUE WHERE user_id=$1`, userID)
	return err
}

// -- SystemSettingsStore --

func (s *PostgresStore) GetSystemSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.pool.QueryRow(ctx, `SELECT value FROM system_settings WHERE key=$1`, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *PostgresStore) SetSystemSetting(ctx context.Context, key, value string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO system_settings (key, value, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value, updated_at=CURRENT_TIMESTAMP`,
		key, value,
	)
	return err
}

// Ping checks that the database connection is healthy.
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
