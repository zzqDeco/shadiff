# Shadiff -- Architecture Overview

## 1. Core Workflow

Shadiff implements a four-stage pipeline for behavioral comparison of API services:

```
 +---------+     +----------+     +--------+     +----------+
 | Record  | --> |  Replay  | --> |  Diff  | --> |  Report  |
 +---------+     +----------+     +--------+     +----------+
     |                |               |               |
  Captures         Sends           Compares        Generates
  old API          recorded         recorded vs.    terminal/
  behavior         requests to      replayed        JSON/HTML
  (req/resp/       new API          behavior        output
   side effects)
```

Each stage reads from and writes to a session directory in `~/.shadiff/sessions/{sessionID}/`.
Stages are decoupled -- they can be run independently, at different times, even on different machines
(by copying the session directory).

---

## 2. Data Flow

### 2.1 Record Phase

```
                          +---------------------+
  Client                  |    Shadiff Proxy     |                Target Service
  (browser,  ----------> |  (HTTP reverse proxy) | ------------> (old API)
   curl,                  |   :18080              |               :8080
   tests)   <------------ |                       | <------------
                          |   captures req/resp   |
                          +----------+------------+
                                     |
                          +----------v------------+
                          |      Recorder         |
                          |  (aggregates records  |
                          |   + side effects)     |
                          +----------+------------+
                                     |
                          +----------v------------+
                          |    FileStore          |
                          |  (appends to JSONL)   |
                          +----------+------------+
                                     |
                    ~/.shadiff/sessions/{id}/records.jsonl
```

**Detailed flow:**

1. Client sends HTTP request to Shadiff's listening address (default `:18080`).
2. `capture.Proxy` (wrapping `httputil.ReverseProxy`) intercepts the request:
   - Reads and buffers the request body.
   - Builds a `model.HTTPRequest` struct.
   - Forwards the request to the target service via the reverse proxy.
   - Wraps the `ResponseWriter` with a `responseRecorder` to capture status code, headers, and body.
3. After the response completes, the proxy builds a `model.Record` containing:
   - The `HTTPRequest` and `HTTPResponse`
   - A monotonically increasing sequence number (atomic counter)
   - Duration in milliseconds
   - An 8-character UUID as record ID
4. The record is passed to `capture.Recorder.Record()`.
5. The Recorder attaches any pending side effects (collected from DB hooks), then calls `FileStore.AppendRecord()`.
6. `FileStore` serializes the record as a single JSON line and appends it to `records.jsonl` with file-level mutex protection.

### 2.2 Side-Effect Capture (DB Hooks)

```
  Target Service
       |
       | SQL / Mongo queries
       v
  +-----------+       +----------------+       +-----------+
  |  App DB   | <---> |  DB Hook Proxy | <---> |  Real DB  |
  |  Client   |       |  (TCP level)   |       |  Server   |
  +-----------+       +-------+--------+       +-----------+
                              |
                     side-effect channel
                      (buffered, cap 1000)
                              |
                              v
                       +------+------+
                       |  Recorder   |
                       | (pending    |
                       |  effects)   |
                       +-------------+
```

DB hooks operate as TCP proxies that sit between the application and the real database. They:

1. Accept connections from the application's DB client.
2. Establish a connection to the real database server.
3. Forward all traffic bidirectionally (`io.Copy` for server-to-client).
4. Sniff client-to-server traffic to extract query information.
5. Emit `model.SideEffect` structs on a buffered channel.
6. The `Recorder`'s background goroutine drains this channel into a `pendingEffects` slice.
7. When the next HTTP record is saved, pending effects are attached to it.

### 2.3 Replay Phase

```
  ~/.shadiff/sessions/{id}/records.jsonl
                     |
                     v
              +------+------+
              | Replay      |
              | Engine      |
              +------+------+
                     |
              +------v------+
              | Worker Pool |  (1..N goroutines)
              +------+------+
                     |
              Transform(req) -- URL rewrite, header manipulation
                     |
                     v
              +------+------+
              | New Target  |
              | Service     |
              +-------------+
                     |
                     v
              +------+------+
              | FileStore   |
              +------+------+
                     |
      ~/.shadiff/sessions/{id}/replay-records.jsonl
```

