.PHONY: dev infra stop server worker frontend migrate tidy

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

# Install frontend deps
frontend-install:
	cd frontend && npm install
