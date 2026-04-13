# record.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: `cmd/record.go`
- Doc file: `doc/src/cmd/record.go.plan.md`
- File type: Go source
- Module: `shadiff/cmd`

## 2. Core Responsibility
- Implements the `shadiff record` command, the record stage of the `record -> replay -> diff -> report` workflow.
- Resolves effective runtime values from `CLI flag > config.json > defaults` for listen address and DB proxy configuration.
- Supports both foreground recording and detached daemon recording through a parent/child self re-exec flow.
- Creates recording sessions, starts the HTTP proxy plus configured DB hooks, and finalizes session status on shutdown.

## 3. Inputs & Outputs
- Input sources:
  - CLI flags: `--target`, `--listen`, `--session`, `--db-proxy`, `--duration`, `--daemon`
  - Hidden internal flag: `--_daemon-child`
  - Runtime configuration from `cmd/runtime.go`, especially:
    - `capture.listenAddr`
    - `capture.maxBodySize`
    - `capture.excludePaths`
    - `capture.dbProxies`
    - `storage.dataDir`
    - `log.level` / `log.logDir`
- Output results:
  - Creates or loads a session in the configured storage directory.
  - Persists captured HTTP records and DB side effects through `capture.Recorder`.
  - In daemon mode, writes `daemon.log` and `pidfile` into the session directory.
  - Prints session/proxy lifecycle information in foreground mode and daemon bootstrap information in parent mode.

## 4. Key Implementation Details
- Package-level seams:
  - `execCommand` and `currentExecutable` wrap `exec.Command` / `os.Executable` for testability.
  - `runDaemonParentFn` and `runRecordLoopFn` allow branch selection to be tested without spawning real child processes.
- Main functions:
  - `runRecord(cmd *cobra.Command, args []string) error`
    - Resolves `recordListen` through `effectiveString(...)`.
    - Resolves DB proxies through `resolveRecordDBProxies(...)`.
    - Dispatches to daemon parent or normal record loop.
  - `runDaemonParent(cobraCmd *cobra.Command, dataDir string, dbProxies []config.DBProxyConfig) error`
    - Prunes old sessions.
    - Creates the daemon session up front.
    - Re-execs the current binary with `buildDaemonChildArgs(...)`.
    - Redirects child stdout/stderr to `{sessionDir}/daemon.log`.
    - Writes PID metadata for `record status` / `record stop`.
  - `buildDaemonChildArgs(sessionID string, dbProxies []config.DBProxyConfig) []string`
    - Reconstructs the effective child CLI including `--config`, `--duration`, and all DB proxies.
  - `runRecordLoop(dataDir string, dbProxies []config.DBProxyConfig) error`
    - Initializes logger and storage.
    - Creates or loads the session depending on daemon child mode.
    - Starts DB hooks through `startDBHooks(...)`.
    - Builds `capture.NewProxy(...)` with `MaxBodySize`, `ExcludePathPrefixes`, and DB-hook flush wiring.
    - Runs the HTTP server until signal or timeout.
    - Finalizes the session with `SessionCompleted`, `RecordCount`, and cleared PID.
- Shutdown behavior:
  - Uses `context.WithTimeout` for bounded recording duration.
  - Uses a 5-second graceful `server.Shutdown(...)`.
  - Stops all DB hooks with `defer stopDBHooks(hooks)`.
  - Leaves DB-hook flush failures as warning-only inside the proxy path so capture keeps succeeding.

## 5. Dependencies
- Internal:
  - `shadiff/internal/capture` for `Recorder` and HTTP proxy creation.
  - `shadiff/internal/config` for DB proxy config types.
  - `shadiff/internal/daemon` for detach, PID file, and signal utilities.
  - `shadiff/internal/logger` for runtime logging.
  - `shadiff/internal/model` for session metadata.
  - `shadiff/internal/storage` for session persistence and pruning.
- External:
  - Standard library `context`, `net/http`, `os`, `os/exec`, `os/signal`, `path/filepath`, `syscall`, `time`.
  - `github.com/spf13/cobra`.

## 6. Change Impact
- Changes to session lifecycle, daemon bootstrap, or PID management affect `record status` and `record stop`.
- Changes to config precedence must stay aligned with `cmd/runtime.go` and `cmd/dbproxy.go`.
- Changes to `capture.NewProxy(...)` options must stay aligned with `capture.maxBodySize` and `capture.excludePaths`.
- Changes to DB hook startup or side-effect fan-in affect whether database operations are attached to recorded HTTP records.

## 7. Maintenance Notes
- Keep the daemon parent/child contract stable: `--_daemon-child` and `--session <id>` are the glue between both processes.
- The test seams in this file are intentional and should remain lightweight; avoid turning them into user-visible APIs.
- Foreground recording and daemon child recording should continue sharing the same `runRecordLoop(...)` path so behavior stays consistent.
- If new capture-related config fields are added, resolve them here rather than re-introducing hardcoded defaults in command code.
