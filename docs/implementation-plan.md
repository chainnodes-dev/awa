# ASM Platform — Implementation Plan

> **Living document.** Update status, notes, and decisions as the project evolves.
> Last updated: 2026-04-04 (M11 complete — DB-backed LLM config UI, multi-provider support, HITL fixes)

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
| M5.2 | Multi-tenancy — tenant model, context propagation, super_admin, tenant API | ✅ | Done |
| M6 | Designer UX — auto-layout, inline config, drag-connect | ✅ | Done |
| M7 | Production readiness — observability, migration tooling | ✅ | Done |
| M8 | Intelligence — AI workflow generator, per-state prompts, script states | ✅ | Done |
| M9 | Code generation — LLM-assisted specialist worker scaffolding | ✅ | Done |
| M10 | Generated MCP servers — sharable deterministic logic as services | ✅ | Done |
| M11 | LLM config UI — DB-backed provider management, hot-reload, HITL fixes | ✅ | Done |

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
- [x] **Ollama provider** — `NewOllamaProvider(baseURL, defaultModel)` for local / self-hosted LLMs; same `Provider` interface; `LLM_PROVIDER=ollama`, `OLLAMA_BASE_URL`, `OLLAMA_MODEL` env vars
- [x] **LLM provider auto-fallback** — if `LLM_PROVIDER` is unset or the named provider is not registered, server falls back to the first registered provider; `LLM_PROVIDER` default changed from `"anthropic"` to `""` so a local-only setup works without configuration
- [x] **`.env.example`** — MCP server URL documentation pattern added
- [x] **21 executor tests** — tool-calling loop, error handling, max iterations, `parseAgentOutput`, `buildAssistantBlocks`, registry merge; all passing
- [x] **Full test suite** — 80 tests, 5 packages, all green; clean build

### Architecture decisions made
- MCP server URLs must come from environment variables — never hardcoded; workflow YAML uses `{{ env.VAR }}` template expansion
- Custom JSON-RPC 2.0 HTTP client rather than an external MCP SDK — keeps the dependency tree minimal and the protocol surface explicit
- Agent tools win over global tools on name collision in `Registry.Merge` — agent-specific behaviour can always shadow a platform default
- Tool errors are serialised and fed back to the LLM rather than aborting — the model can reason about failures and choose a fallback path
- `maxToolIterations = 10` hard-coded constant — prevents runaway loops; configurable per-agent planned for M4+
- LLM provider auto-fallback — `LLM_PROVIDER` default is `""` and the server falls back to the first registered provider; local-only (Ollama) setups work without any configuration change

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

## M5.2 — Multi-Tenancy ✅

### Goal
Every B2B SaaS customer is a tenant. Without this, all customers share the same workflow definitions and run history, making the SaaS model impossible to retrofit later. Added before any production data exists — the cheapest possible moment.

### Completed

#### 5.2.1 Data model
- [x] **`migrations/003_tenants.sql`** — `tenants` table (UUID PK, name, slug UNIQUE, created_at); `tenant_id` FK added to `workflow_definitions`, `workflow_runs`, `users`; per-tenant UNIQUE constraints on `(tenant_id, name, version)` and `(tenant_id, username)`; `super_admin` added to role CHECK constraint; default tenant seeded at `00000000-0000-0000-0000-000000000001`
- [x] **`pkg/asmtypes/runtime.go`** — `TenantID string` added as first field of `WorkflowRun`
- [x] **`internal/store/user.go`** — `Tenant` struct (ID, Name, Slug, CreatedAt); `TenantID` field on `User`; `TenantStore` interface (CreateTenant, GetTenant, GetTenantBySlug, ListTenants, DeleteTenant, HasAnyTenant)

#### 5.2.2 Context propagation
- [x] **`internal/store/interfaces.go`** — `DefaultTenantID` constant; `WithTenantID(ctx, id)` / `TenantIDFromContext(ctx)` helpers; `Store` embeds `TenantStore`; `RunFilter.TenantID` field
- [x] **`internal/auth/jwt.go`** — `TenantID string \`json:"tid"\`` in `Claims`; `RoleSuperAdmin = "super_admin"` constant; `GenerateAccessToken` takes `(userID, tenantID, username, role)`
- [x] **`internal/auth/middleware.go`** — `RequireAuth` propagates `tenant_id` into request context; `RequireRole` grants `super_admin` implicit access everywhere

