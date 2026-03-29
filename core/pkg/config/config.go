package config

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Config holds the application configuration
type Config struct {
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	DBSSLMode           string
	RedisURL            string
	BuilderAPIKey       string
	EnableContainerMode bool
	JWTSecret           string
	LogLevel            string
	LogStructured       bool
	LogDir              string
}

var (
	// Current is the global configuration instance
	Current *Config
	mu      sync.Mutex
	loaded  bool
)

// Load loads the configuration from environment variables and secrets
func Load() (*Config, error) {
	mu.Lock()
	defer mu.Unlock()

	if loaded {
		return Current, nil
	}

	var err error
	Current, err = loadFromEnv()
	if err != nil {
		return nil, err
	}

	loaded = true
	return Current, nil
}

// loadFromEnv loads configuration values from environment variables
func loadFromEnv() (*Config, error) {
	cfg := &Config{
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "maintify"),
		DBName:              getEnv("DB_NAME", "maintify"),
		DBSSLMode:           getEnv("DB_SSLMODE", "disable"),
		RedisURL:            getEnv("REDIS_URL", "redis:6379"),
		EnableContainerMode: getEnv("ENABLE_CONTAINER_MODE", "false") == "true",
		LogLevel:            getEnv("LOG_LEVEL", "INFO"),
		LogStructured:       getEnv("LOG_STRUCTURED", "true") == "true",
		LogDir:              getEnv("LOG_DIR", "/var/log/maintify"),
	}

	var err error
	// Load secrets (prefer file-based secrets for security)
	cfg.DBPassword, err = getSecret("DB_PASSWORD", "maintify")
	if err != nil {
		return nil, err
	}

	cfg.BuilderAPIKey, err = getSecret("BUILDER_API_KEY", "")
	if err != nil {
		return nil, err
	}

	cfg.JWTSecret, err = getSecret("JWT_SECRET", "maintify-secret-key")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getSecret retrieves a secret from a file (Docker secrets) or environment variable
// It checks <KEY>_FILE first, then <KEY>
func getSecret(key, defaultValue string) (string, error) {
	// Check for file-based secret first (common in Docker/Kubernetes)
	fileEnv := key + "_FILE"
	if filePath, exists := os.LookupEnv(fileEnv); exists {
		content, err := os.ReadFile(filePath) // #nosec G304 -- path comes from env var set by ops, not user input
		if err != nil {
			return "", fmt.Errorf("failed to read secret file %s: %w", filePath, err)
		}
		return strings.TrimSpace(string(content)), nil
	}

	// Fallback to environment variable
	if value, exists := os.LookupEnv(key); exists {
		return value, nil
	}

	return defaultValue, nil
}
