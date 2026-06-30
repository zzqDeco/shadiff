# tcp_proxy.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/capture/dbhook/tcp_proxy.go
- Doc file: doc/src/internal/capture/dbhook/tcp_proxy.go.plan.md
- File type: Go source
- Module: shadiff/internal/capture/dbhook

## 2. Core Responsibility
- Provides the shared transparent TCP proxy lifecycle for database hooks.
- Owns listener setup, connection tracking, server/client forwarding, flush barriers, side-effect channel emission, and shutdown coordination.

## 3. Inputs & Outputs
- Inputs: database type, listen address, target address, and a per-connection `protocolParser` factory.
- Outputs: `model.SideEffect` values emitted from parser output on a buffered channel.

## 4. Key Implementation Details
- `protocolParser` exposes `Feed([]byte) []model.SideEffect` so protocol implementations can be stream-oriented.
- `tcpProxy` implements the `DBHook` lifecycle methods used by embedded hook structs.
- Client-to-server bytes are always forwarded before parser output is emitted; malformed parser input does not block traffic.
- `Flush(ctx)` sends per-connection barriers and drains readable client bytes until an idle window expires.
- Flush lifecycle tests cover active connection delivery, context timeout while waiting for ack, closed-connection skipping, and transparent forwarding when a parser emits no side effect.

## 5. Dependencies
- Internal: `logger`, `model`.
- External: `context`, `io`, `net`, `sync`, `time`.

## 6. Change Impact
- Changes to lifecycle or flush semantics affect MySQL, PostgreSQL, MongoDB, and Redis hooks together.

## 7. Maintenance Notes
- Keep protocol-specific framing in parser files. This file should not know SQL, MongoDB, or Redis wire formats.
