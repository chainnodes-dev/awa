# ASM Platform — Implementation Plan

> **Living document.** Update status, notes, and decisions as the project evolves.
> Last updated: 2026-03-30 (M8 in progress — AI designer, per-state prompts, script states)

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

## Milestones

| # | Milestone | Status | Target |
|---|---|---|---|
| M1 | Foundation — server, store, engine, basic API | ✅ | Done |
| M2 | Durability — Temporal executor, worker binary | ✅ | Done |
| M3 | Agent capability — tool-calling loop, MCP registry | ✅ | Done |
| M4 | Multi-queue — specialist workers, per-agent task queues | ✅ | Done |
| M5 | Security — auth layer, RBAC | ✅ | Done |
| M6 | Designer UX — auto-layout, inline config, drag-connect | ✅ | Done |
| M7 | Production readiness — observability, migration tooling | ⬜ | — |
| M8 | Intelligence — AI workflow generator, per-state prompts, script states | 🔄 | — |
| M9 | Code generation — LLM-assisted specialist worker scaffolding | ⬜ | — |
| M10 | Generated MCP servers — sharable deterministic logic as services | ⬜ | — |

---

## M1 — Foundation ✅

### Completed
- [x] Project scaffold — Go module, directory structure
- [x] `pkg/asmtypes` — WorkflowDef, WorkflowRun, Blackboard schema, runtime types
- [x] YAML workflow loader with env var expansion and validation
- [x] MemoryStore — full Store interface implementation for development
- [x] PostgresStore — full Store interface backed by pgx/v5
- [x] LocalBus — in-process event bus
- [x] RedisBus — Redis Pub/Sub event bus for multi-process deployments
- [x] Blackboard — thread-safe, schema-validated shared context
- [x] Engine (direct mode) — driveRun goroutine, agent dispatch, HITL handling, guard evaluation
- [x] DirectExecutor — LLM call, prompt building, streaming token events, structured output parsing
- [x] LLM providers — Anthropic (Claude), OpenAI
- [x] REST API — workflows CRUD, runs lifecycle, HITL signal
- [x] WebSocket hub — real-time event fan-out to browser clients
- [x] Config — environment-based configuration
- [x] Frontend (React/Vite) — workflow designer canvas, monitor view
- [x] Test suite — store, events, blackboard, engine, API handlers (59 tests, all passing)
- [x] Bug fix — nil context panic in RedisBus.Subscribe

---

## M2 — Durability ✅

### Completed
- [x] Temporal Go SDK added (`go.temporal.io/sdk v1.41.1`)
- [x] `internal/temporal/params.go` — shared signal/param types
- [x] `internal/temporal/activities.go` — UpdateRun, RecordTransition, PublishEvent, CreateHITL, ExecuteAgent
- [x] `internal/temporal/workflow.go` — ASMWorkflow: full state machine loop as durable Temporal workflow
- [x] `internal/temporal/client.go` — EngineClient wrapping Temporal SDK
- [x] `internal/temporal/worker.go` — worker registration helper
- [x] `cmd/worker/main.go` — worker binary entry point
- [x] Engine — TemporalEngineClient interface, UseTemporalClient(), branched StartRun/Trigger/SignalHITL
- [x] Engine — exported EvalGuard for use in temporal workflow package
- [x] Engine — wakeupCh fix for external-trigger state race condition
- [x] Server — Temporal client wired when EXECUTOR_MODE=temporal

### Architecture decisions made
- Temporal workflow = full ASM run (not per-activity) — runs survive server restarts between states
- WorkflowDef passed as workflow parameter (not loaded inside workflow) — avoids non-determinism on replay
- TemporalEngineClient interface defined in orchestrator package — avoids import cycle with temporal package
- One task queue by default (`asm-workers`) — multi-queue support planned in M4

---

## M3 — Agent Capability ✅

### Goal
Make LLM agents genuinely useful: they can call tools during reasoning, not just produce text. This is the bridge between LLM-only and the hybrid model.

