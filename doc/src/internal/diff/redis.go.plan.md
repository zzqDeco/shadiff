# redis.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/diff/redis.go
- Doc file: doc/src/internal/diff/redis.go.plan.md
- File type: Go source
- Module: shadiff/internal/diff

## 2. Core Responsibility
- Compares Redis side effects between recorded and replayed traffic.
- Reports semantic differences in Redis command count, command name, primary key, and arguments.

## 3. Inputs & Outputs
- Input sources: Two `[]model.SideEffect` slices containing recorded and replayed side effects.
- Output results: `[]model.Difference` entries with `redis_command_count` or `redis_command` kinds.

## 4. Key Implementation Details
- `CompareRedisSideEffects(original, replay)` filters both inputs to `Type == SideEffectDB`, `database.type == "redis"`, and a non-nil `database.redis` payload.
- Commands are paired positionally by index.
- Count, command, key, and argument differences are all `SeverityError`.
- `filterRedisEffects(effects)` isolates Redis side effects so SQL, MongoDB, and HTTP side effects are ignored by this comparer.

## 5. Dependencies
- Internal: `shadiff/internal/dbtype`, `shadiff/internal/model`
- External: Standard library `fmt` and `slices`.

## 6. Change Impact
- `sideeffects.go` registers this comparer for engine-side record/replay comparison.
- New `model.RedisSideEffect` fields must stay aligned with this comparer.
- Difference kind names are part of JSON output consumed by scripts and E2E assertions.

## 7. Maintenance Notes
- The comparer is intentionally strict: command, key, and arguments must match exactly after capture-time redaction/encoding.
- If an unordered or command-aware Redis diff mode is added later, keep this strict mode as the default for CI stability.
