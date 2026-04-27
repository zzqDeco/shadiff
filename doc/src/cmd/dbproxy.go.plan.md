# dbproxy.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: `cmd/dbproxy.go`
- Doc file: `doc/src/cmd/dbproxy.go.plan.md`
- File type: Go source
- Module: `shadiff/cmd`

## 2. Core Responsibility
- Provides helper logic for the `record` and `replay` commands' database proxy configuration and lifecycle.
- Converts CLI `--db-proxy` strings into structured `config.DBProxyConfig` values.
- Resolves whether DB proxy values come from CLI flags or runtime config.
- Starts and stops DB hook instances and forwards captured side effects into a shared sink channel.

## 3. Inputs & Outputs
- Inputs:
  - `flagChanged bool` and raw CLI values from `record --db-proxy` or `replay --db-proxy`
  - Runtime config `cfg.Capture.DBProxies` or `cfg.Replay.DBProxies`
  - `context.Context` used to manage DB hook lifetimes
  - `chan<- model.SideEffect` used as the sink for emitted side effects
- Outputs:
  - Parsed `[]config.DBProxyConfig`
  - A started `*dbhook.Group`
  - Side effects copied into the supplied sink channel

## 4. Key Implementation Details
- Main helpers:
  - `resolveRecordDBProxies(...)` implements precedence: if the flag changed, parse CLI values; otherwise clone config values.
  - `resolveReplayDBProxies(...)` implements precedence: if the flag changed, parse CLI values; otherwise clone `cfg.Replay.DBProxies`.
  - `parseDBProxySpec(spec string)` parses strings like `mysql://:13306->127.0.0.1:3306`.
  - `startDBHooks(...)` constructs each hook, starts it, and returns a `dbhook.Group` that owns side-effect fan-in plus flush/drain coordination into the provided sink channel.
  - `stopDBHooks(...)` best-effort stops any grouped hook owner that implements `Stop() error`.
- Testability:
  - `newDBHook` defaults to `dbhook.NewHook` and is replaceable in tests so failure cleanup paths can be exercised.
- Failure behavior:
  - If hook construction or startup fails, previously started hooks are stopped before the error is returned.
  - Group forwarders stop early if the controlling context is canceled, which prevents replay shutdown from blocking on sink backpressure.

## 5. Dependencies
- Internal:
  - `shadiff/internal/capture/dbhook`
  - `shadiff/internal/config`
  - `shadiff/internal/model`
- External:
  - Standard library `context`, `fmt`, `strings`

## 6. Change Impact
- The accepted DB proxy URI format is defined here; any CLI syntax change must update this parser and the user documentation.
- Hook startup semantics here must stay aligned with both capture and replay pipelines, each of which expects side effects to arrive on a single sink channel.
- Cleanup behavior matters for tests and for real startup failures, especially when multiple DB proxies are configured.

## 7. Maintenance Notes
- Keep `resolveRecordDBProxies(...)` as the single precedence resolver for DB proxies to avoid flag/config drift.
- Keep replay DB proxy precedence aligned with capture: explicit CLI flags win over config, and omitted flags use config values.
- If new DB types are added in `internal/capture/dbhook`, only `parseDBProxySpec(...)` syntax and downstream validation should need adjustment here.
- If recorder backpressure behavior changes, revisit the side-effect forwarding goroutines in `startDBHooks(...)`.
