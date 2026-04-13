# Replay DB Side-Effects Plan

## Goal

Close the database side-effect loop across `record -> replay -> diff` so replayed traffic can capture MySQL, PostgreSQL, and MongoDB side effects and compare them semantically.

## Current Iteration

This iteration implements the review-finding subset:

- `replay --db-proxy` becomes a real capability driven by CLI flags
- replay with DB proxy capture is forced to `concurrency == 1`
- replay failures are still persisted into `replay-records.jsonl`
- diff integrates the existing SQL and MongoDB semantic comparers
- replay-side DB attribution is synchronized with an explicit DB-hook flush barrier before each replay window is finalized

Deferred follow-up items remain:

- config-driven replay DB proxy settings
- broader replay-time DB proxy ergonomics beyond the existing CLI flag

## Scope

In scope:

- replay-time DB proxy startup from CLI flags
- replay-time DB side-effect collection and request-level attribution
- diff-engine integration with SQL and MongoDB side-effect comparers
- CLI validation and behavior for replay DB proxies
- focused tests for replay, diff, and config behavior

Out of scope:

- new database types beyond mysql, postgres, and mongo
- external HTTP side-effect capture or comparison
- daemon or UI protocol changes

## Approach

- Reuse the existing `DBProxyConfig` shape from CLI parsing so replay can start the same DB hooks already used during record.
- Treat replay-time DB capture as a single-request attribution problem: a side effect belongs to the replayed request whose execution window contains the effect timestamp.
- Enforce serial replay when replay DB proxies are enabled. If DB proxies are configured and effective concurrency is greater than `1`, fail fast with a clear error instead of attempting ambiguous attribution.
- Extend the diff engine to compare side effects by type:
  - SQL side effects through `CompareDBSideEffects`
  - MongoDB side effects through `CompareMongoSideEffects`
  - retain total side-effect count comparison as a coarse fallback signal
- Preserve existing JSONL storage. Replayed records continue to be stored in `replay-records.jsonl`, now with populated `SideEffects` when replay DB capture is enabled.

## Tasks

- Update `cmd/replay.go`:
  - resolve replay DB proxies from CLI flags
  - reject `concurrency > 1` when replay DB proxies are enabled
  - start and stop replay DB hooks around engine execution
- Extend the replay engine and worker flow:
  - capture request start and end timestamps for each replayed request
  - collect DB side effects emitted during that window
  - attach matched side effects to the replayed `model.Record`
- Update diff behavior:
  - compare SQL side effects using the existing SQL comparer
  - compare MongoDB side effects using the existing Mongo comparer
  - keep a total count difference only for unmatched residual cases
- Add tests:
  - replay CLI flag handling for DB proxies
  - replay command rejection when DB proxies are enabled with concurrent replay
  - replay engine attribution of side effects into replay records
  - diff engine integration tests for matching SQL, differing SQL, matching Mongo, and differing Mongo operations
- Sync docs after implementation:
  - `doc/src/cmd/replay.go.plan.md`
  - `doc/src/internal/replay/engine.go.plan.md`
  - `doc/src/internal/diff/engine.go.plan.md`
  - project docs and README files if user-facing replay behavior changes

## Verification

- `go test ./cmd ./internal/replay ./internal/diff`
- Manual checks:
  - replay with `--db-proxy mysql://...` stores DB side effects in `replay-records.jsonl`
  - replay with `--db-proxy ... -c 2` returns a clear validation error
  - diff reports SQL and MongoDB semantic differences instead of only a count mismatch