### Completed
- [x] **Tool-calling loop in `DirectExecutor`** — multi-turn conversation: LLM → `tool_use` → execute → feed result back → repeat (max 10 iterations via `maxToolIterations` constant)
- [x] **`agent.tool_call` events** — emitted to bus on every tool invocation (input + output both published)
- [x] **Structured output extraction** — `parseAgentOutput` extracts first valid JSON object from final text response; non-JSON content falls back gracefully to `AgentOutput.Error`
- [x] **`tools.Tool` interface** — `Name()`, `Description()`, `InputSchema()`, `Execute()` in `internal/tools/registry.go`
- [x] **`tools.Registry`** — `Register`, `Get`, `Has`, `All`, `Merge`; agent tools override global on name collision
- [x] **`DirectExecutor.RegisterTool`** — global tools available to every agent; merged with per-agent tools at execution time
- [x] **MCP client** (`internal/tools/mcp/client.go`) — custom JSON-RPC 2.0 over HTTP POST; `ListTools` → `[]tools.Tool`; `CallTool` → `json.RawMessage`
- [x] **`mcp.LoadRegistry`** (`internal/tools/mcp/tool.go`) — connects to MCP server, discovers tools, returns pre-populated `*tools.Registry`
- [x] **Per-agent MCP server config** — `agent.Config["mcp_servers"]` (comma-separated, env-expanded URLs); referenced in workflow YAML as `{{ env.MY_MCP_URL }}`
- [x] **LLM provider constructors** — `NewAnthropicProvider(apiKey, defaultModel)` and `NewOpenAIProvider(apiKey, defaultModel)`; no hardcoded model names
- [x] **`.env.example`** — MCP server URL documentation pattern added
- [x] **21 executor tests** — tool-calling loop, error handling, max iterations, `parseAgentOutput`, `buildAssistantBlocks`, registry merge; all passing
- [x] **Full test suite** — 80 tests, 5 packages, all green; clean build

### Architecture decisions made
- MCP server URLs must come from environment variables — never hardcoded; workflow YAML uses `{{ env.VAR }}` template expansion
- Custom JSON-RPC 2.0 HTTP client rather than an external MCP SDK — keeps the dependency tree minimal and the protocol surface explicit
- Agent tools win over global tools on name collision in `Registry.Merge` — agent-specific behaviour can always shadow a platform default
- Tool errors are serialised and fed back to the LLM rather than aborting — the model can reason about failures and choose a fallback path
- `maxToolIterations = 10` hard-coded constant — prevents runaway loops; configurable per-agent planned for M4+

### Deferred to later milestones
- MCP SSE transport (currently HTTP POST only) — most servers support POST; SSE support added if needed
- Tool call deduplication inside Temporal activity retries — tools should be idempotent; explicit tracking deferred to M4
- Filesystem / PostgreSQL / HTTP MCP server integrations — operator-deployed; platform only needs the client

---

## M4 — Multi-Queue / Specialist Workers ✅

### Goal
Allow different states to be processed by different worker pools, enabling specialist workers alongside LLM agents.

### Completed
- [x] **`AgentDef.TaskQueue` field** — added to `pkg/asmtypes/workflow.go`; YAML key `task_queue`, defaults to empty (= global queue)
- [x] **Per-agent queue routing in `ASMWorkflow`** — `agentOptsFor()` closure builds `workflow.ActivityOptions` with the correct `TaskQueue`; LLM worker's default `agentOpts` used when `TaskQueue` is empty; fully backward compatible
- [x] **`internal/specialist` SDK** — `specialist.New(taskQueue)`, `Register(agentName, HandlerFunc)`, `Build(client.Client) worker.Worker`
- [x] **`HandlerFunc` type** — `func(ctx context.Context, bb map[string]interface{}) (*asmtypes.AgentOutput, error)` — identical output contract to LLM agents
- [x] **`ExecuteAgent` activity name alignment** — specialist worker uses `activity.RegisterOptions{Name: "ExecuteAgent"}` so Temporal routes correctly by (task queue, activity name)
- [x] **`cmd/specialist-worker/main.go`** — template binary; copy-paste to start a new specialist domain; no Postgres / Redis / LLM keys required
- [x] **Invoice validation example** — `validateInvoice` handler: required field check, positive-amount guard, ISO 4217 currency check; triggers `validation_passed` / `validation_failed`
- [x] **`SPECIALIST_TASK_QUEUE` env var** — documented in `.env.example`
- [x] **20 specialist package tests** — dispatch routing, error propagation, blackboard forwarding, handler overwrite, metadata; all invoice validation cases (missing fields, negative/zero amount, unknown currency); all passing
- [x] **Full test suite** — 100 tests, 6 packages, all green; clean build

