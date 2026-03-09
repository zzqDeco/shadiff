# Test Hardening Plan

## Goal

Raise confidence in the command layer and DB hook layer by adding black-box and white-box tests around the highest-risk runtime paths.

## Scope

In scope:
- `cmd` tests for `record`, `record status`, `record stop`, replay/diff/report/session flows
- CLI black-box tests using a built binary
- DB hook forwarding tests that exercise `handleConn -> sniff -> parse -> emit`
- Small internal test seams needed to verify daemon branch selection without changing runtime behavior

Out of scope:
- Long-running daemon soak tests
- Real database integration environments
- Coverage-driven churn in already well-covered low-risk packages

## Approach

- Keep black-box tests focused on user-visible command behavior and generated artifacts.
- Keep white-box tests focused on branch coverage, cleanup behavior, and protocol parsing paths.
- Add only minimal seams:
  - replaceable DB hook factory in `cmd/dbproxy.go`
  - replaceable command execution/runtime branch functions in `cmd/record.go`

## Tasks

- Add `cmd/cli_blackbox_test.go` for version/session/report/diff command coverage.
- Add `cmd/commands_test.go` for session resolution and command execution helpers.
- Add `cmd/dbproxy_test.go` for DB proxy parsing and startup cleanup behavior.
- Expand `cmd/record_test.go` to cover config precedence, daemon parent bootstrap, and real record loop capture.
- Add `internal/capture/dbhook/proxy_test.go` for MySQL/Postgres/Mongo forwarding and lifecycle coverage.
- Extend package-level tests for logger, replay, diff, and config runtime behavior.

## Verification

- `go test ./...`
- `go test ./cmd ./internal/capture/dbhook -cover`
- Confirm `cmd` and `internal/capture/dbhook` package coverage materially improves after the changes.
