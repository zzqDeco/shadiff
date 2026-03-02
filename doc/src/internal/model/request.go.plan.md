# request.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/model/request.go
- Doc file: doc/src/internal/model/request.go.plan.md
- File type: Go source
- Module: shadiff

## 2. Core Responsibility
- Defines the HTTP request and response data models (`HTTPRequest` and `HTTPResponse`) used throughout the project for recording, replaying, and comparing API traffic.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: Populated by the capture proxy from live HTTP traffic; deserialized from stored records during replay and diff.
- Output results: Used by the replay engine to reconstruct outgoing requests; used by the diff engine to compare recorded vs. replayed responses. Also referenced by `SideEffect` for external HTTP call tracking.

## 4. Key Implementation Details
- Structs/interfaces:
  - `HTTPRequest` -- Fields: `Method` (HTTP method), `Path` (path without host), `Query` (raw query string), `Headers` (multi-value header map: `map[string][]string`), `Body` (raw bytes), `BodyLen` (body length for large body truncation scenarios).
  - `HTTPResponse` -- Fields: `StatusCode` (HTTP status code), `Headers` (multi-value header map), `Body` (raw bytes), `BodyLen` (body length).
- Exported functions/methods: None. This file is purely type definitions.
- Constants: None.

## 5. Dependencies
- Internal: None.
- External: None (no imports).

## 6. Change Impact
- These types are used by `Record` (record.go), `SideEffect` (sideeffect.go via pointer references), and `DiffResult` (diff.go).
- Field changes affect JSON serialization, the capture proxy, replay engine, diff engine, and any API endpoints that expose request/response data.
- The `Headers` field uses `map[string][]string` (multi-value) to match Go's `http.Header` type; changing this would require updates across all HTTP handling code.

## 7. Maintenance Notes
- `Body` is stored as `[]byte` and will be base64-encoded in JSON output; consumers must handle this encoding.
- `BodyLen` exists separately from `len(Body)` to support truncation scenarios where the full body is not stored but its original length is preserved.
- `Path` is stored without the host portion; the host comes from `EndpointConfig.BaseURL` in the session.
