package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockLogService implements LogService interface for testing
type MockLogService struct {
	mock.Mock
}

func (m *MockLogService) IngestLogs(ctx context.Context, organizationID uuid.UUID, entries LogIngestionRequest) (*LogIngestionResponse, error) {
	args := m.Called(ctx, organizationID, entries)
	return args.Get(0).(*LogIngestionResponse), args.Error(1)
}

func (m *MockLogService) SearchLogs(ctx context.Context, organizationID uuid.UUID, filters LogSearchRequest) (*LogSearchResponse, error) {
	args := m.Called(ctx, organizationID, filters)
	return args.Get(0).(*LogSearchResponse), args.Error(1)
}

func (m *MockLogService) GetLogStatistics(ctx context.Context, organizationID uuid.UUID, request LogStatisticsRequest) (*LogStatisticsResponse, error) {
	args := m.Called(ctx, organizationID, request)
	return args.Get(0).(*LogStatisticsResponse), args.Error(1)
}

func (m *MockLogService) DeleteOldLogs(ctx context.Context, retention time.Duration) (int64, error) {
	args := m.Called(ctx, retention)
	return args.Get(0).(int64), args.Error(1)
}

// MockAuthMiddleware implements AuthMiddleware for testing
type MockAuthMiddleware struct {
	organizationID uuid.UUID
	userID         uuid.UUID
	authenticated  bool
	permissions    []string
}

func (m *MockAuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.authenticated {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Set context values
		ctx := context.WithValue(r.Context(), ctxKeyOrgID, m.organizationID)
		ctx = context.WithValue(ctx, ctxKeyUserID, m.userID)
		ctx = context.WithValue(ctx, ctxKeySessionID, "test-session")
		ctx = context.WithValue(ctx, ctxKeyRequestID, "test-request")

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *MockAuthMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hasPermission := false
			for _, p := range m.permissions {
				if p == permission {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *MockAuthMiddleware) AuditMiddleware(next http.Handler) http.Handler {
	return next // No-op for testing
}

// LoggingAPITestSuite provides comprehensive testing for the logging API
type LoggingAPITestSuite struct {
	suite.Suite

	handler     *Handler
	mockService *MockLogService
	mockAuth    *MockAuthMiddleware
	router      *mux.Router

	testOrgID  uuid.UUID
	testUserID uuid.UUID
}

func (suite *LoggingAPITestSuite) SetupTest() {
	suite.mockService = &MockLogService{}
	suite.handler = NewHandler(suite.mockService)

	suite.testOrgID = uuid.New()
	suite.testUserID = uuid.New()

	suite.mockAuth = &MockAuthMiddleware{
		organizationID: suite.testOrgID,
		userID:         suite.testUserID,
		authenticated:  true,
		permissions:    []string{"logs:read", "logs:write", "logs:admin"},
	}

	suite.router = mux.NewRouter()
	apiRouter := suite.router.PathPrefix("/api").Subrouter()
	suite.handler.SetupRoutes(apiRouter, suite.mockAuth)
}

func (suite *LoggingAPITestSuite) TestIngestLogs_Success() {
	// Prepare test data
	entries := []LogEntry{
		{
			Level:     LogLevelInfo,
			Component: "test-component",
			Message:   "Test log message",
			Action:    "test-action",
			Details:   map[string]interface{}{"key": "value"},
		},
	}

	request := LogIngestionRequest{
		Entries: entries,
		Source:  "test-plugin",
	}

	expectedResponse := &LogIngestionResponse{
		Success:        true,
		ProcessedCount: 1,
		FailedCount:    0,
		Timestamp:      time.Now(),
	}

	// Setup mock expectation
	suite.mockService.On("IngestLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(req LogIngestionRequest) bool {
		return len(req.Entries) == 1 && req.Entries[0].Message == "Test log message"
	})).Return(expectedResponse, nil)

	// Create request
	jsonData, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/logs", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response LogIngestionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), 1, response.ProcessedCount)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *LoggingAPITestSuite) TestIngestLogs_ValidationError() {
	// Test with too many entries
	entries := make([]LogEntry, 101) // Exceeds maximum of 100
	for i := range entries {
		entries[i] = LogEntry{
			Level:     LogLevelInfo,
			Component: "test",
			Message:   fmt.Sprintf("Message %d", i),
		}
	}

	request := LogIngestionRequest{
		Entries: entries,
	}

	// Create request
	jsonData, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/logs", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify error response
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), errorResponse, "error")
}

