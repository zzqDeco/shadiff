# Shadiff Implementation Mapping

## Overview

This document maps every package, source file, and key implementation pattern in the Shadiff codebase. Shadiff follows a standard Go project layout with `cmd/` for CLI wiring and `internal/` for all business logic.

---

## 1. Module Responsibility Table

| Package | Import Path | Responsibility |
|---------|-------------|----------------|
| `main` | `shadiff` | Entry point; delegates to `cmd.Execute()` |
| `cmd` | `shadiff/cmd` | CLI command definitions (cobra), flag parsing, orchestration of record/replay/diff/report workflows |
| `capture` | `shadiff/internal/capture` | HTTP reverse proxy and unified recorder that captures request/response pairs and database side effects |
| `dbhook` | `shadiff/internal/capture/dbhook` | Database protocol proxies (MySQL, PostgreSQL, MongoDB, Redis) that sniff wire protocols to extract queries, operations, and commands |
| `config` | `shadiff/internal/config` | Application configuration schema, defaults, and thread-safe persistent config store |
| `diff` | `shadiff/internal/diff` | Diff engine: semantic comparison of recorded vs. replayed records (status codes, headers, JSON bodies, SQL queries, MongoDB operations, Redis commands) with rule-based filtering |
| `logger` | `shadiff/internal/logger` | Global structured logger (slog) with daily-rotated file output and domain-specific convenience methods |
| `model` | `shadiff/internal/model` | Data model types: Session, Record, HTTPRequest, HTTPResponse, SideEffect, DiffResult, Difference, DiffSummary |
| `replay` | `shadiff/internal/replay` | Replay engine with configurable worker pool, request transformation, and concurrent HTTP replay |
| `reporter` | `shadiff/internal/reporter` | Report generation in terminal (ANSI), JSON, and HTML formats |
| `storage` | `shadiff/internal/storage` | Storage interfaces and filesystem-based implementation using JSONL for records and JSON for metadata/results |
| `daemon` | `shadiff/internal/daemon` | Daemon process management: PID file operations (write, read, remove, check liveness), platform-specific process detach (Unix Setsid / Windows CREATE_NEW_PROCESS_GROUP), process alive checking, signal sending (stop/force kill) |

---

## 2. File Listing Table

### Root

| File | Description |
|------|-------------|
| `main.go` | Program entry point; calls `cmd.Execute()` and exits with code 1 on error |

### `cmd/` -- CLI Commands

| File | Description |
|------|-------------|
| `root.go` | Root cobra command (`shadiff`); defines global flags (`--config`, `--verbose`, `--quiet`) and initializes runtime config in `PersistentPreRunE` |
| `version.go` | `shadiff version` command; prints build-time injected Version, Commit, BuildDate |
| `runtime.go` | Shared runtime context for config path, loaded config, data directory, log directory, and flag-over-config precedence helpers |
| `dbproxy.go` | Helper logic for record/replay DB proxy parsing, config fallback for capture and replay, hook startup, grouped side-effect fan-in, and flush-aware shutdown |
| `record.go` | `shadiff record` command; resolves config-backed capture settings, starts HTTP reverse proxy plus DB hooks, wires DB-hook flushing into capture, and supports `--daemon` mode via self-re-exec |
| `record_stop.go` | `shadiff record stop` subcommand; stops a daemon recording session by sending signals (SIGTERM/os.Interrupt), with graceful wait and force kill fallback; includes `findSession()` helper for ID/name resolution |
| `record_status.go` | `shadiff record status` subcommand; lists all active recording sessions or shows detailed status for a specific session including PID, process liveness, record count, and uptime |
| `replay.go` | `shadiff replay` command; resolves session, starts optional replay DB hooks, creates replay engine, executes replay, prints summary; also contains `resolveSession()` helper |
| `diff.go` | `shadiff diff` command; creates diff engine, runs comparison, and renders either terminal output or JSON output |
| `report.go` | `shadiff report` command; loads saved diff results, creates reporter by format, writes output to file or stdout |
| `session.go` | `shadiff session` parent command with `list`, `show`, `delete` subcommands; contains `getStore()` helper |

