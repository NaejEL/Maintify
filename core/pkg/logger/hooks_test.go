package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestHook struct {
	Entries []LogEntry
}

func (h *TestHook) Fire(entry LogEntry) error {
	h.Entries = append(h.Entries, entry)
	return nil
}

func TestLogger_Hooks(t *testing.T) {
	config := Config{
		Level:      "INFO",
		Component:  "test",
		Structured: true,
		Console:    false,
	}

	logger, err := NewLogger(config)
	assert.NoError(t, err)

	hook := &TestHook{}
	logger.AddHook(hook)

	logger.Info("test message", map[string]interface{}{"key": "value"})

	assert.Len(t, hook.Entries, 1)
	assert.Equal(t, "test message", hook.Entries[0].Message)
	assert.Equal(t, "INFO", hook.Entries[0].Level)
	assert.Equal(t, "value", hook.Entries[0].Details["key"])
}
