package logging

import (
	"maintify/core/pkg/logger"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDatabaseHook_Fire(t *testing.T) {
	mockService := new(MockLogService)
	hook := NewDatabaseHook(mockService)

	// Test case 1: Log entry with Organization ID in details
	orgID := uuid.New()
	entry := logger.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Test message",
		Component: "test-component",
		Details: map[string]interface{}{
			"organization_id": orgID.String(),
			"custom_field":    "custom_value",
		},
	}

	mockService.On("IngestLogs", mock.Anything, orgID, mock.MatchedBy(func(req LogIngestionRequest) bool {
		return len(req.Entries) == 1 &&
			req.Entries[0].Message == "Test message" &&
			req.Entries[0].Level == LogLevelInfo &&
			req.Entries[0].Details["custom_field"] == "custom_value"
	})).Return(&LogIngestionResponse{Success: true}, nil)

	err := hook.Fire(entry)
	assert.NoError(t, err)
	mockService.AssertExpectations(t)

	// Test case 2: Log entry without Organization ID (should use system/nil UUID)
	entryNoOrg := logger.LogEntry{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Message:   "System error",
		Component: "core",
	}

	mockService.On("IngestLogs", mock.Anything, uuid.Nil, mock.MatchedBy(func(req LogIngestionRequest) bool {
		return len(req.Entries) == 1 &&
			req.Entries[0].Message == "System error" &&
			req.Entries[0].Level == LogLevelError
	})).Return(&LogIngestionResponse{Success: true}, nil)

	err = hook.Fire(entryNoOrg)
	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}
