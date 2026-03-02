# session.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/model/session.go
- Doc file: doc/src/internal/model/session.go.plan.md
- File type: Go source
- Module: shadiff

## 2. Core Responsibility
- Defines the data model for recording sessions, including the `Session` struct, its status lifecycle, endpoint configuration, and filtering criteria.
- A session groups related API call records and tracks their source/target endpoints, tags, and metadata.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: JSON deserialization from storage layer; programmatic construction by session management logic.
- Output results: Serialized JSON for persistence and API responses; used as a container reference by `Record` instances via `SessionID`.

## 4. Key Implementation Details
- Structs/interfaces:
  - `Session` -- Top-level session entity with fields: `ID` (8-char UUID short ID), `Name`, `Description`, `Source` (EndpointConfig), `Target` (EndpointConfig), `Tags` (string slice), `RecordCount`, `CreatedAt`/`UpdatedAt` (Unix ms), `Status` (SessionStatus), `Metadata` (string map).
  - `EndpointConfig` -- Endpoint descriptor with fields: `BaseURL`, `Headers` (string map).
  - `SessionFilter` -- Query filter with fields: `Name` (fuzzy match), `Status`, `Tags`.
- Exported functions/methods: None. This file is purely type definitions.
- Constants:
  - `SessionRecording` ("recording") -- Session is actively capturing traffic.
  - `SessionCompleted` ("completed") -- Session capture is finished.
  - `SessionReplayed` ("replayed") -- Session has been replayed against a target.

## 5. Dependencies
- Internal: None.
- External: None (no imports).

## 6. Change Impact
- Any struct field changes affect JSON serialization contracts, storage layer, and API responses.
- `SessionStatus` constant changes affect status filtering logic and any UI/CLI that checks session state.
- `EndpointConfig` changes propagate to capture proxy and replay engine configuration.
- `SessionFilter` changes affect session listing and query APIs.

## 7. Maintenance Notes
- All JSON tags use camelCase; maintain this convention when adding fields.
- `CreatedAt` and `UpdatedAt` are Unix milliseconds, not seconds -- ensure consistency across the codebase.
- `SessionFilter` fields use `omitempty` to support partial filter queries.
- Adding new `SessionStatus` values requires updating any switch/case or validation logic elsewhere in the project.
