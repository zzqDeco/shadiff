# filestore_session.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/storage/filestore_session.go
- Doc file: doc/src/internal/storage/filestore_session.go.plan.md
- File type: Go source
- Module: shadiff/internal/storage

## 2. Core Responsibility
- Implements session metadata persistence for `FileStore`.
- Covers session create, get, list, update, delete, pruning, and private session JSON load/save helpers.

## 3. Inputs & Outputs
- Input sources: `model.Session` and `model.SessionFilter`.
- Output results: session directories and `session.json` files under `{baseDir}/sessions/{id}/`.

## 4. Key Implementation Details
- Public methods acquire the `FileStore` mutex before filesystem access.
- `Create()` assigns an 8-character UUID when needed, initializes timestamps, tags, and metadata, and writes `session.json`.
- `List()` skips corrupted session directories and sorts by `UpdatedAt` descending.
- `PruneOldest()` removes oldest non-recording sessions until the configured maximum is satisfied.

## 5. Dependencies
- Internal: `shadiff/internal/model`.
- External: Go standard library plus `github.com/google/uuid`.

## 6. Change Impact
- Changes affect all CLI, capture, replay, diff, and report flows that resolve sessions.
- Session JSON shape remains defined by `internal/model.Session`.

## 7. Maintenance Notes
- Keep session metadata logic here rather than adding it back to `filestore.go`.

