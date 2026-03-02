# session.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: cmd/session.go
- Doc file: doc/src/cmd/session.go.plan.md
- File type: Go source
- Module: shadiff (package cmd)

## 2. Core Responsibility
- Implements the `session` command group with three subcommands: `list`, `show`, and `delete`.
- Provides CRUD-like management of recording sessions stored on disk.
- Contains the shared `getStore()` helper that creates a `storage.FileStore` instance pointing to `~/.shadiff`.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - `session list`: Optional `--tag` flag to filter sessions by tag.
  - `session show <id>`: Positional argument specifying the session ID.
  - `session delete <id>`: Positional argument specifying the session ID.
  - All subcommands read session data from the file store at `~/.shadiff`.
- Output results:
  - `session list`: Tabular output to stdout with columns: ID, NAME, STATUS, RECORDS, CREATED.
  - `session show`: Detailed key-value display of a single session's metadata.
  - `session delete`: Confirmation message after successful deletion.

## 4. Key Implementation Details
- Structs/interfaces: None defined directly; uses `model.Session`, `model.SessionFilter` from `shadiff/internal/model`.
- Exported functions/methods: None (all functions and commands are package-private).
- Unexported functions:
  - `getStore() (*storage.FileStore, error)` -- Creates a FileStore rooted at `~/.shadiff`. Used by `session` subcommands and also available to other cmd files in the same package.
  - `runSessionList(cmd *cobra.Command, args []string) error` -- Lists sessions, optionally filtered by tag.
  - `runSessionShow(cmd *cobra.Command, args []string) error` -- Displays detailed metadata for a single session.
  - `runSessionDelete(cmd *cobra.Command, args []string) error` -- Deletes a session after verifying it exists.
- Package-level variables:
  - `sessionTagFilter string` -- Holds the `--tag` flag value for list filtering.
- Key behaviors:
  - `session list` uses `tabwriter` for aligned columnar output.
  - Timestamps are stored as Unix milliseconds (`CreatedAt`, `UpdatedAt`) and formatted as `"2006-01-02 15:04"` or `"2006-01-02 15:04:05"`.
  - `session delete` performs a two-step process: first verifies the session exists via `store.Get()`, then deletes it. This prevents silent failures on invalid IDs.
  - `session show` displays Source and Target base URLs, tags, record count, and timestamps.

## 5. Dependencies
- Internal:
  - `shadiff/internal/model` -- `Session`, `SessionFilter` types.
  - `shadiff/internal/storage` -- `FileStore` for persistent session storage.
- External:
  - `fmt`, `os` (standard library) -- Output and home directory resolution.
  - `text/tabwriter` (standard library) -- Aligned tabular output for `session list`.
  - `time` (standard library) -- Timestamp formatting.
  - `github.com/spf13/cobra` -- Command definition.

## 6. Change Impact
- `getStore()` is used by other command files (`record.go`, `replay.go`, `diff.go`, `report.go` each create their own store independently). If the data directory logic changes, all files need updating; consider centralizing.
- Changes to `model.Session` fields require updates to the `show` and `list` output formatting.
- Adding new session subcommands (e.g., `session rename`, `session export`) should follow the pattern established here: define a `cobra.Command`, wire it in `init()`, implement a `runSessionXxx` handler.

## 7. Maintenance Notes
- The `getStore()` function duplicates data directory resolution (`~/.shadiff`) that also appears in `record.go`, `replay.go`, `diff.go`, and `report.go`. A future refactor could centralize this into a shared helper or configuration.
- The `session list` command currently only supports filtering by a single tag. Multi-tag or status-based filtering could be added by extending `SessionFilter`.
- The `session show` command accesses `sess.Source.BaseURL` and `sess.Target.BaseURL` directly. If these are empty (e.g., before replay), they print as empty strings. Consider adding conditional display.
- `session delete` does not prompt for confirmation. For destructive operations, consider adding a `--force` flag or interactive confirmation.
