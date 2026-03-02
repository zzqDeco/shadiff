# config.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/config/config.go
- Doc file: doc/src/internal/config/config.go.plan.md
- File type: Go source
- Module: shadiff

## 2. Core Responsibility
- Defines the complete configuration type hierarchy for the shadiff application, covering capture proxy, replay, diff, storage, and logging settings.
- Provides the `DefaultConfig()` factory function that returns sensible defaults for all configuration sections.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: `DefaultConfig()` is called by `Store` (store.go) as the baseline configuration; JSON config files override these defaults.
- Output results: `AppConfig` instances consumed by capture proxy, replay engine, diff engine, storage layer, and logger initialization.

## 4. Key Implementation Details
- Structs/interfaces:
  - `AppConfig` -- Root configuration. Fields: `Capture` (CaptureConfig), `Replay` (ReplayConfig), `Diff` (DiffConfig), `Storage` (StorageConfig), `Log` (LogConfig).
  - `CaptureConfig` -- Fields: `ListenAddr` (default ":18080"), `MaxBodySize` (bytes), `ExcludePaths` (prefix list), `DBProxies` (slice of DBProxyConfig).
  - `DBProxyConfig` -- Fields: `Type` (mysql/postgres/mongo), `ListenAddr`, `TargetAddr`.
  - `ReplayConfig` -- Fields: `Concurrency`, `Timeout` (duration string), `RetryCount`, `DelayMs`.
  - `DiffConfig` -- Fields: `IgnoreHeaders`, `IgnoreOrder` (JSON array order), `MaxDiffs`, `Rules` (slice of Rule), `RulesFile` (external rules file path).
  - `Rule` -- Diff rule definition. Fields: `Name`, `Kind` (ignore/transform/custom), `Paths` (glob-capable JSON paths), `Pattern` (regex, optional), `Matcher` (custom matcher name, optional).
  - `StorageConfig` -- Fields: `DataDir` (default ~/.shadiff), `MaxSessions`.
  - `LogConfig` -- Fields: `Level` (debug/info/warn/error), `LogDir`.
- Exported functions/methods:
  - `DefaultConfig() *AppConfig` -- Returns a new AppConfig with default values: listen on :18080, 10MB max body, concurrency 1, 30s timeout, 1000 max diffs, common headers ignored (Date, X-Request-Id, X-Trace-Id, Server, Content-Length), 100 max sessions, info log level.
- Constants: None.

## 5. Dependencies
- Internal: None.
- External: None (no imports).

## 6. Change Impact
- Adding fields to any config struct requires updating `DefaultConfig()` if a non-zero default is needed.
- Config struct changes affect the JSON config file format; consider backward compatibility for existing user config files.
- `DiffConfig.Rules` and `Rule` struct changes affect the diff rule engine.
- `DBProxyConfig` changes affect database proxy initialization.

## 7. Maintenance Notes
- All JSON tags use camelCase; maintain this convention.
- `ReplayConfig.Timeout` is a string (e.g., "30s") rather than a numeric type; it must be parsed with `time.ParseDuration` at usage sites.
- `DiffConfig.IgnoreHeaders` defaults include common non-deterministic headers; review this list when adding support for new environments.
- `Rule.Paths` supports glob patterns for flexible JSON path matching; document supported glob syntax for users.
- `MaxBodySize` default is 10MB (10 * 1024 * 1024); adjust if targeting environments with larger payloads.
