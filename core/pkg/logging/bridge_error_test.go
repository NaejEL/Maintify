package logging

import (
	"errors"
	"maintify/core/pkg/logger"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDatabaseHook_Fire_Error(t *testing.T) {
	mockService := new(MockLogService)
	hook := NewDatabaseHook(mockService)

	entry := logger.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Test message",
		Component: "test-component",
	}

	mockService.On("IngestLogs", mock.Anything, uuid.Nil, mock.Anything).Return((*LogIngestionResponse)(nil), errors.New("ingestion error"))

	err := hook.Fire(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to ingest log")
	mockService.AssertExpectations(t)
}

func TestDatabaseHook_Fire_OptionalFields(t *testing.T) {
	mockService := new(MockLogService)
	hook := NewDatabaseHook(mockService)

	userID := uuid.New()
	entry := logger.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    "Test message",
		Component:  "test-component",
		UserID:     userID.String(),
		SessionID:  "session-123",
		RequestID:  "req-123",
		PluginName: "plugin-abc",
		Action:     "create",
		Error:      "some error",
	}

	mockService.On("IngestLogs", mock.Anything, uuid.Nil, mock.MatchedBy(func(req LogIngestionRequest) bool {
		if len(req.Entries) != 1 {
			return false
		}
		e := req.Entries[0]
		return e.UserID != nil && *e.UserID == userID &&
			e.SessionID == "session-123" &&
			e.RequestID == "req-123" &&
			e.PluginName == "plugin-abc" &&
			e.Action == "create" &&
			e.ErrorMessage == "some error"
	})).Return(&LogIngestionResponse{Success: true}, nil)

	err := hook.Fire(entry)
	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}