### `internal/daemon/` -- Daemon Process Management

| File | Description |
|------|-------------|
| `pidfile.go` | PID file management; `WritePID`/`ReadPID`/`RemovePID`/`IsRunning`/`PIDFilePath` for daemon session tracking |
| `process_unix.go` | Unix-specific process operations (build tag `!windows`); `Detach` (Setsid), `isProcessAlive` (signal 0), `SendStop` (SIGTERM), `ForceKill` (SIGKILL) |
| `process_windows.go` | Windows-specific process operations (build tag `windows`); `Detach` (CREATE_NEW_PROCESS_GROUP), `isProcessAlive` (OpenProcess+GetExitCodeProcess), `SendStop` (os.Interrupt), `ForceKill` (Kill) |

### `internal/capture/` -- Traffic Capture

| File | Description |
|------|-------------|
| `proxy.go` | HTTP reverse proxy (`httputil.ReverseProxy` wrapper); captures request/response pairs with `responseRecorder`; assigns sequence numbers via `atomic.Int64` |
| `recorder.go` | Unified recorder; manages request scopes, attributes side effects from the channel by timestamp, persists via `FileStore.AppendRecord()`, and exposes request-body artifact persistence for the proxy |
| `recorder_test.go` | Tests for the Recorder |

### `internal/dbtype/` -- Database Type Registry

| File | Description |
|------|-------------|
| `dbtype.go` | Canonical supported DB proxy type constants plus `Supported`, `IsSupported`, and `Names` helpers used by config, dbhook, and diff |

### `internal/capture/dbhook/` -- Database Protocol Proxies

| File | Description |
|------|-------------|
| `hook.go` | `DBHook` interface definition, hook flush coordination, `Group` fan-in/drain helper, `Config` struct, constructor registry-backed `NewHook()`, `UnsupportedDBError` type |
| `tcp_proxy.go` | Shared transparent TCP proxy lifecycle for DB hooks: listener, connection tracking, bidirectional forwarding, flush barriers, and side-effect emission |
| `mysql.go` | MySQL stream parser; parses COM_QUERY/COM_STMT_PREPARE from buffered MySQL packet frames |
| `postgres.go` | PostgreSQL stream parser; handles startup phase and parses frontend Simple Query (`Q`) and Parse (`P`) messages |
| `mongo.go` | MongoDB stream parser; parses OP_MSG wire protocol (opcode 2013); extracts CRUD commands via simplified BSON-to-map parsing; includes `MongoCommandToJSON()` helper |
| `redis.go` | Redis stream parser; parses RESP array and inline commands; captures command, primary key, and redacted/encoded args |
| `hook_test.go` | Tests for DBHook implementations |
| `mongo_parse_test.go` | Tests for MongoDB BSON parsing logic |

### `internal/config/` -- Configuration

| File | Description |
|------|-------------|
| `config.go` | Configuration schema types (`AppConfig`, `CaptureConfig`, `DBProxyConfig`, `ReplayConfig`, `DiffConfig`, `Rule`, `StorageConfig`, `LogConfig`); `DefaultConfig()` returns sensible defaults |
| `store.go` | Thread-safe config store (`Store`); loads from / saves to the configured `config.json`, provides `Get()`, `Update(fn)`, and `DataDir()` methods |
| `validate.go` | Startup validation for config values such as replay timeout, log level, diff limits, and DB proxy declarations |
| `config_test.go` | Tests for configuration loading and defaults |

### `internal/diff/` -- Diff Engine

