# filestore_path.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/storage/filestore_path.go
- Doc file: doc/src/internal/storage/filestore_path.go.plan.md
- File type: Go source
- Module: shadiff/internal/storage

## 2. Core Responsibility
- Provides private path-safety helpers for `FileStore`.
- Currently validates request-body artifact refs before opening artifact files.

## 3. Inputs & Outputs
- Input sources: session ID and session-relative artifact ref.
- Output results: validated absolute filesystem path or a descriptive error.

## 4. Key Implementation Details
- Empty refs and `..` traversal are rejected.
- Resolved paths are checked with `filepath.Rel()` to ensure they remain under the session directory.

## 5. Dependencies
- External: Go standard library only.

## 6. Change Impact
- Changes affect artifact replay safety and request-body artifact reads.

## 7. Maintenance Notes
- Add future storage path validation helpers here rather than scattering path checks across store files.

