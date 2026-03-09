# Config Runtime Integration Plan

## Goal

Make `config.json` the real runtime configuration source instead of a schema-only placeholder.

## Scope

In scope:

- `--config` support
- default config file bootstrap
- runtime config validation
- command/data-dir/logger integration
- config-backed capture, replay, diff, and storage behavior
- regression tests

Out of scope:

- new configuration file formats
- UI or daemon protocol changes

## Approach

- Add a shared runtime bootstrap in `cmd/` that loads, validates, and exposes config state.
- Route all commands through the resolved `storage.dataDir` and `log.logDir`.
- Apply a single precedence rule: CLI flag > config file > defaults.
- Wire previously dead config fields into capture, replay, diff, and storage flows.
- Add focused unit tests for the newly active behavior.

## Tasks

- Add `config.NewStoreWithPath()` and `config.Validate()`.
- Initialize runtime config from the root command before subcommands run.
- Update logger initialization to respect configured log directory and level.
- Update `record` to honor capture config, including DB proxy startup.
- Update `replay` to honor timeout, concurrency, delay, and retry config.
- Update `diff` to honor ignore settings, external rules, and diff truncation.
- Update storage retention behavior via `storage.maxSessions`.
- Add unit tests for config bootstrap, proxy behavior, retry logic, rule loading, and pruning.

## Verification

- `go test ./...`
- Manual spot checks:
  - custom `--config` path
  - custom `storage.dataDir`
  - custom `log.logDir`
  - config-driven `record`, `replay`, and `diff` defaults
