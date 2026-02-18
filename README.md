# NethSecurity Monitoring

A collection of monitoring tools for NethSecurity systems, designed to provide real-time network flow analysis and system observability.

## Overview

NethSecurity Monitoring is a suite of monitoring binaries built in Go that work together to collect, process, and expose network and system metrics from NethSecurity firewall installations. The tools are designed to be lightweight, efficient, and easily deployable in containerized environments.

## Components

### ns-flows

Reads network flow data from netifyd's Unix socket, maintains an in-memory store of active flows enriched with DPI metadata, and exposes them through a paginated REST API served over a Unix socket.

**Usage:**

```bash
ns-flows \
  --socket /var/run/netifyd/flows.sock \
  --api-socket /var/run/nethsecurity-monitoring/flows.sock \
  --expired-persistence 60s \
  --log-level info
```

| Flag | Default | Description |
|---|---|---|
| `--socket` | `/var/run/netifyd/flows.sock` | Unix socket path for netifyd input |
| `--api-socket` | `/var/run/nethsecurity-monitoring/flows.sock` | Unix socket the HTTP API listens on |
| `--expired-persistence` | `60s` | TTL for flows not seen within this window |
| `--log-level` | `info` | One of `debug`, `info`, `warn`, `error` |

**Graceful shutdown** — the daemon listens for `SIGINT` and `SIGTERM`. On receipt it drains in-flight HTTP requests (`Shutdown`), stops all goroutines, and exits cleanly.

## API

The HTTP API is served over the Unix socket specified by `--api-socket`. The full API specification — including all endpoints, query parameters, request/response schemas, and examples — is documented in [openapi.yaml](openapi.yaml).

Quick example:

```bash
curl --unix-socket /var/run/nethsecurity-monitoring/flows.sock \
  'http://localhost/flows?per_page=20&sort_by=download_rate&desc=true'
```

## Building

### Using Make

```bash
# Build the project
make build

# Run tests
make test

# Build and test (default)
make

# Clean build artifacts
make clean
```

## License

GNU General Public License v3.0 see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines on how to contribute to this project.
