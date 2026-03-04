# record_status.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: cmd/record_status.go
- Doc file: doc/src/cmd/record_status.go.plan.md
- File type: Go source
- Module: shadiff (package cmd)

## 2. Core Responsibility
- Implements the `record status` subcommand, which displays the status of daemon recording sessions.
- Supports two modes: listing all active recording sessions (no `-s` flag) or showing detailed status for a specific session (with `-s` flag).
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `--session` / `-s` (optional): Session name or ID to show detailed status for.
  - Session data from the file store at `~/.shadiff`.
  - PID files from session directories for liveness checking.
- Output results:
  - Without `-s`: Tabular output to stdout with columns: ID, NAME, PID, ALIVE, CREATED. Only shows sessions with status `recording`.
  - With `-s`: Detailed key-value output showing Session ID, Name, Status, DaemonMode, PID, Process status (running/dead), Records count, Target URL, Created time, and Uptime.
  - Prints "No active recording sessions." if no recording sessions exist.

## 4. Key Implementation Details
- Structs/interfaces: None defined directly; uses types from internal packages.
- Exported functions/methods: None (all functions are package-private).
- Unexported functions:
  - `runRecordStatus(cmd *cobra.Command, args []string) error` -- Main execution handler; dispatches to `listRecordingSessions()` or `showSessionDetail()` based on whether `-s` is provided.
  - `listRecordingSessions(store *storage.FileStore, dataDir string) error` -- Lists all sessions with status `recording`, showing PID and process liveness for each.
  - `showSessionDetail(store *storage.FileStore, dataDir string, nameOrID string) error` -- Shows detailed information for a specific session including PID, process status, record count, and uptime.
- Package-level variables:
  - `statusSession string` -- Holds the `--session` flag value.
- Key behaviors:
  - **List mode**: Filters sessions by `SessionRecording` status. For each session, reads the PID file and checks process liveness via `daemon.IsRunning()`. Long session names (>25 chars) are truncated with "..." suffix.
  - **Detail mode**: Uses `findSession()` (defined in `record_stop.go`) to resolve the session. Displays PID as "-" if no PID file exists. Shows "running" or "dead (stale pidfile)" for process status. Computes uptime from `CreatedAt` timestamp.
  - **Timestamps**: Stored as Unix milliseconds, formatted as `"2006-01-02 15:04:05"` for display.

## 5. Dependencies
- Internal:
  - `shadiff/internal/daemon` -- `ReadPID`, `IsRunning` for process status checking.
  - `shadiff/internal/model` -- `Session`, `SessionFilter`, `SessionRecording` constant.
  - `shadiff/internal/storage` -- `FileStore` for session persistence.
- External:
  - `fmt`, `os` (standard library) -- Output and home directory resolution.
  - `path/filepath` (standard library) -- Path construction for session directory.
  - `time` (standard library) -- Timestamp formatting and uptime calculation.
  - `github.com/spf13/cobra` -- Command definition.

## 6. Change Impact
- Depends on `findSession()` from `record_stop.go`. If that function moves, this file needs updating.
- Changes to `model.Session` fields (especially `PID`, `DaemonMode`, `CreatedAt`, `RecordCount`) require updates to the detail display.
- The session status filter (`SessionRecording`) determines which sessions appear in list mode. Adding new statuses (e.g., `paused`) would require updating the filter logic.
- Changes to `daemon.ReadPID()` or `daemon.IsRunning()` affect the PID/liveness display.

## 7. Maintenance Notes
- The list mode uses fixed-width `printf` formatting rather than `tabwriter`. If column alignment issues arise with varying data widths, consider switching to `tabwriter` (as used in `session.go`).
- The uptime calculation uses `time.Since()` which gives wall-clock time. If the system clock changes, the uptime may appear incorrect.
- The name truncation (25 chars) is hardcoded. Consider making this responsive to terminal width.
- The command registers as a subcommand of `recordCmd` in `init()`. Go init order within a package is by filename, so `record.go` < `record_status.go` is correct.
