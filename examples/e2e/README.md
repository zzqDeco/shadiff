# Official E2E Demo

This demo runs a full Shadiff loop:

```text
record -> replay -> diff -> report
```

Docker Compose starts five services:

- `old-api`: source API on `127.0.0.1:18081`
- `new-api`: replay target API on `127.0.0.1:18082`
- `mysql`: real MySQL on `127.0.0.1:33306`
- `postgres`: real PostgreSQL on `127.0.0.1:35432`
- `mongo`: real MongoDB on `127.0.0.1:37017`

The API response body is intentionally the same for old and new services. The database side effects intentionally differ so `shadiff diff` reports SQL and MongoDB differences.

## Requirements

- Linux host with Docker Engine and the Docker Compose plugin
- Bash
- Curl
- Go 1.25, unless `SHADIFF_BIN` points to an existing Shadiff binary

The demo does not store hostnames, SSH details, or private paths in the repository.

## Run

From the repository root:

```bash
./examples/e2e/run.sh --assert
```

To use an existing binary:

```bash
SHADIFF_BIN=/path/to/shadiff ./examples/e2e/run.sh --assert
```

To keep containers running for troubleshooting:

```bash
./examples/e2e/run.sh --assert --keep
```

If the Docker build host cannot reach the default Go module proxy, override it for the demo API image build:

```bash
SHADIFF_E2E_GOPROXY=https://goproxy.cn,direct ./examples/e2e/run.sh --assert
```

The script writes isolated runtime data under:

```text
examples/e2e/.work/<run-id>/
```

Important artifacts:

- `artifacts/record.log`
- `artifacts/replay.log`
- `artifacts/diff.json`
- `artifacts/report.html`
- `artifacts/record-response.json`

## Expected Result

`run.sh --assert` exits with `0` when:

- the record stage captures at least one request
- replay writes replay records
- diff writes `diff.json`
- report writes `report.html`
- `diff.json` contains at least one `db_query` difference
- `diff.json` contains at least one `mongo_op` difference

This means HTTP response behavior stayed stable while database side effects changed.

## Ports

| Purpose | Address |
|---|---|
| old API | `127.0.0.1:18081` |
| new API | `127.0.0.1:18082` |
| record HTTP proxy | `127.0.0.1:18080` |
| real MySQL | `127.0.0.1:33306` |
| real PostgreSQL | `127.0.0.1:35432` |
| real MongoDB | `127.0.0.1:37017` |
| record MySQL proxy | `:13306` |
| record PostgreSQL proxy | `:15432` |
| record MongoDB proxy | `:27018` |
| replay MySQL proxy | `:13316` |
| replay PostgreSQL proxy | `:15442` |
| replay MongoDB proxy | `:27028` |

DB proxy listen addresses bind all host interfaces so containers can reach them through `host.docker.internal`. Run this demo only on a trusted development machine.

## Remote Linux Machine

If the Linux development machine is on your LAN, run it with a one-off SSH command. Do not commit the hostname or SSH details:

```bash
ssh <linux-user>@<linux-host> 'cd /path/to/shadiff && git fetch origin && git checkout feature/e2e-demo && ./examples/e2e/run.sh --assert'
```

## Troubleshooting

- Port conflicts: stop the process using the listed ports, or run on a clean development machine.
- Docker networking: the API containers rely on `host.docker.internal:host-gateway`, which is supported by modern Docker Engine on Linux.
- Go module downloads: set `SHADIFF_E2E_GOPROXY` when `go mod download` inside the API image build cannot reach `proxy.golang.org`.
- Failed assertions: inspect `examples/e2e/.work/<run-id>/artifacts/diff.json` and the stage logs.