| File | Description |
|------|-------------|
| `engine.go` | Core diff engine; loads recorded and replayed records, pairs by sequence number, reports replay failures, compares status codes / headers / bodies / side effects through the comparer registry; saves results via `DiffStore` |
| `sideeffects.go` | Side-effect comparer registry and residual side-effect count comparison |
| `json.go` | `JSONDiffer` struct; recursive structural JSON comparison (objects, arrays, primitives); supports ordered and unordered array comparison with best-match pairing |
| `rule_loader.go` | Converts config rules to diff rules and loads external rule files from JSON or YAML |
| `rules.go` | `Rule`, `RuleSet`, `Matcher` interface; path-wildcard-to-regexp compilation; built-in matchers (Timestamp, UUID, NumericTolerance); `DefaultRules()`, `DefaultIgnoreHeaders()`, `FormatDiffSummary()`, `FormatPath()` |
| `db.go` | `CompareDBSideEffects()` for SQL databases (MySQL/PostgreSQL); normalizes SQL (whitespace, case) before comparison; `filterByType()` helper |
| `mongo.go` | `CompareMongoSideEffects()` for MongoDB; compares operation type, collection, and database name |
| `redis.go` | `CompareRedisSideEffects()` for Redis; compares command count, command name, primary key, and arguments |
| `json_test.go` | Tests for JSON structural diff |
| `db_test.go` | Tests for SQL side effect comparison |
| `sideeffects_test.go` | Tests for comparer registry coverage, SQL/MongoDB/Redis dispatch, and residual side-effect counting |
| `rule_loader_test.go` | Tests for JSON/YAML rule loading and config-to-diff rule conversion |
| `mongo_test.go` | Tests for MongoDB side effect comparison |
| `rules_test.go` | Tests for rule matching and path wildcard compilation |

### `internal/logger/` -- Logging

| File | Description |
|------|-------------|
| `logger.go` | Global slog-based logger; writes to stderr + daily-rotated file (`shadiff-YYYY-MM-DD.log`); domain convenience methods: `CaptureEvent`, `ReplayEvent`, `DiffEvent`, `DBHookEvent`, `SessionEvent`, `Error`, `Debug`, `Info`, `Warn` |

### `internal/model/` -- Data Models

| File | Description |
|------|-------------|
| `session.go` | `Session` struct (ID, Name, Status, Source/Target endpoints, Tags, Metadata, timestamps); `SessionStatus` enum (`recording`, `completed`, `replayed`); `EndpointConfig`; `SessionFilter` |
| `record.go` | `Record` struct (ID, SessionID, Sequence, Request, Response, SideEffects, Duration, RecordedAt, Error) |
| `request.go` | `HTTPRequest` struct (Method, Path, Query, Headers, Body, BodyLen); `HTTPResponse` struct (StatusCode, Headers, Body, BodyLen) |
| `sideeffect.go` | `SideEffect` envelope with typed payloads: `DatabaseSideEffect` (`SQL`, `Mongo`, `Redis`) and `HTTPSideEffect`; includes constructors and safe accessors |
| `diff.go` | `DiffResult` struct; `DifferenceKind` enum (status_code, header, body, body_field, db_query, db_query_count, mongo_op, redis_command, redis_command_count, external_call); `Severity` enum; `Difference` struct; `DiffSummary` struct |
| `model_test.go` | Tests for model types, including the v0.4 typed side-effect JSON payload contract |

### `internal/replay/` -- Replay Engine

| File | Description |
|------|-------------|
| `engine.go` | Replay engine; reads recorded records from storage, executes replay, flushes replay DB hooks before request-window attribution, and saves replay records to `replay-records.jsonl` |
| `worker.go` | `WorkerPool` struct; concurrent replay with configurable worker count and inter-request delay; `replayOne()` sends a single HTTP request, preferring stored full-body artifacts when present, and captures the response as a new `Record` |
| `transform.go` | `TransformConfig`, `Transform()`, and `TransformWithBody()`; rewrites recorded requests for the replay target (URL, headers, proxy header removal) and supports artifact-backed request bodies |
| `transform_test.go` | Tests for request transformation |

