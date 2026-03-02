# mysql.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/dbhook/mysql.go
- Doc file: doc/src/internal/capture/dbhook/mysql.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture/dbhook

## 2. Core Responsibility
- Implements a TCP proxy for MySQL that transparently forwards traffic between a client and a real MySQL server while sniffing the client-to-server stream to extract SQL statements from the MySQL wire protocol.
- Emits captured SQL queries as `model.SideEffect` events on a buffered channel for consumption by the `Recorder`.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - TCP connections from MySQL clients connecting to `listenAddr`.
  - Raw MySQL wire protocol packets on the client-to-server data stream.
- Output results:
  - Proxied TCP traffic forwarded transparently to `targetAddr` (the real MySQL server) and back to the client.
  - `model.SideEffect` events (type `SideEffectDB`, dbType `mysql`) emitted on the `sideEffects` channel containing captured SQL query strings.
  - Log events via `logger.DBHookEvent`, `logger.Error`, and `logger.Warn`.

## 4. Key Implementation Details
- Structs/interfaces:
  - `MySQLHook` -- Implements `DBHook`. Holds listen/target addresses, a `net.Listener`, a buffered side-effects channel (capacity 1000), a `done` channel for shutdown, and a `sync.WaitGroup` for goroutine lifecycle management.
- Exported functions/methods:
  - `NewMySQLHook(listenAddr, targetAddr string) *MySQLHook` -- Constructor.
  - `(*MySQLHook).Type() string` -- Returns `"mysql"`.
  - `(*MySQLHook).SideEffects() <-chan model.SideEffect` -- Returns the read-only side-effects channel.
  - `(*MySQLHook).Start(ctx context.Context) error` -- Opens a TCP listener and spawns an accept loop goroutine.
  - `(*MySQLHook).Stop() error` -- Signals shutdown, closes the listener, waits for all goroutines, and closes the side-effects channel.
- Protocol constants:
  - `mysqlComQuery` (0x03) -- COM_QUERY command byte.
  - `mysqlComStmtPrepare` (0x16) -- COM_STMT_PREPARE command byte.
  - `mysqlComStmtExecute` (0x17) -- COM_STMT_EXECUTE command byte (defined but not actively captured).
- Key behaviors:
  - Each accepted client connection spawns a goroutine that dials the real MySQL server, then runs two concurrent goroutines: one for server-to-client passthrough (`io.Copy`) and one for client-to-server sniffing.
  - `sniffClientToServer` reads raw bytes into a 64KB buffer, forwards them to the server, then attempts to parse MySQL packets from the same buffer.
  - `parseMySQLPacket` extracts the 3-byte little-endian payload length, 1-byte sequence number, and 1-byte command byte. For `COM_QUERY` and `COM_STMT_PREPARE`, the payload is interpreted as a SQL string and emitted as a side effect.
  - `COM_STMT_EXECUTE` is defined as a constant but not actively captured because the execute payload contains binary parameter data, not a readable SQL string.
  - `emitSideEffect` sends on the channel non-blockingly; if the channel is full, the event is dropped with a warning log.
  - The helper function `readMySQLPacketLength` exists but is not used by the main sniffing path (uses inline parsing instead).

## 5. Dependencies
- Internal:
  - `shadiff/internal/logger` -- Logging for lifecycle events, errors, and warnings.
  - `shadiff/internal/model` -- `SideEffect` type and `SideEffectDB` constant.
- External:
  - `context` -- `Start` method signature (context not actively used for cancellation in this implementation).
  - `encoding/binary` -- `readMySQLPacketLength` helper uses `binary.LittleEndian`.
  - `io` -- `io.Copy` for server-to-client passthrough.
  - `net` -- TCP listener and dialer.
  - `sync` -- `WaitGroup` for goroutine coordination.
  - `time` -- Dial timeout (10 seconds) and timestamps on side effects.

## 6. Change Impact
- `internal/capture/dbhook/hook.go` -- `NewHook` factory directly calls `NewMySQLHook`; constructor signature changes require a corresponding update.
- `internal/capture/recorder.go` -- Consumes the `SideEffects()` channel; changes to the channel protocol or `SideEffect` field usage affect the recorder.
- `internal/model/` -- Changes to `SideEffect` fields (especially `Type`, `DBType`, `Query`, `Timestamp`) require updates in `emitSideEffect`.

## 7. Maintenance Notes
- The packet parser assumes each `Read` call returns a complete MySQL packet. In practice, TCP reads may return partial packets or multiple packets. For production robustness, implement a framing layer that buffers and reassembles packets based on the 3-byte length header.
- `COM_STMT_EXECUTE` is defined but not captured. To support prepared statement tracking, maintain a mapping of statement IDs to their SQL text from `COM_STMT_PREPARE` responses.
- The `ctx` parameter in `Start` is accepted but not wired into the accept loop or connection handling. Consider using it for graceful cancellation.
- The `readMySQLPacketLength` helper function is unused in the main code path and could be removed or integrated.
- The 10-second dial timeout is hardcoded; consider making it configurable.
