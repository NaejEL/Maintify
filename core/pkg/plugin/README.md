# Maintify Plugin Interface

This document describes the standard plugin interface that all Maintify plugins must implement for consistency, reliability, and seamless integration with the core system.

## Overview

The Maintify plugin interface provides a standardized way for plugins to:

- Define their metadata and capabilities
- Integrate with the core system
- Handle lifecycle operations (start, stop, health checks)
- Expose HTTP endpoints
- Handle database operations (for data plugins)
- Receive and process hooks

## Core Interface

All plugins must implement the `Plugin` interface:

```go
type Plugin interface {
    GetMetadata() Metadata
    Initialize(config Config) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() HealthStatus
    GetRoutes() []Route
}
```

## Plugin Metadata

Each plugin must define comprehensive metadata:

```go
type Metadata struct {
    Name        string   `json:"name" yaml:"name"`
    Version     string   `json:"version" yaml:"version"`
    Description string   `json:"description" yaml:"description"`
    Author      string   `json:"author,omitempty" yaml:"author,omitempty"`
    
    // Service configuration
    BackendURL  string `json:"backend_url" yaml:"backend_url"`
    FrontendURL string `json:"frontend_url,omitempty" yaml:"frontend_url,omitempty"`
    Schema      string `json:"schema,omitempty" yaml:"schema,omitempty"`
    Route       string `json:"route" yaml:"route"`
    
    // Dependencies and resources
    DependsOn []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
    Resources Resources `json:"resources,omitempty" yaml:"resources,omitempty"`
    
    // Capabilities
    Capabilities []string `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
}
```

## Standard Capabilities

Plugins should declare their capabilities using these standard constants:

- `CapabilityAuthentication` - Provides user authentication
- `CapabilityAuthorization` - Provides access control and permissions
- `CapabilityDataStorage` - Manages data storage and retrieval
- `CapabilityNotifications` - Sends notifications and alerts
- `CapabilityReporting` - Generates reports and analytics
- `CapabilityIntegration` - Integrates with external systems
- `CapabilityUI` - Provides user interface components
- `CapabilityAPI` - Exposes API endpoints

## Base Plugin Implementation

For convenience, plugins can embed `BasePlugin` to get default implementations:

```go
type MyPlugin struct {
    *plugin.BasePlugin
    // Custom fields
}

func NewMyPlugin() *MyPlugin {
    metadata := plugin.Metadata{
        Name:        "my-plugin",
        Version:     "1.0.0",
        Description: "My awesome plugin",
        Capabilities: []string{
            plugin.CapabilityAPI,
            plugin.CapabilityDataStorage,
        },
    }
    
    p := &MyPlugin{
        BasePlugin: plugin.NewBasePlugin(metadata),
    }
    
    // Add routes
    p.AddRoute("/api/my-endpoint", "GET", p.handleMyEndpoint)
    
    return p
}
```

## HTTP Plugins

For plugins that need HTTP servers, use `HTTPPlugin`:

```go
type MyHTTPPlugin struct {
    *plugin.HTTPPlugin
}

func NewMyHTTPPlugin() *MyHTTPPlugin {
    metadata := plugin.Metadata{
        Name: "my-http-plugin",
        // ... other metadata
    }
    
    p := &MyHTTPPlugin{
        HTTPPlugin: plugin.NewHTTPPlugin(metadata, 8090),
    }
    
    p.AddRoute("/api/test", "GET", p.handleTest)
    
    return p
}
```

## Database Plugins

Plugins that manage their own databases should implement `DatabasePlugin`:

```go
type DatabasePlugin interface {
    Plugin
    GetDatabaseConfig() DatabaseConfig
    Migrate(ctx context.Context) error
    GetSchemaVersion() string
}
```

## Hook Handling

Plugins that want to receive hooks should implement `HookHandler`:

```go
type HookHandler interface {
    HandleHook(event string, payload string) error
    RegisterHooks() []string
}
```

## Example Plugin Implementation

```go
package main

