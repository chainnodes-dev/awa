# Chain Nodes

**Agentic State Machine** — a workflow orchestration platform for building AI-powered business processes.

Define your process as a state machine in YAML. Assign LLM agents, deterministic scripts, or JavaScript code nodes to each state. The platform runs it durably via Temporal, streams live events to the browser, and keeps every routing decision in the workflow definition — not in the workers.

> **Keep workers as simple as possible. Let the state machine decide.**

---

## What it does

- **Visual workflow designer** — drag-and-drop canvas to build state machines, define transitions and guards, assign agents
- **Multiple state types** — LLM agent, JavaScript code node, deterministic script, human-in-the-loop (HITL), wait (event-driven), parallel fan-out
- **AI-assisted authoring** — describe a process in plain English, decompose it into an optimized workflow, or generate code node JavaScript with one click
- **Live monitoring** — real-time event log, blackboard inspector, LLM Trace panel showing every prompt and response
- **Mandatory durability** — every run is a Temporal workflow, surviving server and worker restarts by default
- **Multi-provider LLM** — Anthropic, OpenAI, Grok, DeepSeek, Gemini, Ollama (local); configurable per agent
- **MCP tool servers** — agents call external tools via the Model Context Protocol
- **RBAC auth** — admin / operator / viewer roles, JWT + refresh tokens
- **Collaborative Social Inbox** — integrated human-agent chat with assignee-based task filtering

---

The fastest way to deploy the full stack (API, Worker, Frontend, Temporal, Postgres, Redis) is using Docker Compose. No configuration or `.env` files are required for the initial launch:

```bash
git clone https://github.com/chainnodes-dev/awa
cd awa
docker compose up -d
```

