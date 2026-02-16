# NethSecurity Monitoring

A collection of monitoring tools for NethSecurity systems, designed to provide real-time network flow analysis and system observability.

## Overview

NethSecurity Monitoring is a suite of monitoring binaries built in Go that work together to collect, process, and expose network and system metrics from NethSecurity firewall installations. The tools are designed to be lightweight, efficient, and easily deployable in containerized environments.

## Components

### ns-flows

Reads network flow data from netifyd's Unix socket and produces a JSON file (`/var/run/netifyd/flows.json`) containing current flow state. Additionally, exposes a REST API for real-time flow data access.

**Usage:**
```bash
ns-flows -socket /var/run/netifyd/flows.sock -log-level info -http-addr 127.0.0.1:19000
```

**Options:**
- `-socket`: Path to the netifyd Unix socket (default: `/var/run/netifyd/flows.sock`)
- `-outfile`: Path to the output JSON file (default: `/var/run/netifyd/flows.json`)
- `-log-level`: Logging level: debug, info, warn, error (default: `info`)
- `-http-addr`: HTTP server address for API (default: `127.0.0.1:19000`)

**API Endpoints:**

- `GET /flows` - Returns the current list of active network flows as a JSON array

**Example:**
```bash
# Query current flows
curl http://127.0.0.1:19000/flows

# Paginated query with limit and offset
curl "http://127.0.0.1:19000/flows?start=0&end=10"
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
