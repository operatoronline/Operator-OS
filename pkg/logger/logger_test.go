package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// captureJSON sets up a JSON writer and returns a buffer for inspection.
func captureJSON(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	mu.Lock()
	zl = zerolog.New(&buf).With().Timestamp().Logger()
	zl = zl.Level(toZerologLevel(currentLevel))
	mu.Unlock()
	return &buf
}

// restoreLogger restores default logger after capture.
func restoreLogger(t *testing.T) {
	t.Helper()
	mu.Lock()
	rebuildLogger()
	mu.Unlock()
}

func TestLogLevelFiltering(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(WARN)
	buf := captureJSON(t)
	defer restoreLogger(t)

	tests := []struct {
		name      string
		level     LogLevel
		shouldLog bool
	}{
		{"DEBUG message", DEBUG, false},
		{"INFO message", INFO, false},
		{"WARN message", WARN, true},
		{"ERROR message", ERROR, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			switch tt.level {
			case DEBUG:
				Debug(tt.name)
			case INFO:
				Info(tt.name)
			case WARN:
				Warn(tt.name)
			case ERROR:
				Error(tt.name)
			}
			got := buf.Len() > 0
			if got != tt.shouldLog {
				t.Errorf("level=%v shouldLog=%v but got output=%v", tt.level, tt.shouldLog, got)
			}
		})
	}
}

func TestLoggerWithComponent(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(DEBUG)
	buf := captureJSON(t)
	defer restoreLogger(t)

	tests := []struct {
		name      string
		component string
		message   string
		fields    map[string]any
	}{
		{"Simple message", "test", "Hello, world!", nil},
		{"Message with component", "discord", "Discord message", nil},
		{"Message with fields", "telegram", "Telegram message", map[string]any{
			"user_id": "12345",
			"count":   42,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			switch {
			case tt.fields == nil && tt.component != "":
				InfoC(tt.component, tt.message)
			case tt.fields != nil:
				InfoCF(tt.component, tt.message, tt.fields)
			default:
				Info(tt.message)
			}

			var entry map[string]any
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("invalid JSON output: %v", err)
			}

			if entry["message"] != tt.message {
				t.Errorf("message=%v, want %v", entry["message"], tt.message)
			}
			if tt.component != "" {
				if entry["component"] != tt.component {
					t.Errorf("component=%v, want %v", entry["component"], tt.component)
				}
			}
			if tt.fields != nil {
				for k := range tt.fields {
					if _, ok := entry[k]; !ok {
						t.Errorf("missing field %q in output", k)
					}
				}
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  string
	}{
		{"DEBUG level", DEBUG, "DEBUG"},
		{"INFO level", INFO, "INFO"},
		{"WARN level", WARN, "WARN"},
		{"ERROR level", ERROR, "ERROR"},
		{"FATAL level", FATAL, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if logLevelNames[tt.level] != tt.want {
				t.Errorf("logLevelNames[%d] = %s, want %s", tt.level, logLevelNames[tt.level], tt.want)
			}
		})
	}
}

func TestSetGetLevel(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	tests := []LogLevel{DEBUG, INFO, WARN, ERROR, FATAL}

	for _, level := range tests {
		SetLevel(level)
		if GetLevel() != level {
			t.Errorf("SetLevel(%v) -> GetLevel() = %v, want %v", level, GetLevel(), level)
		}
	}
}

func TestLoggerHelperFunctions(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(INFO)
	buf := captureJSON(t)
	defer restoreLogger(t)

	// These should not panic and should produce output (or not, based on level)
	buf.Reset()
	Debug("This should not log")
	if buf.Len() > 0 {
		t.Error("DEBUG should not log at INFO level")
	}

	buf.Reset()
	Info("This should log")
	if buf.Len() == 0 {
		t.Error("INFO should log at INFO level")
	}

	buf.Reset()
	Warn("This should log")
	if buf.Len() == 0 {
		t.Error("WARN should log at INFO level")
	}

	buf.Reset()
	Error("This should log")
	if buf.Len() == 0 {
		t.Error("ERROR should log at INFO level")
	}

	InfoC("test", "Component message")
	InfoF("Fields message", map[string]any{"key": "value"})
	WarnC("test", "Warning with component")
	ErrorF("Error with fields", map[string]any{"error": "test"})

	SetLevel(DEBUG)
	DebugC("test", "Debug with component")
	WarnF("Warning with fields", map[string]any{"key": "value"})
}

func TestCorrelationID(t *testing.T) {
	ctx := context.Background()

	// No correlation ID
	if got := CorrelationID(ctx); got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	// With correlation ID
	ctx = WithCorrelationID(ctx, "req-12345")
	if got := CorrelationID(ctx); got != "req-12345" {
		t.Errorf("expected 'req-12345', got %q", got)
	}
}

func TestContextAwareLogging(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(DEBUG)
	buf := captureJSON(t)
	defer restoreLogger(t)

	ctx := WithCorrelationID(context.Background(), "corr-abc-123")

	buf.Reset()
	InfoCFCtx(ctx, "agent", "Processing request", map[string]any{"user": "test"})

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if entry["correlation_id"] != "corr-abc-123" {
		t.Errorf("correlation_id=%v, want 'corr-abc-123'", entry["correlation_id"])
	}
	if entry["component"] != "agent" {
		t.Errorf("component=%v, want 'agent'", entry["component"])
	}
	if entry["message"] != "Processing request" {
		t.Errorf("message=%v, want 'Processing request'", entry["message"])
	}
	if entry["user"] != "test" {
		t.Errorf("user=%v, want 'test'", entry["user"])
	}
}

func TestContextLoggingWithoutCorrelationID(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(DEBUG)
	buf := captureJSON(t)
	defer restoreLogger(t)

	ctx := context.Background()
	buf.Reset()
	InfoCtx(ctx, "No correlation")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := entry["correlation_id"]; ok {
		t.Error("correlation_id should not be present when not set in context")
	}
}

func TestStructuredJSONOutput(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(DEBUG)
	buf := captureJSON(t)
	defer restoreLogger(t)

	buf.Reset()
	InfoCF("mcp", "Server connected", map[string]any{
		"server":    "test-server",
		"tools":     5,
		"transport": "stdio",
	})

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	// Must have level, message, time
	if entry["level"] != "info" {
		t.Errorf("level=%v, want 'info'", entry["level"])
	}
	if _, ok := entry["time"]; !ok {
		t.Error("missing 'time' field")
	}
	if entry["component"] != "mcp" {
		t.Errorf("component=%v, want 'mcp'", entry["component"])
	}
	if entry["server"] != "test-server" {
		t.Errorf("server=%v, want 'test-server'", entry["server"])
	}
}

func TestEnvLogLevel(t *testing.T) {
	// Verify the logLevelFromName mapping works
	for name, level := range logLevelFromName {
		if logLevelNames[level] != name {
			t.Errorf("logLevelFromName[%q]=%v but logLevelNames[%v]=%q", name, level, level, logLevelNames[level])
		}
	}
}

func TestEnableDisableFileLogging(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	tmpFile := t.TempDir() + "/test.log"

	if err := EnableFileLogging(tmpFile); err != nil {
		t.Fatalf("EnableFileLogging failed: %v", err)
	}

	SetLevel(INFO)
	Info("File logging test message")

	DisableFileLogging()

	// Verify the file was written to
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), "File logging test message") {
		t.Error("log file should contain the test message")
	}
}

