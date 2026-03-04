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

### Branch Management (Trunk-based)

`master` is the main branch. All changes enter via pull request.

Branch naming:
- `feature/<desc>` — new features
- `fix/<desc>` — bug fixes
- `refactor/<desc>` — code refactoring
- `docs/<desc>` — documentation-only changes
- `test/<desc>` — test infrastructure or test-only changes
- `release/<version>` — release preparation

Rules:
- Feature branches are created from `master` and merged back
- Delete branches after merge
- Keep branches short-lived

### Commit Message Format (Conventional Commits)

```
<type>(<scope>): <subject>
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `perf`

Scopes: `model`, `config`, `capture`, `dbhook`, `storage`, `replay`, `diff`, `reporter`, `logger`, `cmd`, `daemon`

Examples:
- `feat(capture): add Redis protocol proxy support`
- `fix(diff): handle nil JSON body comparison`
- `refactor(storage): extract JSONL read/write helpers`
- `docs: update architecture documentation`

### Development Workflow

Every feature, fix, or improvement follows a plan-first, doc-synced workflow:

1. **Plan** — Create or update a plan document in `plan/` describing the goal, scope, and approach
2. **Select** — Choose specific items from the plan to implement in the current iteration
3. **Implement** — Write code on a feature branch following the conventions above
4. **Test** — Write unit tests alongside implementation (`*_test.go`)
5. **Sync Docs** — Update all affected documentation:
   - `doc/src/<file>.plan.md` for any modified source files (keep the 7-section template in sync)
   - `doc/` project-level docs if architecture, interfaces, or flows changed
   - `doc/files.index.plan.md` and `doc/files.coverage.plan.md` if files were added/removed
   - `README.md` / `README_CN.md` if user-facing features or CLI commands changed
6. **Verify** — Run `go test ./...` and manually verify the changes work correctly
7. **Commit & Push** — Commit with conventional commit messages, push to remote

### Plan Documents (`plan/`)

Plan documents describe future work before implementation begins. Each plan should include:

- **Goal** — What problem this solves
- **Scope** — What's in/out of scope
- **Approach** — Technical design and key decisions
- **Tasks** — Breakdown of implementation steps
- **Verification** — How to confirm the implementation is correct

`plan/README.md` serves as the index, tracking all phases and their status (Pending / In Progress / Completed).

### Technical Documentation (`doc/`)

The `doc/` directory contains two levels of documentation:

#### Project-level docs (in `doc/`)

| Document | Purpose |
|----------|---------|
| `plan.md` | Main technical document index — project positioning, build facts, modules |
| `architecture.plan.md` | Architecture overview — data flow, protocol proxying, design decisions |
| `interfaces.plan.md` | Interface documentation — CLI commands, Go interfaces, data contracts |
| `implementation.plan.md` | Implementation mapping — module responsibilities, file listing, patterns |
| `files.index.plan.md` | File-to-doc mapping — every source file mapped to its doc file |
| `files.coverage.plan.md` | Documentation coverage statistics |

#### File-level docs (in `doc/src/`)

Each source file has a corresponding `doc/src/<same-path>.plan.md` document. Mapping rules:
- `main.go` → `doc/src/main.plan.md`
- `internal/diff/engine.go` → `doc/src/internal/diff/engine.plan.md`
- `cmd/record.go` → `doc/src/cmd/record.plan.md`

When modifying source files, update the corresponding doc file. When adding/removing files, update `files.index.plan.md` and `files.coverage.plan.md`.

### Testing

- Go tests: `*_test.go` alongside source files
- Test file naming: `<source>_test.go`
- Use standard `testing` package
- Run all tests: `go test ./...`
- Tests should be independent and use `t.TempDir()` for filesystem isolation

### Code Review Expectations

- All PRs require review before merge
- Reviewer checks: correctness, error handling, naming, test coverage
- Follow standard Go conventions (gofmt, effective Go)
- New interfaces must be documented in `doc/interfaces.plan.md`
- New CLI flags must be documented in README
