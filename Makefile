.PHONY: dev infra stop server worker frontend migrate tidy build docker-build stack stack-down k8s-apply k8s-delete k8s-status specialist

# Start infrastructure (postgres, redis, temporal)
infra:
	docker-compose up -d postgres redis temporal temporal-ui
	@echo "Waiting for services..." && sleep 3

# Stop all infrastructure
stop:
	docker-compose down

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

# Build Docker images for all three binaries
docker-build:
	docker build --build-arg CMD=server           -t asm/server:latest .
	docker build --build-arg CMD=worker           -t asm/worker:latest .
	docker build --build-arg CMD=specialist-worker -t asm/specialist-worker:latest .

# Start full stack (infra + app) via docker compose
stack:
	docker compose up --build -d

# Stop full stack
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

# Delete all ASM resources (leaves the namespace intact by default).
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
