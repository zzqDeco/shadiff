package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	instance *slog.Logger
	logFile  *os.File
	mu       sync.Mutex
)

// Init initializes the global logger. Logs are written to stderr and a daily-rotated file.
// When daemonMode is true, logs are written only to the file (no stderr output).
func Init(dataDir string, daemonMode ...bool) error {
	mu.Lock()
	defer mu.Unlock()

	isDaemon := len(daemonMode) > 0 && daemonMode[0]

	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	fileName := fmt.Sprintf("shadiff-%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(logDir, fileName)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logPath, err)
	}

	if logFile != nil {
		logFile.Close()
	}
	logFile = f

	var writer io.Writer
	if isDaemon {
		writer = f
	} else {
		writer = io.MultiWriter(os.Stderr, f)
	}

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	})

	instance = slog.New(handler)
	slog.SetDefault(instance)

	instance.Info("Logger initialized",
		"logPath", logPath,
		"pid", os.Getpid(),
		"daemon", isDaemon,
	)

	return nil
}

// Close flushes and closes the log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		logFile.Sync()
		logFile.Close()
		logFile = nil
	}
}

// L returns the global logger instance.
func L() *slog.Logger {
	if instance == nil {
		return slog.Default()
	}
	return instance
}

// --- Domain convenience methods ---

// CaptureEvent logs a capture event.
func CaptureEvent(event string, attrs ...any) {
	args := []any{"event", event}
	args = append(args, attrs...)
	L().Info("[CAPTURE]", args...)
}

// ReplayEvent logs a replay event.
func ReplayEvent(event string, attrs ...any) {
	args := []any{"event", event}
	args = append(args, attrs...)
	L().Info("[REPLAY]", args...)
}

// DiffEvent logs a diff event.
func DiffEvent(event string, attrs ...any) {
	args := []any{"event", event}
	args = append(args, attrs...)
	L().Info("[DIFF]", args...)
}

// DBHookEvent logs a database hook event.
func DBHookEvent(event string, dbType string, attrs ...any) {
	args := []any{"event", event, "dbType", dbType}
	args = append(args, attrs...)
	L().Info("[DBHOOK]", args...)
}

// SessionEvent logs a session event.
func SessionEvent(event string, sessionID string, attrs ...any) {
	args := []any{"event", event, "session_id", sessionID}
	args = append(args, attrs...)
	L().Info("[SESSION]", args...)
}

// Error logs a general error.
func Error(msg string, err error, attrs ...any) {
	args := []any{"error", err.Error()}
	args = append(args, attrs...)
	L().Error(msg, args...)
}

// Debug logs a debug message.
func Debug(msg string, attrs ...any) {
	L().Debug(msg, attrs...)
}

// Info logs an informational message.
func Info(msg string, attrs ...any) {
	L().Info(msg, attrs...)
}

// Warn logs a warning message.
func Warn(msg string, attrs ...any) {
	L().Warn(msg, attrs...)
}
