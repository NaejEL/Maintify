package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	// For testing
	exitFunc = os.Exit
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var logLevelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

func (l LogLevel) String() string {
	if name, exists := logLevelNames[l]; exists {
		return name
	}
	return "UNKNOWN"
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Source     string                 `json:"source"`
	Component  string                 `json:"component"`
	UserID     string                 `json:"user_id,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	PluginName string                 `json:"plugin_name,omitempty"`
	Action     string                 `json:"action,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// Hook is an interface for custom log handling
type Hook interface {
	Fire(entry LogEntry) error
}

// Logger provides structured logging capabilities for Maintify administrators
type Logger struct {
	mu          sync.RWMutex
	minLevel    LogLevel
	outputs     []io.Writer
	hooks       []Hook
	component   string
	structured  bool
	logDir      string
	maxFileSize int64 // in bytes
	maxFiles    int
}

// Config holds logger configuration
type Config struct {
	Level       string `json:"level"`         // DEBUG, INFO, WARN, ERROR, FATAL
	Component   string `json:"component"`     // core, builder, plugin_name
	Structured  bool   `json:"structured"`    // JSON output vs plain text
	LogDir      string `json:"log_dir"`       // Directory for log files
	MaxFileSize int64  `json:"max_file_size"` // Max size per log file in bytes
	MaxFiles    int    `json:"max_files"`     // Max number of log files to keep
	Console     bool   `json:"console"`       // Also log to console
}

// NewLogger creates a new logger instance
func NewLogger(config Config) (*Logger, error) {
	level := parseLogLevel(config.Level)

	logger := &Logger{
		minLevel:    level,
		component:   config.Component,
		structured:  config.Structured,
		logDir:      config.LogDir,
		maxFileSize: config.MaxFileSize,
		maxFiles:    config.MaxFiles,
		outputs:     make([]io.Writer, 0),
	}

	// Add console output if requested
	if config.Console {
		logger.outputs = append(logger.outputs, os.Stdout)
	}

	// Set up file logging if log directory is specified
	if config.LogDir != "" {
		if err := logger.setupFileLogging(); err != nil {
			return nil, fmt.Errorf("failed to setup file logging: %w", err)
		}
	}

	return logger, nil
}

// setupFileLogging creates the log directory and sets up file rotation
func (l *Logger) setupFileLogging() error {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(l.logDir, 0750); err != nil { // #nosec G301 -- 0750 allows group read for service accounts
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create current log file
	logFile := filepath.Join(l.logDir, fmt.Sprintf("maintify-%s.log", l.component))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640) // #nosec G302 G304 -- 0640 intentional; logFile path is admin-configured, not user input
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.outputs = append(l.outputs, file)
	return nil
}

// AddHook adds a hook to the logger
func (l *Logger) AddHook(hook Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = append(l.hooks, hook)
}

// parseLogLevel converts string level to LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// log writes a log entry with the specified level
func (l *Logger) log(level LogLevel, message string, details map[string]interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if level < l.minLevel {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level.String(),
		Message:   message,
		Component: l.component,
		Details:   details,
	}

	// Add source information for debugging
	if level <= DEBUG {
		if _, file, line, ok := runtime.Caller(2); ok {
			entry.Source = fmt.Sprintf("%s:%d", filepath.Base(file), line)
		}
	}

	// Fire hooks
	for _, hook := range l.hooks {
		if err := hook.Fire(entry); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fire hook: %v\n", err)
		}
	}

	l.writeEntry(entry)
}

// writeEntry writes the log entry to all outputs
func (l *Logger) writeEntry(entry LogEntry) {
	var output string

	if l.structured {
		// JSON output
		jsonBytes, err := json.Marshal(entry)
		if err != nil {
			// Fallback to simple output if JSON marshaling fails
			output = fmt.Sprintf("[%s] %s %s: %s\n",
				entry.Timestamp.Format(time.RFC3339),
				entry.Level,
				entry.Component,
				entry.Message)
		} else {
			output = string(jsonBytes) + "\n"
		}
	} else {
		// Human-readable output
		output = fmt.Sprintf("[%s] %s %s: %s",
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Level,
			entry.Component,
			entry.Message)

		if entry.Error != "" {
			output += fmt.Sprintf(" | Error: %s", entry.Error)
		}

		if len(entry.Details) > 0 {
			output += " | Details: "
			for k, v := range entry.Details {
				output += fmt.Sprintf("%s=%v ", k, v)
			}
		}
		output += "\n"
	}

	for _, writer := range l.outputs {
		writer.Write([]byte(output))
	}
}

