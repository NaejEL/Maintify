package logging

import (
	"time"

	"github.com/google/uuid"
)

// LogLevel represents the severity of a log message (matching database enum)
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelFatal LogLevel = "FATAL"
)

// LogEntry represents a single log entry for API and database operations
type LogEntry struct {
	ID             uuid.UUID              `json:"id,omitempty" db:"id"`
	OrganizationID uuid.UUID              `json:"organization_id" db:"organization_id"`
	Timestamp      time.Time              `json:"timestamp" db:"timestamp"`
	Level          LogLevel               `json:"level" db:"level"`
	Message        string                 `json:"message" db:"message"`
	Component      string                 `json:"component" db:"component"`
	Source         string                 `json:"source,omitempty" db:"source"`
	UserID         *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	SessionID      string                 `json:"session_id,omitempty" db:"session_id"`
	RequestID      string                 `json:"request_id,omitempty" db:"request_id"`
	PluginName     string                 `json:"plugin_name,omitempty" db:"plugin_name"`
	Action         string                 `json:"action,omitempty" db:"action"`
	ErrorMessage   string                 `json:"error_message,omitempty" db:"error_message"`
	Details        map[string]interface{} `json:"details,omitempty" db:"details"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// LogIngestionRequest represents a batch log submission request
type LogIngestionRequest struct {
	// Entries contains the log entries to be ingested
	Entries []LogEntry `json:"entries" validate:"required,max=100,dive"`

	// BatchID is an optional identifier for the batch (for duplicate detection)
	BatchID string `json:"batch_id,omitempty"`

	// Source identifies the origin of the logs (plugin name, service name, etc.)
	Source string `json:"source,omitempty" validate:"max=100"`
}

// LogIngestionResponse represents the response to a log ingestion request
type LogIngestionResponse struct {
	// Success indicates if all entries were successfully ingested
	Success bool `json:"success"`

	// ProcessedCount is the number of successfully processed entries
	ProcessedCount int `json:"processed_count"`

	// FailedCount is the number of entries that failed to process
	FailedCount int `json:"failed_count"`

	// Errors contains details about failed entries (indexed by original entry position)
	Errors map[int]LogIngestionError `json:"errors,omitempty"`

	// BatchID echoes back the provided batch ID
	BatchID string `json:"batch_id,omitempty"`

	// Timestamp when the ingestion was processed
	Timestamp time.Time `json:"timestamp"`
}

// LogIngestionError represents an error for a specific log entry
type LogIngestionError struct {
	// Code is a machine-readable error code
	Code string `json:"code"`

	// Message is a human-readable error message
	Message string `json:"message"`

	// Field identifies the problematic field (if applicable)
	Field string `json:"field,omitempty"`
}

// LogSearchRequest represents a log search request (query parameters)
type LogSearchRequest struct {
	// Time range filters
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Since     string     `json:"since,omitempty"` // Relative time like "1h", "24h", "7d"

	// Content filters
	Levels     []LogLevel `json:"levels,omitempty"`
	Components []string   `json:"components,omitempty"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	PluginName string     `json:"plugin_name,omitempty"`
	Action     string     `json:"action,omitempty"`

	// Search text (full-text search)
	SearchText string `json:"search_text,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`  // Default: 50, Max: 1000
	Offset int `json:"offset,omitempty"` // Default: 0

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`    // timestamp, level, component (default: timestamp)
	SortOrder string `json:"sort_order,omitempty"` // asc, desc (default: desc)
}

// LogSearchResponse represents the response to a log search request
type LogSearchResponse struct {
	// Entries contains the matching log entries
	Entries []LogEntryResponse `json:"entries"`

	// Pagination information
	Pagination PaginationInfo `json:"pagination"`

	// Search metadata
	SearchMeta SearchMetadata `json:"search_meta"`

	// Timestamp when the search was executed
	Timestamp time.Time `json:"timestamp"`
}

// LogEntryResponse represents a log entry in search results (with computed fields)
type LogEntryResponse struct {
	// Core log entry data
	LogEntry

	// Computed fields
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Additional context (if available)
	UserEmail   string `json:"user_email,omitempty"`
	UserName    string `json:"user_name,omitempty"`
	PluginTitle string `json:"plugin_title,omitempty"`
}

// PaginationInfo provides pagination metadata
type PaginationInfo struct {
	// Current page information
	Limit  int `json:"limit"`
	Offset int `json:"offset"`

	// Result counts
	TotalCount    int  `json:"total_count"`
	ReturnedCount int  `json:"returned_count"`
	HasMore       bool `json:"has_more"`

	// Navigation links (relative URLs)
	NextOffset     *int `json:"next_offset,omitempty"`
	PreviousOffset *int `json:"previous_offset,omitempty"`
}

// SearchMetadata provides additional information about the search
type SearchMetadata struct {
	// Query performance
	ExecutionTime time.Duration `json:"execution_time"`

	// Applied filters summary
	FiltersApplied map[string]interface{} `json:"filters_applied"`

	// Search suggestions (for typos, alternative terms)
	Suggestions []string `json:"suggestions,omitempty"`

	// Aggregations (if requested)
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
}

// LogStatisticsRequest represents a request for log statistics
type LogStatisticsRequest struct {
	// Time range for statistics
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Since     string     `json:"since,omitempty"` // Relative time like "1h", "24h", "7d"

	// Grouping options
	GroupBy []string `json:"group_by,omitempty"` // level, component, plugin, user, hour, day

	// Additional filters (same as search)
	Levels     []LogLevel `json:"levels,omitempty"`
	Components []string   `json:"components,omitempty"`
	PluginName string     `json:"plugin_name,omitempty"`
}

// LogStatisticsResponse represents log statistics and metrics
type LogStatisticsResponse struct {
	// Overall counts
	TotalLogs int64 `json:"total_logs"`

	// Breakdown by log level
	LevelCounts map[LogLevel]int64 `json:"level_counts"`

	// Breakdown by component
	ComponentCounts map[string]int64 `json:"component_counts"`

	// Breakdown by plugin
	PluginCounts map[string]int64 `json:"plugin_counts"`

	// Time series data (hourly/daily aggregations)
	TimeSeries []TimeSeriesPoint `json:"time_series"`

	// Top error messages (for ERROR and FATAL levels)
	TopErrors []ErrorFrequency `json:"top_errors"`

	// Performance metrics
	AverageLogsPerHour float64 `json:"average_logs_per_hour"`
	PeakHour           *string `json:"peak_hour,omitempty"`

	// Time range of the statistics
	TimeRange TimeRange `json:"time_range"`

	// Generation timestamp
	Timestamp time.Time `json:"timestamp"`
}

// TimeSeriesPoint represents a point in time-series log data
type TimeSeriesPoint struct {
	Timestamp time.Time        `json:"timestamp"`
	Count     int64            `json:"count"`
	Breakdown map[string]int64 `json:"breakdown,omitempty"` // Optional breakdown by level/component
}

// ErrorFrequency represents frequently occurring error messages
type ErrorFrequency struct {
	Message   string    `json:"message"`
	Count     int64     `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// TimeRange represents a time range for statistics
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// CleanupRequest represents a request to clean up old logs
type CleanupRequest struct {
	// OlderThan specifies logs older than this duration should be deleted
	OlderThan string `json:"older_than"` // e.g., "90d", "1y"

	// DryRun if true, only returns count without actually deleting
	DryRun bool `json:"dry_run,omitempty"`

	// Filters to limit cleanup scope
	Components []string   `json:"components,omitempty"`
	Levels     []LogLevel `json:"levels,omitempty"`
}

// CleanupResponse represents the response to a cleanup request
type CleanupResponse struct {
	// Success indicates if the cleanup was successful
	Success bool `json:"success"`

	// DeletedCount is the number of log entries deleted
	DeletedCount int64 `json:"deleted_count"`

	// DryRun indicates if this was a dry run (no actual deletion)
	DryRun bool `json:"dry_run"`

	// TimeRange of deleted logs
	DeletedRange *TimeRange `json:"deleted_range,omitempty"`

	// Retention policy applied
	Retention string `json:"retention"`

	// Execution timestamp
	Timestamp time.Time `json:"timestamp"`
}

// Validation error responses

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// APIError represents a standard API error response
type APIError struct {
	// Error details
	Error ErrorDetails `json:"error"`

	// Request context
	Path      string    `json:"path,omitempty"`
	Method    string    `json:"method,omitempty"`
	Timestamp time.Time `json:"timestamp"`

	// Request ID for tracking
	RequestID string `json:"request_id,omitempty"`
}

// ErrorDetails provides structured error information
type ErrorDetails struct {
	// Machine-readable error code
	Code string `json:"code"`

	// Human-readable error message
	Message string `json:"message"`

	// Additional error context
	Details interface{} `json:"details,omitempty"`

	// Field validation errors (if applicable)
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
}

// Common error codes for logging API
const (
	// Client errors (4xx)
	ErrorCodeInvalidJSON      = "invalid_json"
	ErrorCodeValidationFailed = "validation_failed"
	ErrorCodeUnauthorized     = "unauthorized"
	ErrorCodeForbidden        = "forbidden"
	ErrorCodeNotFound         = "not_found"
	ErrorCodeTooManyEntries   = "too_many_entries"
	ErrorCodeNoEntries        = "no_entries"
	ErrorCodeInvalidDuration  = "invalid_duration"
	ErrorCodeInvalidTimeRange = "invalid_time_range"

	// Server errors (5xx)
	ErrorCodeIngestionFailed  = "ingestion_failed"
	ErrorCodeSearchFailed     = "search_failed"
	ErrorCodeStatisticsFailed = "statistics_failed"
	ErrorCodeCleanupFailed    = "cleanup_failed"
	ErrorCodeDatabaseError    = "database_error"
	ErrorCodeInternalError    = "internal_error"
)

// HTTP status codes for different error types
var ErrorCodeToHTTPStatus = map[string]int{
	ErrorCodeInvalidJSON:      400, // Bad Request
	ErrorCodeValidationFailed: 400, // Bad Request
	ErrorCodeUnauthorized:     401, // Unauthorized
	ErrorCodeForbidden:        403, // Forbidden
	ErrorCodeNotFound:         404, // Not Found
	ErrorCodeTooManyEntries:   400, // Bad Request
	ErrorCodeNoEntries:        400, // Bad Request
	ErrorCodeInvalidDuration:  400, // Bad Request
	ErrorCodeInvalidTimeRange: 400, // Bad Request

	ErrorCodeIngestionFailed:  500, // Internal Server Error
	ErrorCodeSearchFailed:     500, // Internal Server Error
	ErrorCodeStatisticsFailed: 500, // Internal Server Error
	ErrorCodeCleanupFailed:    500, // Internal Server Error
	ErrorCodeDatabaseError:    500, // Internal Server Error
	ErrorCodeInternalError:    500, // Internal Server Error
}
