# record_stop.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: cmd/record_stop.go
- Doc file: doc/src/cmd/record_stop.go.plan.md
- File type: Go source
- Module: shadiff (package cmd)

## 2. Core Responsibility
- Implements the `record stop` subcommand, which stops a daemon recording session.
- Resolves the target session by name or ID, reads the PID file, sends a stop signal, waits for graceful exit, and force kills if necessary.
- Contains the shared `findSession()` helper function used by both `record stop` and `record status` commands.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `--session` / `-s` (required): Session name or ID to stop.
  - Session data from the file store at `~/.shadiff`.
  - PID file from `{sessionDir}/pidfile`.
- Output results:
  - Prints progress messages to stdout: signal sending, force kill warnings, completion status.
  - Updates session status to `completed` and clears PID field on successful stop.
  - Cleans up stale PID files when the process is already dead.

## 4. Key Implementation Details
- Structs/interfaces: None defined directly; uses types from internal packages.
- Exported functions/methods: None (all functions are package-private).
- Unexported functions:
  - `runRecordStop(cmd *cobra.Command, args []string) error` -- Main execution handler for the record stop command.
  - `findSession(store *storage.FileStore, nameOrID string) (*model.Session, error)` -- Resolves a session by exact ID match first, then falls back to name-based search. Returns an error if no match or multiple matches are found.
- Package-level variables:
  - `stopSession string` -- Holds the `--session` flag value.
- Key behaviors:
  - **Session resolution**: `findSession()` first tries `store.Get(nameOrID)` for direct ID lookup, then `store.List()` with name filter. If multiple sessions match the name, lists them all and requires the user to specify an exact ID.
  - **Stale PID detection**: If the PID file exists but the process is dead, cleans up the PID file, updates session status to `completed`, and reports the stale state.
  - **Graceful stop**: Sends SIGTERM (Unix) or os.Interrupt (Windows) via `daemon.SendStop()`.
  - **Polling wait**: Polls every 500ms for up to 10 seconds (20 iterations) checking `daemon.IsRunning()`.
  - **Force kill**: If the process does not exit within the timeout, calls `daemon.ForceKill()`, removes the PID file, and updates the session.

## 5. Dependencies
- Internal:
  - `shadiff/internal/daemon` -- `ReadPID`, `IsRunning`, `SendStop`, `ForceKill`, `RemovePID` for process management.
  - `shadiff/internal/model` -- `Session`, `SessionFilter`, `SessionCompleted` constant.
  - `shadiff/internal/storage` -- `FileStore` for session persistence.
- External:
  - `fmt`, `os` (standard library) -- Output and home directory resolution.
  - `path/filepath` (standard library) -- Path construction for session directory.
  - `time` (standard library) -- Sleep for polling interval.
  - `github.com/spf13/cobra` -- Command definition.

## 6. Change Impact
- The `findSession()` helper is used by both `record_stop.go` and `record_status.go`. Changes to session resolution logic affect both commands.
- Changes to `daemon.SendStop()` or `daemon.ForceKill()` signal behavior affect how processes are terminated.
- The 10-second timeout and 500ms polling interval are hardcoded. If configurable timeouts are needed, add a `--timeout` flag.
- Session status update on stop must stay in sync with the status values in `model.SessionStatus`.

## 7. Maintenance Notes
- The `findSession()` function is defined in this file but shared with `record_status.go`. If more commands need session resolution, consider moving it to a shared helper file (e.g., `cmd/helpers.go`).
- The force kill path removes the PID file and updates the session, but does not verify that the process actually died after `ForceKill`. A follow-up check could be added for robustness.
- The polling loop uses `time.Sleep(500ms)` which blocks the goroutine. For very responsive UIs, consider a ticker or context-based approach.
- The command registers as a subcommand of `recordCmd` in `init()`. Ensure `record.go` is loaded first (Go init order within a package is by filename, so `record.go` < `record_stop.go` is correct).
