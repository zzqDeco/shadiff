# recorder.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/recorder.go
- Doc file: doc/src/internal/capture/recorder.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture

## 2. Core Responsibility
- Provides a unified recording pipeline that receives `model.Record` entries (from the HTTP proxy) and database side effects (from DB hooks), merges them, and persists the combined records to file-based storage.
- Acts as the central coordination point between the HTTP capture layer and the database hook layer.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `*model.Record` objects passed via the `Record()` method (from `Proxy.ServeHTTP`).
  - `model.SideEffect` events received on the `sideEffectCh` channel (from DB hooks).
  - A session ID string and a `*storage.FileStore` provided at construction.
- Output results:
  - Persisted records written to the `FileStore` via `AppendRecord`.
  - Structured log events emitted via `logger.CaptureEvent`.

## 4. Key Implementation Details
- Structs/interfaces:
  - `Recorder` -- Main struct containing the session ID, file store reference, atomic record counter, a buffered side-effect channel (`chan model.SideEffect`, capacity 1000), a mutex-protected `pendingEffects` slice, and a `done` channel for shutdown signaling.
- Exported functions/methods:
  - `NewRecorder(sessionID string, store *storage.FileStore) *Recorder` -- Creates a recorder and starts a background goroutine (`collectSideEffects`) to drain the side-effect channel.
  - `(*Recorder).Record(record *model.Record) error` -- Sets the session ID on the record, attaches any pending side effects under mutex, appends the record to storage, increments the counter, and logs the event.
  - `(*Recorder).SideEffectChan() chan<- model.SideEffect` -- Returns a send-only channel so external components (DB hooks) can submit side effects without direct coupling.
  - `(*Recorder).Count() int64` -- Returns the number of records persisted so far.
  - `(*Recorder).Stop()` -- Signals the background goroutine to stop by closing the `done` channel.
- Key behaviors:
  - Side effects are collected asynchronously in a background goroutine and accumulated in `pendingEffects`. When `Record()` is called, all accumulated side effects are atomically drained and attached to that record.
  - The side-effect channel has a buffer of 1000 to avoid blocking DB hook goroutines.
  - On `Stop()`, the background goroutine drains any remaining items from the channel before returning, ensuring no side effects are lost during shutdown.
  - The association between side effects and records is temporal: side effects are attached to the next `Record()` call after they arrive. This means side effects from a slow DB operation may be attached to a subsequent HTTP request's record.

## 5. Dependencies
- Internal:
  - `shadiff/internal/logger` -- Structured logging.
  - `shadiff/internal/model` -- `Record` and `SideEffect` data types.
  - `shadiff/internal/storage` -- `FileStore` for persisting records to disk.
- External:
  - `fmt` -- Error wrapping.
  - `sync` -- `Mutex` for protecting `pendingEffects`.
  - `sync/atomic` -- Lock-free record counter.

## 6. Change Impact
- `internal/capture/proxy.go` -- Direct caller of `Record()` and constructor; any API changes here require proxy updates.
- `internal/capture/dbhook/*.go` -- DB hooks send side effects via `SideEffectChan()`; channel type or protocol changes affect all hooks.
- `internal/storage/` -- Changes to `FileStore.AppendRecord` signature or behavior directly impact the recorder.
- `internal/model/` -- Changes to `Record` or `SideEffect` fields affect both the merge logic and storage serialization.

## 7. Maintenance Notes
- The temporal side-effect association (attaching pending effects to the next record) is a simplification. If precise request-to-side-effect correlation is required, consider adding a correlation ID or timestamp-based matching.
- The `pendingEffects` slice grows unbounded between `Record()` calls. If the proxy stops recording but side effects keep arriving, memory usage will increase. Consider adding a cap or periodic flush.
- `Stop()` does not wait for the background goroutine to finish (no `sync.WaitGroup`). If cleanup ordering matters, consider adding a wait mechanism.
- The 1000-element channel buffer is a fixed constant. For high-throughput scenarios, this may need to be configurable.
