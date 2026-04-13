# Diff CI Output Plan

## Goal

Make `shadiff diff` suitable for CI and script integration by supporting stable machine-readable output, file output, and predictable exit semantics.

## Current Iteration

This iteration only fixes the existing `--output` flag so `terminal` and `json` behave as advertised.
The deferred follow-up items are:

- `--output-file <path>`
- `--fail-on none|diff|error`

## Scope

In scope:

- real `diff --output` support for terminal and JSON
- command and black-box tests for terminal and JSON diff output
- docs and plan updates for the now-implemented CLI behavior

Out of scope:

- direct diff output to a dedicated file
- configurable diff failure policy through exit codes
- new report formats such as SARIF or JUnit
- changes to `shadiff report` HTML generation beyond reuse of existing JSON structures
- remote reporting or artifact upload automation

## Approach

- Keep `shadiff report` responsible for terminal/JSON/HTML report generation from stored diff results.
- Upgrade `shadiff diff` itself to support immediate terminal or JSON emission after comparison, using the same `{summary, results}` JSON structure already produced by the JSON reporter.
- Preserve backward compatibility:
  - default output remains terminal
  - default exit behavior remains success unless the command execution itself fails

## Tasks

- Update CLI surface in `cmd/diff.go`:
  - make `--output terminal|json` effective
- Implement output behavior:
  - terminal output continues to use the existing human-readable rendering
  - JSON output uses the same schema as `report -f json`
- Add tests:
  - command tests for terminal output, JSON stdout, and invalid format handling
  - black-box CLI coverage for CI-friendly JSON usage
- Sync docs after implementation:
  - `doc/src/cmd/diff.go.plan.md`
  - `README.md` and `README_CN.md` for new CLI flags
  - interfaces docs for actual `--output` behavior

## Verification

- `go test ./cmd ./internal/diff ./internal/reporter`
- Manual checks:
  - `shadiff diff -s <id> --output json` prints valid JSON with `summary` and `results`
  - `shadiff diff -s <id> --output terminal` preserves the existing human-readable output
  - `shadiff diff -s <id> --output xml` returns a clear validation error