Open **[http://localhost:5174](http://localhost:5174)** to access the platform.

### 🛡 Initial Setup
On your first visit, Chain Nodes will automatically detect that the platform is uninitialized and guide you through the **Super Admin Setup**. 
1. Create your primary administrator account.
2. Go to **Settings** to configure your LLM providers (Anthropic, OpenAI, etc.).
3. You're ready to build!


### Production Deployment
For stable releases using pre-built images from the registry:
```bash
docker compose -f docker-compose.prod.yaml up -d
```

| URL | What |
|---|---|
| http://localhost | Chain Nodes frontend |
| http://localhost:8088 | Temporal UI |

> First startup takes ~60 seconds while Temporal initialises against Postgres.
> Watch progress with `docker compose logs -f temporal`.

---

### Quick Start (Manual Dev)
To start infrastructure dependencies (Postgres, Redis, Temporal) in Docker while running the Chain Nodes services manually for development:

```bash
make dev-infra  # Starts DB, Cache, and Temporal
make server     # API Server  →  :8080
make worker     # Temporal Worker
make frontend   # Vite Dev    →  :5173
```

---

## Workflow definition

Workflows are YAML manifests. Example:

```yaml
apiVersion: chainnodes/v1
kind: Workflow
metadata:
  name: invoice-approval
  version: "1.0.0"
  description: Route invoices through validation and approval

blackboard:
  schema:
    amount:    { type: number }
    vendor:    { type: string }
    approved:  { type: bool }

states:
  - name: VALIDATE
    type: initial
    agent: validator

  - name: APPROVE
    type: prompt
    agent: approver

  - name: AUTO_APPROVE
    type: code
    code:
      language: javascript
      code: |
        bb.approved = true;
        return { trigger: 'done', reasoning: 'Below threshold' };

  - name: DONE
    type: terminal

transitions:
  - from: VALIDATE
    to: APPROVE
    trigger: valid
    guard: "amount > 1000"

  - from: VALIDATE
    to: AUTO_APPROVE
    trigger: auto
    guard: "amount <= 1000"

  - from: APPROVE
    to: DONE
    trigger: approved

  - from: AUTO_APPROVE
    to: DONE
    trigger: done

agents:
  - name: validator
    config:
      prompt: "Validate the invoice. Return 'valid' if amount and vendor are present."
  - name: approver
    config:
      prompt: "Review the invoice for amount {{ bb.amount }} from {{ bb.vendor }}. Approve or reject."
```

---

## State types

| Type | Description |
|---|---|
| `initial` | Entry point of the workflow |
| `terminal` | End state — workflow completes here |
| `prompt` | LLM agent reasons and fires a trigger |
| `code` | JavaScript runs in a sandboxed goja VM; `bb` is mutable |
| `script` | Deterministic expr-lang expressions; zero latency |
| `hitl` | Pauses and waits for a human signal via the API or UI |
| `wait` | Pauses until a condition expression on the blackboard becomes true |
| `subprocess` | Invokes another workflow as a durable child process |
| `emit_event` | Fires a named event to wake up other waiting runs |

### Code node sandbox

```js
// Read and write the blackboard directly
bb.fee = bb.amount * 0.02;

// Early exit
if (bb.amount > 10000) {
  trigger('needs_review');  // fires immediately, stops execution
}

// Return a trigger (and optional reasoning)
return {
  trigger: 'auto_approve',
  reasoning: 'Amount is within auto-approval limit',
};
```

Constraints: no `fetch`, no `require`, no `setTimeout` — fully sandboxed.

---

## Architecture

```
Browser
  │  WebSocket (live events)
  │  REST /api/v1
  ▼
┌──────────────────────────────┐
│    Chain Nodes API Server    │
│  - Workflow CRUD             │
│  - Run lifecycle             │
│  - HITL signals              │
│  - WebSocket hub             │
└────────────┬─────────────────┘
             │ Temporal SDK
             ▼
┌──────────────────────────────┐
│       Temporal Server        │
│  (durable workflow engine)   │
└────────────┬─────────────────┘
             │ Task Queue
             ▼
┌──────────────────────────────┐
│       Temporal Worker        │
│  - Chain Nodes Workflow      │
│  - ExecuteAgent activity     │
│  - ExecuteCode activity      │
│  - ExecuteScript activity    │
│  - HITL / Wait activities    │
└──────────────────────────────┘

Shared infrastructure: Postgres (state), Redis (events pub/sub)
```

Each workflow run is a durable Temporal workflow. The worker loop advances the state machine, calls LLM agents or executes code, and publishes events back through Redis → WebSocket → browser.

---

## Repository layout

```
cmd/
  server/               API server binary
  worker/               Temporal worker binary
  specialist-worker/    Template for custom task-queue workers
  mcp-server-template/  Template for MCP tool servers
internal/
  api/                  HTTP handlers and routes
  auth/                 JWT, RBAC, refresh tokens
  designer/             AI workflow generator, code scaffolder
  events/               Event types and bus (Local + Redis)
  executor/
    code/               goja JavaScript sandbox
    llm/                LLM provider abstraction (Anthropic, OpenAI, …)
  temporal/             Workflow, activities, worker setup
pkg/
  asmtypes/             Shared workflow and run types
frontend/
  src/
    components/
      designer/         Canvas, state panel, code editor, generate panel
      monitor/          Event log, blackboard view, LLM Trace panel
    views/              Designer, Monitor, Workflows list, Skills list
    stores/             Pinia stores (auth, execution, workflows)
```

---

## Configuration

LLM keys, administrator accounts, and provider URLs are configured directly in the web UI under **Settings**.

System-level settings can be configured via environment variables. Copy `.env.example` for a full annotated reference.

Key variables:

| Variable | Description | Default |
|---|---|---|
| `JWT_SECRET` | JWT signing secret — **change in production** | dev default |
| `FRONTEND_PORT` | Host port for the Nginx frontend container | `80` |
| `ENABLE_TELEMETRY` | Opt-in anonymous install beacon (sends OS/Arch on startup) | `false` |

---

## Using Ollama (local LLMs)

You can enable and configure Ollama directly in the **Settings** menu:
1. Enable the **Ollama** provider.
2. Set the **Server URL** (e.g. `http://host.docker.internal:11434` when running inside Docker, or `http://localhost:11434` for local dev).
3. Set the **Default Model** (e.g. `llama3.2`).

You can route individual agents to Ollama by specifying their model in the workflow YAML:

```yaml
agents:
  - name: classifier
    model: llama3.2   # routed to Ollama when Ollama is configured in Settings
```

---

## License

Polyform Perimeter 1.0.0 (Free for private and commercial internal use, prohibited for competing products or re-hosting).
