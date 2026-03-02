# worker.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/replay/worker.go
- Doc file: doc/src/internal/replay/worker.go.plan.md
- File type: Go source
- Module: shadiff/internal/replay

## 2. Core Responsibility
- Implements the concurrent HTTP replay worker pool that executes recorded requests against a target service and captures the responses.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: `[]model.Record` (recorded HTTP traffic), `time.Duration` (inter-request delay), `TransformConfig` (request transformation settings).
- Output results: `[]ReplayResult` containing original record, replayed record (with response), and any error encountered.

## 4. Key Implementation Details
- Structs/interfaces:
  - `WorkerPool` -- holds concurrency level, shared `*http.Client`, and `TransformConfig`.
  - `ReplayResult` -- result struct with `Original` (recorded record), `Replayed` (replay record), and `Error`.
- Exported functions/methods:
  - `NewWorkerPool(concurrency, timeout, transform)` -- creates a pool with a shared HTTP client configured with the given timeout.
  - `Execute(records, delay)` -- replays all records, choosing sequential mode when concurrency <= 1 or concurrent fan-out otherwise. Returns results in input order.
- Unexported functions/methods:
  - `replayOne(original)` -- executes a single replay:
    1. Transforms the recorded request via `Transform()`.
    2. Sends the HTTP request and measures duration.
    3. On success: reads response body, builds a `model.Record` with response data.
    4. On failure: builds a partial `model.Record` with the error message.
    5. Logs replay events with method, path, status, and duration.
  - `cloneHTTPHeaders(h)` -- deep-copies `http.Header` to avoid shared slice references.
- Key behaviors:
  - Concurrent mode uses a channel-based job queue with `sync.WaitGroup`; results are written to a pre-allocated slice by index, avoiding race conditions.
  - Each replayed record gets a new 8-char UUID ID.
  - The `SideEffects` field is initialized to an empty slice (not nil) on successful replays.
  - The delay is applied between requests in both sequential and concurrent modes.

## 5. Dependencies
- Internal:
  - `shadiff/internal/model` -- `Record`, `HTTPResponse`, `SideEffect` types.
  - `shadiff/internal/logger` -- structured replay event logging.
- External:
  - `github.com/google/uuid` -- UUID generation for replay record IDs.
  - Standard library: `bytes`, `fmt`, `io`, `net/http`, `sync`, `time`.

## 6. Change Impact
- Changes to `ReplayResult` affect the replay engine (`engine.go`) and any consumers of replay output.
- HTTP client configuration (timeout, transport) changes affect all replay requests.
- Modifying the concurrency model affects throughput and resource usage.

## 7. Maintenance Notes
- Line 142 (`_ = bytes.NewReader(original.Request.Body)`) creates a reader that is immediately discarded; this appears to be a no-op and could be removed.
- The shared `http.Client` is fine for connection pooling but does not support per-request timeout overrides.
- Consider adding retry logic with backoff for transient network errors.
- The `cloneHTTPHeaders` helper performs a full deep copy; this is necessary because `http.Header` values are slices.
