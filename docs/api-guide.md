# Phaxa — API Integration Guide

This guide covers how to submit items to the platform and consume the results programmatically. It assumes the server is running and reachable at `https://your-phaxa-host`.

---

## 1. Authentication

All API calls (except health checks) require a JWT bearer token. Tokens expire after 15 minutes; use the refresh flow to maintain a session.

### Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "alice",
  "password": "secret",
  "tenant_slug": "acme-corp"   // omit to use the default tenant
}
```

**Response**

```json
{
  "access_token": "eyJ...",
  "refresh_token": "a3f9...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": "550e8400-...",
    "username": "alice",
    "role": "operator",
    "tenant_id": "00000000-..."
  }
}
```

Use `access_token` in the `Authorization` header for all subsequent requests:

```
Authorization: Bearer eyJ...
```

### Refresh

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "a3f9..."
}
```

Returns a new `access_token` and a rotated `refresh_token`. The old refresh token is immediately invalidated.

---

## 2. Submitting an Item (Starting a Run)

Each item processed by the platform is a **run** — a single workflow instance carrying its own data envelope (the blackboard). Starting a run is how you submit an item.

### Prerequisites

You need the workflow `name` and `version` to start a run. List available workflows:

```http
GET /api/v1/workflows
Authorization: Bearer eyJ...
```

**Response (excerpt)**

```json
[
  {
    "name": "invoice-processing",
    "version": "1.0.0",
    "version_number": 3
  }
]
```

### Start a Run

```http
POST /api/v1/runs
Authorization: Bearer eyJ...
Content-Type: application/json

{
  "workflow_name": "invoice-processing",
  "workflow_version": "1.0.0",
  "input": {
    "invoice_id": "INV-001",
    "vendor":     "Acme Ltd",
    "amount":     5000,
    "currency":   "EUR"
  }
}
```

The `input` object seeds the initial blackboard. Its keys must satisfy the workflow's blackboard schema (required fields, types). Extra keys are accepted and carried through.

**Response**

```json
{
  "id": "phaxa-run-a48fca29-...",
  "workflow_name": "invoice-processing",
  "workflow_version": "1.0.0",
  "status": "running",
  "current_state": "VALIDATE",
  "blackboard": {
    "invoice_id": "INV-001",
    "vendor": "Acme Ltd",
    "amount": 5000,
    "currency": "EUR"
  },
  "started_at": "2026-04-11T09:00:00Z",
  "updated_at": "2026-04-11T09:00:00Z"
}
```

Save the `id` — you will use it to poll for results or subscribe to events.

---

## 3. Consuming Processed Results

### Option A — Poll the Run

For simple integrations or low-frequency workflows, polling is the easiest approach.

```http
GET /api/v1/runs/{id}
Authorization: Bearer eyJ...
```

**Response**

```json
{
  "id": "phaxa-run-a48fca29-...",
  "status": "complete",
  "current_state": "APPROVED",
  "blackboard": {
    "invoice_id": "INV-001",
    "vendor": "Acme Ltd",
    "amount": 5000,
    "currency": "EUR",
    "validated": true,
    "vendor_id": "V-42",
    "risk_score": 0.12,
    "approval_timestamp": "2026-04-11T09:04:22Z"
  },
  "completed_at": "2026-04-11T09:04:22Z"
}
```

**Status values**

| Status | Meaning |
|--------|---------|
| `running` | Workflow is actively executing |
| `waiting` | Paused at a human-in-the-loop state |
| `complete` | Reached a terminal state — results are final |
| `failed` | Unrecoverable error; check `failure_reason` |

Poll until `status` is `complete` or `failed`. A sensible polling interval is 2–5 seconds for short workflows, 30 seconds or more for long-running ones.

### Option B — Real-Time Events via WebSocket

For near-real-time integrations, subscribe to the event stream. The WebSocket token is the same JWT used for REST calls.

```
GET /ws?token=eyJ...
Upgrade: websocket
```

Every state change, blackboard update, HITL event, and agent action is pushed as a JSON message:

```json
{
  "type": "state.changed",
  "data": {
    "run_id": "phaxa-run-a48fca29-...",
    "from_state": "VALIDATE",
    "to_state": "APPROVE",
    "trigger": "valid",
    "blackboard": { "validated": true, "vendor_id": "V-42" }
  }
}
```

**Event types**

| Type | When it fires |
|------|--------------|
| `run.created` | A new run has started |
| `state.changed` | The run moved from one state to another |
| `hitl.waiting` | Run paused; human input required |
| `hitl.resolved` | Human input received; run resumed |
| `run.completed` | Run reached a terminal state |
| `run.failed` | Run terminated with an error |
| `agent.tool_call` | An LLM agent called an MCP tool (debug/observability) |

React to `run.completed` or `state.changed` to drive downstream logic without polling.

---

## 4. Reading the Transition History

The full audit trail — every state visited, every trigger fired, every blackboard snapshot — is queryable after the run completes (or at any point during execution).

```http
GET /api/v1/runs/{id}/history
Authorization: Bearer eyJ...
```

**Response**

