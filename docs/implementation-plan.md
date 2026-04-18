# ASM Platform — Implementation Plan

> **Living document.** History through M12 is archived in `implementation-plan-archive.md`.
> Last updated: 2026-04-18

---

## Status Legend

| Symbol | Meaning |
|---|---|
| ✅ | Complete |
| 🔄 | In progress |
| ⬜ | Not started |
| 🚫 | Blocked |
| 💬 | Needs decision |

---

## Completed Milestones (summary)

| Milestone | What was delivered |
|---|---|
| **M1** Foundation | Go scaffold, store interfaces (Memory + Postgres), engine, REST API, WebSocket hub |
| **M2** Durability | Temporal executor — full durable runs, worker binary, replay-safe state machine |
| **M3** Agent capability | Tool-calling loop, MCP client (JSON-RPC 2.0), per-agent MCP config, Ollama provider |
| **M4** Multi-queue | Specialist workers, per-agent task queues, `HandlerFunc` SDK |
| **M5** Security | JWT auth, RBAC (admin / operator / viewer), refresh token rotation, WebSocket auth |
| **M5.2** Multi-tenancy | Tenant model, context propagation, super_admin, full tenant API |
| **M6** Designer UX | Auto-layout (dagre), drag-to-connect, split view, run timeline, agent config panel |
| **M7** Production readiness | Structured logging (slog), Prometheus metrics, golang-migrate, health probes, K8s manifests |
| **M8** Intelligence | Per-state instructions, MCP server registry, AI workflow generator, script state type |
| **M9** Code generation | LLM-assisted specialist worker scaffolding, template project |
| **M10** Generated MCP servers | MCP server scaffolding, MCP registry promoted to DB |
| **M11** LLM config UI | DB-backed providers, Settings page, Grok/Deepseek/Gemini, worker hot-reload, HITL fixes |
| **M12** Orchestration 2.0 | Mandatory Temporal, `wait` state type, parallel `to_nodes` transitions |
| **Post-M12** | Code state (JavaScript / goja), LLM code-gen for code nodes, MCP server picker (chip UI), Docker frontend (Nginx), GitHub repo |

---

## Current State: Skill System — Phase 1 ✅

The foundational Skill layer has been implemented. A **skill** is a higher-level abstraction over a workflow — a named, typed contract (inputs / outputs / capability declarations) backed by a workflow implementation.

### What is in production today

| Component | Location | Status |
|---|---|---|
| `SkillDef`, `SkillPortDef`, `CapabilityDecl`, `SkillCallDef` types | `pkg/asmtypes/skill.go` | ✅ |
| `skill_call` state type + `SkillCall *SkillCallDef` on `StateDef` | `pkg/asmtypes/workflow.go` | ✅ |
| Skill Analyser — description → optimised workflow (LLM) | `internal/designer/skill_analyser.go` | ✅ |
| Skill Renderer — workflow YAML → description + inferred I/O (LLM) | `internal/designer/skill_renderer.go` | ✅ |
| `POST /api/v1/designer/skill/analyse` | `internal/api/designer_handlers.go` | ✅ |
| `POST /api/v1/designer/skill/render` | `internal/api/designer_handlers.go` | ✅ |
| `LoadWorkflowDef` + `GetRunBlackboard` Temporal activities | `internal/temporal/activities.go` | ✅ |
| `skill_call` branch in `ASMWorkflow` — child workflow via `ExecuteChildWorkflow` | `internal/temporal/workflow.go` | ✅ |
| `SkillPanel.vue` — CodeMirror description editor + fullscreen pop-out | `frontend/src/components/designer/SkillPanel.vue` | ✅ |
| `SkillInputMappingEditor.vue` — port ↔ blackboard field mapping rows | `frontend/src/components/designer/SkillInputMappingEditor.vue` | ✅ |
| `skill_call` node — violet style, star icon, config form in StatePanel | `frontend/src/components/designer/StatePanel.vue` | ✅ |
| Skill tab in Designer (replaces AI tab) | `frontend/src/views/Designer.vue` | ✅ |
| `CodeEditor` `language="text"` mode — word-wrap, no JS syntax | `frontend/src/components/designer/CodeEditor.vue` | ✅ |
| `CodeEditorModal` — language + title props, violet theme for text | `frontend/src/components/designer/CodeEditorModal.vue` | ✅ |

### How it works today