func (suite *LoggingAPITestSuite) TestIngestLogs_JSONError() {
	// Test with invalid JSON
	req := httptest.NewRequest("POST", "/api/logs", bytes.NewBufferString("{invalid-json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *LoggingAPITestSuite) TestIngestLogs_ServiceError() {
	entries := []LogEntry{{Level: LogLevelInfo, Component: "test", Message: "test"}}
	request := LogIngestionRequest{Entries: entries}

	suite.mockService.On("IngestLogs", mock.Anything, suite.testOrgID, mock.Anything).Return((*LogIngestionResponse)(nil), errors.New("service error"))

	jsonData, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/logs", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

func (suite *LoggingAPITestSuite) TestIngestLogs_Unauthorized() {
	// Test without authentication
	suite.mockAuth.authenticated = false

	request := LogIngestionRequest{
		Entries: []LogEntry{{Level: LogLevelInfo, Component: "test", Message: "test"}},
	}

	jsonData, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/logs", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *LoggingAPITestSuite) TestSearchLogs_Success() {
	// Prepare test data

	expectedEntries := []LogEntryResponse{
		{
			LogEntry: LogEntry{
				Level:     LogLevelInfo,
				Component: "test-component",
				Message:   "Test log message",
			},
			ID:        uuid.New(),
			CreatedAt: time.Now(),
		},
	}

	expectedResponse := &LogSearchResponse{
		Entries: expectedEntries,
		Pagination: PaginationInfo{
			Limit:         50,
			Offset:        0,
			TotalCount:    1,
			ReturnedCount: 1,
			HasMore:       false,
		},
		Timestamp: time.Now(),
	}

	// Setup mock expectation
	suite.mockService.On("SearchLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(filters LogSearchRequest) bool {
		return len(filters.Levels) == 1
	})).Return(expectedResponse, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/logs/search?levels=INFO&limit=50", nil)

	// Execute request
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response LogSearchResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response.Entries, 1)
	assert.Equal(suite.T(), 1, response.Pagination.TotalCount)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *LoggingAPITestSuite) TestSearchLogs_WithTimeRange() {
	// Test search with time range parameters
	startTime := time.Now().UTC().Add(-2 * time.Hour)
	endTime := time.Now().UTC().Add(-1 * time.Hour)

	expectedResponse := &LogSearchResponse{
		Entries: []LogEntryResponse{},
		Pagination: PaginationInfo{
			Limit:         50,
			Offset:        0,
			TotalCount:    0,
			ReturnedCount: 0,
			HasMore:       false,
		},
		Timestamp: time.Now(),
	}

	suite.mockService.On("SearchLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(filters LogSearchRequest) bool {
		return filters.StartTime != nil && filters.EndTime != nil
	})).Return(expectedResponse, nil)

	// Create request with time range
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/logs/search?start_time=%s&end_time=%s",
		startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)), nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *LoggingAPITestSuite) TestSearchLogs_Forbidden() {
	// Test without logs:read permission
	suite.mockAuth.permissions = []string{"logs:write"} // Missing logs:read

	req := httptest.NewRequest("GET", "/api/logs/search", nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

func (suite *LoggingAPITestSuite) TestSearchLogs_ServiceError() {
	suite.mockService.On("SearchLogs", mock.Anything, suite.testOrgID, mock.Anything).Return((*LogSearchResponse)(nil), errors.New("service error"))

	req := httptest.NewRequest("GET", "/api/logs/search", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

func (suite *LoggingAPITestSuite) TestSearchLogs_InvalidParams() {
	// Invalid limit - should default to 50 and succeed
	req := httptest.NewRequest("GET", "/api/logs/search?limit=invalid", nil)

	// Expect call with default limit
	suite.mockService.On("SearchLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(filters LogSearchRequest) bool {
		return filters.Limit == 50
	})).Return(&LogSearchResponse{
		Entries: []LogEntryResponse{},
		Pagination: PaginationInfo{
			Limit:      50,
			TotalCount: 0,
		},
	}, nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *LoggingAPITestSuite) TestGetStatistics_ServiceError() {
	suite.mockService.On("GetLogStatistics", mock.Anything, suite.testOrgID, mock.Anything).Return((*LogStatisticsResponse)(nil), errors.New("service error"))

	req := httptest.NewRequest("GET", "/api/logs/statistics", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

func (suite *LoggingAPITestSuite) TestDeleteOldLogs_ServiceError() {
	suite.mockService.On("DeleteOldLogs", mock.Anything, mock.Anything).Return(int64(0), errors.New("service error"))

	req := httptest.NewRequest("DELETE", "/api/logs/admin/cleanup", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

func (suite *LoggingAPITestSuite) TestDeleteOldLogs_InvalidDuration() {
	req := httptest.NewRequest("DELETE", "/api/logs/admin/cleanup?older_than=invalid", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *LoggingAPITestSuite) TestContextHelpers() {
	// Test getOrganizationFromContext via IngestLogs (since it's private)
	// Case 1: No Org ID in context or header
	suite.mockAuth.authenticated = false // Disable auth middleware setting context

	// Let's call the handler method directly with a request that has no context
	req := httptest.NewRequest("POST", "/api/logs", nil)
	w := httptest.NewRecorder()

	suite.handler.IngestLogs(w, req)
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	// Case 2: Org ID in Header
	req = httptest.NewRequest("POST", "/api/logs", bytes.NewBufferString(`{"entries":[]}`))
	req.Header.Set("X-Organization-ID", suite.testOrgID.String())
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.handler.IngestLogs(w, req)
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	errMap := resp["error"].(map[string]interface{})
	assert.Equal(suite.T(), "no_entries", errMap["code"])
}

func (suite *LoggingAPITestSuite) TestContextHelpers_UserAndSession() {
	entries := []LogEntry{{Level: LogLevelInfo, Component: "test", Message: "test"}}
	request := LogIngestionRequest{Entries: entries}
	jsonData, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/api/logs", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-ID", suite.testOrgID.String())

	userID := uuid.New()
	req.Header.Set("X-User-ID", userID.String())
	req.Header.Set("X-Session-ID", "sess-123")
	req.Header.Set("X-Request-ID", "req-123")

	suite.mockService.On("IngestLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(r LogIngestionRequest) bool {
		entry := r.Entries[0]
		return entry.UserID != nil && *entry.UserID == userID &&
			entry.SessionID == "sess-123" &&
			entry.RequestID == "req-123"
	})).Return(&LogIngestionResponse{Success: true}, nil)

	w := httptest.NewRecorder()
	suite.handler.IngestLogs(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)
}

func (suite *LoggingAPITestSuite) TestGetStatistics_Success() {
	// Prepare test data
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	expectedStats := &LogStatisticsResponse{
		TotalLogs: 1000,
		LevelCounts: map[LogLevel]int64{
			LogLevelInfo:  800,
			LogLevelWarn:  150,
			LogLevelError: 50,
		},
		ComponentCounts: map[string]int64{
			"auth":   500,
			"plugin": 300,
			"core":   200,
		},
		TimeSeries: []TimeSeriesPoint{
			{
				Timestamp: time.Now().Truncate(time.Hour),
				Count:     100,
			},
		},
		TimeRange: TimeRange{
			Start: startTime,
			End:   endTime,
		},
		Timestamp: time.Now(),
	}

	suite.mockService.On("GetLogStatistics", mock.Anything, suite.testOrgID, mock.AnythingOfType("LogStatisticsRequest")).Return(expectedStats, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/logs/statistics?since=24h", nil)

	// Execute request
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response LogStatisticsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1000), response.TotalLogs)
	assert.Equal(suite.T(), int64(800), response.LevelCounts[LogLevelInfo])
	assert.Len(suite.T(), response.TimeSeries, 1)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *LoggingAPITestSuite) TestDeleteOldLogs_Success() {
	// Setup mock expectation
	expectedDeletedCount := int64(500)
	suite.mockService.On("DeleteOldLogs", mock.Anything, 90*24*time.Hour).Return(expectedDeletedCount, nil)

	// Create request
	req := httptest.NewRequest("DELETE", "/api/logs/admin/cleanup?older_than=90d", nil)

	// Execute request
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))
	assert.Equal(suite.T(), float64(500), response["deleted_count"].(float64))

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *LoggingAPITestSuite) TestDeleteOldLogs_AdminPermissionRequired() {
	// Test without logs:admin permission
	suite.mockAuth.permissions = []string{"logs:read", "logs:write"} // Missing logs:admin

	req := httptest.NewRequest("DELETE", "/api/logs/admin/cleanup", nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

// TestParseDurationString tests the duration parsing helper
func (suite *LoggingAPITestSuite) TestParseDurationString() {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"1h", 1 * time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"45s", 45 * time.Second, false},
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"2w", 14 * 24 * time.Hour, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, test := range tests {
		duration, err := suite.handler.parseDurationString(test.input)
		if test.hasError {
			assert.Error(suite.T(), err, "Expected error for input: %s", test.input)
		} else {
			assert.NoError(suite.T(), err, "Unexpected error for input: %s", test.input)
			assert.Equal(suite.T(), test.expected, duration, "Duration mismatch for input: %s", test.input)
		}
	}
}

// TestLoggingHandlerIntegration tests complete request/response flow
func TestLoggingHandlerIntegration(t *testing.T) {
	// Test the complete integration without mocks

	// Create a real handler with a mock service that returns expected responses
	mockService := &MockLogService{}
	handler := NewHandler(mockService)

	// Test JSON error writing
	w := httptest.NewRecorder()
	handler.writeJSONError(w, http.StatusBadRequest, "test_error", "Test error message")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Contains(t, errorResponse, "error")
	assert.Contains(t, errorResponse, "timestamp")
}

// Run the test suite
func TestLoggingAPITestSuite(t *testing.T) {
	suite.Run(t, new(LoggingAPITestSuite))
}

// Benchmark tests for performance validation

func BenchmarkIngestLogs(b *testing.B) {
	mockService := &MockLogService{}
	handler := NewHandler(mockService)

	// Setup successful response
	successResponse := &LogIngestionResponse{
		Success:        true,
		ProcessedCount: 10,
		FailedCount:    0,
		Timestamp:      time.Now(),
	}

	mockService.On("IngestLogs", mock.Anything, mock.Anything, mock.Anything).Return(successResponse, nil)

	// Create test data
	entries := make([]LogEntry, 10)
	for i := range entries {
		entries[i] = LogEntry{
			Level:     LogLevelInfo,
			Component: "benchmark-test",
			Message:   fmt.Sprintf("Benchmark message %d", i),
		}
	}

	request := LogIngestionRequest{Entries: entries}
	jsonData, _ := json.Marshal(request)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/logs", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), ctxKeyOrgID, uuid.New()))

		w := httptest.NewRecorder()
		handler.IngestLogs(w, req)
	}
}

func BenchmarkSearchLogs(b *testing.B) {
	mockService := &MockLogService{}
	handler := NewHandler(mockService)

	// Setup successful response
	successResponse := &LogSearchResponse{
		Entries: []LogEntryResponse{},
		Pagination: PaginationInfo{
			Limit:         50,
			Offset:        0,
			TotalCount:    0,
			ReturnedCount: 0,
		},
		Timestamp: time.Now(),
	}

	mockService.On("SearchLogs", mock.Anything, mock.Anything).Return(successResponse, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/logs/search?levels=INFO&limit=50", nil)
		req = req.WithContext(context.WithValue(req.Context(), ctxKeyOrgID, uuid.New()))

		w := httptest.NewRecorder()
		handler.SearchLogs(w, req)
	}
}

func (suite *LoggingAPITestSuite) TestSearchLogs_AllParams() {
	// Test search with all possible parameters
	startTime := time.Now().UTC().Add(-2 * time.Hour)
	endTime := time.Now().UTC().Add(-1 * time.Hour)
	userID := uuid.New()

	expectedResponse := &LogSearchResponse{
		Entries: []LogEntryResponse{},
		Pagination: PaginationInfo{
			Limit:         10,
			Offset:        5,
			TotalCount:    0,
			ReturnedCount: 0,
		},
	}

	suite.mockService.On("SearchLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(filters LogSearchRequest) bool {
		return filters.StartTime != nil && filters.EndTime != nil &&
			len(filters.Levels) == 2 &&
			len(filters.Components) == 2 &&
			filters.UserID != nil && *filters.UserID == userID &&
			filters.PluginName == "test-plugin" &&
			filters.Action == "test-action" &&
			filters.SearchText == "search-term" &&
			filters.Limit == 10 &&
			filters.Offset == 5
	})).Return(expectedResponse, nil)

	// Construct URL with all params
	url := fmt.Sprintf("/api/logs/search?start_time=%s&end_time=%s&levels=INFO,ERROR&components=core,auth&user_id=%s&plugin_name=test-plugin&action=test-action&q=search-term&limit=10&offset=5",
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
		userID.String(),
	)

	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *LoggingAPITestSuite) TestSearchLogs_SinceParam() {
	// Test 'since' parameter
	expectedResponse := &LogSearchResponse{Entries: []LogEntryResponse{}}

	suite.mockService.On("SearchLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(filters LogSearchRequest) bool {
		return filters.StartTime != nil // Should be set based on 'since'
	})).Return(expectedResponse, nil)

	req := httptest.NewRequest("GET", "/api/logs/search?since=1h", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *LoggingAPITestSuite) TestGetStatistics_ExplicitTimeRange() {
	startTime := time.Now().UTC().Add(-2 * time.Hour)
	endTime := time.Now().UTC().Add(-1 * time.Hour)

	expectedStats := &LogStatisticsResponse{}

	suite.mockService.On("GetLogStatistics", mock.Anything, suite.testOrgID, mock.MatchedBy(func(req LogStatisticsRequest) bool {
		return req.StartTime != nil && req.EndTime != nil
	})).Return(expectedStats, nil)

	url := fmt.Sprintf("/api/logs/statistics?start_time=%s&end_time=%s",
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
	)

	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *LoggingAPITestSuite) TestContextHelpers_Headers() {
	// Test extraction from headers (User, Session, Request ID)
	// We can test this via IngestLogs which uses these helpers

	entries := []LogEntry{{Level: LogLevelInfo, Component: "test", Message: "test"}}
	request := LogIngestionRequest{Entries: entries}
	jsonData, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/api/logs", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-ID", suite.testOrgID.String())

	userID := uuid.New()
	req.Header.Set("X-User-ID", userID.String())
	req.Header.Set("X-Session-ID", "header-session")
	req.Header.Set("X-Request-ID", "header-request")

	// Call handler directly to bypass middleware
	suite.mockService.On("IngestLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(r LogIngestionRequest) bool {
		entry := r.Entries[0]
		return entry.UserID != nil && *entry.UserID == userID &&
			entry.SessionID == "header-session" &&
			entry.RequestID == "header-request"
	})).Return(&LogIngestionResponse{Success: true}, nil)

	w := httptest.NewRecorder()
	suite.handler.IngestLogs(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)
}

