# rules.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/diff/rules.go
- Doc file: doc/src/internal/diff/rules.go.plan.md
- File type: Go source
- Module: shadiff/internal/diff

## 2. Core Responsibility
- Implements the rule engine for filtering and classifying diff results: path-based ignore rules, custom matchers (timestamp, UUID, numeric tolerance), default rule/header configurations, and diff summary formatting.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: `[]Rule` definitions (from configuration or defaults), `[]model.Difference` produced by the diff comparison phase, `[]model.DiffResult` for summary formatting.
- Output results: Modified `[]model.Difference` with `Ignored` and `Rule` fields set; `model.DiffSummary` with aggregate statistics.

## 4. Key Implementation Details
- Structs/interfaces:
  - `Matcher` -- interface with `Name() string` and `Match(path, expected, actual) (bool, error)` for custom comparison logic.
  - `Rule` -- struct with `Name`, `Kind` ("ignore" or "custom"), `Paths` (wildcard patterns), `Pattern` (value regex, optional), `Matcher` (matcher name, optional). Supports JSON and YAML tags.
  - `RuleSet` -- collection of `Rule` entries, registered `Matcher` instances (by name), and pre-compiled path regex patterns.
  - `TimestampMatcher` -- matches values that look like ISO-8601 timestamps on both sides.
  - `UUIDMatcher` -- matches values that look like UUIDs on both sides.
  - `NumericToleranceMatcher` -- matches numeric values within a configurable `Tolerance`.
- Exported functions/methods:
  - `NewRuleSet(rules, matchers...)` -- creates a `RuleSet`, registers matchers by name, and pre-compiles path wildcard patterns to regexps.
  - `RuleSet.Apply(diffs)` -- iterates all differences, checking each against all rules. For "ignore" rules, marks the diff as ignored. For "custom" rules, invokes the named matcher and marks ignored if it returns true.
  - `DefaultRules()` -- returns a pre-configured `RuleSet` with timestamp ignore rules and request-ID header ignore rules, plus all three built-in matchers.
  - `DefaultIgnoreHeaders()` -- returns `[]string` of headers to skip during comparison: `Date`, `X-Request-Id`, `X-Trace-Id`, `Server`, `Content-Length`.
  - `FormatDiffSummary(results)` -- computes aggregate statistics: total, match, diff, ignore, and error counts plus match rate.
  - `FormatPath(prefix, parts...)` -- builds dot-separated diff paths.
- Unexported functions/methods:
  - `matchesPath(rule, diffPath)` -- checks if a diff path matches any of a rule's compiled path patterns.
  - `pathToRegexp(pattern)` -- converts wildcard path patterns to regex: `*` matches single level, `**` matches multiple levels, `[*]` matches array indices.
  - `looksLikeTimestamp(s)` -- tests string against `^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}` pattern.
  - `looksLikeUUID(s)` -- tests string against standard UUID v4 regex pattern.
  - `toFloat64(v)` -- converts `float64`, `float32`, `int`, `int64`, and `json.Number` to `float64`. Shared with `json.go`.
- Key behaviors:
  - Path patterns support three wildcards: `*` (single path segment), `**` (any depth), `[*]` (any array index).
  - Rules are applied in order; the first matching rule wins (sets `Ignored` and `Rule` name).
  - The `Apply` method mutates the input slice in place.

## 5. Dependencies
- Internal: `shadiff/internal/model` (for `Difference`, `DiffResult`, `DiffSummary`, severity constants)
- External:
  - Standard library: `encoding/json`, `fmt`, `regexp`, `strings`.

## 6. Change Impact
- Adding or modifying built-in matchers affects which differences are automatically classified as ignorable across all diff sessions.
- Changes to `pathToRegexp` affect all path-based rule matching.
- `DefaultIgnoreHeaders` is consumed by `engine.go`'s `NewEngine`; changes propagate to all diff runs.
- `FormatDiffSummary` is consumed by CLI/HTTP output formatting.
- `toFloat64` is used by both `NumericToleranceMatcher` here and `compareValues` in `json.go`.
- `FormatPath` is used by `compareObjects` in `json.go` for building diff paths.

## 7. Maintenance Notes
- The `Pattern` field on `Rule` (value regex) is declared but not currently used in `Apply`; implement value-pattern matching if needed.
- Consider adding rule priority or explicit ordering guarantees for overlapping path patterns.
- The `toFloat64` function does not handle `uint`, `int32`, or other numeric types; extend if the model evolves.
- Pre-compiled regexps are cached by the original path string; duplicate path strings across rules will share the same compiled pattern.
- `FormatDiffSummary` counts ignored differences and errors globally across all results, not per-result; this is intentional for top-level reporting.
