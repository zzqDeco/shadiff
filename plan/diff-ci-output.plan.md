# Diff CI Output Plan

## Goal

Make `shadiff diff` suitable for CI and script integration by supporting stable machine-readable output, file output, and predictable exit semantics.

## Current Iteration

This iteration completes the CI-facing diff output surface:

- `--output terminal|json` controls immediate diff rendering
- `--output-file <path>` writes the selected output format to a file
- `--fail-on none|diff|error` controls non-zero exit behavior for scripts and CI

## Scope

In scope:

- real `diff --output` support for terminal and JSON
- direct diff output to a dedicated file
- configurable diff failure policy through exit codes
- command and black-box tests for terminal, JSON, output-file, and fail-on behavior
- docs and plan updates for the now-implemented CLI behavior

Out of scope:

- new report formats such as SARIF or JUnit
- changes to `shadiff report` HTML generation beyond reuse of existing JSON structures
- remote reporting or artifact upload automation

## Approach

- Keep `shadiff report` responsible for terminal/JSON/HTML report generation from stored diff results.
- Upgrade `shadiff diff` itself to support immediate terminal or JSON emission after comparison, using the same `{summary, results}` JSON structure already produced by the JSON reporter.
- Add optional file output using the same renderer selected by `--output`.
- Preserve backward compatibility:
  - default output remains terminal
  - default exit behavior remains success unless the command execution itself fails
- Add explicit CI failure policies:
  - `none`: never fail due to semantic differences
  - `diff`: fail when any record has unignored differences
  - `error`: fail only when unignored error-severity differences exist

## Tasks

- Update CLI surface in `cmd/diff.go`:
  - make `--output terminal|json` effective
  - add `--output-file <path>`
  - add `--fail-on none|diff|error`
- Implement output behavior:
  - terminal output continues to use the existing human-readable rendering
  - JSON output uses the same schema as `report -f json`
  - output-file writes the selected renderer output and prints a confirmation line
- Implement failure behavior:
  - default `none` preserves compatibility
  - `diff` returns a command error when `DiffSummary.DiffCount > 0`
  - `error` returns a command error when `DiffSummary.ErrorCount > 0`
- Add tests:
  - command tests for terminal output, JSON stdout, and invalid format handling
  - command tests for output files, invalid fail-on, fail-on diff, and fail-on error behavior
  - black-box CLI coverage for CI-friendly JSON, output-file, and fail-on usage
- Sync docs after implementation:
  - `doc/src/cmd/diff.go.plan.md`
  - `README.md` and `README_CN.md` for new CLI flags
  - interfaces docs for actual `--output`, `--output-file`, and `--fail-on` behavior

## Verification

- `go test ./cmd ./internal/diff ./internal/reporter`
- `go test ./...`
- Manual checks:
  - `shadiff diff -s <id> --output json` prints valid JSON with `summary` and `results`
  - `shadiff diff -s <id> --output json --output-file diff.json` writes valid JSON and prints a confirmation
  - `shadiff diff -s <id> --output terminal` preserves the existing human-readable output
  - `shadiff diff -s <id> --output xml` returns a clear validation error
  - `shadiff diff -s <id> --fail-on diff` exits non-zero when unignored diffs exist
  - `shadiff diff -s <id> --fail-on error` exits non-zero only for unignored error-severity differences