**Detailed flow:**

1. `replay.Engine.Run()` loads all records from `records.jsonl`.
2. Records are dispatched to the `WorkerPool`:
   - Sequential mode (concurrency=1): records are replayed one by one with optional delay.
   - Concurrent mode (concurrency>1): records are distributed to N worker goroutines via a buffered job channel.
3. For each record, `replay.Transform()` converts `model.HTTPRequest` into a Go `*http.Request`:
   - Rewrites the base URL to the new target.
   - Copies original headers, applies overrides, removes proxy headers (`X-Forwarded-*`).
4. The worker sends the request via `http.Client` and captures the response.
5. A new `model.Record` is built for the replay result (preserving the original sequence number for pairing).
6. Results are saved to `replay-records.jsonl`.
7. Session status is updated to `replayed`.

### 2.4 Diff Phase

```
  records.jsonl          replay-records.jsonl
       |                         |
       v                         v
  +----+----+              +-----+----+
  | originals|              |  replays  |
  +----+-----+              +-----+----+
       |                          |
       +--------+     +-----------+
                |     |
           +----v-----v----+
           |  Diff Engine  |
           |  (pair by     |
           |   sequence #) |
           +-------+-------+
                   |
     +-------------+-------------+
     |             |             |
  status code   headers       body (JSON)
  comparison   comparison    recursive diff
                   |
              +----v-----+
              | Rule Set |  (ignore/custom matchers)
              +----+-----+
                   |
                   v
           diff-results.json
```

**Detailed flow:**

1. `diff.Engine.Run()` loads both `records.jsonl` (originals) and `replay-records.jsonl` (replays).
2. Replay records are indexed by sequence number into a map for O(1) pairing.
3. For each original record, the engine finds its replay counterpart and calls `compareRecords()`.
4. Comparison proceeds in four layers:
   - **Status code**: direct integer comparison. Severity: `error`.
   - **Response headers**: key-by-key comparison, skipping ignored headers (Date, X-Request-Id, etc.). Severity: `warning`.
   - **Response body**: `JSONDiffer.Compare()` performs recursive structural comparison:
     - Object diff: field-by-field, detects missing/extra fields.
     - Array diff: ordered (index-by-index) or unordered (best-match pairing).
     - Scalar diff: type-aware with numeric type coercion.
     - Non-JSON bodies fall back to byte-level comparison.
   - **Side effects**: count comparison for DB operations.
5. The `RuleSet` is applied to all differences:
   - `ignore` rules: mark matching paths as ignored (e.g., timestamp fields).
   - `custom` rules: invoke `Matcher` implementations (timestamp, UUID, numeric tolerance).
   - Path matching supports glob wildcards (`*` single level, `**` multi-level, `[*]` array index).
6. A record is considered a `match` only if all non-ignored differences are empty.
7. Results are saved to `diff-results.json`.

### 2.5 Report Phase

```
  diff-results.json
        |
        v
  +-----+------+
  |  Reporter  |
  +-----+------+
        |
  +-----+------+------+
  |            |       |
  Terminal    JSON    HTML
  (colored)  (struct) (standalone page)
```

The reporter loads saved diff results, computes summary statistics (`DiffSummary`), and delegates to format-specific implementations:

- **TerminalReporter**: ANSI-colored output with tree-style diff display and summary stats.
- **JSONReporter**: machine-readable JSON with `summary` and `results` fields.
- **HTMLReporter**: self-contained HTML page with CSS styling, summary cards, and per-record diff details.

---

## 3. Database Protocol Proxying

All three DB hooks follow the same architectural pattern: **transparent TCP proxy with client-to-server sniffing**.

### 3.1 MySQL Hook (`MySQLHook`)

- **Protocol**: MySQL Client/Server Protocol (binary)
- **Packet format**: 3-byte length (little-endian) + 1-byte sequence number + payload
- **Captured commands**:
  - `COM_QUERY` (0x03): direct SQL text
  - `COM_STMT_PREPARE` (0x16): prepared statement SQL text
- **Extraction**: payload bytes after the command byte are interpreted as the SQL string
- **Limitation**: `COM_STMT_EXECUTE` (0x17) is recognized but parameter binding is not decoded (only the prepared SQL is captured at prepare time)

