# postgres.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/dbhook/postgres.go
- Doc file: doc/src/internal/capture/dbhook/postgres.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture/dbhook

## 2. Core Responsibility
- Implements the PostgreSQL protocol parser used by the shared DB hook TCP proxy to extract SQL statements from frontend messages.
- Emits captured SQL queries as `model.SideEffect` events on a buffered channel for consumption by the `Recorder`.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - TCP connections from PostgreSQL clients connecting to `listenAddr`.
  - Raw PostgreSQL wire protocol frontend messages on the client-to-server stream.
- Output results:
  - `model.SideEffect` events with `database.type=postgres` and `database.sql.query`.

## 4. Key Implementation Details
- Structs/interfaces:
  - `PostgresHook` -- Embeds the shared `tcpProxy` and supplies a PostgreSQL parser factory.
  - `postgresParser` -- Buffers fragmented frontend messages and tracks the startup phase.
- Exported functions/methods:
  - `NewPostgresHook(listenAddr, targetAddr string) *PostgresHook` -- Constructor.
  - DBHook lifecycle methods are inherited from embedded `tcpProxy`.
  - `(*postgresParser).Feed(data)` -- Parses stream data and returns typed SQL side effects.
- Unexported helpers:
  - `extractNullTermString(data []byte) string` -- Extracts a C-style null-terminated string from a byte slice.
  - `nullTermIndex(data []byte) int` -- Returns the index of the first null byte, or -1 if not found.
- Protocol constants:
  - `pgMsgQuery` ('Q') -- Simple Query message type.
  - `pgMsgParse` ('P') -- Extended Query Parse message type.
- Key behaviors:
  - `postgresParser` skips the initial startup message, then iterates through one or more frontend messages in its buffered stream.
  - Each frontend message has a 1-byte type, 4-byte big-endian length, and variable-length payload.
  - For Simple Query ('Q'), the payload is a null-terminated SQL string.
  - For Parse ('P'), the payload contains a null-terminated statement name followed by a null-terminated query string. The parser skips the statement name to extract the query.

## 5. Dependencies
- Internal:
  - `shadiff/internal/dbtype` -- PostgreSQL type constant.
  - `shadiff/internal/model` -- typed side-effect constructors.
- External: `encoding/binary`, `time`.

## 6. Change Impact
- `internal/capture/dbhook/hook.go` -- Constructor registry calls `NewPostgresHook`; constructor signature changes require a corresponding update.
- `internal/capture/recorder.go` -- Consumes the `SideEffects()` channel; changes to the channel protocol or `SideEffect` field usage affect the recorder.
- `internal/model/` -- Changes to typed SQL payload fields require parser and diff updates.

## 7. Maintenance Notes
- The startup phase detection is simplified: it assumes the first message of 8+ bytes is the startup message and then switches to normal message parsing. This does not handle SSL negotiation requests (`SSLRequest` message) which may precede the startup message. For production use, detect the SSLRequest (protocol version `80877103`) and handle the SSL handshake or rejection before proceeding.
- The parser only captures Simple Query and Parse messages. Other Extended Query protocol messages (Bind, Execute, Describe) are not captured. If full prepared statement tracking is needed, these should be handled as well.
- `extractNullTermString` and `nullTermIndex` are package-level unexported functions that could be reused by other hooks if needed.
- TCP lifecycle behavior lives in `tcp_proxy.go`; keep this file focused on PostgreSQL message parsing.
