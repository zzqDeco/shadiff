# doctor.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: cmd/doctor.go
- Doc file: doc/src/cmd/doctor.go.plan.md
- File type: Go source
- Module: shadiff (package cmd)

## 2. Core Responsibility
- Implements the `shadiff doctor` command for read-only environment diagnostics.
- Reports config validity, data/log directory visibility, supported DB proxy types, Docker availability, Docker Compose availability, and optional official E2E port availability.
- Provides terminal and JSON output for humans and automation.

## 3. Inputs & Outputs
- Input flags:
  - `--format terminal|json` controls report rendering.
  - `--strict` treats warnings as command failures.
  - `--e2e` adds official E2E demo port checks.
  - Global `--config` selects the config file to inspect.
- Output results:
  - Terminal output with a status table and aggregate summary.
  - JSON output with version metadata, resolved paths, check details, and summary counts.

## 4. Key Implementation Details
- `runDoctor()` parses global command state, builds a report, renders it, and enforces exit policy.
- `buildDoctorReport()` performs diagnostics without creating config files or data directories.
- `loadDoctorConfig()` loads config JSON read-only and falls back to defaults when the file is missing.
- Docker checks are warnings because they are required for integration/E2E workflows but not basic CLI usage.
- `--e2e` port checks attempt a temporary `net.Listen` on each official demo address and close it immediately.

## 5. Dependencies
- Internal:
  - `shadiff/internal/config` for defaults and validation.
  - `shadiff/internal/dbtype` for the supported DB proxy type registry.
- External:
  - Standard library packages for JSON, process execution, networking, filesystem inspection, and table output.
  - `github.com/spf13/cobra` for command registration.

## 6. Change Impact
- `cmd/root.go` skips normal runtime initialization for the `doctor` command so diagnostics stay read-only.
- Changes to official E2E ports must update `doctorE2EAddrs`.
- Changes to supported DB proxy types are picked up through `internal/dbtype`.

## 7. Maintenance Notes
- Keep `doctor` diagnostic checks side-effect free by default.
- Add new checks as separate `doctorCheck` entries with stable IDs so JSON consumers can rely on them.
- Warnings should be used for optional tooling; errors should be reserved for conditions that prevent the requested Shadiff workflow from working.
