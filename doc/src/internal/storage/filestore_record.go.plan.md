# filestore_record.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/storage/filestore_record.go
- Doc file: doc/src/internal/storage/filestore_record.go.plan.md
- File type: Go source
- Module: shadiff/internal/storage

## 2. Core Responsibility
- Implements record and replay-record JSONL persistence for `FileStore`.
- Provides append, list, lookup, and count behavior for records.

## 3. Inputs & Outputs
- Input sources: `model.Record` values from capture and replay.
- Output results: JSONL files named `records.jsonl` and `replay-records.jsonl`.

## 4. Key Implementation Details
- `AppendRecord()` writes to `records.jsonl`; `AppendReplayRecord()` writes to `replay-records.jsonl`.
- JSONL lines are appended under the `FileStore` write lock.
- `listRecords()` returns `nil, nil` when the JSONL file is missing.
- Corrupted JSONL lines are skipped, preserving the previous resilience behavior.
- Scanner max line size remains 10 MB.

## 5. Dependencies
- Internal: `shadiff/internal/model`.
- External: Go standard library only.

## 6. Change Impact
- Changes affect capture persistence, replay persistence, diff input loading, and session inspection counts.

## 7. Maintenance Notes
- Keep JSONL format changes explicit and coordinated with replay/diff/report readers.

