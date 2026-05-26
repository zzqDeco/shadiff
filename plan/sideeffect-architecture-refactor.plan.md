# Side-effect Architecture Refactor Plan

## Goal

Reduce DB side-effect coupling after Redis support by replacing the flat side-effect union, scattered DB type hardcoding, manual diff dispatch, and duplicated DB hook TCP lifecycle.

## Scope

- Breaking change for stored session JSONL side effects; v0.3.x sessions are not migrated.
- No CLI flag or config shape changes.
- No new database type; existing MySQL, PostgreSQL, MongoDB, and Redis behavior is preserved.

## Approach

- Replace flat `SideEffect` DB fields with typed payloads under `database.sql`, `database.mongo`, and `database.redis`.
- Add `internal/dbtype` as the single supported DB type source.
- Add a side-effect comparer registry so the diff engine calls one side-effect comparison entrypoint.
- Extract shared DB hook TCP proxy lifecycle into a common private proxy; protocol hooks now provide stream parsers.
- Keep parser behavior transparent: malformed or unsupported protocol data is forwarded and simply emits no side effect.

## Tasks

- [x] Add typed side-effect payload model and tests for the new JSON shape.
- [x] Centralize supported DB types and use them in config validation, hook creation, and diff registration.
- [x] Replace diff engine direct SQL/Mongo/Redis calls with side-effect comparer registry.
- [x] Extract common TCP proxy lifecycle and convert protocol hooks to stream parsers.
- [x] Update integration tests, official E2E assertions, and documentation.

## Verification

- `go test ./...`
- `go test -tags integration ./internal/integration -count=1 -timeout=20m`
- `bash -n examples/e2e/run.sh`
- `./examples/e2e/run.sh --assert`
- `go build -o shadiff .`