#### 5.2.3 Store implementations
- [x] **`internal/store/memory.go`** — full `TenantStore`; definition keys `tenantID:name@version`; user keys `tenantID:username`; all list/get filter by `TenantIDFromContext(ctx)`
- [x] **`internal/store/postgres.go`** — full `TenantStore`; all queries include `tenant_id` WHERE clauses; `ListRuns` filters by `RunFilter.TenantID`; `ListPendingHITL` joins `workflow_runs` for tenant scope

#### 5.2.4 API layer
- [x] **`internal/api/auth_handlers.go`** — Login accepts `tenant_slug` (defaults to `"default"`); user lookup scoped to resolved tenant
- [x] **`internal/api/server.go`** — first-boot: creates default tenant before admin user; admin seeded as `super_admin`
- [x] **`internal/api/tenant_handlers.go`** (new) — `CreateTenant`, `ListTenants`, `GetTenant`, `DeleteTenant`; default tenant protected from deletion
- [x] **`internal/api/routes.go`** — `/api/v1/tenants` CRUD routes guarded by `RequireRole(RoleSuperAdmin)`
- [x] **`internal/api/handlers.go`** — `ListRuns` populates `RunFilter.TenantID` from context
- [x] **`internal/api/user_handlers.go`** — `CreateUser` sets `TenantID` from JWT claims; blocks creating `super_admin` via user API
- [x] **`internal/orchestrator/engine.go`** — `StartRun` sets `run.TenantID = store.TenantIDFromContext(ctx)`
- [x] Test suite updated — `GenerateAccessToken` 4-arg calls; engine HITL triggers aligned; `super_admin` in `ValidRole`; all tests pass

### Architecture decisions made
- **Context-based tenant propagation** — `tenant_id` flows JWT → middleware → Go context; store methods read from context with no signature changes
- **`state_transitions` and `hitl_requests` not tenant-scoped directly** — always accessed via `run_id` FK; tenant isolation is inherited through the run
- **Username uniqueness is per-tenant** — `(tenant_id, username)` UNIQUE constraint; same username can exist in different tenants (B2B SaaS standard)
- **`super_admin` bypasses all `RequireRole` checks** — platform operators have global access; cannot be created via user API
- **Default tenant has a fixed UUID** — `00000000-0000-0000-0000-000000000001`; ensures backfill and first-boot seeding are idempotent

### Deferred
- Frontend tenant switcher UI — tenant determined by `tenant_slug` at login; no in-app tenant display yet
- Tenant deletion safety check — blocking delete if active runs/users exist not yet enforced
- Per-tenant quotas and rate limits

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

## M7 — Production Readiness ✅

### Goal
The platform should be observable, maintainable, and safe to operate in production.

### Completed

#### 7.1 Structured logging ✅
- [x] **`internal/logger/logger.go`** — `Init(logLevel)` configures global slog JSON handler on stdout; `WithRequestID(ctx, id)` / `RequestIDFromContext(ctx)` / `FromContext(ctx)` for structured per-request logging
- [x] **`LOG_LEVEL` env var** — already in config; parsed by `logger.Init` into `slog.Level` (debug/info/warn/error)
- [x] **`internal/auth/requestid.go`** — `RequestID()` gin middleware reads `X-Request-ID` header (or generates a UUID); writes ID back in response header and into Go context
- [x] **All `log.Printf` replaced with `slog`** — `cmd/server`, `cmd/worker`, `cmd/specialist-worker`, `internal/api/server`, `seed_workflows`, `websocket`, `orchestrator/engine`, `executor/executor`, `migrate/migrate`, `scheduler/scheduler`
- [x] Structured key-value fields throughout: `run_id`, `state`, `agent`, `error`, `workflow`, `provider`, etc.

