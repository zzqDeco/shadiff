# filestore.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/storage/filestore.go
- Doc file: doc/src/internal/storage/filestore.go.plan.md
- File type: Go source
- Module: shadiff/internal/storage

## 2. Core Responsibility
- Provides a file-system-based implementation of `SessionStore`, `RecordStore`, and `DiffStore` interfaces.
- Manages on-disk directory layout: `{baseDir}/sessions/{id}/session.json`, `records.jsonl`, `replay-records.jsonl`, `diff-results.json`.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: `model.Session`, `model.Record`, `model.DiffResult` structs from callers; file-system reads from the sessions directory.
- Output results: Persisted JSON/JSONL files on disk; deserialized model structs returned to callers.

## 4. Key Implementation Details
- Structs/interfaces:
  - `FileStore` -- concrete struct with `baseDir string` and `mu sync.RWMutex` for thread-safe file access.
- Exported functions/methods:
  - `NewFileStore(baseDir)` -- constructor; creates the `sessions/` subdirectory and returns a `*FileStore`.
  - `Create(session)` -- generates an 8-char UUID ID, sets timestamps, creates a session directory, and writes `session.json`.
  - `Get(id)` -- reads and deserializes `session.json` for the given ID.
  - `List(filter)` -- iterates all session directories, applies optional name/status/tag filtering, returns sorted by `UpdatedAt` descending.
  - `Update(session)` -- updates `UpdatedAt` timestamp and re-writes `session.json`.
  - `Delete(id)` -- removes the entire session directory with `os.RemoveAll`.
  - `AppendRecord(sessionID, record)` -- appends a JSON line to `records.jsonl`.
  - `AppendReplayRecord(sessionID, record)` -- appends a JSON line to `replay-records.jsonl`.
  - `ListRecords(sessionID)` -- reads and parses `records.jsonl`.
  - `ListReplayRecords(sessionID)` -- reads and parses `replay-records.jsonl`.
  - `GetRecord(sessionID, recordID)` -- linear scan of records to find by ID.
  - `CountRecords(sessionID)` -- returns `len(ListRecords(...))`.
  - `SaveResults(sessionID, results)` -- writes diff results as pretty-printed JSON to `diff-results.json`.
  - `LoadResults(sessionID)` -- reads and deserializes `diff-results.json`.
- Key behaviors:
  - All public methods acquire `sync.RWMutex` (read-lock for reads, write-lock for writes) for goroutine safety.
  - JSONL scanner buffer is set to 10 MB max per line to handle large response bodies.
  - Corrupted session directories and JSONL lines are silently skipped (no error propagation for individual corrupt entries).
  - `Tags` and `Metadata` fields are initialized to empty non-nil values on load and create to prevent JSON `null` serialization.

## 5. Dependencies
- Internal: `shadiff/internal/model`
- External:
  - `github.com/google/uuid` -- UUID generation for session IDs.
  - Standard library: `bufio`, `encoding/json`, `fmt`, `os`, `path/filepath`, `sort`, `strings`, `sync`, `time`.

## 6. Change Impact
- Changes to the directory layout or file naming affect all components that read session data (replay engine, diff engine, CLI/HTTP handlers).
- The mutex strategy affects concurrency guarantees; switching to per-session locks would require refactoring.
- `AppendReplayRecord` is called by the replay engine; `ListReplayRecords` is called by the diff engine.

## 7. Maintenance Notes
- `GetRecord` and `CountRecords` load all records into memory; consider adding an indexed lookup if record counts grow large.
- The 10 MB JSONL line buffer is hardcoded; consider making it configurable if payloads exceed this limit.
- `hasAnyTag` is an unexported helper for tag-based filtering; it uses a set-based lookup for O(n+m) performance.
- Silent skipping of corrupt data is intentional for resilience but may hide data issues; consider adding warning-level logging.
