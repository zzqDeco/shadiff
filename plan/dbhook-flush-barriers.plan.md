# DB Hook Flush Barriers Plan

## Goal

Add explicit DB-hook flush barriers so capture-time and replay-time side-effect attribution does not lose effects that were generated in-window but delivered slightly later through async forwarding.

## Scope

In scope:

- extend the `DBHook` contract with `Flush(context.Context) error`
- add a `dbhook.Group` that owns hook fan-in, flush coordination, and shutdown
- flush DB hooks from `capture.Proxy` before closing a request scope
- flush DB hooks from `replay.Engine` before taking the replay side-effect window
- add focused tests for hook-group flush, capture-time flush, replay-time flush, and timeout behavior

Out of scope:

- new DB types or protocol parsers
- new CLI flags or config schema
- changing the request-scoped attribution rule itself

## Approach

- Keep request/window attribution logic as-is, but add an explicit synchronization point before scopes or replay windows are finalized.
- Each DB hook tracks active sniffed connections and implements `Flush(ctx)` by:
  - injecting a per-connection barrier into the sniff loop
  - finishing protocol parsing for bytes already observed by the proxy
  - reading until a short idle window to capture near-simultaneous trailing bytes
- `dbhook.Group.Flush(ctx)` first flushes every hook and then waits for forwarders to drain captured side effects into the shared sink channel.
- `capture.Proxy` calls the flusher after the upstream response completes and before `Recorder.FinishRequestScope(...)`.
- `replay.Engine` calls the flusher after `replayOne(...)` and before `takeSideEffectsWindow(...)`.
- Flush failures stay non-fatal and are logged as warnings so record/replay behavior does not fail just because telemetry capture timed out.

## Tasks

- Update `internal/capture/dbhook/hook.go`:
  - add `DBHook.Flush(context.Context) error`
  - add `Group` and forwarder drain coordination
- Update MySQL/PostgreSQL/MongoDB hooks:
  - track active sniffed connections
  - implement per-hook `Flush(ctx)` and connection barrier helpers
- Update command wiring:
  - make `startDBHooks(...)` return a `*dbhook.Group`
  - wire the group into both `record` and `replay`
- Update capture/replay orchestration:
  - add flush hooks to `capture.ProxyOptions` and `replay.EngineConfig`
  - flush before closing request scopes or replay windows
- Add tests:
  - hook-group flush and timeout behavior
  - capture-time late side-effect attribution after proxy flush
  - replay-time late side-effect attribution after engine flush
- Sync docs after implementation:
  - `doc/src/internal/capture/dbhook/hook.go.plan.md`
  - `doc/src/internal/capture/dbhook/mysql.go.plan.md`
  - `doc/src/internal/capture/dbhook/postgres.go.plan.md`
  - `doc/src/internal/capture/dbhook/mongo.go.plan.md`
  - `doc/src/internal/capture/proxy.go.plan.md`
  - `doc/src/internal/replay/engine.go.plan.md`
  - `doc/src/cmd/dbproxy.go.plan.md`
  - `doc/src/cmd/record.go.plan.md`
  - `doc/src/cmd/replay.go.plan.md`

## Verification

- `go test ./cmd ./internal/capture ./internal/capture/dbhook ./internal/replay`
- `go test ./...`
- Manual checks:
  - capture with DB proxies still persists in-window late-delivered effects
  - replay with DB proxies still attributes in-window late-delivered effects
  - flush timeouts only emit warnings and do not fail record/replay
