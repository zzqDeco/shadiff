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
- Input sources: `*storage.FileStore` for reading recorded records; `EngineConfig` struct with session ID, target URL, concurrency, timeout, and delay settings.
- Output results: `[]ReplayResult` containing original/replayed record pairs and errors; replay records are also persisted to storage via `AppendReplayRecord`.

## 4. Key Implementation Details
- Structs/interfaces:
  - `Engine` -- main replay orchestrator holding a `FileStore` reference, session ID, `WorkerPool`, and inter-request delay.
  - `EngineConfig` -- configuration struct with fields: `SessionID`, `TargetURL`, `Concurrency`, `Timeout`, `Delay`.
- Exported functions/methods:
  - `NewEngine(store, cfg)` -- constructs an `Engine` with defaults (30s timeout, concurrency 1 minimum); creates a `TransformConfig` from the target URL and initializes the `WorkerPool`.
  - `Run()` -- main execution method that:
    1. Loads recorded records from storage via `ListRecords`.
    2. Returns an error if no records exist.
    3. Delegates execution to `WorkerPool.Execute` with the configured delay.
    4. Iterates results, persisting successful replays via `AppendReplayRecord`.
    5. Logs start/completion events with success/error counts.
    6. Returns all `ReplayResult` entries.
- Key behaviors:
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

## 7. Maintenance Notes
- The default timeout (30s) and minimum concurrency (1) are hardcoded; consider extracting these as package-level constants.
- `fmt.Printf` calls for user output should be replaced with a proper output abstraction if the engine is used in non-CLI contexts (e.g., as a library).
- Error handling during `AppendReplayRecord` is log-only; failed saves will cause missing records in the diff phase, leading to "replay record missing" diff entries.
