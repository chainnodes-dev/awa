# ── Stage 1: Builder ──────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

# Build argument selects which cmd/ entry point to compile.
# Valid values: server | worker | specialist-worker
ARG CMD=server

WORKDIR /app

# Download dependencies first — this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Copy full source and build a statically-linked Linux binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/bin/app \
    ./cmd/${CMD}

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM alpine:3.21

# ca-certificates: required for TLS connections to external APIs
# tzdata: required for correct time-zone handling in workflow timeouts
# nodejs/npm: required to execute JS-based MCP servers (npx)
# python3/py3-pip: required to execute Python-based MCP servers
# bash/curl/gcompat: required for npx wrappers and glibc-compiled native binaries
RUN apk add --no-cache ca-certificates tzdata nodejs npm python3 py3-pip bash curl gcompat \
    && pip install --break-system-packages uv

WORKDIR /app

# Copy the compiled binary from the builder stage.
COPY --from=builder /app/bin/app ./app

# Create data directory for local filesystem MCP servers
RUN mkdir -p ./data

# mcp_registry.yaml is resolved at runtime relative to the project root.
# The server binary has the compile-time path /app/cmd/server/main.go embedded,
# so runtime.Caller resolves the project root to /app — matching this COPY target.
COPY mcp_registry.yaml ./mcp_registry.yaml

EXPOSE 8080

ENTRYPOINT ["/app/app"]
