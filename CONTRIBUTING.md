# Contributing to Chain Nodes

Thank you for your interest in contributing to the Chain Nodes platform! This document outlines how our repository is structured and how you can get involved.

## Codebase Architecture & Boundaries

Chain Nodes uses a monorepo structure to keep everything coordinated, but we maintain strict boundaries between open-source core components and enterprise extensions.

- **`pkg/`** (Public SDKs and types): This is the public API surface area. Structs, interfaces, and utilities meant to be consumed by custom specialist workers live here. Changes here must be strictly backwards-compatible.
- **`internal/`** (Private Core Engine): The core Chain Nodes runtime, API server, executor, and temporal orchestration logic. This code is open for internal commercial use (under Polyform Perimeter) but is *not* a stable API surface. We refactor this heavily.
- **`internal/enterprise/`** (Enterprise Features): Code gated by license keys (e.g., SSO, advanced RBAC, audit logging). We welcome bug fixes here, but new enterprise features are generally managed by the core team.
- **`frontend/`**: The Vue/Vite-based Designer and Monitor applications.

## How to Contribute

### 1. Find an Issue
Look for issues labeled `good first issue` or `help wanted`. If you want to build a new feature, please open an issue first to discuss the design before writing code.

### 2. Local Development Environment
The easiest way to develop locally is by using the Makefile targets. You will need Go 1.23+, Node 20+, and Docker.

```bash
cp .env.example .env
make dev-infra # Starts Postgres, Redis, and Temporal in Docker
make server    # Runs the Go API server on :8080
make worker    # Runs the Temporal worker
make frontend  # Runs the Vite dev server on :5173
```

### 3. Submitting a Pull Request
- Create a fork and a new branch (`feature/your-feature` or `fix/issue-number`).
- Write clear, concise commit messages.
- Ensure all tests pass.
- Submit the PR! A maintainer will review it and merge it once approved.

## Need Help?
Join our [Discord Community](https://discord.gg/phaxa) to ask questions, showcase what you're building, and talk with the maintainers directly.
