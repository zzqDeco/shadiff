# record.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: cmd/record.go
- Doc file: doc/src/cmd/record.go.plan.md
- File type: Go source
- Module: shadiff (package cmd)

## 2. Core Responsibility
- Implements the `record` subcommand, which is the first stage of the shadiff workflow.
- Starts an HTTP reverse proxy that captures all request/response traffic passing through it.
- Creates a recording session, initializes the capture proxy, and persists recorded data to file storage.
- Supports graceful shutdown via OS signals (SIGINT/SIGTERM) and optional time-limited recording via `--duration`.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `--target` / `-t` (required): Target service URL to proxy requests to (e.g., `http://localhost:8080`).
  - `--listen` / `-l`: Local address for the proxy to listen on (default `:18080`).
  - `--session` / `-s`: Session name (auto-generated as `record-YYYYMMDD-HHMMSS` if omitted).
  - `--db-proxy`: Database proxy specifications (e.g., `mysql://:13306->:3306`), supports multiple values.
  - `--duration` / `-d`: Maximum recording duration as a Go duration string (e.g., `30m`, `1h`).
- Output results:
  - Creates a new session in the file store at `~/.shadiff`.
  - Persists all captured HTTP request/response pairs to storage via the `capture.Recorder`.
  - Prints status messages to stdout: session creation, proxy start address, and final record count.

## 4. Key Implementation Details
- Structs/interfaces: None defined directly; uses types from internal packages.
- Exported functions/methods: None (all functions and commands are package-private).
- Unexported functions:
  - `runRecord(cmd *cobra.Command, args []string) error` -- Main execution handler for the record command.
- Package-level variables:
  - `recordTarget string` -- Target service URL.
  - `recordListen string` -- Proxy listen address.
  - `recordSession string` -- Session name.
  - `recordDBProxy []string` -- Database proxy specifications.
  - `recordDuration string` -- Maximum recording duration.
- Key behaviors:
  - **Session creation**: Creates a `model.Session` with status `SessionRecording` and the target URL as the source endpoint.
  - **Proxy setup**: Creates a `capture.Recorder` for persisting traffic and a `capture.Proxy` that forwards requests to the target while recording them.
  - **Server lifecycle**: Runs an `http.Server` in a goroutine with the proxy as the handler.
  - **Signal handling**: Listens for `SIGINT` and `SIGTERM` via a buffered channel. Also supports context-based timeout via `--duration`.
  - **Graceful shutdown**: Uses a 5-second timeout context for `server.Shutdown()` to allow in-flight requests to complete.
  - **Session finalization**: After shutdown, updates the session status to `SessionCompleted` and persists the final record count.
  - **Logger initialization**: Initializes the file-based logger at `~/.shadiff` and defers cleanup.

## 5. Dependencies
- Internal:
  - `shadiff/internal/capture` -- `NewRecorder()` and `NewProxy()` for traffic capture.
  - `shadiff/internal/logger` -- File-based logging with `Init()`, `Error()`, `Close()`.
  - `shadiff/internal/model` -- `Session`, `EndpointConfig`, `SessionRecording`, `SessionCompleted` constants.
  - `shadiff/internal/storage` -- `FileStore` for session persistence.
- External:
  - `context` (standard library) -- Context for cancellation and timeout.
  - `fmt`, `os` (standard library) -- Output and home directory.
  - `net/http` (standard library) -- HTTP server for the reverse proxy.
  - `os/signal` (standard library) -- OS signal notification.
  - `syscall` (standard library) -- `SIGINT`, `SIGTERM` signal constants.
  - `time` (standard library) -- Duration parsing, timestamp formatting.
  - `github.com/spf13/cobra` -- Command definition.

## 6. Change Impact
- Changes to the proxy initialization affect how traffic is captured; any modification to `capture.NewProxy` or `capture.NewRecorder` signatures requires updating this file.
- The session model fields (`Status`, `Source`, `RecordCount`) must stay in sync with `model.Session`.
- The `--db-proxy` flag is declared but the DB proxy functionality is not wired into the current `runRecord` implementation; only the HTTP proxy is started. Implementing DB proxying would require additional logic here.
- The data directory path (`~/.shadiff`) is hardcoded; centralizing this would reduce duplication across command files.

## 7. Maintenance Notes
- The `--db-proxy` flag is registered but not yet used in `runRecord`. When implementing database side-effect capture, add proxy setup logic after the HTTP proxy creation.
- The `recordDuration` is parsed as a `time.Duration` string. If more complex scheduling is needed (e.g., stop after N requests), add a new flag and a separate stop condition in the `select` block.
- The recorder's `Count()` method returns an `int64` but is cast to `int` for the session's `RecordCount`. If very large recordings are expected, verify this does not cause truncation.
- Error from `store.Update(session)` at shutdown is logged but not returned; this is intentional to avoid masking earlier errors, but means session metadata may silently fail to update.
- Consider adding a `PersistentPreRunE` on the root command to centralize logger initialization and store creation instead of repeating it in each subcommand.
