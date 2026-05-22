# Deploying Phaxa on a MacBook

## Prerequisites (one-time)

**1. Install Docker Desktop**
→ https://www.docker.com/products/docker-desktop/
Download the Apple Silicon (M1/M2/M3) or Intel version as appropriate. Install, launch it, and wait for the whale icon in the menu bar to stop animating.

That's the only prerequisite. No Go, no Node, no Temporal — Docker handles everything.

---

## Deploy

**2. Clone the repo**
```bash
git clone https://github.com/chainnodes-dev/awa
cd awa
```

**3. Create the config file**
```bash
cp .env.example .env
```

Open `.env` in any editor and set these two lines — everything else has safe defaults:
```bash
ANTHROPIC_API_KEY=sk-ant-...    # or OPENAI_API_KEY=sk-...
ADMIN_PASSWORD=something-safe
```

**4. Start everything**
```bash
docker compose up -d
```

First run pulls ~1 GB of images and builds the Go + frontend containers — takes 3–5 minutes. Subsequent starts take about 10 seconds.

**5. Wait for Temporal to be ready**
```bash
docker compose logs -f temporal
```
Wait until you see `Starting frontend server` — that means Temporal is fully up. Press `Ctrl+C` to stop following logs.

**6. Open the app**
→ **http://localhost**
Log in with `admin` / the password you set in `.env`.

---

## Useful commands

```bash
# Check all containers are running
docker compose ps

# Follow all logs
docker compose logs -f

# Follow a specific service
docker compose logs -f server
docker compose logs -f worker

# Stop everything (keeps data)
docker compose down

# Stop everything AND wipe the database
docker compose down -v

# Pull latest code and rebuild
git pull
docker compose up --build -d
```

---

## Ports used

| Port | Service | Note |
|---|---|---|
| `80` | Frontend (Nginx) | Main entry point |
| `8088` | Temporal UI | Workflow execution inspector |
| `5433` | Postgres | External access for DB tools |
| `6379` | Redis | External access if needed |
| `7233` | Temporal gRPC | External access if needed |

> If port 80 is already taken (e.g. Apache), add `FRONTEND_PORT=8080` to `.env` and access the app at `http://localhost:8080` instead.

---

## Troubleshooting

**Containers keep restarting**
```bash
docker compose logs server   # usually a missing env var
```

**"connection refused" on http://localhost**
```bash
docker compose ps   # check frontend container is Up, not Restarting
```

**Temporal not healthy after 2+ minutes**
```bash
docker compose down -v        # wipe state
docker compose up -d          # fresh start
```
This can happen if Postgres wasn't fully ready the first time Temporal tried to initialise its schema.

**Want to use a local Ollama instead of a cloud API key**

If Ollama is running on the MacBook, add this to `.env`:
```bash
OLLAMA_URL=http://host.docker.internal:11434
LLM_PROVIDER=ollama
OLLAMA_MODEL=llama3.2
```
`host.docker.internal` is Docker's built-in hostname that resolves to the Mac's localhost from inside a container.