### Architecture decisions made
- Workers registered explicitly in code (not a registry table) — simplest, most discoverable; no additional DB schema needed
- Activity name pinned to `"ExecuteAgent"` via `RegisterOptions` — both LLM and specialist workers share the same activity name; task queue is the only routing discriminant
- `HandlerFunc` receives the full blackboard map — handlers may read context without coupling to the engine; they must not modify the map in place (output goes via `BlackboardUpdates`)
- Specialist binary needs no store/bus/LLM deps — only Temporal address and namespace; operationally independent

### Workflow YAML pattern

```yaml
agents:
  - name: invoice-validator
    task_queue: validate-workers   # omit → routes to LLM worker on asm-workers
    # model not needed — handler is Go code

  - name: enrichment-agent
    model: "{{ env.DEFAULT_MODEL }}"  # no task_queue → LLM worker handles this
    config:
      mcp_servers: "{{ env.MCP_ENRICH_URL }}"
```

### Running a specialist worker

```bash
SPECIALIST_TASK_QUEUE=validate-workers go run ./cmd/specialist-worker
```

---

## M5 — Security ✅

### Goal
The platform should not be exposed beyond localhost without authentication.

### Completed

#### 5.1 JWT authentication
- [x] **`internal/auth/jwt.go`** — `JWTService` (HS256, 15min access token TTL); `Claims` struct (userID, username, role); `GenerateAccessToken`, `ValidateToken`
- [x] **`internal/auth/password.go`** — `HashPassword` (bcrypt cost 12), `CheckPassword`; `GenerateRefreshToken` (32 random bytes → 64 hex chars); `HashToken` (SHA-256 hex for DB storage)
- [x] **`internal/auth/middleware.go`** — `RequireAuth` gin middleware (reads `Authorization: Bearer`); `RequireRole(roles...)` middleware; `ClaimsFrom` context helper
- [x] **`POST /api/v1/auth/login`** — credential check → issue access token + refresh token → store hash in DB
- [x] **`POST /api/v1/auth/refresh`** — validate stored hash → revoke old → issue new access + refresh tokens (rotation)
- [x] **`POST /api/v1/auth/logout`** — revoke all refresh tokens for the caller
- [x] **WebSocket auth** — `?token=<jwt>` query param validated before upgrade; 1008 close code on rejection

#### 5.2 User management
- [x] **`migrations/002_users.sql`** — `users` table (UUID PK, username unique, password_hash, role, created_at); `refresh_tokens` table (user FK, token_hash unique, expires_at, revoked)
- [x] **`internal/store/user.go`** — `User` and `RefreshToken` types; `UserStore` and `RefreshTokenStore` interfaces
- [x] **`MemoryStore`** — full `UserStore` + `RefreshTokenStore` implementation (development / test)
- [x] **`PostgresStore`** — full `UserStore` + `RefreshTokenStore` SQL implementation
- [x] **Admin seeding** — `seedAdmin` goroutine: creates admin user from `ADMIN_USERNAME` / `ADMIN_PASSWORD` env vars when no users exist
- [x] **`GET /api/v1/users`** (admin) — list all users
- [x] **`POST /api/v1/users`** (admin) — create user with role
- [x] **`PUT /api/v1/users/:id/role`** (admin) — change role
- [x] **`DELETE /api/v1/users/:id`** (admin, prevents self-deletion)
- [x] **`GET /api/v1/users/me`** (any authenticated user)

#### 5.3 RBAC
- [x] Roles: `admin | operator | viewer` — enforced at route level via `RequireRole` middleware
- [x] `viewer` — GET on runs, workflows, HITL pending
- [x] `operator` — viewer + POST runs, triggers, HITL signal; workflow GET
- [x] `admin` — full access including workflow POST/DELETE and all user management routes

