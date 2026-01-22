# NethSecurity Monitoring

A collection of monitoring tools for NethSecurity systems, designed to provide real-time network flow analysis and system observability.

## Overview

NethSecurity Monitoring is a suite of monitoring binaries built in Go that work together to collect, process, and expose network and system metrics from NethSecurity firewall installations. The tools are designed to be lightweight, efficient, and easily deployable in containerized environments.

## Components

### ns-flows

Reads network flow data from netifyd's Unix socket and produces a JSON file (`/var/run/netifyd/flows.json`) containing current flow state.

**Usage:**
```bash
ns-flows -socket /var/run/netifyd/flows.sock -log-level info
```

## Building

### Using Docker

```bash
docker buildx bake dist
```

### Using Go
```bash
go build -o ns-flows
```

## License

GNU General Public License v3.0 see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines on how to contribute to this project.
