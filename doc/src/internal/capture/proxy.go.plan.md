# proxy.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/proxy.go
- Doc file: doc/src/internal/capture/proxy.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture

## 2. Core Responsibility
- Implements an HTTP reverse proxy that transparently forwards client requests to a target backend service while capturing complete request/response pairs as `model.Record` entries.
- Acts as the primary HTTP traffic interception point in shadiff's capture pipeline.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources:
  - Incoming HTTP requests from clients (via `http.Handler` interface).
  - Target backend URL string provided at construction time.
  - A `*Recorder` instance for persisting captured data.
- Output results:
  - Proxied HTTP responses returned to the client (unchanged from the target).
  - `*model.Record` objects sent to the `Recorder`, containing the full HTTP request, response, timing, and sequence metadata.
  - Structured log events emitted via `logger.CaptureEvent` and `logger.Error`.

## 4. Key Implementation Details
- Structs/interfaces:
  - `Proxy` -- Main struct holding the parsed target URL, the stdlib `httputil.ReverseProxy`, a `*Recorder` reference, and an atomic sequence counter.
  - `responseRecorder` -- An `http.ResponseWriter` wrapper that intercepts `WriteHeader` and `Write` calls to capture the response status code and body while still forwarding data to the real writer.
- Exported functions/methods:
  - `NewProxy(targetURL string, recorder *Recorder) (*Proxy, error)` -- Parses the target URL and constructs a configured reverse proxy.
  - `(*Proxy).ServeHTTP(w http.ResponseWriter, r *http.Request)` -- Implements `http.Handler`. Reads the request body, proxies to the target, captures the response, builds a `model.Record`, and hands it to the recorder.
- Key behaviors:
  - Request bodies are fully read into memory before forwarding so they can be recorded; the body is then restored via `io.NopCloser(bytes.NewReader(...))`.
  - Each request gets a monotonically increasing sequence number via `atomic.Int64`.
  - Record IDs are the first 8 characters of a UUID v4.
  - `responseRecorder` uses a `wroteHeader` flag to ensure `WriteHeader` is called at most once on the underlying writer, preventing double-write panics.
  - Headers are deep-copied via the unexported `cloneHeaders` helper to avoid mutation after capture.
  - Duration is measured in milliseconds from just before proxying to just after.

## 5. Dependencies
- Internal:
  - `shadiff/internal/logger` -- Structured logging for capture events and errors.
  - `shadiff/internal/model` -- Data types (`HTTPRequest`, `HTTPResponse`, `Record`, `SideEffect`).
- External:
  - `net/http`, `net/http/httputil` -- Reverse proxy and HTTP handler primitives.
  - `net/url` -- Target URL parsing.
  - `bytes`, `io` -- Request body buffering and restoration.
  - `sync/atomic` -- Lock-free sequence counter.
  - `time` -- Duration measurement and timestamps.
  - `github.com/google/uuid` -- Record ID generation.

## 6. Change Impact
- `internal/capture/recorder.go` -- `Proxy` calls `Recorder.Record()` directly; changes to the `Record` struct or recorder API require updates here.
- `internal/model/` -- Any changes to `HTTPRequest`, `HTTPResponse`, or `Record` fields affect the record-building logic in `ServeHTTP`.
- Any code that instantiates `NewProxy` (typically the CLI or server setup) is affected if the constructor signature changes.
- `responseRecorder` is unexported and self-contained; changes are localized unless the capture semantics change.

## 7. Maintenance Notes
- The full request body is buffered in memory. For large payloads, consider adding a configurable size limit or streaming capture to avoid excessive memory usage.
- The `responseRecorder` does not capture trailers; if HTTP trailer support is needed, `Write` and `Flush` should be extended.
- The `cloneHeaders` function is a simple deep copy; it does not handle canonical header casing differences beyond what Go's `http.Header` already provides.
- The UUID truncation to 8 characters is sufficient for short-lived capture sessions but may collide at scale; consider using full UUIDs if sessions become very large.
