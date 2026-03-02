# version.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: cmd/version.go
- Doc file: doc/src/cmd/version.go.plan.md
- File type: Go source
- Module: shadiff (package cmd)

## 2. Core Responsibility
- Defines the `version` subcommand that displays build-time version information.
- Declares build-time injectable variables (`Version`, `Commit`, `BuildDate`) used by the root command and the version subcommand.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: No flags or arguments. Build-time values are injected via `go build -ldflags`.
- Output results: Prints version, commit hash, and build date to stdout in a human-readable format.

## 4. Key Implementation Details
- Structs/interfaces: None defined.
- Exported variables:
  - `Version string` -- Semantic version string, defaults to `"0.1.0"`.
  - `Commit string` -- Git commit hash, defaults to `"dev"`.
  - `BuildDate string` -- Build timestamp, defaults to `"unknown"`.
- Exported functions/methods: None (variables are exported, the command is package-private).
- Key behaviors:
  - The `init()` function registers `versionCmd` as a subcommand of `rootCmd`.
  - Output format:
    ```
    shadiff 0.1.0
      commit:  dev
      built:   unknown
    ```
  - These same variables are referenced in `cmd/root.go` to set `rootCmd.Version`, which enables the built-in `--version` flag on the root command.

## 5. Dependencies
- Internal: References `rootCmd` from `cmd/root.go` (same package).
- External:
  - `fmt` (standard library) -- Formatted output.
  - `github.com/spf13/cobra` -- Command definition.

## 6. Change Impact
- Modifying the exported variables' default values affects version display when built without `-ldflags`.
- The root command in `root.go` depends on `Version`, `Commit`, and `BuildDate` for its own `--version` output; changes here propagate there.
- Build scripts or CI pipelines that inject values via `-ldflags "-X shadiff/cmd.Version=..."` depend on the exact variable names and package path.

## 7. Maintenance Notes
- When setting up CI/CD, inject version info at build time using:
  ```
  go build -ldflags "-X shadiff/cmd.Version=v1.0.0 -X shadiff/cmd.Commit=$(git rev-parse HEAD) -X shadiff/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  ```
- Keep the default values meaningful for local development (`"dev"`, `"unknown"`).
- If additional build metadata is needed (e.g., Go version, OS/arch), add new exported variables here and update both the `versionCmd` handler and the `rootCmd.Version` format string in `root.go`.
