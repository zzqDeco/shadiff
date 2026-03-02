# html.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/reporter/html.go
- Doc file: doc/src/internal/reporter/html.go.plan.md
- File type: Go source
- Module: shadiff/internal/reporter

## 2. Core Responsibility
- Implements the `Reporter` interface for HTML output.
- Renders a self-contained HTML page with styled diff results and summary statistics using Go's `html/template` package.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `results []model.DiffResult` -- the list of per-request comparison results.
  - `summary model.DiffSummary` -- aggregate statistics.
  - `w io.Writer` -- the destination for the HTML document.
- Output results: A complete HTML document written to `w`, including inline CSS. Returns an error if template parsing or execution fails.

## 4. Key Implementation Details
- Structs/interfaces:
  - `HTMLReporter` (struct, empty) -- implements `Reporter`.
- Exported functions/methods:
  - `(*HTMLReporter) Generate(results []model.DiffResult, summary model.DiffSummary, w io.Writer) error` -- parses and executes the HTML template, writing the result to `w`.
- Key behaviors:
  - The HTML template is stored as a package-level `const htmlTemplate` string (not loaded from an external file).
  - Two custom template functions are registered:
    - `severityClass(s model.Severity) string` -- maps severity levels to CSS class names (`"error"`, `"warning"`, `"info"`).
    - `pct(f float64) string` -- converts a 0-1 float to a percentage string (e.g., `0.95` becomes `"95.0"`).
  - The generated page is fully self-contained with inline `<style>` block; no external CSS or JavaScript dependencies.
  - Records with differences get a red left border (`has-diff` class); matching records get a green left border.
  - Ignored differences are rendered with a line-through style and gray text.
  - Severity badges are color-coded: red background for error, amber for warning, blue for info.
  - The template is parsed fresh on every `Generate` call (not cached).

## 5. Dependencies
- Internal: `shadiff/internal/model` (for `model.DiffResult`, `model.DiffSummary`, `model.Severity`, `model.SeverityError`, `model.SeverityWarning`).
- External:
  - `fmt` (error wrapping, percentage formatting)
  - `html/template` (safe HTML template rendering with auto-escaping)
  - `io` (`io.Writer` interface)

## 6. Change Impact
- Modifications to the `htmlTemplate` constant change the visual appearance and structure of all HTML reports.
- Changes to `model.DiffResult`, `model.Difference`, or `model.DiffSummary` fields require corresponding template updates.
- The custom template functions (`severityClass`, `pct`) are scoped to this file and do not affect other reporters.
- This file does not affect other reporters (terminal, JSON).

## 7. Maintenance Notes
- The template is parsed on every call to `Generate`. For high-throughput scenarios, consider parsing once at init time and caching the `*template.Template`.
- The `html/template` package provides automatic HTML escaping, which protects against XSS when diff values contain user-controlled strings.
- The inline CSS approach means all styling lives in this Go file. For significant style changes, consider extracting the template to an embedded file using `go:embed`.
- When adding new severity levels to the model, update both the `severityClass` template function and the corresponding CSS rules in the `<style>` block.
