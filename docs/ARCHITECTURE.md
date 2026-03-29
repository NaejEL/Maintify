# Maintify Architecture

Maintify is a modular, plugin-based CMMS orchestrator. The system is designed for security and extensibility, with dynamic discovery and loading of plugins at runtime.

## Core Principles

- **Security**: Plugins are treated as untrusted code. They run in isolated, sandboxed environments with minimal privileges.
- **Modularity**: The `core` service is a lightweight orchestrator. All business logic is implemented in plugins.
- **Dynamic Discovery**: The `core` and `frontend` have no hardcoded knowledge of plugins. They are discovered and loaded at runtime.
- **Polyglot Persistence**: Plugins are free to choose their own data storage technology (e.g., Elasticsearch, InfluxDB) and are responsible for managing it.

## System Components

### 1. Core Service (`/core`)

The `core` service is a Go application that acts as the central nervous system of Maintify. Its primary responsibilities are:
- **Plugin Discovery**: Scans the `/plugins` directory for `plugin.yaml` files at startup.
- **Plugin Orchestration**: Manages the lifecycle of plugin containers. It communicates with the `builder` service to get plugin images and then uses the Docker API to run and manage them.
- **API Gateway**: Provides a central API for the frontend to fetch plugin metadata, routes, and other configuration.
- **Hook System**: Offers an event-driven mechanism for plugins to interact and extend core functionalities. This system is backed by a Redis Pub/Sub message broker for scalability and decoupling.
- **RBAC System**: Implements comprehensive Role-Based Access Control with multi-tenant organization support, dynamic resource hierarchies, and granular permission management. All API endpoints are secured with JWT-based authentication and organization-scoped authorization.

### 2. Builder Service (`/builder`)

The `builder` is a dedicated, isolated Go service responsible for securely building plugin container images.
- **Isolated Environment**: It is the only service with access to the Docker socket for building images, minimizing the attack surface of the main `core` service.
- **API-driven**: It exposes a single `/build` endpoint. The `core` service calls this endpoint, providing the plugin's name and context directory.
- **Build Process**: The `builder` creates a Docker image from the plugin's `Dockerfile` and returns the image name to the `core` service. It does not have access to any databases or application secrets.

### 3. Plugins (`/plugins`)

Each plugin is a self-contained application, typically consisting of a Go backend and a Flutter frontend.
- **Containerization**: Every plugin backend is run in its own Docker container.
- **Restricted Runtimes**: Plugin containers are hardened for security:
  - They run as a **non-root user**.
  - They are subject to **resource limits** (CPU and memory) defined in their `plugin.yaml`.
- **Communication**: Plugins must not communicate directly with each other. All interaction is mediated through the `core` service's Redis-based hook system or public APIs.
- **Data Services (Sidecar Model)**: If a plugin requires a specialized database (e.g., Elasticsearch), it is responsible for defining and managing it as a "sidecar" service via a `docker-compose.plugin.yml` file within its own directory. The `core` service does not manage these specialized databases.

### 4. Frontend (`/frontend`)

The main frontend is a Flutter web application that provides the main user interface.
- **Dynamic UI**: At startup, it fetches the list of available plugins and their frontend entry points from the `core` service's API.
- **Dynamic Routing**: It builds the navigation menu and routes dynamically based on the registered plugins. No plugin UI or routes are hardcoded.

## Infrastructure

- **Docker Compose**: The entire system is orchestrated using Docker Compose. It defines and links all core services: `core`, `builder`, `frontend`, `db`, and `redis`. Plugin-specific services are managed separately.
- **PostgreSQL**: A central PostgreSQL database is provided for general-purpose use by plugins that don't require a specialized data store. The database also hosts the comprehensive RBAC system with multi-tenant organization support, dynamic resource hierarchies, and role management.
- **Redis**: A central Redis instance serves as a high-speed message broker for the hook system.
- **Network**: All services are connected to a shared Docker network, allowing them to communicate via service names (e.g., `http://builder:8081`).

## Security Model

- **RBAC Integration**: All core services are secured using the integrated Role-Based Access Control system. Access to plugins, logs, and administrative functions is controlled through organization-scoped permissions.
- **Multi-tenant Isolation**: Each organization operates in complete isolation with its own users, roles, and resources.
- **JWT Authentication**: Secure token-based authentication with organization context embedded in claims.
- **Plugin Security**: All plugins run as non-root users in isolated containers with resource limits and restricted network access.

For detailed RBAC architecture information, see the [RBAC Architecture Guide](./RBAC_ARCHITECTURE.md).
