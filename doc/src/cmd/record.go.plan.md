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
  - `--daemon` / `-D`: Run as a background daemon process (default `false`).
  - `--_daemon-child`: Internal flag for daemon child process (hidden from help output).
- Output results:
  - Creates a new session in the file store at `~/.shadiff`.
  - Persists all captured HTTP request/response pairs to storage via the `capture.Recorder`.
  - Prints status messages to stdout: session creation, proxy start address, and final record count.

## 4. Key Implementation Details
- Structs/interfaces: None defined directly; uses types from internal packages.
- Exported functions/methods: None (all functions and commands are package-private).
- Unexported functions:
  - `runRecord(cmd *cobra.Command, args []string) error` -- Main execution handler for the record command. Dispatches to `runDaemonParent()` or `runRecordLoop()`.
  - `runDaemonParent(cobraCmd *cobra.Command, dataDir string) error` -- Creates the session, re-execs the binary as a detached child with `--_daemon-child`, writes the PID file, and exits.
  - `runRecordLoop(dataDir string) error` -- Runs the actual recording proxy, used in both foreground mode and daemon child mode.
- Package-level variables:
  - `recordTarget string` -- Target service URL.
  - `recordListen string` -- Proxy listen address.
  - `recordSession string` -- Session name.
  - `recordDBProxy []string` -- Database proxy specifications.
  - `recordDuration string` -- Maximum recording duration.
  - `recordDaemon bool` -- Whether to run as a background daemon.
  - `daemonChild bool` -- Whether the current process is the daemon child (set by `--_daemon-child`).
- Key behaviors:
  - **Session creation**: Creates a `model.Session` with status `SessionRecording` and the target URL as the source endpoint.
  - **Proxy setup**: Creates a `capture.Recorder` for persisting traffic and a `capture.Proxy` that forwards requests to the target while recording them.
  - **Server lifecycle**: Runs an `http.Server` in a goroutine with the proxy as the handler.
  - **Signal handling**: Listens for `SIGINT` and `SIGTERM` via a buffered channel. Also supports context-based timeout via `--duration`.
  - **Graceful shutdown**: Uses a 5-second timeout context for `server.Shutdown()` to allow in-flight requests to complete.
  - **Session finalization**: After shutdown, updates the session status to `SessionCompleted` and persists the final record count. Clears the PID field.
  - **Logger initialization**: Initializes the file-based logger at `~/.shadiff` and defers cleanup. In daemon child mode, passes `daemonMode=true` to suppress stderr output.
  - **Daemon parent logic**: When `--daemon` is set, `runDaemonParent()` creates the session, re-execs the current binary with `--_daemon-child --session {id}`, calls `daemon.Detach()` for platform-specific process detach, redirects child stdout/stderr to `{sessionDir}/daemon.log`, writes the PID file after the child starts, updates the session with the child PID, and exits immediately.
  - **Daemon child logic**: When `--_daemon-child` is set, `runRecordLoop()` loads the existing session by ID, updates the session PID to its own `os.Getpid()`, defers `daemon.RemovePID()` for cleanup on exit, and suppresses interactive console output.

## 5. Dependencies
- Internal:
  - `shadiff/internal/capture` -- `NewRecorder()` and `NewProxy()` for traffic capture.
  - `shadiff/internal/daemon` -- PID file management (`WritePID`, `ReadPID`, `RemovePID`) and process detach (`Detach`).
  - `shadiff/internal/logger` -- File-based logging with `Init()`, `Error()`, `Close()`.
  - `shadiff/internal/model` -- `Session`, `EndpointConfig`, `SessionRecording`, `SessionCompleted` constants.
  - `shadiff/internal/storage` -- `FileStore` for session persistence.
- External:
  - `context` (standard library) -- Context for cancellation and timeout.
  - `fmt`, `os` (standard library) -- Output and home directory.
  - `net/http` (standard library) -- HTTP server for the reverse proxy.
  - `os/exec` (standard library) -- Re-exec for daemon child process.
  - `os/signal` (standard library) -- OS signal notification.
  - `path/filepath` (standard library) -- Path construction for session directory.
  - `syscall` (standard library) -- `SIGINT`, `SIGTERM` signal constants.
  - `time` (standard library) -- Duration parsing, timestamp formatting.
  - `github.com/spf13/cobra` -- Command definition.

## 6. Change Impact
- Changes to the proxy initialization affect how traffic is captured; any modification to `capture.NewProxy` or `capture.NewRecorder` signatures requires updating this file.
- The session model fields (`Status`, `Source`, `RecordCount`, `PID`, `DaemonMode`) must stay in sync with `model.Session`.
- The `--db-proxy` flag is declared but the DB proxy functionality is not wired into the current `runRecordLoop` implementation; only the HTTP proxy is started. Implementing DB proxying would require additional logic here.
- The data directory path (`~/.shadiff`) is hardcoded; centralizing this would reduce duplication across command files.
- The daemon parent/child protocol depends on the `--_daemon-child` and `--session` flags. Changes to the flag names or session resolution logic must be kept in sync between `runDaemonParent()` and `runRecordLoop()`.
- The `daemon` package API (`WritePID`, `ReadPID`, `RemovePID`, `Detach`) is used in this file; signature changes require updates here.

## 7. Maintenance Notes
- The `--db-proxy` flag is registered but not yet used in `runRecordLoop`. When implementing database side-effect capture, add proxy setup logic after the HTTP proxy creation.
- The `recordDuration` is parsed as a `time.Duration` string. If more complex scheduling is needed (e.g., stop after N requests), add a new flag and a separate stop condition in the `select` block.
- The recorder's `Count()` method returns an `int64` but is cast to `int` for the session's `RecordCount`. If very large recordings are expected, verify this does not cause truncation.
- Error from `store.Update(session)` at shutdown is logged but not returned; this is intentional to avoid masking earlier errors, but means session metadata may silently fail to update.
- Consider adding a `PersistentPreRunE` on the root command to centralize logger initialization and store creation instead of repeating it in each subcommand.
- The daemon child process removes the PID file on graceful exit via `defer daemon.RemovePID()`. If the child is force-killed, the stale PID file will remain; `record stop` and `record status` handle this case by checking process liveness.
