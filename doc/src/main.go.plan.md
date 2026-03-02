# main.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: main.go
- Doc file: doc/src/main.go.plan.md
- File type: Go source
- Module: shadiff

## 2. Core Responsibility
- Serves as the application entry point for the shadiff CLI tool.
- Delegates all command execution to the `cmd` package via `cmd.Execute()`.
- Handles top-level error propagation by exiting with code 1 on failure.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: Command-line arguments passed by the OS to the process (implicitly via `os.Args`).
- Output results: Delegates to `cmd.Execute()` which drives all CLI output. Returns exit code 0 on success, 1 on error.

## 4. Key Implementation Details
- Structs/interfaces: None defined.
- Exported functions/methods: None (package `main`).
- Key behaviors:
  - Calls `cmd.Execute()` which runs the Cobra root command.
  - If `cmd.Execute()` returns a non-nil error, the process exits with status code 1.
  - No logging, configuration, or initialization logic exists here; all of that is handled in `cmd/root.go` and subcommand files.

## 5. Dependencies
- Internal: `shadiff/cmd` (the command package containing all CLI logic).
- External: `os` (standard library, for `os.Exit`).

## 6. Change Impact
- Changing this file affects the entire application startup flow.
- If the call to `cmd.Execute()` is removed or altered, no CLI commands will function.
- This file is intentionally minimal; most changes should be made in the `cmd` package instead.

## 7. Maintenance Notes
- This file should remain as thin as possible. All command registration, flag parsing, and business logic belong in the `cmd` package.
- Build-time variable injection (e.g., `-ldflags`) targets variables in `cmd/version.go`, not this file.
- If global initialization is needed before command execution (e.g., tracing, profiling), it can be added here before the `cmd.Execute()` call.
