# record.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/model/record.go
- Doc file: doc/src/internal/model/record.go.plan.md
- File type: Go source
- Module: shadiff

## 2. Core Responsibility
- Defines the `Record` struct, which represents the complete behavior of a single API call, including its HTTP request/response pair, side effects, timing, and error state.
- This is the central data unit captured during recording and consumed during replay and diff operations.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: Populated by the capture proxy during traffic recording; deserialized from storage during replay and diff.
- Output results: Serialized JSON for persistence; consumed by the replay engine and diff engine for comparison.

## 4. Key Implementation Details
- Structs/interfaces:
  - `Record` -- Fields: `ID` (unique record ID), `SessionID` (owning session reference), `Sequence` (ordering within session, used for pairing during diff), `Request` (HTTPRequest), `Response` (HTTPResponse), `SideEffects` (slice of SideEffect), `Duration` (request duration in ms), `RecordedAt` (Unix ms), `Error` (optional collection error message).
- Exported functions/methods: None. This file is purely a type definition.
- Constants: None.

## 5. Dependencies
- Internal: References `HTTPRequest`, `HTTPResponse`, and `SideEffect` types from the same `model` package (defined in `request.go` and `sideeffect.go`).
- External: None (no imports).

## 6. Change Impact
- Field changes affect the JSON serialization contract shared with storage, API, and diff layers.
- The `Sequence` field is critical for record pairing during replay comparison; changes to its semantics require updates in the diff engine.
- The `SideEffects` slice ties records to database and external call tracking; structural changes here propagate to the side-effect capture hooks.

## 7. Maintenance Notes
- The `Error` field uses `omitempty` so it is only present when a capture error occurred; other fields are always serialized.
- `Duration` and `RecordedAt` are both in milliseconds -- maintain consistency with `Session.CreatedAt`/`UpdatedAt`.
- When adding fields, consider backward compatibility with existing stored session data.
