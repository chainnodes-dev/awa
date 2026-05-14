# Phaxa — Architecture & Functionality

## Overview

The **Phaxa** platform is a workflow orchestration system designed for processing business items (invoices, applications, orders, etc.) through a series of well-defined states. Its core philosophy is:

> **Keep workers stupid. Let the state machine decide.**

Each worker's sole responsibility is to complete a task and return a result. It does not know what happens next. The workflow definition owns all routing logic — transitions, conditions, and branching. This clean separation means you can replace, upgrade, or specialise a worker without ever touching the workflow definition, and vice versa.

The platform is powered by **Temporal**, ensuring every workflow execution is durable, resumable, and auditable. Durability is not an afterthought — it is mandatory and universal. Every run is executed as a Temporal workflow, surviving server and worker restarts by default.

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

A **run** is a single item moving through a workflow. When 1,000 invoices arrive, 1,000 runs are created — each independent, each carrying its own blackboard. Runs are durable: they survive server and worker restarts via mandatory Temporal orchestration.

A run has a lifecycle status:
- `running` — actively progressing
- `waiting` — paused at a human-in-the-loop state
- `complete` — reached a terminal state
- `failed` — an unrecoverable error occurred

All execution is performed by Temporal workers. There is no "direct" or non-durable mode.

### Blackboard

The **blackboard** is the shared data envelope that travels with a run through all its states. It is:

- **Initialised** with the input provided when the run starts (e.g. `{invoice_id: "INV-001", amount: 5000}`)
- **Enriched** by each worker as the run progresses (e.g. the validate worker adds `{vendor_id: "V-42"}`)
- **Snapshotted** at every state transition for a complete audit trail
- **Schema-validated** — the workflow definition declares types and required fields

Workers read the blackboard to understand what to do. They write their results back to it. The state machine reads it to evaluate guards and decide which transition to take.

### State Types

Every state in a workflow has a `type` that determines how the engine processes it. There are **nine** types. They form a natural progression from the structural (`initial`, `terminal`) to the operational (`prompt`, `script`, `hitl`, `wait`).

---

#### `initial`

The entry point of every workflow. When a run starts, the engine places it in the `initial` state automatically. The input JSON supplied at run creation seeds the blackboard and is immediately available to the initial state.

**Rules:**
- Every workflow must have **exactly one** `initial` state. The loader rejects definitions with zero or more than one.
- The `initial` state advances automatically: no external trigger is required.
- An agent or script may optionally be assigned for validation or enrichment before advancing.

**Runtime:**
The engine enters the initial state and behaves as follows depending on what is assigned:

| Config | Behaviour |
|---|---|
| No agent, no script | Auto-advances immediately using the first outgoing transition whose guard passes. The run creation itself is the trigger. |
| Script assigned | Evaluates the script expressions against the blackboard, computes the trigger, then advances. |
| Agent assigned | Dispatches the LLM or specialist worker, applies its blackboard updates, then advances using the returned trigger. |

**YAML example:**
```yaml
states:
  - name: INTAKE
    type: initial
    # No agent needed — run creation auto-advances to the next state.
    # Add an agent or script here only for optional validation/enrichment.

  - name: INTAKE_WITH_VALIDATION
    type: initial
    agent: intake-agent      # optional; validates/enriches the input before advancing
    instructions: |
      Validate that invoice_id, vendor, amount, and currency are present and well-formed.
      Fire "valid" if all fields pass, "invalid" otherwise.
```

---

#### `prompt`

A standard processing step. This is the most common type — it represents any state where work needs to be done before the run can progress.

A `prompt` state can be handled in two ways:

1. **With an agent** — the engine dispatches the named agent (LLM or specialist worker). The agent reads the blackboard, performs its task, and returns a trigger + updated blackboard fields.
2. **Without an agent** — the engine parks and waits for an external `POST /api/v1/runs/{id}/trigger` call. Use this for states driven by external systems (e.g. "waiting for document upload").

**Runtime:** Engine dispatches the agent (or blocks on `wakeupCh`). On return, applies blackboard updates, evaluates outgoing guards, and follows the matching transition.

**Available fields:** `agent`, `instructions`, `timeout`, `on_timeout`, `on_enter`

