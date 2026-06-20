# main.go Technical Reference

## Purpose

- Source file: `examples/e2e/api/main.go`
- Provides the old/new API used by the official E2E demo.
- The service returns stable HTTP JSON while producing intentionally different database side effects for Shadiff to detect.

## Runtime Behavior

- `API_VARIANT=old|new` selects old or new behavior.
- `GET /health` returns a lightweight health response without touching databases.
- `GET /users/1` queries MySQL, PostgreSQL, MongoDB, and Redis, then returns a shared JSON response.
- Old and new variants use different SQL comments, different MongoDB collections, and different Redis keys/values so `shadiff diff` reports DB side-effect differences while HTTP output remains stable.

## Dependencies

- MySQL uses `github.com/go-sql-driver/mysql`.
- PostgreSQL uses `github.com/lib/pq`.
- MongoDB uses `go.mongodb.org/mongo-driver/v2`.
- Redis uses `github.com/redis/go-redis/v9`.
- The demo intentionally opens and closes database clients per request so Shadiff DB proxy shutdown is deterministic during scripted runs.
- The Docker build accepts a `GOPROXY` build argument, surfaced by the E2E runner as `SHADIFF_E2E_GOPROXY`, for hosts that cannot reach the default Go module proxy.
- The API image build uses Docker host networking so build-time proxy variables can point at host-local proxies such as Clash/Mihomo on `127.0.0.1`.
- The E2E runner can validate an existing release binary with `--binary`, print `--summary`, and write machine-readable acceptance data with `--summary-file`.

## Operational Notes

- The service is only intended for `examples/e2e`.
- Environment variables provide all database connection strings.
- It must not contain LAN hostnames, SSH details, or private paths.
