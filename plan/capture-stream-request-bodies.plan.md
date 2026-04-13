# Capture Stream Request Bodies Plan

## Goal

Remove the eager full-buffer request body read in `capture.Proxy` so large or skipped requests no longer force the proxy to load the entire body into memory before forwarding.

## Scope

In scope:

- request-body capture in `internal/capture/proxy.go`
- early skip-path handling before any request-body capture work
- proxy tests covering truncation, passthrough, and excluded paths
- documentation updates for the new capture behavior

Out of scope:

- response-body capture changes
- side-effect attribution changes in `Recorder`
- new user-facing CLI flags or config keys

## Approach

- Move the skip-path check ahead of any request-body capture logic.
- Replace the `io.ReadAll` + body restoration pattern with a streaming tap `io.ReadCloser` that:
  - forwards the full body to the upstream service unchanged
  - counts the original body length as bytes are read
  - stores only the first `maxBodySize` bytes for recording
- Keep the stored `Request.Body` and `Request.BodyLen` semantics unchanged:
  - `Body` contains the captured/truncated prefix
  - `BodyLen` reflects the full body size observed by the proxy

## Tasks

- Refactor `internal/capture/proxy.go`:
  - check `ExcludePathPrefixes` before request-body capture
  - add a streaming request-body capture wrapper
  - snapshot request metadata before proxy mutation and finish body capture after proxying
- Add tests in `internal/capture/proxy_test.go`:
  - existing truncation behavior still records only the configured prefix
  - large request bodies are fully forwarded upstream while capture stays truncated
  - excluded paths bypass recording and still forward the full request body
- Sync docs after implementation:
  - `doc/src/internal/capture/proxy.go.plan.md`
  - `doc/architecture.plan.md`
  - `doc/implementation.plan.md`
  - `plan/README.md`

## Verification

- `go test ./internal/capture`
- `go test ./...`
- Manual checks:
  - a request larger than `capture.maxBodySize` is still fully received by the target service
  - an excluded path is forwarded without creating a record
