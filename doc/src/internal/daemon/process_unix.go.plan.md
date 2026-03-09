# process_unix.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: `internal/daemon/process_unix.go`
- Doc file: `doc/src/internal/daemon/process_unix.go.plan.md`
- File type: Go source
- Module: `shadiff/internal/daemon`
- Build tag: `!windows`

## 2. Core Responsibility
- Implements Unix-specific daemon process operations used by `record --daemon`.
- Handles process detach, liveness checks, graceful stop, and forced termination.

## 3. Inputs & Outputs
- Inputs:
  - `*exec.Cmd` for detach setup
  - process IDs for liveness and signal operations
- Outputs:
  - configured `SysProcAttr`
  - boolean liveness status
  - signal-related errors

## 4. Key Implementation Details
- `Detach(cmd *exec.Cmd)` sets `SysProcAttr.Setsid = true` so the child gets a new session and process group.
- `isProcessAlive(pid int)` uses `Signal(0)` to test process existence without affecting it.
- `SendStop(pid int)` sends `SIGTERM`.
- `ForceKill(pid int)` sends `SIGKILL`.

## 5. Dependencies
- External:
  - Standard library `os`, `os/exec`, `syscall`

## 6. Change Impact
- Signal semantics here affect daemon shutdown behavior on Unix-like systems.
- `record stop` and `record status` depend on these helpers to report and control background recordings correctly.

## 7. Maintenance Notes
- Keep this file behaviorally aligned with `process_windows.go`, even though the implementation is platform-specific.
- If daemon behavior changes, update both platform files together and preserve build-tag separation.
