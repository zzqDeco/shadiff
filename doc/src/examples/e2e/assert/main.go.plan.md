# main.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: examples/e2e/assert/main.go
- Doc file: doc/src/examples/e2e/assert/main.go.plan.md
- File type: Go source
- Module: shadiff/examples/e2e/assert

## 2. Core Responsibility
- Provides the official E2E acceptance helper used by `examples/e2e/run.sh`.
- Parses structured `diff.json`, verifies expected HTTP/SQL/MongoDB/Redis outcomes, and writes or prints summary JSON.

## 3. Inputs & Outputs
- Input flags: `--records`, `--replay-records`, `--diff`, `--report`, `--run-id`, `--session-name`, `--work-dir`, `--config-file`, `--artifacts-dir`, `--assert`, `--summary-file`, `--print-summary`.
- Output results: optional summary JSON with artifact paths, total/diff counts, `httpMatch`, `hasSQLDiff`, `hasMongoDiff`, and `hasRedisDiff`.

## 4. Key Implementation Details
- `buildSummary()` validates artifact files are present and non-empty, then parses `diff.json` into `model.DiffResult` values.
- HTTP match is false when any status/header/body/body-field difference is present.
- SQL, MongoDB, and Redis flags are based on `db_query`, `mongo_op`, and `redis_command` differences.
- `assertSummary()` fails when records/diffs are missing, HTTP differs, or any expected DB side-effect difference is absent.

## 5. Dependencies
- Internal: `shadiff/internal/model` for diff result and difference-kind constants.
- External: Go standard library only.

## 6. Change Impact
- Changes affect `examples/e2e/run.sh --assert`, `--summary`, and `--summary-file`.
- Difference kind changes in the diff model must be reflected here.

## 7. Maintenance Notes
- Keep this helper dependency-light and avoid adding `jq` or other external JSON tools to the official E2E path.

