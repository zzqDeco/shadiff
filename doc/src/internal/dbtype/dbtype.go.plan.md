# dbtype.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/dbtype/dbtype.go
- Doc file: doc/src/internal/dbtype/dbtype.go.plan.md
- File type: Go source
- Module: shadiff/internal/dbtype

## 2. Core Responsibility
- Defines the canonical database proxy type identifiers used across config validation, capture hook construction, and diff side-effect registration.

## 3. Inputs & Outputs
- Inputs: database type strings from CLI/config or internal registries.
- Outputs: supported type constants, a defensive supported list, boolean support checks, and a human-readable names string.

## 4. Key Implementation Details
- Constants: `MySQL`, `Postgres`, `Mongo`, and `Redis`.
- `Supported()` returns a copy of the supported type list.
- `IsSupported(dbType)` checks exact support.
- `Names()` formats supported types for validation errors.

## 5. Dependencies
- External: `strings`.

## 6. Change Impact
- Adding a database type starts here, then hook construction and side-effect comparison tests should be updated to prove coverage.

## 7. Maintenance Notes
- Keep this package free of capture/diff/model imports so it remains the shared source of truth without cycles.
