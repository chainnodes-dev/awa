# Phaxa — User Manual

> **Version:** post-M17 · **Last updated:** May 2026

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Architecture Overview](#2-architecture-overview)
3. [Core Concepts](#3-core-concepts)
4. [The Designer](#4-the-designer)
   - [Creating a Workflow](#41-creating-a-workflow)
   - [State Types](#42-state-types)
   - [Transitions and Guards](#43-transitions-and-guards)
   - [Blackboard Schema](#44-blackboard-schema)
   - [Agent Configuration](#45-agent-configuration)
   - [MCP Servers](#46-mcp-servers)
   - [HITL Form Schemas & Preview](#47-hitl-form-schemas--preview)
   - [Sentinel (Perpetual) Workflows](#48-sentinel-perpetual-workflows)
5. [AI-Assisted Authoring](#5-ai-assisted-authoring)
   - [AI Workflow Generator](#51-ai-workflow-generator)
   - [Code Node Generation](#52-code-node-generation)
   - [Converting an Agent to a Script Node](#53-converting-an-agent-to-a-script-node)
6. [Running Workflows](#6-running-workflows)
7. [Monitoring and Debugging](#7-monitoring-and-debugging)
   - [Event Log](#71-event-log)
   - [Blackboard Inspector](#72-blackboard-inspector)
   - [LLM Debug Panel](#73-llm-debug-panel)
   - [Deep Observability (Stack Traces)](#74-deep-observability-stack-traces)
   - [Temporal UI](#75-temporal-ui)
8. [State Type Reference](#8-state-type-reference)
   - [Script Node](#81-script-node)
   - [Code Node](#82-code-node)
   - [HITL State](#83-hitl-state)
9. [Workflow YAML Reference](#9-workflow-yaml-reference)
10. [Execution Engines](#10-execution-engines)

---

## 1. Introduction

The **Phaxa** platform is a workflow orchestration system for building AI-powered business processes. You describe a process as a directed graph of states and transitions in a YAML definition. The platform runs it durably, connecting LLM agents, deterministic scripts, JavaScript code, and human approvals in a single coherent flow.

**Core philosophy:**

> *Keep workers as simple as possible. Let the state machine decide.*

Each worker's only job is to complete its task and return a trigger. It never knows what happens next. All routing — transitions, conditions, branching — lives in the workflow definition. This means you can swap, upgrade, or specialise a worker without touching the workflow, and redesign the flow without touching the workers.

**Key properties:**

- Every run is **durable** — survives server and worker restarts via Temporal
- Every routing decision is **auditable** — full event log per run
- LLM agents, code, scripts, and human approvals are **first-class** state types
- Workflows are **versioned** — multiple versions coexist; runs pin to the version they started on

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        Browser                              │
│         Designer (canvas)   ·   Monitor (live events)       │
└────────────────────┬────────────────────────────────────────┘
                     │  REST /api/v1  +  WebSocket /ws
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                     Go API Server                           │
│  Workflow CRUD · Run lifecycle · HITL signals               │
│  AI Generator · Code Scaffolder · WebSocket hub             │
└────────────────────┬────────────────────────────────────────┘
                     │  Temporal SDK (gRPC)
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                   Temporal Server                           │
│          Durable workflow engine — all state is             │
│          persisted; replays guarantee correctness           │
└────────────────────┬────────────────────────────────────────┘
                     │  Task Queue  (phaxa-workers)
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                  Temporal Worker                            │
│                                                             │
│  PhaxaWorkflow (state machine loop)                           │
│  ├── ExecuteAgent   — LLM call via configured provider      │
│  ├── ExecuteCode    — JavaScript sandbox (goja)             │
│  ├── ExecuteScript  — Deterministic expressions (expr-lang) │
│  ├── CreateHITL     — Human-in-the-loop pause               │
│  └── WaitForCondition — Event-driven resume                 │
└─────────────────────────────────────────────────────────────┘

Shared infrastructure
  Postgres  — workflow definitions, run state, blackboard
  Redis     — pub/sub event bus (worker → server → WebSocket)
```

### Data flow for a single state execution

1. The Temporal worker picks up the workflow task from the queue
2. It reads the current state from the blackboard and evaluates any guards
3. It dispatches to the appropriate activity (agent, code, script, HITL, wait)
4. The activity publishes events to Redis as it runs (prompt sent, response received, tool calls, etc.)
5. The server's event bus forwards those events over WebSocket to every connected browser
6. The activity returns an `AgentOutput` — a trigger name plus optional blackboard updates
7. The workflow loop applies the updates, evaluates outgoing transition guards, advances to the next state

---

## 3. Core Concepts

### Workflow Definition

A workflow is a YAML manifest (`apiVersion: chainnodes/v1, kind: Workflow`). It declares:

- **States** — the nodes of the graph
- **Transitions** — directed edges labelled by trigger names
- **Guards** — optional `expr-lang` expressions on a transition that must be true for that edge to be taken
- **Agents** — LLM workers assigned to states
- **Blackboard schema** — the typed data envelope that travels with the run

### Blackboard

The **blackboard** is the shared memory of a run. It starts with the input data supplied when the run is created and accumulates writes from every state. It is:

- **Typed** — each field has a declared type (`string`, `number`, `bool`, `object`)
- **Persisted** — stored in Postgres after every state transition
- **Accessible everywhere** — available to LLM agents as context, to script/code nodes as variables, and to guards as named variables

### Runs

A **run** is one instance of a workflow executing against one item (invoice, application, order, …). Multiple runs of the same workflow execute independently. Each run has its own blackboard, event log, and current state.

### Triggers

When a state finishes, it fires a **trigger** — a string name. The workflow engine looks for an outgoing transition whose label matches and whose guard (if any) evaluates to true. The first matching transition wins and the run advances to the target state.

---

## 4. The Designer

Open the Designer from the main navigation. Select an existing workflow or click **New Workflow** to start from scratch.

### 4.1 Creating a Workflow

**Start from scratch**

1. Click **New Workflow** in the Workflows list
2. Enter a name (e.g. `invoice-approval`) and an optional description in the left sidebar
3. The canvas opens with an empty grid
4. Double-click anywhere on the canvas (or use the **Add State** button) to add your first state
5. Click **Save** (or `Cmd+S`) when done — the workflow is versioned automatically

**Import from YAML**

Click the **YAML** tab in the top toolbar. Paste a workflow definition and click **Import**. The canvas will render it immediately.

**AI-generated** — see [Section 5.1](#51-ai-workflow-generator).

---

### 4.2 State Types

Click any state node to open the **State Panel** on the right. The **Type** dropdown controls what the state does:

| Type | Icon | Purpose |
|---|---|---|
| `initial` | Green dot | Entry point — exactly one per workflow |
| `terminal` | Grey dot | End point — run completes here |
| `prompt` | Indigo dot | LLM agent reasons and fires a trigger |
| `code` | Teal `</>` | User-written JavaScript runs in a sandbox |
| `script` | Amber dot | Deterministic expr-lang expressions; zero latency |
| `hitl` | Amber person | Pauses and waits for a human signal |
| `wait` | Indigo clock | Pauses until a blackboard condition becomes true |

**Naming convention:** state names are uppercase by convention (`VALIDATE`, `APPROVE`, `DONE`) though any string works.

---

### 4.3 Transitions and Guards

**Drawing a transition:**
Hover over a state node — a **+** handle appears on the bottom edge. Drag from it to another state. A dialog prompts for a trigger name (e.g. `approved`).

**Editing a transition:**
Click the edge label on the canvas to open the **Edge Panel**. You can change the trigger name and add a guard expression.

**Guards:**
A guard is an `expr-lang` expression evaluated against the current blackboard. The transition is only taken if the guard returns `true`. Example:

```
amount > 10000
```

If multiple outgoing transitions have the same trigger, the first one whose guard passes wins. Transitions with no guard always pass.

**Shorthand Transitions (YAML only)**
For faster iteration when editing YAML directly, you can define transitions using the `to_nodes` or `to` shorthand inside a state definition. These are automatically converted to formal transitions when the workflow is loaded:

```yaml
- name: market_check
  to_nodes: ["decide_action"]
```

---

### 4.4 Blackboard Schema

Click the **Schema** button (database icon) in the top toolbar to open the schema editor. Add fields that describe your item's data envelope:

| Field | Type | Required | Default |
|---|---|---|---|
| `amount` | `number` | yes | — |
| `vendor` | `string` | no | `""` |
| `approved` | `bool` | no | `false` |

The schema is used for:
- **Validation** — input data is checked against it when a run starts
- **Autocomplete** — the code editor offers `bb.fieldName` completions
- **AI context** — the generator and scaffolder are told the field names and types

---

### 4.5 Agent Configuration

When a state's type is `prompt` (or `initial`), expand the **Agent Config** section in the State Panel:

| Field | Purpose |
|---|---|
| **Agent** | The agent name — must match an entry in the workflow's `agents` list |
| **Provider** | Override the global LLM provider for this agent (`anthropic`, `openai`, `ollama`, …) |
| **Model** | Override the model (e.g. `claude-opus-4-5`, `gpt-4o`, `llama3.2`) |
| **Task Queue** | Route to a specialist Temporal worker instead of the default LLM worker |
| **MCP Servers** | Tool servers this agent can call — see [Section 4.6](#46-mcp-servers) |
| **Instructions** | Plain-language description of what this state should do |

**System prompt overrides:**
Global system prompts for the AI Designer (Base, Skill Analyser, and Refinement) can be overridden in **Settings** (admin only). Individual state **Instructions** are appended to the relevant system prompt during LLM calls.

---

### 4.6 MCP Servers

MCP (Model Context Protocol) servers expose tools that LLM agents can call during their reasoning loop. Examples: a database lookup, a currency conversion API, a document parser.

**Assigning MCP servers to a state:**

1. Open the state and expand **Agent Config**
2. In the **MCP Servers** row, click any server chip to select it (click again to deselect)
3. Multiple servers can be selected — the agent will have access to all their tools
4. The resolved value stored in the workflow (shown in small text below the chips) uses the `{{ env.ENV_VAR }}` template syntax, e.g. `{{ env.MCP_GLEIF_URL }}`. This means the actual URL is supplied via the environment and never hardcoded in the workflow definition.

**Local (STDIO) MCP Servers:**
Phaxa supports local tool servers running via `npx`, `docker`, or direct binaries. 
1. Ensure the API server is started with `PHAXA_ALLOW_STDIO_MCP=true`.
2. In the MCP Management screen, create a new server with **Transport: stdio**.
3. Set the **Command** (e.g., `npx`) and **Arguments** (e.g., `-y @modelcontextprotocol/server-postgres`).
4. Local servers are managed automatically—they are started when an agent needs them and shut down when the run finishes.

---

### 4.7 HITL Form Schemas & Preview

When using a **Human-in-the-Loop (HITL)** state, you can define a custom data entry form that the operator will use to resolve the task. This is powered by **JSON Schema**.

**Editing the Schema:**
1. Select an HITL state in the Designer.
2. Find the **HITL Form Schema** section in the right panel.
3. Use the **Insert Template** button to get started with a valid boilerplate.
4. The editor provides **real-time JSON validation**. If you have a syntax error, a message will appear immediately below the editor.

**Live Preview:**
Click the **Preview Form** button to open an interactive modal. This shows exactly how the form will appear to the end-user in the Monitoring view or Task Inbox. You can test inputs, dropdowns, and toggles without saving or running the workflow.

**Persistence:**
Once you click **Apply** in the state panel and **Save** the workflow, the schema is stored within the workflow definition and will be used for every run that hits that state.

---

### 4.8 Sentinel (Perpetual) Workflows

Phaxa workflows are typically short-lived, but you can build **Sentinels**—perpetual agents that monitor data and react indefinitely.

**The "Pulse" Pattern:**
1. Create a `wait` state (e.g., `IDLE_PULSE`) at the end of your workflow logic.
2. Set the `wait_time` (e.g., `7200` for 2 hours).
3. Draw a transition from this wait state back to your initial logic state (e.g., `MARKET_CHECK`).
4. Start the workflow run once—it will loop forever, sleeping for the specified duration between cycles.

**Benefits:**
- **Memory Persistence:** The Blackboard is NOT reset between pulses. The agent can remember previous values to calculate trends.
- **Low Overhead:** The workflow consumes zero resources while in the `wait` state (it is persisted to the database and woken up by Temporal).

---

## 5. AI-Assisted Authoring

### 5.1 AI Workflow Generator

The AI generator creates a complete workflow from a natural-language description.

1. Open the Designer for a new or existing workflow
2. Click the **Generate** secondary tab in the bottom panel (replaces the old Skill tab).
3. The generator uses a two-stage interface:
   - **Description**: Add the high-level process description here. Changing this will trigger a fresh workflow generation.
   - **Prompt**: Once a workflow is generated, use this field to request specific adaptations (e.g., "add a timeout to the validation state" or "change the trigger name to 'verified'").
4. Select the desired **LLM Provider** and **Model** from the dropdowns next to the Generate button.
5. Click **Generate** (or **Restart** to start a fresh process).
6. The AI produces a workflow YAML. Review it in the **YAML** sub-tab.
7. Click **Apply to Canvas** to update the designer.

The generator is aware of the registered MCP servers and will suggest relevant tools. It also automatically generates a concise professional abstract for the workflow in `metadata.description` (if empty).

**Persistence**: Text in the Description and Prompt fields is preserved when switching between Overview and Generate modes, ensuring no work is lost.

**Tips:**
- Be specific about routing conditions ("if amount > 10000") — the AI will turn them into transition guards
- Mention data fields you expect ("the invoice has vendor, amount, currency") — they appear in the blackboard schema
- You can regenerate multiple times and pick the best result

---

### 5.2 Code Node Generation

When a state is of type `code`, the AI can write the JavaScript for you.

1. Select the state and set its type to `code`
2. Fill in the **Instructions** field with a description of what the code should do, e.g.:

   > *Convert amount to EUR using bb.exchange_rate. If converted amount exceeds 5000 trigger 'high_value', otherwise trigger 'standard'.*

3. Click **⚡ Generate Code** (appears below the Instructions field)
4. The code appears in the editor immediately — review it and make any edits
5. Click **Apply** to save the state

If the editor already contains code, clicking **Generate Code** sends it to the LLM as the "existing code" — the result is a refinement rather than a blank-slate generation. This means you can iterate: write the basic logic yourself, then ask the AI to add error handling, or vice versa.

The generator is given:
- Your instructions
- The valid outgoing trigger names (so it uses the correct trigger strings)
- The blackboard schema (so it uses the correct `bb.fieldName` references)
- The existing code if present

---

### 5.3 Converting an Agent to a Script Node

Once an LLM agent is working correctly you may want to replace it with a deterministic, zero-latency script for performance and cost reasons. The **Convert to Script** feature automates this.

1. Select a prompt state that has an agent assigned
2. Scroll to the **Convert to Script** section at the bottom of the Agent Config panel
3. Click **Generate Script**
4. The AI analyses the state's instructions and transitions and produces:
   - A **trigger expression** — an `expr-lang` expression that evaluates to one of the outgoing trigger names
   - **Blackboard updates** — `expr-lang` expressions for any fields the agent was expected to set
5. Review the preview (trigger expression + updates table)
6. Click **Apply to State** — the state type changes to `script` and the agent is removed

This is a one-way conversion in the UI but the original YAML version still exists — use the version picker to go back if needed.

---

## 6. Running Workflows

### Starting a run from the UI

1. Navigate to the workflow in the **Workflows** list
2. Click **▶ Run** (or open the workflow and click the Run button in the toolbar)
3. The **Start Run** modal appears — provide the initial blackboard data as JSON:

   ```json
   {
     "invoice_id": "INV-2024-001",
     "vendor": "Acme Ltd",
     "amount": 12500,
     "currency": "EUR"
   }
   ```

4. Click **Start** — the run is created and you are taken to the Monitor view

### Starting a run via the API

```http
POST /api/v1/runs
Authorization: Bearer eyJ...
Content-Type: application/json

{
  "workflow_name": "invoice-approval",
  "workflow_version": "1.0.0",
  "input": {
    "invoice_id": "INV-2024-001",
    "vendor": "Acme Ltd",
    "amount": 12500
  }
}
```

Response includes the `run_id` for subsequent status queries.

### Sending a HITL signal

When a run is paused at a `hitl` state, it waits for a human decision:

```http
POST /api/v1/runs/{run_id}/signal
Authorization: Bearer eyJ...
Content-Type: application/json

{
  "trigger": "approved",
  "data": { "reviewer": "alice", "comment": "Looks good" }
}
```

Or click the **Signal** button in the Monitor view and choose the trigger from the dropdown.

### Workflow schedules

A workflow can run automatically on a cron schedule. Set the `schedule` field in the workflow metadata:

```yaml
metadata:
  name: daily-reconciliation
  schedule: "0 0 9 * * *"   # daily at 09:00
```

Uses the 6-field robfig/cron format: `seconds minutes hours day-of-month month day-of-week`.

---

## 7. Monitoring and Debugging

Open the **Monitor** view by clicking a run in the Runs list, or clicking **Monitor** after starting a run.

The **Runs list** (Dashboard) provides high-level observability through visual status indicators:
- **Pulsing Blue Dot**: The run is actively executing (`running`).
- **Pulsing Amber Dot**: The run is paused and waiting for human input or an event (`waiting`).
- **Green Dot**: The run completed successfully (`complete`).
- **Red Dot**: The run failed (`failed`).

The Monitor has four main areas: **Timeline**, **Blackboard**, **Trace**, and **Debug**.

---

### 7.1 Event Log

The **Timeline** tab shows a real-time stream of events as the run executes. Each event has a colour-coded type:

| Colour | Event types |
|---|---|
| Green | `run.started`, `run.completed` |
| Indigo | `state.entered`, `state.changed` |
| Violet | `agent.prompt`, `agent.response` |
| Cyan | `agent.tool_call` |
| Amber | `hitl.created`, `hitl.resolved` |
| Red | `run.failed`, errors |

Click any event to expand it and see the full payload.

**`agent.response` events** include a `reasoning` summary — a one-line explanation of why the agent fired the trigger it did. This is shown inline in the event row without expanding.

---

### 7.2 Blackboard Inspector

The **Blackboard** tab shows the current state of all blackboard fields as a live key-value table. Fields highlighted in amber have changed since the previous state transition.

When a state is currently executing, the panel shows a pulsing indicator and updates in real time as writes arrive.

---

### 7.3 LLM Debug Panel

The **Debug** tab is visible to **admin** users only.

It shows a card for every LLM call made during the run, in reverse chronological order (newest first). Each card contains:

- **State and agent** — which state and which agent made the call
- **Trigger fired** — the transition name chosen by the model, shown as a badge
- **Pending indicator** — an amber pulsing dot while the call is in flight; turns green on completion
- Three collapsible sections:
  - **System prompt** — the full system prompt sent to the model
  - **Messages** — the full conversation thread (user messages in indigo, assistant responses in emerald)
  - **Response** — the raw model output, plus the `reasoning` field if the model provided one

This panel is the primary tool for diagnosing unexpected agent behaviour — you can see the exact prompt that was sent and compare it to what the model returned.

---

### 7.4 Deep Observability (Stack Traces)

For developers using **Code Nodes**, Phaxa provides deep observability into execution failures. When a JavaScript node throws an error (e.g., a type error or a manual `throw`), the platform captures the full **JavaScript Stack Trace**.

- **Display**: The failure reason in the Monitor header will show the primary error message.
- **Detailed Trace**: Hover over or click the failure banner to see the full file and line number where the error occurred within your code block.
- **Durability**: Like all event data, these traces are persisted in the database and visible even after the worker that executed the code has finished.

---

### 7.5 Temporal UI

For deep execution inspection, the **Temporal UI** is available at `http://localhost:8088`.

It shows:
- Every workflow execution as a **workflow run** with its full event history
- The exact sequence of activities executed, their inputs and outputs
- Retry history for failed activities
- The ability to terminate or reset a stuck run

The Temporal workflow ID matches the Phaxa `run_id`, so you can find any run by searching for its ID.

---

## 8. State Type Reference

### 8.1 Script Node

A script node evaluates deterministic `expr-lang` expressions — no LLM, no network, sub-millisecond execution.

**YAML:**
```yaml
- name: ROUTE
  type: script
  script:
    trigger: 'amount > 10000 ? "needs_review" : "auto_approve"'
    updates:
      fee: "amount * 0.02"
      priority: 'amount > 50000 ? "high" : "normal"'
```

**`trigger`** is an expression that must evaluate to a string matching one of the outgoing transition labels.

**`updates`** is a map of blackboard field names to expressions that compute their new values. All updates are applied atomically before the transition fires.

**Available syntax:**

| Category | Examples |
|---|---|
| Arithmetic | `amount * 0.2`, `(a + b) / 2` |
| Comparison | `amount > 1000`, `status == "active"` |
| Boolean | `valid && amount > 0`, `!rejected` |
| Ternary | `amount > 1000 ? "high" : "low"` |
| String | `contains(vendor, "GmbH")`, `upper(status)` |
| Null-safe | `status ?? "pending"` |
| Collections | `len(items) > 0`, `any(items, # > 100)` |

**Execution engine:** [expr-lang/expr](https://github.com/expr-lang/expr) — a Go expression evaluator. Fully sandboxed, no side effects.

---

### 8.2 Code Node

A code node runs user-written JavaScript in a sandboxed VM for cases that require loops, complex logic, or transformations that expr-lang cannot express.

**YAML:**
```yaml
- name: ENRICH
  type: code
  code:
    language: javascript
    code: |
      // Compute VAT and classify the invoice
      const vatRate = bb.country === 'DE' ? 0.19 : 0.20;
      bb.vat = bb.amount * vatRate;
      bb.total = bb.amount + bb.vat;

      if (bb.total > 10000) {
        trigger('needs_review');   // early exit
      }

      return {
        trigger: 'auto_approve',
        reasoning: `Total ${bb.total} is within auto-approval limit`,
      };
```

**Sandbox API:**

| Symbol | Description |
|---|---|
| `bb` | Mutable blackboard object — read and write fields directly |
| `trigger('name')` | Fire a trigger immediately and stop execution |
| `return { trigger, reasoning?, blackboard_updates? }` | Return a trigger at the end of the script |
| `console.log/warn/error()` | Output captured in server logs |

**`blackboard_updates`** in the return value is an explicit map of field writes. If provided, it **overrides** any `bb` mutations made during the script. Use it when you want atomic all-or-nothing updates.

**Constraints:**
- No `fetch`, `XMLHttpRequest`, or any network access
- No `require` / `import`
- No `setTimeout` / `setInterval`
- Default timeout: **60 seconds** (override with the `timeout` field on the state)

**Execution engine:** [dop251/goja](https://github.com/dop251/goja) — a pure-Go ES5.1+ JavaScript engine with partial ES6 support. No CGO, no V8.

---

### 8.3 HITL State

The HITL (Human-in-the-Loop) state pauses execution and waits for an external signal (usually from a human operator). By providing a `form_schema`, you can generate a structured UI for data entry.

**Task Inbox & "My Tasks"**
The Task Inbox allows operators to manage pending work across the entire tenant:
- **"All Tasks"**: Shows every unresolved HITL request in the tenant.
- **"My Tasks"**: Filters the list to show only tasks assigned to the current user (based on the `assignee` field in the state definition matching the user's username).

Operators can claim tasks by resolving them, which advances the workflow and clears the item from the inbox.

**Component Mapping:**

| JSON Schema Type | UI Component |
|---|---|
| `string` | Single-line text input |
| `string` + `enum` | Dropdown selection box |
| `number` / `integer` | Numeric input |
| `boolean` | Toggle switch |

**Comprehensive Example:**

```json
{
  "type": "object",
  "title": "Approval Request",
  "required": ["decision"],
  "properties": {
    "reviewer_name": {
      "type": "string",
      "title": "Reviewer Name"
    },
    "impact_score": {
      "type": "number",
      "title": "Impact Score (1-100)",
      "default": 50
    },
    "category": {
      "type": "string",
      "title": "Category",
      "enum": ["Financial", "Operational", "Strategic"]
    },
    "urgent": {
      "type": "boolean",
      "title": "Mark as Urgent",
      "default": false
    },
    "decision": {
      "type": "string",
      "title": "Final Decision",
      "enum": ["Approved", "Rejected", "Needs More Info"]
    }
  }
}
```

When the operator submits this form, the values are **automatically merged into the Blackboard**, making them available for subsequent states to use in reasoning or logic.

---

## 9. Workflow YAML Reference

A complete annotated example:

```yaml
apiVersion: chainnodes/v1
kind: Workflow

metadata:
  name: invoice-approval          # unique identifier
  version: "1.0.0"                # human-readable version string
  description: "Invoice approval workflow"
  system_prompt: |                # prepended to every agent call
    You are an invoice processing assistant. Be concise and precise.
  schedule: "0 0 9 * * 1-5"      # optional: run Mon-Fri at 09:00

blackboard:
  schema:
    invoice_id: { type: string,  required: true }
    vendor:     { type: string,  required: true }
    amount:     { type: number,  required: true }
    currency:   { type: string,  default: "EUR" }
    approved:   { type: bool,    default: false }
    fee:        { type: number }

states:
  - name: VALIDATE
    type: initial
    agent: validator
    instructions: "Check that vendor and amount are valid. Return 'valid' or 'invalid'."
    timeout: 2m
    on_timeout: timeout_error

  - name: HIGH_VALUE_REVIEW
    type: hitl
    assignee: finance-team        # informational
    form_schema:                  # Generative UI definition
      type: object
      properties:
        comment: { type: string, title: "Reviewer Comment" }
        risk_level: { type: string, enum: ["Low", "Medium", "High"] }
    timeout: 48h
    on_timeout: auto_reject

  - name: CALCULATE_FEE
    type: code
    code:
      language: javascript
      code: |
        bb.fee = bb.amount * 0.015;
        return { trigger: 'done', reasoning: 'Fee calculated' };

  - name: AUTO_APPROVE
    type: script
    script:
      trigger: '"approved"'
      updates:
        approved: "true"
        fee: "amount * 0.01"

  - name: APPROVED
    type: terminal

  - name: REJECTED
    type: terminal

  - name: ERROR
    type: terminal

transitions:
  - from: VALIDATE
    to: HIGH_VALUE_REVIEW
    trigger: valid
    guard: "amount > 10000"

  - from: VALIDATE
    to: CALCULATE_FEE
    trigger: valid
    guard: "amount <= 10000"

  - from: VALIDATE
    to: REJECTED
    trigger: invalid

  - from: HIGH_VALUE_REVIEW
    to: APPROVED
    trigger: approved

  - from: HIGH_VALUE_REVIEW
    to: REJECTED
    trigger: rejected

  - from: HIGH_VALUE_REVIEW
    to: REJECTED
    trigger: auto_reject

  - from: CALCULATE_FEE
    to: AUTO_APPROVE
    trigger: done

  - from: AUTO_APPROVE
    to: APPROVED
    trigger: approved

  - from: VALIDATE
    to: ERROR
    trigger: timeout_error

agents:
  - name: validator
    config:
      prompt: "Validate the invoice fields. vendor and amount must be non-empty and amount must be positive."
      provider: anthropic         # optional override
      mcp_servers: "{{ env.MCP_INVOICE_PARSER_URL }}"
```

---

### Key YAML fields

| Field | Description |
|---|---|
| `metadata.schedule` | Cron schedule (6-field robfig format, seconds first) |
| `metadata.system_prompt` | Prepended to every agent call |
| `state.instructions` | State-specific context appended to the system prompt |
| `state.timeout` | Max duration before `on_timeout` trigger fires (Go duration: `30s`, `5m`, `2h`) |
| `state.on_timeout` | Trigger name fired on timeout |
| `state.condition` | `expr-lang` expression for `wait` nodes — fires `on_condition` when true |
| `state.on_condition` | Trigger fired when the wait condition is met (default: `condition_met`) |
| `transition.guard` | `expr-lang` expression that must be true for this edge to be taken |
| `transition.to_nodes` | List of target states for parallel fan-out (instead of `to`) |
| `agent.model` | Override the model for this agent |
| `agent.max_output_tokens` | Configurable per provider in Settings; prevents truncation of large workflows |
| `agent.task_queue` | Route to a specialist Temporal worker |
| `agent.config.mcp_servers` | Comma-separated MCP server URLs or `{{ env.VAR }}` references |

---

## 10. Execution Engines

The platform uses two open-source execution engines for deterministic state logic:

### expr-lang/expr — Script nodes

**Repository:** https://github.com/expr-lang/expr

A fast, sandboxed Go expression evaluator. Used by `script` nodes to evaluate trigger expressions and blackboard update expressions.

- Compiles expressions to bytecode at load time — sub-microsecond evaluation
- Strictly sandboxed: no I/O, no side effects, no external calls
- Type-safe: variables and return types are checked at compile time
- Blackboard fields are bound as top-level variables
- Full operator set: arithmetic, comparison, boolean, ternary, string builtins, collection functions

### dop251/goja — Code nodes

**Repository:** https://github.com/dop251/goja

A pure-Go JavaScript engine implementing ECMAScript 5.1 with significant ES6 additions. Used by `code` nodes to run user-written JavaScript.

- No CGO, no external dependencies — runs anywhere Go runs
- Supports: `let`/`const`, arrow functions, template literals, destructuring, spread, `Promise` (limited), `Map`/`Set`, `Symbol`
- Does **not** support: `fetch`, `require`, DOM APIs, async I/O of any kind
- Interrupted cleanly via `vm.Interrupt()` — used for both `trigger()` early exit and timeout enforcement
- Heartbeat support via Temporal activity heartbeats — long-running scripts don't time out the workflow

### robfig/cron — Scheduled workflows

**Repository:** https://github.com/robfig/cron

Used for the `metadata.schedule` cron field. Uses a 6-field format with seconds as the first field:

```
┌──────────── second (0-59)
│ ┌────────── minute (0-59)
│ │ ┌──────── hour (0-23)
│ │ │ ┌────── day of month (1-31)
│ │ │ │ ┌──── month (1-12)
│ │ │ │ │ ┌── day of week (0-6, Sunday=0)
│ │ │ │ │ │
0 0 9 * * 1-5   →  Mon–Fri at 09:00:00
```
