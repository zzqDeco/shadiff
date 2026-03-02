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

// Init 初始化全局日志。日志写入 stderr 和按日轮转的文件。
func Init(dataDir string) error {
	mu.Lock()
	defer mu.Unlock()

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

	multiWriter := io.MultiWriter(os.Stderr, f)

	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	})

	instance = slog.New(handler)
	slog.SetDefault(instance)

	instance.Info("Logger initialized",
		"logPath", logPath,
		"pid", os.Getpid(),
	)

	return nil
}

// Close 刷新并关闭日志文件
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		logFile.Sync()
		logFile.Close()
		logFile = nil
	}
}

// L 返回全局日志实例
func L() *slog.Logger {
	if instance == nil {
		return slog.Default()
	}
	return instance
}

// --- 领域便捷方法 ---

// CaptureEvent 记录采集事件
func CaptureEvent(event string, attrs ...any) {
	args := []any{"event", event}
	args = append(args, attrs...)
	L().Info("[CAPTURE]", args...)
}

// ReplayEvent 记录回放事件
func ReplayEvent(event string, attrs ...any) {
	args := []any{"event", event}
	args = append(args, attrs...)
	L().Info("[REPLAY]", args...)
}

// DiffEvent 记录对拍事件
func DiffEvent(event string, attrs ...any) {
	args := []any{"event", event}
	args = append(args, attrs...)
	L().Info("[DIFF]", args...)
}

// DBHookEvent 记录数据库钩子事件
func DBHookEvent(event string, dbType string, attrs ...any) {
	args := []any{"event", event, "dbType", dbType}
	args = append(args, attrs...)
	L().Info("[DBHOOK]", args...)
}

// SessionEvent 记录会话事件
func SessionEvent(event string, sessionID string, attrs ...any) {
	args := []any{"event", event, "session_id", sessionID}
	args = append(args, attrs...)
	L().Info("[SESSION]", args...)
}

// Error 记录通用错误
func Error(msg string, err error, attrs ...any) {
	args := []any{"error", err.Error()}
	args = append(args, attrs...)
	L().Error(msg, args...)
}

// Debug 记录调试信息
func Debug(msg string, attrs ...any) {
	L().Debug(msg, attrs...)
}

// Info 记录信息
func Info(msg string, attrs ...any) {
	L().Info(msg, attrs...)
}

// Warn 记录警告
func Warn(msg string, attrs ...any) {
	L().Warn(msg, attrs...)
}