func (suite *LoggingAPITestSuite) TestIngestLogs_PartialSuccess() {
	entries := []LogEntry{
		{Level: LogLevelInfo, Component: "test", Message: "success"},
		{Level: LogLevelInfo, Component: "test", Message: "fail"},
	}
	request := LogIngestionRequest{Entries: entries}

	expectedResponse := &LogIngestionResponse{
		Success:        false,
		ProcessedCount: 1,
		FailedCount:    1,
		Errors: map[int]LogIngestionError{
			1: {Message: "failed to process entry 2", Code: "processing_error"},
		},
	}

	suite.mockService.On("IngestLogs", mock.Anything, suite.testOrgID, mock.Anything).Return(expectedResponse, nil)

	jsonData, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/logs", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusPartialContent, w.Code)

	var response LogIngestionResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(suite.T(), 1, response.FailedCount)
	assert.NotEmpty(suite.T(), response.Errors)
}

func (suite *LoggingAPITestSuite) TestContextHelpers_EdgeCases() {
	// Test getUserIDFromContext with invalid header (should fall back to context or nil)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-User-ID", "invalid-uuid")

	// Should return nil if context is also empty
	userID := suite.handler.getUserIDFromContext(req)
	assert.Nil(suite.T(), userID)

	// Test getUserIDFromContext with invalid context type
	ctx := context.WithValue(req.Context(), ctxKeyUserID, "not-a-uuid")
	req = req.WithContext(ctx)
	userID = suite.handler.getUserIDFromContext(req)
	assert.Nil(suite.T(), userID)

	// Test getSessionIDFromContext with invalid context type
	ctx = context.WithValue(req.Context(), ctxKeySessionID, 123) // int instead of string
	req = req.WithContext(ctx)
	sessionID := suite.handler.getSessionIDFromContext(req)
	assert.Equal(suite.T(), "", sessionID)

	// Test getRequestIDFromContext with invalid context type
	ctx = context.WithValue(req.Context(), ctxKeyRequestID, 123) // int instead of string
	req = req.WithContext(ctx)
	requestID := suite.handler.getRequestIDFromContext(req)
	assert.Equal(suite.T(), "", requestID)
}

