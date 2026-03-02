# hook.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/dbhook/hook.go
- Doc file: doc/src/internal/capture/dbhook/hook.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture/dbhook

## 2. Core Responsibility
- Defines the `DBHook` interface that all database protocol proxies must implement, providing a uniform API for starting, stopping, and receiving captured database side effects.
- Provides a factory function (`NewHook`) that instantiates the correct database-specific hook based on configuration.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `Config` struct containing the database type (`mysql`, `postgres`, `mongo`), the proxy listen address, and the real database target address.
- Output results:
  - A `DBHook` implementation matching the requested database type, or an `*UnsupportedDBError` if the type is not recognized.

## 4. Key Implementation Details
- Structs/interfaces:
  - `DBHook` (interface) -- Contract for all database hook implementations with four methods:
    - `Start(ctx context.Context) error` -- Start the TCP listener and begin proxying/sniffing.
    - `Stop() error` -- Gracefully shut down the proxy.
    - `SideEffects() <-chan model.SideEffect` -- Return a receive-only channel of captured side effects.
    - `Type() string` -- Return the database type identifier.
  - `Config` -- Configuration struct with fields `DBType`, `ListenAddr`, and `TargetAddr`.
  - `UnsupportedDBError` -- Custom error type for unrecognized database types; implements the `error` interface.
- Exported functions/methods:
  - `NewHook(cfg Config) (DBHook, error)` -- Factory function that switches on `cfg.DBType` and delegates to `NewMySQLHook`, `NewPostgresHook`, or `NewMongoHook`.
- Key behaviors:
  - The factory pattern centralizes hook creation, making it easy to add new database types by adding a case to the switch statement.
  - The `DBHook` interface decouples the capture pipeline from database-specific protocol parsing logic.

## 5. Dependencies
- Internal:
  - `shadiff/internal/model` -- `SideEffect` type used in the `DBHook` interface's channel signature.
  - (Indirectly) `mysql.go`, `postgres.go`, `mongo.go` in the same package -- implementations created by `NewHook`.
- External:
  - `context` -- Used in the `Start` method signature for cancellation support.

## 6. Change Impact
- All `DBHook` implementations (`MySQLHook`, `PostgresHook`, `MongoHook`) must conform to this interface; adding or changing methods here requires updating all three.
- Any code that calls `NewHook` (typically the application startup/wiring code) is affected by changes to `Config` or the factory function.
- `Recorder.SideEffectChan()` in `internal/capture/recorder.go` consumes the channel returned by `SideEffects()`.

## 7. Maintenance Notes
- To add support for a new database (e.g., Redis, SQLite), add a new case to the `NewHook` switch and implement the `DBHook` interface in a new file.
- The `Config` struct is minimal. If database-specific configuration is needed (e.g., TLS settings, authentication), consider embedding database-specific config sub-structs or using a map of options.
- The `UnsupportedDBError` type enables callers to use `errors.As` for typed error handling if needed.
