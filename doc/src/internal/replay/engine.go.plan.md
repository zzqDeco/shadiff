# engine.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/replay/engine.go
- Doc file: doc/src/internal/replay/engine.go.plan.md
- File type: Go source
- Module: shadiff/internal/replay

## 2. Core Responsibility
- Orchestrates the replay workflow: reads recorded HTTP traffic from storage, dispatches it to a worker pool for replay against a target service, and persists the replay results.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: `*storage.FileStore` for reading recorded records; `EngineConfig` struct with session ID, target URL, concurrency, timeout, delay settings, and an optional replay DB side-effect channel.
- Output results: `[]ReplayResult` containing original/replayed record pairs and errors; replay records are also persisted to storage via `AppendReplayRecord`.

## 4. Key Implementation Details
- Structs/interfaces:
  - `Engine` -- main replay orchestrator holding a `FileStore` reference, session ID, `WorkerPool`, inter-request delay, an optional side-effect channel, and pending replay side effects awaiting attribution.
  - `EngineConfig` -- configuration struct with fields: `SessionID`, `TargetURL`, `Concurrency`, `Timeout`, `RetryCount`, `Delay`, `SideEffectCh`.
- Exported functions/methods:
  - `NewEngine(store, cfg)` -- constructs an `Engine` with defaults (30s timeout, concurrency 1 minimum); creates a `TransformConfig` from the target URL and initializes the `WorkerPool`.
  - `Run()` -- main execution method that:
    1. Loads recorded records from storage via `ListRecords`.
    2. Returns an error if no records exist.
    3. Rejects `concurrency > 1` when replay DB side-effect capture is enabled.
    4. Delegates execution either to `WorkerPool.Execute` or to a serial replay path that captures DB side effects by request window.
    5. Iterates results and persists every replay record, including failures, via `AppendReplayRecord`.
    5. Logs start/completion events with success/error counts.
    6. Returns all `ReplayResult` entries.
- Key behaviors:
  - Replay failures are still materialized as replay records with `Error` populated so downstream diff can report the actual failure rather than a missing replay record.
  - When replay DB side-effect capture is enabled, each replay result records request start and finish timestamps, and the engine attaches only the side effects whose timestamps fall within that window.
  - Side effects that arrive outside any replay request window are logged as replay orphans and dropped instead of leaking into later replay records.
  - Errors during individual record persistence are logged but do not abort the replay.
  - Console output is printed directly via `fmt.Printf` for user feedback.

## 5. Dependencies
- Internal:
  - `shadiff/internal/storage` -- `FileStore` for record I/O.
  - `shadiff/internal/logger` -- structured logging for replay events.
- External:
  - Standard library: `fmt`, `time`.

## 6. Change Impact
- Changes to `EngineConfig` fields affect all callers that construct a replay engine (CLI commands, HTTP handlers).
- The `Run` method's return type (`[]ReplayResult`) is consumed by the diff engine indirectly (via persisted replay records).
- Modifying the persistence logic (which records get saved) directly affects downstream diff comparisons.
- Replay DB side-effect attribution relies on DB hook timestamps being comparable to request start/end timestamps in milliseconds.

## 7. Maintenance Notes
- The default timeout (30s) and minimum concurrency (1) are hardcoded; consider extracting these as package-level constants.
- `fmt.Printf` calls for user output should be replaced with a proper output abstraction if the engine is used in non-CLI contexts (e.g., as a library).
- Replay DB side-effect attribution is request-window based, not trace-ID based. If future requirements demand exact cross-request attribution under concurrency, the current serial-only constraint will need a stronger correlation mechanism.
- Error handling during `AppendReplayRecord` is still log-only; failed saves will cause replay information to be missing from downstream diff.
