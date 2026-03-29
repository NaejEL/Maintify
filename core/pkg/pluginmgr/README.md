# Plugin Manager

This package handles dynamic discovery, loading, registration, and unregistration of plugins at runtime. Plugins are loaded from the `/plugins` directory and can be cloned from Git repositories.

## Key Features
- No hardcoded plugin names or APIs
- Supports runtime plugin registration/unregistration
- Discovers plugins by reading and validating `plugin.yaml` metadata
- Detects invalid metadata and plugin conflicts (name/route)
- Provides REST APIs for plugin metadata and lifecycle control

## API Endpoints
- `GET /api/plugins` — list discovered plugins
- `GET /api/plugins/status` — list runtime lifecycle statuses
- `POST /api/plugins/{name}/start` — start a plugin container stack
- `POST /api/plugins/{name}/stop` — stop a plugin container stack
- `POST /api/plugins/{name}/restart` — restart a plugin container stack

All endpoints are protected by the existing RBAC middleware and require `system.admin`.
