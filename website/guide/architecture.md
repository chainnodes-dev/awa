# Platform Architecture

Chain Nodes is built on a **"Durable Reasoning"** architecture. Unlike traditional agent frameworks that rely on ephemeral loops, Chain Nodes treats every step of an agent's reasoning process as a persistent state. This ensures that workflows can survive infrastructure failures, network partitions, and long human-in-the-loop wait times without losing context.

## System Overview

The platform is a modular distributed system composed of the following layers:

### 1. Neural Designer (Visual Workspace)
The **Neural Designer** is a sophisticated Vue.js application that provides a visual interface for orchestrating agentic logic. It allows authors to:
- **Scaffold Workflows**: Use natural language to describe a business process, which the AI then converts into a deterministic YAML definition.
- **Visual Debugging**: Trace the flow of data through the "Blackboard" in real-time.
- **Hot-Reloading**: Edit workflow definitions and apply changes to running instances (for compatible versions).

### 2. Orchestrator & API (The Brain)
The Go-based **Orchestrator** is the central nervous system of Chain Nodes. It handles:
- **State Management**: Persisting the "Blackboard" state and ensuring tenant isolation at the database level.
- **Identity & Access**: Managing RBAC, SSO integrations, and encrypted storage of provider API keys.
- **Event Bus**: Using Redis to fan out updates to workers and the UI in real-time.

### 3. Durable Execution (Temporal.io)
At its core, Chain Nodes uses **Temporal.io** for workflow orchestration. Unlike standard task queues (like Celery or BullMQ), Temporal provides **Workflows-as-Code**. This means:
- **Infinite Retries**: If an LLM provider is down for 3 hours, Chain Nodes will simply wait and retry the exact same execution state once the service returns.
- **Durable Timers**: We can pause a workflow for 3 months (waiting for a legal approval, for example) and resume it without any resource overhead.
- **Execution History**: Every step, every variable change, and every external call is recorded in an immutable history, allowing developers to "time travel" through a failed run to find the exact cause.

### 4. Agent Executors & MCP
Workers host **Agent Executors** — secure, ephemeral environments that execute the actual logic. These executors are designed to be "tool-aware" from the ground up using the **Model Context Protocol (MCP)**.
- **Protocol-First Design**: By leveraging MCP, Chain Nodes decouples the agent logic from the tools it uses. You can swap a Google Search tool for a Bing Search tool without changing a single line of your workflow YAML.
- **Context Management**: The platform automatically manages the token budget of tool responses, summarizing long outputs or truncating them to ensure the model remains within its cognitive limits.
- **Streaming State**: Real-time feedback from tool execution is streamed back to the dashboard, allowing users to watch the agent "think" and "act" live.

## Security & Compliance Model

Chain Nodes is built for the most demanding enterprise environments where security is a first-class citizen.

### Multi-Tenant Isolation
Every organization (Tenant) in Chain Nodes is logically isolated.
- **Namespace Security**: Temporal namespaces and database schemas are used to ensure that Tenant A can never see or interfere with the workflows of Tenant B.
- **Secret Management**: API keys for OpenAI, Anthropic, or custom internal tools are encrypted at rest using industry-standard AES-256-GCM.

### Auditability & Compliance
Every action taken by an agent is logged and can be audited.
- **Model Traces**: Capture every prompt sent and every completion received, including hidden reasoning steps (like Chain-of-Thought).
- **Tool Call Logs**: Detailed records of which tool was called, with what arguments, and what the raw response was.
- **License Gating**: Enterprise features like SSO and audit logging are strictly enforced via the Chain Nodes License Engine, which supports both cloud and air-gapped on-prem installations.
