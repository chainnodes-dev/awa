# ASM Platform

**Agentic State Machine** — a workflow orchestration platform for building AI-powered business processes.

Define your process as a state machine in YAML. Assign LLM agents, deterministic scripts, or JavaScript code nodes to each state. The platform runs it durably via Temporal, streams live events to the browser, and keeps every routing decision in the workflow definition — not in the workers.

> **Keep workers stupid. Let the state machine decide.**

---

## What it does

- **Visual workflow designer** — drag-and-drop canvas to build state machines, define transitions and guards, assign agents
- **Multiple state types** — LLM agent, JavaScript code node, deterministic script, human-in-the-loop (HITL), wait (event-driven), parallel fan-out
- **AI-assisted authoring** — describe a process in plain English, decompose it into an optimized workflow, or generate code node JavaScript with one click
- **Live monitoring** — real-time event log, blackboard inspector, LLM Trace panel showing every prompt and response (available to all users)
- **Mandatory durability** — every run is a Temporal workflow, surviving server and worker restarts by default
- **Multi-provider LLM** — Anthropic, OpenAI, Grok, DeepSeek, Gemini, Ollama (local); configurable per agent
- **MCP tool servers** — agents call external tools via the Model Context Protocol
- **RBAC auth** — admin / operator / viewer roles, JWT + refresh tokens
- **Multi-tenancy** — isolated tenant contexts with super_admin management

---

## Quick start (Docker)

**Prerequisites:** Docker and Docker Compose.

```bash
git clone https://github.com/bauhaus28/asm-platform
cd asm-platform
cp .env.example .env
```

Open `.env` and set the two required values:

```bash
ANTHROPIC_API_KEY=sk-ant-...   # or OPENAI_API_KEY / OLLAMA_URL
ADMIN_PASSWORD=your-password
```

Then start everything:

```bash
docker compose up -d
```

Open **http://localhost** — log in with `admin` / your password.

| URL | What |
|---|---|
| http://localhost | ASM frontend |
| http://localhost:8088 | Temporal UI |

> First startup takes ~60 seconds while Temporal initialises against Postgres.
> Watch progress with `docker compose logs -f temporal`.

---

## Quick start (local dev)

**Prerequisites:** Go 1.24+, Node 20+, Docker (for infrastructure).

```bash
# 1. Start infrastructure (Postgres, Redis, Temporal)
make infra

# 2. Copy and fill in config
cp .env.example .env  # set at least LLM_PROVIDER + API key

# 3. Start everything in tmux panes (requires tmux)
make dev
```

Or run each piece individually:

```bash
make server    # Go API server  →  :8080
make worker    # Temporal worker
make frontend  # Vite dev server →  :5173
```

---

## Workflow definition

Workflows are YAML manifests. Example:

```yaml
apiVersion: asm/v1
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
    type: intermediate
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
| `intermediate` | LLM agent reasons and fires a trigger |
| `code` | JavaScript runs in a sandboxed goja VM; `bb` is mutable |
| `script` | Deterministic expr-lang expressions; zero latency |
| `hitl` | Pauses and waits for a human signal via the API or UI |
| `wait` | Pauses until a condition expression on the blackboard becomes true |

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
┌─────────────────────────────┐
│         Go API Server        │
│  - Workflow CRUD             │
│  - Run lifecycle             │
│  - HITL signals              │
│  - WebSocket hub             │
└────────────┬────────────────┘
             │ Temporal SDK
             ▼
┌─────────────────────────────┐
│       Temporal Server        │
│  (durable workflow engine)   │
└────────────┬────────────────┘
             │ Task Queue
             ▼
┌─────────────────────────────┐
│       Temporal Worker        │
│  - ASMWorkflow               │
│  - ExecuteAgent activity     │
│  - ExecuteCode activity      │
│  - ExecuteScript activity    │
│  - HITL / Wait activities    │
└─────────────────────────────┘

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

All configuration is via environment variables. Copy `.env.example` for a full annotated reference.

Key variables:

| Variable | Description | Default |
|---|---|---|
| `ANTHROPIC_API_KEY` | Anthropic API key | — |
| `OPENAI_API_KEY` | OpenAI API key | — |
| `OLLAMA_URL` | Ollama base URL for local LLMs | — |
| `ADMIN_USERNAME` | Seeded admin username | `admin` |
| `ADMIN_PASSWORD` | Seeded admin password (required) | — |
| `JWT_SECRET` | JWT signing secret — **change in production** | dev default |
| `FRONTEND_PORT` | Host port for the Nginx frontend container | `80` |

---

## Using Ollama (local LLMs)

```bash
# In .env
OLLAMA_URL=http://host.docker.internal:11434  # from Docker, or http://localhost:11434 for local dev
LLM_PROVIDER=ollama
OLLAMA_MODEL=llama3.2
```

Or keep the global provider as Anthropic/OpenAI and route individual agents to Ollama:

```yaml
agents:
  - name: classifier
    model: llama3.2   # routed to Ollama when OLLAMA_URL is set
```

---

## Specialist workers

For deterministic, high-throughput states that don't need LLM reasoning, you can build a typed Go worker on its own task queue:

```bash
# Scaffold a new specialist worker
make specialist name=invoice-worker queue=invoice-workers
```

This copies the template in `cmd/specialist-worker-template/` and gives you a typed Go handler to fill in.

---

## License

MIT