#### 7.2 Metrics ✅
- [x] **`internal/metrics/metrics.go`** — `promauto`-based package; defines all counters, gauges, and histograms with `asm_` prefix: `RunsStartedTotal`, `RunsCompletedTotal`, `RunsFailedTotal`, `RunsActive`, `RunDurationSeconds`, `StateTransitionsTotal`, `AgentExecutionsTotal`, `AgentDurationSeconds`, `HITLRequestsTotal`, `HITLResolutionDurationSeconds`, `HTTPRequestsTotal`, `HTTPRequestDurationSeconds`
- [x] **Engine instrumented** — `StartRun`, `completeRun`, `failRun`, `transitionTo`, `handleHITL` all emit the corresponding metrics
- [x] **Executor instrumented** — `Execute` measures agent duration and execution count with success/error label at every return point
- [x] **`GET /metrics`** — `promhttp.Handler()` mounted in `registerRoutes`; no auth required (standard Prometheus scrape target)
- [x] **`internal/api/metrics_middleware.go`** — `MetricsMiddleware()` gin handler records `HTTPRequestsTotal` and `HTTPRequestDurationSeconds`; uses `c.FullPath()` (route template) to keep cardinality bounded; unmatched routes get the label `unmatched`
- [x] **`dashboards/asm-overview.json`** — importable Grafana dashboard; panels for run lifecycle, agent duration, HITL resolution time, HTTP request rate and latency; templated datasource variable

#### 7.3 Database migrations ✅
- [x] **golang-migrate** added as dependency (`github.com/golang-migrate/migrate/v4`)
- [x] Migration files renamed to `000001_init.up.sql` / `.down.sql` format; `.down.sql` files written for all three migrations
- [x] **`migrations/embed.go`** — exposes `migrations.FS embed.FS` so SQL files are compiled into the binary; no filesystem path required at runtime
- [x] **`internal/migrate/migrate.go`** — `migrate.Up(databaseURL)` applies all pending migrations idempotently; uses `iofs` source from embedded FS
- [x] **`cmd/server/main.go`** — calls `migrate.Up` before `store.NewPostgresStore`; server refuses to start if migration fails

#### 7.4 Health checks ✅
- [x] `GET /health` — backward-compat alias for `/health/live`
- [x] `GET /health/live` — always 200 while process is running (Kubernetes liveness probe)
- [x] `GET /health/ready` — pings all registered deps (postgres, redis); 200 if all healthy, 503 with per-dep status if any fails (Kubernetes readiness probe)
- [x] **`internal/health/health.go`** — `Checker` with `Add(name, Pinger)`, `Live`, `Ready` handlers; 3-second timeout per readiness check
- [x] `PostgresStore.Ping(ctx)` and `RedisBus.Ping(ctx)` methods added

#### 7.5 Graceful degradation ✅
- [x] **Temporal fallback**: `client.Dial` is lazy (no TCP handshake); `temporalpkg.CheckHealth` is called with a 5 s deadline after the client is created. If the health check fails, the server logs a warning and continues in direct mode rather than exiting — operators can re-deploy once Temporal recovers
- [x] **Redis fallback**: `NewRedisBus` only parses the URL; a 3 s `Ping` is now performed immediately after creation. If Redis is unreachable the server falls back to the in-process `LocalBus` with a warning noting that multi-process fan-out is disabled
- [x] **Postgres fallback**: already present — falls back to `MemoryStore` with a warning
- [x] **LLM provider fallback**: already present — cycles through registered providers; warns if none are available

**Trade-offs documented:**

| Degraded component | Fallback | What you lose |
|---|---|---|
| **Temporal** | Direct in-process executor | Run durability across restarts; no replay on crash; no visibility in Temporal UI |
| **Redis** | `LocalBus` (in-process) | WebSocket events visible only to the process that generated them; separate server replicas don't share events |
| **Postgres** | `MemoryStore` | Persistence across restarts; all run history is lost on shutdown |
| **LLM provider** | Log warning; agent states fail at runtime | Agent states error with "no LLM providers registered" |