```
User types description → SkillPanel → POST /skill/analyse → LLM → YAML
                                                                    ↓
                                                        Canvas (workflow)
                                                                    ↓
                                                POST /skill/render ← User
                                                        ↓
                                                description + inferred I/O

skill_call state → resolves child workflow by name → ExecuteChildWorkflow
                 → maps input_mappings: bb → child bb
                 → waits for child terminal state
                 → maps output_mappings: child bb → parent bb
                 → fires completion_trigger / failure_trigger
```

---

## Skill System — Phase 2: Skill Registry ✅

> Skills are now first-class, discoverable entities stored in the database. The Skill Catalog lets operators browse, compose, and reuse skills across workflows.

### S2.1 — DB-backed Skill store ✅

**Why:** Today, a skill only exists implicitly — the `skill_ref` in a `skill_call` node is just a workflow name string. There is no registry of available skills, no type-checked I/O contracts, and no way to browse what skills exist. A DB-backed registry fixes this and enables composition.

**Tasks:**

- [x] `migrations/000008_skills.up.sql` — `skills` table: `(tenant_id, name)` UNIQUE; `inputs`, `outputs`, `capabilities` as JSONB
- [x] `SkillStore` interface — `CreateSkill`, `GetSkill`, `ListSkills`, `UpdateSkill`, `DeleteSkill`; `MemoryStore` + `PostgresStore` implementations
- [x] `SkillStore` embedded in `Store` aggregate interface
- [x] `internal/api/skill_handlers.go` — `ListSkills`, `GetSkill`, `CreateSkill`, `UpdateSkill`, `DeleteSkill`
- [x] Routes: `GET/POST /api/v1/skills`, `GET/PUT/DELETE /api/v1/skills/:name`

### S2.2 — Skill Catalog UI ✅

- [x] `frontend/src/views/Skills.vue` — table with search, name + capability chips, input/output badges, workflow link, row-hover edit/delete
- [x] Sidebar star icon NavItem between Designer and MCP Servers; `/skills` route
- [x] `SkillModal.vue` — create/edit with name, description, workflow_ref + "Open ↗" link, I/O port editor rows (name/type/description/required), MCP server capability picker

### S2.3 — skill_call picker ✅

- [x] `SkillPicker.vue` — searchable dropdown fetching `GET /api/v1/skills`; shows name, description, input/output counts; selected state shows port summary (names + types) + workflow_ref; close on outside click; "Manage skill registry ↗" footer link
- [x] `StatePanel.vue` — replaced free-text `skill_ref` input with `SkillPicker`; `onSkillSelected()` auto-populates input/output mapping rows from the skill's declared ports (only adds rows not already present, preserving manual edits)

---

## Skill System — Phase 3: Subskills & Composition

> **Goal:** Skills can be composed from smaller, reusable subskills. The analyser is taught about existing skills in the registry so it can generate `skill_call` nodes instead of reimplementing logic inline.

### S3.1 — Registry-aware Analyser ✅

**Why:** Today the analyser only knows the YAML schema. If a skill for "fetch GLEIF data" already exists in the registry, a new skill that needs GLEIF data should reference it via `skill_call` rather than re-implementing the lookup.

**Tasks:**

- [x] `Generator` struct gains a `skillStore store.SkillStore` field (injected at startup via `NewGenerator`)
- [x] `AnalyseSkill` calls `ListSkills(ctx)` at request time and passes the result into the prompt builder
- [x] `buildSkillAnalyserPrompt()` injects a `## Registered Skills` section listing each skill's name, description, and typed I/O ports when the registry is non-empty
- [x] Decomposition rules updated: "use **skill_call** FIRST when a registered skill already implements the capability"
- [x] `skill_call` state type added to `buildSystemPrompt` schema example + Rule 21 documenting required fields
- [x] `cmd/server/main.go` passes the store to `NewGenerator`

### S3.2 — Skill dependency graph ✅

**Why:** If skill A calls skill B via `skill_call`, and skill B is updated, operators need to know which callers are affected.

**Tasks:**

