.PHONY: dev infra stop server worker frontend migrate tidy build docker-build stack stack-down

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
