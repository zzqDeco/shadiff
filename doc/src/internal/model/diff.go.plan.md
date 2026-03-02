# diff.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/model/diff.go
- Doc file: doc/src/internal/model/diff.go.plan.md
- File type: Go source
- Module: shadiff

## 2. Core Responsibility
- Defines the data models for diff/comparison results, including per-record diff results, individual difference entries, and aggregate summary statistics.
- Provides the vocabulary of difference kinds and severity levels used throughout the diff engine.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: Populated by the diff engine after comparing recorded vs. replayed records.
- Output results: Consumed by report generators, CLI output formatters, and API responses to present comparison results to users.

## 4. Key Implementation Details
- Structs/interfaces:
  - `DiffResult` -- Per-record comparison result. Fields: `RecordID`, `Sequence`, `Request` (HTTPRequest, for context), `Match` (bool), `Differences` (slice of Difference).
  - `Difference` -- A single difference entry. Fields: `Kind` (DifferenceKind), `Path` (e.g., "body.data.items[0].name"), `Expected` (recorded value), `Actual` (replayed value), `Message` (human-readable description), `Severity`, `Ignored` (bool, whether suppressed by a rule), `Rule` (matched rule name).
  - `DiffSummary` -- Aggregate statistics. Fields: `SessionID`, `TotalCount`, `MatchCount`, `DiffCount`, `ErrorCount`, `IgnoreCount`, `MatchRate` (float64, 0-1).
- Exported functions/methods: None. This file is purely type definitions.
- Constants:
  - `DifferenceKind`: `DiffStatusCode` ("status_code"), `DiffHeader` ("header"), `DiffBody` ("body"), `DiffBodyField` ("body_field"), `DiffDBQuery` ("db_query"), `DiffDBQueryCount` ("db_query_count"), `DiffMongoOp` ("mongo_op"), `DiffExternalCall` ("external_call").
  - `Severity`: `SeverityError` ("error"), `SeverityWarning` ("warning"), `SeverityInfo` ("info").

## 5. Dependencies
- Internal: References `HTTPRequest` from `request.go` (same package) in `DiffResult.Request`.
- External: None (no imports).

## 6. Change Impact
- Adding new `DifferenceKind` values requires corresponding comparison logic in the diff engine.
- Adding new `Severity` values requires updates to filtering, reporting, and summary counting logic.
- `DiffSummary` field changes affect summary reporting endpoints and CLI output.
- `Difference.Path` format is used for rule matching in `DiffConfig.Rules`; changes to path conventions require rule engine updates.

## 7. Maintenance Notes
- `Expected` and `Actual` in `Difference` use `any` type to accommodate different value types (strings, numbers, objects) from JSON comparison.
- `Ignored` and `Rule` fields work together: when a difference is suppressed by a configured rule, `Ignored` is set to `true` and `Rule` contains the matching rule name. Ignored differences are still recorded but excluded from error counts.
- `MatchRate` in `DiffSummary` is a float64 between 0 and 1; multiply by 100 for percentage display.
- The `DiffResult.Request` field is included for context so that diff reports can show which API endpoint each comparison belongs to without requiring a separate record lookup.
