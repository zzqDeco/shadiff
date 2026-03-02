# engine.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/diff/engine.go
- Doc file: doc/src/internal/diff/engine.go.plan.md
- File type: Go source
- Module: shadiff/internal/diff

## 2. Core Responsibility
- Orchestrates the behavioral diff comparison between recorded and replayed HTTP traffic, comparing status codes, headers, response bodies, and side effects.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: `*storage.FileStore` for loading recorded and replay records; `EngineConfig` with session ID, custom rules, ignore-order flag, and headers to ignore.
- Output results: `[]model.DiffResult` containing per-record comparison results with match status and detailed differences; results are also persisted to storage.

## 4. Key Implementation Details
- Structs/interfaces:
  - `Engine` -- diff orchestrator holding `FileStore`, session ID, `RuleSet`, `JSONDiffer`, and `ignoreHeaders` map.
  - `EngineConfig` -- configuration with `SessionID`, `Rules []Rule`, `IgnoreOrder bool`, `IgnoreHeaders []string`.
- Exported functions/methods:
  - `NewEngine(store, cfg)` -- constructs an `Engine`:
    1. Merges `DefaultIgnoreHeaders()` with user-specified headers into a lookup map.
    2. Creates a `RuleSet` from user rules plus built-in matchers (`TimestampMatcher`, `UUIDMatcher`, `NumericToleranceMatcher` with 0.001 tolerance).
    3. Initializes a `JSONDiffer` with the `IgnoreOrder` setting.
  - `Run()` -- main execution method that:
    1. Loads original records via `ListRecords` and replay records via `ListReplayRecords`.
    2. Builds a replay lookup map indexed by sequence number.
    3. Compares each original record against its replay counterpart (or emits a "replay record missing" difference).
    4. Saves results to storage and logs events.
- Unexported functions/methods:
  - `compareRecords(original, replay)` -- performs four-phase comparison:
    1. Status code comparison.
    2. Header comparison (respecting `ignoreHeaders`).
    3. JSON body comparison via `JSONDiffer.Compare`.
    4. Side effect count comparison.
    5. Applies `RuleSet` to mark ignorable differences.
    6. Determines overall match (true only if all non-ignored differences are absent).
  - `compareHeaders(expected, actual)` -- iterates expected headers, skipping ignored ones, and reports missing or differing headers as warnings.
- Key behaviors:
  - Records are matched by sequence number, not by ID or request content.
  - A result is marked as `Match: true` only if every difference is `Ignored`.
  - Header differences are reported as warnings; status code and body differences are errors.

## 5. Dependencies
- Internal:
  - `shadiff/internal/model` -- `Record`, `DiffResult`, `Difference`, severity/kind constants.
  - `shadiff/internal/storage` -- `FileStore` for data access.
  - `shadiff/internal/logger` -- structured logging for diff events.
- External:
  - Standard library: `fmt`.

## 6. Change Impact
- Changes to the comparison phases in `compareRecords` directly affect diff output and match rates.
- Adding new comparison dimensions (e.g., response time tolerance) requires modifying `compareRecords`.
- Changes to `EngineConfig` affect all callers (CLI diff command, HTTP handler).
- The `RuleSet` configuration (built-in matchers and their parameters) affects which differences are automatically ignored.

## 7. Maintenance Notes
- The `NumericToleranceMatcher` tolerance (0.001) is hardcoded; consider making it configurable via `EngineConfig`.
- Side effect comparison in `compareRecords` only checks count; detailed DB/Mongo comparison functions exist in `db.go` and `mongo.go` but are not currently called from the engine. Integration may be needed.
- Header comparison only checks expected headers against actual; extra headers in the replay response are not reported.
- Errors during `SaveResults` are logged but do not fail the run; callers should be aware that results may not be persisted.