#### 5.4 Frontend
- [x] **`frontend/src/stores/auth.js`** — `useAuthStore` (login, logout, refresh, fetchMe, init, hasRole); shared `api` axios instance with `Authorization: Bearer` interceptor and 401→refresh→retry logic; access token in `sessionStorage`, refresh token in `localStorage`
- [x] **`frontend/src/views/Login.vue`** — centered login form, redirects to `?redirect` or `/dashboard` on success
- [x] **`frontend/src/router/index.js`** — `beforeEach` guard: calls `authStore.init()` on first nav, redirects to `/login` if unauthenticated; public routes marked `meta: { public: true }`
- [x] **`frontend/src/App.vue`** — renders `<Sidebar />` only when authenticated; WebSocket lifecycle tied to auth state
- [x] **`frontend/src/composables/useWebSocket.js`** — appends `?token=` to WS URL; on 1008 close calls `authStore.logout()` and redirects to login
- [x] **Execution and workflow stores** — migrated to shared `api` instance from auth store
- [x] **25 auth package tests** — JWT round-trip, wrong secret, malformed/empty token, all roles, TTL values, `ValidRole`, `HashPassword`/`CheckPassword`, `GenerateRefreshToken` uniqueness/length/hash-match, `HashToken` determinism; all passing
- [x] **2 additional API handler tests** — missing token → 401, invalid token → 401
- [x] **Full test suite** — 125 tests, 7 packages, all green; clean build

### Architecture decisions made
- Access tokens are stateless JWTs (15min TTL) — server validates signature only; no DB lookup on every request
- Refresh tokens are opaque random values stored as `hex(sha256(raw))` — DB leak cannot replay tokens; raw value lives in memory only during request/response
- Token rotation on every refresh — limits the window of a stolen refresh token to one use
- WebSocket uses query param token (not header) — browser WebSocket API cannot set custom headers during the upgrade handshake
- Admin seeding runs in a goroutine — non-blocking; `HasAnyUser` check prevents duplicate seeding on restart
- `RequireRole` middleware is variadic — a route can allow multiple roles without combinatorial middleware chains
- Refresh token revocation is per-user (logout revokes all), not per-device — simpler without meaningful UX regression for the target audience

### Deferred
- HITL `assignee` field — user can only resolve their own tasks; tracked for M6 HITL UX work
- HITL assignee notifications (email/webhook) — separate concern, not in core auth layer
- Per-device token tracking — not needed until multi-device concurrent sessions become a real use case

---

## M6 — Designer UX ✅

### Goal
The workflow designer should be fast and intuitive to use without needing to edit YAML directly.

### Completed

#### 6.1 Auto-layout
- [x] **`frontend/src/composables/useLayout.js`** — `useLayout()` composable wrapping dagre's Sugiyama layout; configurable direction (TB default); handles disconnected nodes and stale edge refs gracefully
- [x] **Auto-layout button in Designer toolbar** — "Layout" button calls `applyLayout()` + `fitView({ duration: 300 })` for smooth animation
- [x] **Auto-layout on workflow load** — applied automatically when a workflow has no explicitly saved positions (all at default `{200, 100}`)
- [x] **Position persisted** — `buildDefinition()` includes `position` on every state; layout is saved on next "Save"
- [x] **Monitor auto-layout** — `flowNodes` in Monitor.vue applies dagre when no explicit positions are present in the workflow definition

#### 6.2 Drag-to-connect
- [x] **`PendingConnectionPanel`** — replaces `window.prompt()`; shown in the right panel when user completes an edge drag; displays `source → target` arrow, trigger name input, guard expression input; Enter confirms; Cancel dismisses
- [x] **`confirmConnection()` / `cancelConnection()`** — manage `pendingConnection` ref; edge created only when trigger name is non-empty
- [x] **Edge panel** — clicking an existing edge shows `EdgePanel` with trigger name + guard editing; displays `from → to` path for context

#### 6.3 Agent config panel
- [x] **`StatePanel` expanded** — shows Name, Type, Agent name, Timeout, On-timeout fields; when `agent` is non-empty, reveals an "Agent Config" section
- [x] **Agent Config section** — Model input (text, placeholder `claude-sonnet-4-6`), Task Queue input, MCP Servers input, System Prompt textarea (resizable)
- [x] **`agents` array in Designer** — `ref([])` tracking all agent definitions; `getAgent(name)` lookup; `upsertAgent(def)` on Apply
- [x] **Round-trip** — `loadDefinition` restores agents from YAML; `buildDefinition` serialises them back; empty/undefined fields omitted cleanly
- [x] **Agents summary** in workflow panel — shows list of configured agents and their model when no node is selected

