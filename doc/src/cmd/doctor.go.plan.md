# doctor.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: cmd/doctor.go
- Doc file: doc/src/cmd/doctor.go.plan.md
- File type: Go source
- Module: shadiff (package cmd)

## 2. Core Responsibility
- Implements the `shadiff doctor` command for read-only environment diagnostics.
- Acts as the Cobra adapter for diagnostics report generation and exit-policy enforcement.
- Provides terminal and JSON output for humans and automation while delegating diagnostic checks to `internal/diagnostics`.

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
- `runDoctor()` builds `diagnostics.Options` from CLI/global state, calls `diagnostics.BuildReport()`, renders terminal or JSON output, and enforces error/strict warning exit policy.
- The command passes version metadata into the diagnostics service so JSON and terminal output keep the same build fields.
- Diagnostic checks remain read-only and live in `internal/diagnostics`.

## 5. Dependencies
- Internal:
  - `shadiff/internal/diagnostics` for report construction and terminal rendering.
- External:
  - Standard library packages for JSON output and string normalization.
  - `github.com/spf13/cobra` for command registration.

## 6. Change Impact
- `cmd/root.go` skips normal runtime initialization for the `doctor` command so diagnostics stay read-only.
- Changes to actual diagnostic checks belong in `internal/diagnostics`.
- User-visible output format changes must keep terminal and JSON behavior aligned.

## 7. Maintenance Notes
- Keep this file as a thin command adapter.
- Preserve existing exit behavior: any error check fails, and `--strict` also fails on warnings.
