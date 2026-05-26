# Redis Side Effects

## Goal

Add Redis as a first-class database side-effect source across record, replay, diff, report, integration tests, and the official E2E demo.

## Scope

- `--db-proxy redis://<listen_addr>-><target_addr>` for record and replay.
- `capture.dbProxies` and `replay.dbProxies` accept `type: "redis"`.
- Redis plaintext TCP RESP command capture with command/key/args semantic diff.
- Docker integration coverage and official `examples/e2e` demo coverage.

Out of scope:
- Redis TLS termination.
- Redis Cluster routing awareness.
- Command-specific multi-key extraction beyond the first primary key.

## Approach

- Implement `RedisHook` using the existing `DBHook` lifecycle and `dbhook.Group` flush/drain barrier.
- Parse RESP array commands and inline commands from a per-connection stream buffer so fragmented reads and pipelined commands are handled.
- Store Redis side effects as `Type=database`, `DBType=redis`, `redisCommand`, `redisKey`, and `redisArgs`.
- Redact authentication-related Redis arguments before storing side effects.
- Add `CompareRedisSideEffects` and call it from the diff engine alongside SQL and MongoDB comparers.
- Extend integration and E2E tests with real Redis traffic.

## Tasks

- Add Redis model fields and difference kinds.
- Add Redis config validation and hook factory support.
- Add Redis hook, parser, unit tests, and proxy forwarding tests.
- Add Redis diff comparer and engine tests.
- Add Redis Docker integration tests and official E2E demo coverage.
- Sync README, README_CN, project docs, file index, and file-level docs.

## Verification

- `go test ./...`
- `go test -tags integration ./internal/integration -count=1 -timeout=20m`
- `bash -n examples/e2e/run.sh`
- `./examples/e2e/run.sh --assert`
- LAN Linux E2E with the release binary before publishing `v0.3.0`.
