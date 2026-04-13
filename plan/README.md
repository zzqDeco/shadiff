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
| `plan/capture-stream-request-bodies.plan.md` | Stream request body capture and avoid eager buffering for skipped or large requests | In Progress |
| `plan/replay-db-sideeffects.plan.md` | Close the DB side-effect replay and diff loop for MySQL, PostgreSQL, and MongoDB | Pending |
| `plan/request-scoped-sideeffect-attribution.plan.md` | Replace next-record side-effect attachment with request-scoped attribution | Pending |
| `plan/diff-ci-output.plan.md` | Make `shadiff diff` produce stable machine-readable output and CI-friendly exit codes | In Progress |

## Phase History

| Phase | Description | Status |
|-------|-------------|--------|
| Phase 1 | Foundation (model/config/logger/storage/CLI) | Completed |
| Phase 2 | Core Record & Replay (capture/replay) | Completed |
| Phase 3 | Diff Engine (diff) | Completed |
| Phase 4 | Reporting & DB Hooks (reporter/dbhook) | Completed |
| Phase 5 | Config Runtime Integration | Completed |
| Phase 6 | Test Hardening | Completed |