```json
[
  {
    "from_state": "START",
    "to_state": "VALIDATE",
    "trigger": "begin",
    "timestamp": "2026-04-11T09:00:01Z",
    "blackboard_snapshot": { "invoice_id": "INV-001", "amount": 5000 }
  },
  {
    "from_state": "VALIDATE",
    "to_state": "APPROVE",
    "trigger": "valid",
    "timestamp": "2026-04-11T09:00:03Z",
    "blackboard_snapshot": { "validated": true, "vendor_id": "V-42", "amount": 5000 },
    "agent_output": {
      "trigger": "valid",
      "reasoning": "All required fields present. Vendor matched.",
      "blackboard_updates": { "validated": true, "vendor_id": "V-42" }
    }
  }
]
```

Each entry is immutable and append-only. The `blackboard_snapshot` field gives you the full state of the data envelope at the moment of each transition.

---

## 5. Handling Human-in-the-Loop States

Some workflows pause at `hitl` states pending a human decision. The run status becomes `waiting`. Your integration can either present the decision to a user or make it programmatically.

### List Pending HITL Requests

```http
GET /api/v1/hitl/pending
Authorization: Bearer eyJ...
```

**Response**

```json
[
  {
    "id": "hitl-req-...",
    "run_id": "phaxa-run-a48fca29-...",
    "state_name": "HUMAN_REVIEW",
    "assignee": "finance-team",
    "timeout_at": "2026-04-13T09:00:00Z",
    "created_at": "2026-04-11T09:00:05Z"
  }
]
```

### Resolve a HITL Request

The `resolution` value is used directly as the trigger name. The workflow YAML defines which transitions map to which resolution values — confirm the correct trigger names with the workflow author.

```http
POST /api/v1/runs/{id}/signal
Authorization: Bearer eyJ...
Content-Type: application/json

{
  "resolution": "approved",
  "resolver":   "alice@acme.com"
}
```

**Response**

```json
{ "ok": true }
```

---

## 6. Sending an External Trigger

States without an agent (neither LLM nor script) wait for an external trigger before advancing. This is how you signal the workflow from an external system — for example, notifying the platform that a document upload is complete.

```http
POST /api/v1/runs/{id}/trigger
Authorization: Bearer eyJ...
Content-Type: application/json

{
  "trigger": "document_uploaded",
  "payload": {
    "document_url": "https://storage.acme.com/inv-001.pdf",
    "pages": 3
  }
}
```

The `payload` fields are merged into the blackboard before the engine evaluates the next transition. Only operator and admin roles can fire triggers.

---

## 7. Listing and Filtering Runs

```http
GET /api/v1/runs?workflow=invoice-processing&status=complete&limit=100&offset=0
Authorization: Bearer eyJ...
```

**Query parameters**

| Parameter | Description | Example |
|-----------|-------------|---------|
| `workflow` | Filter by workflow name | `invoice-processing` |
| `status` | Filter by run status | `complete`, `failed`, `waiting` |
| `state` | Filter by current state name | `APPROVED` |
| `started_from` | Lower bound on start time (RFC 3339) | `2026-04-01T00:00:00Z` |
| `started_to` | Upper bound on start time (RFC 3339) | `2026-04-11T23:59:59Z` |
| `limit` | Page size (default 50) | `100` |
| `offset` | Pagination offset (default 0) | `200` |

---

## 8. End-to-End Example

The following pseudocode shows the complete integration flow for a document processing use case:

```python
import httpx, time

base = "https://your-phaxa-host/api/v1"

# 1. Authenticate
r = httpx.post(f"{base}/auth/login", json={
    "username": "alice", "password": "secret", "tenant_slug": "acme-corp"
})
token = r.json()["access_token"]
headers = {"Authorization": f"Bearer {token}"}

# 2. Submit an item
r = httpx.post(f"{base}/runs", headers=headers, json={
    "workflow_name":    "invoice-processing",
    "workflow_version": "1.0.0",
    "input": {"invoice_id": "INV-999", "amount": 12500, "currency": "EUR"}
})
run_id = r.json()["id"]

# 3. Poll for result
while True:
    r = httpx.get(f"{base}/runs/{run_id}", headers=headers)
    run = r.json()

    if run["status"] == "complete":
        print("Final state:", run["current_state"])
        print("Blackboard:", run["blackboard"])
        break

    if run["status"] == "failed":
        print("Run failed:", run.get("failure_reason"))
        break

    if run["status"] == "waiting":
        # Programmatic HITL approval — only if your process owns this decision
        httpx.post(f"{base}/runs/{run_id}/signal", headers=headers,
                   json={"resolution": "approved", "resolver": "system"})

    time.sleep(3)

# 4. Read full audit trail
r = httpx.get(f"{base}/runs/{run_id}/history", headers=headers)
for step in r.json():
    print(step["from_state"], "→", step["to_state"], "via", step["trigger"])
```

---

## 9. Required Roles Summary

| Operation | Minimum Role |
|-----------|-------------|
| View runs and workflows | `viewer` |
| Start a run | `operator` |
| Fire a trigger | `operator` |
| Resolve a HITL | `operator` |
| Upload a workflow definition | `operator` |
| Delete a workflow / run | `admin` |
| Manage users | `admin` |
| Manage tenants | `super_admin` |
