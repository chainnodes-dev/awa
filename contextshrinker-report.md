# ContextShrinker Architecture Health Report

Generated on: Thu, 28 May 2026 16:43:00 CEST
Target Workspace: `/Users/egongemmel/dev/claude/asm-platform`

---

## 1. God Objects (High Outbound Coupling)
Identifies classes/structs containing the highest count of fields, functions, or methods, indicating poor separation of concerns.

| Class/Struct | File | Outbound Containment Count |
|---|---|---|
| `StateDef` | `pkg/asmtypes/workflow.go` | 26 |
| `Config` | `internal/config/config.go` | 25 |
| `step1Response` | `internal/designer/pipeline.go` | 22 |
| `MemoryStore` | `internal/store/memory.go` | 19 |
| `HITLRequest` | `pkg/asmtypes/runtime.go` | 15 |
| `MCPServer` | `internal/store/mcp_server.go` | 13 |
| `MarketEntry` | `internal/mcp/marketplace.go` | 13 |
| `Handlers` | `internal/api/handlers.go` | 13 |
| `MCPAuditLog` | `pkg/asmtypes/workflow.go` | 13 |
| `WorkflowRun` | `pkg/asmtypes/runtime.go` | 12 |

## 2. Inbound Coupling hotspots (Black Holes)
Identifies functions called heavily from elsewhere in the codebase, highlighting critical shared dependencies.

| Function | File | Inbound Call Count |
|---|---|---|
| `TenantIDFromContext` | `internal/store/interfaces.go` | 68 |
| `col` | `scripts/load_test.py` | 36 |
| `do` | `internal/api/handlers_test.go` | 35 |
| `NewMemoryStore` | `internal/store/memory.go` | 28 |
| `Publish` | `internal/events/bus.go` | 28 |
| `getEnv` | `internal/config/config.go` | 25 |
| `jsonBody` | `internal/api/handlers_test.go` | 22 |
| `testRouter` | `internal/api/handlers_test.go` | 21 |
| `Complete` | `internal/executor/executor_test.go` | 18 |
| `New` | `internal/events/types.go` | 17 |

## 3. Cyclic Import Dependencies
Identifies circular import paths (up to depth 5) that violate clean dependency hierarchy rules.

✅ **Clean!** *No cyclic import dependencies detected.*

## 4. Unused Private/Unexported Functions
Identifies unexported functions that have zero inbound callers.

| Unused Function | File |
|---|---|
| `main` | `cmd/license-gen/main.go` |
| `main` | `cmd/mcp-exchange/main.go` |
| `main` | `cmd/mcp-gleif/main.go` |
| `main` | `cmd/mcp-openfigi/main.go` |
| `main` | `cmd/mcp-server-template/main.go` |
| `main` | `cmd/server/main.go` |
| `main` | `cmd/specialist-worker/main.go` |
| `main` | `cmd/specialist-worker-template/main.go` |
| `main` | `cmd/worker/main.go` |
| `init` | `internal/api/handlers_test.go` |
| `main` | `scratch/gen_license.go` |

