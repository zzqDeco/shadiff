# filestore_artifact.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/storage/filestore_artifact.go
- Doc file: doc/src/internal/storage/filestore_artifact.go.plan.md
- File type: Go source
- Module: shadiff/internal/storage

## 2. Core Responsibility
- Implements request-body artifact read/write support for `FileStore`.
- Stores full request bodies outside JSONL while preserving session-relative references.

## 3. Inputs & Outputs
- Input sources: session ID, record ID, request-body `io.Reader`, and session-relative artifact refs.
- Output results: files under `artifacts/request-bodies/` and opened `io.ReadCloser` handles.

## 4. Key Implementation Details
- `SaveRequestBodyArtifact()` writes `artifacts/request-bodies/<recordID>.bin` and returns a slash-normalized session-relative path.
- `OpenRequestBodyArtifact()` validates refs through `requestBodyArtifactPath()` before opening files.
- Artifact reads and writes use the same `FileStore` mutex discipline as other store operations.

## 5. Dependencies
- External: Go standard library only.

## 6. Change Impact
- Changes affect large request-body capture and replay reconstruction.

## 7. Maintenance Notes
- Keep path validation in the shared path helper so artifact reads cannot escape the session directory.

