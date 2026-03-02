# json.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/diff/json.go
- Doc file: doc/src/internal/diff/json.go.plan.md
- File type: Go source
- Module: shadiff/internal/diff

## 2. Core Responsibility
- Implements structured JSON comparison for HTTP response bodies, producing field-level differences with path information.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: Two `[]byte` slices representing expected (recorded) and actual (replayed) JSON response bodies.
- Output results: `[]model.Difference` with field-level diff entries including JSON paths, expected/actual values, and severity.

## 4. Key Implementation Details
- Structs/interfaces:
  - `JSONDiffer` -- comparator struct with `IgnoreOrder bool` flag controlling array comparison mode.
- Exported functions/methods:
  - `Compare(expected, actual)` -- entry point that:
    1. Attempts to unmarshal both inputs as JSON.
    2. Falls back to raw byte comparison if expected is not valid JSON.
    3. Reports an error if expected is JSON but actual is not.
    4. Delegates to `compareValues` for recursive structural comparison.
- Unexported functions/methods:
  - `compareValues(path, expected, actual)` -- recursive comparator that:
    - Handles type mismatches with special numeric coercion (int/float interop from `json.Unmarshal`).
    - Dispatches to `compareObjects` for maps, `compareArrays` for slices, or `reflect.DeepEqual` for primitives.
  - `compareObjects(path, expected, actual)` -- compares JSON objects:
    - Collects all keys from both sides, sorts them for deterministic output.
    - Reports missing fields (error severity) and extra fields (warning severity).
    - Recurses into shared keys.
  - `compareArrays(path, expected, actual)` -- compares JSON arrays:
    - Reports length differences.
    - Delegates to ordered (index-by-index) or unordered comparison based on `IgnoreOrder`.
  - `compareArraysUnordered(path, expected, actual)` -- best-match pairing:
    - For each expected element, finds the first unused actual element with zero differences.
    - Reports unmatched expected elements as errors and unmatched actual elements as warnings.
- Key behaviors:
  - All diff paths use dot notation with array indices (e.g., `body.items[0].name`).
  - Numeric type coercion via `toFloat64` handles `json.Unmarshal` producing `float64` for all numbers.
  - The unordered array comparison has O(n*m) worst-case complexity.

## 5. Dependencies
- Internal: `shadiff/internal/model` (for `Difference`, `DiffBody`, `DiffBodyField`, severity constants)
- External:
  - Standard library: `encoding/json`, `fmt`, `reflect`, `sort`.

## 6. Change Impact
- Changes to comparison logic affect all diff results for JSON response bodies.
- The `IgnoreOrder` flag is set by `EngineConfig.IgnoreOrder` in `engine.go`.
- Path formatting (dot notation) must be consistent with the rule-matching patterns in `rules.go`.

## 7. Maintenance Notes
- The `toFloat64` helper is defined in `rules.go` and shared across the package; changes there affect numeric comparison here.
- Unordered array comparison uses a greedy first-match strategy, which may not produce optimal pairings for complex nested objects. A Hungarian-algorithm approach would be more accurate but significantly more complex.
- The `compare` method falls back to string comparison for non-JSON bodies; consider supporting other content types (XML, form-encoded) in the future.
- Large response bodies parsed into `any` via `json.Unmarshal` may consume significant memory; consider streaming comparison for very large payloads.
