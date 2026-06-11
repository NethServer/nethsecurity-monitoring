# NethSecurity Monitoring — Agent Instructions

Go monitoring suite for NethSecurity firewalls. Two daemons live under [cmd/](cmd/):

- **ns-flows** — reads live network flows from netifyd's Unix socket, keeps an in-memory store enriched with DPI metadata, and serves them over an HTTP API. See [README.md](README.md).
- **ns-stats** — ingests aggregated traffic stats over HTTP, persists them in SQLite, and periodically exports per-host JSON reports.

## Build, test, lint

Use the [Makefile](Makefile) targets — do not invoke `go build`/`go test` directly when a target exists.

| Task | Command |
|---|---|
| Build both binaries → `dist/` | `make build` |
| Run all tests (with coverage) | `make test` |
| Format (goimports, gofumpt, golines) | `make format` |
| Lint | `make lint` |
| Test + build (default) | `make` |

**After making any code change, always run `make format`, then `make lint`, then `make test` and fix anything they report before considering the work done.** CI runs the linter, `make test`, and `make build` on every PR ([.github/workflows/ci.yml](.github/workflows/ci.yml)) — all must pass.

## Architecture

- **netifyd flow protocol** lives in [flows/parser.go](flows/parser.go). `FlowEvent` is a polymorphic wrapper whose `Flow` field is decoded by a custom `UnmarshalJSON` into one of `FlowComplete`, `FlowPurge`, or `FlowStats` based on the `type` field (`flow_dpi_complete`, `flow_purge`, `flow_stats`). Unknown types return `ErrUnsupportedFlowType` and are silently ignored, not treated as errors. When touching flow structs or the protocol, read the **netifyd-flow-protocol** skill first.
- **In-memory store**: [flows/processor.go](flows/processor.go) keeps flows in a `map[string]FlowEvent` keyed by flow `Digest`, guarded by a `sync.RWMutex`. Access goes through the `FlowAccessor` / `FlowIngestor` interfaces — the API depends on these interfaces, not the concrete type, so tests use mocks.
- **Expiry**: a goroutine in [cmd/ns-flows/main.go](cmd/ns-flows/main.go) periodically calls `PurgeFlowsOlderThan`; timestamps are **Unix milliseconds** (`time.UnixMilli`), not seconds.
- **HTTP layer**: [api/](api/) uses Fiber v3. The server binds to loopback only. Each API type holds its dependencies as interfaces.
- **Stats persistence**: [stats/stats.go](stats/stats.go) uses `modernc.org/sqlite` (pure-Go, no cgo). Writes are transactional with deferred rollback. [stats/export.go](stats/export.go) writes reports atomically (`.tmp` then rename) to `{dir}/{YYYY}/{MM}/{DD}/{local_ip}/{HH}.json`.
- **reverse_dns**: [reverse_dns/reverse_dns.go](reverse_dns/reverse_dns.go) is a TTL + LRU reverse-DNS cache using `singleflight` to dedupe concurrent lookups; falls back to the raw IP on failure.

## Conventions

- **Logging**: structured `slog` only, via the custom handler in [internal/logger/logger.go](internal/logger/logger.go). Use key/value attrs (`slog.Info("msg", "digest", d)`) — never printf-style formatting.
- **Concurrency**: any shared state must be guarded (`sync.RWMutex`, `atomic`, `singleflight`). Match the existing locking patterns when adding fields.
- **Errors**: wrap with context using `fmt.Errorf("...: %w", err)`. Sentinel errors (e.g. `ErrUnsupportedFlowType`) are compared with `errors.Is`.
- **Tests**: table-driven with `t.Run`; assertions via `github.com/go-playground/assert/v2`; HTTP handlers tested with Fiber's `app.Test()` and `httptest`; mock interfaces instead of concrete stores. Mark shared setup helpers with `t.Helper()`.

## API changes require OpenAPI sync

Any change to an HTTP endpoint or its query params **must** update [openapi.yaml](openapi.yaml) in the same change — query params, response schemas, and `sort_by` enum values must stay in sync with [api/flows.go](api/flows.go). See [CONTRIBUTING.md](CONTRIBUTING.md). PRs that diverge are rejected.

## Commits & PRs

PR titles **must** follow [Conventional Commits](https://www.conventionalcommits.org/) (e.g. `feat(flows): ...`, `fix(stats): ...`) — they become the squashed commit message and are checked by CI. See [CONTRIBUTING.md](CONTRIBUTING.md).
