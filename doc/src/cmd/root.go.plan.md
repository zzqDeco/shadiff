# root.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: cmd/root.go
- Doc file: doc/src/cmd/root.go.plan.md
- File type: Go source
- Module: shadiff (package cmd)

## 2. Core Responsibility
- Defines the root Cobra command (`shadiff`) that acts as the parent for all subcommands.
- Registers global persistent flags (`--config`, `--verbose`, `--quiet`) available to all subcommands.
- Provides the `Execute()` function called by `main.go` to start the CLI.
- Sets up stdout/stderr output streams and embeds version information into the root command.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: Command-line arguments (parsed by Cobra), global flags (`--config`, `--verbose`, `--quiet`).
- Output results: Routes execution to the appropriate subcommand. Displays help text when no subcommand is provided.

## 4. Key Implementation Details
- Structs/interfaces: None defined directly; uses `cobra.Command` from spf13/cobra.
- Exported functions/methods:
  - `Execute() error` -- Runs the root Cobra command; called from `main.go`.
- Package-level variables:
  - `cfgFile string` -- Stores the config file path from `--config` flag.
  - `verbose bool` -- Enables verbose logging when `--verbose` / `-v` is set.
  - `quiet bool` -- Suppresses non-error output when `--quiet` / `-q` is set.
  - `rootCmd *cobra.Command` -- The root command instance; all subcommands attach to this.
- Key behaviors:
  - The `init()` function registers three persistent flags and configures output streams.
  - Version string is composed from `Version`, `Commit`, and `BuildDate` variables defined in `cmd/version.go`.
  - The root command itself has no `Run` or `RunE` handler; it displays help text by default.
  - The long description documents the full four-stage workflow: record, replay, diff, report.

## 5. Dependencies
- Internal: References `Version`, `Commit`, `BuildDate` from `cmd/version.go` (same package).
- External:
  - `fmt` (standard library) -- String formatting for version display.
  - `os` (standard library) -- Stdout/stderr stream assignment.
  - `github.com/spf13/cobra` -- CLI framework for command/flag registration and execution.

## 6. Change Impact
- Adding new global flags here makes them available to all subcommands.
- Changing `rootCmd.Use` or `rootCmd.Long` affects help output for the entire CLI.
- All subcommand files (`record.go`, `replay.go`, `diff.go`, `report.go`, `session.go`, `version.go`) depend on `rootCmd` to register themselves via `rootCmd.AddCommand()`.
- Altering `Execute()` affects the application entry point in `main.go`.

## 7. Maintenance Notes
- Keep global flags minimal; prefer subcommand-specific flags in their respective files.
- The `cfgFile` variable is declared but not yet used for configuration loading. When implementing config file support, add the loading logic in `init()` or a `PersistentPreRun` hook on `rootCmd`.
- The `verbose` and `quiet` flags are registered but must be checked explicitly in subcommand handlers or passed to the logger to take effect.
- When adding new subcommands, create a separate file in `cmd/` and call `rootCmd.AddCommand()` in its `init()` function.
