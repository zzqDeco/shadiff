# Shadiff -- Project Technical Document

## 1. Project Positioning

| Field | Value |
|-------|-------|
| Repository | `shadiff` (local, `E:\record\src\shadiff`) |
| Type | CLI tool |
| Name | **Shadiff** -- Shadow Traffic Semantic Diff Tool |
| Language | Go 1.24 |
| Source files | 36 `.go` files (25 production, 11 test) |
| Business purpose | Validate behavioral consistency of cross-framework / cross-language API migrations through a black-box record-replay-diff workflow |

Shadiff targets the scenario where a team is migrating or rewriting an API service (e.g., from Java Spring to Go, or from monolith to microservices). Instead of writing integration tests by hand, the operator points Shadiff at the old service, records live traffic (including database side effects), replays it against the new service, and gets a semantic diff report showing exactly what behaves differently.

---

## 2. Implementation Facts

| Aspect | Detail |
|--------|--------|
| Build system | `go build` / `go install`, module path `shadiff` |
| Go module | `go 1.24`, dependencies: `cobra` (CLI), `uuid` (ID generation) |
| Entry point | `main.go` -> `cmd.Execute()` (Cobra root command) |
| Configuration | JSON file at `~/.shadiff/config.json`, loaded by `internal/config/Store` with defaults via `DefaultConfig()` |
| Storage | File-system based, data directory at `~/.shadiff/sessions/` |
| Storage format | Session metadata in `session.json`, records in `records.jsonl` / `replay-records.jsonl` (JSONL streaming), diff results in `diff-results.json` |
| Logging | `log/slog` with daily-rotated files at `~/.shadiff/logs/shadiff-YYYY-MM-DD.log`, dual output to stderr + file |
| Version | `0.1.0`, build-time injected via `cmd.Version`, `cmd.Commit`, `cmd.BuildDate` |
| Test coverage | Unit tests for: config, models, filestore, recorder, JSON diff, DB diff, MongoDB diff, rules, transforms, reporters, DB hook factory |

---

## 3. Key Entry Points and Run Modes

Shadiff operates as a four-stage CLI pipeline. Each stage is a Cobra subcommand:

### `shadiff record`

Start an HTTP reverse proxy that transparently forwards traffic to the target service while capturing every request/response pair and database side effects.

```
shadiff record -t http://localhost:8080 -l :18080 -s "my-session"
```

- Flags: `--target` (required), `--listen`, `--session`, `--db-proxy`, `--duration`
- Creates a new `Session` in storage with status `recording`
- Runs until Ctrl+C or duration timeout, then transitions to `completed`

### `shadiff replay`

Read recorded requests from a session and send them sequentially (or concurrently) to a new target service, capturing the new responses.

```
shadiff replay -s abc123 -t http://new-api:9090 -c 5
```

- Flags: `--session` (required), `--target` (required), `--concurrency`, `--delay`, `--db-proxy`
- Supports concurrent replay via worker pool
- Transforms requests (URL rewriting, header manipulation, proxy header removal)
- Session status transitions to `replayed`

### `shadiff diff`

Compare recorded vs. replayed data at the semantic level and produce a structured diff report.

```
shadiff diff -s abc123 --ignore-order
```

- Flags: `--session` (required), `--rules`, `--ignore-order`, `--ignore-headers`, `--output`
- Compares: status codes, response headers, JSON body (recursive field-level), side effect counts
- Applies rule-based filtering (ignore, custom matchers for timestamps/UUIDs/numeric tolerance)
- Saves results to `diff-results.json`

### `shadiff report`

Generate a formatted report from saved diff results.

```
shadiff report -s abc123 -f html -o report.html
```

- Flags: `--session` (required), `--format` (terminal/json/html), `--output`
- Three output formats: colored terminal, structured JSON, standalone HTML page

### `shadiff session`

Manage recording sessions (list, show, delete).

```
shadiff session list
shadiff session show abc123
shadiff session delete abc123
```

### `shadiff version`

Display version, commit hash, and build date.

---

## 4. Module Summary

