# mysql.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/dbhook/mysql.go
- Doc file: doc/src/internal/capture/dbhook/mysql.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture/dbhook

## 2. Core Responsibility
- Implements the MySQL protocol parser used by the shared DB hook TCP proxy to extract SQL statements from client-to-server traffic.
- Emits captured SQL queries as `model.SideEffect` events on a buffered channel for consumption by the `Recorder`.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - TCP connections from MySQL clients connecting to `listenAddr`.
  - Raw MySQL wire protocol packets on the client-to-server data stream.
- Output results:
  - `model.SideEffect` events with `database.type=mysql` and `database.sql.query`.

## 4. Key Implementation Details
- Structs/interfaces:
  - `MySQLHook` -- Embeds the shared `tcpProxy` and supplies a MySQL parser factory.
  - `mysqlParser` -- Buffers fragmented TCP reads and parses one or more MySQL packets per `Feed` call.
- Exported functions/methods:
  - `NewMySQLHook(listenAddr, targetAddr string) *MySQLHook` -- Constructor.
  - DBHook lifecycle methods are inherited from embedded `tcpProxy`.
  - `(*mysqlParser).Feed(data)` -- Parses stream data and returns typed SQL side effects.
- Protocol constants:
  - `mysqlComQuery` (0x03) -- COM_QUERY command byte.
  - `mysqlComStmtPrepare` (0x16) -- COM_STMT_PREPARE command byte.
  - `mysqlComStmtExecute` (0x17) -- COM_STMT_EXECUTE command byte (defined but not actively captured).
- Key behaviors:
  - `mysqlParser` extracts the 3-byte little-endian payload length, 1-byte sequence number, and 1-byte command byte. For `COM_QUERY` and `COM_STMT_PREPARE`, the payload is interpreted as a SQL string.
  - Fragmented reads and multiple packets in one read are handled by the parser buffer.
  - `COM_STMT_EXECUTE` is defined as a constant but not actively captured because the execute payload contains binary parameter data, not a readable SQL string.
  - The helper function `readMySQLPacketLength` exists but is not used by the main sniffing path (uses inline parsing instead).

## 5. Dependencies
- Internal:
  - `shadiff/internal/dbtype` -- MySQL type constant.
  - `shadiff/internal/model` -- typed side-effect constructors.
- External: `encoding/binary`, `time`.

## 6. Change Impact
- `internal/capture/dbhook/hook.go` -- Constructor registry calls `NewMySQLHook`; constructor signature changes require a corresponding update.
- `internal/capture/recorder.go` -- Consumes the `SideEffects()` channel; changes to the channel protocol or `SideEffect` field usage affect the recorder.
- `internal/model/` -- Changes to typed SQL payload fields require parser and diff updates.

## 7. Maintenance Notes
- `COM_STMT_EXECUTE` is defined but not captured. To support prepared statement tracking, maintain a mapping of statement IDs to their SQL text from `COM_STMT_PREPARE` responses.
- The `readMySQLPacketLength` helper function is unused in the main code path and could be removed or integrated.
- TCP lifecycle behavior lives in `tcp_proxy.go`; keep this file focused on MySQL packet parsing.