func TestAllContextLogFunctions(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(DEBUG)
	buf := captureJSON(t)
	defer restoreLogger(t)

	ctx := WithCorrelationID(context.Background(), "test-ctx")

	// Verify all context-aware functions work without panicking
	funcs := []struct {
		name string
		fn   func()
	}{
		{"DebugCtx", func() { DebugCtx(ctx, "debug msg") }},
		{"DebugCCtx", func() { DebugCCtx(ctx, "comp", "debug comp msg") }},
		{"DebugCFCtx", func() { DebugCFCtx(ctx, "comp", "debug cf msg", map[string]any{"k": "v"}) }},
		{"InfoCtx", func() { InfoCtx(ctx, "info msg") }},
		{"InfoCCtx", func() { InfoCCtx(ctx, "comp", "info comp msg") }},
		{"InfoCFCtx", func() { InfoCFCtx(ctx, "comp", "info cf msg", map[string]any{"k": "v"}) }},
		{"WarnCtx", func() { WarnCtx(ctx, "warn msg") }},
		{"WarnCCtx", func() { WarnCCtx(ctx, "comp", "warn comp msg") }},
		{"WarnCFCtx", func() { WarnCFCtx(ctx, "comp", "warn cf msg", map[string]any{"k": "v"}) }},
		{"ErrorCtx", func() { ErrorCtx(ctx, "error msg") }},
		{"ErrorCCtx", func() { ErrorCCtx(ctx, "comp", "error comp msg") }},
		{"ErrorCFCtx", func() { ErrorCFCtx(ctx, "comp", "error cf msg", map[string]any{"k": "v"}) }},
	}

	for _, f := range funcs {
		t.Run(f.name, func(t *testing.T) {
			buf.Reset()
			f.fn()
			if buf.Len() == 0 {
				t.Errorf("%s produced no output", f.name)
			}
			var entry map[string]any
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("%s: invalid JSON: %v", f.name, err)
			}
			if entry["correlation_id"] != "test-ctx" {
				t.Errorf("%s: correlation_id=%v, want 'test-ctx'", f.name, entry["correlation_id"])
			}
		})
	}
}
