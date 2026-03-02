# replay.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: cmd/replay.go
- Doc file: doc/src/cmd/replay.go.plan.md
- File type: Go source
- Module: shadiff (package cmd)

## 2. Core Responsibility
- Implements the `replay` subcommand, which is the second stage of the shadiff workflow.
- Reads recorded HTTP requests from a session, replays them against a target service, and stores the new responses.
- Supports configurable concurrency and inter-request delay for load control.
- Provides the shared `resolveSession()` helper used by `replay`, `diff`, and `report` commands to look up sessions by ID or name.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `--session` / `-s` (required): Session ID or name to replay.
  - `--target` / `-t` (required): Target service URL to replay requests against (e.g., `http://localhost:9090`).
  - `--concurrency` / `-c`: Number of concurrent replay workers (default 1).
  - `--delay`: Delay between requests as a Go duration string (e.g., `100ms`).
  - `--db-proxy`: Database proxy specifications for capturing DB side effects during replay.
  - Reads recorded request data from the file store at `~/.shadiff`.
- Output results:
  - Stores replay responses in the file store linked to the session.
  - Updates session status to `SessionReplayed` and sets the target endpoint configuration.
  - Prints a replay summary to stdout showing total, succeeded, and failed counts.

## 4. Key Implementation Details
- Structs/interfaces: None defined directly; uses `replay.Engine`, `replay.EngineConfig` from `shadiff/internal/replay`.
- Exported functions/methods: None (all functions and commands are package-private).
- Unexported functions:
  - `runReplay(cmd *cobra.Command, args []string) error` -- Main execution handler for the replay command.
  - `resolveSession(store *storage.FileStore, nameOrID string) (string, error)` -- Resolves a session by ID or name. Tries ID lookup first via `store.Get()`, falls back to name-based search via `store.List()` with a `SessionFilter`. If multiple sessions match by name, uses the latest (first in the list) and prints a warning.
- Package-level variables:
  - `replaySession string` -- Session identifier (ID or name).
  - `replayTarget string` -- Target service URL.
  - `replayConcurrency int` -- Concurrency level.
  - `replayDelay string` -- Inter-request delay.
  - `replayDBProxy []string` -- Database proxy specifications.
- Key behaviors:
  - **Session resolution**: The `resolveSession()` function provides flexible lookup -- accepts either a session UUID or a human-readable name. When multiple name matches exist, it selects the first (latest) and prints a disambiguation message.
  - **Engine configuration**: Creates a `replay.EngineConfig` with session ID, target URL, concurrency, and delay parameters, then delegates execution to `engine.Run()`.
  - **Error counting**: Iterates over results to count entries with non-nil `Error` fields for the summary.
  - **Session update**: After replay, updates the session's status to `SessionReplayed` and records the target URL. Errors during update are silently ignored.

## 5. Dependencies
- Internal:
  - `shadiff/internal/logger` -- File-based logging.
  - `shadiff/internal/model` -- `Session`, `SessionFilter`, `EndpointConfig`, `SessionReplayed` constant.
  - `shadiff/internal/replay` -- `Engine`, `EngineConfig` for replay execution.
  - `shadiff/internal/storage` -- `FileStore` for session and record storage.
- External:
  - `fmt`, `os` (standard library) -- Output and home directory.
  - `time` (standard library) -- Duration parsing.
  - `github.com/spf13/cobra` -- Command definition.

## 6. Change Impact
- `resolveSession()` is called by `cmd/diff.go` and `cmd/report.go` as well. Changes to its signature or behavior affect all three commands.
- Changes to `replay.EngineConfig` fields require corresponding updates here.
- The `--db-proxy` flag is declared but not wired into the replay engine configuration. Implementing it requires changes to both this file and the replay engine.
- Session status transitions (`SessionReplayed`) are defined in `model`; adding new statuses or changing the workflow sequence affects this file.

## 7. Maintenance Notes
- The `--db-proxy` flag is registered but not yet connected to replay logic. When implementing DB replay capture, pass these values to `replay.EngineConfig` or a separate DB proxy component.
- The `store.Update(session)` error after replay is silently ignored. Consider logging the error at minimum.
- The `resolveSession()` function assumes `store.List()` returns sessions ordered by creation time (latest first). Verify this assumption holds in the storage implementation.
- Concurrency default is 1 (sequential replay). For production use with high-traffic sessions, document recommended concurrency settings and any rate-limiting considerations.
- The replay summary counts errors by checking `r.Error != nil`. If partial errors or retries are introduced, this counting logic may need refinement.
