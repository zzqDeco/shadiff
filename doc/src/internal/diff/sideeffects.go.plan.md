# sideeffects.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/diff/sideeffects.go
- Doc file: doc/src/internal/diff/sideeffects.go.plan.md
- File type: Go source
- Module: shadiff/internal/diff

## 2. Core Responsibility
- Provides the side-effect comparer registry used by the diff engine.
- Decouples `Engine.compareRecords` from SQL, MongoDB, and Redis comparer implementation details.

## 3. Inputs & Outputs
- Inputs: recorded and replayed `[]model.SideEffect`.
- Outputs: combined `[]model.Difference` from all registered comparers plus residual side-effect count differences.

## 4. Key Implementation Details
- `SideEffectComparer` declares handled DB types and a comparison function.
- `defaultSideEffectComparers` registers SQL (`mysql`, `postgres`), MongoDB, and Redis comparers.
- `CompareSideEffects()` is the public package entrypoint used by the engine.
- Residual side effects are counted based on DB types not claimed by registered comparers.
- Regression tests verify registry coverage for every supported DB type, SQL/MongoDB/Redis dispatch through the registry, and residual counting for unhandled side effects.

## 5. Dependencies
- Internal: `dbtype`, `model`.
- External: `fmt`.

## 6. Change Impact
- Adding a new semantic side-effect comparer should update this registry and its coverage test.

## 7. Maintenance Notes
- Keep registry order deterministic because diff output order is user-visible in reports and JSON.