#### 7.6 Docker / deployment ✅
- [x] Multi-stage Dockerfile for server and worker binaries (`ARG CMD=server|worker|specialist-worker`)
- [x] `docker-compose.yml` for full local stack (server + worker + postgres + redis + temporal)
- [x] `Makefile` targets — `docker-build`, `stack`, `stack-down`
- [x] **`deploy/k8s/namespace.yaml`** — `asm` namespace with `app.kubernetes.io/part-of` label
- [x] **`deploy/k8s/configmap.yaml`** — non-secret config (PORT, LOG_LEVEL, EXECUTOR_MODE, Temporal coords, LLM model names, ADMIN_USERNAME)
- [x] **`deploy/k8s/secret.yaml`** — Opaque secret template for DATABASE_URL, REDIS_URL, JWT_SECRET, ANTHROPIC_API_KEY, OPENAI_API_KEY, ADMIN_PASSWORD; placeholder values + comment to use external-secrets/vault in production
- [x] **`deploy/k8s/server-deployment.yaml`** — 2-replica Deployment; `RollingUpdate` with `maxUnavailable: 0`; readiness (`/health/ready`) + liveness (`/health/live`) probes; `topologySpreadConstraints` across nodes; `preStop` sleep for connection drain; Prometheus scrape annotations; non-root / read-only filesystem security context
- [x] **`deploy/k8s/server-service.yaml`** — ClusterIP Service on port 80→8080; commented-out LoadBalancer and nginx Ingress examples (incl. WebSocket / SSE proxy settings)
- [x] **`deploy/k8s/worker-deployment.yaml`** — 2-replica Deployment; `terminationGracePeriodSeconds: 60` for Temporal long-poll drain; always overrides `EXECUTOR_MODE=temporal`; exec liveness probe
- [x] **`deploy/k8s/specialist-worker-deployment.yaml`** — same pattern as worker, separate image
- [x] **`deploy/k8s/hpa.yaml`** — `autoscaling/v2` HPA for server (2–10 replicas, CPU 70% / mem 80%) and worker (2–20 replicas, CPU 70%); conservative scale-down window (5 min)
- [x] **`Makefile`** — `k8s-apply`, `k8s-delete`, `k8s-status` targets

---

## M8 — Intelligence ✅

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

## M9 — Code Generation ✅

### Goal
For states where the logic is fully understood but writing the Go boilerplate is tedious, let an LLM scaffold the specialist worker code. A human reviews and approves before deployment.

### Completed

#### 9.1 LLM-assisted specialist worker scaffolding
- [x] **`POST /api/v1/designer/scaffold`** — accepts `agent_name`, `state_name`, `description`, optional triggers + blackboard schema; returns `{ "code": "..." }` (operator/admin only)
- [x] **`Generator.Scaffold()`** — system prompt in `internal/designer/scaffolder.go` teaches the LLM the `HandlerFunc` signature, `AgentOutput` struct, import paths, type assertion patterns, and idempotency requirements
- [x] **"Generate Code" panel in Designer** — appears in the State Panel when `type ≠ script` and `agent` is set; shows spinner while generating, then renders code in a `<pre>` block with **Copy** and **Download .go** buttons; no Monaco dep needed

#### 9.2 Specialist worker template project
- [x] **`cmd/specialist-worker-template/main.go`** — documented, self-contained starter project with placeholder handler, env var wiring, and graceful shutdown
- [x] **`make specialist name=… queue=…`** — copies the template directory and prints setup instructions

---

## M10 — Generated MCP Servers ✅

### Goal
For deterministic logic that multiple workflows share, package it as a standalone MCP server so any workflow can consume it via `mcp_servers` config without touching the platform binary.

### Completed

#### 10.1 MCP server scaffolding
- [x] **`POST /api/v1/designer/scaffold-mcp`** — accepts `server_name`, `description`, `tools[]` (name, description, input_schema); returns `{ "code": "..." }` (operator/admin only)
- [x] **`Generator.ScaffoldMCP()`** — system prompt in `internal/designer/mcp_scaffolder.go` teaches JSON-RPC 2.0 protocol, `tools/list` + `tools/call` methods, CORS, `/health`, graceful shutdown, and standard-library-only constraint
- [x] **`cmd/mcp-server-template/main.go`** — standalone JSON-RPC 2.0 server; ships with a `greet` example tool; includes Dockerfile comment; no external dependencies

#### 10.2 MCP Server Registry promotion to DB
- [x] **`migrations/000004_mcp_servers.up.sql`** — `mcp_servers` table with `(tenant_id, name)` unique constraint; applied automatically at startup
- [x] **`MCPServerStore` interface** — `Create`, `Get`, `List`, `Update`, `Delete`, `HasAny`; implemented by both `MemoryStore` and `PostgresStore`
- [x] **`GET /api/v1/mcp-servers`** — DB-backed with YAML fallback; returns full store model for admin UI, compatible with existing `mcp_servers` picker in AI generator dialog
- [x] **`POST/PUT/DELETE /api/v1/mcp-servers`** — admin-only CRUD; `POST` creates with `tenant_id` from JWT claims
- [x] **MCP Servers admin panel** — "MCP Server Registry" section in Dashboard (admin only); table with inline Edit/Delete; `MCPServerModal` component for add/edit; seeded from `mcp_registry.yaml` on first boot (idempotent)

---

