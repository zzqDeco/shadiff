# db.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/diff/db.go
- Doc file: doc/src/internal/diff/db.go.plan.md
- File type: Go source
- Module: shadiff/internal/diff

## 2. Core Responsibility
- Compares SQL database (MySQL/PostgreSQL) side effects between recorded and replayed traffic, detecting differences in query counts and SQL statement content.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: Two `[]model.SideEffect` slices (original and replay) containing captured database operations.
- Output results: `[]model.Difference` reporting query count mismatches and SQL statement differences.

## 4. Key Implementation Details
- Structs/interfaces: None (package-level functions only).
- Exported functions/methods:
  - `CompareDBSideEffects(original, replay)` -- compares SQL side effects:
    1. Filters both slices to include only `mysql` and `postgres` typed effects.
    2. Compares query counts between original and replay.
    3. Pairs queries by order (index) and compares normalized SQL statements.
    4. Returns differences with `DiffDBQueryCount` or `DiffDBQuery` kinds.
- Unexported functions/methods:
  - `normalizeSQL(sql)` -- normalizes SQL for comparison by trimming whitespace, collapsing multiple spaces, and converting to uppercase.
  - `filterByType(effects, dbTypes...)` -- filters side effects where `Type == SideEffectDB` and `DBType` matches one of the specified types.
- Key behaviors:
  - SQL normalization enables comparison that ignores formatting differences (whitespace, casing).
  - Queries are compared positionally (by index), not by content matching; reordered queries will be reported as different.
  - Only the minimum overlapping count of queries is compared element-by-element; extra queries on either side are captured only by the count difference.

## 5. Dependencies
- Internal: `shadiff/internal/model` (for `SideEffect`, `Difference`, `DiffDBQueryCount`, `DiffDBQuery`, `SideEffectDB`, severity constants)
- External:
  - Standard library: `fmt`, `strings`.

## 6. Change Impact
- This function is not currently called from `engine.go`'s `compareRecords`; integrating it would add SQL-level diff granularity to the diff engine output.
- Changes to `normalizeSQL` affect what SQL differences are detected vs. ignored.
- Changes to `filterByType` affect which database types are included in comparison.

## 7. Maintenance Notes
- The SQL normalization is basic (whitespace + case); it does not handle parameter placeholder differences, comments, or dialect-specific syntax.
- Consider adding support for comparing query parameters/bindings, not just the SQL text.
- The function is currently standalone; to activate it, `engine.go` needs to call it within `compareRecords` for detailed side-effect analysis.
- Positional pairing may produce misleading results if query order legitimately differs between recorded and replayed runs; consider adding an unordered comparison mode.
