# Shadiff - Shadow Traffic Semantic Comparison Tool

[中文文档](README_CN.md)

[![CI](https://github.com/zzqDeco/shadiff/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/zzqDeco/shadiff/actions/workflows/ci.yml)

## About

Shadiff is a shadow traffic semantic comparison tool for cross-framework / cross-language API migration validation. It uses a **record-replay-diff** three-stage pipeline: transparently captures the old API's complete behavior (inputs, outputs, database side effects) via reverse proxy, then replays the same inputs against the new API and performs semantic-level comparison of both sides' behavior.

## Features

- **HTTP Reverse Proxy Recording** — Transparent traffic capture via `httputil.ReverseProxy`, records full request/response pairs with timing
- **Database Protocol Proxying** — TCP-level black-box capture for MySQL (COM_QUERY), PostgreSQL (Simple/Extended Query), MongoDB (OP_MSG Wire Protocol), and Redis (RESP commands)
- **Concurrent Replay Engine** — Worker pool-based replay with configurable concurrency, request transformation (host/header substitution)
- **Semantic JSON Diff** — Recursive structural comparison with path tracking (e.g., `body.data.items[0].name`)
- **Configurable Rule System** — Ignore timestamps, UUIDs, numeric tolerance, array ordering via YAML rules
- **Multi-format Reporting** — Terminal (colored), JSON, and HTML reports with summary statistics
- **Session Management** — Full session lifecycle with JSONL streaming storage

## Tech Stack

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.25 | Primary language |
| Cobra | v1.9 | CLI framework |
| slog | stdlib | Structured logging with daily rotation |
| JSONL | - | Streaming record storage |

## Project Structure

```
shadiff/
├── main.go                            # CLI entry point
├── go.mod                             # Go 1.25 module
├── CLAUDE.md                          # Developer guide
├── cmd/                               # CLI commands
│   ├── root.go                        # Cobra root, global flags
│   ├── record.go                      # shadiff record
│   ├── record_stop.go                 # shadiff record stop
│   ├── record_status.go              # shadiff record status
│   ├── replay.go                      # shadiff replay
│   ├── diff.go                        # shadiff diff
│   ├── report.go                      # shadiff report
│   ├── session.go                     # shadiff session (list/show/delete)
│   └── version.go                     # shadiff version
├── internal/
│   ├── model/                         # Core data models
│   │   ├── session.go                 # Recording session
│   │   ├── record.go                  # Single behavior record (request+response+side effects)
│   │   ├── request.go                 # HTTP request/response models
│   │   ├── sideeffect.go             # Side effect model (DB operations, external calls)
│   │   └── diff.go                    # Diff result model
│   ├── config/                        # Configuration management
│   │   ├── config.go                  # Config type definitions + DefaultConfig()
│   │   └── store.go                   # JSON file store (~/.shadiff/config.json)
│   ├── capture/                       # Traffic capture layer
│   │   ├── proxy.go                   # HTTP reverse proxy (httputil.ReverseProxy)
│   │   ├── recorder.go               # Unified recorder, assembles Record and persists
│   │   └── dbhook/                    # Database protocol proxies
│   │       ├── hook.go                # DBHook interface definition
│   │       ├── mysql.go               # MySQL protocol proxy (COM_QUERY parsing)
│   │       ├── postgres.go            # PostgreSQL protocol proxy (Simple/Extended Query)
│   │       ├── mongo.go               # MongoDB protocol proxy (OP_MSG Wire Protocol)
│   │       └── redis.go               # Redis protocol proxy (RESP command parsing)
│   ├── storage/                       # Storage layer
│   │   ├── store.go                   # SessionStore/RecordStore/DiffStore interfaces
│   │   └── filestore.go              # Filesystem implementation (JSONL)
│   ├── replay/                        # Replay engine
│   │   ├── engine.go                  # Replay orchestrator
│   │   ├── worker.go                  # Concurrent worker pool
│   │   └── transform.go              # Request transformation (host/header substitution)
│   ├── diff/                          # Semantic diff engine
│   │   ├── engine.go                  # Diff orchestrator, pairs records by sequence
│   │   ├── json.go                    # JSON structural recursive diff
│   │   ├── db.go                      # SQL database diff (MySQL/PostgreSQL)
│   │   ├── mongo.go                   # MongoDB operation diff
│   │   ├── redis.go                   # Redis command diff
│   │   └── rules.go                   # Diff rules + built-in matchers
│   ├── reporter/                      # Report generation
│   │   ├── reporter.go                # Reporter interface + factory
│   │   ├── terminal.go                # Colored terminal output
│   │   ├── json.go                    # JSON format
│   │   └── html.go                    # HTML report (embedded template)
│   ├── daemon/                        # Daemon process management
│   │   ├── pidfile.go                 # PID file read/write/check
│   │   ├── process_unix.go            # Unix process detach + signals
│   │   └── process_windows.go         # Windows process detach + signals
│   └── logger/                        # Structured logging
│       └── logger.go                  # slog + daily rotation
├── plan/                              # Development roadmap
└── logs/                              # Runtime logs (gitignored)
```

## Getting Started

### Prerequisites

- **Go** >= 1.25

### Installation

```bash
go install github.com/zzqDeco/shadiff@latest
```

Or build from source:

```bash
git clone https://github.com/zzqDeco/shadiff.git
cd shadiff
go build -o shadiff .
```

## Development Branch Flow

- `main` is the stable promotion branch and the current default branch.
- `dev` is the integration branch for ongoing work.
- `master` is a deprecated legacy branch and is no longer used for new work.
- Create short-lived working branches from `dev`.
- Open feature, fix, docs, refactor, and test PRs into `dev` first.
- Promote `dev` into `main` with a separate PR when ready.

GitHub Actions run `go test ./...` and `go build -o shadiff .` on pushes and pull requests for `main` and `dev`. Release tags matching `v*.*.*` build Linux, macOS, and Windows archives for amd64 and arm64, verify archive contents and version metadata, and publish SHA-256 checksums.

Release assets can be checked locally after building `dist/`:

```bash
bash scripts/verify-release-assets.sh dist v0.1.1
```

Docker-backed database integration tests are opt-in and are not part of the default unit test command:

```bash
go test -v -tags integration ./internal/integration -count=1 -timeout=20m
```

### Official E2E Demo

Run the reproducible Docker Compose demo to exercise the real CLI across `record -> replay -> diff -> report` with HTTP, MySQL, PostgreSQL, MongoDB, and Redis side effects:

```bash
./examples/e2e/run.sh --assert
```

The demo writes isolated artifacts under `examples/e2e/.work/<run-id>/`, including `diff.json` and `report.html`. See `examples/e2e/README.md` for ports, expected differences, and troubleshooting.

## Usage

### 1. Record Traffic

Start a reverse proxy to capture traffic from the old API:

```bash
# Basic HTTP recording
shadiff record -t http://old-api:8080 -l :18080 -s "migration-v1"

# With MySQL protocol proxy
shadiff record -t http://old-api:8080 -l :18080 \
  --db-proxy mysql://:13306->:3306 -s "mysql-migration"

# Run as background daemon
shadiff record -D -t http://old-api:8080 -l :18080 -s "bg-session"

# With MongoDB protocol proxy
shadiff record -t http://old-api:8080 -l :18080 \
  --db-proxy mongo://:27018->:27017 -s "mongo-migration"

# With Redis protocol proxy
shadiff record -t http://old-api:8080 -l :18080 \
  --db-proxy redis://:16379->:6379 -s "redis-migration"

# Multiple database proxies
shadiff record -t http://old-api:8080 -l :18080 \
  --db-proxy mysql://:13306->:3306 \
  --db-proxy mongo://:27018->:27017 \
  --db-proxy redis://:16379->:6379 -s "full-migration"
```

Point your traffic to `localhost:18080` instead of the old API. All requests, responses, and database operations are recorded.
When recording uses DB proxies, Shadiff flushes hook-delivered side effects before each request scope closes so late-delivered in-window effects are less likely to be lost.

#### Daemon Mode

Run recording in the background and manage it with `stop` and `status`:

```bash
# Start daemon
shadiff record -D -t http://localhost:8080 -l :18080 -s "long-run"

# Check status
shadiff record status
shadiff record status -s "long-run"

# Stop daemon
shadiff record stop -s "long-run"
```

### 2. Replay Traffic

Replay recorded traffic against the new API:

```bash
shadiff replay -s "migration-v1" -t http://new-api:9090 -c 5
shadiff replay -s "migration-v1" -t http://new-api:9090 \
  --db-proxy mysql://:13307->:3306
```

When replay uses `--db-proxy`, DB side effects are captured into `replay-records.jsonl` and replay must stay serial (`--concurrency 1`).
If `--db-proxy` is omitted, replay falls back to `replay.dbProxies` from the config file.
Replay also flushes DB-hook telemetry before each request window is finalized so semantic diff sees in-window SQL, Mongo, and Redis side effects more reliably.

### 3. Compare Results

Run semantic diff on recorded vs replayed behavior:

```bash
# Basic diff
shadiff diff -s "migration-v1"

# With custom rules (ignore timestamps, UUIDs)
shadiff diff -s "migration-v1" -r rules.yaml --ignore-order

# JSON output for scripts/CI
shadiff diff -s "migration-v1" -o json

# Write CI JSON to a file
shadiff diff -s "migration-v1" -o json --output-file diff.json

# Fail CI when unignored differences are found
shadiff diff -s "migration-v1" --fail-on diff
```

`--fail-on` accepts `none` (default), `diff`, or `error`. Use `diff` to fail on any unignored difference, or `error` to fail only when unignored error-severity differences exist.

### 4. Generate Report

```bash
# Terminal output (default)
shadiff report -s "migration-v1"

# HTML report
shadiff report -s "migration-v1" -f html -o report.html

# JSON report
shadiff report -s "migration-v1" -f json -o report.json
```

### 5. Manage Sessions

```bash
shadiff session list
shadiff session show <session-id>
shadiff session delete <session-id>
```

## Configuration

App configuration is stored at `~/.shadiff/config.json`:

Precedence:

```text
CLI flag > config.json > built-in defaults
```

If the config file does not exist, Shadiff creates it automatically on first run. You can also point to another file with `--config /path/to/config.json`.

| Block | Description |
|-------|-------------|
| `capture` | `listenAddr`, `maxBodySize`, `excludePaths`, `dbProxies` |
| `replay` | `concurrency`, `timeout`, `retryCount`, `delayMs`, `dbProxies` |
| `diff` | `ignoreHeaders`, `ignoreOrder`, `maxDiffs`, `rules`, `rulesFile` |
| `storage` | `dataDir`, `maxSessions` |
| `log` | `level`, `logDir` |

Example:

```json
{
  "capture": {
    "listenAddr": ":18080",
    "maxBodySize": 1048576,
    "excludePaths": ["/healthz"],
    "dbProxies": [
      {
        "type": "mysql",
        "listenAddr": ":13306",
        "targetAddr": "127.0.0.1:3306"
      }
    ]
  },
  "replay": {
    "concurrency": 5,
    "timeout": "30s",
    "retryCount": 1,
    "delayMs": 100,
    "dbProxies": [
      {
        "type": "mysql",
        "listenAddr": ":13307",
        "targetAddr": "127.0.0.1:3306"
      }
    ]
  },
  "diff": {
    "ignoreOrder": true,
    "maxDiffs": 500,
    "rulesFile": "rules.yaml"
  },
  "storage": {
    "dataDir": "D:/shadiff-data",
    "maxSessions": 100
  },
  "log": {
    "level": "info",
    "logDir": "D:/shadiff-data/logs"
  }
}
```

Notes:

- `capture.maxBodySize` truncates the inline recorded request/response body preview but keeps the original `bodyLen`.
- When a request body is truncated inline, Shadiff stores the full request body under the session directory and replay uses that artifact automatically.
- `capture.excludePaths` skips recording matching HTTP paths while still proxying them.
- `capture.dbProxies` uses the same format as `--db-proxy`.
- `diff.rulesFile` accepts JSON, YAML, or YML rule files.
- `storage.maxSessions` prunes the oldest non-recording sessions before creating a new one.

## Data Storage

All persistent data is stored under `~/.shadiff/`:

```
~/.shadiff/
├── config.json                        # Global configuration
├── logs/                              # Log files
└── sessions/
    └── {session-id}/
        ├── session.json               # Session metadata
        ├── records.jsonl              # Recorded behavior (JSONL streaming)
        ├── replay-records.jsonl       # Replay results
        ├── diff-results.json          # Diff results
        ├── artifacts/
        │   └── request-bodies/        # Full request-body artifacts used for faithful replay
        ├── pidfile                    # Daemon PID file (daemon mode only)
        └── daemon.log                 # Daemon stdout/stderr log (daemon mode only)
```

## DB Proxy Format

`--db-proxy` format: `<type>://<listen_addr>-><target_addr>`

Supported types: `mysql`, `postgres`, `mongo`, `redis`. Can be specified multiple times.

## Documentation

- **Dev Guide**: `CLAUDE.md` — Architecture overview + engineering conventions
- **Roadmap**: `plan/` — Development phases and progress
