# mongo.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/diff/mongo.go
- Doc file: doc/src/internal/diff/mongo.go.plan.md
- File type: Go source
- Module: shadiff/internal/diff

## 2. Core Responsibility
- Compares MongoDB side effects between recorded and replayed traffic, detecting differences in operation counts, collection names, operation types, and database names.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: Two `[]model.SideEffect` slices (original and replay) containing captured MongoDB operations.
- Output results: `[]model.Difference` reporting operation count mismatches and per-operation field differences.

## 4. Key Implementation Details
- Structs/interfaces: None (package-level functions only).
- Exported functions/methods:
  - `CompareMongoSideEffects(original, replay)` -- compares MongoDB side effects:
    1. Filters both slices to include only `mongo` typed effects.
    2. Compares operation counts.
    3. Pairs operations by order (index) and compares three fields:
       - `Collection` -- error severity if different.
       - `Operation` -- error severity if different.
       - `Database` -- warning severity if different (less critical).
- Unexported functions/methods:
  - `filterMongoEffects(effects)` -- filters side effects where `Type == SideEffectDB` and `DBType == "mongo"`.
- Key behaviors:
  - Database name differences are treated as warnings (lower severity than collection/operation differences) since database routing may legitimately differ between environments.
  - Operations are compared positionally by index, same as the SQL comparator in `db.go`.
  - All differences use the `DiffMongoOp` kind constant.

## 5. Dependencies
- Internal: `shadiff/internal/model` (for `SideEffect`, `Difference`, `DiffMongoOp`, `SideEffectDB`, severity constants)
- External:
  - Standard library: `fmt`.

## 6. Change Impact
- Like `db.go`, this function is not currently called from `engine.go`'s `compareRecords`; integration would add MongoDB-level granularity to diff results.
- Changes to the comparison fields or severity levels affect how MongoDB operation differences are reported and whether they cause match failures.

## 7. Maintenance Notes
- Consider comparing MongoDB query filters and update documents for deeper semantic analysis.
- The positional pairing assumption may not hold if MongoDB operations are non-deterministically ordered; consider an unordered comparison mode or grouping by collection+operation.
- The separation between `db.go` (SQL) and `mongo.go` (MongoDB) keeps concerns clean; maintain this separation for future NoSQL database support (e.g., Redis, DynamoDB).
- Integration into `engine.go` requires calling `CompareMongoSideEffects` alongside `CompareDBSideEffects` within the side-effect comparison phase of `compareRecords`.
