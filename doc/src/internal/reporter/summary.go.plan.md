# summary.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/reporter/summary.go
- Doc file: doc/src/internal/reporter/summary.go.plan.md
- File type: Go source
- Module: shadiff/internal/reporter

## 2. Core Responsibility
- Groups `model.Difference` values into operator-facing report categories.
- Provides the shared `DifferenceSummary` payload used by terminal, JSON, and HTML reporters.

## 3. Inputs & Outputs
- Input sources: `[]model.DiffResult`.
- Output results: `DifferenceSummary` with HTTP, SQL, MongoDB, Redis, unknown side-effect, and ignored counts.

## 4. Key Implementation Details
- `SummarizeDifferences()` skips ignored differences from category counts and increments `Ignored`.
- HTTP includes status, header, body, and body-field differences.
- SQL includes query differences and SQL query-count differences.
- MongoDB and Redis map to their protocol-specific difference kinds.
- Residual or unrecognized side-effect differences are counted as `UnknownSideEffect`.

## 5. Dependencies
- Internal: `shadiff/internal/model` for difference kinds and result types.

## 6. Change Impact
- Adding a new `model.DifferenceKind` should update this categorizer so reports do not misclassify it.
- JSON reports expose `DifferenceSummary`, so field names are user-visible.

## 7. Maintenance Notes
- Keep the categories compact and operator-facing rather than mirroring every internal difference kind.
