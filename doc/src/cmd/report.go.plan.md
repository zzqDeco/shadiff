# report.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: cmd/report.go
- Doc file: doc/src/cmd/report.go.plan.md
- File type: Go source
- Module: shadiff (package cmd)

## 2. Core Responsibility
- Implements the `report` subcommand, which is the fourth and final stage of the shadiff workflow.
- Loads previously computed diff results from storage and generates a formatted report.
- Supports multiple output formats (terminal, JSON, HTML) and can write to stdout or a file.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `--session` / `-s` (required): Session ID or name to generate a report for.
  - `--format` / `-f`: Report format (`terminal`, `json`, or `html`; default `terminal`).
  - `--output` / `-o`: Output file path (defaults to stdout when omitted).
  - Reads diff results from the file store at `~/.shadiff` via `store.LoadResults()`.
- Output results:
  - Generates a formatted report and writes it to the specified output (stdout or file).
  - When writing to a file, prints a confirmation message with the file path.

## 4. Key Implementation Details
- Structs/interfaces: None defined directly; uses `reporter.Reporter` interface from `shadiff/internal/reporter`.
- Exported functions/methods: None (all functions and commands are package-private).
- Unexported functions:
  - `runReport(cmd *cobra.Command, args []string) error` -- Main execution handler for the report command.
- Package-level variables:
  - `reportSession string` -- Session identifier.
  - `reportFormat string` -- Output format.
  - `reportOutput string` -- Output file path.
- Key behaviors:
  - **Result loading**: Calls `store.LoadResults(sessionID)` to retrieve previously computed diff results. Returns an error if no results exist, instructing the user to run `diff` first.
  - **Summary generation**: Uses `diff.FormatDiffSummary()` to compute aggregate statistics and sets the `SessionID` on the summary.
  - **Reporter factory**: Uses `reporter.NewReporter(reportFormat)` to create a format-specific reporter instance. The factory pattern allows easy extension with new formats.
  - **Output routing**: Writes to `os.Stdout` by default. When `--output` is specified, creates the file with `os.Create()` and defers its closure.
  - **Report generation**: Calls `rep.Generate(results, summary, w)` passing the diff results, summary, and output writer.

## 5. Dependencies
- Internal:
  - `shadiff/internal/diff` -- `FormatDiffSummary()` for computing aggregate statistics from diff results.
  - `shadiff/internal/logger` -- File-based logging.
  - `shadiff/internal/reporter` -- `NewReporter()` factory and reporter interface for format-specific output generation.
  - `shadiff/internal/storage` -- `FileStore` for loading persisted diff results.
- External:
  - `fmt`, `os` (standard library) -- Output, file creation, home directory.
  - `github.com/spf13/cobra` -- Command definition.

## 6. Change Impact
- Changes to the `reporter.Reporter` interface or `reporter.NewReporter()` factory affect report generation.
- Changes to the `diff.DiffSummary` struct (returned by `FormatDiffSummary()`) affect the data passed to reporters.
- The `store.LoadResults()` method must return `[]model.DiffResult`; changes to the storage format or diff result model propagate here.
- `resolveSession()` is defined in `cmd/replay.go` and shared; changes there affect this command.
- Adding new report formats requires updating the `--format` flag description and implementing a new reporter in `shadiff/internal/reporter`.

## 7. Maintenance Notes
- The `report` command depends on diff results being previously persisted by the `diff` command (or its engine). If the diff engine does not persist results, this command will always fail with "no diff results". Ensure the diff workflow saves results to storage.
- When adding new report formats, update the flag help string (`"Report format: terminal, json, html"`) to list the new format.
- The output file is created with `os.Create()`, which truncates any existing file. Consider adding a `--force` flag or existence check to prevent accidental overwrites.
- The `summary.SessionID` is set manually after `FormatDiffSummary()` returns. If the summary struct changes to include session context automatically, this assignment can be removed.
- Error handling for file creation is thorough, but the deferred `f.Close()` error is not checked. For critical report outputs, consider handling close errors explicitly.
- The `reporter.NewReporter()` factory pattern makes it straightforward to add new formats (e.g., Markdown, PDF, CSV) without modifying this command file.
