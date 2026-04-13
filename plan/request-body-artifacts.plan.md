# Request Body Artifacts Plan

## Goal

Make recorded request bodies faithfully replayable even when:

- the upstream handler does not fully read the incoming request body
- the full request body is larger than `capture.maxBodySize`

## Scope

In scope:

- full request-body snapshotting in `capture.Proxy`
- storing replayable request-body artifacts under the session directory
- recording `HTTPRequest.BodyRef` alongside the existing inline body preview
- replay-time request reconstruction that prefers stored full-body artifacts
- tests and docs covering backward compatibility and artifact lifecycle

Out of scope:

- response-body artifacts
- new CLI flags or config keys
- DB hook flush/barrier changes

## Approach

- Keep `Request.Body` as the inline preview field controlled by `capture.maxBodySize`.
- Add `Request.BodyRef` as an optional session-relative artifact path for the full request body.
- Snapshot the full incoming request body before proxying:
  - buffer in memory up to 1 MiB
  - spill to a temporary file after 1 MiB
- Persist the full body to `sessions/<id>/artifacts/request-bodies/<record-id>.bin` whenever replay would otherwise lose fidelity because the inline preview is truncated.
- Make replay prefer the stored artifact body when `BodyRef` is present, while preserving the old inline-body path for historical sessions.

## Tasks

- Update request and storage models:
  - add `HTTPRequest.BodyRef`
  - add `FileStore` helpers to save and open request body artifacts
- Refactor `internal/capture/proxy.go`:
  - snapshot full request bodies before forwarding
  - keep inline preview semantics for `Body` and `BodyLen`
  - save full-body artifacts when needed and store the returned `BodyRef`
- Update replay request construction:
  - use stored artifacts when `BodyRef` is present
  - fall back to inline `Request.Body` when replaying old sessions
- Add tests:
  - upstream does not read body but record still has correct `BodyLen` and preview
  - truncated inline preview also stores a full-body artifact
  - replay sends the full original request body when `BodyRef` exists
  - artifact lookup rejects invalid relative paths
- Sync docs:
  - `plan/README.md`
  - relevant `doc/src/...` file docs
  - project-level architecture / implementation / interfaces docs
  - `README.md` and `README_CN.md`

## Verification

- `go test ./internal/capture ./internal/replay ./internal/storage ./internal/model`
- `go test ./...`
- Manual checks:
  - excluded paths still proxy without creating records or artifacts
  - deleting a session removes any stored request-body artifacts with it
