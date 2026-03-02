# json.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/reporter/json.go
- Doc file: doc/src/internal/reporter/json.go.plan.md
- File type: Go source
- Module: shadiff/internal/reporter

## 2. Core Responsibility
- Implements the `Reporter` interface for JSON output.
- Serializes diff results and summary statistics into a pretty-printed JSON document.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `results []model.DiffResult` -- the list of per-request comparison results.
  - `summary model.DiffSummary` -- aggregate statistics.
  - `w io.Writer` -- the destination for JSON output.
- Output results: A JSON document written to `w` with two top-level keys: `"summary"` and `"results"`. Returns an error if JSON encoding fails.

## 4. Key Implementation Details
- Structs/interfaces:
  - `JSONReporter` (struct, empty) -- implements `Reporter`.
  - `jsonReport` (unexported struct) -- wrapper that holds `Summary` and `Results` fields for JSON serialization. Uses json struct tags `"summary"` and `"results"`.
- Exported functions/methods:
  - `(*JSONReporter) Generate(results []model.DiffResult, summary model.DiffSummary, w io.Writer) error` -- encodes the report as indented JSON.
- Key behaviors:
  - Uses `json.NewEncoder` with `SetIndent("", "  ")` for human-readable output (2-space indentation).
  - The JSON structure places `summary` before `results` in the output, matching the struct field order.
  - Encoding errors (e.g., from unencodable values in `any`-typed fields like `Expected`/`Actual`) are propagated to the caller.

## 5. Dependencies
- Internal: `shadiff/internal/model` (for `model.DiffResult`, `model.DiffSummary`).
- External:
  - `encoding/json` (JSON encoding)
  - `io` (`io.Writer` interface)

## 6. Change Impact
- Changes to `model.DiffResult` or `model.DiffSummary` struct tags directly affect the JSON output schema, which may break downstream consumers parsing this format.
- The `jsonReport` struct controls the top-level JSON shape; adding fields here changes the output contract.
- This file does not affect other reporters (terminal, HTML).

## 7. Maintenance Notes
- The `any`-typed fields `Expected` and `Actual` in `model.Difference` can hold arbitrary values. Ensure all values stored there are JSON-serializable to avoid runtime encoding errors.
- If streaming or large-report support is needed, the current single-pass `Encode` approach may need to be replaced with incremental writing.
- The unexported `jsonReport` struct is intentionally not exposed; keep it private to allow schema evolution without API breakage.