### `internal/reporter/` -- Report Generation

| File | Description |
|------|-------------|
| `reporter.go` | `Reporter` interface; `NewReporter(format)` factory function dispatching to terminal/json/html |
| `terminal.go` | `TerminalReporter`; ANSI-colored output with unicode tree connectors; color-coded severity levels |
| `json.go` | `JSONReporter`; outputs `{ "summary": ..., "results": [...] }` with indentation |
| `html.go` | `HTMLReporter`; self-contained HTML page with embedded CSS via Go `html/template`; responsive card layout with color-coded match/diff status |
| `reporter_test.go` | Tests for reporter implementations, including SQL/MongoDB/Redis side-effect difference output |

### `internal/storage/` -- Persistence

| File | Description |
|------|-------------|
| `store.go` | Interface definitions: `SessionStore`, `RecordStore`, `DiffStore` |
| `filestore.go` | `FileStore` implementation; filesystem layout `{baseDir}/sessions/{id}/`; mutex-protected reads/writes; JSONL append for records; JSON for sessions and diff results; tag-based filtering |
| `filestore_test.go` | Tests for FileStore |

---

## 3. Key Implementation Patterns

### 3.1 Reverse Proxy with Response Capture

**Location**: `internal/capture/proxy.go`

The `Proxy` struct wraps Go's `net/http/httputil.ReverseProxy`. To capture both the forwarded response and still write it to the original client, it uses a custom `responseRecorder` that wraps `http.ResponseWriter`:

```
Client  --->  Proxy.ServeHTTP()  --->  ReverseProxy  --->  Target Service
                    |                                            |
                    |  <-- responseRecorder intercepts <----------+
                    |      WriteHeader() and Write()
                    v
       Recorder.BeginRequestScope()/FinishRequestScope()
                    |
                    v
              FileStore.AppendRecord()
```

- Excluded paths are checked before request capture begins.
- Included request bodies are fully snapshotted before proxying, keep only the configured inline preview in `Request.Body`, and store replayable full-body artifacts when the preview is truncated.
- Included requests open a recorder scope before proxying and, when DB hooks are configured, flush hook-delivered side effects before the scope is closed so in-window effects reach the recorder sink first.
- `responseRecorder.Write()` tees data: it writes to both its internal `bytes.Buffer` (for capture) and the underlying `ResponseWriter` (for the client).
- `responseRecorder.WriteHeader()` uses a `wroteHeader` guard to prevent double writes.
- Sequence numbers are assigned atomically via `atomic.Int64.Add(1)`.

### 3.2 Protocol Sniffing (TCP Man-in-the-Middle)

**Location**: `internal/capture/dbhook/tcp_proxy.go`, `mysql.go`, `postgres.go`, `mongo.go`, `redis.go`

All database hooks share the same lifecycle through `tcpProxy`:

1. **TCP Listener**: Accept incoming connections on the proxy listen address.
2. **Bidirectional Forwarding**: For each connection, establish a connection to the real database server and set up two goroutines:
   - **Server-to-Client**: Plain `io.Copy` passthrough (no inspection needed for server responses).
   - **Client-to-Server**: Read data, forward it to the server, then feed the same bytes into the protocol parser.
3. **Protocol Parsing**: Each hook supplies a stream parser:
   - **MySQL**: 3-byte little-endian length + 1-byte sequence number + command byte. Captures `COM_QUERY` (0x03) and `COM_STMT_PREPARE` (0x16).
   - **PostgreSQL**: 1-byte message type + 4-byte big-endian length. Captures Simple Query (`Q`) and Extended Query Parse (`P`). Handles the startup phase (no type byte) separately.
   - **MongoDB**: 16-byte wire protocol header (4-byte LE message length, request ID, response-to, opcode). Captures OP_MSG (opcode 2013) with BSON document body parsing.
   - **Redis**: RESP arrays of bulk strings and inline commands. Captures normalized command, primary key, and redacted/encoded arguments.
