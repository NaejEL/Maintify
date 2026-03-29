package logging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgreSQLLogService(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgreSQLLogService(db)
	assert.NotNil(t, service)
}

func TestIngestLogs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgreSQLLogService(db)
	ctx := context.Background()
	orgID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		entries := []LogEntry{
			{
				Level:     LogLevelInfo,
				Component: "test",
				Message:   "test message",
			},
		}
		req := LogIngestionRequest{
			Entries: entries,
			Source:  "test-source",
		}

		mock.ExpectBegin()
		mock.ExpectPrepare("INSERT INTO logs").ExpectExec().
			WithArgs(
				sqlmock.AnyArg(), // id
				orgID,
				sqlmock.AnyArg(), // timestamp
				LogLevelInfo,
				"test message",
				"test",
				"test-source",
				sqlmock.AnyArg(), // user_id
				sqlmock.AnyArg(), // session_id
				sqlmock.AnyArg(), // request_id
				sqlmock.AnyArg(), // plugin_name
				sqlmock.AnyArg(), // action
				sqlmock.AnyArg(), // error_message
				sqlmock.AnyArg(), // details
				sqlmock.AnyArg(), // created_at
			).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		resp, err := service.IngestLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, 1, resp.ProcessedCount)
	})

	t.Run("Empty Request", func(t *testing.T) {
		req := LogIngestionRequest{Entries: []LogEntry{}}
		resp, err := service.IngestLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, 0, resp.ProcessedCount)
	})

	t.Run("Validation Failure", func(t *testing.T) {
		entries := []LogEntry{
			{
				Level:     LogLevelInfo,
				Component: "", // Missing component
				Message:   "test",
			},
		}
		req := LogIngestionRequest{Entries: entries}

		resp, err := service.IngestLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, 1, resp.FailedCount)
		assert.Contains(t, resp.Errors[0].Message, "component is required")
	})

	t.Run("Transaction Error", func(t *testing.T) {
		entries := []LogEntry{
			{
				Level:     LogLevelInfo,
				Component: "test",
				Message:   "test",
			},
		}
		req := LogIngestionRequest{Entries: entries}

		mock.ExpectBegin().WillReturnError(errors.New("tx error"))

		resp, err := service.IngestLogs(ctx, orgID, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Prepare Error", func(t *testing.T) {
		entries := []LogEntry{{Level: LogLevelInfo, Component: "test", Message: "test"}}
		req := LogIngestionRequest{Entries: entries}

		mock.ExpectBegin()
		mock.ExpectPrepare("INSERT INTO logs").WillReturnError(errors.New("prepare error"))
		mock.ExpectRollback()

		resp, err := service.IngestLogs(ctx, orgID, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Exec Error", func(t *testing.T) {
		entries := []LogEntry{{Level: LogLevelInfo, Component: "test", Message: "test"}}
		req := LogIngestionRequest{Entries: entries}

		mock.ExpectBegin()
		mock.ExpectPrepare("INSERT INTO logs").ExpectExec().WillReturnError(errors.New("exec error"))
		mock.ExpectCommit() // It continues on error but counts as failed

		resp, err := service.IngestLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.FailedCount)
		assert.Equal(t, 0, resp.ProcessedCount)
	})

	t.Run("Commit Error", func(t *testing.T) {
		entries := []LogEntry{{Level: LogLevelInfo, Component: "test", Message: "test"}}
		req := LogIngestionRequest{Entries: entries}

		mock.ExpectBegin()
		mock.ExpectPrepare("INSERT INTO logs").ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit().WillReturnError(errors.New("commit error"))

		resp, err := service.IngestLogs(ctx, orgID, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
	t.Run("JSON Marshal Error", func(t *testing.T) {
		// To trigger JSON marshal error, we need a type that cannot be marshaled.
		// But LogEntry.Details is map[string]interface{}.
		// We can put a channel or function in it.
		entries := []LogEntry{{
			Level:     LogLevelInfo,
			Component: "test",
			Message:   "test",
			Details:   map[string]interface{}{"bad": make(chan int)},
		}}
		req := LogIngestionRequest{Entries: entries}

		mock.ExpectBegin()
		mock.ExpectPrepare("INSERT INTO logs")
		// No exec because marshal fails before loop body completes or inside loop
		// The code:
		// for _, entry := range validEntries {
		//     detailsJSON, err := json.Marshal(entry.Details)
		//     if err != nil { response.FailedCount++; continue }
		// }
		// So it continues.

		mock.ExpectCommit()

		resp, err := service.IngestLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.FailedCount)
	})
}

func TestSearchLogs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgreSQLLogService(db)
	ctx := context.Background()
	orgID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		req := LogSearchRequest{
			Limit: 10,
		}

		// Mock Search Query
		rows := sqlmock.NewRows([]string{
			"id", "organization_id", "timestamp", "level", "message", "component", "source",
			"user_id", "session_id", "request_id", "plugin_name", "action", "error_message", "details", "created_at",
		}).AddRow(
			uuid.New(), orgID, time.Now(), LogLevelInfo, "msg", "comp", "src",
			uuid.New(), "sess", "req", "plug", "act", "", "{}", time.Now(),
		)

		mock.ExpectQuery("SELECT .* FROM logs").
			WithArgs(orgID, 10). // orgID and Limit
			WillReturnRows(rows)

		// Mock Count Query
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM logs").
			WithArgs(orgID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		resp, err := service.SearchLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.Len(t, resp.Entries, 1)
		assert.Equal(t, 1, resp.Pagination.TotalCount)
	})

	t.Run("All Filters", func(t *testing.T) {
		startTime := time.Now().Add(-1 * time.Hour)
		endTime := time.Now()
		userID := uuid.New()
		req := LogSearchRequest{
			StartTime:  &startTime,
			EndTime:    &endTime,
			Levels:     []LogLevel{LogLevelInfo, LogLevelError},
			Components: []string{"comp1"},
			UserID:     &userID,
			PluginName: "plugin1",
			Action:     "action1",
			SearchText: "search",
			Limit:      10,
			Offset:     5,
		}

		// Mock Search Query
		mock.ExpectQuery("SELECT .* FROM logs").
			WithArgs(
				orgID,
				startTime,
				endTime,
				sqlmock.AnyArg(), // levels array
				sqlmock.AnyArg(), // components array
				userID,
				"plugin1",
				"action1",
				"search",
				10, // limit
				5,  // offset
			).
			WillReturnRows(sqlmock.NewRows([]string{}))

		// Mock Count Query
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM logs").
			WithArgs(
				orgID,
				startTime,
				endTime,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				userID,
				"plugin1",
				"action1",
				"search",
			).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		resp, err := service.SearchLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.Empty(t, resp.Entries)
	})

	t.Run("Query Error", func(t *testing.T) {
		req := LogSearchRequest{}
		mock.ExpectQuery("SELECT .* FROM logs").WillReturnError(errors.New("db error"))

		resp, err := service.SearchLogs(ctx, orgID, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Scan Error", func(t *testing.T) {
		req := LogSearchRequest{}
		// Return row with wrong number of columns
		rows := sqlmock.NewRows([]string{"id"}).AddRow(uuid.New())
		mock.ExpectQuery("SELECT .* FROM logs").WillReturnRows(rows)

		resp, err := service.SearchLogs(ctx, orgID, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Count Error", func(t *testing.T) {
		req := LogSearchRequest{}
		mock.ExpectQuery("SELECT .* FROM logs").WillReturnRows(sqlmock.NewRows([]string{}))
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM logs").WillReturnError(errors.New("count error"))

		resp, err := service.SearchLogs(ctx, orgID, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
	t.Run("JSON Unmarshal Error", func(t *testing.T) {
		req := LogSearchRequest{Limit: 10}

		// Return invalid JSON in details
		rows := sqlmock.NewRows([]string{
			"id", "organization_id", "timestamp", "level", "message", "component", "source",
			"user_id", "session_id", "request_id", "plugin_name", "action", "error_message", "details", "created_at",
		}).AddRow(
			uuid.New(), orgID, time.Now(), LogLevelInfo, "msg", "comp", "src",
			uuid.New(), "sess", "req", "plug", "act", "", "{invalid-json}", time.Now(),
		)

		mock.ExpectQuery("SELECT .* FROM logs").WillReturnRows(rows)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM logs").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		resp, err := service.SearchLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.Len(t, resp.Entries, 1)
		// Check if details contains error
		assert.Contains(t, resp.Entries[0].Details, "_parse_error")
	})
	t.Run("Rows Err Error", func(t *testing.T) {
		req := LogSearchRequest{}
		// Mock rows that return error after some rows
		rows := sqlmock.NewRows([]string{
			"id", "organization_id", "timestamp", "level", "message", "component", "source",
			"user_id", "session_id", "request_id", "plugin_name", "action", "error_message", "details", "created_at",
		}).AddRow(
			uuid.New(), orgID, time.Now(), LogLevelInfo, "msg", "comp", "src",
			uuid.New(), "sess", "req", "plug", "act", "", "{}", time.Now(),
		).RowError(0, errors.New("rows iteration error"))

		mock.ExpectQuery("SELECT .* FROM logs").WillReturnRows(rows)

		resp, err := service.SearchLogs(ctx, orgID, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
	t.Run("Default Limit", func(t *testing.T) {
		req := LogSearchRequest{Limit: 0}

		// Mock Search Query
		rows := sqlmock.NewRows([]string{
			"id", "organization_id", "timestamp", "level", "message", "component", "source",
			"user_id", "session_id", "request_id", "plugin_name", "action", "error_message", "details", "created_at",
		}).AddRow(
			uuid.New(), orgID, time.Now(), LogLevelInfo, "msg", "comp", "src",
			uuid.New(), "sess", "req", "plug", "act", "", "{}", time.Now(),
		)

		mock.ExpectQuery("SELECT .* FROM logs").
			WithArgs(orgID). // No limit arg
			WillReturnRows(rows)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM logs").
			WithArgs(orgID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		resp, err := service.SearchLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Pagination.Limit)
	})
}

func TestGetLogStatistics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgreSQLLogService(db)
	ctx := context.Background()
	orgID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		req := LogStatisticsRequest{}

		rows := sqlmock.NewRows([]string{"component", "level", "count"}).
			AddRow("comp1", LogLevelInfo, 10).
			AddRow("comp1", LogLevelError, 2).
			AddRow("comp2", LogLevelInfo, 5)

		mock.ExpectQuery("SELECT .* FROM logs").
			WithArgs(orgID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		resp, err := service.GetLogStatistics(ctx, orgID, req)
		require.NoError(t, err)
		assert.Equal(t, int64(17), resp.TotalLogs)
		assert.Equal(t, int64(15), resp.LevelCounts[LogLevelInfo])
		assert.Equal(t, int64(2), resp.LevelCounts[LogLevelError])
	})

	t.Run("Scan Error", func(t *testing.T) {
		req := LogStatisticsRequest{}
		// Wrong columns
		rows := sqlmock.NewRows([]string{"component"}).AddRow("comp1")
		mock.ExpectQuery("SELECT .* FROM logs").WillReturnRows(rows)

		resp, err := service.GetLogStatistics(ctx, orgID, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
	t.Run("Query Error", func(t *testing.T) {
		req := LogStatisticsRequest{}
		mock.ExpectQuery("SELECT .* FROM logs").WillReturnError(errors.New("query error"))

		resp, err := service.GetLogStatistics(ctx, orgID, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
	t.Run("With Time Range", func(t *testing.T) {
		startTime := time.Now().Add(-1 * time.Hour)
		endTime := time.Now()
		req := LogStatisticsRequest{
			StartTime: &startTime,
			EndTime:   &endTime,
		}

		rows := sqlmock.NewRows([]string{"component", "level", "count"}).
			AddRow("comp1", LogLevelInfo, 10)

		mock.ExpectQuery("SELECT .* FROM logs").
			WithArgs(orgID, startTime, endTime).
			WillReturnRows(rows)

		resp, err := service.GetLogStatistics(ctx, orgID, req)
		require.NoError(t, err)
		assert.Equal(t, int64(10), resp.TotalLogs)
	})
}

func TestDeleteOldLogs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgreSQLLogService(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM logs").
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 100))

		count, err := service.DeleteOldLogs(ctx, 24*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, int64(100), count)
	})

	t.Run("Exec Error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM logs").WillReturnError(errors.New("exec error"))
		_, err := service.DeleteOldLogs(ctx, 24*time.Hour)
		assert.Error(t, err)
	})

	t.Run("RowsAffected Error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM logs").
			WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))

		_, err := service.DeleteOldLogs(ctx, 24*time.Hour)
		assert.Error(t, err)
	})
}

func TestValidateLogEntry(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	service := NewPostgreSQLLogService(db)
	ctx := context.Background()
	orgID := uuid.New()

	t.Run("Invalid Level", func(t *testing.T) {
		req := LogIngestionRequest{
			Entries: []LogEntry{{
				Level:     "INVALID",
				Component: "test",
				Message:   "msg",
			}},
		}
		resp, err := service.IngestLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Errors[0].Message, "invalid log level")
	})

	t.Run("Message Too Long", func(t *testing.T) {
		longMsg := make([]byte, 10001)
		req := LogIngestionRequest{
			Entries: []LogEntry{{
				Level:     LogLevelInfo,
				Component: "test",
				Message:   string(longMsg),
			}},
		}
		resp, err := service.IngestLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Errors[0].Message, "message too long")
	})

	t.Run("Component Too Long", func(t *testing.T) {
		longComp := make([]byte, 101)
		req := LogIngestionRequest{
			Entries: []LogEntry{{
				Level:     LogLevelInfo,
				Component: string(longComp),
				Message:   "msg",
			}},
		}
		resp, err := service.IngestLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Errors[0].Message, "component name too long")
	})

	t.Run("Empty Message", func(t *testing.T) {
		req := LogIngestionRequest{
			Entries: []LogEntry{{
				Level:     LogLevelInfo,
				Component: "test",
				Message:   "",
			}},
		}
		resp, err := service.IngestLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Errors[0].Message, "message is required")
	})

	t.Run("Empty Level", func(t *testing.T) {
		req := LogIngestionRequest{
			Entries: []LogEntry{{
				Level:     "",
				Component: "test",
				Message:   "msg",
			}},
		}
		resp, err := service.IngestLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Errors[0].Message, "level is required")
	})
}

func TestGetLogCount_Coverage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgreSQLLogService(db)
	ctx := context.Background()
	orgID := uuid.New()

	t.Run("Time Filters", func(t *testing.T) {
		startTime := time.Now().Add(-1 * time.Hour)
		endTime := time.Now()
		req := LogSearchRequest{
			StartTime: &startTime,
			EndTime:   &endTime,
			Limit:     10,
		}

		// Mock Search Query
		rows := sqlmock.NewRows([]string{
			"id", "organization_id", "timestamp", "level", "message", "component", "source",
			"user_id", "session_id", "request_id", "plugin_name", "action", "error_message", "details", "created_at",
		}).AddRow(
			uuid.New(), orgID, time.Now(), LogLevelInfo, "msg", "comp", "src",
			uuid.New(), "sess", "req", "plug", "act", "", "{}", time.Now(),
		)

		mock.ExpectQuery("SELECT .* FROM logs").
			WithArgs(orgID, startTime, endTime, 10).
			WillReturnRows(rows)

		// Mock Count Query - Explicitly checking for StartTime and EndTime args
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM logs").
			WithArgs(orgID, startTime, endTime).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		resp, err := service.SearchLogs(ctx, orgID, req)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Pagination.TotalCount)
	})
}
