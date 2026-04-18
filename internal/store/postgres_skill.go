package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

func (s *PostgresStore) CreateSkill(ctx context.Context, sk *Skill) error {
	if sk.ID == "" {
		sk.ID = uuid.NewString()
	}
	now := time.Now()
	if sk.CreatedAt.IsZero() {
		sk.CreatedAt = now
	}
	sk.UpdatedAt = now

	inputs, err := json.Marshal(nullSlice(skillPortsToAny(sk.Inputs)))
	if err != nil {
		return fmt.Errorf("marshal skill inputs: %w", err)
	}
	outputs, err := json.Marshal(nullSlice(skillPortsToAny(sk.Outputs)))
	if err != nil {
		return fmt.Errorf("marshal skill outputs: %w", err)
	}
	caps, err := json.Marshal(nullSlice(capsToAny(sk.Capabilities)))
	if err != nil {
		return fmt.Errorf("marshal skill capabilities: %w", err)
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO skills (id, tenant_id, name, description, workflow_ref, inputs, outputs, capabilities, created_at, updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		sk.ID, sk.TenantID, sk.Name, sk.Description, sk.WorkflowRef,
		inputs, outputs, caps, sk.CreatedAt, sk.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create skill: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetSkill(ctx context.Context, name string) (*Skill, error) {
	tenantID := TenantIDFromContext(ctx)
	row := s.pool.QueryRow(ctx,
		`SELECT s.id, s.tenant_id, s.name, s.description, s.workflow_ref,
		        s.inputs, s.outputs, s.capabilities, s.created_at, s.updated_at,
		        COALESCE(d.cnt, 0)
		 FROM skills s
		 LEFT JOIN (
		     SELECT depends_on_id, COUNT(*) AS cnt
		     FROM skill_dependencies
		     GROUP BY depends_on_id
		 ) d ON d.depends_on_id = s.id
		 WHERE s.tenant_id=$1 AND s.name=$2`,
		tenantID, name,
	)
	return scanSkillWithCount(row)
}

func (s *PostgresStore) ListSkills(ctx context.Context) ([]*Skill, error) {
	tenantID := TenantIDFromContext(ctx)
	rows, err := s.pool.Query(ctx,
		`SELECT s.id, s.tenant_id, s.name, s.description, s.workflow_ref,
		        s.inputs, s.outputs, s.capabilities, s.created_at, s.updated_at,
		        COALESCE(d.cnt, 0)
		 FROM skills s
		 LEFT JOIN (
		     SELECT depends_on_id, COUNT(*) AS cnt
		     FROM skill_dependencies
		     GROUP BY depends_on_id
		 ) d ON d.depends_on_id = s.id
		 WHERE s.tenant_id=$1
		 ORDER BY s.name`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()
	var out []*Skill
	for rows.Next() {
		sk, err := scanSkillWithCount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sk)
	}
	return out, rows.Err()
}

func (s *PostgresStore) UpdateSkill(ctx context.Context, sk *Skill) error {
	tenantID := TenantIDFromContext(ctx)
	inputs, err := json.Marshal(nullSlice(skillPortsToAny(sk.Inputs)))
	if err != nil {
		return fmt.Errorf("marshal skill inputs: %w", err)
	}
	outputs, err := json.Marshal(nullSlice(skillPortsToAny(sk.Outputs)))
	if err != nil {
		return fmt.Errorf("marshal skill outputs: %w", err)
	}
	caps, err := json.Marshal(nullSlice(capsToAny(sk.Capabilities)))
	if err != nil {
		return fmt.Errorf("marshal skill capabilities: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`UPDATE skills SET description=$1, workflow_ref=$2, inputs=$3, outputs=$4, capabilities=$5, updated_at=$6
         WHERE tenant_id=$7 AND name=$8`,
		sk.Description, sk.WorkflowRef, inputs, outputs, caps, time.Now(), tenantID, sk.Name,
	)
	if err != nil {
		return fmt.Errorf("update skill: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteSkill(ctx context.Context, name string) error {
	tenantID := TenantIDFromContext(ctx)
	_, err := s.pool.Exec(ctx,
		`DELETE FROM skills WHERE tenant_id=$1 AND name=$2`, tenantID, name,
	)
	if err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	return nil
}

func (s *PostgresStore) UpsertSkillDependencies(ctx context.Context, skillID string, dependsOnIDs []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("upsert skill deps begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM skill_dependencies WHERE skill_id = $1`, skillID,
	); err != nil {
		return fmt.Errorf("clear skill deps: %w", err)
	}

	for _, depID := range dependsOnIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO skill_dependencies (skill_id, depends_on_id) VALUES ($1, $2)
             ON CONFLICT DO NOTHING`,
			skillID, depID,
		); err != nil {
			return fmt.Errorf("insert skill dep: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (s *PostgresStore) ListSkillDependents(ctx context.Context, name string) ([]*Skill, error) {
	tenantID := TenantIDFromContext(ctx)
	rows, err := s.pool.Query(ctx,
		`SELECT s.id, s.tenant_id, s.name, s.description, s.workflow_ref,
		        s.inputs, s.outputs, s.capabilities, s.created_at, s.updated_at,
		        COALESCE(d.cnt, 0)
		 FROM skills s
		 JOIN skill_dependencies sd ON sd.skill_id = s.id
		 JOIN skills dep ON dep.id = sd.depends_on_id
		 LEFT JOIN (
		     SELECT depends_on_id, COUNT(*) AS cnt
		     FROM skill_dependencies
		     GROUP BY depends_on_id
		 ) d ON d.depends_on_id = s.id
		 WHERE dep.tenant_id = $1 AND dep.name = $2
		   AND s.tenant_id = $1
		 ORDER BY s.name`,
		tenantID, name,
	)
	if err != nil {
		return nil, fmt.Errorf("list skill dependents: %w", err)
	}
	defer rows.Close()
	var out []*Skill
	for rows.Next() {
		sk, err := scanSkillWithCount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sk)
	}
	return out, rows.Err()
}

// ── helpers ──────────────────────────────────────────────────────────────────

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanSkillWithCount scans a skill row that includes a trailing dependents_count column.
func scanSkillWithCount(row scanner) (*Skill, error) {
	var (
		sk         Skill
		inputsRaw  []byte
		outputsRaw []byte
		capsRaw    []byte
	)
	if err := row.Scan(
		&sk.ID, &sk.TenantID, &sk.Name, &sk.Description, &sk.WorkflowRef,
		&inputsRaw, &outputsRaw, &capsRaw,
		&sk.CreatedAt, &sk.UpdatedAt,
		&sk.DependentsCount,
	); err != nil {
		return nil, fmt.Errorf("scan skill: %w", err)
	}
	if err := json.Unmarshal(inputsRaw, &sk.Inputs); err != nil {
		sk.Inputs = nil
	}
	if err := json.Unmarshal(outputsRaw, &sk.Outputs); err != nil {
		sk.Outputs = nil
	}
	if err := json.Unmarshal(capsRaw, &sk.Capabilities); err != nil {
		sk.Capabilities = nil
	}
	return &sk, nil
}

// nullSlice returns an empty (non-nil) slice if v is nil, so JSON encodes as []
// rather than null.
func nullSlice[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}

func skillPortsToAny(ports []asmtypes.SkillPortDef) []asmtypes.SkillPortDef { return ports }
func capsToAny(caps []asmtypes.CapabilityDecl) []asmtypes.CapabilityDecl    { return caps }
