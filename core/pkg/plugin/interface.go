// Package plugin is a Phase 2 placeholder that defines the Plugin interface
// and related types for the Maintify plugin system.
// None of the types in this package are used in production yet;
// they will be wired up when the first real plugin is implemented.
package plugin

import (
	"context"
	"net/http"
)

// Plugin represents the core interface that all Maintify plugins must implement
type Plugin interface {
	// GetMetadata returns plugin metadata
	GetMetadata() Metadata

	// Initialize sets up the plugin with configuration
	Initialize(config Config) error

	// Start begins the plugin's operation (e.g., start HTTP server)
	Start(ctx context.Context) error

	// Stop gracefully shuts down the plugin
	Stop(ctx context.Context) error

	// Health returns the current health status of the plugin
	Health() HealthStatus

	// GetRoutes returns HTTP routes that this plugin exposes
	GetRoutes() []Route
}

// Metadata contains plugin identification and configuration
type Metadata struct {
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description" yaml:"description"`
	Author      string `json:"author,omitempty" yaml:"author,omitempty"`

	// Service configuration
	BackendURL  string `json:"backend_url" yaml:"backend_url"`
	FrontendURL string `json:"frontend_url,omitempty" yaml:"frontend_url,omitempty"`
	Schema      string `json:"schema,omitempty" yaml:"schema,omitempty"`
	Route       string `json:"route" yaml:"route"`

	// Dependencies
	DependsOn []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`

	// Resource requirements
	Resources Resources `json:"resources,omitempty" yaml:"resources,omitempty"`

	// Capabilities
	Capabilities []string `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
}

// Resources defines plugin resource requirements
type Resources struct {
	MemoryMB      int `json:"memory_mb,omitempty" yaml:"memory_mb,omitempty"`
	CPUMilliCores int `json:"cpu_milli_cores,omitempty" yaml:"cpu_milli_cores,omitempty"`
}

// Config contains plugin configuration from environment or config files
type Config struct {
	Environment map[string]string      `json:"environment,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// HealthStatus represents the current state of a plugin
type HealthStatus struct {
	Status  string `json:"status"` // "healthy", "unhealthy", "starting", "stopping"
	Message string `json:"message,omitempty"`
	Uptime  int64  `json:"uptime,omitempty"` // seconds since start
}

// Route defines an HTTP route exposed by the plugin
type Route struct {
	Path       string                            `json:"path"`
	Method     string                            `json:"method"`
	Handler    http.HandlerFunc                  `json:"-"`
	Middleware []func(http.Handler) http.Handler `json:"-"`
}

// HookHandler defines the interface for plugins that want to receive hooks
type HookHandler interface {
	// HandleHook processes incoming hook events
	HandleHook(event string, payload string) error

	// RegisterHooks returns list of events this plugin wants to receive
	RegisterHooks() []string
}

// DatabasePlugin defines the interface for plugins that manage their own databases
type DatabasePlugin interface {
	Plugin

	// GetDatabaseConfig returns database connection configuration
	GetDatabaseConfig() DatabaseConfig

	// Migrate runs database migrations
	Migrate(ctx context.Context) error

	// GetSchemaVersion returns current database schema version
	GetSchemaVersion() string
}

// DatabaseConfig defines database connection parameters
type DatabaseConfig struct {
	Type     string            `json:"type"` // "postgres", "mysql", "mongodb", etc.
	Host     string            `json:"host"`
	Port     int               `json:"port"`
	Database string            `json:"database"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Options  map[string]string `json:"options,omitempty"`
}

// PluginRegistry interface for registering and managing plugins
type Registry interface {
	// Register adds a plugin to the registry
	Register(plugin Plugin) error

	// Unregister removes a plugin from the registry
	Unregister(name string) error

	// Get retrieves a plugin by name
	Get(name string) (Plugin, error)

	// List returns all registered plugins
	List() []Plugin

	// GetByCapability returns plugins that have a specific capability
	GetByCapability(capability string) []Plugin
}

// Standard capabilities that plugins can declare
const (
	CapabilityAuthentication = "authentication"
	CapabilityAuthorization  = "authorization"
	CapabilityDataStorage    = "data_storage"
	CapabilityNotifications  = "notifications"
	CapabilityReporting      = "reporting"
	CapabilityIntegration    = "integration"
	CapabilityUI             = "ui"
	CapabilityAPI            = "api"
)

// Standard health statuses
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusUnhealthy = "unhealthy"
	HealthStatusStarting  = "starting"
	HealthStatusStopping  = "stopping"
	HealthStatusUnknown   = "unknown"
)
