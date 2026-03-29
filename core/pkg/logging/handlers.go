package logging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// contextKey is the unexported type for context keys in this package.
type contextKey string

const (
	ctxKeyOrgID     contextKey = "organization_id"
	ctxKeyUserID    contextKey = "user_id"
	ctxKeySessionID contextKey = "session_id"
	ctxKeyRequestID contextKey = "request_id"
)

// Handler provides HTTP handlers for logging operations
type Handler struct {
	logService LogService
}

// NewHandler creates a new logging HTTP handler
func NewHandler(logService LogService) *Handler {
	return &Handler{
		logService: logService,
	}
}

// SetupRoutes configures the HTTP routes for logging API
func (h *Handler) SetupRoutes(r *mux.Router, authMiddleware AuthMiddleware) {
	// Log ingestion endpoints (for plugins and core services)
	logs := r.PathPrefix("/logs").Subrouter()
	logs.Use(authMiddleware.RequireAuth)
	logs.Use(authMiddleware.AuditMiddleware)

	// Log submission endpoint (requires logs:write permission)
	logs.Handle("", authMiddleware.RequirePermission("logs:write")(http.HandlerFunc(h.IngestLogs))).Methods("POST")

	// Log search endpoints (require logs:read permission)
	logs.Handle("/search", authMiddleware.RequirePermission("logs:read")(http.HandlerFunc(h.SearchLogs))).Methods("GET")

	// Log statistics for dashboards (require logs:read permission)
	logs.Handle("/statistics", authMiddleware.RequirePermission("logs:read")(http.HandlerFunc(h.GetStatistics))).Methods("GET")

	// Admin endpoints (require logs:admin permission)
	admin := logs.PathPrefix("/admin").Subrouter()
	admin.Handle("/cleanup", authMiddleware.RequirePermission("logs:admin")(http.HandlerFunc(h.DeleteOldLogs))).Methods("DELETE")
}

// AuthMiddleware interface for dependency injection (matches RBAC middleware)
type AuthMiddleware interface {
	RequireAuth(http.Handler) http.Handler
	RequirePermission(permission string) func(http.Handler) http.Handler
	AuditMiddleware(http.Handler) http.Handler
}

// IngestLogs handles batch log submission from plugins and core services
func (h *Handler) IngestLogs(w http.ResponseWriter, r *http.Request) {
	// Get organization from authenticated user context (set by auth middleware)
	orgID, err := h.getOrganizationFromContext(r)
	if err != nil {
		h.writeJSONError(w, http.StatusUnauthorized, "organization_required", "Organization context required")
		return
	}

	// Parse request body
	var request LogIngestionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON in request body")
		return
	}

	// Validate request
	if len(request.Entries) == 0 {
		h.writeJSONError(w, http.StatusBadRequest, "no_entries", "At least one log entry is required")
		return
	}

	if len(request.Entries) > 100 {
		h.writeJSONError(w, http.StatusBadRequest, "too_many_entries", "Maximum 100 log entries per request")
		return
	}

	// Add request context to log entries
	userID := h.getUserIDFromContext(r)
	sessionID := h.getSessionIDFromContext(r)
	requestID := h.getRequestIDFromContext(r)

	for i := range request.Entries {
		// Set context fields if not already provided
		if request.Entries[i].UserID == nil {
			if userID != nil {
				request.Entries[i].UserID = userID
			}
		}
		if request.Entries[i].SessionID == "" {
			if sessionID != "" {
				request.Entries[i].SessionID = sessionID
			}
		}
		if request.Entries[i].RequestID == "" {
			if requestID != "" {
				request.Entries[i].RequestID = requestID
			}
		}
	}

	// Process log ingestion
	response, err := h.logService.IngestLogs(r.Context(), orgID, request)
	if err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, "ingestion_failed", err.Error())
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	if response.Success {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusPartialContent) // Some entries failed
	}

	json.NewEncoder(w).Encode(response)
}

