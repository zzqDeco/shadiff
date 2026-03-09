# rule_loader.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: `internal/diff/rule_loader.go`
- Doc file: `doc/src/internal/diff/rule_loader.go.plan.md`
- File type: Go source
- Module: `shadiff/internal/diff`

## 2. Core Responsibility
- Bridges config-layer diff rules into runtime diff engine rules.
- Loads external rule files from JSON or YAML for the `diff` command.

## 3. Inputs & Outputs
- Inputs:
  - in-memory `[]config.Rule`
  - file path to a `.json`, `.yaml`, or `.yml` rules file
- Outputs:
  - `[]diff.Rule` ready for `NewRuleSet(...)`
  - parse errors wrapped with format-specific context

## 4. Key Implementation Details
- `RulesFromConfig(...)` performs a field-by-field conversion from `config.Rule` to `diff.Rule`, cloning path slices.
- `LoadRulesFile(path string)`:
  - reads the entire file
  - switches on file extension
  - uses YAML parsing for `.yaml` / `.yml`
  - uses JSON parsing for all other extensions
- The loader does not apply rules; it only prepares them for the diff engine.

## 5. Dependencies
- Internal:
  - `shadiff/internal/config`
- External:
  - Standard library `encoding/json`, `fmt`, `os`, `path/filepath`, `strings`
  - `gopkg.in/yaml.v3`

## 6. Change Impact
- Any schema drift between `config.Rule` and `diff.Rule` must be reconciled here.
- Supported rules file formats are defined here; changes affect `cmd/diff.go` behavior and user expectations.
- Error wrapping here directly affects CLI diagnostics for invalid rules files.

## 7. Maintenance Notes
- Keep this file focused on conversion/loading, not rule evaluation.
- If additional rules file formats are ever supported, extend the extension switch and add tests in `internal/diff/rule_loader_test.go`.
- Preserve slice cloning in `RulesFromConfig(...)` to avoid aliasing config-owned data.