#### 6.4 YAML split view
- [x] **Three-way view toggle** — Canvas | Split | YAML (replaces the old Visual / YAML toggle)
- [x] **Split view** — canvas (60%) + YAML editor (40%) side by side; canvas is fully interactive (drag, click-to-edit, add state)
- [x] **Canvas → YAML live sync** — `watch([nodes, edges], ...)` and `watch(meta, ...)` update YAML textarea in real time when in Split mode; guarded by `fromYamlUpdate` flag to prevent loops
- [x] **YAML → Canvas debounced sync** — `onYamlInput` parses YAML after 400ms; updates canvas via `loadDefinition(def, null, false)`; parse errors shown inline in Split header bar
- [x] **Switching views** — switching to Split or YAML snapshots the current canvas state to `yamlSource`

#### 6.5 Run monitor improvements
- [x] **Timeline tab** — replaces the basic history list; vertical timeline with guide line, dots, trigger badges, relative timestamps ("just now", "2m ago"), and a pulsing green dot for the currently active state
- [x] **Active state glow ring** — `StateNode.vue` applies type-specific glow CSS class (`glow-initial`, `glow-intermediate`, `glow-hitl`, `glow-terminal`) instead of a hard-coded indigo ring; each color matches the state type indicator
- [x] **Blackboard live updates** — `BlackboardPanel` already reacts to `run.blackboard` via Pinia store; no changes needed — works with existing WebSocket event flow
- [x] **Agent thinking panel** — added pulsing violet dot indicator; improved heading visibility

### Architecture decisions made
- Dagre chosen over ELK.js — smaller bundle, synchronous API (no async layout required), sufficient for the typical 5–20 node graphs
- `fromYamlUpdate` flag (not a full lock) prevents YAML-parse → canvas-update → YAML-overwrite loop while keeping the implementation simple
- `PendingConnectionPanel` in the right panel (not a floating modal overlay) — keeps the canvas unobstructed during connection naming; consistent placement with node/edge panels
- Agent config stored in `agents[]` array at workflow level (not inline per state) — matches the Go `AgentDef` struct and allows multiple states to share one agent definition
- Split view canvas remains fully interactive — users can drag nodes, the YAML mirror updates in real time; editing in YAML also updates the canvas after debounce

### Deferred
- HITL `assignee` filtering — viewer should only see/resolve HITL tasks assigned to them; needs user identity in HITL events
- MCP server template expansion preview in agent panel — show resolved URL in split view
- Undo/redo for canvas actions — useful but not critical for user interviews
- Edge routing: orthogonal / elbow paths — current bezier curves are fine for interviews

---

## M7 — Production Readiness ⬜

### Goal
The platform should be observable, maintainable, and safe to operate in production.

### Tasks

#### 7.1 Structured logging
- [ ] Replace `log.Printf` with `slog` (structured JSON logs)
- [ ] Log levels from config
- [ ] Request ID propagation through API → engine → activities

#### 7.2 Metrics
- [ ] Prometheus metrics endpoint (`/metrics`)
- [ ] Key metrics: runs started/completed/failed per workflow, activity duration, queue depth, HITL time-to-resolution
- [ ] Grafana dashboard template

#### 7.3 Database migrations
- [ ] Adopt a migration tool (golang-migrate or goose)
- [ ] Version all schema changes as numbered migration files
- [ ] Auto-run pending migrations on server startup

#### 7.4 Health checks
- [ ] `GET /health` — already exists (basic)
- [ ] `GET /health/ready` — checks DB + Redis + Temporal connectivity
- [ ] `GET /health/live` — lightweight liveness probe

#### 7.5 Graceful degradation
- [ ] If Temporal is unavailable at startup, fall back to direct mode with a warning
- [ ] If Redis is unavailable, use local bus with a warning (already partially done)
- [ ] Document trade-offs of each fallback mode

#### 7.6 Docker / deployment
- [ ] Multi-stage Dockerfile for server binary
- [ ] Multi-stage Dockerfile for worker binary
- [ ] `docker-compose.yml` for full local stack (server + worker + postgres + redis + temporal)
- [ ] Kubernetes manifests (Deployment, Service, ConfigMap) — optional

---

---

## M8 — Intelligence 🔄

### Goal
Make the platform smarter: let an LLM draft entire workflows from plain-language descriptions, give each state its own instructions/context, and allow fully-deterministic states to run without an LLM at all.

### M8.1 — Per-State Prompts ✅

