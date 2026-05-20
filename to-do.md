# Phaxa Strategic Roadmap: Enterprise Agentic Automation

This document tracks the core missing features required to elevate Phaxa from a powerful developer engine to a dominant Enterprise Agentic Automation platform, capturing market share from legacy iPaaS tools (like n8n, Make) and unstable AI frameworks.

## 1. ⚡ Dynamic Trigger Engine (Event Gateway)
Currently, workflows are started manually or via basic cron schedules. We need to implement an event-driven "Wake Up" layer.
- [x] **Workflow DSL Update**: Add a `triggers:` array to the workflow YAML definition.
- [x] **Webhook Router**: Build a dynamic endpoint router (`POST /api/v1/triggers/webhook/...`) that intercepts payloads and starts workflow runs.
- [x] **Background Listeners**: Implement long-polling listener threads for messenger services (Telegram, Discord) that run natively in the background and trigger workflows on new messages.
- [x] **Payload Injection**: Automatically map trigger payloads (headers, body, chat_ids) directly into the blackboard schema upon workflow start.

## 2. 🎨 True Drag & Drop Visual Canvas
The current visual editor requires a multi-step click path (create node -> select type -> configure). We need to upgrade this to a seamless, industry-standard visual experience.
- [ ] **Fluid Drag & Drop**: Implement a pure drag-and-drop node graph canvas (e.g., using Vue Flow or React Flow).
- [ ] **Bidirectional Sync**: Ensure dragging a node or connecting an edge on the canvas instantly updates the underlying YAML manifest (and vice versa).
- [ ] **Visual Data Mapping**: Allow users to visually map blackboard variables between nodes without typing JSONPath or expressions.

## 3. 🔐 Dynamic Secrets Vault (Security Layer)
Enterprises cannot restart Docker containers to inject new API keys via `.env` files.
- [ ] **Secure Storage**: Create an AES-256 encrypted database table for storing sensitive API keys and tokens.
- [ ] **Vault UI**: Build a frontend dashboard to manage, rotate, and scope secrets.
- [ ] **Runtime Injection**: Modify the Temporal worker to securely decrypt and inject these secrets into MCP Stdio subprocesses at runtime (e.g., `{{ secrets.TELEGRAM_TOKEN }}`).

## 4. 🧠 Agentic Observability ("The Glass Brain")
When AI agents fail, hallucinate, or loop infinitely, developers need to know exactly *why*.
- [ ] **Tool-Call Interception**: Modify the `prompt` state executor to intercept every MCP tool call the LLM makes.
- [ ] **Execution History Logging**: Record the exact tool name, the inputs provided by the LLM, and the exact response returned by the MCP server.
- [ ] **UI Visualization**: Display these granular tool-call steps inside the Run Details event log so users can trace the AI's entire "thought process".

## 5. 🐝 Multi-Agent "Swarm" Orchestration
Complex tasks require specialized agents working together (e.g., a Manager delegating to a Researcher and a Writer).
- [x] **Skills as Tools**: Expose Phaxa's reusable workflows (Skills) directly as MCP tools to the LLM agent.
- [x] **Seamless Delegation**: When an agent calls a "skill" tool, dynamically start a child Temporal workflow, wait for its durable completion, and return the result to the parent agent. 
