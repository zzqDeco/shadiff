# Shadiff Interface Documentation

## Overview

Shadiff is a shadow traffic semantic diff tool that validates behavioral consistency of API migrations through a black-box **record-replay-diff** three-stage workflow. This document describes every CLI command, internal Go interface, and the data flow contracts that bind modules together.

---

## 1. CLI Commands

All commands are built with [cobra](https://github.com/spf13/cobra). The binary name is `shadiff`.

### Global Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | | `~/.shadiff/config.json` | Config file path |
| `--verbose` | `-v` | `false` | Show verbose logs |
| `--quiet` | `-q` | `false` | Show errors only |

---

### `shadiff record`

Start an HTTP reverse proxy to record all requests, responses, and database side effects passing through it.

```
shadiff record -t http://localhost:8080 -l :18080 -s "user-module-migration"
shadiff record -t http://old-api:8080 --db-proxy mysql://:13306->:3306
```

| Flag | Short | Default | Required | Description |
|------|-------|---------|----------|-------------|
| `--target` | `-t` | | Yes | Target service address (e.g. `http://localhost:8080`) |
| `--listen` | `-l` | `:18080` | No | Proxy listen address |
| `--session` | `-s` | auto-generated (`record-YYYYMMDD-HHMMSS`) | No | Session name |
| `--db-proxy` | | | No | DB proxy specification (repeatable, e.g. `mysql://:13306->:3306`) |
| `--duration` | `-d` | | No | Maximum recording duration (e.g. `30m`) |
| `--daemon` | `-D` | `false` | No | Run as background daemon |

**Behavior**: Creates a session, starts an HTTP reverse proxy from `--listen` to `--target`, captures every request/response pair as a `Record`, and persists them via JSONL streaming. Stops on SIGINT/SIGTERM or when `--duration` expires. On shutdown, updates the session status to `completed` with the final record count.

In daemon mode (`-D`), the parent process creates the session, re-execs the binary as a detached child, writes the PID file, and exits immediately. The child runs the proxy in the background with output redirected to `daemon.log`.

---

### `shadiff record stop`

Stop a daemon recording session.

```
shadiff record stop -s my-session
shadiff record stop -s a1b2c3d4
```

| Flag | Short | Default | Required | Description |
|------|-------|---------|----------|-------------|
| `--session` | `-s` | | Yes | Session name or ID |

**Behavior**: Resolves the session by name or ID, reads the PID file from the session directory, and checks if the process is alive. Sends a stop signal (SIGTERM on Unix, os.Interrupt on Windows). Waits up to 10 seconds for graceful exit (polling every 500ms), then force kills if still alive. Cleans up stale PID files when the process is already dead.

---

### `shadiff record status`

Show recording session status.

```
shadiff record status
shadiff record status -s my-session
```

| Flag | Short | Default | Required | Description |
|------|-------|---------|----------|-------------|
| `--session` | `-s` | | No | Session name or ID |

**Behavior**: Without `-s`: lists all sessions with status `recording` in a table showing ID, Name, PID, process alive status, and creation time. With `-s`: shows detailed session information including ID, Name, Status, DaemonMode, PID, process alive status, record count, target URL, creation time, and uptime.

---

### `shadiff replay`

Read requests from a recorded session, send them to the target service, and record new responses.

```
shadiff replay -s abc123 -t http://new-api:9090
shadiff replay -s "user-module-migration" -t http://localhost:9090 -c 5
```

| Flag | Short | Default | Required | Description |
|------|-------|---------|----------|-------------|
| `--session` | `-s` | | Yes | Session ID or name |
| `--target` | `-t` | | Yes | Replay target address (e.g. `http://localhost:9090`) |
| `--concurrency` | `-c` | `1` | No | Concurrency level (worker pool size) |
| `--delay` | | | No | Delay between requests (e.g. `100ms`) |
| `--db-proxy` | | | No | DB proxy specification (repeatable) |

**Behavior**: Resolves the session by ID or name, loads all recorded records, replays them against the target using a configurable worker pool, and saves replay records to `replay-records.jsonl`. Updates the session status to `replayed`.

---

### `shadiff diff`

Compare behavioral differences between recorded and replayed traffic.

```
shadiff diff -s abc123
shadiff diff -s "user-module-migration" --ignore-order -r rules.yaml
```

| Flag | Short | Default | Required | Description |
|------|-------|---------|----------|-------------|
| `--session` | `-s` | | Yes | Session ID or name |
| `--rules` | `-r` | | No | Diff rules file (JSON/YAML) |
| `--ignore-order` | | `false` | No | Ignore JSON array element order |
| `--ignore-headers` | | | No | Additional headers to ignore (repeatable) |
| `--output` | `-o` | `terminal` | No | Output format: `terminal`, `json` |

**Behavior**: Loads recorded and replayed records, pairs them by sequence number, and compares status codes, response headers, JSON response bodies (structural diff), and side effect counts. Applies built-in and user-defined rules to mark expected differences as ignored. Saves results to `diff-results.json` and prints a summary.

---

### `shadiff report`

Generate a detailed report from diff results.

```
shadiff report -s abc123
shadiff report -s abc123 -f html -o report.html
shadiff report -s abc123 -f json -o result.json
```

| Flag | Short | Default | Required | Description |
|------|-------|---------|----------|-------------|
| `--session` | `-s` | | Yes | Session ID or name |
| `--format` | `-f` | `terminal` | No | Report format: `terminal`, `json`, `html` |
| `--output` | `-o` | stdout | No | Output file path |

**Behavior**: Loads saved diff results, computes summary statistics, and delegates to the appropriate `Reporter` implementation. When `--output` is set, writes to a file; otherwise writes to stdout.

---

### `shadiff session`

Manage recording sessions. This is a parent command with three subcommands.

#### `shadiff session list`

```
shadiff session list
shadiff session list --tag regression
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--tag` | | | Filter sessions by tag |

Lists all sessions in a table: ID, Name, Status, Records, Created.

#### `shadiff session show <id>`

Positional argument: session ID. Displays all session metadata fields.

#### `shadiff session delete <id>`

Positional argument: session ID. Deletes the session and all associated data (records, replay records, diff results).

---

### `shadiff version`

Display build version information.

```
shadiff version
```

Output:
```
shadiff 0.1.0
  commit:  <git-commit>
  built:   <build-date>
```

Version, Commit, and BuildDate are injected at build time via `-ldflags`.

---

## 2. Internal Go Interfaces

### 2.1 `SessionStore`

**Package**: `internal/storage`
**File**: `store.go`

Manages session lifecycle (CRUD).

```go
type SessionStore interface {
    Create(session *model.Session) error
    Get(id string) (*model.Session, error)
    List(filter *model.SessionFilter) ([]model.Session, error)
    Update(session *model.Session) error
    Delete(id string) error
}
```

| Method | Description |
|--------|-------------|
| `Create` | Generates a UUID short ID, sets timestamps, creates the session directory and `session.json` |
| `Get` | Loads a session by ID from `{baseDir}/{id}/session.json` |
| `List` | Scans all session directories, applies optional filter (name substring, status, tags), returns sorted by `UpdatedAt` descending |
| `Update` | Overwrites `session.json` with updated fields and refreshed `UpdatedAt` |
| `Delete` | Removes the entire session directory (`os.RemoveAll`) |

**Implementor**: `FileStore` (`internal/storage/filestore.go`)

---

### 2.2 `RecordStore`

**Package**: `internal/storage`
**File**: `store.go`

Handles append-only record persistence using JSONL streaming.

```go
type RecordStore interface {
    AppendRecord(sessionID string, record *model.Record) error
    ListRecords(sessionID string) ([]model.Record, error)
    GetRecord(sessionID string, recordID string) (*model.Record, error)
    CountRecords(sessionID string) (int, error)
}
```

| Method | Description |
|--------|-------------|
| `AppendRecord` | Appends a JSON-serialized record followed by a newline to `records.jsonl` |
| `ListRecords` | Reads and deserializes all lines from `records.jsonl` (max 10 MB per line) |
| `GetRecord` | Linear scan of `ListRecords` output to find by record ID |
| `CountRecords` | Returns `len(ListRecords(...))` |

**Implementor**: `FileStore` (`internal/storage/filestore.go`)

`FileStore` also provides two additional methods not in the interface:
- `AppendReplayRecord(sessionID, record)` -- appends to `replay-records.jsonl`
- `ListReplayRecords(sessionID)` -- reads from `replay-records.jsonl`

---

### 2.3 `DiffStore`

**Package**: `internal/storage`
**File**: `store.go`

Persists and retrieves diff comparison results.

```go
type DiffStore interface {
    SaveResults(sessionID string, results []model.DiffResult) error
    LoadResults(sessionID string) ([]model.DiffResult, error)
}
```

| Method | Description |
|--------|-------------|
| `SaveResults` | Writes the full results array as pretty-printed JSON to `diff-results.json` |
| `LoadResults` | Reads and deserializes `diff-results.json`; returns `nil, nil` if file does not exist |

**Implementor**: `FileStore` (`internal/storage/filestore.go`)

---

### 2.4 `DBHook`

**Package**: `internal/capture/dbhook`
**File**: `hook.go`

Captures database operations by acting as a transparent TCP proxy that sniffs the database wire protocol.

```go
type DBHook interface {
    Start(ctx context.Context) error
    Stop() error
    SideEffects() <-chan model.SideEffect
    Type() string
}
```

| Method | Description |
|--------|-------------|
| `Start` | Begins listening on the proxy address and accepting connections; sniffs protocol traffic in background goroutines |
| `Stop` | Closes the listener, waits for all connection goroutines to finish, closes the side-effect channel |
| `SideEffects` | Returns a read-only channel (buffered, capacity 1000) that emits captured `model.SideEffect` values |
| `Type` | Returns the database type identifier string |

**Implementors**:

| Struct | Type | File | Protocol Details |
|--------|------|------|------------------|
| `MySQLHook` | `"mysql"` | `mysql.go` | Parses MySQL packet format (3-byte length + 1-byte seq + payload); captures `COM_QUERY` (0x03) and `COM_STMT_PREPARE` (0x16) |
| `PostgresHook` | `"postgres"` | `postgres.go` | Parses PostgreSQL frontend messages; captures Simple Query (`Q`) and Extended Query Parse (`P`) messages |
| `MongoHook` | `"mongo"` | `mongo.go` | Parses MongoDB OP_MSG wire protocol (opcode 2013); extracts CRUD commands (find, insert, update, delete, aggregate, count, distinct, findAndModify) via simplified BSON parsing |

**Factory**: `NewHook(cfg Config) (DBHook, error)` dispatches on `cfg.DBType`.

---

### 2.5 `Matcher`

**Package**: `internal/diff`
**File**: `rules.go`

Custom value matcher used by the diff rule engine to determine whether a detected difference should be ignored.

```go
type Matcher interface {
    Name() string
    Match(path string, expected, actual any) (match bool, err error)
}
```

| Method | Description |
|--------|-------------|
| `Name` | Returns a unique identifier string for this matcher |
| `Match` | Given a JSON path and two values, returns `true` if the difference is acceptable |

**Built-in Implementors**:

| Struct | Name | Logic |
|--------|------|-------|
| `TimestampMatcher` | `"timestamp"` | Both values are strings matching `YYYY-MM-DDTHH:MM` or `YYYY-MM-DD HH:MM` patterns |
| `UUIDMatcher` | `"uuid"` | Both values are strings matching the standard UUID format (`8-4-4-4-12` hex) |
| `NumericToleranceMatcher` | `"numeric_tolerance"` | Both values are numeric and differ by no more than `Tolerance` (default 0.001) |

---

### 2.6 `Reporter`

**Package**: `internal/reporter`
**File**: `reporter.go`

Generates formatted output from diff results and summary statistics.

```go
type Reporter interface {
    Generate(results []model.DiffResult, summary model.DiffSummary, w io.Writer) error
}
```

| Method | Description |
|--------|-------------|
| `Generate` | Writes the formatted report to the provided `io.Writer` |

**Implementors**:

| Struct | Format | File | Description |
|--------|--------|------|-------------|
| `TerminalReporter` | `"terminal"` | `terminal.go` | ANSI-colored output with unicode symbols (checkmarks, crosses, tree connectors) |
| `JSONReporter` | `"json"` | `json.go` | Pretty-printed JSON with `summary` and `results` top-level keys |
| `HTMLReporter` | `"html"` | `html.go` | Self-contained HTML page with embedded CSS; uses Go `html/template` |

**Factory**: `NewReporter(format string) (Reporter, error)` dispatches on the format string.

---

## 3. Data Flow Contracts

### 3.1 End-to-End Pipeline

```
                  +-----------+       +------------+       +----------+       +----------+
  HTTP Traffic -> |  record   | ----> |   replay   | ----> |   diff   | ----> |  report  |
                  +-----------+       +------------+       +----------+       +----------+
                        |                   |                    |                  |
                        v                   v                    v                  v
                  records.jsonl     replay-records.jsonl   diff-results.json   stdout/file
```

All data is scoped under a single **Session** (identified by an 8-character UUID prefix). The filesystem layout under `~/.shadiff/sessions/{sessionID}/` is:

```
session.json            -- Session metadata (JSON)
records.jsonl           -- Recorded request/response pairs (JSONL, append-only)
replay-records.jsonl    -- Replayed request/response pairs (JSONL, append-only)
diff-results.json       -- Diff comparison results (JSON array)
```

### 3.2 Channels for Side Effects

Database side effects flow through Go channels, not through the filesystem directly:

```
DBHook.SideEffects()  -->  chan model.SideEffect (buffered, cap 1000)
                                     |
                                     v
                           Recorder.sideEffectCh  (background goroutine collects into pendingEffects)
                                     |
                                     v
                           Recorder.Record()  (attaches pendingEffects to the current Record)
                                     |
                                     v
                           FileStore.AppendRecord()  (serialized into records.jsonl)
```

1. Each `DBHook` implementation (MySQL, PostgreSQL, MongoDB) emits `model.SideEffect` values on a buffered channel (capacity 1000). If the channel is full, the side effect is dropped with a warning log.
2. The `Recorder` runs a background goroutine (`collectSideEffects`) that drains the channel and accumulates side effects into `pendingEffects` (mutex-protected).
3. When `Recorder.Record()` is called (triggered by the HTTP proxy after each request/response round-trip), it atomically moves all `pendingEffects` onto the `Record.SideEffects` slice, then appends the complete record to storage.
4. On `Recorder.Stop()`, the background goroutine drains any remaining channel items before exiting.

### 3.3 JSONL for Persistence

Records use **JSONL** (JSON Lines) format for streaming append-only writes:

- Each record is a single JSON object followed by a newline character (`\n`).
- Writes are mutex-protected (`sync.RWMutex` in `FileStore`) and use `O_APPEND` for crash safety.
- Reads use `bufio.Scanner` with a 10 MB per-line buffer limit.
- Corrupted lines (invalid JSON) are silently skipped during reads.

Diff results use standard JSON (a single JSON array), since they are written once after the full comparison completes.

### 3.4 Record Pairing via Sequence Numbers

Records are paired between the `record` and `replay` phases using the `Sequence` field:

1. During recording, the `Proxy` assigns a monotonically increasing sequence number (via `atomic.Int64`) to each captured request.
2. During replay, each replayed record inherits the `Sequence` from its corresponding original record.
3. During diff, the engine builds a map (`map[int]model.Record`) from replay records keyed by sequence, then iterates over original records and looks up the matching replay record by sequence number. Missing replay records produce an error-level "replay record missing" difference.

### 3.5 Request Transformation

Before replay, recorded requests are transformed via `replay.Transform()`:

1. The target base URL is replaced (e.g., old host to new host).
2. Original headers are copied, then specified headers are removed or overridden.
3. Proxy-related headers (`X-Forwarded-For`, `X-Forwarded-Host`, `X-Forwarded-Proto`) are always stripped.

### 3.6 Diff Rule Application

After raw differences are computed, the `RuleSet.Apply()` method processes them:

1. Each `Difference` is checked against all rules.
2. Rule path patterns support wildcards: `*` (single level), `**` (multiple levels), `[*]` (array index). Patterns are pre-compiled to regexps.
3. For `"ignore"` rules, matching differences are marked `Ignored = true` with the rule name.
4. For `"custom"` rules, the named `Matcher` is invoked; if it returns `true`, the difference is marked ignored.
5. A `DiffResult` is considered a `Match` only if all its differences are `Ignored`.

### 3.7 Module Communication Summary

| Producer | Consumer | Contract | Medium |
|----------|----------|----------|--------|
| `Proxy` | `Recorder` | `Recorder.Record(*model.Record)` | Direct method call |
| `DBHook` | `Recorder` | `model.SideEffect` | Buffered channel (cap 1000) |
| `Recorder` | `FileStore` | `AppendRecord(sessionID, *model.Record)` | JSONL file append |
| `replay.Engine` | `FileStore` | `AppendReplayRecord(sessionID, *model.Record)` | JSONL file append |
| `diff.Engine` | `FileStore` | `SaveResults(sessionID, []model.DiffResult)` | JSON file write |
| `FileStore` | `diff.Engine` | `ListRecords` / `ListReplayRecords` | JSONL file read |
| `FileStore` | `reporter` | `LoadResults(sessionID)` | JSON file read |
| `diff.Engine` | `RuleSet` | `RuleSet.Apply([]model.Difference)` | Direct method call |
| `RuleSet` | `Matcher` | `Matcher.Match(path, expected, actual)` | Interface method call |
| `cmd/report` | `Reporter` | `Reporter.Generate(results, summary, w)` | Interface method call |
