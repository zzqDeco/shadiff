# dbproxy.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: `cmd/dbproxy.go`
- Doc file: `doc/src/cmd/dbproxy.go.plan.md`
- File type: Go source
- Module: `shadiff/cmd`

## 2. Core Responsibility
- Provides helper logic for the `record` command's database proxy configuration and lifecycle.
- Converts CLI `--db-proxy` strings into structured `config.DBProxyConfig` values.
- Resolves whether DB proxy values come from CLI flags or runtime config.
- Starts and stops DB hook instances and forwards captured side effects into the shared recorder.

## 3. Inputs & Outputs
- Inputs:
  - `flagChanged bool` and raw CLI values from `record --db-proxy`
  - Runtime config `cfg.Capture.DBProxies`
  - `context.Context` used to manage DB hook lifetimes
  - `capture.Recorder` used as the sink for emitted side effects
- Outputs:
  - Parsed `[]config.DBProxyConfig`
  - Started `[]dbhook.DBHook`
  - Side effects copied into `recorder.SideEffectChan()`

## 4. Key Implementation Details
- Main helpers:
  - `resolveRecordDBProxies(...)` implements precedence: if the flag changed, parse CLI values; otherwise clone config values.
  - `parseDBProxySpec(spec string)` parses strings like `mysql://:13306->127.0.0.1:3306`.
  - `startDBHooks(...)` constructs each hook, starts it, and launches one goroutine per hook to fan side effects into the recorder.
  - `stopDBHooks(...)` best-effort stops all hooks.
- Testability:
  - `newDBHook` defaults to `dbhook.NewHook` and is replaceable in tests so failure cleanup paths can be exercised.
- Failure behavior:
  - If hook construction or startup fails, previously started hooks are stopped before the error is returned.

## 5. Dependencies
- Internal:
  - `shadiff/internal/capture`
  - `shadiff/internal/capture/dbhook`
  - `shadiff/internal/config`
- External:
  - Standard library `context`, `fmt`, `strings`

## 6. Change Impact
- The accepted DB proxy URI format is defined here; any CLI syntax change must update this parser and the user documentation.
- Hook startup semantics here must stay aligned with the capture pipeline's assumption that side effects arrive on a single recorder channel.
- Cleanup behavior matters for tests and for real startup failures, especially when multiple DB proxies are configured.

## 7. Maintenance Notes
- Keep `resolveRecordDBProxies(...)` as the single precedence resolver for DB proxies to avoid flag/config drift.
- If new DB types are added in `internal/capture/dbhook`, only `parseDBProxySpec(...)` syntax and downstream validation should need adjustment here.
- If recorder backpressure behavior changes, revisit the side-effect forwarding goroutines in `startDBHooks(...)`.