#### Completed
- [x] **`StateDef.Instructions string`** — new optional field in `pkg/asmtypes/workflow.go`; holds a short plain-language description of what should happen in this state; shown in the designer panel and injected into the LLM system prompt
- [x] **`AgentDef.Config["prompt"]` wired** — `buildSystemPrompt()` in `internal/executor/executor.go` now checks `agent.Config["prompt"]` as a fallback when `PromptTemplate` is not set; enables inline prompts defined directly in the workflow YAML without requiring a file on disk
- [x] **`{{.Instructions}}` in system prompt template** — state-level instructions are injected between the rules section and the final JSON format spec; the LLM knows the purpose of the current state even when using a generic shared agent
- [x] **Designer StatePanel "Instructions" field** — textarea at the top of the state panel (above agent config); round-trips correctly through `loadDefinition` / `buildDefinition`

### M8.2 — MCP Server Registry ✅

#### Completed
- [x] **`mcp_registry.yaml`** — static config file in project root; maps logical server names to env var keys and human descriptions; version-controlled
- [x] **`internal/designer/registry.go`** — `LoadMCPRegistry(path)` loads the YAML file; `MCPServerEntry` struct with name, env_var, and description; cached after first load
- [x] **`GET /api/v1/mcp-servers`** — returns the list of registered MCP servers (name + description only; env var key not exposed); used by the frontend AI dialog to populate the server picker

### M8.3 — AI Workflow Generator ✅

#### Completed
- [x] **`internal/designer/generator.go`** — `Generator` struct using the platform's existing `llm.Provider` infrastructure; `Generate(ctx, description, mcpServerNames)` returns a validated `WorkflowDef`; system prompt embeds the full YAML schema + all available MCP server descriptions; response validated via `asmtypes.LoadFromYAML()`
- [x] **`POST /api/v1/designer/generate`** — accepts `{ "description": "...", "mcp_servers": ["name", ...] }`; returns `{ "yaml": "...", "definition": {...} }`; operator/admin only
- [x] **Dashboard "Generate with AI" dialog** — modal with description textarea and MCP server multi-select; on Generate, opens the result directly in the Designer for review before saving

### M8.4 — Script State Type ✅

#### Completed
- [x] **`StateType = "script"`** — new type constant in `pkg/asmtypes/workflow.go`
- [x] **`ScriptDef` struct** — `Trigger string` (expr-lang expression that evaluates to a trigger name) and `Updates map[string]string` (blackboard field → expr-lang expression); attached as `Script *ScriptDef` on `StateDef`
- [x] **Engine: script state execution** — when `state.Type == "script"`, the engine evaluates the `ScriptDef` expressions using the same `expr-lang` evaluator already used for guards; no LLM call; output is an `AgentOutput` with computed `Trigger` and `BlackboardUpdates`
- [x] **Designer: script state editor** — when type is set to `"script"`, the agent section is replaced with a script editor showing Trigger Expression and a key-value editor for Updates; syntax-highlighted via monospace font

### Architecture decisions made
- Per-state `Instructions` is on `StateDef`, not `AgentDef` — states have a purpose independent of which agent serves them; multiple states can share one agent but each has unique instructions
- `Config["prompt"]` fallback before file-based `PromptTemplate` — inline prompts reduce the need for a filesystem alongside the workflow YAML; file-based prompts remain for complex multi-paragraph prompts
- MCP server registry is a static YAML file (not a DB table) — version-controlled; curated by operators; promotes to DB in M10 if dynamic management is needed
- `ScriptDef.Trigger` evaluates to a string (not bool) — allows a script to choose between multiple outgoing transitions depending on computed values; more expressive than a single bool guard
- Script states call no external services — deterministic, synchronous, no retries needed; the engine short-circuits the executor call entirely

### Deferred to M9
- LLM-assisted specialist worker code generation — operator marks a state as "generate code", LLM produces a Go file implementing `HandlerFunc`, Monaco editor for review, downloads as `.go` file

---

## M9 — Code Generation ⬜

### Goal
For states where the logic is fully understood but writing the Go boilerplate is tedious, let an LLM scaffold the specialist worker code. A human reviews and approves before deployment.

### Tasks

#### 9.1 LLM-assisted specialist worker scaffolding
- [ ] **`POST /api/v1/designer/scaffold`** — accepts state description + blackboard schema + possible triggers; returns generated Go file content
- [ ] **Generator prompt** — system prompt teaches the LLM the `HandlerFunc` signature, `AgentOutput` struct, import paths, and best practices; blackboard schema and trigger list are injected as context
- [ ] **Monaco editor panel in Designer** — "Generate Code" button in the state panel when type is `intermediate`; opens a full-height Monaco editor with the generated Go; read-only + copy + download buttons
- [ ] **Download as `.go` file** — browser download of `<agent_name>_handler.go`; operator compiles this into a specialist worker binary