### 3.2 PostgreSQL Hook (`PostgresHook`)

- **Protocol**: PostgreSQL Frontend/Backend Protocol v3 (binary)
- **Message format**: 1-byte message type + 4-byte length (big-endian, includes self) + payload
- **Startup handling**: the first message (StartupMessage) has no type byte and is skipped
- **Captured messages**:
  - `'Q'` (Simple Query): null-terminated SQL string
  - `'P'` (Parse / Extended Query): statement name (null-terminated) + SQL string (null-terminated)
- **Multi-message parsing**: the parser iterates through concatenated messages in a single TCP read buffer

### 3.3 MongoDB Hook (`MongoHook`)

- **Protocol**: MongoDB Wire Protocol (binary, little-endian)
- **Message format**: 16-byte header (4-byte length + 4-byte requestID + 4-byte responseTo + 4-byte opCode) + body
- **Captured opcode**: `OP_MSG` (2013) only (modern MongoDB 3.6+)
- **OP_MSG parsing**:
  - 4-byte flagBits, then sections
  - Section kind 0 (body): single BSON document containing the command
  - Section kind 1 (document sequence): skipped (length-delimited)
- **BSON parsing**: simplified in-house parser (`simpleBSONToMap`) that extracts string, int32, int64, boolean, null, sub-document, and ObjectId types without the full `bson` library
- **Recognized CRUD commands**: `find`, `insert`, `update`, `delete`, `aggregate`, `count`, `distinct`, `findAndModify`
- **Extracted fields**: `$db` (database), collection name (from command value), `filter`, `updates`, `documents`

### 3.4 Common Hook Architecture

```
  net.Listen(listenAddr)
        |
        v
  for { Accept() }
        |
        v (per connection)
  +-----------------------+
  | Dial targetAddr       |
  +-----------------------+
        |
   +----+----+
   |         |
  server    client
  ->client  ->server
  (io.Copy) (sniff + forward)
   |         |
   +----+----+
        |
  emit SideEffect on buffered channel
```

- Each hook has a `done` channel for graceful shutdown.
- `Stop()` closes `done`, closes the listener, waits on `WaitGroup`, then closes the side-effect channel.
- Channel capacity is 1000; overflow drops the event with a warning log (non-blocking send).
- All hooks satisfy the `dbhook.DBHook` interface: `Start(ctx)`, `Stop()`, `SideEffects()`, `Type()`.
- The factory function `dbhook.NewHook(Config)` routes to the correct implementation by `DBType`.

---

## 4. Storage Format

### 4.1 Directory Structure

```
~/.shadiff/
  config.json                          # Global configuration
  logs/
    shadiff-2024-01-15.log             # Daily-rotated log files
  sessions/
    {sessionID}/                       # 8-char UUID prefix
      session.json                     # Session metadata (JSON, pretty-printed)
      records.jsonl                    # Recorded request/response pairs (JSONL)
      replay-records.jsonl             # Replayed request/response pairs (JSONL)
      diff-results.json                # Diff comparison results (JSON array, pretty-printed)
```

### 4.2 JSONL Streaming Format

Records are stored as JSONL (JSON Lines) -- one complete JSON object per line, newline-delimited. This format was chosen for:

- **Append-only writes**: new records are appended without reading/rewriting the entire file.
- **Streaming reads**: records can be read line-by-line with constant memory overhead per line.
- **Crash resilience**: a partial write only corrupts the last line; the scanner skips malformed lines.
- **Large file handling**: the `bufio.Scanner` is configured with a 10MB per-line buffer to handle large request/response bodies.

Each JSONL line contains a complete `model.Record`:

```json
{"id":"a1b2c3d4","sessionID":"e5f6g7h8","sequence":1,"request":{...},"response":{...},"sideEffects":[...],"duration":42,"recordedAt":1705312345678}
```

### 4.3 Session Metadata

`session.json` is a pretty-printed JSON file containing the `model.Session` struct:

```json
{
  "id": "a1b2c3d4",
  "name": "user-module-migration",
  "description": "",
  "source": {"baseURL": "http://localhost:8080", "headers": {}},
  "target": {"baseURL": "http://localhost:9090", "headers": {}},
  "tags": [],
  "recordCount": 150,
  "createdAt": 1705312345678,
  "updatedAt": 1705312999999,
  "status": "replayed",
  "metadata": {}
}
```

