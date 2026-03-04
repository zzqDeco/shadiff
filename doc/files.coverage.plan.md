# Documentation Coverage Statistics

## Overall Statistics

| Metric | Count |
|---|---|
| Total source files (non-test `.go`) | 41 |
| Test files (`*_test.go`) | 11 |
| Project-level docs (`doc/*.md`) | 6 |
| File-level docs (`doc/src/**/*.plan.md`) | 41 |
| Project-level coverage | 100% |
| File-level coverage | 100% |

---

## Per-Module Statistics

| Module | Package Path | Source Files | Test Files | File-Level Docs | Coverage |
|---|---|---|---|---|---|
| root | `.` | 1 | 0 | 1 | 100% |
| cmd | `cmd/` | 9 | 0 | 9 | 100% |
| daemon | `internal/daemon/` | 3 | 1 | 3 | 100% |
| model | `internal/model/` | 5 | 1 | 5 | 100% |
| config | `internal/config/` | 2 | 1 | 2 | 100% |
| capture | `internal/capture/` | 2 | 1 | 2 | 100% |
| dbhook | `internal/capture/dbhook/` | 4 | 2 | 4 | 100% |
| storage | `internal/storage/` | 2 | 1 | 2 | 100% |
| replay | `internal/replay/` | 3 | 1 | 3 | 100% |
| diff | `internal/diff/` | 5 | 4 | 5 | 100% |
| reporter | `internal/reporter/` | 4 | 1 | 4 | 100% |
| logger | `internal/logger/` | 1 | 0 | 1 | 100% |
| **Total** | | **41** | **13** | **41** | **100%** |

---

## File Types Breakdown

| File Type | Count | Files |
|---|---|---|
| Entry Point | 1 | `main.go` |
| CLI Command | 9 | `cmd/root.go`, `cmd/version.go`, `cmd/session.go`, `cmd/record.go`, `cmd/record_stop.go`, `cmd/record_status.go`, `cmd/replay.go`, `cmd/diff.go`, `cmd/report.go` |
| Data Model | 5 | `internal/model/session.go`, `internal/model/record.go`, `internal/model/request.go`, `internal/model/sideeffect.go`, `internal/model/diff.go` |
| Configuration | 2 | `internal/config/config.go`, `internal/config/store.go` |
| Interface Definition | 3 | `internal/capture/dbhook/hook.go`, `internal/storage/store.go`, `internal/reporter/reporter.go` |
| Daemon Management | 3 | `internal/daemon/pidfile.go`, `internal/daemon/process_unix.go`, `internal/daemon/process_windows.go` |
| Implementation | 14 | `internal/capture/proxy.go`, `internal/capture/recorder.go`, `internal/capture/dbhook/mysql.go`, `internal/capture/dbhook/postgres.go`, `internal/capture/dbhook/mongo.go`, `internal/storage/filestore.go`, `internal/replay/engine.go`, `internal/replay/worker.go`, `internal/replay/transform.go`, `internal/diff/engine.go`, `internal/diff/json.go`, `internal/diff/db.go`, `internal/diff/mongo.go`, `internal/diff/rules.go` |
| Reporter Implementation | 3 | `internal/reporter/terminal.go`, `internal/reporter/json.go`, `internal/reporter/html.go` |
| Logger | 1 | `internal/logger/logger.go` |
| **Total** | **41** | |

---

## Excluded Directories

The following directories and file patterns are excluded from the documentation index:

| Exclusion | Reason |
|---|---|
| `vendor/` | Third-party dependencies |
| `logs/` | Runtime log output |
| `build/` | Build artifacts |
| `.git/` | Version control metadata |
| `*_test.go` | Test files (documented separately if needed) |

---

## Update Log

| Event | Date |
|---|---|
| Project-level docs created | 2026-03-02 |
| File-level docs created | 2026-03-02 |
| Coverage reached 100% | 2026-03-02 |
| Daemon support docs added | 2026-03-04 |
| Last updated | 2026-03-04 |