import (
    "context"
    "net/http"
    "maintify/core/pkg/plugin"
)

type AuthPlugin struct {
    *plugin.HTTPPlugin
}

func NewAuthPlugin() *AuthPlugin {
    metadata := plugin.Metadata{
        Name:        "auth",
        Version:     "1.0.0",
        Description: "Authentication and authorization plugin",
        BackendURL:  "http://localhost:8091/api",
        Route:       "/auth",
        Capabilities: []string{
            plugin.CapabilityAuthentication,
            plugin.CapabilityAuthorization,
            plugin.CapabilityAPI,
        },
        Resources: plugin.Resources{
            MemoryMB:      512,
            CPUMilliCores: 1000,
        },
    }
    
    p := &AuthPlugin{
        HTTPPlugin: plugin.NewHTTPPlugin(metadata, 8091),
    }
    
    // Add routes
    p.AddRoute("/api/auth/login", "POST", p.handleLogin)
    p.AddRoute("/api/auth/logout", "POST", p.handleLogout)
    p.AddRoute("/api/users", "GET", p.handleListUsers)
    
    return p
}

func (p *AuthPlugin) handleLogin(w http.ResponseWriter, r *http.Request) {
    // Login implementation
}

func (p *AuthPlugin) handleLogout(w http.ResponseWriter, r *http.Request) {
    // Logout implementation
}

func (p *AuthPlugin) handleListUsers(w http.ResponseWriter, r *http.Request) {
    // List users implementation
}

func main() {
    plugin := NewAuthPlugin()
    
    ctx := context.Background()
    
    // Initialize and start
    config := plugin.Config{
        Environment: map[string]string{
            "DATABASE_URL": "postgres://user:pass@db:5432/auth",
        },
    }
    
    if err := plugin.Initialize(config); err != nil {
        log.Fatal(err)
    }
    
    if err := plugin.Start(ctx); err != nil {
        log.Fatal(err)
    }
    
    // Handle shutdown gracefully
    // ... signal handling code
}
```

## Plugin Registration

The core system uses a plugin registry to manage plugins:

```go
registry := plugin.NewSimpleRegistry()

// Register a plugin
err := registry.Register(myPlugin)

// Get a plugin
authPlugin, err := registry.Get("auth")

// List all plugins
allPlugins := registry.List()

// Get plugins by capability
authPlugins := registry.GetByCapability(plugin.CapabilityAuthentication)
```

## Health Checks

All plugins must implement health checks that return current status:

```go
type HealthStatus struct {
    Status  string `json:"status"`  // "healthy", "unhealthy", "starting", "stopping"
    Message string `json:"message,omitempty"`
    Uptime  int64  `json:"uptime,omitempty"` // seconds since start
}
```

## Configuration

Plugins receive configuration through the `Config` struct:

```go
type Config struct {
    Environment map[string]string      `json:"environment,omitempty"`
    Settings    map[string]interface{} `json:"settings,omitempty"`
}
```

## Best Practices

1. **Use Base Implementations**: Embed `BasePlugin` or `HTTPPlugin` to reduce boilerplate
2. **Declare Capabilities**: Always declare what your plugin provides
3. **Handle Graceful Shutdown**: Implement proper cleanup in the `Stop` method
4. **Provide Health Checks**: Return meaningful health status and messages
5. **Follow Naming Conventions**: Use clear, descriptive names for plugins and routes
6. **Document Dependencies**: Clearly specify what your plugin depends on
7. **Set Resource Limits**: Define reasonable memory and CPU requirements
8. **Version Your Plugin**: Use semantic versioning for plugin releases

## Integration with Core

The core system will:

- Automatically discover and register plugins
- Route requests to plugin endpoints
- Monitor plugin health
- Handle plugin lifecycle (start/stop)
- Provide configuration from environment variables
- Deliver hooks to subscribed plugins

This interface ensures all plugins work consistently within the Maintify ecosystem while providing flexibility for diverse plugin functionality.
