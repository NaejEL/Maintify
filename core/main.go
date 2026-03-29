package main

import (
	"context"
	"fmt"
	"maintify/core/pkg/config"
	"maintify/core/pkg/health"
	"maintify/core/pkg/hooks"
	"maintify/core/pkg/logger"
	"maintify/core/pkg/logging"
	"maintify/core/pkg/pluginmgr"
	"maintify/core/pkg/rbac"
	"maintify/core/pkg/registry"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func initDatabase(cfg *config.Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connected successfully", map[string]interface{}{
		"host":     cfg.DBHost,
		"port":     cfg.DBPort,
		"database": cfg.DBName,
	})

	return db, nil
}

func main() {
	// Load configuration first
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logging system first
	logConfig := logger.Config{
		Level:       cfg.LogLevel,
		Component:   "core",
		Structured:  cfg.LogStructured,
		LogDir:      cfg.LogDir,
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		MaxFiles:    5,
		Console:     true,
	}

	err = logger.InitDefaultLogger(logConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.LogSystemEvent("service_start", "Maintify Core service starting", map[string]interface{}{
		"version": "1.0.0",
		"port":    8080,
	})

	logger.Info("[core] main() started")

	// Initialize database connection
	db, err := initDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize database", err)
	}
	defer db.Close()

	// Run database migrations
	migrationService := rbac.NewMigrationService(db, "./migrations")
	err = migrationService.ApplyMigrations()
	if err != nil {
		logger.Fatal("Failed to apply database migrations", err)
	}

	// Initialize RBAC services
	rbacService := rbac.NewPostgreSQLRBACService(db)
	authService := rbac.NewAuthService(rbacService, []byte(cfg.JWTSecret))
	authMiddleware := rbac.NewAuthMiddleware(authService, rbacService)
	rbacHandler := rbac.NewHandler(rbacService, authService)

	// Initialize time-based access processor
	timeBasedProcessor := rbac.NewTimeBasedAccessProcessor(rbacService)
	timeBasedProcessor.Start()
	defer timeBasedProcessor.Stop()

	// Initialize emergency access processor
	emergencyProcessor := rbac.NewEmergencyAccessProcessor(rbacService)
	emergencyProcessorCtx, emergencyProcessorCancel := context.WithCancel(context.Background())
	defer emergencyProcessorCancel()
	emergencyProcessor.Start(emergencyProcessorCtx)

	// Initialize logging service
	loggingService := logging.NewPostgreSQLLogService(db.DB)
	loggingHandler := logging.NewHandler(loggingService)

	// Add database hook to logger
	dbHook := logging.NewDatabaseHook(loggingService)
	logger.AddHook(dbHook)

	logger.Info("Logging service initialized", map[string]interface{}{
		"service": "logging",
		"status":  "ready",
	})

	pluginmgr.Init()
	registry.Init()
	hooks.Init()
	health.Initialize()

	logger.Info("Core components initialized successfully", map[string]interface{}{
		"plugin_manager":    "ready",
		"registry":          "ready",
		"hooks":             "ready",
		"health":            "ready",
		"rbac":              "ready",
		"time_based_access": "ready",
		"emergency_access":  "ready",
		"database":          "connected",
	})

	// Setup HTTP router with RBAC
	router := mux.NewRouter()

	// Public health endpoints
	router.HandleFunc("/health", health.HealthHandler).Methods("GET")
	router.HandleFunc("/health/live", health.LivenessHandler).Methods("GET")
	router.HandleFunc("/health/ready", health.ReadinessHandler).Methods("GET")

	// Setup RBAC routes
	rbacHandler.SetupRoutes(router.PathPrefix("/api/rbac").Subrouter(), authMiddleware)

	// Setup Logging routes
	loggingHandler.SetupRoutes(router.PathPrefix("/api").Subrouter(), authMiddleware)

	// Protected Core API endpoints
	coreAPI := router.PathPrefix("/api").Subrouter()
	coreAPI.Use(authMiddleware.RequireAuth)
	coreAPI.Use(authMiddleware.AuditMiddleware)
	coreAPI.Handle("/plugins", authMiddleware.RequirePermission("system.admin")(http.HandlerFunc(pluginmgr.PluginListHandler))).Methods("GET")
	coreAPI.Handle("/plugins/status", authMiddleware.RequirePermission("system.admin")(http.HandlerFunc(pluginmgr.PluginStatusHandler))).Methods("GET")
	coreAPI.Handle("/plugins/{name}/start", authMiddleware.RequirePermission("system.admin")(http.HandlerFunc(pluginmgr.PluginStartHandler))).Methods("POST")
	coreAPI.Handle("/plugins/{name}/stop", authMiddleware.RequirePermission("system.admin")(http.HandlerFunc(pluginmgr.PluginStopHandler))).Methods("POST")
	coreAPI.Handle("/plugins/{name}/restart", authMiddleware.RequirePermission("system.admin")(http.HandlerFunc(pluginmgr.PluginRestartHandler))).Methods("POST")
	coreAPI.Handle("/plugins/diagnostics", authMiddleware.RequirePermission("system.admin")(http.HandlerFunc(pluginmgr.PluginDiagnosticsHandler))).Methods("GET")

	// Add CORS for development
	router.Use(corsMiddleware)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Create a channel to receive OS signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	// Start server in a goroutine
	go func() {
		logger.Info("HTTP server starting", map[string]interface{}{
			"address":   ":8080",
			"endpoints": []string{"/api/plugins", "/api/plugins/status", "/api/plugins/{name}/start", "/api/plugins/{name}/stop", "/api/plugins/{name}/restart", "/health", "/health/live", "/health/ready"},
		})
		logger.Info("Maintify Core starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", err)
		}
	}()

	// Wait for signal
	<-signalChan
	logger.LogSystemEvent("shutdown_initiated", "Graceful shutdown initiated", map[string]interface{}{
		"signal": "SIGTERM/SIGINT",
	})

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed", err)
	} else {
		logger.LogSystemEvent("shutdown_complete", "Maintify Core shutdown completed gracefully", nil)
	}

	// Cleanup resources
	logger.Info("Cleaning up resources")
	hooks.Cleanup()

	logger.Info("Graceful shutdown completed")
}
