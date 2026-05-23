# API Reference

The Chain Nodes API is a RESTful interface for managing workflows, runs, and configurations.

## Authentication
All requests (except `/auth/login`) require a Bearer token or an X-API-Key.

```bash
Authorization: Bearer <token>
# OR
X-API-Key: chain nodes_sk_...
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

## Enterprise Management
The following endpoints require a valid license key with specific feature flags enabled.
- `GET /api/v1/enterprise/status`: Retrieve the active license tier and granted features.
- `GET /api/v1/enterprise/branding`: Get custom tenant branding (requires `branding` feature).
- `POST /api/v1/enterprise/branding`: Update custom tenant branding (requires `branding` feature).
- `GET /api/v1/enterprise/secrets`: List configured environment secrets (requires `secrets` feature).
- `POST /api/v1/enterprise/secrets`: Update environment secrets (requires `secrets` feature).
- `GET /api/v1/enterprise/audit-logs`: Fetch security and audit trails (requires `audit_logs` feature).
- `GET /api/v1/enterprise/analytics`: Retrieve performance analytics (requires `analytics` feature).
- `PUT /api/v1/enterprise/oidc`: Configure external SSO/OIDC providers (requires `sso` feature).
