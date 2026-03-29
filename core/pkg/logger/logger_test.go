package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Run("creates logger with console output", func(t *testing.T) {
		cfg := Config{
			Level:     "INFO",
			Component: "test",
			Console:   true,
		}
		l, err := NewLogger(cfg)
		require.NoError(t, err)
		assert.NotNil(t, l)
		assert.Len(t, l.outputs, 1)
	})

	t.Run("creates logger with file output", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := Config{
			Level:     "INFO",
			Component: "test",
			LogDir:    tmpDir,
		}
		l, err := NewLogger(cfg)
		require.NoError(t, err)
		assert.NotNil(t, l)
		assert.Len(t, l.outputs, 1)

		// Check file exists
		logFile := filepath.Join(tmpDir, "maintify-test.log")
		_, err = os.Stat(logFile)
		assert.NoError(t, err)
	})

	t.Run("fails if log dir cannot be created", func(t *testing.T) {
		// Use a file as dir to force error
		tmpFile := filepath.Join(t.TempDir(), "file")
		os.WriteFile(tmpFile, []byte("content"), 0644)

		cfg := Config{
			Level:     "INFO",
			Component: "test",
			LogDir:    tmpFile, // Should fail MkdirAll
		}
		l, err := NewLogger(cfg)
		assert.Error(t, err)
		assert.Nil(t, l)
	})
}

func TestLogger_Log(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		minLevel:  INFO,
		component: "test",
		outputs:   []io.Writer{&buf},
	}

	t.Run("logs info message", func(t *testing.T) {
		buf.Reset()
		l.Info("test message")
		assert.Contains(t, buf.String(), "test message")
		assert.Contains(t, buf.String(), "INFO")
	})

	t.Run("filters debug message", func(t *testing.T) {
		buf.Reset()
		l.Debug("debug message")
		assert.Empty(t, buf.String())
	})

	t.Run("logs error with details", func(t *testing.T) {
		buf.Reset()
		l.Error("error message", errors.New("some error"), map[string]interface{}{"key": "value"})
		output := buf.String()
		assert.Contains(t, output, "error message")
		assert.Contains(t, output, "ERROR")
		assert.Contains(t, output, "some error")
		assert.Contains(t, output, "key=value")
	})
}

func TestLogger_Structured(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		minLevel:   INFO,
		component:  "test",
		structured: true,
		outputs:    []io.Writer{&buf},
	}

	t.Run("logs json", func(t *testing.T) {
		buf.Reset()
		l.Info("json message", map[string]interface{}{"foo": "bar"})

		var entry LogEntry
		err := json.Unmarshal(buf.Bytes(), &entry)
		require.NoError(t, err)

		assert.Equal(t, "INFO", entry.Level)
		assert.Equal(t, "json message", entry.Message)
		assert.Equal(t, "test", entry.Component)
		assert.Equal(t, "bar", entry.Details["foo"])
	})
}

func TestLogger_Fatal(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		minLevel:  INFO,
		component: "test",
		outputs:   []io.Writer{&buf},
	}

	origExit := exitFunc
	defer func() { exitFunc = origExit }()

	exitCode := -1
	exitFunc = func(code int) {
		exitCode = code
	}

	l.Fatal("fatal error", nil)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, buf.String(), "FATAL")

	// Test with error
	buf.Reset()
	l.Fatal("fatal error with err", errors.New("oops"))
	assert.Contains(t, buf.String(), "oops")

	// Test with details
	buf.Reset()
	l.Fatal("fatal error with details", nil, map[string]interface{}{"k": "v"})
	assert.Contains(t, buf.String(), "k=v")
}

func TestGlobalFunctions(t *testing.T) {
	// Reset default logger
	DefaultLogger = nil

	// Test fallback to std log (we can't easily capture std log output without pipe,
	// but we can ensure it doesn't panic)
	Info("fallback info")

	// Initialize default logger
	var buf bytes.Buffer
	l := &Logger{
		minLevel:  INFO,
		component: "global",
		outputs:   []io.Writer{&buf},
	}
	DefaultLogger = l

	Info("global info")
	assert.Contains(t, buf.String(), "global info")

	buf.Reset()
	LogUserAction("user1", "sess1", "login", "user logged in", nil)
	assert.Contains(t, buf.String(), "user logged in")
	assert.Contains(t, buf.String(), "user_id=user1")
}

