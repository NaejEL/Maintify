package logging

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// LogService provides database operations for logs
type LogService interface {
	// IngestLogs stores multiple log entries in batch
	IngestLogs(ctx context.Context, orgID uuid.UUID, request LogIngestionRequest) (*LogIngestionResponse, error)

	// SearchLogs retrieves logs based on filters with RBAC support
	SearchLogs(ctx context.Context, orgID uuid.UUID, request LogSearchRequest) (*LogSearchResponse, error)

	// GetLogStatistics returns aggregated log metrics for dashboards
	GetLogStatistics(ctx context.Context, orgID uuid.UUID, request LogStatisticsRequest) (*LogStatisticsResponse, error)

	// DeleteOldLogs removes logs older than specified duration (admin operation)
	DeleteOldLogs(ctx context.Context, olderThan time.Duration) (int64, error)
}

// PostgreSQLLogService implements LogService using PostgreSQL with TimescaleDB
type PostgreSQLLogService struct {
	db *sql.DB
}

// NewPostgreSQLLogService creates a new PostgreSQL-based log service
func NewPostgreSQLLogService(db *sql.DB) *PostgreSQLLogService {
	return &PostgreSQLLogService{db: db}
}

// IngestLogs stores multiple log entries in batch for optimal performance
func (s *PostgreSQLLogService) IngestLogs(ctx context.Context, orgID uuid.UUID, request LogIngestionRequest) (*LogIngestionResponse, error) {
	response := &LogIngestionResponse{
		Timestamp: time.Now(),
		Errors:    make(map[int]LogIngestionError),
		BatchID:   request.BatchID,
	}

	if len(request.Entries) == 0 {
		response.Success = true
		return response, nil
	}

	// Validate and prepare entries
	validEntries := make([]LogEntry, 0, len(request.Entries))
	for i, entry := range request.Entries {
		if err := s.validateLogEntry(entry, orgID); err != nil {
			response.FailedCount++
			response.Errors[i] = LogIngestionError{
				Code:    ErrorCodeValidationFailed,
				Message: err.Error(),
			}
			continue
		}

		// Set required fields
		entry.ID = uuid.New()
		entry.OrganizationID = orgID
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now().UTC()
		}
		entry.CreatedAt = time.Now().UTC()
		if entry.Source == "" {
			entry.Source = request.Source
		}

		validEntries = append(validEntries, entry)
	}

	if len(validEntries) == 0 {
		response.Success = false
		return response, nil
	}

	// Batch insert for performance (following guidelines)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Use PostgreSQL COPY for high-performance batch insert
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO logs (
			id, organization_id, timestamp, level, message, component, source,
			user_id, session_id, request_id, plugin_name, action, error_message, details, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, entry := range validEntries {
		detailsJSON, err := json.Marshal(entry.Details)
		if err != nil {
			response.FailedCount++
			continue
		}

		_, err = stmt.ExecContext(ctx,
			entry.ID, entry.OrganizationID, entry.Timestamp, entry.Level, entry.Message,
			entry.Component, entry.Source, entry.UserID, entry.SessionID, entry.RequestID,
			entry.PluginName, entry.Action, entry.ErrorMessage, string(detailsJSON), entry.CreatedAt,
		)
		if err != nil {
			response.FailedCount++
			continue
		}

		response.ProcessedCount++
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	response.Success = response.ProcessedCount > 0
	return response, nil
}

// SearchLogs retrieves logs based on filters with proper indexing and RBAC
func (s *PostgreSQLLogService) SearchLogs(ctx context.Context, orgID uuid.UUID, request LogSearchRequest) (*LogSearchResponse, error) {
	// Build dynamic query with proper parameter binding (following guidelines - SQL injection prevention)
	query := `
		SELECT id, organization_id, timestamp, level, message, component, source,
		       user_id, session_id, request_id, plugin_name, action, error_message, details, created_at
		FROM logs 
		WHERE organization_id = $1
	`

	args := []interface{}{orgID}
	argIndex := 2

	// Add time range filters (optimized for TimescaleDB)
	if request.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, *request.StartTime)
		argIndex++
	}
	if request.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, *request.EndTime)
		argIndex++
	}

	// Add level filters
	if len(request.Levels) > 0 {
		query += fmt.Sprintf(" AND level = ANY($%d)", argIndex)
		levels := make([]string, len(request.Levels))
		for i, level := range request.Levels {
			levels[i] = string(level)
		}
		args = append(args, pq.Array(levels))
		argIndex++
	}

	// Add component filters
	if len(request.Components) > 0 {
		query += fmt.Sprintf(" AND component = ANY($%d)", argIndex)
		args = append(args, pq.Array(request.Components))
		argIndex++
	}

	// Add user filter
	if request.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *request.UserID)
		argIndex++
	}

	// Add plugin filter
	if request.PluginName != "" {
		query += fmt.Sprintf(" AND plugin_name = $%d", argIndex)
		args = append(args, request.PluginName)
		argIndex++
	}

	// Add action filter
	if request.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, request.Action)
		argIndex++
	}

	// Add full-text search using tsvector
	if request.SearchText != "" {
		query += fmt.Sprintf(" AND message_vector @@ plainto_tsquery('english', $%d)", argIndex)
		args = append(args, request.SearchText)
		argIndex++
	}

	// Order by timestamp (optimized for time-series queries)
	query += " ORDER BY timestamp DESC"

	// Add pagination
	if request.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex) // #nosec G202 -- appending parameterized placeholder index, not user value
		args = append(args, request.Limit)
		argIndex++
	}
	if request.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex) // #nosec G202 -- appending parameterized placeholder index, not user value
		args = append(args, request.Offset)
	}

	startTime := time.Now()

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	entries := make([]LogEntryResponse, 0)
	for rows.Next() {
		var log LogEntry
		var detailsJSON sql.NullString

		err := rows.Scan(
			&log.ID, &log.OrganizationID, &log.Timestamp, &log.Level, &log.Message,
			&log.Component, &log.Source, &log.UserID, &log.SessionID, &log.RequestID,
			&log.PluginName, &log.Action, &log.ErrorMessage, &detailsJSON, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}

		// Parse JSONB details
		if detailsJSON.Valid && detailsJSON.String != "" {
			if err := json.Unmarshal([]byte(detailsJSON.String), &log.Details); err != nil {
				// Log parsing error but continue (following guidelines - graceful degradation)
				log.Details = map[string]interface{}{"_parse_error": err.Error()}
			}
		}

		entries = append(entries, LogEntryResponse{
			LogEntry:  log,
			ID:        log.ID,
			CreatedAt: log.CreatedAt,
			UpdatedAt: log.CreatedAt, // Assuming updated_at is same as created_at for logs
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating log rows: %w", err)
	}

	// Get total count for pagination (separate optimized query)
	totalCount, err := s.getLogCount(ctx, orgID, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Calculate pagination info
	limit := request.Limit
	if limit == 0 {
		limit = len(entries)
	}

	hasMore := int64(request.Offset+len(entries)) < totalCount

	return &LogSearchResponse{
		Entries: entries,
		Pagination: PaginationInfo{
			Limit:         limit,
			Offset:        request.Offset,
			TotalCount:    int(totalCount),
			ReturnedCount: len(entries),
			HasMore:       hasMore,
		},
		SearchMeta: SearchMetadata{
			ExecutionTime: time.Since(startTime),
			FiltersApplied: map[string]interface{}{
				"organization_id": orgID,
				// Add other filters if needed
			},
		},
		Timestamp: time.Now(),
	}, nil
}

// GetLogStatistics returns aggregated metrics for monitoring dashboards
func (s *PostgreSQLLogService) GetLogStatistics(ctx context.Context, orgID uuid.UUID, request LogStatisticsRequest) (*LogStatisticsResponse, error) {
	startTime := time.Now().Add(-24 * time.Hour)
	if request.StartTime != nil {
		startTime = *request.StartTime
	}
	endTime := time.Now()
	if request.EndTime != nil {
		endTime = *request.EndTime
	}

	query := `
		SELECT 
			component,
			level,
			COUNT(*) as count
		FROM logs 
		WHERE organization_id = $1 
		  AND timestamp >= $2 
		  AND timestamp <= $3
		GROUP BY component, level
	`

	rows, err := s.db.QueryContext(ctx, query, orgID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query log statistics: %w", err)
	}
	defer rows.Close()

	response := &LogStatisticsResponse{
		LevelCounts:     make(map[LogLevel]int64),
		ComponentCounts: make(map[string]int64),
		TimeRange: TimeRange{
			Start: startTime,
			End:   endTime,
		},
		Timestamp: time.Now(),
	}

	for rows.Next() {
		var component string
		var level LogLevel
		var count int64

		err := rows.Scan(&component, &level, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan statistics: %w", err)
		}

		response.TotalLogs += count
		response.LevelCounts[level] += count
		response.ComponentCounts[component] += count
	}

	return response, nil
}

// DeleteOldLogs removes logs older than specified duration (admin operation)
func (s *PostgreSQLLogService) DeleteOldLogs(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	result, err := s.db.ExecContext(ctx, "DELETE FROM logs WHERE timestamp < $1", cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return rowsAffected, nil
}

// Helper functions

func (s *PostgreSQLLogService) validateLogEntry(entry LogEntry, orgID uuid.UUID) error {
	if entry.Message == "" {
		return fmt.Errorf("message is required")
	}
	if entry.Component == "" {
		return fmt.Errorf("component is required")
	}
	if entry.Level == "" {
		return fmt.Errorf("level is required")
	}

	// Validate log level
	validLevels := map[LogLevel]bool{
		LogLevelDebug: true,
		LogLevelInfo:  true,
		LogLevelWarn:  true,
		LogLevelError: true,
		LogLevelFatal: true,
	}
	if !validLevels[entry.Level] {
		return fmt.Errorf("invalid log level: %s", entry.Level)
	}

	// Validate message length (prevent extremely large messages)
	if len(entry.Message) > 10000 {
		return fmt.Errorf("message too long (max 10,000 characters)")
	}

	// Validate component name
	if len(entry.Component) > 100 {
		err := fmt.Errorf("component name too long (max 100 characters)")
		return err
	}

	return nil
}

func (s *PostgreSQLLogService) getLogCount(ctx context.Context, orgID uuid.UUID, request LogSearchRequest) (int64, error) {
	// Simplified count query (reuse filter logic but just count)
	query := "SELECT COUNT(*) FROM logs WHERE organization_id = $1"
	args := []interface{}{orgID}
	argIndex := 2

	// Add same filters as search query
	if request.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, *request.StartTime)
		argIndex++
	}
	if request.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, *request.EndTime)
		argIndex++
	}
	if len(request.Levels) > 0 {
		query += fmt.Sprintf(" AND level = ANY($%d)", argIndex)
		levels := make([]string, len(request.Levels))
		for i, level := range request.Levels {
			levels[i] = string(level)
		}
		args = append(args, pq.Array(levels))
		argIndex++
	}
	if len(request.Components) > 0 {
		query += fmt.Sprintf(" AND component = ANY($%d)", argIndex)
		args = append(args, pq.Array(request.Components))
		argIndex++
	}
	if request.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *request.UserID)
		argIndex++
	}
	if request.PluginName != "" {
		query += fmt.Sprintf(" AND plugin_name = $%d", argIndex)
		args = append(args, request.PluginName)
		argIndex++
	}
	if request.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, request.Action)
		argIndex++
	}
	if request.SearchText != "" {
		query += fmt.Sprintf(" AND message_vector @@ plainto_tsquery('english', $%d)", argIndex)
		args = append(args, request.SearchText)
	}

	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count logs: %w", err)
	}

	return count, nil
}