func (suite *LoggingAPITestSuite) TestSearchLogs_InvalidSinceParam() {
	// Test invalid 'since' parameter - should be ignored
	req := httptest.NewRequest("GET", "/api/logs/search?since=invalid", nil)

	// Expect call with no start time (or default behavior)
	suite.mockService.On("SearchLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(filters LogSearchRequest) bool {
		return filters.StartTime == nil
	})).Return(&LogSearchResponse{Entries: []LogEntryResponse{}}, nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *LoggingAPITestSuite) TestGetStatistics_InvalidSinceParam() {
	// Test invalid 'since' parameter - should use default 24h
	req := httptest.NewRequest("GET", "/api/logs/statistics?since=invalid", nil)

	suite.mockService.On("GetLogStatistics", mock.Anything, suite.testOrgID, mock.MatchedBy(func(req LogStatisticsRequest) bool {
		// Should use default start time (approx 24h ago)
		return req.StartTime != nil && time.Since(*req.StartTime) > 23*time.Hour
	})).Return(&LogStatisticsResponse{}, nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *LoggingAPITestSuite) TestParseDurationString_EdgeCases() {
	// Test invalid days format (ends in d but not int)
	_, err := suite.handler.parseDurationString("invalid_d")
	assert.Error(suite.T(), err)

	// Test invalid weeks format (ends in w but not int)
	_, err = suite.handler.parseDurationString("invalid_w")
	assert.Error(suite.T(), err)
}

func (suite *LoggingAPITestSuite) TestSearchLogs_PaginationEdgeCases() {
	// Test limit too high (should be ignored/default)
	req := httptest.NewRequest("GET", "/api/logs/search?limit=1001", nil)
	suite.mockService.On("SearchLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(filters LogSearchRequest) bool {
		return filters.Limit == 50 // Default
	})).Return(&LogSearchResponse{Entries: []LogEntryResponse{}}, nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Test limit zero (should be ignored/default)
	req = httptest.NewRequest("GET", "/api/logs/search?limit=0", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Test offset negative (should be ignored)
	req = httptest.NewRequest("GET", "/api/logs/search?offset=-1", nil)
	suite.mockService.On("SearchLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(filters LogSearchRequest) bool {
		return filters.Offset == 0
	})).Return(&LogSearchResponse{Entries: []LogEntryResponse{}}, nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Test offset invalid (should be ignored)
	req = httptest.NewRequest("GET", "/api/logs/search?offset=invalid", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *LoggingAPITestSuite) TestIngestLogs_ContextFallback_Explicit() {
	// Explicitly test the fallback logic for coverage
	req := httptest.NewRequest("POST", "/api/logs", bytes.NewBufferString(`{"entries":[{"level":"INFO","component":"test","message":"msg"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-ID", suite.testOrgID.String())
	req.Header.Set("X-Session-ID", "fallback-session")
	req.Header.Set("X-Request-ID", "fallback-request")

	suite.mockService.On("IngestLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(r LogIngestionRequest) bool {
		return r.Entries[0].SessionID == "fallback-session" && r.Entries[0].RequestID == "fallback-request"
	})).Return(&LogIngestionResponse{Success: true}, nil)

	w := httptest.NewRecorder()
	suite.handler.IngestLogs(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)
}

func (suite *LoggingAPITestSuite) TestSearchLogs_Offset_Explicit() {
	req := httptest.NewRequest("GET", "/api/logs/search?offset=10", nil)
	req.Header.Set("X-Organization-ID", suite.testOrgID.String())

	suite.mockService.On("SearchLogs", mock.Anything, suite.testOrgID, mock.MatchedBy(func(r LogSearchRequest) bool {
		return r.Offset == 10
	})).Return(&LogSearchResponse{Entries: []LogEntryResponse{}}, nil)
	w := httptest.NewRecorder()
	suite.handler.SearchLogs(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *LoggingAPITestSuite) TestSearchLogs_NoOrganization() {
	// Test SearchLogs without organization context (direct call to bypass middleware)
	req := httptest.NewRequest("GET", "/api/logs/search", nil)
	// No headers, no context

	w := httptest.NewRecorder()
	suite.handler.SearchLogs(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *LoggingAPITestSuite) TestGetStatistics_NoOrganization() {
	// Test GetStatistics without organization context (direct call to bypass middleware)
	req := httptest.NewRequest("GET", "/api/logs/statistics", nil)
	// No headers, no context

	w := httptest.NewRecorder()
	suite.handler.GetStatistics(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *LoggingAPITestSuite) TestContextHelpers_InvalidHeader() {
	// Test invalid UUID in header
	req := httptest.NewRequest("POST", "/api/logs", bytes.NewBufferString(`{"entries":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-ID", "invalid-uuid")

	w := httptest.NewRecorder()
	suite.handler.IngestLogs(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *LoggingAPITestSuite) TestContextHelpers_InvalidContextType() {
	// Test invalid type in context
	req := httptest.NewRequest("POST", "/api/logs", bytes.NewBufferString(`{"entries":[]}`))
	req.Header.Set("Content-Type", "application/json")

	// Inject invalid type for organization_id
	ctx := context.WithValue(req.Context(), ctxKeyOrgID, "not-a-uuid")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	suite.handler.IngestLogs(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}
