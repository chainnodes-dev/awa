# Phaxa — Implementation Plan

> **Goal:** Build the best-in-class Agentic Orchestration platform, surpassing traditional automation tools like n8n.

---

## 1. Vision: Agentic vs. Deterministic
n8n is for **Deterministic Automation** (A always leads to B). 
Phaxa is for **Agentic Orchestration** (The goal is X, the Agent figures out how to get there using MCP).

### Our Differentiators:
- **Zero-Node Integration**: Infinite tools via the Model Context Protocol (MCP).
- **Time Travel Debugging**: X-ray vision into the Agent's thought process.
- **Collaborative HITL**: Active chat between humans and agents.
- **Self-Healing**: AI that corrects its own errors and finds alternate paths.

---

## 2. Roadmap: The "Better than n8n" Phase

### Milestone 14: The MCP Marketplace ⬜
- [ ] **MCP Discovery UI**: A gallery to browse community MCP servers.
- [ ] **One-Click Install**: Auto-registration of `npx`-based servers via UI.
- [ ] **Capability Search**: Find the right tool for the job automatically.

### Milestone 15: Deep Observability ⬜
- [ ] **Thought Trace**: Display LLM chain-of-thought in the Monitor.
- [ ] **Blackboard Time Travel**: Slider to see state diffs at any point in history.
- [ ] **[UX.4] JS Stack Traces**: Surface code node errors with full traces.

### Milestone 16: Social Inbox (Collaborative HITL) ⬜
- [ ] **Agent-Human Chat**: Interactive messaging inside the Inbox.
- [ ] **Contextual Hand-off**: Human sees "thoughts" before taking over.
- [ ] **[Debt] Assignee Filtering**: Security and multi-user task management.

### Milestone 17: Proactive Designer ⬜
- [ ] **Incremental Generation**: Natural language editing of existing graphs.
- [ ] **[UX.1] Schema Validation**: Real-time feedback in the YAML editor.
- [ ] **[UX.2] Live Type Hints**: Port mapping hints from the skill registry.

---

## 3. Technical Debt & Maintenance

| Feature | Status | Priority |
|---|---|---|
| Designer Keyboard Shortcuts (Save, Undo, Del) | ⬜ | High |
| `skill_call` timeout & failure_trigger | ⬜ | Medium |
| `skill_ref` versioning (`@v1`) | ⬜ | Medium |
| Tenant deletion safety | ⬜ | Low |

---

## 4. Completed Milestones (Archive)

| Milestone | Delivered |
|---|---|
| **M1-M12** | Foundation, Temporal, MCP, Designer, Multi-tenancy, Skills (Phases 1-3). |
| **M13** Designer 2.0 | Two-stage generation, Trace for all, Settings UI. |

---

## 5. Running the Platform

```bash
# Infrastructure
docker compose up -d

# Backend
go run ./cmd/server
go run ./cmd/worker

# Frontend
cd frontend && npm run dev
```
