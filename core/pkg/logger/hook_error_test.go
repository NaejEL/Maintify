package logger

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ErrorHook struct{}

func (h *ErrorHook) Fire(entry LogEntry) error {
	return errors.New("hook error")
}

func TestLogger_HookError(t *testing.T) {
	config := Config{
		Level:      "INFO",
		Component:  "test",
		Structured: true,
		Console:    false,
	}

	logger, err := NewLogger(config)
	assert.NoError(t, err)

	hook := &ErrorHook{}
	logger.AddHook(hook)

	// This should not panic, but print to stderr (which we can't easily capture here without redirecting os.Stderr)
	// But we can at least ensure it runs without crashing
	logger.Info("test message")
}
