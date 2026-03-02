# store.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/storage/store.go
- Doc file: doc/src/internal/storage/store.go.plan.md
- File type: Go source
- Module: shadiff/internal/storage

## 2. Core Responsibility
- Defines the core storage interfaces (contracts) for the shadiff persistence layer: session management, record storage, and diff result storage.
- All concrete storage implementations must satisfy these interfaces.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: This file does not receive data directly; it declares method signatures that accept `model.Session`, `model.Record`, `model.DiffResult`, `model.SessionFilter`, and primitive identifiers (`string`).
- Output results: Method signatures return the same model types, slices thereof, counts (`int`), and errors.

## 4. Key Implementation Details
- Structs/interfaces:
  - `SessionStore` -- interface for CRUD operations on sessions.
  - `RecordStore` -- interface for appending, listing, getting, and counting recorded HTTP traffic records (JSONL streaming model).
  - `DiffStore` -- interface for saving and loading diff comparison results.
- Exported functions/methods:
  - `SessionStore.Create(session)` -- persist a new session.
  - `SessionStore.Get(id)` -- retrieve a session by ID.
  - `SessionStore.List(filter)` -- list sessions with optional filtering.
  - `SessionStore.Update(session)` -- update session metadata.
  - `SessionStore.Delete(id)` -- remove a session.
  - `RecordStore.AppendRecord(sessionID, record)` -- append a record in JSONL streaming format.
  - `RecordStore.ListRecords(sessionID)` -- read all records for a session.
  - `RecordStore.GetRecord(sessionID, recordID)` -- retrieve a single record.
  - `RecordStore.CountRecords(sessionID)` -- return the number of records.
  - `DiffStore.SaveResults(sessionID, results)` -- persist diff results.
  - `DiffStore.LoadResults(sessionID)` -- load diff results.
- Key behaviors: This file is purely declarative; it contains no logic, only interface definitions.

## 5. Dependencies
- Internal: `shadiff/internal/model` (for `Session`, `SessionFilter`, `Record`, `DiffResult` types)
- External: none

## 6. Change Impact
- Any change to these interfaces requires updating all implementations (currently `FileStore` in `filestore.go`).
- Consumers across the codebase (replay engine, diff engine, HTTP handlers) depend on these contracts.

## 7. Maintenance Notes
- Keep interface methods minimal; avoid adding methods that only one implementation needs.
- When adding a new storage concern (e.g., replay records), consider whether it belongs as a new interface or an extension of an existing one.
- The `RecordStore` comment explicitly mentions JSONL streaming; any new implementation should preserve this streaming-write semantic.