**YAML example:**
```yaml
states:
  - name: VALIDATING
    type: prompt
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

### Deep Observability

The platform provides granular visibility into every execution step. For automated logic (LLM, Script, Code), the engine captures:
- **LLM Monologue** — the internal reasoning used by the agent to reach a decision.
- **JavaScript Stack Traces** — full error traces for `code` nodes, allowing precise debugging of logic errors.
- **MCP Call Logs** — detailed audit of all tool calls made to external Model Context Protocol servers.

### Multi-User Task Management (HITL)

Human-in-the-loop states support advanced task management for collaborative environments:
- **Assignees** — tasks can be assigned to specific users or teams via the `assignee` field.
- **Filtered Inbox** — the Task Inbox allows users to switch between "All Tasks" and "My Tasks" for efficient processing.
- **Form Schemas** — customizable JSON Form Schemas ensure humans provide data in the exact format required by downstream automated steps.

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

#### `wait`

A join or conditional pause state. The run pauses here until a specific condition is met, an external trigger is received, or a timeout occurs.

`wait` states are commonly used as **join points** in parallel workflows, or as "guards" that wait for background processes to finish before continuing.

A `wait` state resolves when **any** of these occur:
1. The `condition` expression evaluates to `true` → fires the `on_condition` trigger (defaults to `"condition_met"` if omitted).
2. An external trigger is received via `POST /api/v1/runs/{id}/trigger` → fires that trigger directly.
3. The `timeout` duration is reached → fires the `on_timeout` trigger (defaults to `"timeout"` if omitted).

**Available fields**

| Field | Required | Description |
|---|---|---|
| `condition` | No | expr-lang boolean expression evaluated against the blackboard |
| `on_condition` | No | Trigger fired when `condition` becomes true. Defaults to `"condition_met"` |
| `timeout` | No | Duration string (e.g. `1h`, `30m`). Defaults to 24 h if omitted |
| `on_timeout` | No | Trigger fired on timeout. Defaults to `"timeout"` |

**YAML example:**
```yaml
states:
  - name: AWAIT_CLEANUP
    type: wait
    condition: 'cleanup_status == "done"'
    on_condition: cleanup_done      # fires this trigger when condition passes
    timeout: 1h
    on_timeout: cleanup_failed

transitions:
  - from: AWAIT_CLEANUP
    to:   NEXT_STEP
    trigger: cleanup_done

  - from: AWAIT_CLEANUP
    to:   HANDLE_FAILURE
    trigger: cleanup_failed
```

**Constraints:**
- `agent` is ignored when type is `wait`.
- If the `condition` is met, the workflow takes the transition matching the trigger that originally brought the run to this state (or its default "continue" path).
- If a `timeout` occurs and no signal was received, the `on_timeout` trigger is fired.

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

#### `code`

A JavaScript state executed directly in the engine using the [goja](https://github.com/dop251/goja) pure-Go JavaScript runtime. No LLM, no external service — pure deterministic logic with full JavaScript expressiveness.

Use `code` states when the logic is too complex for a single-line `script` expression but doesn't warrant a full specialist worker deployment.

The code receives the blackboard as `bb` and must return `{ trigger, blackboard_updates?, reasoning? }` or call `trigger('name')` for early exit.

```yaml
states:
  - name: COMPUTE_TOTALS
    type: code
    code:
      language: javascript
      code: |
        bb.vat        = bb.amount * 0.2;
        bb.net_amount = bb.amount - bb.vat;
        return { trigger: bb.net_amount > 10000 ? "needs_review" : "auto_approve" };
```

The Designer includes an **LLM code generation** button: describe the logic in the Instructions field and click "Generate Code" — the platform sends the description, blackboard schema, and outgoing trigger names to the LLM and populates the editor with working JavaScript.

---

#### `subprocess`

Invokes another registered **process** (reusable workflow) as a Temporal child workflow. The parent run pauses until the child reaches a terminal state, then maps the child's outputs back onto the parent blackboard.

This is the primary composition mechanism — complex workflows are built from smaller, reusable building blocks.

```yaml
states:
  - name: ENRICH_COMPANY
    type: subprocess
    subprocess:
      process_ref: gleif-lookup-workflow    # name of the child workflow
      process_version: latest               # optional specific version
      completion_trigger: enriched
      failure_trigger:    enrichment_failed
      input_mappings:
        company_name: vendor              # child port ← parent bb field
      output_mappings:
        lei_code: vendor_lei              # parent bb field ← child port
        revenue_eur: enriched_revenue
```

**Runtime:**
1. `LoadWorkflowDef` activity resolves the child workflow definition by `process_ref` (and optional `process_version`).
2. `ExecuteChildWorkflow` launches it with `childBB` built from `input_mappings`.
3. Parent parks on the child workflow future.
4. On completion: `GetRunBlackboard` reads the child's terminal blackboard; `output_mappings` writes values back to the parent.
5. Fires `completion_trigger`; on failure fires `failure_trigger`.

The child workflow is a fully durable Temporal workflow — it survives restarts independently of the parent.

---

#### `emit_event`

Fires a named platform event to the event bus. Other runs (in the same tenant) that are paused at a `wait` state listening for the same event name will be woken up and advanced.

Use `emit_event` states for fire-and-forget fan-out: notify a downstream process that data is ready without pausing the current run.

```yaml
states:
  - name: NOTIFY_DOWNSTREAM
    type: emit_event
    emit_event:
      event_name: data_ready          # name other runs listen for
      payload_fields: [result, run_id] # which bb fields to include (omit = full bb)
      completion_trigger: notified    # trigger fired after the event is delivered
