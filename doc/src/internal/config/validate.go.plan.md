# validate.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: `internal/config/validate.go`
- Doc file: `doc/src/internal/config/validate.go.plan.md`
- File type: Go source
- Module: `shadiff/internal/config`

## 2. Core Responsibility
- Validates `AppConfig` values before commands start using them.
- Rejects unsupported or internally inconsistent config values instead of silently falling back at runtime.

## 3. Inputs & Outputs
- Input:
  - `*config.AppConfig`
- Output:
  - `nil` when the config is valid
  - descriptive `error` when a value is unsupported or malformed

## 4. Key Implementation Details
- `Validate(cfg *AppConfig) error` checks:
  - non-nil config
  - `capture.maxBodySize >= 0`
  - `replay.concurrency >= 1`
  - `replay.delayMs >= 0`
  - `replay.retryCount >= 0`
  - `replay.timeout` parses as `time.Duration`
  - `diff.maxDiffs >= 1`
  - `log.level` in `debug|info|warn|error`
  - each capture and replay DB proxy uses `mysql|postgres|mongo` and has non-empty listen/target addresses
- Validation is intentionally synchronous and side-effect free.

## 5. Dependencies
- External:
  - Standard library `fmt`, `strings`, `time`

## 6. Change Impact
- New config fields that can fail at runtime should usually gain validation here.
- Error messages from this file are user-visible during CLI startup, so wording changes affect troubleshooting UX.
- `cmd/runtime.go` depends on this file to reject invalid config before command execution.

## 7. Maintenance Notes
- Prefer explicit validation errors over silent normalization.
- Keep accepted value sets aligned with downstream packages such as logger and dbhook.
- When adding config validation, also extend `internal/config/validate_test.go`.
