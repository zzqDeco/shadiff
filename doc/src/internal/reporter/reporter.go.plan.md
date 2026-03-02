# reporter.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/reporter/reporter.go
- Doc file: doc/src/internal/reporter/reporter.go.plan.md
- File type: Go source
- Module: shadiff/internal/reporter

## 2. Core Responsibility
- Defines the `Reporter` interface that all report format implementations must satisfy.
- Provides the `NewReporter` factory function that instantiates the correct reporter based on a format string ("terminal", "json", "html").
- Acts as the single entry point for creating report generators throughout the application.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: A `format` string parameter passed to `NewReporter` (valid values: `"terminal"`, `""`, `"json"`, `"html"`).
- Output results: A concrete `Reporter` implementation (`TerminalReporter`, `JSONReporter`, or `HTMLReporter`), or an error for unsupported formats.

## 4. Key Implementation Details
- Structs/interfaces:
  - `Reporter` (interface) -- defines `Generate(results []model.DiffResult, summary model.DiffSummary, w io.Writer) error`.
- Exported functions/methods:
  - `NewReporter(format string) (Reporter, error)` -- factory function that maps a format string to a concrete reporter. An empty string defaults to `"terminal"`.
- Key behaviors:
  - Uses a switch statement for format dispatch; returns a descriptive error for any unrecognized format.
  - The empty-string case is treated as equivalent to `"terminal"`, making terminal output the default.

## 5. Dependencies
- Internal: `shadiff/internal/model` (for `model.DiffResult` and `model.DiffSummary` types used in the `Reporter` interface signature).
- External:
  - `fmt` (error formatting)
  - `io` (the `io.Writer` type in the interface)

## 6. Change Impact
- Adding a new report format requires: (1) creating a new struct that implements `Reporter`, and (2) adding a new case in the `NewReporter` switch.
- Changing the `Reporter` interface signature affects all three implementations: `TerminalReporter`, `JSONReporter`, and `HTMLReporter`.
- Any caller that invokes `NewReporter` is affected if the accepted format strings change.

## 7. Maintenance Notes
- When adding a new output format, remember to add both the implementation file and the corresponding switch case in this file.
- The default format (empty string mapping to terminal) is intentional; do not remove it without updating all callers that rely on the default.
- Keep the `Reporter` interface minimal to avoid forcing unnecessary methods on simple format implementations.
