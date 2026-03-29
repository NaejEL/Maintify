package plugin

import (
	"context"
	"fmt"
	"maintify/core/pkg/logger"
	"net/http"
	"time"
)

// BasePlugin provides a default implementation of the Plugin interface
// that can be embedded in actual plugin implementations to reduce boilerplate
type BasePlugin struct {
	metadata  Metadata
	config    Config
	startTime time.Time
	isRunning bool
	routes    []Route
	healthMsg string
}

// NewBasePlugin creates a new BasePlugin with the given metadata
func NewBasePlugin(metadata Metadata) *BasePlugin {
	return &BasePlugin{
		metadata:  metadata,
		routes:    make([]Route, 0),
		healthMsg: "Plugin initialized",
	}
}

// GetMetadata returns the plugin metadata
func (p *BasePlugin) GetMetadata() Metadata {
	return p.metadata
}

// Initialize sets up the plugin with configuration
func (p *BasePlugin) Initialize(config Config) error {
	p.config = config
	p.healthMsg = "Plugin initialized"
	logger.Info(fmt.Sprintf("[%s] Plugin initialized", p.metadata.Name))
	return nil
}

// Start begins the plugin's operation
func (p *BasePlugin) Start(ctx context.Context) error {
	p.startTime = time.Now()
	p.isRunning = true
	p.healthMsg = "Plugin running"
	logger.Info(fmt.Sprintf("[%s] Plugin started", p.metadata.Name))
	return nil
}

// Stop gracefully shuts down the plugin
func (p *BasePlugin) Stop(ctx context.Context) error {
	p.isRunning = false
	p.healthMsg = "Plugin stopped"
	logger.Info(fmt.Sprintf("[%s] Plugin stopped", p.metadata.Name))
	return nil
}

// Health returns the current health status
func (p *BasePlugin) Health() HealthStatus {
	status := HealthStatusUnknown
	if p.isRunning {
		status = HealthStatusHealthy
	} else {
		status = HealthStatusUnhealthy
	}

	uptime := int64(0)
	if !p.startTime.IsZero() {
		uptime = int64(time.Since(p.startTime).Seconds())
	}

	return HealthStatus{
		Status:  status,
		Message: p.healthMsg,
		Uptime:  uptime,
	}
}

// GetRoutes returns HTTP routes that this plugin exposes
func (p *BasePlugin) GetRoutes() []Route {
	return p.routes
}

// AddRoute adds a new route to the plugin
func (p *BasePlugin) AddRoute(path, method string, handler http.HandlerFunc) {
	p.routes = append(p.routes, Route{
		Path:    path,
		Method:  method,
		Handler: handler,
	})
}

// AddRouteWithMiddleware adds a new route with middleware
func (p *BasePlugin) AddRouteWithMiddleware(path, method string, handler http.HandlerFunc, middleware ...func(http.Handler) http.Handler) {
	p.routes = append(p.routes, Route{
		Path:       path,
		Method:     method,
		Handler:    handler,
		Middleware: middleware,
	})
}

// GetConfig returns the plugin configuration
func (p *BasePlugin) GetConfig() Config {
	return p.config
}

// SetHealthMessage sets a custom health message
func (p *BasePlugin) SetHealthMessage(msg string) {
	p.healthMsg = msg
}

// IsRunning returns whether the plugin is currently running
func (p *BasePlugin) IsRunning() bool {
	return p.isRunning
}

// HTTPPlugin extends BasePlugin with HTTP server functionality
type HTTPPlugin struct {
	*BasePlugin
	server *http.Server
	port   int
}

// NewHTTPPlugin creates a new HTTP-enabled plugin
func NewHTTPPlugin(metadata Metadata, port int) *HTTPPlugin {
	return &HTTPPlugin{
		BasePlugin: NewBasePlugin(metadata),
		port:       port,
	}
}

// Start begins the HTTP plugin operation
func (p *HTTPPlugin) Start(ctx context.Context) error {
	if err := p.BasePlugin.Start(ctx); err != nil {
		return err
	}

	mux := http.NewServeMux()

	// Register all routes
	for _, route := range p.routes {
		handler := http.Handler(route.Handler)

		// Apply middleware in reverse order
		for i := len(route.Middleware) - 1; i >= 0; i-- {
			handler = route.Middleware[i](handler)
		}

		pattern := fmt.Sprintf("%s %s", route.Method, route.Path)
		mux.Handle(pattern, handler)
	}

	p.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", p.port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info(fmt.Sprintf("[%s] HTTP server starting on port %d", p.metadata.Name, p.port))
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(fmt.Sprintf("[%s] HTTP server error", p.metadata.Name), err)
			p.SetHealthMessage(fmt.Sprintf("HTTP server error: %v", err))
		}
	}()

	return nil
}

// Stop gracefully shuts down the HTTP plugin
func (p *HTTPPlugin) Stop(ctx context.Context) error {
	if p.server != nil {
		if err := p.server.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("[%s] HTTP server shutdown error", p.metadata.Name), err)
		} else {
			logger.Info(fmt.Sprintf("[%s] HTTP server shut down gracefully", p.metadata.Name))
		}
	}

	return p.BasePlugin.Stop(ctx)
}

// GetPort returns the port the HTTP server is running on
func (p *HTTPPlugin) GetPort() int {
	return p.port
}
