# ASM Platform — Architecture & Functionality

## Overview

The **Agentic State Machine (ASM) Platform** is a workflow orchestration system designed for processing business items (invoices, applications, orders, etc.) through a series of well-defined states. Its core philosophy is:

> **Keep workers stupid. Let the state machine decide.**

Each worker's sole responsibility is to complete a task and return a result. It does not know what happens next. The workflow definition owns all routing logic — transitions, conditions, and branching. This clean separation means you can replace, upgrade, or specialise a worker without ever touching the workflow definition, and vice versa.

The platform supports a **hybrid execution model**: start with LLM agents for everything (fast to build, zero domain code), then progressively replace individual states with specialist workers as the business logic stabilises. LLMs remain where judgement is genuinely needed.

---

## Core Concepts

### Workflow Definition

A workflow is a YAML manifest that describes the complete lifecycle of a business item. It contains:

- **States** — the stops an item can be at (e.g. `validate`, `enrich`, `approve`, `done`)
- **Transitions** — the edges between states, each named by a trigger (e.g. `valid`, `invalid`, `approved`)
- **Guards** — optional expressions on transitions that must evaluate to true for that path to be taken (e.g. `amount > 10000`)
- **Agents** — the workers assigned to each state, identified by name
- **Blackboard schema** — the typed data contract for the item as it moves through the workflow

Workflow definitions are versioned. Multiple versions can coexist, allowing gradual migrations.

```
┌─────────────────────────────────────────────────────────────┐
│                    Workflow Definition                       │
│                                                             │
│  validate ──[valid]──► approve ──[approved]──► done         │
│      │                    │                                 │
│      └──[invalid]──► enrich    └──[rejected]──► rejected    │
│                         │                                   │
│                         └──[enriched]──► validate           │
└─────────────────────────────────────────────────────────────┘
```

### Run (Workflow Instance)

A **run** is a single item moving through a workflow. When 1,000 invoices arrive, 1,000 runs are created — each independent, each carrying its own blackboard. Runs are durable: they survive server and worker restarts via Temporal.

A run has a lifecycle status:
- `running` — actively progressing
- `waiting` — paused at a human-in-the-loop state
- `complete` — reached a terminal state
- `failed` — an unrecoverable error occurred

### Blackboard

The **blackboard** is the shared data envelope that travels with a run through all its states. It is:

- **Initialised** with the input provided when the run starts (e.g. `{invoice_id: "INV-001", amount: 5000}`)
- **Enriched** by each worker as the run progresses (e.g. the validate worker adds `{vendor_id: "V-42"}`)
- **Snapshotted** at every state transition for a complete audit trail
- **Schema-validated** — the workflow definition declares types and required fields

Workers read the blackboard to understand what to do. They write their results back to it. The state machine reads it to evaluate guards and decide which transition to take.

### State Types

Every state in a workflow has a `type` that determines how the engine processes it. There are five types. They form a natural progression from the structural (`initial`, `terminal`) to the operational (`intermediate`, `script`, `hitl`).

---

#### `initial`

The entry point of every workflow. When a run starts, the engine places it in the `initial` state automatically.

**Rules:**
- Every workflow must have **exactly one** `initial` state. The loader rejects definitions with zero or more than one.
- The `initial` state may have an agent or script assigned (to kick off processing immediately), or it may wait for an external trigger.

**Runtime:** Engine enters the state, publishes a `state.entered` event, then dispatches the assigned worker — or waits for an external trigger if none is assigned.

**YAML example:**
```yaml
states:
  - name: INTAKE
    type: initial
    agent: intake-agent      # optional; omit to wait for an external trigger
    instructions: |
      Extract the invoice number, vendor, amount, and currency from the raw input.
```

---

#### `intermediate`

A standard processing step. This is the most common type — it represents any state where work needs to be done before the run can progress.

An `intermediate` state can be handled in two ways:

1. **With an agent** — the engine dispatches the named agent (LLM or specialist worker). The agent reads the blackboard, performs its task, and returns a trigger + updated blackboard fields.
2. **Without an agent** — the engine parks and waits for an external `POST /api/v1/runs/{id}/trigger` call. Use this for states driven by external systems (e.g. "waiting for document upload").

**Runtime:** Engine dispatches the agent (or blocks on `wakeupCh`). On return, applies blackboard updates, evaluates outgoing guards, and follows the matching transition.

**Available fields:** `agent`, `instructions`, `timeout`, `on_timeout`, `on_enter`

