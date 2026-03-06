package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// LogLevel represents logging severity levels.
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// Context key for correlation IDs.
type ctxKey string

const correlationIDKey ctxKey = "correlation_id"

var (
	logLevelNames = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
		FATAL: "FATAL",
	}

	logLevelFromName = map[string]LogLevel{
		"DEBUG": DEBUG,
		"INFO":  INFO,
		"WARN":  WARN,
		"ERROR": ERROR,
		"FATAL": FATAL,
	}

	currentLevel LogLevel = INFO
	zl           zerolog.Logger
	fileWriter   *os.File
	mu           sync.RWMutex
	initialized  bool
)

func init() {
	initLogger()
}

func initLogger() {
	// Check env for log level
	if envLevel := os.Getenv("OPERATOR_LOG_LEVEL"); envLevel != "" {
		if lvl, ok := logLevelFromName[strings.ToUpper(envLevel)]; ok {
			currentLevel = lvl
		}
	}

	// Choose output format: JSON (production) or console (development)
	var writer io.Writer
	format := strings.ToLower(os.Getenv("OPERATOR_LOG_FORMAT"))
	if format == "json" {
		writer = os.Stderr
	} else {
		// Console writer for human-readable output (default)
		writer = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
			NoColor:    os.Getenv("NO_COLOR") != "",
		}
	}

	zl = zerolog.New(writer).With().Timestamp().Logger()
	zl = zl.Level(toZerologLevel(currentLevel))
	initialized = true
}

func toZerologLevel(level LogLevel) zerolog.Level {
	switch level {
	case DEBUG:
		return zerolog.DebugLevel
	case INFO:
		return zerolog.InfoLevel
	case WARN:
		return zerolog.WarnLevel
	case ERROR:
		return zerolog.ErrorLevel
	case FATAL:
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// SetLevel updates the minimum log level.
func SetLevel(level LogLevel) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
	zl = zl.Level(toZerologLevel(level))
}

// GetLevel returns the current minimum log level.
func GetLevel() LogLevel {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

// EnableFileLogging enables writing structured JSON logs to a file
// in addition to console output.
func EnableFileLogging(filePath string) error {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	if fileWriter != nil {
		fileWriter.Close()
	}
	fileWriter = file

	// Rebuild logger with multi-writer: console + file
	rebuildLogger()
	return nil
}

// DisableFileLogging stops writing logs to file.
func DisableFileLogging() {
	mu.Lock()
	defer mu.Unlock()

	if fileWriter != nil {
		fileWriter.Close()
		fileWriter = nil
		rebuildLogger()
	}
}

// rebuildLogger reconstructs the zerolog logger with current settings.
// Must be called with mu held.
func rebuildLogger() {
	var writer io.Writer
	format := strings.ToLower(os.Getenv("OPERATOR_LOG_FORMAT"))

	if format == "json" {
		writer = os.Stderr
	} else {
		writer = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
			NoColor:    os.Getenv("NO_COLOR") != "",
		}
	}

	if fileWriter != nil {
		// File always gets JSON for machine parsing
		writer = zerolog.MultiLevelWriter(writer, fileWriter)
	}

	zl = zerolog.New(writer).With().Timestamp().Logger()
	zl = zl.Level(toZerologLevel(currentLevel))
}

// WithCorrelationID returns a new context with the given correlation ID.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// CorrelationID extracts the correlation ID from context, or returns "".
func CorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

// getLogger returns a zerolog.Logger, optionally enriched with context fields.
func getLogger() zerolog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return zl
}

// --- Core log function ---

func logMessage(level LogLevel, component string, message string, fields map[string]any) {
	l := getLogger()
	var evt *zerolog.Event

	switch level {
	case DEBUG:
		evt = l.Debug()
	case INFO:
		evt = l.Info()
	case WARN:
		evt = l.Warn()
	case ERROR:
		evt = l.Error()
	case FATAL:
		evt = l.Fatal()
	default:
		evt = l.Info()
	}

	if component != "" {
		evt = evt.Str("component", component)
	}

	for k, v := range fields {
		evt = evt.Interface(k, v)
	}

	evt.Msg(message)
}

