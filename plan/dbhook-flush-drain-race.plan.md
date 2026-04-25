# DB Hook Flush Drain Race Plan

## Goal

Fix the DB hook group flush drain race so `Group.Flush()` waits until hook-emitted side effects have actually reached the shared sink channel.

## Scope

In scope:

- tighten `sideEffectForwarder.WaitDrained()` synchronization
- keep the existing `DBHook` interface unchanged
- add a regression test for side effects consumed from the hook channel but not yet delivered to the sink
- update DB hook technical documentation

Out of scope:

- new database protocol behavior
- new CLI flags or config schema
- changes to record/replay attribution windows

## Approach

- Add a barrier channel to each side-effect forwarder.
- Have the forwarder process source events, barrier acknowledgements, and context cancellation in the same goroutine.
- Make `WaitDrained(ctx)` send a barrier and wait for its acknowledgement before declaring the forwarder drained.
- Preserve timeout behavior when sink delivery is blocked.

## Tasks

- Update `internal/capture/dbhook/hook.go` with barrier-based drain synchronization.
- Add regression coverage in `internal/capture/dbhook/hook_test.go`.
- Update `doc/src/internal/capture/dbhook/hook.go.plan.md`.
- Verify with repeated DB hook flush tests and `go test ./...`.

## Verification

- `go test ./internal/capture/dbhook -run 'Test.*Flush|Test.*Forwarder' -count=50`
- `go test ./...`