#### 9.2 Specialist worker template project
- [ ] **`cmd/specialist-worker-template/`** — pre-wired Go project with go.mod, main.go, and a placeholder handler; operator replaces the handler with LLM-generated code and runs `go build`
- [ ] **`Makefile` target** — `make specialist name=validate-invoices queue=validate-workers` scaffolds the directory

---

## M10 — Generated MCP Servers ⬜

### Goal
For deterministic logic that multiple workflows share, package it as a standalone MCP server so any workflow can consume it via `mcp_servers` config without touching the platform binary.

### Tasks

#### 10.1 MCP server scaffolding
- [ ] **`POST /api/v1/designer/scaffold-mcp`** — accepts a tool description (name, input/output schema, logic description); returns a complete Go MCP server implementation
- [ ] **Generated server** — Go HTTP server exposing JSON-RPC 2.0 `tools/list` + `tools/call` endpoints; single-binary; Docker-friendly

#### 10.2 MCP Server Registry promotion to DB
- [ ] **`mcp_servers` DB table** — admin-manageable via API; stores name, URL pattern, description, created_by
- [ ] **Admin UI** — manage registered MCP servers from the platform dashboard
- [ ] **Migrate from static YAML** — `mcp_registry.yaml` still supported as seed source; DB takes precedence when populated

---

## Technical Debt & Known Issues

| Issue | Severity | Notes |
|---|---|---|
| Tool call deduplication inside Temporal activity retries | Low | Tools should be idempotent; explicit tracking deferred to M4 |
| No authentication on any endpoint | ~~High~~ | ✅ Resolved in M5 |
| `internal/scheduler/scheduler.go` is a stub | Medium | Cron-triggered workflows not yet functional |
| Frontend designer requires prompt to create edges (not drag) | ~~Medium~~ | ✅ Resolved in M6 — `PendingConnectionPanel` replaces `window.prompt()` |
| Worker binary requires Postgres — no memory store fallback | Low | Intentional: temporal mode needs DB |
| `PublishEventParams.Data` is `interface{}` — deserialises as map on replay | Low | Safe but loses type info; consider typed event union |
| No database migration tooling | Medium | Tracked in M7.3 |

---

## Architecture Decisions Log

