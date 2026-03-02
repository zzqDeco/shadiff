# Shadiff - Shadow Traffic Semantic Comparison Tool

[中文文档](README_CN.md)

## About

Shadiff is a shadow traffic semantic comparison tool for cross-framework / cross-language API migration validation. It uses a **record-replay-diff** three-stage pipeline: transparently captures the old API's complete behavior (inputs, outputs, database side effects) via reverse proxy, then replays the same inputs against the new API and performs semantic-level comparison of both sides' behavior.

## Features

- **HTTP Reverse Proxy Recording** — Transparent traffic capture via `httputil.ReverseProxy`, records full request/response pairs with timing
- **Database Protocol Proxying** — TCP-level black-box capture for MySQL (COM_QUERY), PostgreSQL (Simple/Extended Query), and MongoDB (OP_MSG Wire Protocol)
- **Concurrent Replay Engine** — Worker pool-based replay with configurable concurrency, request transformation (host/header substitution)
- **Semantic JSON Diff** — Recursive structural comparison with path tracking (e.g., `body.data.items[0].name`)
- **Configurable Rule System** — Ignore timestamps, UUIDs, numeric tolerance, array ordering via YAML rules
- **Multi-format Reporting** — Terminal (colored), JSON, and HTML reports with summary statistics
- **Session Management** — Full session lifecycle with JSONL streaming storage

## Tech Stack

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.24 | Primary language |
| Cobra | v1.9 | CLI framework |
| slog | stdlib | Structured logging with daily rotation |
| JSONL | - | Streaming record storage |

## Project Structure

```
shadiff/
├── main.go                            # CLI entry point
├── go.mod                             # Go 1.24 module
├── CLAUDE.md                          # Developer guide
├── cmd/                               # CLI commands
│   ├── root.go                        # Cobra root, global flags
│   ├── record.go                      # shadiff record
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
│   │       └── mongo.go               # MongoDB protocol proxy (OP_MSG Wire Protocol)
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
│   │   └── rules.go                   # Diff rules + built-in matchers
│   ├── reporter/                      # Report generation
│   │   ├── reporter.go                # Reporter interface + factory
│   │   ├── terminal.go                # Colored terminal output
│   │   ├── json.go                    # JSON format
│   │   └── html.go                    # HTML report (embedded template)
│   └── logger/                        # Structured logging
│       └── logger.go                  # slog + daily rotation
├── plan/                              # Development roadmap
└── logs/                              # Runtime logs (gitignored)
```

## Getting Started

### Prerequisites

- **Go** >= 1.24

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

## Usage

### 1. Record Traffic

Start a reverse proxy to capture traffic from the old API:

```bash
# Basic HTTP recording
shadiff record -t http://old-api:8080 -l :18080 -s "migration-v1"

# With MySQL protocol proxy
shadiff record -t http://old-api:8080 -l :18080 \
  --db-proxy mysql://:13306->:3306 -s "mysql-migration"

# With MongoDB protocol proxy
shadiff record -t http://old-api:8080 -l :18080 \
  --db-proxy mongo://:27018->:27017 -s "mongo-migration"

# Multiple database proxies
shadiff record -t http://old-api:8080 -l :18080 \
  --db-proxy mysql://:13306->:3306 \
  --db-proxy mongo://:27018->:27017 -s "full-migration"
```

Point your traffic to `localhost:18080` instead of the old API. All requests, responses, and database operations are recorded.

### 2. Replay Traffic

Replay recorded traffic against the new API:

```bash
shadiff replay -s "migration-v1" -t http://new-api:9090 -c 5
```

### 3. Compare Results

Run semantic diff on recorded vs replayed behavior:

```bash
# Basic diff
shadiff diff -s "migration-v1"

# With custom rules (ignore timestamps, UUIDs)
shadiff diff -s "migration-v1" -r rules.yaml --ignore-order
```

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

| Block | Description |
|-------|-------------|
| `capture` | Proxy settings (listen address, timeouts) |
| `replay` | Replay settings (concurrency, delay, timeouts) |
| `diff` | Diff settings (default rules, ignore patterns) |
| `storage` | Storage settings (data directory) |
| `log` | Logging settings (level, directory, rotation) |

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
        └── diff-results.json          # Diff results
```

## DB Proxy Format

`--db-proxy` format: `<type>://<listen_addr>-><target_addr>`

Supported types: `mysql`, `postgres`, `mongo`. Can be specified multiple times.

## Documentation

- **Dev Guide**: `CLAUDE.md` — Architecture overview + engineering conventions
- **Roadmap**: `plan/` — Development phases and progress