## M11 — LLM Config UI, Multi-Provider Support & HITL Fixes ✅

### Goal
Make LLM provider selection observable and manageable without touching env vars or restarting processes. Add Grok, Deepseek, and Gemini alongside existing providers. Fix HITL workflow stuck in Temporal mode and live UI update issues.

### Completed

#### 11.1 DB-backed LLM configuration
- [x] **`migrations/000006_llm_configs.up.sql`** — `llm_configs` table: `(tenant_id, provider)` unique; fields: `api_key`, `base_url`, `default_model`, `enabled`, `is_default`
- [x] **`internal/store/llm_config.go`** — `LLMConfig` struct + `LLMConfigStore` interface (6 methods: List, Get, Upsert, Delete, SetDefault, HasAny)
- [x] **`internal/store/postgres_llm.go`** — full PostgreSQL implementation; `UpsertLLMConfig` uses `ON CONFLICT DO UPDATE`; `SetDefaultProvider` uses a transaction to clear all `is_default` flags then set one
- [x] **`internal/store/memory_llm.go`** — in-memory implementation for dev/test
- [x] **`LLMConfigStore` added to `Store` aggregate interface**

#### 11.2 New LLM providers
- [x] **`internal/executor/llm/grok.go`** — `NewGrokProvider(apiKey, defaultModel)` → OpenAI-compatible wrapper at `https://api.x.ai/v1/chat/completions`; default model `grok-3`
- [x] **`internal/executor/llm/deepseek.go`** — endpoint `https://api.deepseek.com/v1/chat/completions`; default model `deepseek-chat`
- [x] **`internal/executor/llm/gemini.go`** — endpoint `https://generativelanguage.googleapis.com/v1beta/openai/chat/completions`; default model `gemini-2.0-flash`
- [x] **`internal/executor/llm/loader.go`** — `BuildRegistryFromDB(ctx, store) (*Registry, string, error)`: loads enabled providers, constructs appropriate instances; `ProviderDefaultModels` map; `ProviderEndpoint(provider)` helper
- [x] **Thread-safe Registry** — `sync.RWMutex` on all methods; `Reload(snapshot, defaultName)` for atomic hot-reload; `Default()`, `Snapshot()` methods added

#### 11.3 Provider resolution & fallback
- [x] **`internal/executor/executor.go`** — resolution order: `registry.Default()` → `e.defaultProv` fallback → per-agent `config["provider"]` override → Ollama fallback if provider not found → hard error
- [x] **Per-agent provider override** — `config.provider` field in agent YAML; Designer adds provider `<select>` dropdown before the Model field

#### 11.4 First-boot seeding
- [x] **`cmd/server/main.go`** — `seedLLMConfigs(ctx, store, cfg)` inserts env-var-configured providers only when `HasAnyLLMConfig` returns false; idempotent on restart
- [x] **`BuildRegistryFromDB`** replaces the old env-var-only LLM block in both server and worker binaries

#### 11.5 API (Settings)
- [x] **`internal/api/llm_handlers.go`** — `ListLLMConfigs` (keys masked as `***`), `UpsertLLMConfig` (preserves existing key when `***` or empty is sent), `DeleteLLMConfig`, `SetDefaultProvider`, `TestLLMConnection`, `reloadLLMRegistry` helper (hot-reload on save)
- [x] **`internal/api/routes.go`** — `/api/v1/llm-configs` group (admin only): `GET`, `PUT /:provider`, `DELETE /:provider`, `POST /default`, `POST /:provider/test`
- [x] **`NewHandlers` signature** updated to accept 6th `llmRegistry *llm.Registry` parameter; `handlers_test.go` updated to pass `nil` for the new arg

#### 11.6 Worker hot-reload
- [x] **`cmd/worker/main.go`** — background goroutine polls DB every 15 seconds; calls `llmRegistry.Reload(reg.Snapshot(), defaultName)` atomically; no worker restart required after UI config changes

#### 11.7 Settings UI
- [x] **`frontend/src/stores/llm.js`** — Pinia store: `PROVIDERS` metadata, `DEFAULT_MODELS`, `fetchAll`, `upsert`, `remove`, `setDefault`, `testConnection`
- [x] **`frontend/src/views/Settings.vue`** — per-provider cards with enable toggle, API key input, model input, base URL (Ollama), Test/Save buttons; global default provider selector; admin guard
- [x] **`frontend/src/router/index.js`** — `/settings` route added
- [x] **`frontend/src/components/shared/Sidebar.vue`** — gear icon NavItem linking to `/settings`

