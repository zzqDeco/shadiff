# File-Level Documentation Index

## Mapping Rules

Each non-test Go source file maps to a file-level document under `doc/src/` that preserves the original relative path and appends `.plan.md` to the full filename.

Pattern:

```text
<source-path>.go -> doc/src/<source-path>.go.plan.md
```

Examples:

```text
main.go -> doc/src/main.go.plan.md
cmd/record.go -> doc/src/cmd/record.go.plan.md
internal/diff/engine.go -> doc/src/internal/diff/engine.go.plan.md
```

Exclusions:
- Test files (`*_test.go`)
- Vendor, build artifacts, logs, and `.git`

## Complete File Mapping

### Root

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `main.go` | `doc/src/main.go.plan.md` | Entry Point | root |

### examples/e2e/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `examples/e2e/api/main.go` | `doc/src/examples/e2e/api/main.go.plan.md` | Demo API | examples |
| `examples/e2e/api/Dockerfile` | - | Demo Tooling | examples |
| `examples/e2e/docker-compose.yml` | - | Demo Tooling | examples |
| `examples/e2e/init/mongo.js` | - | Demo Data Initialization | examples |
| `examples/e2e/init/mysql.sql` | - | Demo Data Initialization | examples |
| `examples/e2e/init/postgres.sql` | - | Demo Data Initialization | examples |
| `examples/e2e/run.sh` | - | Demo Tooling | examples |

### scripts/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `scripts/verify-release-assets.sh` | - | Release Tooling | scripts |

### .github/workflows/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `.github/workflows/ci.yml` | - | GitHub Actions Workflow | ci |
| `.github/workflows/integration.yml` | - | GitHub Actions Workflow | ci |
| `.github/workflows/release.yml` | - | GitHub Actions Workflow | ci |

### cmd/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `cmd/root.go` | `doc/src/cmd/root.go.plan.md` | CLI Command (Root) | cmd |
| `cmd/version.go` | `doc/src/cmd/version.go.plan.md` | CLI Command | cmd |
| `cmd/session.go` | `doc/src/cmd/session.go.plan.md` | CLI Command | cmd |
| `cmd/runtime.go` | `doc/src/cmd/runtime.go.plan.md` | Runtime Helper | cmd |
| `cmd/dbproxy.go` | `doc/src/cmd/dbproxy.go.plan.md` | CLI Helper | cmd |
| `cmd/record.go` | `doc/src/cmd/record.go.plan.md` | CLI Command | cmd |
| `cmd/record_stop.go` | `doc/src/cmd/record_stop.go.plan.md` | CLI Command | cmd |
| `cmd/record_status.go` | `doc/src/cmd/record_status.go.plan.md` | CLI Command | cmd |
| `cmd/replay.go` | `doc/src/cmd/replay.go.plan.md` | CLI Command | cmd |
| `cmd/diff.go` | `doc/src/cmd/diff.go.plan.md` | CLI Command | cmd |
| `cmd/report.go` | `doc/src/cmd/report.go.plan.md` | CLI Command | cmd |

### internal/daemon/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/daemon/pidfile.go` | `doc/src/internal/daemon/pidfile.go.plan.md` | Daemon Management | daemon |
| `internal/daemon/process_unix.go` | `doc/src/internal/daemon/process_unix.go.plan.md` | Daemon Management | daemon |
| `internal/daemon/process_windows.go` | `doc/src/internal/daemon/process_windows.go.plan.md` | Daemon Management | daemon |

### internal/model/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/model/session.go` | `doc/src/internal/model/session.go.plan.md` | Data Model | model |
| `internal/model/record.go` | `doc/src/internal/model/record.go.plan.md` | Data Model | model |
| `internal/model/request.go` | `doc/src/internal/model/request.go.plan.md` | Data Model | model |
| `internal/model/sideeffect.go` | `doc/src/internal/model/sideeffect.go.plan.md` | Data Model | model |
| `internal/model/diff.go` | `doc/src/internal/model/diff.go.plan.md` | Data Model | model |

