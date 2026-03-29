package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnv(t *testing.T) {
	t.Run("returns default value when env var is not set", func(t *testing.T) {
		key := "TEST_ENV_VAR_MISSING"
		os.Unsetenv(key)
		val := getEnv(key, "default")
		assert.Equal(t, "default", val)
	})

	t.Run("returns env value when set", func(t *testing.T) {
		key := "TEST_ENV_VAR_SET"
		expected := "custom_value"
		os.Setenv(key, expected)
		defer os.Unsetenv(key)

		val := getEnv(key, "default")
		assert.Equal(t, expected, val)
	})
}

func TestGetSecret(t *testing.T) {
	t.Run("returns default value when neither file nor env var set", func(t *testing.T) {
		key := "TEST_SECRET_MISSING"
		os.Unsetenv(key)
		os.Unsetenv(key + "_FILE")

		val, err := getSecret(key, "default_secret")
		require.NoError(t, err)
		assert.Equal(t, "default_secret", val)
	})

	t.Run("returns env value when set", func(t *testing.T) {
		key := "TEST_SECRET_ENV"
		expected := "env_secret"
		os.Setenv(key, expected)
		os.Unsetenv(key + "_FILE")
		defer os.Unsetenv(key)

		val, err := getSecret(key, "default")
		require.NoError(t, err)
		assert.Equal(t, expected, val)
	})

	t.Run("returns file content when _FILE env var set", func(t *testing.T) {
		key := "TEST_SECRET_FILE"
		expected := "file_secret"

		// Create temp file
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret_file")
		err := os.WriteFile(secretFile, []byte(expected+"\n"), 0600) // Add newline to test trimming
		require.NoError(t, err)

		os.Setenv(key+"_FILE", secretFile)
		defer os.Unsetenv(key + "_FILE")

		// Ensure direct env var is ignored if file is present (priority check)
		os.Setenv(key, "ignored_env_value")
		defer os.Unsetenv(key)

		val, err := getSecret(key, "default")
		require.NoError(t, err)
		assert.Equal(t, expected, val)
	})

	t.Run("returns error when secret file cannot be read", func(t *testing.T) {
		key := "TEST_SECRET_FILE_ERROR"

		os.Setenv(key+"_FILE", "/non/existent/file")
		defer os.Unsetenv(key + "_FILE")

		val, err := getSecret(key, "default")
		assert.Error(t, err)
		assert.Empty(t, val)
	})
}

func TestLoadFromEnv(t *testing.T) {
	t.Run("loads defaults when no env vars set", func(t *testing.T) {
		// Ensure clean state
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")

		cfg, err := loadFromEnv()
		require.NoError(t, err)
		assert.Equal(t, "localhost", cfg.DBHost)
		assert.Equal(t, "5432", cfg.DBPort)
		assert.Equal(t, "maintify", cfg.DBUser)
	})

	t.Run("loads values from env vars", func(t *testing.T) {
		os.Setenv("DB_HOST", "custom-host")
		os.Setenv("ENABLE_CONTAINER_MODE", "true")
		defer os.Unsetenv("DB_HOST")
		defer os.Unsetenv("ENABLE_CONTAINER_MODE")

		cfg, err := loadFromEnv()
		require.NoError(t, err)
		assert.Equal(t, "custom-host", cfg.DBHost)
		assert.True(t, cfg.EnableContainerMode)
	})

	t.Run("returns error when DB_PASSWORD_FILE is invalid", func(t *testing.T) {
		os.Setenv("DB_PASSWORD_FILE", "/non/existent/pwd")
		defer os.Unsetenv("DB_PASSWORD_FILE")

		cfg, err := loadFromEnv()
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("returns error when BUILDER_API_KEY_FILE is invalid", func(t *testing.T) {
		os.Setenv("BUILDER_API_KEY_FILE", "/non/existent/key")
		defer os.Unsetenv("BUILDER_API_KEY_FILE")

		cfg, err := loadFromEnv()
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("returns error when JWT_SECRET_FILE is invalid", func(t *testing.T) {
		os.Setenv("JWT_SECRET_FILE", "/non/existent/jwt")
		defer os.Unsetenv("JWT_SECRET_FILE")

		cfg, err := loadFromEnv()
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})
}

func TestLoad(t *testing.T) {
	// Reset singleton state
	resetSingleton := func() {
		mu.Lock()
		loaded = false
		Current = nil
		mu.Unlock()
	}

	t.Run("successfully loads config", func(t *testing.T) {
		resetSingleton()

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, cfg, Current)

		// Call again to ensure singleton behavior
		cfg2, err := Load()
		require.NoError(t, err)
		assert.Equal(t, cfg, cfg2)
	})

	t.Run("retries loading if first attempt fails", func(t *testing.T) {
		resetSingleton()

		// Force failure
		os.Setenv("DB_PASSWORD_FILE", "/non/existent/pwd")

		cfg, err := Load()
		assert.Error(t, err)
		assert.Nil(t, cfg)

		// Fix failure
		os.Unsetenv("DB_PASSWORD_FILE")

		// Retry
		cfg, err = Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, cfg, Current)
	})
}
