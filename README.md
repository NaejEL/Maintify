# Maintify

Maintify is a plugin-based Computerized Maintenance Management System (CMMS) for organizations that need fine-grained access control, structured audit trails, and an extensible plugin architecture. It is self-hosted and runs on Docker Compose.

**Status:** Core infrastructure phase complete. Not yet production-ready.

---

## Table of contents

- [Services](#services)
- [Quick start](#quick-start)
- [What is implemented](#what-is-implemented)
- [API overview](#api-overview)
- [Architecture](#architecture)
- [Local development](#local-development)
- [Security](#security)
- [Contributing](#contributing)

---

## Services

| Service | Description | Port |
|---------|-------------|------|
| core | Main API — RBAC, logging, plugin lifecycle | 8080 |
| builder | Docker image builder for plugins | 8081 |
| web | React frontend (dev/demo) | 3000 |
| db | PostgreSQL 15 | 5432 |
| redis | Redis 7 (inter-plugin event bus) | 6379 |

---

## Quick start

Requires Docker and Docker Compose.

```bash
git clone <repository>
cd Maintify

cp .env.example .env
# Fill in DB_PASSWORD, JWT_SECRET, BUILDER_API_KEY

docker compose up --build
```

The core service applies database migrations automatically on startup.

**Endpoints:**

- Core API: `http://localhost:8080`
- Health check: `http://localhost:8080/health`
- Frontend: `http://localhost:3000`
- Builder API: `http://localhost:8081`

**Default credentials** (created on first run):

- Email: `admin@maintify.com`
- Password: `admin123`
- Organization: `system`

---

## What is implemented

### Authentication and RBAC

JWT-based authentication with a multi-tenant RBAC system:

- Organization-level tenant isolation — each org sees only its own data
- Role and permission model with resource-level access control
- Time-based role assignments (scheduled access windows)
- Emergency break-glass access with escalation workflows and audit trail
- All RBAC operations are audit-logged with actor, timestamp, and context

Database schema: 16 tables, 3 views, 6 stored functions (PostgreSQL 15).

### Logging service

Structured log ingestion, storage, and search:

- HTTP API for log ingestion from services and plugins
- Full-text search with filtering by level, component, organization, time range, and request ID
- RBAC-gated endpoints — organization-scoped log access
- Structured logger with file and console output, bridged to the database

### Plugin manager

Plugin lifecycle management via Docker Compose:

- Plugin discovery from a configured directory — validates `plugin.yaml` metadata, detects naming conflicts
- Start, stop, and restart lifecycle commands via HTTP API
- Health monitoring with configurable HTTP probes and timeouts
- Diagnostics endpoint aggregating runtime status and health results for all plugins
- All lifecycle endpoints require `system.admin` permission

### Builder service

On-demand Docker image builds for plugin development:

- Accepts a build context via HTTP, produces a tagged image
- API key authentication
- Image cleanup to reclaim disk space

### Reference plugin

A minimal example plugin at `plugins/asset-tracker/`:

- REST CRUD API for physical assets (name, category, location)
- Demonstrates the `plugin.yaml` contract and Dockerfile conventions

---

## API overview

Full specification: [docs/openapi.yaml](docs/openapi.yaml)

### Health

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Overall health |
| GET | /health/live | Liveness probe |
| GET | /health/ready | Readiness probe |

### Plugin lifecycle

All plugin endpoints require `system.admin` permission.

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/plugins | List discovered plugins |
| GET | /api/plugins/status | Runtime status for all plugins |
| GET | /api/plugins/diagnostics | Status and health aggregated |
| POST | /api/plugins/{name}/start | Start a plugin |
| POST | /api/plugins/{name}/stop | Stop a plugin |
| POST | /api/plugins/{name}/restart | Restart a plugin |

---

## Architecture

```
                   HTTP
Client ─────────────────> core (Go, :8080)
                               |
             ┌─────────────────┼─────────────────┐
             v                 v                 v
       PostgreSQL 15      Redis 7           Plugins
       (RBAC, logging,    (event bus,       (Docker Compose
        audit trail)       hooks)            per plugin)
                                                  |
                                            builder (:8081)
                                            (builds plugin images)
```

Each plugin runs as an isolated Docker Compose stack. The core service communicates with plugins over HTTP using the `backend_url` registered in `plugin.yaml`. Plugins access RBAC and logging through the core API.

---

## Local development

Requires Go 1.24+ and Docker.

```bash
# Run all tests with the race detector
cd core && go test -race ./...

# Static analysis
go vet ./...
staticcheck ./...
gosec -quiet ./...
govulncheck ./...

# Reference plugin tests
cd plugins/asset-tracker && go test -race ./...
```

Install pre-commit hooks:

```bash
pip install pre-commit
pre-commit install
```

CI runs on every pull request and push to `main`. See `.github/workflows/` for the full pipeline.

---

## Security

- All containers run as non-root users
- RBAC enforces organization-level tenant isolation on every API call
- Audit trail records all access-control decisions
- JWT tokens scoped to organization
- CI runs Gitleaks (secret scanning), Trivy (filesystem vulnerability scan), and dependency review on every pull request
- `go vet`, `staticcheck`, `gosec`, and `govulncheck` run against all modules in CI
- Dependabot keeps Go modules, Docker base images, and GitHub Actions versions up to date

---

## Contributing

All contributions must follow the TDD workflow and pass the quality gates enforced in CI:

- Tests written before implementation
- `go test -race ./...` passes
- `go vet`, `staticcheck`, `gosec`, `govulncheck` clean
- No secrets committed (Gitleaks enforced via pre-commit hook and CI)
- PR security checklist completed (enforced by `pr-security-checklist` workflow)

See [ROADMAP.md](ROADMAP.md) for the development plan.

---

## License

To be determined.
