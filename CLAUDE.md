# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Shadiff is a shadow traffic semantic comparison tool for cross-framework / cross-language API migration validation. It uses a record-replay-diff three-stage pipeline to verify behavioral consistency between old and new APIs.

## Build & Development Commands

```bash
go build -o shadiff .          # Build
go run . version               # Run version command
go run . record --help         # Show record command help
go test ./...                  # Run all tests
```

## Tech Stack

- **Language:** Go 1.24
- **CLI Framework:** Cobra
- **Storage:** Filesystem (JSONL), ~/.shadiff/
- **Logging:** slog + daily rotation

## Architecture

### Core Workflow

```
record → replay → diff → report
```

### Package Structure

| Package | Responsibility |
|---------|---------------|
| `cmd/` | CLI commands (Cobra) |
| `internal/model/` | Core data models: Session, Record, SideEffect, DiffResult |
| `internal/config/` | Configuration management (~/.shadiff/config.json) |
| `internal/logger/` | Structured slog logging + rotation |
| `internal/capture/` | HTTP reverse proxy + DB protocol proxy capture |
| `internal/capture/dbhook/` | Database protocol proxies (MySQL/PostgreSQL/MongoDB) |
| `internal/storage/` | JSONL file storage |
| `internal/replay/` | Traffic replay engine |
| `internal/diff/` | Semantic diff engine |
| `internal/reporter/` | Report generation (terminal/JSON/HTML) |

### Data Storage

All persistent data is stored under `~/.shadiff/`:
- `config.json` — Global configuration
- `sessions/{id}/session.json` — Session metadata
- `sessions/{id}/records.jsonl` — Recorded behavior (JSONL streaming)
- `sessions/{id}/replay-records.jsonl` — Replay records
- `sessions/{id}/diff-results.json` — Diff results

## Key Conventions

- Go code follows standard Go conventions (gofmt, effective Go)
- Comments in English, identifiers in English
- Config management follows starxo's config.Store pattern (thread-safe JSON read/write)
- Logging uses slog + daily rotation, following starxo's logger pattern

## Engineering Conventions

### Commit Message Format (Conventional Commits)

```
<type>(<scope>): <subject>
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`

Scopes: `model`, `config`, `capture`, `dbhook`, `storage`, `replay`, `diff`, `reporter`, `logger`, `cmd`

### Development Workflow

1. **Plan** — Create plan documents in `plan/`
2. **Implement** — Develop on feature branch
3. **Verify** — End-to-end testing
4. **Commit** — Conventional Commits format

### Testing

- Go tests: `*_test.go` alongside source
- Test file naming: `<source>_test.go`