#### 11.8 HITL fixes
- [x] **Temporal trigger mismatch** — `ASMWorkflow` was mapping resolutions to `"human_approved"`/`"human_rejected"` but workflow YAML only declared `"approved"`/`"rejected"` transitions. Fixed: `trigger = sig.Resolution` (the resolution value IS the trigger, mirroring direct mode)
- [x] **`temporal_workflow_id` never persisted** — `UpdateRun` SQL did not include `temporal_workflow_id`; `GetRun` returned `""` so `SignalHITL` rejected with "run has no temporal workflow ID". Fixed: `UpdateRun` now includes `COALESCE(NULLIF($8,''), temporal_workflow_id)` — persists the ID on first call, preserves it on subsequent updates
- [x] **HITL controls not auto-appearing** — `hitl.waiting` WebSocket event did not update `activeRun.status`. Fixed in `execution.js`: `hitl.waiting` patches `status = 'waiting'`; `hitl.resolved` patches `status = 'running'` and removes from `pendingHITL`
- [x] **HITL errors silently swallowed** — `resolve()` in Monitor.vue had no try/catch. Fixed: errors displayed in an orange banner; `loadRun()` called on failure to restore correct state

### Architecture decisions made
- **DB is authoritative for LLM config after first boot** — env vars seed once; subsequent changes go through the UI/API only; eliminates config drift between env and runtime state
- **API key masking** — `"***"` is both the display value and the "no change" sentinel on upsert; server never returns the raw key after initial insertion
- **Worker 15s poll over push** — avoids a dedicated pub/sub channel for config changes; 15s lag is acceptable for provider config; `Reload` is atomic so in-flight requests see a consistent snapshot
- **`COALESCE(NULLIF($8,''), temporal_workflow_id)` in UpdateRun** — preserves the Temporal workflow ID across all activity-driven `UpdateRun` calls (which pass `TemporalID = ""`); set once on `StartRun` and never overwritten
- **`sig.Resolution` directly as HITL trigger** — no internal mapping layer; workflow YAML trigger names must match the resolution values sent by the client; documented explicitly in architecture.md

---

## Technical Debt & Known Issues

