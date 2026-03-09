# runtime.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: `cmd/runtime.go`
- Doc file: `doc/src/cmd/runtime.go.plan.md`
- File type: Go source
- Module: `shadiff/cmd`

## 2. Core Responsibility
- Centralizes runtime configuration loading for all Cobra commands.
- Builds a shared `appRuntime` context from `--config`, config defaults, and derived paths such as `DataDir` and `LogDir`.
- Exposes helper functions used by commands to read runtime state and apply flag-over-config precedence.

## 3. Inputs & Outputs
- Inputs:
  - Global CLI flags from `cmd/root.go`: `--config`, `--verbose`, `--quiet`
  - `internal/config` store and validation logic
- Outputs:
  - Shared `runtimeCtx` with `ConfigPath`, `Config`, `DataDir`, and `LogDir`
  - Helper values such as effective log level and flag/config merged scalar values

## 4. Key Implementation Details
- `initRuntime()` is called from `rootCmd.PersistentPreRunE`.
- Runtime initialization flow:
  - `config.NewStoreWithPath(cfgFile)`
  - `store.Get()`
  - `config.Validate(cfg)`
  - derive `dataDir` and `logDir`
  - populate `runtimeCtx`
- Accessors:
  - `currentConfig()`
  - `currentDataDir()`
  - `currentLogDir()`
  - `currentConfigPath()`
- Merge helpers:
  - `effectiveLogLevel()`
  - `effectiveString(...)`
  - `effectiveStrings(...)`
  - `effectiveBool(...)`
  - `effectiveInt(...)`
- Safety:
  - `mustRuntime()` panics if commands try to read runtime state before initialization.

## 5. Dependencies
- Internal:
  - `shadiff/internal/config`
- External:
  - Standard library `fmt`, `path/filepath`, `strings`

## 6. Change Impact
- Any change to config loading, path derivation, or validation affects every command because they all read from `runtimeCtx`.
- Precedence semantics must remain consistent across commands; helper changes here alter user-visible behavior broadly.
- `effectiveLogLevel()` must stay aligned with logger expectations and global `--verbose` / `--quiet` semantics.

## 7. Maintenance Notes
- Keep runtime initialization side-effect free beyond config loading and derived path computation.
- New config-backed command behavior should use the helper functions here instead of re-implementing precedence logic.
- If runtime grows more complex, prefer extending `appRuntime` over introducing package-global ad hoc state in commands.
