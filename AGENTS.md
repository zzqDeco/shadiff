# AGENTS.md

This file provides guidance to coding agents working in this repository.

## Project Summary

Shadiff is a shadow-traffic semantic comparison tool for validating API migrations across frameworks or languages. It follows a record-replay-diff pipeline:

```text
record -> replay -> diff -> report
```

Primary stack:

- Go 1.24
- Cobra for CLI commands
- Filesystem-backed storage using JSONL under `~/.shadiff/`
- `slog` with daily log rotation

## Common Commands

```bash
go build -o shadiff .
go run . version
go run . record --help
go test ./...
```

## Repository Layout

- `cmd/`: Cobra CLI entrypoints and commands
- `internal/model/`: core models such as session, record, side effects, diff results
- `internal/config/`: configuration management for `~/.shadiff/config.json`
- `internal/logger/`: structured logging
- `internal/capture/`: HTTP and DB capture pipeline
- `internal/capture/dbhook/`: database protocol proxy hooks
- `internal/storage/`: JSONL-backed storage
- `internal/replay/`: replay engine
- `internal/diff/`: semantic diff engine
- `internal/reporter/`: terminal, JSON, and HTML reporting
- `plan/`: implementation plans and phase tracking
- `doc/`: project-level and file-level technical documentation

## Data Storage

Persistent runtime data lives under `~/.shadiff/`:

- `config.json`: global configuration
- `sessions/{id}/session.json`: session metadata
- `sessions/{id}/records.jsonl`: recorded traffic
- `sessions/{id}/replay-records.jsonl`: replay output
- `sessions/{id}/diff-results.json`: diff results

## Working Rules

- Follow standard Go conventions and keep code `gofmt`-ready.
- Use English for comments and identifiers.
- Preserve the existing config-store pattern for thread-safe JSON read/write.
- Preserve the existing logging pattern based on `slog` and daily rotation.

## Required Delivery Workflow

For every feature, fix, or meaningful improvement, follow this sequence:

1. Plan: create or update a plan document in `plan/` covering goal, scope, approach, tasks, and verification.
2. Select: choose the specific plan items for the current iteration.
3. Implement: make the code change on a short-lived branch from `dev` when working with git branches.
4. Test: add or update `*_test.go` tests alongside the implementation.
5. Sync Docs: update all impacted documentation.
6. Verify: run `go test ./...` and perform any relevant manual verification.
7. Commit: use a conventional commit message.

If the task is documentation-only, keep the same discipline where applicable, but skip code-only steps that do not apply.

## Documentation Sync Requirements

When source code changes, update the matching file-level doc in `doc/src/`.

Mapping examples:

- `main.go` -> `doc/src/main.go.plan.md`
- `cmd/record.go` -> `doc/src/cmd/record.go.plan.md`
- `internal/diff/engine.go` -> `doc/src/internal/diff/engine.go.plan.md`

Also update the relevant project-level docs in `doc/` when behavior, architecture, interfaces, or file inventory changes:

- `doc/architecture.plan.md`
- `doc/interfaces.plan.md`
- `doc/implementation.plan.md`
- `doc/files.index.plan.md`
- `doc/files.coverage.plan.md`
- `README.md`
- `README_CN.md`

Specific expectations:

- New interfaces must be documented in `doc/interfaces.plan.md`.
- New CLI flags or user-facing commands must be documented in `README.md` and `README_CN.md`.
- Added or removed source files must be reflected in `doc/files.index.plan.md` and `doc/files.coverage.plan.md`.

## Plan Documents

Plan documents live in `plan/`. Each plan should include:

- Goal
- Scope
- Approach
- Tasks
- Verification

`plan/README.md` is the index for plan phases and statuses.

## Testing Expectations

- Place tests next to the code they cover.
- Name test files `<source>_test.go`.
- Use the standard `testing` package.
- Prefer independent tests and use `t.TempDir()` for filesystem isolation where relevant.
- Run `go test ./...` before finishing work unless the user explicitly limits verification.

## Git Conventions

Base branches:

- `main`: stable promotion branch and current default branch
- `dev`: integration branch for day-to-day changes
- `master`: deprecated legacy branch; do not target new work here

Workflow rules:

- Create short-lived working branches from `dev`.
- Open normal feature, fix, docs, refactor, test, and release-prep PRs into `dev` first.
- Promote `dev` into `main` with a separate PR when the integration branch is ready.

Branch naming:

- `feature/<desc>`
- `fix/<desc>`
- `refactor/<desc>`
- `docs/<desc>`
- `test/<desc>`
- `release/<version>`

Commit format:

```text
<type>(<scope>): <subject>
```

Allowed types:

- `feat`
- `fix`
- `refactor`
- `docs`
- `test`
- `chore`
- `perf`

Common scopes:

- `model`
- `config`
- `capture`
- `dbhook`
- `storage`
- `replay`
- `diff`
- `reporter`
- `logger`
- `cmd`
- `daemon`

Examples:

- `feat(capture): add Redis protocol proxy support`
- `fix(diff): handle nil JSON body comparison`
- `refactor(storage): extract JSONL read/write helpers`
- `docs: update architecture documentation`

## Review Checklist

Before considering the task complete, confirm:

- The implementation is correct and handles errors reasonably.
- Naming stays consistent with Go conventions and existing package boundaries.
- Tests cover the change.
- Documentation has been synchronized.
- `go test ./...` passes, or any inability to run it is explicitly reported.
