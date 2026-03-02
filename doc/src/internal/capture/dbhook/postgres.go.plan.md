# postgres.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/dbhook/postgres.go
- Doc file: doc/src/internal/capture/dbhook/postgres.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture/dbhook

## 2. Core Responsibility
- Implements a TCP proxy for PostgreSQL that transparently forwards traffic between a client and a real PostgreSQL server while sniffing the client-to-server stream to extract SQL statements from PostgreSQL frontend messages.
- Emits captured SQL queries as `model.SideEffect` events on a buffered channel for consumption by the `Recorder`.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - TCP connections from PostgreSQL clients connecting to `listenAddr`.
  - Raw PostgreSQL wire protocol frontend messages on the client-to-server stream.
- Output results:
  - Proxied TCP traffic forwarded transparently to `targetAddr` (the real PostgreSQL server) and back to the client.
  - `model.SideEffect` events (type `SideEffectDB`, dbType `postgres`) emitted on the `sideEffects` channel containing captured SQL query strings.
  - Log events via `logger.DBHookEvent`, `logger.Error`, and `logger.Warn`.

## 4. Key Implementation Details
- Structs/interfaces:
  - `PostgresHook` -- Implements `DBHook`. Holds listen/target addresses, a `net.Listener`, a buffered side-effects channel (capacity 1000), a `done` channel for shutdown, and a `sync.WaitGroup` for goroutine lifecycle management.
- Exported functions/methods:
  - `NewPostgresHook(listenAddr, targetAddr string) *PostgresHook` -- Constructor.
  - `(*PostgresHook).Type() string` -- Returns `"postgres"`.
  - `(*PostgresHook).SideEffects() <-chan model.SideEffect` -- Returns the read-only side-effects channel.
  - `(*PostgresHook).Start(ctx context.Context) error` -- Opens a TCP listener and spawns an accept loop goroutine.
  - `(*PostgresHook).Stop() error` -- Signals shutdown, closes the listener, waits for all goroutines, and closes the side-effects channel.
- Unexported helpers:
  - `extractNullTermString(data []byte) string` -- Extracts a C-style null-terminated string from a byte slice.
  - `nullTermIndex(data []byte) int` -- Returns the index of the first null byte, or -1 if not found.
- Protocol constants:
  - `pgMsgQuery` ('Q') -- Simple Query message type.
  - `pgMsgParse` ('P') -- Extended Query Parse message type.
- Key behaviors:
  - Connection handling follows the same pattern as the MySQL hook: each client gets a dedicated goroutine pair for bidirectional proxying.
  - `sniffClientToServer` includes a `startup` flag to skip the initial PostgreSQL startup message, which has a different format (no message type byte, just 4-byte length + 4-byte protocol version). After the first message of 8+ bytes, the flag is cleared and subsequent messages are parsed as standard frontend messages.
  - `parsePGMessage` iterates through potentially multiple messages in a single read buffer. Each message has a 1-byte type, 4-byte big-endian length (inclusive of the length field itself), and a variable-length payload.
  - For Simple Query ('Q'), the payload is a null-terminated SQL string.
  - For Parse ('P'), the payload contains a null-terminated statement name followed by a null-terminated query string. The parser skips the statement name to extract the query.
  - Side effects are emitted non-blockingly; full channels cause dropped events with a warning.

## 5. Dependencies
- Internal:
  - `shadiff/internal/logger` -- Logging for lifecycle events, errors, and warnings.
  - `shadiff/internal/model` -- `SideEffect` type and `SideEffectDB` constant.
- External:
  - `context` -- `Start` method signature (context not actively used for cancellation).
  - `encoding/binary` -- `binary.BigEndian.Uint32` for parsing message lengths.
  - `io` -- `io.Copy` for server-to-client passthrough.
  - `net` -- TCP listener and dialer.
  - `sync` -- `WaitGroup` for goroutine coordination.
  - `time` -- Dial timeout (10 seconds) and timestamps on side effects.

## 6. Change Impact
- `internal/capture/dbhook/hook.go` -- `NewHook` factory directly calls `NewPostgresHook`; constructor signature changes require a corresponding update.
- `internal/capture/recorder.go` -- Consumes the `SideEffects()` channel; changes to the channel protocol or `SideEffect` field usage affect the recorder.
- `internal/model/` -- Changes to `SideEffect` fields (especially `Type`, `DBType`, `Query`, `Timestamp`) require updates in `emitSideEffect`.

## 7. Maintenance Notes
- The startup phase detection is simplified: it assumes the first message of 8+ bytes is the startup message and then switches to normal message parsing. This does not handle SSL negotiation requests (`SSLRequest` message) which may precede the startup message. For production use, detect the SSLRequest (protocol version `80877103`) and handle the SSL handshake or rejection before proceeding.
- Like the MySQL hook, the parser assumes each `Read` call returns complete messages. TCP fragmentation can cause partial reads. A proper framing layer with buffered reads based on the message length field would improve robustness.
- The `ctx` parameter in `Start` is accepted but not wired into the accept loop. Consider using it for context-based cancellation.
- The parser only captures Simple Query and Parse messages. Other Extended Query protocol messages (Bind, Execute, Describe) are not captured. If full prepared statement tracking is needed, these should be handled as well.
- `extractNullTermString` and `nullTermIndex` are package-level unexported functions that could be reused by other hooks if needed.
