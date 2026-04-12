# Diff CI Output Plan

## Goal

Make `shadiff diff` suitable for CI and script integration by supporting stable machine-readable output, file output, and predictable exit semantics.

## Scope

In scope:

- real `diff --output` support for terminal and JSON
- direct diff output to stdout or a file
- configurable diff failure policy through exit codes
- command and black-box tests for JSON output and exit behavior

Out of scope:

- new report formats such as SARIF or JUnit
- changes to `shadiff report` HTML generation beyond reuse of existing JSON structures
- remote reporting or artifact upload automation

## Approach

- Keep `shadiff report` responsible for terminal/JSON/HTML report generation from stored diff results.
- Upgrade `shadiff diff` itself to support immediate terminal or JSON emission after comparison, using the same `{summary, results}` JSON structure already produced by the JSON reporter.
- Introduce a failure policy flag so CI can fail on any diff or only on error-level diffs without parsing free-form terminal text.
- Preserve backward compatibility:
  - default output remains terminal
  - default exit behavior remains success unless the command execution itself fails

## Tasks

- Update CLI surface in `cmd/diff.go`:
  - make `--output terminal|json` effective
  - add `--output-file <path>` for writing diff output to disk
  - add `--fail-on none|diff|error`
- Implement output behavior:
  - terminal output continues to use the existing human-readable rendering
  - JSON output uses the same schema as `report -f json`
  - output can go either to stdout or to the specified file
- Implement exit semantics:
  - `none`: always return exit code `0` when diff execution succeeds
  - `diff`: return non-zero when any non-ignored difference exists
  - `error`: return non-zero only when any non-ignored error-severity difference exists
  - invalid `--fail-on` values fail during argument validation
- Add tests:
  - command tests for terminal output, JSON stdout, and JSON file output
  - exit-code tests for match, ignored-only, warning-only, and error diff cases
  - black-box CLI coverage for CI-friendly JSON usage
- Sync docs after implementation:
  - `doc/src/cmd/diff.go.plan.md`
  - `README.md` and `README_CN.md` for new CLI flags
  - interfaces docs for JSON schema and exit-code behavior

## Verification

- `go test ./cmd ./internal/diff ./internal/reporter`
- Manual checks:
  - `shadiff diff -s <id> --output json` prints valid JSON with `summary` and `results`
  - `shadiff diff -s <id> --output json --output-file out.json` writes the file and keeps stored diff results intact
  - `shadiff diff -s <id> --fail-on diff` returns non-zero on any non-ignored difference
