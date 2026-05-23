.PHONY: dev infra stop server worker frontend migrate tidy build docker-build stack stack-down k8s-apply k8s-delete k8s-status specialist

# Start infrastructure only (for development)
dev-infra:
	docker compose -f docker-compose.dev.yaml up -d postgres redis temporal temporal-ui
	@echo "Infrastructure is ready. You can now start the server, worker, and frontend manually."

# Stop all infrastructure
stop:
	docker-compose down

# Reset the database (empty all tables)
reset-db:
	@DB_URL=$$(grep -E '^(DATABASE_URL|DB_URL)=' .env | head -n 1 | cut -d '=' -f 2-); \
	if [ -z "$$DB_URL" ]; then DB_URL="postgres://phaxa:phaxa_secret@localhost:5433/phaxa?sslmode=disable"; fi; \
	psql "$$DB_URL" -c "TRUNCATE TABLE users, tenants, workflow_definitions, workflow_runs, state_transitions, hitl_requests, mcp_servers, llm_configs, api_keys, system_settings, audit_logs CASCADE;"

# Run the API server
server:
	go run ./cmd/server

# Run the Temporal worker
worker:
	go run ./cmd/worker

# Run frontend dev server
frontend:
	cd frontend && npm run dev

# Run server + frontend concurrently (requires: brew install tmux)
dev:
	tmux new-session -d -s asm -n server 'make server' \; \
		new-window -n worker 'make worker' \; \
		new-window -n frontend 'make frontend' \; \
		attach-session -t asm

# Apply DB migrations manually (auto-applied via docker-compose initdb)
migrate:
	psql $$DATABASE_URL -f migrations/001_init.sql

# Tidy go modules
tidy:
	go mod tidy

# Run tests
test:
	go test ./...

# Build binaries
build:
	go build -o bin/server ./cmd/server
	go build -o bin/worker ./cmd/worker
	go build -o bin/specialist-worker ./cmd/specialist-worker
	go build -o bin/license-gen ./cmd/license-gen

# Build Docker images for all three binaries
docker-build:
	docker build --build-arg CMD=server           -t phaxa/server:latest .
	docker build --build-arg CMD=worker           -t phaxa/worker:latest .
	docker build --build-arg CMD=specialist-worker -t phaxa/specialist-worker:latest .
	cd frontend && docker build -t phaxa/frontend:latest .

# Start full stack (infra + app + frontend) via docker compose.
# This is the recommended way to run for demos / other machines.
# Quick start (Docker — Full Stack):
#   cp .env.example .env
#   # Set ANTHROPIC_API_KEY (or OPENAI_API_KEY) and ADMIN_PASSWORD below, then:
#   docker compose up -d
#   # Open http://localhost:5174
#
# Quick start (Local — Developer Mode):
#   make dev-infra
#   # In separate terminals:
#   make server
#   make worker
#   make frontend
#   # Open http://localhost:5173
stack:
	docker compose -f docker-compose.yaml up --build -d
	@echo ""
	@echo "  Phaxa Platform is starting (Full Stack)."
	@echo "  Frontend  →  http://localhost:5174"
	@echo "  Temporal  →  http://localhost:8088"
	@echo ""

# Stop all Phaxa-related containers
stack-down:
	docker compose down

# Install frontend deps
frontend-install:
	cd frontend && npm install

# ── Kubernetes ────────────────────────────────────────────────────────────────
K8S_DIR := deploy/k8s

# Apply all manifests in dependency order.
# Edit secret.yaml (or use external-secrets) before running this.
k8s-apply:
	kubectl apply -f $(K8S_DIR)/namespace.yaml
	kubectl apply -f $(K8S_DIR)/configmap.yaml
	kubectl apply -f $(K8S_DIR)/secret.yaml
	kubectl apply -f $(K8S_DIR)/server-deployment.yaml
	kubectl apply -f $(K8S_DIR)/server-service.yaml
	kubectl apply -f $(K8S_DIR)/worker-deployment.yaml
	kubectl apply -f $(K8S_DIR)/specialist-worker-deployment.yaml
	kubectl apply -f $(K8S_DIR)/hpa.yaml

# Delete all Phaxa resources (leaves the namespace intact by default).
k8s-delete:
	kubectl delete -f $(K8S_DIR)/hpa.yaml --ignore-not-found
	kubectl delete -f $(K8S_DIR)/specialist-worker-deployment.yaml --ignore-not-found
	kubectl delete -f $(K8S_DIR)/worker-deployment.yaml --ignore-not-found
	kubectl delete -f $(K8S_DIR)/server-service.yaml --ignore-not-found
	kubectl delete -f $(K8S_DIR)/server-deployment.yaml --ignore-not-found
	kubectl delete -f $(K8S_DIR)/secret.yaml --ignore-not-found
	kubectl delete -f $(K8S_DIR)/configmap.yaml --ignore-not-found

# Show pod / deployment status in the asm namespace.
k8s-status:
	kubectl get deployments,pods,services,hpa -n asm

# Scaffold a new specialist worker binary from the template.
# Usage: make specialist name=invoice-worker queue=invoice-workers
specialist:
	@test -n "$(name)" || (echo "Usage: make specialist name=<binary-name> queue=<task-queue>"; exit 1)
	@test -n "$(queue)" || (echo "Usage: make specialist name=<binary-name> queue=<task-queue>"; exit 1)
	cp -r cmd/specialist-worker-template cmd/$(name)
	@echo "Created cmd/$(name)/"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Edit cmd/$(name)/main.go — set SPECIALIST_TASK_QUEUE default to '$(queue)'"
	@echo "  2. Rename 'my-agent-name' → your agent name and 'myAgentHandler' → your func name"
	@echo "  3. Add the agent to your workflow YAML with task_queue: $(queue)"
	@echo "  4. Run: go run ./cmd/$(name)"
