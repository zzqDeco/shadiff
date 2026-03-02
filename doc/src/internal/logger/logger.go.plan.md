# logger.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/logger/logger.go
- Doc file: doc/src/internal/logger/logger.go.plan.md
- File type: Go source
- Module: shadiff

## 2. Core Responsibility
- Provides a global structured logger based on `log/slog`, with output to both stderr and a daily-rotated log file.
- Offers domain-specific convenience functions for logging capture, replay, diff, database hook, and session events with consistent prefixes and attributes.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: `Init()` receives a data directory path to determine where log files are written; convenience functions receive event names, messages, and structured attributes.
- Output results: Log entries written to stderr (for real-time visibility) and to a daily log file at `{dataDir}/logs/shadiff-YYYY-MM-DD.log`.

## 4. Key Implementation Details
- Structs/interfaces: None (uses package-level singleton pattern).
- Exported functions/methods:
  - `Init(dataDir string) error` -- Initializes the global logger. Creates the log directory, opens a daily log file (append mode), sets up a `slog.TextHandler` with `MultiWriter` (stderr + file), and sets the instance as the default logger. Log level is hardcoded to `slog.LevelDebug`.
  - `Close()` -- Syncs and closes the log file. Safe to call when no file is open.
  - `L() *slog.Logger` -- Returns the global logger instance; falls back to `slog.Default()` if `Init()` has not been called.
  - `CaptureEvent(event string, attrs ...any)` -- Logs with "[CAPTURE]" prefix.
  - `ReplayEvent(event string, attrs ...any)` -- Logs with "[REPLAY]" prefix.
  - `DiffEvent(event string, attrs ...any)` -- Logs with "[DIFF]" prefix.
  - `DBHookEvent(event string, dbType string, attrs ...any)` -- Logs with "[DBHOOK]" prefix and `dbType` attribute.
  - `SessionEvent(event string, sessionID string, attrs ...any)` -- Logs with "[SESSION]" prefix and `session_id` attribute.
  - `Error(msg string, err error, attrs ...any)` -- Logs at error level with the error string extracted.
  - `Debug(msg string, attrs ...any)` -- Logs at debug level.
  - `Info(msg string, attrs ...any)` -- Logs at info level.
  - `Warn(msg string, attrs ...any)` -- Logs at warning level.
- Constants: None.
- Package-level variables: `instance` (*slog.Logger), `logFile` (*os.File), `mu` (sync.Mutex) -- all unexported, guarding the singleton.

## 5. Dependencies
- Internal: None.
- External:
  - `fmt` -- Error message formatting.
  - `io` -- `MultiWriter` for dual output.
  - `log/slog` -- Structured logging (Go 1.21+ stdlib).
  - `os` -- File operations, stderr, PID retrieval.
  - `path/filepath` -- Log directory path construction.
  - `sync` -- Mutex for thread-safe initialization and closure.
  - `time` -- Daily log file naming.

## 6. Change Impact
- All packages that call `logger.L()`, `logger.Info()`, `logger.Error()`, or any domain event function depend on this initialization.
- Changing the log format (e.g., from `TextHandler` to `JSONHandler`) affects log parsing tools and monitoring integrations.
- Changing the file naming pattern affects log rotation and cleanup scripts.
- The hardcoded `LevelDebug` in `Init()` means all log levels are emitted; to respect `LogConfig.Level`, a level-mapping step must be added.

## 7. Maintenance Notes
- The `Init()` function sets the log level to `slog.LevelDebug` regardless of the `LogConfig.Level` setting in the config. This should be connected to `config.LogConfig.Level` for production use.
- `Init()` closes any previously opened log file before opening a new one, making it safe to call multiple times (e.g., on config reload).
- `Error()` calls `err.Error()` directly; passing a nil error will cause a panic. Callers must nil-check before calling.
- Daily rotation is based on file naming (`shadiff-YYYY-MM-DD.log`), not file size. Old log files are not automatically cleaned up; a separate cleanup mechanism may be needed.
- The `AddSource: false` setting in the handler options suppresses source file/line information from log entries. Set to `true` during development for easier debugging.
- Domain convenience functions (`CaptureEvent`, `ReplayEvent`, etc.) use `slog.Info` level; consider parameterizing the level if finer control is needed.