### 4.4 Diff Results

`diff-results.json` is a pretty-printed JSON array of `model.DiffResult` objects. This uses standard JSON (not JSONL) because diff results are written atomically after the full comparison completes, and they are typically smaller than record files.

---

## 5. Key Design Decisions and Tradeoffs

### 5.1 Black-Box Approach

**Decision**: Shadiff treats both old and new services as opaque black boxes. It does not require source code access, instrumentation, or framework-specific adapters.

**Tradeoff**: This limits side-effect capture to what can be observed at the network level. In-process events (e.g., cache writes, message queue publishes) are invisible unless they traverse a network protocol that Shadiff can proxy.

### 5.2 TCP-Level Protocol Proxying (vs. Driver-Level Hooking)

**Decision**: Database operations are captured via TCP protocol proxying rather than database driver middleware.

**Benefit**: Language-agnostic. Works with any application regardless of programming language, ORM, or driver. No code changes required in the target service.

**Tradeoff**: Protocol parsing is inherently fragile. The simplified BSON parser covers common cases but cannot handle all BSON types. TLS-encrypted database connections would require additional TLS termination/re-encryption logic (not currently supported). Packet fragmentation across TCP reads may cause missed queries in edge cases.

### 5.3 Sequence-Based Record Pairing

**Decision**: Records are paired between original and replay by sequence number (monotonic counter), not by request content matching.

**Benefit**: Simple, deterministic, O(1) lookup. No ambiguity when multiple requests share the same path/method.

**Tradeoff**: Requires replay to preserve request order. If a request fails during replay (and is skipped or retried), the sequence alignment can drift. The current implementation handles missing replay records gracefully (reports them as "missing") but does not attempt re-alignment.

### 5.4 JSONL for Records, JSON for Diff Results

**Decision**: Records use append-only JSONL; diff results use atomic JSON writes.

**Rationale**: During recording, data arrives incrementally over potentially long durations -- JSONL supports crash-safe appending. Diff results are computed in a single batch and are typically much smaller -- standard JSON is simpler to read/write atomically and is easier to consume by downstream tools.

### 5.5 File-System Storage (vs. Database)

**Decision**: All data is stored on the local file system with no external database dependency.

**Benefit**: Zero-dependency deployment. A single binary with no setup. Sessions are portable (copy the directory).

**Tradeoff**: No indexing, no concurrent multi-process access, no built-in retention policy beyond `maxSessions` config. File-level mutex serializes all writes within a single process.

### 5.6 Simplified BSON Parser

**Decision**: MongoDB wire protocol parsing uses an in-house simplified BSON parser instead of importing `go.mongodb.org/mongo-driver/bson`.

**Benefit**: Zero external dependency for MongoDB support. Keeps the binary small.

**Tradeoff**: Cannot parse all BSON types (e.g., Decimal128, Regex, Code, Binary). Falls back to returning a partial map when encountering unknown types. Sufficient for extracting command type, collection name, database name, and basic filter/update documents.

### 5.7 Rule-Based Diff Filtering

**Decision**: Differences can be suppressed via a declarative rule system with glob-style path matching and pluggable matchers.

**Rationale**: API migrations commonly introduce expected differences (new timestamps, regenerated UUIDs, reordered fields). Without filtering, the diff report would be overwhelmed with noise. The rule system allows operators to progressively whittle down the report to only unexpected behavioral changes.

**Built-in matchers**: `timestamp` (ISO 8601 patterns), `uuid` (RFC 4122 format), `numeric_tolerance` (configurable epsilon for floating-point drift).

### 5.8 Minimal Dependencies

**Decision**: The project uses only two direct dependencies: `cobra` (CLI framework) and `uuid` (ID generation).

**Benefit**: Small binary size, fast compilation, minimal supply-chain risk. All protocol parsing, JSON diffing, HTML templating, and storage are implemented in-house using the Go standard library.

**Tradeoff**: More code to maintain (e.g., the BSON parser, the JSON recursive differ). A library like `jsondiff` or `bson` would provide more robust handling at the cost of dependency weight.