### internal/config/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/config/config.go` | `doc/src/internal/config/config.go.plan.md` | Configuration Definition | config |
| `internal/config/store.go` | `doc/src/internal/config/store.go.plan.md` | Configuration Store | config |
| `internal/config/validate.go` | `doc/src/internal/config/validate.go.plan.md` | Configuration Validation | config |

### internal/capture/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/capture/proxy.go` | `doc/src/internal/capture/proxy.go.plan.md` | HTTP Reverse Proxy | capture |
| `internal/capture/recorder.go` | `doc/src/internal/capture/recorder.go.plan.md` | Request Recorder | capture |

### internal/capture/dbhook/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/capture/dbhook/hook.go` | `doc/src/internal/capture/dbhook/hook.go.plan.md` | Interface Definition | dbhook |
| `internal/capture/dbhook/mysql.go` | `doc/src/internal/capture/dbhook/mysql.go.plan.md` | DB Hook Implementation | dbhook |
| `internal/capture/dbhook/postgres.go` | `doc/src/internal/capture/dbhook/postgres.go.plan.md` | DB Hook Implementation | dbhook |
| `internal/capture/dbhook/mongo.go` | `doc/src/internal/capture/dbhook/mongo.go.plan.md` | DB Hook Implementation | dbhook |

### internal/storage/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/storage/store.go` | `doc/src/internal/storage/store.go.plan.md` | Interface Definition | storage |
| `internal/storage/filestore.go` | `doc/src/internal/storage/filestore.go.plan.md` | Storage Implementation | storage |

### internal/replay/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/replay/engine.go` | `doc/src/internal/replay/engine.go.plan.md` | Replay Engine | replay |
| `internal/replay/worker.go` | `doc/src/internal/replay/worker.go.plan.md` | Replay Worker | replay |
| `internal/replay/transform.go` | `doc/src/internal/replay/transform.go.plan.md` | Request Transform | replay |

### internal/diff/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/diff/engine.go` | `doc/src/internal/diff/engine.go.plan.md` | Diff Engine | diff |
| `internal/diff/json.go` | `doc/src/internal/diff/json.go.plan.md` | JSON Differ | diff |
| `internal/diff/db.go` | `doc/src/internal/diff/db.go.plan.md` | DB Diff | diff |
| `internal/diff/mongo.go` | `doc/src/internal/diff/mongo.go.plan.md` | MongoDB Diff | diff |
| `internal/diff/rules.go` | `doc/src/internal/diff/rules.go.plan.md` | Diff Rules | diff |
| `internal/diff/rule_loader.go` | `doc/src/internal/diff/rule_loader.go.plan.md` | Rule Loader | diff |

### internal/reporter/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/reporter/reporter.go` | `doc/src/internal/reporter/reporter.go.plan.md` | Interface Definition | reporter |
| `internal/reporter/terminal.go` | `doc/src/internal/reporter/terminal.go.plan.md` | Reporter Implementation | reporter |
| `internal/reporter/json.go` | `doc/src/internal/reporter/json.go.plan.md` | Reporter Implementation | reporter |
| `internal/reporter/html.go` | `doc/src/internal/reporter/html.go.plan.md` | Reporter Implementation | reporter |

### internal/logger/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/logger/logger.go` | `doc/src/internal/logger/logger.go.plan.md` | Logger | logger |

## Summary

| Module | Source Files | Doc Files |
|---|---|---|
| root | 1 | 1 |
| cmd | 11 | 11 |
| daemon | 3 | 3 |
| model | 5 | 5 |
| config | 3 | 3 |
| capture | 2 | 2 |
| dbhook | 4 | 4 |
| storage | 2 | 2 |
| replay | 3 | 3 |
| diff | 6 | 6 |
| reporter | 4 | 4 |
| logger | 1 | 1 |
| examples | 1 | 1 |
| **Total** | **46** | **46** |
