# sessioninspect.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/sessioninspect/sessioninspect.go
- Doc file: doc/src/internal/sessioninspect/sessioninspect.go.plan.md
- File type: Go source
- Module: shadiff/internal/sessioninspect

## 2. Core Responsibility
- Builds `shadiff session inspect` reports from storage data and session artifact paths.
- Counts record/replay/diff artifacts and DB side effects by supported DB type.
- Renders terminal session inspection output used by the CLI.

## 3. Inputs & Outputs
- Input sources: a `Store` implementation, data directory path, and resolved session ID.
- Output results: `Report` with session metadata, artifact file status, record counts, replay counts, diff result counts, side-effect counts, and warnings for missing replay/diff artifacts.

## 4. Key Implementation Details
- `Store` is the narrow storage capability required for inspection: session metadata, records, replay records, and diff results.
- `BuildReport()` treats missing replay records and missing diff results as warnings rather than command failures when the session exists.
- `CountRecordDBSideEffects()` uses `internal/dbtype` to initialize known DB type buckets and counts malformed or non-database side effects as `unknown`.
- `PrintReport()` centralizes terminal rendering so `cmd/session.go` stays focused on Cobra wiring and output format selection.

## 5. Dependencies
- Internal:
  - `shadiff/internal/dbtype` for supported DB type enumeration.
  - `shadiff/internal/model` for sessions, records, side effects, and diff results.
- External: Go standard library only.

## 6. Change Impact
- Changes to side-effect payload types or supported DB names affect side-effect count output here.
- Changes to session artifact filenames must update `BuildReport()` file status construction.
- JSON output shape is exposed by `shadiff session inspect --format json`.

## 7. Maintenance Notes
- Keep inspection read-only.
- Avoid adding command flag behavior in this package; keep CLI concerns in `cmd/session.go`.

