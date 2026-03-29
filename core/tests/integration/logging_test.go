//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maintify/core/pkg/logging"
)

func getDB(t *testing.T) *sql.DB {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	if user == "" {
		user = "maintify"
	}
	if pass == "" {
		pass = "maintify"
	}
	if name == "" {
		name = "maintify_core"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, name)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)

	err = db.Ping()
	require.NoError(t, err)

	return db
}

func TestLoggingServiceIntegration(t *testing.T) {
	db := getDB(t)
	defer db.Close()

	service := logging.NewPostgreSQLLogService(db)
	ctx := context.Background()
	orgID := uuid.New()

	// Test IngestLogs
	t.Run("IngestLogs", func(t *testing.T) {
		req := logging.LogIngestionRequest{
			Entries: []logging.LogEntry{
				{
					Level:     logging.LogLevelInfo,
					Component: "integration-test",
					Message:   "Test log message",
					Timestamp: time.Now(),
				},
			},
		}

		resp, err := service.IngestLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, 1, resp.ProcessedCount)
	})

	// Test SearchLogs
	t.Run("SearchLogs", func(t *testing.T) {
		// Wait a bit for logs to be indexed/committed if necessary (Postgres is immediate usually)

		req := logging.LogSearchRequest{
			Components: []string{"integration-test"},
			Limit:      10,
		}

		resp, err := service.SearchLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Entries)
		assert.Equal(t, "Test log message", resp.Entries[0].Message)
	})
}
