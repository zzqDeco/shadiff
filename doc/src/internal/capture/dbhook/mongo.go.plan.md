# mongo.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/dbhook/mongo.go
- Doc file: doc/src/internal/capture/dbhook/mongo.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture/dbhook

## 2. Core Responsibility
- Implements the MongoDB protocol parser used by the shared DB hook TCP proxy to extract database commands from OP_MSG traffic.
- Parses BSON documents with a simplified built-in parser to identify CRUD operations and emit typed MongoDB side effects.

## 3. Inputs & Outputs
- Input sources: raw MongoDB wire protocol bytes from client-to-server traffic.
- Output results:
  - `model.SideEffect` values with `database.type=mongo` and a `database.mongo` typed payload.
  - `MongoCommandToJSON(effect)` formatting helper for readable command JSON.

## 4. Key Implementation Details
- `MongoHook` embeds the shared `tcpProxy` and supplies a `mongoParser` factory.
- `mongoParser.Feed(data)` buffers fragmented TCP reads and parses complete wire messages by length header.
- `parseOpMsg` handles OP_MSG body sections; section kind 0 is parsed as a BSON command document and section kind 1 is skipped.
- `extractMongoCommand` recognizes `find`, `insert`, `update`, `delete`, `aggregate`, `count`, `distinct`, and `findAndModify`.
- `simpleBSONToMap` supports a limited BSON subset sufficient for the demo/integration paths.

## 5. Dependencies
- Internal: `dbtype`, `model`.
- External: `encoding/binary`, `encoding/json`, `time`.

## 6. Change Impact
- Changes to `MongoSideEffect` require updates here and in `internal/diff/mongo.go`.
- TCP lifecycle behavior lives in `tcp_proxy.go`; this file should remain focused on MongoDB framing and command extraction.

## 7. Maintenance Notes
- The BSON parser is intentionally simplified and does not handle all BSON types.
- The 16MB message size limit matches MongoDB's default max BSON document size, but the actual server limit is configurable.
- If richer MongoDB diffing is added later, extend typed payload extraction and comparer tests together.
