# File-Level Documentation Index

## Mapping Rules

Each Go source file (excluding test files) maps to a corresponding documentation file under `doc/src/` following the same relative path, with the `.go` extension replaced by `.plan.md`.

**Pattern:**

```
<source-path>.go  -->  doc/src/<source-path>.plan.md
```

**Example:**

```
internal/model/session.go  -->  doc/src/internal/model/session.plan.md
```

**Exclusions:**
- Test files (`*_test.go`) are not documented in this index.
- Vendor, build artifacts, logs, and `.git` directories are excluded.

---

## Complete File Mapping

### Root

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `main.go` | `doc/src/main.plan.md` | Entry Point | root |

### cmd/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `cmd/root.go` | `doc/src/cmd/root.plan.md` | CLI Command (Root) | cmd |
| `cmd/version.go` | `doc/src/cmd/version.plan.md` | CLI Command | cmd |
| `cmd/session.go` | `doc/src/cmd/session.plan.md` | CLI Command | cmd |
| `cmd/record.go` | `doc/src/cmd/record.plan.md` | CLI Command | cmd |
| `cmd/replay.go` | `doc/src/cmd/replay.plan.md` | CLI Command | cmd |
| `cmd/diff.go` | `doc/src/cmd/diff.plan.md` | CLI Command | cmd |
| `cmd/report.go` | `doc/src/cmd/report.plan.md` | CLI Command | cmd |

### internal/model/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/model/session.go` | `doc/src/internal/model/session.plan.md` | Data Model | model |
| `internal/model/record.go` | `doc/src/internal/model/record.plan.md` | Data Model | model |
| `internal/model/request.go` | `doc/src/internal/model/request.plan.md` | Data Model | model |
| `internal/model/sideeffect.go` | `doc/src/internal/model/sideeffect.plan.md` | Data Model | model |
| `internal/model/diff.go` | `doc/src/internal/model/diff.plan.md` | Data Model | model |

### internal/config/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/config/config.go` | `doc/src/internal/config/config.plan.md` | Configuration Definition | config |
| `internal/config/store.go` | `doc/src/internal/config/store.plan.md` | Configuration Store | config |

### internal/capture/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/capture/proxy.go` | `doc/src/internal/capture/proxy.plan.md` | HTTP Reverse Proxy | capture |
| `internal/capture/recorder.go` | `doc/src/internal/capture/recorder.plan.md` | Request Recorder | capture |

### internal/capture/dbhook/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/capture/dbhook/hook.go` | `doc/src/internal/capture/dbhook/hook.plan.md` | Interface Definition | dbhook |
| `internal/capture/dbhook/mysql.go` | `doc/src/internal/capture/dbhook/mysql.plan.md` | DB Hook Implementation | dbhook |
| `internal/capture/dbhook/postgres.go` | `doc/src/internal/capture/dbhook/postgres.plan.md` | DB Hook Implementation | dbhook |
| `internal/capture/dbhook/mongo.go` | `doc/src/internal/capture/dbhook/mongo.plan.md` | DB Hook Implementation | dbhook |

### internal/storage/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/storage/store.go` | `doc/src/internal/storage/store.plan.md` | Interface Definition | storage |
| `internal/storage/filestore.go` | `doc/src/internal/storage/filestore.plan.md` | Storage Implementation | storage |

### internal/replay/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/replay/engine.go` | `doc/src/internal/replay/engine.plan.md` | Replay Engine | replay |
| `internal/replay/worker.go` | `doc/src/internal/replay/worker.plan.md` | Replay Worker | replay |
| `internal/replay/transform.go` | `doc/src/internal/replay/transform.plan.md` | Request Transform | replay |

### internal/diff/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/diff/engine.go` | `doc/src/internal/diff/engine.plan.md` | Diff Engine | diff |
| `internal/diff/json.go` | `doc/src/internal/diff/json.plan.md` | JSON Differ | diff |
| `internal/diff/db.go` | `doc/src/internal/diff/db.plan.md` | DB Diff | diff |
| `internal/diff/mongo.go` | `doc/src/internal/diff/mongo.plan.md` | MongoDB Diff | diff |
| `internal/diff/rules.go` | `doc/src/internal/diff/rules.plan.md` | Diff Rules | diff |

### internal/reporter/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/reporter/reporter.go` | `doc/src/internal/reporter/reporter.plan.md` | Interface Definition | reporter |
| `internal/reporter/terminal.go` | `doc/src/internal/reporter/terminal.plan.md` | Reporter Implementation | reporter |
| `internal/reporter/json.go` | `doc/src/internal/reporter/json.plan.md` | Reporter Implementation | reporter |
| `internal/reporter/html.go` | `doc/src/internal/reporter/html.plan.md` | Reporter Implementation | reporter |

### internal/logger/

| Source File | Doc File | File Type | Module |
|---|---|---|---|
| `internal/logger/logger.go` | `doc/src/internal/logger/logger.plan.md` | Logger | logger |

---

## Summary

| Module | Source Files | Doc Files |
|---|---|---|
| root | 1 | 1 |
| cmd | 7 | 7 |
| model | 5 | 5 |
| config | 2 | 2 |
| capture | 2 | 2 |
| dbhook | 4 | 4 |
| storage | 2 | 2 |
| replay | 3 | 3 |
| diff | 5 | 5 |
| reporter | 4 | 4 |
| logger | 1 | 1 |
| **Total** | **36** | **36** |