**YAML example:**
```yaml
states:
  - name: VALIDATING
    type: intermediate
    agent: invoice-validator
    timeout: 5m
    on_timeout: validation_timeout
    instructions: |
      Validate that all required fields are present and the amounts balance.
      Fire "validation_passed" or "validation_failed".

agents:
  - name: invoice-validator
    task_queue: validate-workers   # route to specialist worker queue
```

---

#### `script`

A deterministic, code-free processing step evaluated entirely by the engine using [expr-lang](https://github.com/expr-lang/expr) expressions. No LLM is called, no external service is contacted, and no tokens are consumed.

Use `script` states for transformations and routing decisions that are fully deterministic — for example, threshold checks, field derivations, or format conversions. They execute synchronously in microseconds.

The `script` field contains two sub-fields:
- **`trigger`** — an expression that evaluates to a `string`: the name of the transition to fire. All blackboard fields are available as top-level variables.
- **`updates`** — a map of `fieldName: expression` pairs. Each expression is evaluated against the blackboard and the result is written back.

**Runtime:** Engine evaluates the `trigger` expression, evaluates each `updates` expression, writes the results to the blackboard, and fires the computed trigger — all inline, without calling any executor. Execution is synchronous and isolated; expressions cannot perform network calls or access the filesystem.

**YAML example:**
```yaml
states:
  - name: CLASSIFY_AMOUNT
    type: script
    script:
      trigger: 'amount > 10000 ? "needs_review" : "auto_approve"'
      updates:
        vat:        "amount * 0.2"
        net_amount: "amount - vat"
        tier:       'amount > 50000 ? "large" : amount > 10000 ? "medium" : "small"'
```

**Constraints:**
- `script.trigger` is required and must evaluate to a non-empty string matching an outgoing transition trigger.
- `script.updates` is optional. Evaluation order across fields is not guaranteed.
- Expressions run in a sandboxed evaluator — no I/O, no side effects.
- `agent` is ignored when type is `script`.

---

#### `hitl`

Human-in-the-loop. The run pauses at this state and waits for a human to make a decision before it can continue.

When a run enters a `hitl` state:
1. The run's status changes to `waiting`.
2. A HITL request is recorded in the database with the current blackboard snapshot.
3. The engine blocks on a signal channel — no CPU is consumed while waiting.
4. When a human resolves the request via `POST /api/v1/runs/{id}/signal`, the engine wakes up, records the resolution, and fires the provided trigger.

HITL states are designed to integrate with notification systems (email, Slack, ticketing) through the event bus — the `hitl.created` event carries all the context a notification service needs.

**YAML example:**
```yaml
states:
  - name: HUMAN_REVIEW
    type: hitl
    assignee: finance-team         # optional; carried in the hitl.created event
    timeout: 48h
    on_timeout: escalate

transitions:
  - from: HUMAN_REVIEW
    to:   APPROVED
    trigger: approve

  - from: HUMAN_REVIEW
    to:   REJECTED
    trigger: reject
```

**API call to resolve:**
```http
POST /api/v1/runs/{id}/signal
{ "resolution": "approved", "resolver": "alice@acme.com" }
```

**Constraints:**
- `agent` is ignored when type is `hitl`.
- The `resolution` value is used **directly as the trigger name** — the workflow YAML must declare outgoing transitions whose `trigger` fields match the expected resolution values (e.g. `approved`, `rejected`).
- The resolver identity is stored in the transition history for audit purposes.
- Multiple outgoing transitions are supported (approved / rejected / escalate / etc.).

---

#### `terminal`

The end of a run. When the engine enters a `terminal` state it marks the run as `complete` and shuts down its execution context. No further transitions are possible.

A workflow can have **multiple** terminal states, each representing a different outcome (e.g. `APPROVED`, `REJECTED`, `CANCELLED`). The run's final state name is queryable and appears in the monitor UI.

**Runtime:** Engine calls `completeRun()`, which sets `run.Status = complete` and `run.CompletedAt`, publishes a `run.completed` event, and removes the run from the in-memory context. In Temporal mode, the workflow function returns normally.

**YAML example:**
```yaml
states:
  - name: APPROVED
    type: terminal

  - name: REJECTED
    type: terminal

  - name: CANCELLED
    type: terminal
```

**Rules:**
- Every workflow must have **at least one** `terminal` state.
- No outgoing transitions can be defined from a `terminal` state.
- `agent` and `script` fields are ignored when type is `terminal`.

---

### Choosing the Right Type

| I need to… | Use |
|---|---|
| Start processing when a run is created | `initial` |
| Run an LLM agent or specialist worker | `intermediate` |
| Make a deterministic decision without an LLM | `script` |
| Pause and wait for a human decision | `hitl` |
| Mark the run as finished | `terminal` |

A typical workflow follows this spine: `initial → intermediate(s) → hitl (optional) → terminal`

### Transitions & Guards

A transition connects a `from` state to a `to` state and is activated by a named trigger. Guards are optional boolean expressions evaluated against the blackboard:

```yaml
transitions:
  - from: approve
    to: done
    trigger: decision_made
    guard: "amount <= 50000"

  - from: approve
    to: escalate
    trigger: decision_made
    guard: "amount > 50000"
```

The worker fires the trigger `decision_made`. The state machine evaluates both guards against the current blackboard and takes the matching path. The worker had no knowledge of this branching.

---

## Architecture

### System Components

```
┌──────────────────────────────────────────────────────────────────────┐
│                          ASM Platform                                │
│                                                                      │
│  ┌─────────────┐    ┌──────────────┐    ┌────────────────────────┐  │
│  │  REST API   │    │   Engine     │    │   Temporal Server      │  │
│  │  (Gin)      │───►│ (State       │───►│   (Workflow history,   │  │
│  │             │    │  Machine)    │    │    durability, retry)  │  │
│  └─────────────┘    └──────────────┘    └────────────────────────┘  │
│         │                  │                        │               │
│  ┌──────▼──────┐    ┌──────▼──────┐    ┌───────────▼────────────┐  │
│  │  WebSocket  │    │  PostgreSQL │    │   Worker Pool          │  │
│  │  Hub        │    │  Store      │    │   (Activities)         │  │
│  │  (events)   │    │             │    │                        │  │
│  └─────────────┘    └─────────────┘    └────────────────────────┘  │
│         │                                           │               │
│  ┌──────▼──────┐                       ┌───────────▼────────────┐  │
│  │  Redis      │                       │   Workers              │  │
│  │  Event Bus  │                       │   (LLM / Specialist)   │  │
│  └─────────────┘                       └────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
```

### The API Server

The server exposes a REST API and a WebSocket endpoint. Its responsibilities are:

- Accepting workflow definitions (YAML upload)
- Starting runs with initial input
- Sending external triggers to active runs (e.g. "this document is now complete")
- Resolving HITL decisions
- Streaming real-time state change events to connected clients via WebSocket

The server does **not** execute any business logic. It delegates everything to the engine.

### The Engine

The engine is the state machine runtime. In **direct mode** (development/single-process) it drives runs in-process via goroutines. In **Temporal mode** (production) it delegates to Temporal workflows.

Responsibilities:
- Validating and starting runs
- Routing external triggers to the correct run
- Passing HITL resolutions to waiting runs
- Loading workflow definitions from the store

The engine intentionally has no knowledge of what a worker does. It only knows: current state → activity completes → trigger returned → evaluate guards → next state.

### Temporal (Durability Layer)

Each run becomes a **Temporal workflow** — a durable, resumable execution that survives any process crash. Temporal journals every step in its event history. If the server or a worker dies mid-execution, Temporal resumes exactly where it left off when the process restarts.

The ASM workflow in Temporal maps directly to the state machine loop:
- Each agent execution is a **Temporal activity** (retriable, with configurable timeouts)
- HITL states use **Temporal signals** to park and resume
- External triggers (API calls) also arrive as signals

### Workers

Workers are the execution units that perform the actual work for each state. A worker:

1. Picks up an activity from its task queue
2. Reads the blackboard to understand the item
3. Performs its task (LLM call, API call, algorithm, database query, etc.)
4. Returns a result: updated blackboard fields + a trigger name

That's it. The worker does not know which state comes next. It does not know what other workers exist. It fires a trigger and returns.

**Worker / execution types:**

| Type | Description | When to use |
|---|---|---|
| Script state | expr-lang expressions evaluated inline by the engine; no worker | Threshold checks, field derivations, deterministic routing |
| LLM Agent | Calls Claude, GPT, or a local Ollama model. Can use MCP tools. | Ambiguous decisions, natural language, unstructured data |
| MCP-enabled Agent | LLM with access to MCP servers (validation APIs, ERP, etc.) | Structured operations with LLM decision-making on top |
| Specialist Worker | Hand-coded Go logic; API integrations, database queries | Well-defined, stable business rules that no longer need an LLM |

### Task Queues

Workers listen on named **task queues**. By default, all workers share one queue (`asm-workers`). As the system matures, you can introduce specialist queues:

```
asm-workers          ← general / LLM agents
validate-workers     ← specialist validation workers
enrich-workers       ← ERP integration workers
gpu-workers          ← compute-heavy ML workers
```

The workflow definition (or agent config) specifies which queue a given state's activity should be scheduled on. Only workers registered on that queue will process it.

### Event Bus

All state changes, blackboard updates, HITL events, and agent activity are published to the **event bus** (Redis Pub/Sub in production, in-memory in development). The WebSocket hub subscribes to this bus and fans events out to all connected browser clients in real time.

This powers the live monitoring UI — you can watch 1,000 invoices move through states in real time.

### PostgreSQL Store

All durable state lives in PostgreSQL:

- `tenants` — one row per customer/tenant (id, name, slug, created_at)
- `workflow_definitions` — versioned workflow YAML + parsed definition, scoped to a tenant
- `workflow_runs` — one row per run, current state + blackboard + status, scoped to a tenant
- `state_transitions` — append-only history of every transition with blackboard snapshot
- `hitl_requests` — pending and resolved human review requests
- `users` — user accounts scoped to a tenant; username unique per-tenant
- `refresh_tokens` — opaque tokens for JWT refresh rotation

The transition history is immutable. Every run has a complete, auditable record of every decision made, which agent made it, and what the blackboard contained at that moment.

---

## The Hybrid Execution Model

### Stage 1 — LLM Only

Every state uses an LLM agent. Zero domain code required. The LLM reads the blackboard, reasons about the item, writes results, and fires a trigger. Fast to bootstrap, suitable for exploration and low-volume workflows.

```
validate state → LLM agent reads invoice → decides valid/invalid → fires trigger
```

### Stage 2 — LLM + MCP Tools

LLM agents are given access to MCP servers — external tools they can call during reasoning. The LLM remains the decision-maker but now has access to real data:

```
validate state → LLM agent calls validate_schema() MCP tool
               → tool returns {valid: false, missing: ["vendor_id"]}
               → LLM decides to fire "needs_enrichment" trigger
```

MCP tools available could include:
- Schema validators
- Duplicate detection
- ERP / vendor lookup
- Tax rate APIs
- Internal databases

### Stage 3 — Script States

For purely deterministic logic (threshold checks, field derivations, routing decisions), use a `script` state. No worker, no LLM, no external call — the engine evaluates [expr-lang](https://github.com/expr-lang/expr) expressions inline in microseconds.

```yaml
- name: ROUTE_BY_AMOUNT
  type: script
  script:
    trigger: 'amount > 10000 ? "needs_review" : "auto_approve"'
    updates:
      tier: 'amount > 50000 ? "large" : "small"'
```

### Stage 4 — Specialist Workers

Once a path is fully understood but too complex for a one-line expression, replace the LLM with a specialist worker that implements the logic directly in Go. The workflow definition does not change — only the worker assigned to that state changes.

```
validate state → Specialist worker runs schema check + duplicate detection
               → Returns result + fires trigger
               → 10x faster, 100x cheaper than LLM
```

LLMs remain for states where genuine judgement is needed — edge cases, natural language, novel situations. Script states and specialist workers handle everything that can be codified.

---

## Data Flow: Single Invoice

```
1.  POST /api/v1/runs  {workflow: "invoice", input: {id: "INV-001", amount: 5000}}

2.  Engine creates run, initialises blackboard
    Blackboard: {invoice_id: "INV-001", amount: 5000}

3.  Temporal workflow starts, current state = "validate"
    → Schedules ExecuteAgent activity on "asm-workers" queue

4.  Validate worker picks up activity
    → Calls validate_schema MCP tool → {valid: true}
    → Returns: {blackboard_updates: {validated: true}, trigger: "valid"}

5.  State machine evaluates transitions from "validate" on trigger "valid"
    → Transition to "approve" (guard: amount > 1000 = true)
    → Blackboard updated, transition recorded

6.  Current state = "approve", type = hitl
    → Run status set to "waiting"
    → HITL request created, assignee notified

7.  POST /api/v1/runs/{id}/signal  {resolution: "approved", resolver: "alice"}

8.  Engine fires the resolution value ("approved") directly as the trigger
    → State machine transitions to "done"
    → Run status = complete

9.  Full transition history stored in DB with blackboard snapshots at each step
```

---

---

## Multi-Tenancy

The platform is built for multi-tenant SaaS deployment. Every customer is a **tenant**. Workflow definitions, runs, and users are all scoped to a tenant — one tenant cannot see or affect another's data.

### Tenant Isolation Model

```
tenants
  └── users          (login scoped to tenant slug)
  └── workflow_definitions  (YAML + parsed def, per-tenant uniqueness)
  └── workflow_runs         (each run belongs to one tenant)
        └── state_transitions  (accessed via run_id; isolation inherited)
        └── hitl_requests      (accessed via run_id; isolation inherited)
```

`state_transitions` and `hitl_requests` are not tenant-scoped directly — they are always accessed via a `run_id` FK, so tenant isolation is inherited through the owning run.

### Context-Based Propagation

Tenant identity flows through the system without any method signature changes:

```
POST /api/v1/auth/login  { "username": "alice", "password": "...", "tenant_slug": "acme-corp" }
  → Resolve tenant by slug
  → Look up user within that tenant
  → Issue JWT with { "tid": "<tenant_id>", "uid": "<user_id>", "role": "admin" }

Every subsequent API call:
  RequireAuth middleware
    → validates JWT
    → calls store.WithTenantID(ctx, claims.TenantID)
    → store methods read tenant from context via TenantIDFromContext(ctx)
```

### Roles

| Role | Scope | Permissions |
|---|---|---|
| `super_admin` | Platform-wide | Manage tenants; bypass all role checks; reserved for platform operators |
| `admin` | Tenant | Full access within tenant: user management, workflow CRUD, run control |
| `operator` | Tenant | Start runs, trigger transitions, resolve HITL, read workflows |
| `viewer` | Tenant | Read-only: runs, workflows, HITL pending list |

`super_admin` is created at first boot and cannot be assigned via the user management API. Tenant admins can create `admin`, `operator`, and `viewer` accounts for their team.

### Tenant Management API

```
POST   /api/v1/tenants          — create tenant (super_admin only)
GET    /api/v1/tenants          — list all tenants (super_admin only)
GET    /api/v1/tenants/:id      — get tenant (super_admin only)
DELETE /api/v1/tenants/:id      — delete tenant (super_admin only; default tenant protected)
```

### First-Boot Seeding

On startup, if no tenants exist, the server creates the **default tenant** (`slug: "default"`, fixed UUID `00000000-0000-0000-0000-000000000001`) and seeds the initial `super_admin` user from the `ADMIN_USERNAME` / `ADMIN_PASSWORD` environment variables. This is idempotent — safe to run on every restart.

---

## Key Design Principles

1. **Workers are stateless and stupid.** They receive input, do work, return output. No routing logic, no knowledge of workflow structure.

2. **The workflow definition is the single source of truth for routing.** All branching logic lives in transitions and guards, not in worker code.

3. **The blackboard is the contract.** Workers communicate through it. Every worker reads from it, every worker writes to it. The schema defines what is expected.

4. **Durability is not optional.** Every run is durable. A crashed server or worker does not lose work. Temporal replays from its event history.

5. **Observable by default.** Every transition, every blackboard update, every HITL event is published in real time. The full history is always queryable.

6. **Evolutionary.** Start with LLM agents, add MCP tools for structure, promote to script states or specialist workers when logic is stable. No rewrites — only replacements at the state level.

---

## LLM Provider Support

The platform abstracts LLM providers behind a **DB-backed, hot-reloadable** registry. Providers are configured via the Settings UI (or seeded from env vars on first boot). Any combination can be enabled; the active default is stored in the `llm_configs` table.

| Provider | Notes |
|---|---|
| Anthropic (Claude) | Cloud; best reasoning quality |
| OpenAI (GPT) | Cloud; wide model selection |
| Ollama | Local / self-hosted; no API key — uses `base_url` field |
| Grok (xAI) | Cloud; OpenAI-compatible API at `api.x.ai`; default model `grok-3` |
| Deepseek | Cloud; OpenAI-compatible API; default model `deepseek-chat` |
| Gemini (Google AI Studio) | Cloud; OpenAI-compatible endpoint; default model `gemini-2.0-flash` |

### Provider resolution order (per agent execution)
1. Per-agent `config.provider` override (set in workflow YAML or Designer)
2. Global default provider (configured in Settings UI, stored in `llm_configs`)
3. Ollama fallback (if a configured provider is not available)
4. Hard error with a descriptive message

### DB-backed configuration
- `llm_configs` table: one row per `(tenant_id, provider)` pair
- Env vars (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `OLLAMA_URL`, `OLLAMA_MODEL`, etc.) are read **once at first boot** and inserted if the table is empty; from that point the DB is authoritative
- API keys are masked as `"***"` in list responses; sending `"***"` or empty string on upsert preserves the existing key
- The **server** hot-reloads the registry on every Settings save
- The **worker** polls the DB every 15 seconds and calls `Registry.Reload()` atomically

### Settings UI
Route `/settings` (admin only). Per-provider cards with enable toggle, API key input, model override, base URL (Ollama), and a Test/Save button. A global default provider dropdown applies to all agent states that do not declare a per-agent provider override.
