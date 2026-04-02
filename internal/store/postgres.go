package store

import (
	"context"
	"encoding/json"
	"fmt"
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

// -- WorkflowStore --

func (s *PostgresStore) SaveDefinition(ctx context.Context, def *asmtypes.WorkflowDef, yamlSource string) error {
	defJSON, err := json.Marshal(def)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO workflow_definitions (name, version, definition, yaml_source)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (name, version) DO UPDATE
		  SET definition = EXCLUDED.definition,
		      yaml_source = EXCLUDED.yaml_source`,
		def.Metadata.Name, def.Metadata.Version, defJSON, yamlSource,
	)
	return err
}

func (s *PostgresStore) GetDefinition(ctx context.Context, name, version string) (*asmtypes.WorkflowDef, string, error) {
	var defJSON []byte
	var yamlSource string
	err := s.pool.QueryRow(ctx,
		`SELECT definition, yaml_source FROM workflow_definitions WHERE name=$1 AND version=$2`,
		name, version,
	).Scan(&defJSON, &yamlSource)
	if err != nil {
		return nil, "", fmt.Errorf("workflow '%s@%s' not found: %w", name, version, err)
	}
	var def asmtypes.WorkflowDef
	if err := json.Unmarshal(defJSON, &def); err != nil {
		return nil, "", err
	}
	return &def, yamlSource, nil
}

func (s *PostgresStore) ListDefinitions(ctx context.Context) ([]*asmtypes.WorkflowDef, error) {
	rows, err := s.pool.Query(ctx, `SELECT definition FROM workflow_definitions ORDER BY name, version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*asmtypes.WorkflowDef
	for rows.Next() {
		var defJSON []byte
		if err := rows.Scan(&defJSON); err != nil {
			return nil, err
		}
		var def asmtypes.WorkflowDef
		if err := json.Unmarshal(defJSON, &def); err != nil {
			return nil, err
		}
		out = append(out, &def)
	}
	return out, nil
}

func (s *PostgresStore) DeleteDefinition(ctx context.Context, name, version string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM workflow_definitions WHERE name=$1 AND version=$2`, name, version)
	return err
}

// -- RunStore --

func (s *PostgresStore) CreateRun(ctx context.Context, run *asmtypes.WorkflowRun) error {
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	bb, _ := json.Marshal(run.Blackboard)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO workflow_runs
		  (id, workflow_name, workflow_version, status, current_state, blackboard, temporal_workflow_id, started_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		run.ID, run.WorkflowName, run.WorkflowVersion, string(run.Status),
		run.CurrentState, bb, run.TemporalID, run.StartedAt, run.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) GetRun(ctx context.Context, id string) (*asmtypes.WorkflowRun, error) {
	var r asmtypes.WorkflowRun
	var status string
	var bb []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, workflow_name, workflow_version, status, current_state,
		       blackboard, coalesce(temporal_workflow_id,''), started_at, updated_at, completed_at
		FROM workflow_runs WHERE id=$1`, id,
	).Scan(&r.ID, &r.WorkflowName, &r.WorkflowVersion, &status, &r.CurrentState,
		&bb, &r.TemporalID, &r.StartedAt, &r.UpdatedAt, &r.CompletedAt)
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
	_, err := s.pool.Exec(ctx, `
		UPDATE workflow_runs
		SET status=$2, current_state=$3, blackboard=$4, updated_at=$5, completed_at=$6
		WHERE id=$1`,
		run.ID, string(run.Status), run.CurrentState, bb, run.UpdatedAt, run.CompletedAt,
	)
	return err
}

func (s *PostgresStore) ListRuns(ctx context.Context, filter RunFilter) ([]*asmtypes.WorkflowRun, error) {
	query := `SELECT id, workflow_name, workflow_version, status, current_state,
	                 blackboard, coalesce(temporal_workflow_id,''), started_at, updated_at, completed_at
	          FROM workflow_runs WHERE TRUE`
	args := []interface{}{}
	i := 1
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
		if err := rows.Scan(&r.ID, &r.WorkflowName, &r.WorkflowVersion, &status, &r.CurrentState,
			&bb, &r.TemporalID, &r.StartedAt, &r.UpdatedAt, &r.CompletedAt); err != nil {
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

// -- HITLStore --

func (s *PostgresStore) CreateHITL(ctx context.Context, req *asmtypes.HITLRequest) error {
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	meta, _ := json.Marshal(req.Metadata)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO hitl_requests (id, run_id, state_name, assignee, timeout_at, metadata, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		req.ID, req.RunID, req.StateName, req.Assignee, req.TimeoutAt, meta, req.CreatedAt,
	)
	return err
}

func (s *PostgresStore) GetHITL(ctx context.Context, runID string) (*asmtypes.HITLRequest, error) {
	var req asmtypes.HITLRequest
	var meta []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, run_id, state_name, coalesce(assignee,''), timeout_at, resolved,
		       resolved_at, coalesce(resolution,''), coalesce(resolver,''), metadata, created_at
		FROM hitl_requests WHERE run_id=$1 AND resolved=FALSE ORDER BY created_at DESC LIMIT 1`, runID,
	).Scan(&req.ID, &req.RunID, &req.StateName, &req.Assignee, &req.TimeoutAt, &req.Resolved,
		&req.ResolvedAt, &req.Resolution, &req.Resolver, &meta, &req.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("no HITL request for run '%s': %w", runID, err)
	}
	_ = json.Unmarshal(meta, &req.Metadata)
	return &req, nil
}

func (s *PostgresStore) ResolveHITL(ctx context.Context, runID, resolution, resolver string) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx, `
		UPDATE hitl_requests
		SET resolved=TRUE, resolved_at=$3, resolution=$4, resolver=$5
		WHERE run_id=$1 AND resolved=FALSE`,
		runID, now, resolution, resolver,
	)
	return err
}

func (s *PostgresStore) ListPendingHITL(ctx context.Context) ([]*asmtypes.HITLRequest, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, run_id, state_name, coalesce(assignee,''), timeout_at, created_at
		FROM hitl_requests WHERE resolved=FALSE ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*asmtypes.HITLRequest
	for rows.Next() {
		var r asmtypes.HITLRequest
		if err := rows.Scan(&r.ID, &r.RunID, &r.StateName, &r.Assignee, &r.TimeoutAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
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
		`INSERT INTO users (id, username, password_hash, role, created_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		u.ID, u.Username, u.PasswordHash, u.Role, u.CreatedAt,
	)
	return err
}

func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, role, created_at FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("user '%s' not found: %w", id, err)
	}
	return &u, nil
}

func (s *PostgresStore) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, role, created_at FROM users WHERE username=$1`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
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

func (s *PostgresStore) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, username, role, created_at FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
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
