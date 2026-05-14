# State Nodes: The Building Blocks of Reasoning

Workflows in Phaxa are composed of specialized nodes, each handling a different aspect of the "Durable Reasoning" chain. These nodes are deterministic units of logic that can be linked together to form complex, non-deterministic processes.

## 1. Prompt Nodes
The **Prompt Node** is the workhorse of the platform. It is designed to transform unstructured or semi-structured data into structured blackboard state by interfacing with Large Language Models (LLMs).

### Deep Configuration
- **Model Selection**: Choose from various providers (OpenAI, Anthropic, local models) with specific parameters like temperature, top-p, and max tokens.
- **Advanced Templating**: Uses a sophisticated template engine that allows you to inject any variable from the Blackboard. For example: `&#123;&#123; user.name &#125;&#125;`.
- **JSON Mode & Schemas**: Enforce structured outputs by providing a JSON Schema. Phaxa will automatically retry or repair responses that do not comply with the schema.
- **Output Mapping**: Map specific parts of the LLM response to Blackboard keys. You can store the raw completion, the token usage, or specific extracted fields.

### Use Case: Sentiment Analysis
A Prompt Node can be configured to analyze customer feedback. It takes the "comment" from the blackboard, sends it to a model with a "System: You are a sentiment classifier" prompt, and stores the result as "sentiment: positive/negative".

---

## 2. Agent Nodes (Autonomous Tool-Use)
The **Agent Node** is a long-running, autonomous state that has access to specific **MCP Tools**. Unlike a prompt node, which is a single "thought," an agent node is a "loop" that can reason, act, and observe until a goal is met.

### Core Capabilities
- **MCP Tool Binding**: Dynamically bind servers from the **MCP Marketplace**. An agent can have access to "Slack", "PostgreSQL", and "GitHub" simultaneously.
- **Reasoning Loop**: The agent operates in a "Think-Act-Observe" cycle. It decides which tool to use, executes it via the worker, observes the result, and repeats until the task is complete.
- **Context Management**: Phaxa manages the cognitive load of the agent. As the conversation history grows, the platform automatically summarizes past tool calls to keep the model focused on the objective.
- **Safety Rails**: Define a "Max Loops" or "Max Tokens" budget to prevent runaway agentic loops and ensure cost predictability.

### Use Case: Automated Research
An agent is tasked with "Find the latest news on MCP and summarize it in Slack." It uses a "Web Search" tool to find articles, a "Browser" tool to read them, and finally a "Slack" tool to post the summary.

---

## 3. HITL (Human-in-the-Loop Validation)
The **HITL Node** represents a pause in the automated flow, waiting for an external human signal. This is critical for enterprise workflows that require oversight or expert judgment.

### Enterprise Features
- **Dynamic Form Engine**: Define exactly what fields a human should interact with using standard JSON Schema. The Phaxa Inbox will render these forms automatically.
- **Durable Suspension**: Thanks to Temporal, a workflow can wait for weeks for a human response without consuming any active compute or memory resources on the server.
- **Expert Routing**: Assign tasks to specific users, departments, or "Expert" roles. For example, a "Legal Approval" step can be routed specifically to the legal department group.
- **Blackboard Injection**: Data entered by the human is merged back into the Blackboard, allowing the next automated node to use that data immediately.

### Use Case: Expense Approval
A workflow processes an invoice. If the amount is > $1000, it enters a HITL state. A manager receives a notification in their Phaxa Inbox, reviews the data, and clicks "Approve" or "Reject," providing a reason that is then used in the next step.

---

## 4. Control Flow & Logic Nodes
These nodes provide the "glue" that binds reasoning steps together into a coherent business process.

### Switch Node (Conditional Branching)
The **Switch Node** evaluates a logical expression (using the `expr` engine) against the Blackboard.
- **Example**: `blackboard.amount > 5000` -> Path A, else -> Path B.
- Supports complex logic, including string manipulation and array operations.

### Parallel Node (Concurrency)
Execute multiple branches simultaneously.
- **Fan-out**: Trigger "Legal Review" and "Risk Analysis" at the same time.
- **Join/Sync**: The workflow waits for both branches to complete before moving to the final "Contract Generation" step.

### Terminal Node
The final state of any workflow. It cleans up ephemeral resources, records the final outcome, and can trigger downstream webhooks or notifications.
