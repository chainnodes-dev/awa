-- Skill dependency graph
-- Records which skills invoke other skills via skill_call nodes.
-- Populated automatically whenever a skill is created or updated.

CREATE TABLE IF NOT EXISTS skill_dependencies (
    skill_id       UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    depends_on_id  UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (skill_id, depends_on_id)
);

-- Used by ListSkillDependents (find callers of a given skill).
CREATE INDEX IF NOT EXISTS idx_skill_deps_depends_on ON skill_dependencies(depends_on_id);
