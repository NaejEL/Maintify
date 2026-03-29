package logging

import (
	"context"
	"fmt"
	"maintify/core/pkg/logger"
	"time"

	"github.com/google/uuid"
)

// DatabaseHook implements logger.Hook to send logs to the database
type DatabaseHook struct {
	service LogService
}

// NewDatabaseHook creates a new DatabaseHook
func NewDatabaseHook(service LogService) *DatabaseHook {
	return &DatabaseHook{
		service: service,
	}
}

// Fire implements the logger.Hook interface
func (h *DatabaseHook) Fire(entry logger.LogEntry) error {
	// Extract OrganizationID from details if present
	var orgID uuid.UUID
	if val, ok := entry.Details["organization_id"]; ok {
		if idStr, ok := val.(string); ok {
			if parsed, err := uuid.Parse(idStr); err == nil {
				orgID = parsed
			}
		}
	}

	// Convert logger.LogEntry to logging.LogEntry
	logEntry := LogEntry{
		OrganizationID: orgID,
		Timestamp:      entry.Timestamp,
		Level:          LogLevel(entry.Level),
		Message:        entry.Message,
		Component:      entry.Component,
		Source:         entry.Source,
		Details:        entry.Details,
		CreatedAt:      time.Now(),
	}

	// Handle optional fields
	if entry.UserID != "" {
		if uid, err := uuid.Parse(entry.UserID); err == nil {
			logEntry.UserID = &uid
		}
	}
	if entry.SessionID != "" {
		logEntry.SessionID = entry.SessionID
	}
	if entry.RequestID != "" {
		logEntry.RequestID = entry.RequestID
	}
	if entry.PluginName != "" {
		logEntry.PluginName = entry.PluginName
	}
	if entry.Action != "" {
		logEntry.Action = entry.Action
	}
	if entry.Error != "" {
		logEntry.ErrorMessage = entry.Error
	}

	// Create ingestion request
	request := LogIngestionRequest{
		Entries: []LogEntry{logEntry},
		Source:  "logger-bridge",
	}

	// Ingest logs
	// Use a background context with timeout to avoid blocking the logger too long
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := h.service.IngestLogs(ctx, orgID, request)
	if err != nil {
		return fmt.Errorf("failed to ingest log: %w", err)
	}

	return nil
}
