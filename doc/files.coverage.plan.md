# Documentation Coverage Statistics

## Overall Statistics

| Metric | Count |
|---|---|
| Total source files (non-test `.go`) | 46 |
| Test files (`*_test.go`) | 29 |
| Project tooling files | 10 |
| Project-level docs (`doc/*.md`) | 6 |
| File-level docs (`doc/src/**/*.plan.md`) | 46 |
| Project-level coverage | 100% |
| File-level coverage | 100% |

## Per-Module Statistics

| Module | Package Path | Source Files | Test Files | File-Level Docs | Coverage |
|---|---|---|---|---|---|
| root | `.` | 1 | 0 | 1 | 100% |
| cmd | `cmd/` | 11 | 5 | 11 | 100% |
| daemon | `internal/daemon/` | 3 | 1 | 3 | 100% |
| model | `internal/model/` | 5 | 1 | 5 | 100% |
| config | `internal/config/` | 3 | 2 | 3 | 100% |
| capture | `internal/capture/` | 2 | 2 | 2 | 100% |
| dbhook | `internal/capture/dbhook/` | 4 | 4 | 4 | 100% |
| storage | `internal/storage/` | 2 | 1 | 2 | 100% |
| replay | `internal/replay/` | 3 | 3 | 3 | 100% |
| diff | `internal/diff/` | 6 | 6 | 6 | 100% |
| reporter | `internal/reporter/` | 4 | 1 | 4 | 100% |
| logger | `internal/logger/` | 1 | 2 | 1 | 100% |
| integration | `internal/integration/` | 0 | 1 | 0 | n/a |
| examples | `examples/e2e/api/` | 1 | 0 | 1 | 100% |
| **Total** | | **46** | **29** | **46** | **100%** |

## File Types Breakdown

| File Type | Count | Files |
|---|---|---|
| Entry Point | 1 | `main.go` |
| Release Tooling | 1 | `scripts/verify-release-assets.sh` |
| Demo Tooling | 3 | `examples/e2e/api/Dockerfile`, `examples/e2e/docker-compose.yml`, `examples/e2e/run.sh` |
| Demo Data Initialization | 3 | `examples/e2e/init/mongo.js`, `examples/e2e/init/mysql.sql`, `examples/e2e/init/postgres.sql` |
| GitHub Actions Workflows | 3 | `.github/workflows/ci.yml`, `.github/workflows/integration.yml`, `.github/workflows/release.yml` |
| Demo API | 1 | `examples/e2e/api/main.go` |
| CLI Command / Helper | 11 | `cmd/root.go`, `cmd/version.go`, `cmd/session.go`, `cmd/runtime.go`, `cmd/dbproxy.go`, `cmd/record.go`, `cmd/record_stop.go`, `cmd/record_status.go`, `cmd/replay.go`, `cmd/diff.go`, `cmd/report.go` |
| Data Model | 5 | `internal/model/session.go`, `internal/model/record.go`, `internal/model/request.go`, `internal/model/sideeffect.go`, `internal/model/diff.go` |
| Configuration | 3 | `internal/config/config.go`, `internal/config/store.go`, `internal/config/validate.go` |
| Interface Definition | 3 | `internal/capture/dbhook/hook.go`, `internal/storage/store.go`, `internal/reporter/reporter.go` |
| Daemon Management | 3 | `internal/daemon/pidfile.go`, `internal/daemon/process_unix.go`, `internal/daemon/process_windows.go` |
| Capture / Diff / Replay / Storage Implementation | 18 | `internal/capture/proxy.go`, `internal/capture/recorder.go`, `internal/capture/dbhook/mysql.go`, `internal/capture/dbhook/postgres.go`, `internal/capture/dbhook/mongo.go`, `internal/storage/filestore.go`, `internal/replay/engine.go`, `internal/replay/worker.go`, `internal/replay/transform.go`, `internal/diff/engine.go`, `internal/diff/json.go`, `internal/diff/db.go`, `internal/diff/mongo.go`, `internal/diff/rules.go`, `internal/diff/rule_loader.go`, `internal/reporter/terminal.go`, `internal/reporter/json.go`, `internal/reporter/html.go` |
| Logger | 1 | `internal/logger/logger.go` |
| **Total** | **46** | |

## Excluded Directories

The following directories and file patterns are excluded from the documentation index:

| Exclusion | Reason |
|---|---|
| `vendor/` | Third-party dependencies |
| `logs/` | Runtime log output |
| `build/` | Build artifacts |
| `.git/` | Version control metadata |
| `*_test.go` | Test files |

## Update Log

| Event | Date |
|---|---|
| Project-level docs created | 2026-03-02 |
| File-level docs created | 2026-03-02 |
| Daemon support docs added | 2026-03-04 |
| Config runtime docs synchronized | 2026-03-09 |
| Test inventory and file coverage refreshed | 2026-03-09 |
| Official E2E demo inventory added | 2026-05-26 |
| Last updated | 2026-05-26 |