- [x] `migrations/000009_skill_dependencies.up.sql` — `skill_dependencies` join table: `(skill_id, depends_on_id)` PK; index on `depends_on_id`
- [x] `SkillStore` interface gains `UpsertSkillDependencies` and `ListSkillDependents`; `MemoryStore` + `PostgresStore` both implemented
- [x] `Skill.DependentsCount int` computed field — populated by `GetSkill` and `ListSkills` via LEFT JOIN aggregate (Postgres) / O(n) scan (Memory)
- [x] `syncSkillDependencies` helper in skill handlers — scans the skill's workflow for `skill_call` states, resolves each `skill_ref` to an ID, calls `UpsertSkillDependencies`; runs after every Create and Update; errors are non-fatal (logged, not returned)
- [x] `GET /api/v1/skills/:name/dependents` — returns skills that call this skill
- [x] `DELETE /api/v1/skills/:name` returns 409 with `dependents` list when callers exist; bypassed with `?force=true`
- [x] Skill Catalog table: new "Used by" column with violet badge showing dependent count
- [x] Delete confirm dialog includes caller count warning when `dependents_count > 0`
- [x] Edit modal shows amber warning banner when skill has dependents

---

## Skill System — Phase 4: A2A Communication ✅

> **Goal:** Skills can communicate with each other at runtime, not just via parent-child Temporal hierarchy. This enables peer-to-peer skill invocation and event-driven skill activation.

### S4.1 — Skill-to-skill events (emit_event + wait-on-event) ✅

**Why:** The current `skill_call` model is synchronous — the parent waits for the child. Some use cases need fire-and-forget or event-driven patterns (e.g., one skill emits a `data_ready` event; another skill wakes up when it sees it).

**Design — tenant-scoped named events, delivered via Temporal signals:**

```
emit_event state
  └─ EmitWorkflowEvent activity
       ├─ publishes events.WorkflowEvent to the bus (WebSocket clients see it)
       └─ queries event_subscriptions (tenant_id, event_name)
            └─ for each match: client.SignalWorkflow → triggerCh → wait state wakes
```

Wait state with `on_event`:
```
RegisterEventSubscription activity (idempotent ID = temporalID + "__" + stateName)
  └─ blocks on triggerCh (same channel used by manual triggers)
       └─ on signal: UnregisterEventSubscription activity → applyTransition
```

**Tasks:**

- [x] `StateEmitEvent` + `EmitEventDef` type in `pkg/asmtypes/workflow.go`
- [x] `OnEvent` / `OnEventMatch` fields on `StateDef` for wait-on-event states
- [x] `migrations/000010_event_subscriptions.up.sql` — `event_subscriptions` table with index on `(tenant_id, event_name)`
- [x] `EventSubscription` struct + `EventSubscriptionStore` interface + Memory + Postgres implementations
- [x] `events.WorkflowEvent = "workflow.event"` constant
- [x] `RegisterEventSubscription`, `UnregisterEventSubscription`, `EmitWorkflowEvent` Temporal activities
- [x] `Activities.Client client.Client` — injected at worker startup so `EmitWorkflowEvent` can signal workflows
- [x] `WorkflowParams.TenantID string` — propagated from `run.TenantID` at workflow start
- [x] `StateEmitEvent` branch in `execBranch` — builds payload (named fields or full blackboard), calls activity, advances on `completion_trigger` (default `event_emitted`)
- [x] `StateWait` block extended — registers subscription before blocking when `state.OnEvent != ""`, unregisters after selector fires (on event, timeout, or condition)

**YAML example:**
```yaml
states:
  - name: notify_downstream
    type: emit_event
    emit_event:
      event_name: data_ready
      payload_fields: [result, run_id]
      completion_trigger: notified

  - name: wait_for_data
    type: wait
    on_event: data_ready
    on_event_match: event_received
    timeout: 1h
    on_timeout: timed_out
```

### S4.2 — External skill invocation API + API key auth ✅

**Why:** A skill should be callable from outside the platform (e.g., a human-facing app, a webhook, another ASM tenant) without issuing a full user JWT.

**API key format:** `asm_sk_` + base64url(32 random bytes). Only the SHA-256 hex hash is stored. The raw key is shown once at creation time.

**Auth middleware — `RequireAuthOrAPIKey`:**
- Checks `X-API-Key` header first, then `Authorization: Bearer asm_sk_...`
- Falls back to standard JWT validation if no API key is present
- Scopes the request context to the key's tenant via `store.WithTenantID`
- Calls `TouchAPIKey` asynchronously (non-blocking `last_used_at` update)

**Tasks:**