// Public logging methods
func (l *Logger) Debug(message string, details ...map[string]interface{}) {
	var d map[string]interface{}
	if len(details) > 0 {
		d = details[0]
	}
	l.log(DEBUG, message, d)
}

func (l *Logger) Info(message string, details ...map[string]interface{}) {
	var d map[string]interface{}
	if len(details) > 0 {
		d = details[0]
	}
	l.log(INFO, message, d)
}

func (l *Logger) Warn(message string, details ...map[string]interface{}) {
	var d map[string]interface{}
	if len(details) > 0 {
		d = details[0]
	}
	l.log(WARN, message, d)
}

func (l *Logger) Error(message string, err error, details ...map[string]interface{}) {
	var d map[string]interface{}
	if len(details) > 0 {
		d = details[0]
	} else {
		d = make(map[string]interface{})
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     ERROR.String(),
		Message:   message,
		Component: l.component,
		Details:   d,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	l.writeEntry(entry)
}

func (l *Logger) Fatal(message string, err error, details ...map[string]interface{}) {
	var d map[string]interface{}
	if len(details) > 0 {
		d = details[0]
	} else {
		d = make(map[string]interface{})
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     FATAL.String(),
		Message:   message,
		Component: l.component,
		Details:   d,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	l.writeEntry(entry)

	// Fatal logs should terminate the program
	exitFunc(1)
}

// Contextual logging methods for administrative actions
func (l *Logger) LogUserAction(userID, sessionID, action, message string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["user_id"] = userID
	details["session_id"] = sessionID
	details["action"] = action

	l.log(INFO, message, details)
}

func (l *Logger) LogPluginAction(pluginName, action, message string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["plugin_name"] = pluginName
	details["action"] = action

	l.log(INFO, message, details)
}

func (l *Logger) LogSecurityEvent(event, message string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["security_event"] = event
	details["severity"] = "high"

	l.log(WARN, message, details)
}

func (l *Logger) LogSystemEvent(event, message string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["system_event"] = event

	l.log(INFO, message, details)
}

// DefaultLogger provides a global logger instance
var DefaultLogger *Logger

// InitDefaultLogger initializes the global logger
func InitDefaultLogger(config Config) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}
	DefaultLogger = logger
	return nil
}

// Convenience functions using the default logger
func Debug(message string, details ...map[string]interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Debug(message, details...)
	} else {
		log.Printf("[DEBUG] %s", message)
	}
}

func Info(message string, details ...map[string]interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Info(message, details...)
	} else {
		log.Printf("[INFO] %s", message)
	}
}

func Warn(message string, details ...map[string]interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Warn(message, details...)
	} else {
		log.Printf("[WARN] %s", message)
	}
}

func Error(message string, err error, details ...map[string]interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Error(message, err, details...)
	} else {
		if err != nil {
			log.Printf("[ERROR] %s: %v", message, err)
		} else {
			log.Printf("[ERROR] %s", message)
		}
	}
}

func Fatal(message string, err error, details ...map[string]interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Fatal(message, err, details...)
	} else {
		if err != nil {
			log.Printf("[FATAL] %s: %v", message, err)
		} else {
			log.Printf("[FATAL] %s", message)
		}
		exitFunc(1)
	}
}

// AddHook adds a hook to the default logger
func AddHook(hook Hook) {
	if DefaultLogger != nil {
		DefaultLogger.AddHook(hook)
	}
}

// Contextual logging convenience functions
func LogUserAction(userID, sessionID, action, message string, details map[string]interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.LogUserAction(userID, sessionID, action, message, details)
	} else {
		log.Printf("[USER_ACTION] %s (%s): %s", userID, action, message)
	}
}

func LogPluginAction(pluginName, action, message string, details map[string]interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.LogPluginAction(pluginName, action, message, details)
	} else {
		log.Printf("[PLUGIN_ACTION] %s (%s): %s", pluginName, action, message)
	}
}

func LogSecurityEvent(event, message string, details map[string]interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.LogSecurityEvent(event, message, details)
	} else {
		log.Printf("[SECURITY] %s: %s", event, message)
	}
}

func LogSystemEvent(event, message string, details map[string]interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.LogSystemEvent(event, message, details)
	} else {
		log.Printf("[SYSTEM] %s: %s", event, message)
	}
}
