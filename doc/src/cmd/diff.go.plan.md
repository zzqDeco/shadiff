# diff.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: cmd/diff.go
- Doc file: doc/src/cmd/diff.go.plan.md
- File type: Go source
- Module: shadiff (package cmd)

## 2. Core Responsibility
- Implements the `diff` subcommand, which is the third stage of the shadiff workflow.
- Reads recorded (original) and replayed (new) response data from a session and performs semantic-level comparison.
- Outputs a structured diff report to the terminal showing matches, differences, severities, and an overall match rate.
- Supports configurable diff rules, array order ignoring, and header exclusion.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `--session` / `-s` (required): Session ID or name to diff.
  - `--rules` / `-r`: Path to a diff rules file (JSON/YAML format) for custom comparison logic.
  - `--ignore-order`: Boolean flag to ignore JSON array element ordering during comparison.
  - `--ignore-headers`: List of additional HTTP headers to exclude from comparison.
  - `--output` / `-o`: Output format selection (`terminal` or `json`; default `terminal`).
  - Reads recorded and replayed data from the file store at `~/.shadiff`.
- Output results:
  - For `--output terminal`, prints a formatted diff report to stdout with per-request match/diff status.
  - For `--output json`, prints a JSON object with top-level `summary` and `results` keys using the same schema as the JSON reporter.
  - For terminal output, shows the JSON path, expected vs actual values, severity level, and whether the difference was ignored by a rule.
  - For terminal output, prints a summary line with total records, matches, differences, and match rate percentage.

## 4. Key Implementation Details
- Structs/interfaces: None defined directly; uses `diff.Engine`, `diff.EngineConfig`, `model.DiffResult` from internal packages.
- Exported functions/methods: None (all functions and commands are package-private).
- Unexported functions:
  - `runDiff(cmd *cobra.Command, args []string) error` -- Main execution handler for the diff command.
  - `writeDiffResults(cmd *cobra.Command, results []model.DiffResult, summary model.DiffSummary, w io.Writer) error` -- Routes diff output to terminal or JSON rendering.
  - `currentDiffOutput(cmd *cobra.Command) string` -- Resolves the effective diff output mode.
  - `printDiffResults(w io.Writer, results []model.DiffResult, summary model.DiffSummary)` -- Formats and prints terminal diff results.
- Package-level variables:
  - `diffSession string` -- Session identifier.
  - `diffRulesFile string` -- Path to diff rules file.
  - `diffIgnoreOrder bool` -- Whether to ignore JSON array order.
  - `diffIgnoreHeaders []string` -- Headers to exclude from comparison.
  - `diffOutput string` -- Output format.
- Key behaviors:
  - **Diff engine**: Creates a `diff.EngineConfig` with session ID, order-ignoring, and header-ignoring settings, then delegates to `engine.Run()`.
  - **Output routing**: `writeDiffResults()` selects terminal or JSON output based on `--output` and rejects unsupported formats.
  - **Result formatting**: terminal rendering iterates over results and prints:
    - A checkmark line for matching requests.
    - A cross mark line for differing requests, followed by indented difference details.
    - For ignored differences: shows the rule name that caused the ignore.
    - For actual differences: shows severity (`error`, `warning`, `info`), path, and expected vs actual values.
  - **Summary generation**: Uses `diff.FormatDiffSummary()` to compute aggregate statistics (`TotalCount`, `MatchCount`, `DiffCount`, `MatchRate`) and includes `SessionID` for JSON output.
  - **Severity mapping**: Maps `model.SeverityWarning` and `model.SeverityInfo` to string labels; defaults to `"error"` for unrecognized severities.

## 5. Dependencies
- Internal:
  - `shadiff/internal/diff` -- `Engine`, `EngineConfig`, `FormatDiffSummary()` for comparison logic and summary formatting.
  - `shadiff/internal/logger` -- File-based logging.
  - `shadiff/internal/model` -- `DiffResult`, `SeverityWarning`, `SeverityInfo` types and constants.
  - `shadiff/internal/reporter` -- `JSONReporter` for machine-readable diff output.
  - `shadiff/internal/storage` -- `FileStore` for reading session data.
- External:
  - `fmt`, `io`, `os`, `strings` (standard library) -- Output routing and format normalization.
  - `github.com/spf13/cobra` -- Command definition.

## 6. Change Impact
- Changes to `model.DiffResult` or `model.Difference` fields directly affect `printDiffResults()` formatting.
- Adding new severity levels in `model` requires updating the severity switch statement in `printDiffResults()`.
- The `--rules` flag (`diffRulesFile`) is declared but not passed to `diff.EngineConfig` in the current implementation. Wiring it in requires changes to both this file and the diff engine.
- The `--output` flag is user-visible and must remain aligned with the supported formats in `writeDiffResults()`.
- `resolveSession()` is defined in `cmd/replay.go` and shared across commands.

## 7. Maintenance Notes
- The `diffRulesFile` flag is registered but not yet used in `runDiff`. When implementing custom diff rules, load the file and pass the rules to `diff.EngineConfig`.
- `diff` JSON output intentionally matches the `report -f json` schema so scripts can consume either command consistently.
- The `printDiffResults()` function uses Unicode characters (checkmark, cross, box-drawing) for terminal output. Ensure these render correctly on the target terminals, or provide a `--no-unicode` fallback.
- The severity default of `"error"` for unrecognized severity values is a safe fallback but could mask new severity levels added to the model. Consider logging a warning for unknown severities.
- Diff results are computed on-the-fly but not persisted by this command. The `report` command loads results via `store.LoadResults()`, implying the diff engine or a separate step should save results. Verify the diff engine handles persistence internally.