- [x] `migrations/000011_api_keys.up.sql` — `api_keys` table (`key_hash` UNIQUE, `revoked_at` nullable)
- [x] `APIKey` struct + `APIKeyStore` interface + Memory + Postgres implementations
- [x] `internal/api/apikey_middleware.go` — `RequireAuthOrAPIKey` middleware + `hashAPIKey` helper
- [x] `internal/api/api_key_handlers.go` — `CreateAPIKey` (generates + returns raw key once), `ListAPIKeys`, `RevokeAPIKey`
- [x] `GET /api/v1/api-keys`, `POST /api/v1/api-keys`, `DELETE /api/v1/api-keys/:id` (admin only)
- [x] `internal/api/skill_invoke_handlers.go` — `InvokeSkill` (POST, starts run, returns `run_id`), `GetSkillInvocation` (GET, returns run)
- [x] `POST /api/v1/invoke/:name`, `GET /api/v1/invoke/:name/runs/:run_id` — protected by `RequireAuthOrAPIKey`

---

## Planned: UX Polish & Developer Experience

### UX.1 — Workflow YAML schema validation feedback ⬜

The designer currently shows a red error banner when YAML is invalid, but the message is the raw parse error. Provide structured, field-level validation feedback based on the known schema so users can fix errors without reading Go structs.

### UX.2 — skill_call node: live port mapping hints ⬜

When a `skill_call` node has a `skill_ref` set, fetch the skill's declared I/O from the registry and show type hints next to each mapping row (e.g., `company_name: string`).

### UX.3 — Run replay / debug mode ⬜

Show the blackboard diff at each state transition in the Monitor timeline — what fields were added/changed/removed when a state completed.

### UX.4 — Code node: runtime error display ⬜

When a `code` state fails (goja throws), surface the JS stack trace in the Monitor timeline rather than just an `execution_failed` event.

### UX.5 — Designer keyboard shortcuts ⬜

- `Cmd+S` → Save
- `Cmd+Z` / `Cmd+Shift+Z` → Undo / Redo canvas actions
- `Del` / `Backspace` → Delete selected node or edge

---

## Technical Debt

| Issue | Severity | Notes |
|---|---|---|
| HITL `assignee` filtering | Low | Users should only see/resolve tasks assigned to them; needs `assignee` field in HITL events |
| Tenant deletion safety | Low | Delete not blocked if tenant has active runs or users |
| `skill_call` timeout | Low | No timeout field on `skill_call` state — child could run indefinitely; add `timeout` + `failure_trigger` handling in the Temporal branch |
| `skill_ref` resolved by name only | Medium | No version pinning — `skill_ref: "company-lookup"` always uses the latest workflow version; add `skill_ref: "company-lookup@v3"` syntax |
| Worker binary requires Postgres | Low | Intentional for Temporal mode; documented |

---

## Architecture Decisions — Skill System

| Decision | Rationale |
|---|---|
| Workflow is the single source of truth for skill logic | The skill description is a "projection" rendered from the workflow; editing the description triggers re-analysis (proposed changes), not auto-apply. Avoids two-source-of-truth conflicts. |
| `SkillDef` and `WorkflowDef` are separate types | Workflow is the implementation; skill is the contract/abstraction. Keeps `workflow.go` focused on execution semantics. |
| Child workflow output read via `GetRunBlackboard` activity | Instead of changing `ASMWorkflow`'s return signature (which would break Temporal replay determinism), the child's terminal blackboard is read from the store after `ExecuteChildWorkflow` resolves. Clean side-channel. |
| Analyser instructs LLM to prefer code/script nodes | The key quality lever: "minimise LLM usage — prefer code or script nodes wherever the logic is deterministic". Reduces latency and cost of generated workflows. |
| Skill description uses CodeMirror text mode | Consistent editor UX with the code node (same pop-out, same dark theme, same keyboard shortcuts). `language="text"` disables JS syntax, enables word-wrap. |
| AI tab replaced by Skill tab | The Skill panel is a superset — it covers both the AI generation use case (describe → generate) and the reverse (render description from existing workflow). No need for a separate AI chat panel. |

---

## Running the Platform

```bash
# Infrastructure (Docker)
docker compose up -d

# API server + workers (dev)
go run ./cmd/server
go run ./cmd/worker

# Frontend (dev)
cd frontend && npm run dev   # → http://localhost:5173

# Full stack via Docker
docker compose up -d   # → http://localhost
```

See `docs/deploy-macos.md` for a full macOS setup walkthrough.