4. **Side Effect Emission**: Parsed queries/commands are wrapped in typed `model.SideEffect` payloads and sent on a buffered channel (capacity 1000). Channel-full drops are logged as warnings.

### 3.3 Worker Pool for Concurrent Replay

**Location**: `internal/replay/worker.go`

The `WorkerPool` supports two execution modes:

- **Sequential** (concurrency = 1): Iterates over records in order with optional inter-request delay.
- **Concurrent** (concurrency > 1): Creates a channel of job indices, spawns N worker goroutines, each pulls indices from the channel, executes `replayOne()`, and writes results to a pre-allocated results slice (no lock needed since each worker writes to a distinct index).

```
records[0..N]  -->  jobs channel  -->  worker goroutines (N)  -->  results[0..N]
                                            |
                                            v
                                    HTTP client.Do(req)
```

Each `replayOne()` call:
1. Transforms the recorded request via `Transform()`.
2. Sends it with a shared `http.Client` (with configurable timeout).
3. Captures the response into a new `model.Record` preserving the original sequence number.
4. On failure, populates `ReplayResult.Error` and creates a partial record with the error message.

### 3.4 JSONL Streaming for Record Persistence

**Location**: `internal/storage/filestore.go`

Records use JSONL (JSON Lines) format rather than a single JSON array for two reasons:

1. **Append-only writes**: Each record is atomically appended as a single line. No need to read-modify-write the entire file. File is opened with `O_CREATE|O_APPEND|O_WRONLY`.
2. **Streaming reads**: Records can be read line-by-line with bounded memory. The scanner buffer is set to max 10 MB per line to handle large response bodies.

Corruption tolerance: if a line fails `json.Unmarshal`, it is silently skipped. This prevents a single corrupted record from breaking the entire session.

Thread safety: all file operations are protected by a `sync.RWMutex`. Reads use `RLock`, writes use `Lock`.

### 3.5 Structured JSON Diff with Recursive Descent

**Location**: `internal/diff/json.go`

The `JSONDiffer` performs recursive structural comparison of two JSON values:

1. **Type check**: If Go types differ, attempt numeric coercion (JSON unmarshal may mix `int` and `float64`); otherwise report a type mismatch.
2. **Object comparison**: Union all keys from both objects, sort for deterministic output, then recurse on each key. Missing keys produce "field missing" (error) or "extra field" (warning) differences.
3. **Array comparison**:
   - **Ordered mode** (default): Compare element-by-element up to the shorter length. Length difference is reported separately.
   - **Unordered mode** (`IgnoreOrder = true`): For each expected element, search for a matching actual element (zero differences). Unmatched expected elements produce errors; unmatched actual elements produce warnings.
4. **Path tracking**: Every difference includes a dot-separated JSON path (e.g., `body.data.items[0].name`) built by `FormatPath()`.

### 3.6 Rule Engine with Path Wildcards

**Location**: `internal/diff/rules.go`

Rules are evaluated after raw differences are computed:

1. **Path patterns** support three wildcards:
   - `*` -- matches a single path level (compiled to `[^.]*`)
   - `**` -- matches any number of levels (compiled to `.*`)
   - `[*]` -- matches any array index (compiled to `\[\d+\]`)
2. Patterns are pre-compiled to `*regexp.Regexp` at `RuleSet` construction time.
3. Two rule kinds:
   - `"ignore"` -- unconditionally marks matching differences as ignored.
   - `"custom"` -- delegates to a named `Matcher` implementation; marks ignored only if `Match()` returns `true`.
4. Default rules ignore timestamp fields (`**.createdAt`, `**.updatedAt`, etc.) and request-tracking headers (`X-Request-Id`, `X-Trace-Id`, `Date`, `Server`).

