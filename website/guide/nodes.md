# State Nodes: The Building Blocks of Reasoning

Workflows in Chain Nodes are composed of specialized state nodes, each representing a distinct step in the "Durable Reasoning" chain. These nodes are executing blocks of logic tied together by a state machine.

---

## 1. Initial State (`initial`)
The entry point of every workflow run. When a run starts, it begins at the `initial` state, evaluates its entry logic, and automatically launches the transitions.
- **Usage**: Used to process starting inputs or setup default blackboard state keys.
- **Example**:
  ```yaml
  - name: INTAKE
    type: initial
    script:
      trigger: '"start"'
  ```

---

## 2. Prompt State (`prompt`)
The core reasoning unit. Executes an AI Agent on your worker task queues. It takes plain-text instructions (with optional Blackboard templating) and parses the agent's LLM response back into blackboard variables.
- **Capabilities**:
  - **Tool Binding**: Connects agents to registered **MCP servers** (SSE or stdio).
  - **Templating**: Direct double-brace substitution (e.g. `{{ ticket_id }}`) to inject current blackboard data.
  - **Provider Assignment**: Configurable per state to target specific active LLM providers (e.g. OpenAI vs Ollama).
- **Example**:
  ```yaml
  - name: ANALYZE
    type: prompt
    agent: classifier-bot
    instructions: "Evaluate the sentiment of {{ customer_message }}. Set sentiment: positive/negative."
  ```

---

## 3. Human-in-the-Loop State (`hitl`)
Suspends automated execution and creates an interactive task in the collaborative **Social Inbox**.
- **Features**:
  - **Zero Compute Idle**: Powered by Temporal, the process completely sleeps (requiring zero memory/CPU) until a human responds.
  - **Interactive Forms**: Captures input schema fields defined in the workflow's input registry.
  - **Assignees**: Assign tasks to specific roles (e.g. `billing-managers`).
- **Example**:
  ```yaml
  - name: MANAGER_APPROVAL
    type: hitl
    assignee: billing-managers
    instructions: "Review mathematical mismatch on invoice total."
  ```

---

## 4. Script State (`script`)
An ultra-fast, zero-latency execution node designed for mathematical evaluations and fast routing decisions. It evaluates expressions using Go's `expr` package.
- **Usage**: Modifying variable values or checking conditions instantly.
- **Example**:
  ```yaml
  - name: ASSIGN_QUEUE
    type: script
    script:
      trigger: "urgency == 'critical' ? 'route_priority' : 'route_standard'"
      updates:
        assigned_team: "urgency == 'critical' ? 'tier_3' : 'tier_1'"
  ```

---

## 5. Code State (`code`)
Executes custom Javascript (ES6) inside a secure, fully isolated Goja VM sandbox. Ideal for parsing complex JSON, array manipulation, and advanced math.
- **Features**:
  - **Direct Mutation**: Mutate the global `bb` object directly.
  - **Execution Rules**: No `fetch`, `require`, or timers are exposed, keeping execution safe and sandboxed.
- **Example**:
  ```yaml
  - name: CALCULATE_TAX
    type: code
    code:
      language: javascript
      code: |
        bb.tax = bb.subtotal * 0.19;
        bb.total = bb.subtotal + bb.tax;
        return { trigger: 'done', reasoning: 'Tax computed successfully' };
  ```

---

## 6. Wait State (`wait`)
Halts execution at a gate until a specific logical condition expression evaluates to `true` or an optional timeout occurs.
- **Example**:
  ```yaml
  - name: WAIT_FOR_PAYMENT
    type: wait
    condition: "blackboard.payment_received == true"
    on_condition: "continue_fulfillment"
    timeout: "24h"
    on_timeout: "cancel_order"
  ```

---

## 7. Sub-Process State (`subprocess`)
Invokes another workflow as a durable child process, allowing you to compose modular, reusable workflows (Skills) together.
- **Example**:
  ```yaml
  - name: RUN_AUDIT
    type: subprocess
    subprocess:
      name: invoice-auditor
      version: 1
      inputs:
        invoice_id: "blackboard.invoice_id"
  ```

---

## 8. Emit Event State (`emit_event`)
Broadcasts a named platform event to the Redis event bus, waking up or signaling other active/waiting workflow runs in the workspace.
- **Example**:
  ```yaml
  - name: SIGNAL_COMPLETED
    type: emit_event
    emit_event:
      name: "order.shipped"
      payload:
        order_id: "blackboard.order_id"
  ```

---

## 9. Terminal State (`terminal`)
Ends execution of the workflow run. The status is set to completed and clean-up operations are executed.
- **Example**:
  ```yaml
  - name: SUCCESS
    type: terminal
  ```