func TestLogLevel_String(t *testing.T) {
	assert.Equal(t, "DEBUG", DEBUG.String())
	assert.Equal(t, "INFO", INFO.String())
	assert.Equal(t, "UNKNOWN", LogLevel(99).String())
}

func TestParseLogLevel(t *testing.T) {
	assert.Equal(t, DEBUG, parseLogLevel("debug"))
	assert.Equal(t, INFO, parseLogLevel("INFO"))
	assert.Equal(t, WARN, parseLogLevel("WARN"))
	assert.Equal(t, WARN, parseLogLevel("WARNING"))
	assert.Equal(t, ERROR, parseLogLevel("error"))
	assert.Equal(t, FATAL, parseLogLevel("FATAL"))
	assert.Equal(t, INFO, parseLogLevel("invalid"))
}

func TestLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		minLevel:  INFO,
		component: "test",
		outputs:   []io.Writer{&buf},
	}

	l.Warn("warn message")
	assert.Contains(t, buf.String(), "warn message")
	assert.Contains(t, buf.String(), "WARN")
}

func TestLogger_Contextual(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		minLevel:  INFO,
		component: "test",
		outputs:   []io.Writer{&buf},
	}

	t.Run("LogPluginAction", func(t *testing.T) {
		buf.Reset()
		l.LogPluginAction("plugin1", "start", "plugin started", nil)
		output := buf.String()
		assert.Contains(t, output, "plugin started")
		assert.Contains(t, output, "plugin_name=plugin1")
		assert.Contains(t, output, "action=start")
	})

	t.Run("LogSecurityEvent", func(t *testing.T) {
		buf.Reset()
		l.LogSecurityEvent("login_failed", "invalid password", nil)
		output := buf.String()
		assert.Contains(t, output, "invalid password")
		assert.Contains(t, output, "security_event=login_failed")
		assert.Contains(t, output, "severity=high")
		assert.Contains(t, output, "WARN")
	})

	t.Run("LogSystemEvent", func(t *testing.T) {
		buf.Reset()
		l.LogSystemEvent("startup", "system started", nil)
		output := buf.String()
		assert.Contains(t, output, "system started")
		assert.Contains(t, output, "system_event=startup")
		assert.Contains(t, output, "INFO")
	})
}

func TestInitDefaultLogger(t *testing.T) {
	cfg := Config{
		Level:     "INFO",
		Component: "test",
		Console:   true,
	}
	err := InitDefaultLogger(cfg)
	require.NoError(t, err)
	assert.NotNil(t, DefaultLogger)
	assert.Equal(t, INFO, DefaultLogger.minLevel)
}

func TestGlobalConvenienceFunctions(t *testing.T) {
	// Setup default logger with buffer
	var buf bytes.Buffer
	l := &Logger{
		minLevel:  DEBUG,
		component: "global",
		outputs:   []io.Writer{&buf},
	}
	DefaultLogger = l

	t.Run("Debug", func(t *testing.T) {
		buf.Reset()
		Debug("debug msg")
		assert.Contains(t, buf.String(), "debug msg")
	})

	t.Run("Warn", func(t *testing.T) {
		buf.Reset()
		Warn("warn msg")
		assert.Contains(t, buf.String(), "warn msg")
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		Error("error msg", errors.New("oops"))
		assert.Contains(t, buf.String(), "error msg")
		assert.Contains(t, buf.String(), "oops")
	})

	t.Run("Fatal", func(t *testing.T) {
		buf.Reset()
		origExit := exitFunc
		defer func() { exitFunc = origExit }()
		exitFunc = func(code int) {}

		Fatal("fatal msg", nil)
		assert.Contains(t, buf.String(), "fatal msg")
	})

	t.Run("LogPluginAction", func(t *testing.T) {
		buf.Reset()
		LogPluginAction("p1", "a1", "msg", nil)
		assert.Contains(t, buf.String(), "plugin_name=p1")
	})

	t.Run("LogSecurityEvent", func(t *testing.T) {
		buf.Reset()
		LogSecurityEvent("evt", "msg", nil)
		assert.Contains(t, buf.String(), "security_event=evt")
	})

	t.Run("LogSystemEvent", func(t *testing.T) {
		buf.Reset()
		LogSystemEvent("evt", "msg", nil)
		assert.Contains(t, buf.String(), "system_event=evt")
	})
}

