# sideeffect.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/model/sideeffect.go
- Doc file: doc/src/internal/model/sideeffect.go.plan.md
- File type: Go source
- Module: shadiff

## 2. Core Responsibility
- Defines the `SideEffect` model and its type constants, representing observable side effects (database operations, external HTTP calls) produced during API request processing.
- Enables shadiff to capture and compare not just HTTP responses but also the underlying database queries and outbound HTTP calls triggered by each API invocation.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: Populated by database proxy hooks (MySQL, PostgreSQL, MongoDB) and external HTTP call interceptors during traffic capture.
- Output results: Stored as part of `Record.SideEffects`; consumed by the diff engine to detect behavioral differences in database queries and external calls between recorded and replayed runs.

## 4. Key Implementation Details
- Structs/interfaces:
  - `SideEffect` -- A union-style struct with shared fields and database/HTTP-specific fields:
    - Shared: `Type` (SideEffectType), `Timestamp` (Unix ms), `Duration` (ms).
    - SQL databases (MySQL/PostgreSQL): `DBType`, `Query`, `Args` ([]any), `RowCount`.
    - MongoDB: `Database`, `Collection`, `Operation` (find/insert/update/delete/aggregate), `Filter`, `Update`, `Documents`, `DocCount`.
    - External HTTP: `HTTPReq` (*HTTPRequest), `HTTPResp` (*HTTPResponse).
- Exported functions/methods: None. This file is purely type definitions.
- Constants:
  - `SideEffectDB` ("database") -- Database operation side effect.
  - `SideEffectHTTP` ("http_call") -- External HTTP call side effect.

## 5. Dependencies
- Internal: References `HTTPRequest` and `HTTPResponse` from `request.go` (same package) via pointer fields.
- External: None (no imports).

## 6. Change Impact
- Changes affect `Record` (record.go) since records contain a `SideEffect` slice.
- Diff engine logic that compares side effects (db_query, mongo_op, external_call difference kinds in diff.go) depends on these field names and types.
- Adding new `SideEffectType` values requires corresponding diff logic and capture hook implementations.

## 7. Maintenance Notes
- The struct uses a flat union approach with `omitempty` on all type-specific fields. Only populate fields relevant to the `Type` value.
- MongoDB fields (`Filter`, `Update`, `Documents`) use `any` type to accommodate arbitrary BSON-to-JSON conversions.
- `HTTPReq` and `HTTPResp` are pointers to allow nil when the side effect is not an HTTP call; always nil-check before access.
- When adding a new side effect type (e.g., message queue, cache), add a new `SideEffectType` constant and extend the struct with new `omitempty` fields.
