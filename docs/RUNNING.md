# Running Phaxa

This guide describes how to start and operate the Phaxa platform in different environments.

## Prerequisites

Before starting, ensure you have the following installed:
- **Docker & Docker Compose** (for infrastructure and deployment)
- **Go 1.22+** (for manual backend development)
- **Node.js 18+ & npm** (for manual frontend development)
- **psql** (optional, for database inspection)

---

## Configuration

The platform is configured via environment variables. Copy the example file and adjust as needed:

```bash
cp .env.example .env
```

Key variables:
- `DATABASE_URL`: Postgres connection string.
- `TEMPORAL_ADDRESS`: Address of the Temporal service (default `localhost:7233`).
- `LITE_MODE`: Set to `true` to run without Temporal (using `internal/orchestrator/direct`).
- `LLM_PROVIDER`: Default LLM provider (e.g., `google`, `openai`, `ollama`).

### Mode Selection
The orchestrator can run in two modes controlled by `EXECUTOR_MODE`:
- `temporal` (default): Uses Temporal for workflow persistence and state management. Requires the Temporal server.
- `direct`: Runs workflows in-process. Useful for quick local testing without Temporal.

---

## 1. Development Mode (Manual Startup)

This mode is recommended for active development. You start the storage and orchestration dependencies in Docker, but run the Phaxa services manually for faster feedback loops and easier debugging.

### Step 1: Start 3rd-Party Infrastructure
Start Postgres, Redis, and Temporal:

```bash
make infra
```

This command starts:
- **Postgres** (Port 5433)
- **Redis** (Port 6379)
- **Temporal Server** (Port 7233)
- **Temporal Web UI** (Port 8088)

### Step 2: Start Backend Services
In separate terminal windows (or using a multiplexer like `tmux`), start the server and worker:

```bash
# Start API Server
make server

# Start Temporal Worker
make worker
```

### Step 3: Start Frontend
Start the Vite development server:

```bash
make frontend
```
The frontend will be available at [http://localhost:5173](http://localhost:5173).

---

## 2. Deployment Mode (Full Stack Docker)

This mode is recommended for demos, final testing, or deployment to other machines. It starts the entire stack (infrastructure + app + frontend) using containers.

### Start the Stack
```bash
make stack
```

This will:
1. Build the backend binaries into Docker images.
2. Build the frontend into a production-ready Nginx container.
3. Start all services in the background.

### Accessing the Platform
- **Main App**: [http://localhost:80](http://localhost:80)
- **Temporal UI**: [http://localhost:8088](http://localhost:8088)
- **API (Internal)**: [http://localhost:8080](http://localhost:8080) (if proxied)

### Management Commands
- **Check Logs**: `docker compose logs -f`
- **Stop Stack**: `make stack-down`
- **Rebuild**: `make stack` (uses `--build` automatically)

---

## 3. Production Mode (Pre-built Images)

For stable deployments or shared environments where you don't want to build from source, use the **Production Stack**. This pulls pre-built, versioned images directly from the GitHub Container Registry (GHCR).

### Start Production Stack
```bash
docker compose -f docker-compose.prod.yaml up -d
```

### Advantages
- **Faster Startup**: No local compilation or `npm build` required.
- **Consistency**: Uses exact binary versions released via GitHub tags.
- **Clean Environment**: No Go or Node.js toolchains needed on the host machine.

> [!NOTE]
> Production images are automatically published to `ghcr.io` whenever a new version tag (e.g. `v1.0.0`) is pushed to the repository.

---

## Verification & Troubleshooting

### Database Troubleshooting
If migrations fail to apply, you can manually reset the database (DANGER):
```bash
docker compose down -v
make infra
```

### Temporal Connectivity
If the worker cannot connect to Temporal, ensure `TEMPORAL_ADDRESS` in `.env` is set to `localhost:7233` for manual runs, or `temporal:7233` when running inside Docker.

### Tooling
- **Makefile**: Reference the `Makefile` in the root for a full list of helper targets.
- **Docker Desktop**: Use the dashboard to monitor service health and resource usage.

---

## Uninstall & Cleanup

To completely remove Phaxa and its associated data (Postgres, Redis, Temporal) from your machine, follow these steps.

### 1. Stop and Remove Containers
Stop all Phaxa services and dependencies:
```bash
docker compose down
```

### 2. Wipe Persistent Data (DANGER)
If you want to delete all your workflows, users, and execution history, remove the Docker volumes:
```bash
docker compose down -v
```

### 3. Cleanup Docker Images
To free up disk space by removing the Phaxa images:
```bash
docker rmi $(docker images 'phaxa/*' -q)
```

### 4. Remove Configuration
Optionally delete your local environment configuration:
```bash
rm .env
```
