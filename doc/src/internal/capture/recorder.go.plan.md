# recorder.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/recorder.go
- Doc file: doc/src/internal/capture/recorder.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture

## 2. Core Responsibility
- Provides a unified recording pipeline that receives `model.Record` entries (from the HTTP proxy) and database side effects (from DB hooks), attributes effects to request scopes, and persists the combined records to file-based storage.
- Acts as the central coordination point between the HTTP capture layer and the database hook layer.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - Request-scope lifecycle events from `Proxy.ServeHTTP` via `BeginRequestScope()` and `FinishRequestScope()`.
  - Optional standalone `*model.Record` objects passed via the `Record()` method.
  - `model.SideEffect` events received on the `sideEffectCh` channel (from DB hooks).
  - A session ID string and a `*storage.FileStore` provided at construction.
- Output results:
  - Persisted records written to the `FileStore` via `AppendRecord`.
  - Structured log events emitted via `logger.CaptureEvent`.

## 4. Key Implementation Details
- Structs/interfaces:
  - `Recorder` -- Main struct containing the session ID, file store reference, atomic record counter, a request-scope ID counter, a buffered side-effect channel (`chan model.SideEffect`, capacity 1000), a flush barrier channel, a mutex-protected `activeScopes` map, and shutdown coordination primitives.
  - `requestScope` -- Internal struct that tracks a request's open/close timestamps and the side effects attributed to that request.
- Exported functions/methods:
  - `NewRecorder(sessionID string, store *storage.FileStore) *Recorder` -- Creates a recorder and starts a background goroutine (`collectSideEffects`) to drain the side-effect channel.
  - `(*Recorder).BeginRequestScope(startedAt int64) int64` -- Opens a request attribution scope and returns its ID.
  - `(*Recorder).FinishRequestScope(scopeID int64, record *model.Record) error` -- Closes a request scope, flushes side effects through the collector, attaches the attributed effects to the record, and persists it.
  - `(*Recorder).Record(record *model.Record) error` -- Persists a standalone record without request-scope attribution.
  - `(*Recorder).SideEffectChan() chan<- model.SideEffect` -- Returns a send-only channel so external components (DB hooks) can submit side effects without direct coupling.
  - `(*Recorder).Count() int64` -- Returns the number of records persisted so far.
  - `(*Recorder).Stop()` -- Signals the background goroutine to stop, drains remaining side effects, and waits for collector shutdown.
- Key behaviors:
  - Side effects are collected asynchronously in a background goroutine and attributed to the best matching request scope by timestamp.
  - A matching scope is the active or closing request scope whose `startedAt` is not later than the effect timestamp and is the most recent such scope.
  - `FinishRequestScope()` marks the scope closed at `record.RecordedAt`, issues a collector flush barrier, and only then removes the scope and persists the record. This keeps in-window effects attachable while preventing post-response effects from leaking in.
  - Effects with no matching scope are treated as orphans: they are logged and discarded instead of being attached to the next unrelated request.
  - The side-effect channel has a buffer of 1000 to avoid blocking DB hook goroutines.
  - On `Stop()`, the background goroutine drains any remaining items from the channel before returning, ensuring no already-sent side effects are lost during shutdown.

## 5. Dependencies
- Internal:
  - `shadiff/internal/logger` -- Structured logging.
  - `shadiff/internal/model` -- `Record` and `SideEffect` data types.
  - `shadiff/internal/storage` -- `FileStore` for persisting records to disk.
- External:
  - `fmt` -- Error wrapping.
  - `sync` -- `Mutex`, `Once`, and `WaitGroup` for request-scope and shutdown coordination.
  - `sync/atomic` -- Lock-free record and scope counters.

## 6. Change Impact
- `internal/capture/proxy.go` -- Direct caller of `BeginRequestScope()` / `FinishRequestScope()` and constructor; any API changes here require proxy updates.
- `internal/capture/dbhook/*.go` -- DB hooks send side effects via `SideEffectChan()`; channel type or protocol changes affect all hooks.
- `internal/storage/` -- Changes to `FileStore.AppendRecord` signature or behavior directly impact the recorder.
- `internal/model/` -- Changes to `Record` or `SideEffect` fields affect both the merge logic and storage serialization.

## 7. Maintenance Notes
- Request-scoped attribution is still timestamp-based, not application-trace-based. If multiple requests share nearly identical timing and side effects, exact attribution still depends on capture timing rather than explicit correlation IDs.
- Closed scopes remain in memory until `FinishRequestScope()` completes its flush-and-persist path. Preserve that ordering if shutdown or persistence logic changes, otherwise in-window effects may regress into orphans.
- The 1000-element channel buffer is a fixed constant. For high-throughput scenarios, this may need to be configurable.
