# Getting Started

Phaxa is a distributed system consisting of a Go Backend, a Vue Frontend, and a set of Workers.

## Prerequisites

- **Go 1.22+**
- **Node.js 20+**
- **Docker & Docker Compose** (for infrastructure)
- **Temporal.io** (running locally or via Docker)

## Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/egongemmel/asm-platform.git
   cd asm-platform
   ```

2. **Start Infrastructure**:
   Phaxa requires PostgreSQL, Redis, and Temporal.
   ```bash
   docker compose -f docker-compose.infra.yaml up -d
   ```

3. **Configure Environment**:
   Copy `.env.example` to `.env` and add your LLM API keys.
   ```bash
   cp .env.example .env
   ```

4. **Start the Platform**:
   In separate terminals, run:
   ```bash
   # Terminal 1: Backend
   go run ./cmd/server

   # Terminal 2: Worker
   go run ./cmd/worker

   # Terminal 3: Frontend
   cd frontend && npm run dev
   ```

## Your First Workflow

Navigate to `http://localhost:5173`, log in with `admin/pwd`, and click on **Generator**. Describe a task like "Analyze a PDF and store the results in Google Sheets" to see the engine in action.