### 3.7 Session Resolution

**Location**: `cmd/replay.go` (`resolveSession()`)

Sessions can be referenced by ID or name. The resolution strategy is:

1. Try `store.Get(nameOrID)` -- treats the input as an ID.
2. On failure, `store.List(&SessionFilter{Name: nameOrID})` -- searches by name substring.
3. If multiple sessions match, the most recent (first in the list, sorted by `UpdatedAt` desc) is used with a warning printed.
4. If no session matches, return an error.

This function is reused by `replay`, `diff`, and `report` commands.

### 3.8 Configuration Layering

**Location**: `cmd/runtime.go`, `internal/config/config.go`, `internal/config/store.go`, `internal/config/validate.go`

Configuration follows a layered approach:

1. `DefaultConfig()` provides sensible defaults (listen on `:18080`, 10 MB max body, 1 concurrency, 30s timeout, standard ignore headers).
2. `Store.Load()` reads the selected config file and unmarshals on top of defaults (fields not present in the file retain default values).
3. `config.Validate()` rejects unsupported values before command execution starts.
4. `cmd/runtime.go` derives `DataDir` and `LogDir`, then exposes helper functions so each command can apply `CLI flag > config.json > default`.

The `Store` is thread-safe (`sync.RWMutex`) and supports atomic updates via `Update(fn func(*AppConfig))`.

### 3.9 Filesystem Storage Layout

**Location**: `internal/storage/filestore.go`

```
~/.shadiff/
  config.json                          -- Application configuration
  logs/
    shadiff-2024-01-15.log             -- Runtime log file
  sessions/
    {sessionID}/
      session.json                     -- Session metadata
      records.jsonl                    -- Recorded request/response pairs
      replay-records.jsonl             -- Replayed request/response pairs
      diff-results.json                -- Diff comparison results
```

- Session IDs are 8-character UUID prefixes (generated by `uuid.New().String()[:8]`).
- `session.json` uses pretty-printed JSON (`json.MarshalIndent`).
- Record files use JSONL (one JSON object per line).
- Diff results use a pretty-printed JSON array.
- `Delete()` removes the entire session directory via `os.RemoveAll`.

### 3.10 Daemon Process Management

**Location**: `cmd/record.go`, `internal/daemon/`

Shadiff supports running the recording proxy as a background daemon using a **self-re-exec pattern**:

1. **Parent process** (`--daemon` flag):
   - Creates the session and session directory.
   - Resolves effective DB proxies before spawning the child.
   - Determines the current executable path via `os.Executable()`.
   - Re-execs itself with `--_daemon-child --session {sessionID}` plus all effective flags, including `--config`, `--duration`, and `--db-proxy`.
   - Calls `daemon.Detach(cmd)` to configure platform-specific process group detach:
     - **Unix**: `SysProcAttr.Setsid = true` -- new session, detached from terminal.
     - **Windows**: `SysProcAttr.CreationFlags = CREATE_NEW_PROCESS_GROUP` -- new process group.
   - Redirects child stdout/stderr to `{sessionDir}/daemon.log`.
   - Starts the child via `cmd.Start()`, writes the child PID to `{sessionDir}/pidfile`.
   - Updates the session with the PID and exits.

2. **Child process** (`--_daemon-child` flag):
   - Loads the pre-created session from storage by ID.
   - Updates the session PID to its own `os.Getpid()`.
   - Runs the normal recording loop (HTTP proxy, DB hooks, signal handling, graceful shutdown).
   - On exit, removes the PID file via `defer daemon.RemovePID()`.

3. **PID file lifecycle**:
   - Written by parent after child starts successfully.
   - Read by `record stop` and `record status` to find the daemon process.
   - Removed by child on graceful exit.
   - Stale PID files (process dead) are detected by checking process liveness and cleaned up by `record stop`.
   - Location: `~/.shadiff/sessions/{sessionID}/pidfile`.
