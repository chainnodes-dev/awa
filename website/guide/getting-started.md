# Getting Started

::: info LICENSE INFORMATION
Chain Nodes is distributed under the **Chain Nodes Enterprise License**. The core platform is free for individual development and small-scale automation, while advanced features like Multi-Tenancy, SSO, and Fleet Management require a valid Enterprise License Key. By using this software, you agree to the terms of the Chain Nodes EULA.
:::

Chain Nodes is a distributed, high-performance system consisting of a Go Backend, a Vue Frontend, and a set of Workers. It is designed to orchestrate complex, long-running agentic workflows with the reliability of Temporal.io.

## Prerequisites

- **Go 1.22+**
- **Node.js 20+**
- **Docker & Docker Compose** (for infrastructure)
- **Temporal.io** (running locally or via Docker)

## Installation

The easiest way to get Chain Nodes up and running is via **Docker Compose**. This will start the API server, the worker, the frontend, and all necessary infrastructure (PostgreSQL, Redis, Temporal).

1. **Clone the repository**:
   ```bash
   git clone https://github.com/chainnodes-dev/awa
   cd awa
   ```

2. **One-Command Start**:
   ```bash
   docker compose up -d
   ```

3. **Access the Platform**:
   - **Frontend**: [http://localhost:5174](http://localhost:5174)
   - **Temporal UI**: [http://localhost:8088](http://localhost:8088)

> [!NOTE]
> On the first startup, it may take up to 60 seconds for Temporal to initialize. You can monitor progress with `docker compose logs -f temporal`.

Once the platform is running, navigate to `http://localhost:5174`. 

On the first boot, Chain Nodes will prompt you to initialize the **Super Admin** account. 
1. **Choose a username and a secure password** for the super admin account.
2. **Download your Recovery Key** (used if you lose access to the database).
3. **Log in** with the credentials you just created.

After logging in, you will be taken to the Global Dashboard where you can manage tenants, configure LLM providers, and monitor the system health.
 
---
## ⚙️ Mandatory: Define your first LLM

Before you can create or run your first workflow, you **must** define at least one LLM provider in the UI. While you can pre-configure them via the `.env` file (which Chain Nodes will automatically seed into the database on first boot), we recommend managing them through the **Settings > LLM Providers** screen for better flexibility.

Chain Nodes supports a wide range of providers. You will need an API key from at least one of these:

| Provider | Best For | Link |
| :--- | :--- | :--- |
| **Anthropic** | Complex reasoning & long context | [Console](https://console.anthropic.com/) |
| **OpenAI** | General purpose (GPT-4o) | [Platform](https://platform.openai.com/) |
| **Ollama** | Local, private execution | [Download](https://ollama.com/) |
| **Google Gemini** | Large context windows | [AI Studio](https://aistudio.google.com/) |
| **DeepSeek** | High-performance open models | [Platform](https://platform.deepseek.com/) |
| **xAI Grok** | Real-time world knowledge | [Console](https://console.x.ai/) |

Once a provider is added and set as "Active," the **Designer** and your **Agent States** will be ready for use.
 ## Your First Workflow
 
 Navigate to the **Designer** tab. Describe a task like "Analyze a PDF and store the results in Google Sheets" to see the Designer scaffold a workflow in real-time.
