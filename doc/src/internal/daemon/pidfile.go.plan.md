# pidfile.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/daemon/pidfile.go
- Doc file: doc/src/internal/daemon/pidfile.go.plan.md
- File type: Go source
- Module: shadiff (package daemon)

## 2. Core Responsibility
- Provides PID file management for daemon recording sessions.
- Supports writing, reading, removing, and checking liveness of daemon processes via PID files stored in the session directory.
- Platform-independent: all functions operate on the filesystem; process liveness checking delegates to the platform-specific `isProcessAlive()` function defined in `process_unix.go` or `process_windows.go`.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `sessionDir string` -- The session directory path (e.g., `~/.shadiff/sessions/{id}`), passed to all exported functions.
  - `pid int` -- The process ID to write, passed to `WritePID()`.
- Output results:
  - `WritePID()` writes the PID as a decimal integer to `{sessionDir}/pidfile`.
  - `ReadPID()` returns the PID read from the pidfile, or an error if the file is missing or malformed.
  - `RemovePID()` deletes the pidfile; returns `nil` if already absent.
  - `IsRunning()` returns `true` if a PID can be read and the process is alive.
  - `PIDFilePath()` returns the full filesystem path to the pidfile.

## 4. Key Implementation Details
- Structs/interfaces: None defined.
- Exported functions:
  - `WritePID(sessionDir string, pid int) error` -- Writes the PID to `{sessionDir}/pidfile` with permissions `0644`.
  - `ReadPID(sessionDir string) (int, error)` -- Reads and parses the PID from the pidfile. Trims whitespace before parsing.
  - `RemovePID(sessionDir string) error` -- Removes the pidfile. Returns `nil` if the file does not exist (`os.IsNotExist` check).
  - `IsRunning(sessionDir string) bool` -- Reads the PID and delegates to `isProcessAlive(pid)` (platform-specific).
  - `PIDFilePath(sessionDir string) string` -- Returns `filepath.Join(sessionDir, pidFileName)`.
- Package-level constants:
  - `pidFileName = "pidfile"` -- The filename used for PID files within session directories.
- Key behaviors:
  - The PID file format is a plain text file containing a single decimal integer.
  - `RemovePID` is idempotent: removing a non-existent file is not an error.
  - `IsRunning` returns `false` on any error (missing file, parse failure, dead process).

## 5. Dependencies
- Internal:
  - `isProcessAlive(pid int) bool` -- Platform-specific function defined in `process_unix.go` / `process_windows.go` within the same package.
- External:
  - `fmt` (standard library) -- Error wrapping.
  - `os` (standard library) -- File read/write/remove operations.
  - `path/filepath` (standard library) -- Path construction.
  - `strconv` (standard library) -- Integer-to-string and string-to-integer conversion.
  - `strings` (standard library) -- Whitespace trimming.

## 6. Change Impact
- All daemon-related commands (`record` with `--daemon`, `record stop`, `record status`) depend on these functions.
- The pidfile location is derived from `sessionDir` + `pidFileName` constant. Changing the constant or path logic affects all consumers.
- The `IsRunning` function depends on the platform-specific `isProcessAlive` implementation; changes to process checking logic in `process_unix.go` or `process_windows.go` affect behavior here.

## 7. Maintenance Notes
- The PID file uses plain text format (not binary). This keeps it human-readable and debuggable.
- `WritePID` does not use file locking. In the current design this is safe because only the parent process writes the PID file, and only after the child has started. If multiple writers become possible, file locking should be added.
- `ReadPID` trims whitespace to handle potential trailing newlines from manual editing or platform differences.
- Consider adding a `WritePIDWithLock` variant if concurrent daemon starts become a concern.
