# Request-Scoped Side-Effect Attribution Plan

## Goal

Replace the current "attach pending side effects to the next record" behavior with deterministic request-scoped attribution so recorded side effects are assigned to the HTTP request that actually produced them.

## Scope

In scope:

- recorder-side request lifecycle tracking
- request-window side-effect attribution for capture-time DB hooks
- orphan side-effect handling and shutdown behavior
- recorder and proxy tests covering overlapping and delayed side effects

Out of scope:

- application-level tracing or correlation IDs
- post-response asynchronous work that finishes after the request window closes
- storage schema migrations for historical session data

## Approach

- Replace the single `pendingEffects` queue with explicit active request scopes in the recorder.
- A request scope opens when proxy handling begins and closes after the response has been captured and the record is ready to persist.
- Each side effect is attributed by timestamp to the active request scope whose start time is not later than the effect time and is the most recent such scope.
- If no active request scope matches, treat the effect as an orphan:
  - log a warning with enough context to diagnose the miss
  - do not attach the effect to the next unrelated request
- Make recorder shutdown wait for side-effect collection to finish draining before returning so pending attribution is deterministic.

## Tasks

- Refactor `capture.Recorder`:
  - add explicit request-scope lifecycle methods for begin/end
  - track active scopes and their attributed side effects
  - remove the next-record `pendingEffects` attachment model
  - make `Stop()` wait for collector shutdown and final drain
- Update `capture.Proxy`:
  - open a request scope before proxying
  - close the scope after response capture and persist the record with attributed side effects
  - keep excluded paths out of scope creation entirely
- Define attribution rules:
  - assign an effect to the active scope with the latest start time not after the effect timestamp
  - do not re-open closed scopes
  - discard orphan effects after logging instead of leaking them into future records
- Add tests:
  - sequential requests with in-window DB side effects
  - overlapping requests where later-started scope should win
  - delayed side effect after a request closes becomes orphaned
  - excluded paths do not create scopes and do not inherit unrelated effects
  - `Stop()` drains the collector without leaving undelivered effects
- Sync docs after implementation:
  - `doc/src/internal/capture/recorder.go.plan.md`
  - `doc/src/internal/capture/proxy.go.plan.md`
  - architecture and implementation docs describing the new attribution rule

## Verification

- `go test ./internal/capture ./internal/capture/dbhook`
- Manual checks:
  - a slow DB query is attached to the request that was still active when the side effect occurred
  - excluded paths no longer absorb side effects from surrounding traffic
  - recorder shutdown does not leave undrained side effects attached to later requests