// SearchLogs handles log search requests with filtering and pagination
func (h *Handler) SearchLogs(w http.ResponseWriter, r *http.Request) {
	// Get organization from authenticated user context
	orgID, err := h.getOrganizationFromContext(r)
	if err != nil {
		h.writeJSONError(w, http.StatusUnauthorized, "organization_required", "Organization context required")
		return
	}

	// Parse query parameters
	filters := LogSearchRequest{}

	// Parse time range
	if startTimeStr := r.URL.Query().Get("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filters.StartTime = &startTime
		}
	}
	if endTimeStr := r.URL.Query().Get("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filters.EndTime = &endTime
		}
	}

	// Parse relative time ranges (e.g., "1h", "24h", "7d")
	if since := r.URL.Query().Get("since"); since != "" {
		if duration, err := h.parseDurationString(since); err == nil {
			startTime := time.Now().Add(-duration)
			filters.StartTime = &startTime
		}
	}

	// Parse log levels
	if levelsStr := r.URL.Query().Get("levels"); levelsStr != "" {
		levelStrings := strings.Split(levelsStr, ",")
		for _, levelStr := range levelStrings {
			level := LogLevel(strings.ToUpper(strings.TrimSpace(levelStr)))
			filters.Levels = append(filters.Levels, level)
		}
	}

	// Parse components
	if componentsStr := r.URL.Query().Get("components"); componentsStr != "" {
		filters.Components = strings.Split(componentsStr, ",")
		for i := range filters.Components {
			filters.Components[i] = strings.TrimSpace(filters.Components[i])
		}
	}

	// Parse user ID
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if userUUID, err := uuid.Parse(userIDStr); err == nil {
			filters.UserID = &userUUID
		}
	}

	// Parse plugin name
	filters.PluginName = r.URL.Query().Get("plugin_name")

	// Parse action
	filters.Action = r.URL.Query().Get("action")

	// Parse search text
	filters.SearchText = r.URL.Query().Get("q")

	// Parse pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			filters.Limit = limit
		}
	}
	if filters.Limit == 0 {
		filters.Limit = 50 // Default limit
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err == nil {
			if offset >= 0 {
				filters.Offset = offset
			}
		}
	}

	// Execute search
	response, err := h.logService.SearchLogs(r.Context(), orgID, filters)
	if err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, "search_failed", err.Error())
		return
	}

	// Return results
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetStatistics handles requests for log statistics and metrics
func (h *Handler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	// Get organization from authenticated user context
	orgID, err := h.getOrganizationFromContext(r)
	if err != nil {
		h.writeJSONError(w, http.StatusUnauthorized, "organization_required", "Organization context required")
		return
	}

	// Parse time range (default to last 24 hours)
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if startTimeStr := r.URL.Query().Get("start_time"); startTimeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = parsed
		}
	}
	if endTimeStr := r.URL.Query().Get("end_time"); endTimeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = parsed
		}
	}

	// Parse relative time range
	if since := r.URL.Query().Get("since"); since != "" {
		if duration, err := h.parseDurationString(since); err == nil {
			startTime = time.Now().Add(-duration)
			endTime = time.Now()
		}
	}

	// Get statistics
	statsRequest := LogStatisticsRequest{
		StartTime: &startTime,
		EndTime:   &endTime,
	}
	stats, err := h.logService.GetLogStatistics(r.Context(), orgID, statsRequest)
	if err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, "statistics_failed", err.Error())
		return
	}

	// Return statistics
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// DeleteOldLogs handles admin requests to clean up old log data
func (h *Handler) DeleteOldLogs(w http.ResponseWriter, r *http.Request) {
	// Parse retention period (default to 90 days as per schema)
	retentionStr := r.URL.Query().Get("older_than")
	if retentionStr == "" {
		retentionStr = "90d" // Default retention period
	}

	retention, err := h.parseDurationString(retentionStr)
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid_duration", "Invalid duration format")
		return
	}

	// Execute cleanup
	deletedCount, err := h.logService.DeleteOldLogs(r.Context(), retention)
	if err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, "cleanup_failed", err.Error())
		return
	}

	// Return cleanup results
	response := map[string]interface{}{
		"success":       true,
		"deleted_count": deletedCount,
		"retention":     retentionStr,
		"cleaned_at":    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Helper functions for context extraction (these would be set by auth middleware)

func (h *Handler) getOrganizationFromContext(r *http.Request) (uuid.UUID, error) {
	if orgIDStr := r.Header.Get("X-Organization-ID"); orgIDStr != "" {
		return uuid.Parse(orgIDStr)
	}
	// Fallback to context value set by auth middleware
	if orgID := r.Context().Value(ctxKeyOrgID); orgID != nil {
		if orgUUID, ok := orgID.(uuid.UUID); ok {
			return orgUUID, nil
		}
	}
	return uuid.Nil, fmt.Errorf("organization ID not found in request context")
}

func (h *Handler) getUserIDFromContext(r *http.Request) *uuid.UUID {
	if userIDStr := r.Header.Get("X-User-ID"); userIDStr != "" {
		if userUUID, err := uuid.Parse(userIDStr); err == nil {
			return &userUUID
		}
	}
	// Fallback to context value set by auth middleware
	if userID := r.Context().Value(ctxKeyUserID); userID != nil {
		if userUUID, ok := userID.(uuid.UUID); ok {
			return &userUUID
		}
	}
	return nil
}

func (h *Handler) getSessionIDFromContext(r *http.Request) string {
	if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
		return sessionID
	}
	if sessionID := r.Context().Value(ctxKeySessionID); sessionID != nil {
		if sessionStr, ok := sessionID.(string); ok {
			return sessionStr
		}
	}
	return ""
}

func (h *Handler) getRequestIDFromContext(r *http.Request) string {
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		return requestID
	}
	if requestID := r.Context().Value(ctxKeyRequestID); requestID != nil {
		if requestStr, ok := requestID.(string); ok {
			return requestStr
		}
	}
	return ""
}

// parseDurationString parses human-readable duration strings like "1h", "24h", "7d"
func (h *Handler) parseDurationString(durationStr string) (time.Duration, error) {
	// Handle common suffixes
	durationStr = strings.ToLower(strings.TrimSpace(durationStr))

	if strings.HasSuffix(durationStr, "d") {
		// Handle days
		daysStr := strings.TrimSuffix(durationStr, "d")
		if days, err := strconv.Atoi(daysStr); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}

	if strings.HasSuffix(durationStr, "w") {
		// Handle weeks
		weeksStr := strings.TrimSuffix(durationStr, "w")
		if weeks, err := strconv.Atoi(weeksStr); err == nil {
			return time.Duration(weeks) * 7 * 24 * time.Hour, nil
		}
	}

	// Use Go's standard duration parsing for other formats (1h, 30m, 45s, etc.)
	return time.ParseDuration(durationStr)
}

// writeJSONError writes a standardized JSON error response
func (h *Handler) writeJSONError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    errorCode,
			"message": message,
		},
		"timestamp": time.Now(),
	}

	json.NewEncoder(w).Encode(errorResponse)
}