| Module | Package | Responsibility | Key Types |
|--------|---------|---------------|-----------|
| **CLI** | `cmd/` | Cobra command definitions, flag parsing, orchestration | `rootCmd`, `recordCmd`, `replayCmd`, `diffCmd`, `reportCmd`, `sessionCmd` |
| **Models** | `internal/model/` | Domain types shared across all modules | `Session`, `Record`, `HTTPRequest`, `HTTPResponse`, `SideEffect`, `DiffResult`, `Difference`, `DiffSummary` |
| **Config** | `internal/config/` | Configuration schema, defaults, file persistence | `AppConfig`, `CaptureConfig`, `DBProxyConfig`, `ReplayConfig`, `DiffConfig`, `StorageConfig`, `LogConfig`, `Store` |
| **Capture** | `internal/capture/` | HTTP reverse proxy, request/response recording, side-effect aggregation | `Proxy`, `Recorder`, `responseRecorder` |
| **DB Hooks** | `internal/capture/dbhook/` | TCP-level database protocol proxies for MySQL, PostgreSQL, MongoDB | `DBHook` (interface), `MySQLHook`, `PostgresHook`, `MongoHook`, `Config` |
| **Replay** | `internal/replay/` | Replay engine, concurrent worker pool, request transformation | `Engine`, `EngineConfig`, `WorkerPool`, `ReplayResult`, `TransformConfig`, `Transform()` |
| **Diff** | `internal/diff/` | Semantic comparison engine: JSON, SQL, MongoDB, rule system | `Engine`, `JSONDiffer`, `RuleSet`, `Rule`, `Matcher`, `TimestampMatcher`, `UUIDMatcher`, `NumericToleranceMatcher` |
| **Reporter** | `internal/reporter/` | Multi-format report generation | `Reporter` (interface), `TerminalReporter`, `JSONReporter`, `HTMLReporter` |
| **Storage** | `internal/storage/` | File-system storage with JSONL streaming and mutex-guarded access | `SessionStore`, `RecordStore`, `DiffStore` (interfaces), `FileStore` |
| **Logger** | `internal/logger/` | Structured logging with domain-specific convenience methods | `Init()`, `Close()`, `CaptureEvent()`, `ReplayEvent()`, `DiffEvent()`, `DBHookEvent()` |

---

## 5. Cross-Cutting Concerns

### 5.1 Logging

- Built on Go's standard `log/slog` (structured, leveled)
- Dual output: stderr (for operator visibility) + daily-rotated file (for audit)
- Domain-specific helper functions: `CaptureEvent()`, `ReplayEvent()`, `DiffEvent()`, `DBHookEvent()`, `SessionEvent()`
- Each event carries a structured `event` key plus domain-specific attributes (session ID, sequence, method, path, duration, etc.)
- Log directory: `~/.shadiff/logs/`

### 5.2 Configuration

- JSON-based config file at `~/.shadiff/config.json`
- Thread-safe `config.Store` with `Load()`, `Save()`, `Get()`, `Update(fn)` API
- Sensible defaults via `DefaultConfig()`: proxy listen on `:18080`, 10MB max body, 30s replay timeout, concurrency 1, 1000 max diffs
- Default ignored headers: `Date`, `X-Request-Id`, `X-Trace-Id`, `Server`, `Content-Length`
- Supports per-command flag overrides (flags take precedence over config file)

### 5.3 Error Handling

- All commands return `error` from `RunE` handlers; Cobra prints and exits with code 1
- Errors are wrapped with `fmt.Errorf("context: %w", err)` for stack-traceable chains
- Storage operations skip corrupted entries silently (e.g., malformed JSONL lines, unreadable session directories) to maintain resilience
- DB hook channel overflow is handled via non-blocking send with a warning log, preventing back-pressure from blocking the proxy pipeline
- Replay errors are collected per-record and summarized; they do not abort the entire replay

### 5.4 Concurrency

- `storage.FileStore` uses `sync.RWMutex` for thread-safe reads/writes
- `capture.Recorder` uses `sync.Mutex` for pending side-effect aggregation, with a background goroutine draining the side-effect channel
- `replay.WorkerPool` uses goroutine fan-out with `sync.WaitGroup` for concurrent replay
- DB hook proxies spawn per-connection goroutines with bidirectional `io.Copy` + sniffing
- Buffered channels (capacity 1000) between DB hooks and the recorder prevent tight coupling

### 5.5 Session Lifecycle

```
recording  -->  completed  -->  replayed
   (record)       (record done)    (replay done)
```

Sessions are identified by 8-character UUID short IDs and can be resolved by ID or name.

### 5.6 Extensibility Points

- `storage.SessionStore` / `RecordStore` / `DiffStore` interfaces allow alternative storage backends
- `dbhook.DBHook` interface allows additional database protocol support
- `reporter.Reporter` interface allows additional output formats
- `diff.Matcher` interface allows custom comparison logic
- `diff.Rule` system supports `ignore` and `custom` rule kinds with glob-style path matching
