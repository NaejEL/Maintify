# Maintify Development Guide

This guide provides instructions for developing and contributing to the Maintify platform, with a strong focus on security and best practices.

## Getting Started

### Prerequisites
- Docker and Docker Compose
- Go (version 1.23 or higher)
- Node.js (version 18 or higher) for frontend development

### Running the Application
To build and run the backend services (core, builder, database, and redis), use the following command. This is the recommended approach for backend development as it skips the unnecessary build of the frontend.

```bash
docker-compose up --build core builder db redis
```

To run the full application including the React frontend:
```bash
docker-compose up --build
```

### RBAC Development and Testing
The system includes a comprehensive RBAC (Role-Based Access Control) system. To test RBAC functionality:

1. **Run RBAC Tests**: Use the comprehensive RBAC test command:
   ```bash
   make rbac-test
   ```

2. **Database Migrations**: The RBAC system includes automatic database migrations:
   ```bash
   make migrate-up    # Apply migrations
   make migrate-down  # Rollback migrations
   ```

3. **Default Admin Setup**: The setup script creates the system organization and prompts you to
   choose your own admin credentials:
   ```bash
   bash scripts/dev-setup.sh
   ```
   There are no hardcoded default passwords. The admin email and password you enter
   are the only credentials that exist.

For detailed RBAC architecture information, see the [RBAC Architecture Guide](./RBAC_ARCHITECTURE.md).

## Plugin Development

### Creating a New Plugin
1.  **Copy the Template**: Start by copying the `plugins/template_plugin` directory to a new directory for your plugin (e.g., `plugins/inventory`).
2.  **Update Metadata**: Modify `plugin.yaml` with your plugin's specific information (name, description, etc.).
3.  **Implement Backend**: Write your Go backend logic in the `backend/` directory.
4.  **Implement Frontend**: Write your React frontend code in the `frontend/` directory.

### Security Requirements for Plugins
Adhering to these security requirements is mandatory for all plugins.

#### 1. Use a Non-Root Docker User
Your plugin's `Dockerfile` must create and use a non-root user to run the application. The `template_plugin/backend/Dockerfile` provides a reference implementation.

#### 2. Configure Resource Limits
Your `plugin.yaml` should specify reasonable resource limits for your plugin's container to prevent it from destabilizing the system.

```yaml
# plugin.yaml
resources:
  memory_mb: 256      # Memory limit in Megabytes
  cpu_milli_cores: 500 # CPU limit in milli-cores (e.g., 500 = 0.5 CPU core)
```

### Data Services and Communication

#### Inter-Plugin Communication
All communication between plugins **must** be asynchronous and mediated through the central Redis-based hook system. Direct API calls or database connections between plugins are strictly forbidden.

#### Specialized Data Stores (Sidecar Model)
If your plugin requires a specialized database (e.g., Elasticsearch, InfluxDB), you are responsible for managing it as a "sidecar" service. 

**📖 For detailed instructions, see the [Plugin Sidecar Pattern Guide](./PLUGIN_SIDECAR_PATTERN.md)**

Quick overview:
1.  Create a `docker-compose.plugin.yml` file in your plugin's root directory.
2.  Define the database service in this file.
3.  Add sidecar configuration to your `plugin.yaml`.
4.  Connect to the sidecar service from your plugin backend.

This model ensures that your plugin is self-contained and does not create a dependency on the `core` infrastructure.

## Testing
-   **Backend**: Run `go test ./...` in the `core` directory and in each plugin's `backend` directory.
-   **Frontend**: Run `npm test` in the `web` directory.
