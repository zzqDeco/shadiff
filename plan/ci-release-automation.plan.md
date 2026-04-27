# CI and Release Automation Plan

## Goal

Add GitHub Actions coverage for routine CI and release asset generation so PRs, `dev`, `main`, and tagged releases are verified consistently.

## Scope

In scope:

- CI workflow for `main`, `dev`, and PRs into those branches
- release workflow for semantic-version tags and manual dispatch
- cross-platform release archives for Linux, macOS, and Windows on amd64/arm64
- checksum generation
- README and plan documentation updates

Out of scope:

- deleting the deprecated `master` branch
- Docker-based database integration tests
- package manager publishing

## Approach

- Use `actions/setup-go@v5` with Go `1.24.x`.
- Keep CI lightweight: dependency download, `go test ./...`, and `go build -o shadiff .`.
- Build release assets with `CGO_ENABLED=0`, injecting `Version`, `Commit`, and `BuildDate` through `-ldflags`.
- Support both tag-push releases and manual dispatch so existing releases can be backfilled with assets.

## Tasks

- Add `.github/workflows/ci.yml`.
- Add `.github/workflows/release.yml`.
- Document the workflows in README files.
- Add this plan and update `plan/README.md`.

## Verification

- `go test ./...`
- `go build -o shadiff .`
- local cross-platform release build loop for all release targets
- PR CI on `feature/ci-release-automation -> dev`
- CI after `dev -> main` promotion
- manual `release.yml` dispatch for `v0.1.0`
