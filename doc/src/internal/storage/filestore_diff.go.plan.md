# filestore_diff.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/storage/filestore_diff.go
- Doc file: doc/src/internal/storage/filestore_diff.go.plan.md
- File type: Go source
- Module: shadiff/internal/storage

## 2. Core Responsibility
- Implements diff result persistence for `FileStore`.
- Saves and loads `diff-results.json` for report generation and session inspection.

## 3. Inputs & Outputs
- Input sources: `[]model.DiffResult`.
- Output results: pretty-printed JSON diff result files under each session directory.

## 4. Key Implementation Details
- `SaveResults()` writes the full result slice to `diff-results.json`.
- `LoadResults()` returns `nil, nil` when no diff result file exists.
- JSON decode errors are returned with context.

## 5. Dependencies
- Internal: `shadiff/internal/model`.
- External: Go standard library only.

## 6. Change Impact
- Changes affect `shadiff diff`, `shadiff report`, and `shadiff session inspect`.

## 7. Maintenance Notes
- Keep result file naming stable unless all readers and docs are updated together.

