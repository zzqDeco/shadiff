# mongo.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/dbhook/mongo.go
- Doc file: doc/src/internal/capture/dbhook/mongo.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture/dbhook

## 2. Core Responsibility
- Implements a TCP proxy for MongoDB that transparently forwards traffic between a client and a real MongoDB server while sniffing the client-to-server stream to extract database commands from the MongoDB OP_MSG wire protocol.
- Parses BSON documents without a third-party BSON library using a simplified built-in parser to identify CRUD operations, and emits them as `model.SideEffect` events.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - TCP connections from MongoDB clients connecting to `listenAddr`.
  - Raw MongoDB wire protocol messages on the client-to-server stream.
- Output results:
  - Proxied TCP traffic forwarded transparently to `targetAddr` (the real MongoDB server) and back to the client.
  - `model.SideEffect` events (type `SideEffectDB`, dbType `mongo`) emitted on the `sideEffects` channel, containing operation type, collection name, database name, filter, update, and document data.
  - A utility function `MongoCommandToJSON` for converting side effects to readable JSON strings.
  - Log events via `logger.DBHookEvent`, `logger.Error`, and `logger.Warn`.

## 4. Key Implementation Details
- Structs/interfaces:
  - `MongoHook` -- Implements `DBHook`. Holds listen/target addresses, a `net.Listener`, a buffered side-effects channel (capacity 1000), a `done` channel for shutdown, and a `sync.WaitGroup` for goroutine lifecycle management.
- Exported functions/methods:
  - `NewMongoHook(listenAddr, targetAddr string) *MongoHook` -- Constructor.
  - `(*MongoHook).Type() string` -- Returns `"mongo"`.
  - `(*MongoHook).SideEffects() <-chan model.SideEffect` -- Returns the read-only side-effects channel.
  - `(*MongoHook).Start(ctx context.Context) error` -- Opens a TCP listener and spawns an accept loop goroutine.
  - `(*MongoHook).Stop() error` -- Signals shutdown, closes the listener, waits for all goroutines, and closes the side-effects channel.
  - `MongoCommandToJSON(effect model.SideEffect) string` -- Converts a MongoDB side effect into a human-readable JSON string containing operation, collection, database, filter, update, and documents fields.
- Unexported helpers:
  - `simpleBSONToMap(data []byte) map[string]any` -- A simplified BSON parser that extracts key-value pairs from a BSON document without requiring the official MongoDB driver. Supports types: UTF-8 string (0x02), embedded document (0x03), array (0x04), int32 (0x10), int64 (0x12), double (0x01, skipped), boolean (0x08), null (0x0A), and ObjectId (0x07, skipped).
- Protocol constants:
  - `opMsgOpCode` (2013) -- MongoDB OP_MSG opcode.
- Key behaviors:
  - Unlike the MySQL and PostgreSQL hooks which do streaming `Read` calls, the MongoDB hook uses `io.ReadFull` to read the 16-byte wire protocol header first, then reads the exact remaining body based on the message length. This provides proper message framing.
  - Message length is validated against a 16MB upper bound to prevent memory exhaustion from malformed messages. Invalid lengths cause a fallback to `io.Copy` passthrough.
  - `parseOpMsg` processes the OP_MSG body by iterating over sections. Section kind 0 (body) contains a single BSON document that is parsed for command extraction. Section kind 1 (document sequence) is skipped.
  - `extractMongoCommand` identifies CRUD operations by checking for known command keys: `find`, `insert`, `update`, `delete`, `aggregate`, `count`, `distinct`, `findAndModify`. The corresponding value is treated as the collection name.
  - Additional fields are extracted: `$db` for database name, `filter` for query conditions, `updates` for update operations, and `documents` for inserted documents.
  - Non-CRUD commands (e.g., `isMaster`, `buildInfo`) are silently skipped.

## 5. Dependencies
- Internal:
  - `shadiff/internal/logger` -- Logging for lifecycle events, errors, and warnings.
  - `shadiff/internal/model` -- `SideEffect` type and `SideEffectDB` constant. Uses MongoDB-specific fields: `Operation`, `Collection`, `Database`, `Filter`, `Update`, `Documents`.
- External:
  - `context` -- `Start` method signature (context not actively used for cancellation).
  - `encoding/binary` -- Little-endian parsing for wire protocol headers and BSON documents.
  - `encoding/json` -- JSON marshaling in `MongoCommandToJSON`.
  - `io` -- `io.ReadFull` for framed reads and `io.Copy` for passthrough.
  - `net` -- TCP listener and dialer.
  - `sync` -- `WaitGroup` for goroutine coordination.
  - `time` -- Dial timeout (10 seconds) and timestamps on side effects.

## 6. Change Impact
- `internal/capture/dbhook/hook.go` -- `NewHook` factory directly calls `NewMongoHook`; constructor signature changes require a corresponding update.
- `internal/capture/recorder.go` -- Consumes the `SideEffects()` channel; changes to the channel protocol or `SideEffect` field usage affect the recorder.
- `internal/model/` -- This file uses more `SideEffect` fields than the SQL hooks (`Operation`, `Collection`, `Database`, `Filter`, `Update`, `Documents`). Changes to any of these fields require updates here.
- Any code calling `MongoCommandToJSON` is affected by changes to that function's output format.

## 7. Maintenance Notes
- The `simpleBSONToMap` parser is intentionally simplified and does not handle all BSON types (e.g., Decimal128, regex, binary, timestamps, min/max keys). For full fidelity, consider using `go.mongodb.org/mongo-driver/bson` to decode documents.
- Double values (type 0x01) are read but discarded (offset advanced without storing). If numeric filter values or aggregation parameters need to be captured, extend this case.
- ObjectId values (type 0x07) are similarly skipped. If document IDs need to be captured, decode and store them.
- The 16MB message size limit matches MongoDB's default max BSON document size, but the actual server limit is configurable. Align if needed.
- The `ctx` parameter in `Start` is accepted but not used for cancellation.
- The `MongoCommandToJSON` utility function is stateless and could be moved to a shared formatting package if used by multiple consumers.
- Unlike the SQL hooks that only capture query strings, this hook captures structured command data (filter, update, documents), which makes it more informative but also more coupled to the `SideEffect` model.