| Issue | Severity | Notes |
|---|---|---|
| ~~Tool call deduplication inside Temporal activity retries~~ | ~~Low~~ | ✅ Resolved — within-execution cache in `executor.go` (SHA-256 key); cross-retry deduplication requires idempotent tool implementations by design |
| ~~No authentication on any endpoint~~ | ~~High~~ | ✅ Resolved in M5 |
| ~~`internal/scheduler/scheduler.go` is a stub~~ | ~~Medium~~ | ✅ Resolved — full cron scheduler with `AddWorkflow`/`RemoveWorkflow`, per-definition `entries` map, graceful drain on shutdown |
| ~~Frontend designer requires prompt to create edges (not drag)~~ | ~~Medium~~ | ✅ Resolved in M6 — `PendingConnectionPanel` replaces `window.prompt()` |
| Worker binary requires Postgres — no memory store fallback | Low | Intentional: temporal mode needs DB |
| ~~`PublishEventParams.Data` is `interface{}` — deserialises as map on replay~~ | ~~Low~~ | ✅ Resolved — `events.Event.Data` changed to `json.RawMessage`; `events.New()` marshals immediately; `Event.Decode()` helper added; `PublishEvent` activity re-marshals Temporal-deserialized data |
| ~~No database migration tooling~~ | ~~Medium~~ | ✅ Resolved in M7.3 — golang-migrate with embedded SQL, applied at startup |
| ~~HITL workflow stuck in Temporal mode~~ | ~~High~~ | ✅ Resolved in M11 — two root causes: trigger name mismatch (`human_approved` vs `approved`) and `temporal_workflow_id` never persisted by `UpdateRun` |
| ~~Worker LLM registry stale after UI config changes~~ | ~~Medium~~ | ✅ Resolved in M11 — worker polls DB every 15s and calls `Registry.Reload()` atomically |
| HITL `assignee` filtering — users should only see/resolve tasks assigned to them | Low | Needs user identity in HITL events and query filtering; tracked for future milestone |
| Tenant deletion safety check | Low | Delete not blocked if tenant has active runs or users; safe for single-tenant deployments |

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
| 2026-04-03 | Context-based tenant propagation (not method params) | Tenant_id flows JWT → middleware → Go context; store methods read via `TenantIDFromContext(ctx)`; zero method signature changes required |
| 2026-04-03 | `state_transitions` and `hitl_requests` not tenant-scoped directly | Both always accessed via `run_id` FK; tenant isolation inherited from owning run; no redundant column needed |
| 2026-04-03 | Username uniqueness is per-tenant, not global | `(tenant_id, username)` UNIQUE constraint; standard B2B SaaS pattern — same username can exist across different tenants |
| 2026-04-03 | `super_admin` cannot be created via user API | Platform role reserved for operators; tenant admins cannot escalate beyond `admin` scope |
| 2026-04-03 | Default tenant has fixed UUID `00000000-…-0001` | Idempotent backfill in migration; first-boot seed is re-runnable; pre-tenant data gets a home without logic changes |
| 2026-04-03 | HITL resolution value used directly as the trigger name | `sig.Resolution` (e.g. `"approve"`) IS the trigger; workflow YAML must match; eliminates hidden string mapping in the engine |
| 2026-04-03 | `promauto` for zero-boilerplate metric registration | Metrics are registered at package init time via `promauto`; no `MustRegister` calls scattered through `main.go`; adding a new metric is one line |
| 2026-04-03 | `c.FullPath()` (route template) for HTTP metric labels | Using the raw request URL would create unbounded cardinality (one series per run ID); the route template (e.g. `/api/v1/runs/:id`) gives bounded, useful labels |
| 2026-04-03 | `client.Dial` is lazy — explicit `CheckHealth` ping at startup | Temporal's gRPC client doesn't connect until the first RPC; without an explicit health check the server would start in Temporal mode but fail on the first `StartWorkflow` call |
| 2026-04-03 | Temporal unavailability → warn + fall back, not `os.Exit` | Matches the Redis and Postgres fallback pattern; platform remains operational with reduced guarantees; operators are informed via structured log warning |
| 2026-04-03 | Redis `Ping` immediately after `NewRedisBus` | `NewRedisBus` only parses the URL (lazy TCP); without a probe the server would silently fall through to `LocalBus` only after the first publish fails at runtime |
| 2026-04-03 | K8s `maxUnavailable: 0` on server Deployment | Zero-downtime rolling updates; new pods must pass the `/health/ready` probe before any old pod is terminated; ensures there is always a healthy server pod during deploys |
| 2026-04-03 | `terminationGracePeriodSeconds: 60` for workers | Temporal workers hold long-poll connections; 60 s lets in-flight activities complete before the pod is killed; server uses 30 s + preStop sleep |
| 2026-04-03 | `autoscaling/v2` HPA over v1 | v2 supports multiple metrics (CPU + memory) and fine-grained scale-up/down behaviour policies; v1 is deprecated in Kubernetes 1.25+ |
| 2026-04-03 | Secret manifest is a template (placeholder values) | Real secrets must never be committed; placeholder base64 values make the file syntactically valid for `kubectl apply` while making it obvious the values must be replaced |
| 2026-04-03 | `Event.Data` as `json.RawMessage` (not `interface{}`) | Preserves payload type through Redis round-trips; `interface{}` decodes as `map[string]interface{}` after JSON unmarshal, losing type info; `json.RawMessage` stays as bytes and can be decoded into the correct struct with `event.Decode(&p)` |
| 2026-04-03 | Tool call cache scoped to one `Execute` invocation, not the executor | Within-call dedup is safe and stateless; cross-call state would require locking and TTL logic; cross-retry dedup is left to tool implementors (idempotency) because the activity may run on a different worker instance |
| 2026-04-03 | Scaffold (`POST /designer/scaffold`) on `Generator`, not a new struct | Reuses the same `llm.Provider` already held by `Generator`; avoids a new dependency in `NewHandlers`; scaffold and generate are complementary LLM-backed designer tools |
| 2026-04-03 | MCP server scaffolding generates standard-library-only Go | Eliminates dependency management for generated servers; `encoding/json` + `net/http` cover the JSON-RPC 2.0 protocol; single `go run .` to start; trivial to Dockerise |
| 2026-04-03 | MCP registry DB promotion with YAML fallback | Teams that manage servers via `mcp_registry.yaml` continue to work; once any DB entry exists for a tenant, the DB takes precedence; YAML is seeded into the DB on first boot for a smooth migration path |
| 2026-04-03 | `MCPServerStore` added to `Store` interface | Consistent access pattern with `UserStore`, `TenantStore`, etc.; memory and Postgres implementations provided; no extra constructor parameter needed |
| 2026-04-04 | DB is authoritative for LLM config after first boot | Env vars seed once via `seedLLMConfigs` when `HasAnyLLMConfig` is false; subsequent changes via Settings UI persist to DB; eliminates config drift between env and in-process state across restarts |
| 2026-04-04 | `"***"` as both display mask and "no change" sentinel | Simplest possible API contract: server never returns a raw key; client that sends `***` (or empty) signals "keep existing"; no separate boolean flag needed |
| 2026-04-04 | Worker polls DB every 15s for LLM config changes | Push (pub/sub) would require a dedicated channel and more complexity; 15s lag is acceptable for provider config; `Reload` is lock-protected and atomic so in-flight executions see a consistent registry snapshot |
| 2026-04-04 | `COALESCE(NULLIF($8,''), temporal_workflow_id)` in `UpdateRun` | `CreateRun` inserts before `StartWorkflow` returns the Temporal ID; subsequent `UpdateRun` calls from Temporal activities pass `TemporalID = ""`; the COALESCE preserves the stored value while still allowing the first real update to persist it |
| 2026-04-04 | `sig.Resolution` used directly as HITL trigger (no internal mapping) | Removing `human_approved`/`human_rejected` indirection makes the contract explicit in the YAML; workflow authors choose their trigger names; direct mode already used this pattern; Temporal mode now consistent |
| 2026-04-04 | HITL resolve errors displayed in orange banner (not swallowed) | `resolve()` previously had no try/catch; silent failures left the UI in an inconsistent state (buttons hidden, workflow stuck); banner makes the problem visible immediately and `loadRun()` on error restores correct state |
| 2026-04-04 | Grok/Deepseek/Gemini implemented as OpenAI-compatible wrappers | All three providers expose OpenAI-compatible `/chat/completions` endpoints; `NewOpenAICompatibleProvider(endpoint, key, name, model)` handles them all; no per-provider request/response mapping code needed |

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