```

**Runtime:**
1. `EmitWorkflowEvent` activity publishes a `workflow.event` event to the bus (WebSocket clients see it).
2. Queries `event_subscriptions` for all runs in this tenant waiting for `event_name`.
3. For each match: calls `client.SignalWorkflow` with the subscription's `on_match_trigger` and the payload.
4. Removes each delivered subscription from the store.
5. Fires `completion_trigger` (default: `event_emitted`) and advances.

---

### Process Registry & Unified Model

As of v1.1, the platform uses a **Unified Process Model**. All workflows designated as `reusable: true` in their metadata are automatically available as sub-processes. 

The **Process Catalog** provides:
- Name and description of all reusable logic.
- Explicit version selection for stable sub-process calls.
- Typed I/O port contracts (inputs/outputs) for type-safe composition.

### Bi-Directional Conversion

The platform provides two LLM-powered conversions between the process abstraction and its workflow implementation:

| Direction | Endpoint | What it does |
|---|---|---|
| **Analyse** (description → workflow) | `POST /designer/process/analyse` | Decomposes a plain-English process description into an optimised workflow YAML. Prefers `subprocess` nodes for capabilities that already exist in the registry. |
| **Render** (workflow → description) | `POST /designer/process/render` | Converts an existing workflow YAML back into a human-readable description + inferred I/O port contract. |

### Process Composition

Processes are composed using `subprocess` states. Callers only need to know the port names and types — not how the child is implemented. This enables high-level "Orchestrator" workflows to delegate specialized tasks to "Worker" workflows.

---

## Technical Details

### Choosing the Right Type

| I need to… | Use |
|---|---|
| Start processing when a run is created | `initial` |
| Run an LLM agent or specialist worker | `prompt` |
| Make a deterministic decision without an LLM | `script` |
| Run JavaScript logic (more than one line) | `code` |
| Invoke a reusable sub-process | `subprocess` |
| Fire a named event to wake other runs | `emit_event` |
| Pause and wait for a human decision | `hitl` |
| Wait for a condition or platform event | `wait` |
| Mark the run as finished | `terminal` |

### Transitions & Guards

A transition connects a `from` state to one or more target states and is activated by a named trigger. Guards are optional boolean expressions evaluated against the blackboard.

### Parallel Execution

Phaxa supports parallel execution via the `to_nodes` field in a transition. When a trigger matches a transition with `to_nodes`, the engine splits the run into multiple parallel branches.

---

## Multi-Tenancy

The platform is built for multi-tenant SaaS deployment. Every customer is a **tenant**. Workflow definitions, runs, and users are all scoped to a tenant — one tenant cannot see or affect another's data.

### Roles

| Role | Scope | Permissions |
|---|---|---|
| `super_admin` | Platform-wide | Manage tenants; bypass all role checks. |
| `admin` | Tenant | Full access within tenant: user management, workflow CRUD, run control. |
| `operator` | Tenant | Start runs, trigger transitions, resolve HITL, read workflows. |
| `viewer` | Tenant | Read-only access to runs and status. |

---

## Enterprise Security & Identity

The platform includes advanced security features designed for enterprise-grade compliance and operational visibility.

### OIDC Federation

In addition to local email/password authentication, the platform supports **OpenID Connect (OIDC)** federation. Enterprise tenants can configure external Identity Providers (IdP) such as **Okta**, **Microsoft Entra ID (Azure AD)**, or **Auth0**.

- **Single Sign-On (SSO)**: Users authenticate via their corporate identity.
- **Dynamic Role Mapping**: External groups from the IdP can be mapped to platform roles (`admin`, `operator`, `viewer`) automatically upon login.
- **Tenant-Specific IdPs**: Each tenant manages their own IdP configuration (Issuer, Client ID, Secret) directly in the platform.

### Audit Trails

The **Audit Logging** system provides a permanent, tamper-evident record of all security-sensitive actions performed within a tenant. Every audit log captures:
- **Who**: The user ID or system service that performed the action.
- **What**: The specific action (e.g., `workflow.deleted`, `license.updated`, `auth.oidc_login`).
- **Where**: The IP address and origin of the request.
- **Context**: Detailed metadata about the change (e.g., old vs. new values).

Audit logs are scoped to the tenant and cannot be modified or deleted by tenant administrators.

### Tiered Licensing & Quotas

The platform uses **Signed JWT Licenses** to enforce usage quotas and capability gating at the orchestration layer:

- **Resource Limits**: Enforces strict quotas on the number of unique workflow definitions and the total number of monthly runs.
- **Capability Gating**: Disables advanced features (like `subprocess` nodes or custom MCP servers) for lower-tier licenses.
- **Offline Verification**: All Temporal workers can verify license claims locally using the platform’s public key, ensuring performance and high availability.

---

## Key Design Principles

1. **Stateless Workers.** They receive input, do work, return output.
2. **Workflow-owned Routing.** All branching logic lives in transitions and guards.
3. **Blackboard as Contract.** Universal data envelope for all communications.
4. **Mandatory Durability.** Powered by Temporal; surviving restarts is the default.
5. **Evolutionary Architecture.** Promote logic from LLM to script to specialist worker as it stabilizes.