func TestGlobalFallback(t *testing.T) {
	DefaultLogger = nil

	// Just ensure these don't panic
	Debug("msg")
	Warn("msg")
	Error("msg", nil)
	LogPluginAction("p", "a", "m", nil)
	LogSecurityEvent("e", "m", nil)
	LogSystemEvent("e", "m", nil)
}

func TestLogger_VariadicDetails(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		minLevel:  DEBUG,
		component: "test",
		outputs:   []io.Writer{&buf},
	}

	t.Run("Debug with details", func(t *testing.T) {
		buf.Reset()
		l.Debug("msg", map[string]interface{}{"k": "v"})
		assert.Contains(t, buf.String(), "k=v")
	})

	t.Run("Warn with details", func(t *testing.T) {
		buf.Reset()
		l.Warn("msg", map[string]interface{}{"k": "v"})
		assert.Contains(t, buf.String(), "k=v")
	})
}

func TestLogger_JSONMarshalError(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		minLevel:   INFO,
		component:  "test",
		structured: true,
		outputs:    []io.Writer{&buf},
	}

	// Pass a value that cannot be marshaled to JSON (e.g. a channel)
	l.Info("msg", map[string]interface{}{"bad": make(chan int)})

	// Should fall back to text format
	assert.Contains(t, buf.String(), "msg")
	assert.Contains(t, buf.String(), "INFO")
}

func TestInitDefaultLogger_Error(t *testing.T) {
	// Use a file as dir to force error
	tmpFile := filepath.Join(t.TempDir(), "file")
	os.WriteFile(tmpFile, []byte("content"), 0644)

	cfg := Config{
		Level:     "INFO",
		Component: "test",
		LogDir:    tmpFile, // Should fail MkdirAll
	}
	err := InitDefaultLogger(cfg)
	assert.Error(t, err)
}

func TestGlobalFallback_NilError(t *testing.T) {
	DefaultLogger = nil

	Error("error msg without err", nil)

	origExit := exitFunc
	defer func() { exitFunc = origExit }()
	exitFunc = func(code int) {}

	Fatal("fatal msg without err", nil)
}

func TestGlobalFallback_WithError(t *testing.T) {
	DefaultLogger = nil

	Error("error msg with err", errors.New("oops"))

	origExit := exitFunc
	defer func() { exitFunc = origExit }()
	exitFunc = func(code int) {}

	Fatal("fatal msg with err", errors.New("oops"))
}

func TestGlobalFallback_UserAction(t *testing.T) {
	DefaultLogger = nil
	LogUserAction("u", "s", "a", "m", nil)
}

func TestSetupFileLogging_OpenFileError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory where the log file should be, to cause OpenFile to fail
	// Log file name is fmt.Sprintf("maintify-%s.log", l.component)
	// component = "test" -> maintify-test.log

	logFile := filepath.Join(tmpDir, "maintify-test.log")
	err := os.Mkdir(logFile, 0755) // Create dir with same name
	require.NoError(t, err)

	cfg := Config{
		Level:     "INFO",
		Component: "test",
		LogDir:    tmpDir,
	}

	l, err := NewLogger(cfg)
	assert.Error(t, err)
	assert.Nil(t, l)
	assert.Contains(t, err.Error(), "failed to open log file")
}

func TestGlobalAddHook(t *testing.T) {
	// Setup default logger
	l := &Logger{
		minLevel: DEBUG,
		hooks:    make([]Hook, 0),
	}
	DefaultLogger = l

	hook := &TestHook{}
	AddHook(hook)

	assert.Equal(t, 1, len(l.hooks))
}

func TestGlobalAddHook_NoDefaultLogger(t *testing.T) {
	DefaultLogger = nil
	hook := &TestHook{}
	// Should not panic
	AddHook(hook)
}
