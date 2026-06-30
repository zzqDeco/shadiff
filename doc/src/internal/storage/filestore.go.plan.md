# filestore.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/storage/filestore.go
- Doc file: doc/src/internal/storage/filestore.go.plan.md
- File type: Go source
- Module: shadiff/internal/storage

## 2. Core Responsibility
- Defines the `FileStore` concrete type and constructor.
- Owns the top-level filesystem layout root: `{baseDir}/sessions/`.
- Method implementations are split across responsibility-specific files in the same package.

## 3. Inputs & Outputs
- Input sources: base data directory path.
- Output results: initialized `*FileStore` with a created `sessions/` directory.

## 4. Key Implementation Details
- `FileStore` keeps `baseDir string` and `mu sync.RWMutex` shared by all split implementation files.
- `NewFileStore(baseDir)` creates the `sessions/` subdirectory and returns the store.
- Session, record, artifact, diff-result, and path helper methods live in `filestore_session.go`, `filestore_record.go`, `filestore_artifact.go`, `filestore_diff.go`, and `filestore_path.go`.

## 5. Dependencies
- External: Go standard library only.

## 6. Change Impact
- Changes to `FileStore` fields or constructor affect all split implementation files.
- Changes to the root sessions directory layout affect every storage caller.

## 7. Maintenance Notes
- Keep this file small; add responsibility-specific behavior to the relevant sibling file.
