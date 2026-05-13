# API Reference

The Phaxa API is a RESTful interface for managing workflows, runs, and configurations.

## Authentication
All requests (except `/auth/login`) require a Bearer token or an X-API-Key.

```bash
Authorization: Bearer <token>
# OR
X-API-Key: phaxa_sk_...
```

## Workflows
- `GET /api/v1/workflows`: List all workflows.
- `POST /api/v1/workflows`: Create/Update a workflow (YAML).
- `GET /api/v1/workflows/:name`: Get a specific workflow definition.

## Runs
- `POST /api/v1/runs`: Start a new workflow run.
- `GET /api/v1/runs/:id`: Get the status and history of a run.
- `POST /api/v1/runs/:id/signal`: Send a signal (e.g., HITL approval) to a running workflow.

## File Uploads
- `POST /api/v1/uploads`: Upload a file (multipart or base64).
  - Returns `file_id`.
  - Use this ID in the blackboard for `file` type fields.

## External Invocation (Webhooks)
- `POST /api/v1/invoke/:name`: Start a workflow by name and return the run result.
