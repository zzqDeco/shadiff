# process_windows.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: `internal/daemon/process_windows.go`
- Doc file: `doc/src/internal/daemon/process_windows.go.plan.md`
- File type: Go source
- Module: `shadiff/internal/daemon`
- Build tag: `windows`

## 2. Core Responsibility
- Implements Windows-specific daemon process operations used by `record --daemon`.
- Handles process group detachment, process liveness checks, graceful stop, and forced kill.

## 3. Inputs & Outputs
- Inputs:
  - `*exec.Cmd` for detach setup
  - process IDs for liveness and signal operations
- Outputs:
  - configured `SysProcAttr`
  - boolean liveness status
  - Windows process control errors

## 4. Key Implementation Details
- `Detach(cmd *exec.Cmd)` sets `CREATE_NEW_PROCESS_GROUP`.
- `isProcessAlive(pid int)`:
  - opens the process with `PROCESS_QUERY_LIMITED_INFORMATION`
  - calls `GetExitCodeProcess`
  - treats `STILL_ACTIVE` as running
- `SendStop(pid int)` signals `os.Interrupt`.
- `ForceKill(pid int)` calls `proc.Kill()`.

## 5. Dependencies
- External:
  - Standard library `os`, `os/exec`, `syscall`, `unsafe`

## 6. Change Impact
- Windows daemon status, graceful stop, and force kill behavior all depend on this file.
- Any changes to Win32 constants or process querying logic affect the correctness of stale PID detection.

## 7. Maintenance Notes
- Keep Windows behavior semantically aligned with Unix behavior where possible.
- Re-test `record status` and `record stop` on Windows if liveness or signal handling logic changes.
- Preserve the build-tag isolation from Unix-specific code.
