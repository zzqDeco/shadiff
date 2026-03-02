# transform.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/replay/transform.go
- Doc file: doc/src/internal/replay/transform.go.plan.md
- File type: Go source
- Module: shadiff/internal/replay

## 2. Core Responsibility
- Transforms recorded HTTP requests to target a different service endpoint during replay, adapting URLs, headers, and removing proxy artifacts.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: `model.HTTPRequest` (the recorded request), `TransformConfig` (target URL, header overrides, header removals).
- Output results: `*http.Request` ready to be executed by an HTTP client, or `nil` if request construction fails.

## 4. Key Implementation Details
- Structs/interfaces:
  - `TransformConfig` -- configuration struct with:
    - `TargetBaseURL string` -- base URL of the replay target service.
    - `HeaderOverride map[string]string` -- headers to set/override on the request.
    - `HeaderRemove []string` -- headers to delete from the request.
- Exported functions/methods:
  - `Transform(req, cfg)` -- builds an `*http.Request` by:
    1. Constructing the full URL from `TargetBaseURL` + `req.Path` + optional query string.
    2. Creating an `http.Request` with the original method and body.
    3. Copying all original headers.
    4. Removing headers listed in `cfg.HeaderRemove`.
    5. Applying header overrides from `cfg.HeaderOverride`.
    6. Stripping proxy-related headers (`X-Forwarded-For`, `X-Forwarded-Host`, `X-Forwarded-Proto`).
- Key behaviors:
  - Returns `nil` (not an error) if `http.NewRequest` fails, which the caller must handle.
  - Proxy headers are always removed regardless of configuration, ensuring clean replay requests.
  - Header operations are applied in order: copy -> remove -> override -> strip proxy headers.

## 5. Dependencies
- Internal: `shadiff/internal/model` (for `HTTPRequest` type)
- External:
  - Standard library: `net/http`, `strings`.

## 6. Change Impact
- Changes to `TransformConfig` affect `Engine` construction in `engine.go` and `WorkerPool` initialization.
- Modifying the `Transform` function's header handling order could change replay behavior for requests with overlapping header rules.
- Adding new transformation steps (e.g., body rewriting, path prefix stripping) would affect all replayed requests.

## 7. Maintenance Notes
- The request body is converted from `[]byte` to `string` and wrapped in `strings.NewReader`; this creates a copy. For large bodies, consider using `bytes.NewReader` directly to avoid the allocation.
- Error handling returns `nil` instead of propagating the error from `http.NewRequest`; callers (specifically `replayOne` in `worker.go`) must check for nil.
- The hardcoded proxy header removal could be made configurable if proxy headers need to be preserved in certain replay scenarios.
