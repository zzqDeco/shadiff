# terminal.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/reporter/terminal.go
- Doc file: doc/src/internal/reporter/terminal.go.plan.md
- File type: Go source
- Module: shadiff/internal/reporter

## 2. Core Responsibility
- Implements the `Reporter` interface for terminal (CLI) output with ANSI color codes.
- Renders a human-readable diff report including per-request match/diff status, difference details with severity coloring, and an aggregate summary section.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `results []model.DiffResult` -- the list of per-request comparison results.
  - `summary model.DiffSummary` -- aggregate statistics (totals, match rate, error count).
  - `w io.Writer` -- the destination for the formatted output (typically `os.Stdout`).
- Output results: ANSI-colored text written to the provided `io.Writer`. Returns `nil` error (output errors from `fmt.Fprintf` are not checked).

## 4. Key Implementation Details
- Structs/interfaces:
  - `TerminalReporter` (struct, empty) -- implements `Reporter`.
- Exported functions/methods:
  - `(*TerminalReporter) Generate(results []model.DiffResult, summary model.DiffSummary, w io.Writer) error` -- writes the full terminal report.
- Key behaviors:
  - Each result is printed with a green checkmark (match) or red cross (diff).
  - Differences are listed with tree-drawing characters (`├` / `└`) for visual hierarchy.
  - Ignored differences are printed in gray with the matched rule name.
  - Non-ignored differences show expected vs. actual values and a severity badge colored by level: red for error, yellow for warning, cyan for info.
  - If the request path is empty, it defaults to `"/"`.
  - The summary section shows total/matched/diff counts, optional ignored and critical counts, and the match rate as a bold percentage.

## 5. Dependencies
- Internal: `shadiff/internal/model` (for `model.DiffResult`, `model.DiffSummary`, `model.SeverityWarning`, `model.SeverityInfo`, `model.Severity`).
- External:
  - `fmt` (formatted output with ANSI escape codes)
  - `io` (`io.Writer` interface)

## 6. Change Impact
- Changes to ANSI color codes or output format affect the visual appearance for all CLI users.
- Changes to `model.DiffResult`, `model.Difference`, or `model.DiffSummary` fields require corresponding updates here.
- This file does not affect other reporters (JSON, HTML) since each implements `Generate` independently.

## 7. Maintenance Notes
- ANSI escape codes are hardcoded; if cross-platform no-color support is needed, consider using an environment variable check (e.g., `NO_COLOR`) or a library.
- The method currently always returns `nil`. If write errors need to be surfaced, wrap `fmt.Fprintf` calls with error checking.
- The tree-drawing prefix logic (`├` vs `└`) depends on iteration index; keep it in sync if the differences slice structure changes.