| Date | Decision | Rationale |
|---|---|---|
| 2026-03-29 | Temporal workflow = full run, not per-activity | Full durability: runs survive restarts between any two states, not just during agent execution |
| 2026-03-29 | WorkflowDef passed as Temporal workflow parameter | Avoids non-determinism on replay; def is immutable once a run starts |
| 2026-03-29 | TemporalEngineClient interface in orchestrator package | Prevents import cycle: `internal/temporal` imports `internal/orchestrator`; interface cannot live in both |
| 2026-03-29 | Workers return trigger + blackboard updates only | Keeps workers stateless and decoupled from routing logic; state machine owns all transitions |
| 2026-03-29 | Single task queue by default | Simplest start; per-agent queues added in M4 when specialist workers are introduced |
| 2026-03-29 | Blackboard schema in workflow YAML, not enforced by DB | Schema validation at runtime in Go; DB stores raw JSON. Simpler, faster iterations on schema |
| 2026-03-29 | MCP server URLs from env vars only, never hardcoded | Security: server addresses are infrastructure config, not application code. Referenced in YAML as `{{ env.VAR }}` |
| 2026-03-29 | Custom MCP client over official SDK | Minimal dependencies; HTTP POST covers all current server implementations; SSE added if needed |
| 2026-03-29 | Tool errors serialised and fed back to LLM | LLM can reason about failures and choose fallback paths; hard abort on tool error would waste earlier context |
| 2026-03-29 | Agent tools override global tools on name collision | Specialist agents can shadow platform defaults without modifying shared config |
| 2026-03-30 | Task queue as the sole routing discriminant for specialist workers | Temporal routes by (task queue, activity name); both LLM and specialist workers register activity name `ExecuteAgent`; no additional routing metadata needed |
| 2026-03-30 | Specialist worker registration in code, not a registry table | Explicit `sw.Register(name, fn)` is discoverable, type-safe, and requires no DB schema; a worker that doesn't register a handler fails fast with a clear error |
| 2026-03-30 | Specialist workers need no store/bus/LLM dependencies | Reduces operational burden; a specialist team only needs `TEMPORAL_ADDRESS` to deploy |
| 2026-03-30 | Stateless JWT access tokens (15min) + opaque refresh tokens (7 days) | Short TTL limits blast radius of a leaked access token; refresh tokens are revocable without invalidating sessions |
| 2026-03-30 | Refresh tokens stored as `hex(sha256(raw))` in DB | DB leak cannot replay tokens; raw value only lives in memory during the request/response cycle |
| 2026-03-30 | Token rotation on every `/auth/refresh` call | Stolen refresh token can only be used once before it is invalidated; limits replay window to the TTL of the next token |
| 2026-03-30 | WebSocket token via `?token=` query param | Browser WebSocket API cannot set custom HTTP headers during the upgrade handshake; query param is the standard alternative |
| 2026-03-30 | `RequireRole` middleware is variadic | A route can allow multiple roles (`RequireRole(admin, operator)`) without combinatorial middleware chains |
| 2026-03-30 | Admin seeding in a background goroutine | Non-blocking: server starts immediately; `HasAnyUser` guard prevents duplicate seed on restart |
| 2026-03-30 | Dagre over ELK.js for auto-layout | Synchronous API, smaller bundle, sufficient for typical 5–20 node state machine graphs |
| 2026-03-30 | `fromYamlUpdate` flag prevents canvas↔YAML loop in split view | Avoids a full mutex; the flag is held for a single Vue tick, keeping the reactive system simple |
| 2026-03-30 | `PendingConnectionPanel` in right panel (not modal overlay) | Canvas stays unobstructed during edge naming; consistent panel placement with node/edge property panels |
| 2026-03-30 | Agent config in `agents[]` array (not inline per state) | Matches Go `AgentDef` struct; allows multiple states to share one agent definition |
| 2026-03-30 | Three-way view toggle (Canvas / Split / YAML) | Clear mental model: Canvas = visual designer, Split = see YAML while designing, YAML = text editor for power users |
| 2026-03-30 | `StateDef.Instructions` on state (not agent) | States have a purpose independent of which agent serves them; shared agents can serve multiple states each with unique context |
| 2026-03-30 | `Config["prompt"]` inline before file-based `PromptTemplate` | Inline prompts reduce filesystem dependency; file-based prompts remain for complex multi-paragraph system prompts |
| 2026-03-30 | MCP server registry as static YAML (not DB table) | Version-controlled; operator-curated; zero additional schema changes; promotes to DB in M10 if dynamic management is needed |
| 2026-03-30 | `ScriptDef.Trigger` evaluates to a string | Allows choosing between multiple outgoing transitions based on computed values; more expressive than a bool guard |
| 2026-03-30 | Script states skip the executor entirely | Deterministic, synchronous, no LLM tokens consumed; engine evaluates expr-lang expressions inline |
| 2026-03-30 | AI generator validates output via `LoadFromYAML` | LLM output may be malformed; round-tripping through the existing YAML loader gives schema validation for free |

---

## Running the Platform

### Development (direct mode, no external dependencies)
```bash
cd asm-platform
go run ./cmd/server
# Server starts on :8080, uses in-memory store and local event bus
```

### With Postgres + Redis
```bash
# Start infrastructure
docker compose up -d postgres redis

# Run server
source .env
go run ./cmd/server
```

### With Temporal (full production mode)
```bash
# Start all infrastructure
docker compose up -d

# Terminal 1 — API server
EXECUTOR_MODE=temporal go run ./cmd/server

# Terminal 2 — Temporal worker
EXECUTOR_MODE=temporal go run ./cmd/worker
```

### With specialist workers (multi-queue mode)
```bash
# Terminal 3 — specialist worker (validates invoices without LLM)
SPECIALIST_TASK_QUEUE=validate-workers go run ./cmd/specialist-worker
```

The workflow YAML must declare `task_queue: validate-workers` on the agent that should route here.

### Frontend (dev server)
```bash
cd frontend
npm run dev
# → http://localhost:5173  (proxied to :8080)
# Login with ADMIN_USERNAME / ADMIN_PASSWORD from .env
```

### Run tests
```bash
go test ./... -timeout 30s
# 125 tests, 7 packages
```