### With Temporal (full development mode)

To run the API server and workers on your host while using Docker for infrastructure:

```bash
# 1. Start only the infrastructure (Postgres, Redis, Temporal)
docker compose up -d postgres redis temporal temporal-ui

# 2. Terminal 1 — API server
EXECUTOR_MODE=temporal go run ./cmd/server

# 3. Terminal 2 — Temporal worker (handles LLM agents)
EXECUTOR_MODE=temporal go run ./cmd/worker
```

> [!IMPORTANT]
> Do NOT run `docker compose up -d` without service names if you intend to run the server on your host, as the Dockerized server will bind to port 8080 first and cause a conflict.

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
# Login with ADMIN_USERNAME / ADMIN_PASSWORD from .env; tenant_slug defaults to "default"
```

### Kubernetes (production)
```bash
# 1. Build and push images to your registry
docker build --build-arg CMD=server           -t registry.example.com/asm/server:latest .
docker build --build-arg CMD=worker           -t registry.example.com/asm/worker:latest .
docker build --build-arg CMD=specialist-worker -t registry.example.com/asm/specialist-worker:latest .
docker push registry.example.com/asm/server:latest
docker push registry.example.com/asm/worker:latest
docker push registry.example.com/asm/specialist-worker:latest

# 2. Edit deploy/k8s/configmap.yaml and deploy/k8s/secret.yaml with real values
#    (or replace secret.yaml with External Secrets Operator resources)

# 3. Update image references in the deployment manifests to match your registry

# 4. Apply all manifests in order
make k8s-apply

# Check rollout status
make k8s-status
kubectl rollout status deployment/asm-server -n asm
kubectl rollout status deployment/asm-worker -n asm
```


### Troubleshooting

#### Go Build: "operation not permitted" on module cache
If you see `go: could not create module cache: ... operation not permitted` during build, your environment may have restricted access to the global `GOMODCACHE`. Use a local cache within the project:

```bash
mkdir -p .cache/go-mod-cache
GOMODCACHE=$(pwd)/.cache/go-mod-cache go build ./cmd/server
```

#### Port 8080: "address already in use"
If the server fails to bind to 8080, check if a Dockerized instance is already running:
```bash
docker compose ps
docker compose stop server
```