// logMessageCtx logs with context (correlation ID propagation).
func logMessageCtx(ctx context.Context, level LogLevel, component string, message string, fields map[string]any) {
	l := getLogger()
	var evt *zerolog.Event

	switch level {
	case DEBUG:
		evt = l.Debug()
	case INFO:
		evt = l.Info()
	case WARN:
		evt = l.Warn()
	case ERROR:
		evt = l.Error()
	case FATAL:
		evt = l.Fatal()
	default:
		evt = l.Info()
	}

	if cid := CorrelationID(ctx); cid != "" {
		evt = evt.Str("correlation_id", cid)
	}

	if component != "" {
		evt = evt.Str("component", component)
	}

	for k, v := range fields {
		evt = evt.Interface(k, v)
	}

	evt.Msg(message)
}

// --- Existing API (preserved for backward compatibility) ---

func Debug(message string)                                          { logMessage(DEBUG, "", message, nil) }
func DebugC(component string, message string)                       { logMessage(DEBUG, component, message, nil) }
func DebugF(message string, fields map[string]any)                  { logMessage(DEBUG, "", message, fields) }
func DebugCF(component string, message string, fields map[string]any) {
	logMessage(DEBUG, component, message, fields)
}

func Info(message string)                                          { logMessage(INFO, "", message, nil) }
func InfoC(component string, message string)                       { logMessage(INFO, component, message, nil) }
func InfoF(message string, fields map[string]any)                  { logMessage(INFO, "", message, fields) }
func InfoCF(component string, message string, fields map[string]any) {
	logMessage(INFO, component, message, fields)
}

func Warn(message string)                                          { logMessage(WARN, "", message, nil) }
func WarnC(component string, message string)                       { logMessage(WARN, component, message, nil) }
func WarnF(message string, fields map[string]any)                  { logMessage(WARN, "", message, fields) }
func WarnCF(component string, message string, fields map[string]any) {
	logMessage(WARN, component, message, fields)
}

func Error(message string)                                          { logMessage(ERROR, "", message, nil) }
func ErrorC(component string, message string)                       { logMessage(ERROR, component, message, nil) }
func ErrorF(message string, fields map[string]any)                  { logMessage(ERROR, "", message, fields) }
func ErrorCF(component string, message string, fields map[string]any) {
	logMessage(ERROR, component, message, fields)
}

func Fatal(message string)                                          { logMessage(FATAL, "", message, nil) }
func FatalC(component string, message string)                       { logMessage(FATAL, component, message, nil) }
func FatalF(message string, fields map[string]any)                  { logMessage(FATAL, "", message, fields) }
func FatalCF(component string, message string, fields map[string]any) {
	logMessage(FATAL, component, message, fields)
}

// --- Context-aware API (new) ---
// These propagate correlation_id from context into log entries.

func DebugCtx(ctx context.Context, message string)                   { logMessageCtx(ctx, DEBUG, "", message, nil) }
func DebugCCtx(ctx context.Context, component string, message string) {
	logMessageCtx(ctx, DEBUG, component, message, nil)
}
func DebugCFCtx(ctx context.Context, component string, message string, fields map[string]any) {
	logMessageCtx(ctx, DEBUG, component, message, fields)
}

func InfoCtx(ctx context.Context, message string)                   { logMessageCtx(ctx, INFO, "", message, nil) }
func InfoCCtx(ctx context.Context, component string, message string) {
	logMessageCtx(ctx, INFO, component, message, nil)
}
func InfoCFCtx(ctx context.Context, component string, message string, fields map[string]any) {
	logMessageCtx(ctx, INFO, component, message, fields)
}

func WarnCtx(ctx context.Context, message string)                   { logMessageCtx(ctx, WARN, "", message, nil) }
func WarnCCtx(ctx context.Context, component string, message string) {
	logMessageCtx(ctx, WARN, component, message, nil)
}
func WarnCFCtx(ctx context.Context, component string, message string, fields map[string]any) {
	logMessageCtx(ctx, WARN, component, message, fields)
}

func ErrorCtx(ctx context.Context, message string)                   { logMessageCtx(ctx, ERROR, "", message, nil) }
func ErrorCCtx(ctx context.Context, component string, message string) {
	logMessageCtx(ctx, ERROR, component, message, nil)
}
func ErrorCFCtx(ctx context.Context, component string, message string, fields map[string]any) {
	logMessageCtx(ctx, ERROR, component, message, fields)
}
