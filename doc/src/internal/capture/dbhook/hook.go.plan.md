# hook.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/dbhook/hook.go
- Doc file: doc/src/internal/capture/dbhook/hook.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture/dbhook

## 2. Core Responsibility
- Defines the `DBHook` interface that all database protocol proxies must implement, providing a uniform API for starting, flushing, stopping, and receiving captured database side effects.
- Provides a `Group` helper that fans hook side effects into a shared sink and coordinates flush/drain barriers across multiple hooks.
- Provides a factory function (`NewHook`) that instantiates the correct database-specific hook based on configuration.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `Config` struct containing the database type (`mysql`, `postgres`, `mongo`, `redis`), the proxy listen address, and the real database target address.
- Output results:
  - A `DBHook` implementation matching the requested database type, or an `*UnsupportedDBError` if the type is not recognized.
  - A `Group` value that can coordinate a shared flush barrier across multiple started hooks.

## 4. Key Implementation Details
- Structs/interfaces:
  - `DBHook` (interface) -- Contract for all database hook implementations with five methods:
    - `Start(ctx context.Context) error` -- Start the TCP listener and begin proxying/sniffing.
    - `Flush(ctx context.Context) error` -- Block until traffic already observed by the hook has been parsed into `SideEffects()`.
    - `Stop() error` -- Gracefully shut down the proxy.
    - `SideEffects() <-chan model.SideEffect` -- Return a receive-only channel of captured side effects.
    - `Type() string` -- Return the database type identifier.
  - `Group` -- Coordinates multiple hooks plus their side-effect forwarders; exposes `Flush(ctx)` to wait for hook parsing and sink delivery, and `Stop()` for grouped shutdown.
  - `sideEffectForwarder` -- Internal helper that forwards one hook's side-effect channel into a shared sink and supports drain detection through a barrier acknowledgement channel.
  - `Config` -- Configuration struct with fields `DBType`, `ListenAddr`, and `TargetAddr`.
  - `UnsupportedDBError` -- Custom error type for unrecognized database types; implements the `error` interface.
- Exported functions/methods:
  - `NewHook(cfg Config) (DBHook, error)` -- Factory function that switches on `cfg.DBType` and delegates to `NewMySQLHook`, `NewPostgresHook`, `NewMongoHook`, or `NewRedisHook`.
  - `NewGroup(ctx, hooks, sink)` -- Starts grouped side-effect forwarders for already-started hooks.
- Key behaviors:
  - The factory pattern centralizes hook creation, making it easy to add new database types by adding a case to the switch statement.
  - The `DBHook` interface decouples the capture pipeline from database-specific protocol parsing logic.
  - `Group.Flush(ctx)` first calls `Flush(ctx)` on every hook, then waits until forwarded side effects have reached the shared sink channel.
  - Forwarder drain uses a barrier handshake rather than only checking channel length. `WaitDrained(ctx)` sends a barrier through the forwarder goroutine and waits for the acknowledgement, preventing an early return when a side effect has already been consumed from the source channel but has not yet been delivered to the sink.

## 5. Dependencies
- Internal:
  - `shadiff/internal/model` -- `SideEffect` type used in the `DBHook` interface's channel signature.
  - (Indirectly) `mysql.go`, `postgres.go`, `mongo.go`, `redis.go` in the same package -- implementations created by `NewHook`.
- External:
  - `context` -- Used in the `Start` method signature for cancellation support.

## 6. Change Impact
- All `DBHook` implementations (`MySQLHook`, `PostgresHook`, `MongoHook`, `RedisHook`) must conform to this interface; adding or changing methods here requires updating all four.
- Any code that calls `NewHook` or `NewGroup` (typically command startup/wiring code) is affected by changes to `Config`, hook lifecycle, or sink expectations.
- `Recorder.SideEffectChan()` and replay-side sinks consume the channels returned by `SideEffects()` via `Group`.

## 7. Maintenance Notes
- To add support for a new database (e.g., Redis, SQLite), add a new case to the `NewHook` switch and implement the full `DBHook` interface, including `Flush(ctx)`, in a new file.
- The `Config` struct is minimal. If database-specific configuration is needed (e.g., TLS settings, authentication), consider embedding database-specific config sub-structs or using a map of options.
- The `UnsupportedDBError` type enables callers to use `errors.As` for typed error handling if needed.
- Keep `sideEffectForwarder.WaitDrained()` synchronized through the barrier channel. Replacing it with only `len(source)` / `inFlight` checks can reintroduce a race where `Group.Flush()` returns before sink delivery completes.
