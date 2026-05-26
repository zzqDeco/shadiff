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
- Input sources: Populated by database proxy hooks (MySQL, PostgreSQL, MongoDB, Redis) and external HTTP call interceptors during traffic capture.
- Output results: Stored as part of `Record.SideEffects`; consumed by the diff engine to detect behavioral differences in database queries and external calls between recorded and replayed runs.

## 4. Key Implementation Details
- Structs/interfaces:
  - `SideEffect` -- Shared envelope with `Type`, `Timestamp`, `Duration`, and typed optional payloads `Database` / `HTTP`.
  - `DatabaseSideEffect` -- Database payload discriminator with `Type` (`mysql`, `postgres`, `mongo`, `redis`) and protocol-specific `SQL`, `Mongo`, or `Redis` payload.
  - `SQLSideEffect` -- SQL query, args, and row count.
  - `MongoSideEffect` -- Database, collection, operation, filter, update, documents, and document count.
  - `RedisSideEffect` -- Command, primary key, and normalized/redacted args.
  - `HTTPSideEffect` -- External HTTP request/response payload.
- Exported functions/methods:
  - `NewSQLSideEffect`, `NewMongoSideEffect`, `NewRedisSideEffect` constructors.
  - `DatabaseType`, `SQL`, `Mongo`, and `Redis` safe accessors.
- Constants:
  - `SideEffectDB` ("database") -- Database operation side effect.
  - `SideEffectHTTP` ("http_call") -- External HTTP call side effect.

## 5. Dependencies
- Internal: References `HTTPRequest` and `HTTPResponse` from `request.go`; uses `internal/dbtype` constants for constructor payloads.
- External: None.

## 6. Change Impact
- Changes affect `Record` (record.go) since records contain a `SideEffect` slice.
- Diff engine logic that compares side effects consumes typed payload accessors rather than flat fields.
- Adding new `SideEffectType` values requires corresponding diff logic and capture hook implementations.
- This v0.4.0 shape intentionally breaks compatibility with v0.3.x flat side-effect JSON.

## 7. Maintenance Notes
- Keep one typed database payload populated per database side effect.
- MongoDB fields (`Filter`, `Update`, `Documents`) use `any` type to accommodate arbitrary BSON-to-JSON conversions.
- Redis arguments are already normalized, base64-encoded when non-UTF-8, and redacted for known credential-bearing commands before they reach the model.
- `HTTP` is a pointer to allow nil when the side effect is not an HTTP call; always nil-check before access.
- When adding a new side-effect type, prefer a typed nested payload rather than reintroducing flat union fields.
