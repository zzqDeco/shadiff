# redis.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/dbhook/redis.go
- Doc file: doc/src/internal/capture/dbhook/redis.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture/dbhook

## 2. Core Responsibility
- Implements a transparent TCP proxy for Redis plaintext traffic.
- Parses client-to-server RESP array commands and inline commands while forwarding traffic unchanged.
- Emits Redis command side effects for record/replay attribution and diffing.

## 3. Inputs & Outputs
- Input sources:
  - TCP connections from Redis clients connecting to the proxy listen address.
  - Raw Redis client command bytes using RESP or inline command format.
- Output results:
  - Transparent bidirectional TCP forwarding to the real Redis target.
  - `model.SideEffect` events with `DBType=redis`, `RedisCommand`, `RedisKey`, and `RedisArgs`.

## 4. Key Implementation Details
- `RedisHook` implements `DBHook` and mirrors the lifecycle used by the MySQL/PostgreSQL/MongoDB hooks.
- Each connection owns a `redisParser` buffer so fragmented reads and pipelined commands are parsed correctly.
- `Flush(ctx)` sends a barrier to active connection sniff loops and waits for traffic already on the wire to be parsed.
- The parser supports:
  - RESP arrays of bulk strings, the normal Redis client command format.
  - Inline commands such as `GET key`.
  - Multiple commands in one read.
  - Commands split across multiple reads.
- Redis arguments are stored as UTF-8 strings when valid; non-UTF-8 arguments are stored as `base64:<encoded>`.
- Sensitive arguments are redacted for `AUTH`, `HELLO ... AUTH ...`, `ACL SETUSER`, and `CONFIG SET requirepass/masterauth`.

## 5. Dependencies
- Internal:
  - `shadiff/internal/logger`
  - `shadiff/internal/model`
- External:
  - Standard library networking, buffering, base64, string, and synchronization packages.

## 6. Change Impact
- `hook.go` factory support must stay aligned with this file's `NewRedisHook` constructor.
- `internal/model.SideEffect` Redis fields and `internal/diff/redis.go` determine how captured Redis commands are persisted and compared.
- The parser intentionally does not terminate TLS or implement Redis Cluster routing semantics.

## 7. Maintenance Notes
- Keep transparent forwarding independent from parse success; malformed or unsupported command framing should not block traffic.
- If command-specific multi-key support is added later, extend `redisPrimaryKey` and diff tests together.
- Keep redaction rules conservative because Redis commands can carry credentials in command arguments.
