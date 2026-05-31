# Shadiff Development Plan Index

## Status Legend

| Mark | Meaning |
|------|---------|
| Pending | Not yet started |
| In Progress | Currently being implemented |
| Completed | Done and verified |

## Active Plan Documents

| Document | Description | Status |
|----------|-------------|--------|
| `plan/capture-stream-request-bodies.plan.md` | Stream request body capture and avoid eager buffering for skipped or large requests | Completed |
| `plan/request-body-artifacts.plan.md` | Persist full request-body artifacts so replay can reconstruct large or partially-consumed requests faithfully | Completed |
| `plan/dbhook-flush-barriers.plan.md` | Add explicit DB-hook flush barriers so capture/replay attribution does not lose late-delivered side effects | Completed |
| `plan/dbhook-flush-drain-race.plan.md` | Fix DB hook group flush drain synchronization so forwarded side effects reach the shared sink before flush returns | Completed |
| `plan/replay-db-sideeffects.plan.md` | Close the DB side-effect replay and diff loop for MySQL, PostgreSQL, and MongoDB | Completed |
| `plan/request-scoped-sideeffect-attribution.plan.md` | Replace next-record side-effect attachment with request-scoped attribution | Completed |
| `plan/diff-ci-output.plan.md` | Make `shadiff diff` produce stable machine-readable output and CI-friendly exit codes | Completed |
| `plan/ci-release-automation.plan.md` | Add GitHub Actions CI checks and cross-platform release asset automation | Completed |
| `plan/v0.1.1-stability.plan.md` | Harden release automation and add Docker-backed DB side-effect integration coverage | Completed |
| `plan/v0.2.0-e2e-usability.plan.md` | Add an official reproducible E2E demo across HTTP and DB side effects | Completed |
| `plan/redis-sideeffects.plan.md` | Add Redis DB proxy capture, replay attribution, semantic diff, integration tests, and E2E demo coverage | Completed |
| `plan/sideeffect-architecture-refactor.plan.md` | Refactor side-effect payloads, DB type registry, diff comparer dispatch, and DB hook TCP lifecycle | Completed |
| `plan/v0.4-sideeffect-regression.plan.md` | Harden v0.4 typed side-effect JSON, diff registry, reporter output, and E2E assertions | Completed |

## Phase History

| Phase | Description | Status |
|-------|-------------|--------|
| Phase 1 | Foundation (model/config/logger/storage/CLI) | Completed |
| Phase 2 | Core Record & Replay (capture/replay) | Completed |
| Phase 3 | Diff Engine (diff) | Completed |
| Phase 4 | Reporting & DB Hooks (reporter/dbhook) | Completed |
| Phase 5 | Config Runtime Integration | Completed |
| Phase 6 | Test Hardening | Completed |
