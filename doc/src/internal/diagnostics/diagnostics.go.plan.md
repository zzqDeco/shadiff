# diagnostics.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/diagnostics/diagnostics.go
- Doc file: doc/src/internal/diagnostics/diagnostics.go.plan.md
- File type: Go source
- Module: shadiff/internal/diagnostics

## 2. Core Responsibility
- Builds the read-only `shadiff doctor` diagnostic report.
- Checks config load/validation, data/log path visibility, supported DB proxy types, Docker, Docker Compose, and optional official E2E port availability.
- Renders the terminal doctor report used by the CLI.

## 3. Inputs & Outputs
- Input sources: `Options` with version metadata, config path, strict/E2E flags, optional command runner, and optional E2E port overrides.
- Output results: `Report` containing version/build metadata, resolved paths, check list, and pass/warn/error/skip summary.

## 4. Key Implementation Details
- `BuildReport()` never creates config files or data directories; missing config is a warning and defaults are used for validation context.
- `CommandRunner` makes Docker and Docker Compose checks testable without depending on the host environment.
- `DefaultE2EAddrs()` returns the official E2E demo listen addresses.
- `RunExternalCommand()` is the production external-command runner with a short timeout.
- `PrintReport()` centralizes terminal rendering so `cmd/doctor.go` stays a thin command adapter.

## 5. Dependencies
- Internal:
  - `shadiff/internal/config` for config defaults and validation.
  - `shadiff/internal/dbtype` for supported DB proxy types.
- External: Go standard library only.

## 6. Change Impact
- New doctor checks should be added here, not in `cmd/doctor.go`.
- Stable check IDs are part of the JSON automation contract and should not be renamed casually.
- Changes to official E2E ports should update `DefaultE2EAddrs()`.

## 7. Maintenance Notes
- Keep diagnostics read-only unless a future command explicitly introduces a separate repair mode.
- Use warnings for optional tools and errors for conditions that prevent the requested Shadiff workflow from working.

