---
name: netifyd-flow-protocol
description: "Use when working with netifyd network flow data in nethsecurity-monitoring: parsing/modifying the flow JSON protocol, adding or changing FlowEvent/FlowComplete/FlowPurge/FlowStats structs or their fields, handling flow event types (flow_dpi_complete, flow_purge, flow_stats), the in-memory flow store/processor, flow expiry, or the aggregator stats payload ingested by ns-stats. Covers the netifyd Unix-socket message format, polymorphic unmarshalling, digest keying, stats accumulation rules, and byte-counter semantics."
---

# netifyd Flow Protocol

netifyd (Netify Agent v5) is a DPI daemon. It streams newline-delimited JSON
events over a Unix socket (default `/var/run/netifyd/flows.sock`). `ns-flows`
consumes these; `ns-stats` ingests aggregated summaries produced by the Aggregator plugin.

Source files:
- Protocol structs & unmarshal: [flows/parser.go](../../../flows/parser.go)
- In-memory store: [flows/processor.go](../../../flows/processor.go)
- Stats ingestion: [stats/stats.go](../../../stats/stats.go)

Upstream reference docs (load on demand):
- [references/flow.md](references/flow.md) — `flow_dpi_complete` full attribute reference
- [references/flow-stats.md](references/flow-stats.md) — `flow_stats` attribute reference
- [references/flow-purge.md](references/flow-purge.md) — `flow_purge` attribute reference
- [references/aggregator-type3.md](references/aggregator-type3.md) — Aggregator Type 3 payload (ns-stats input)

---

## Event envelope

Every message is a `FlowEvent`:

```go
type FlowEvent struct {
    Type      string `json:"type"`       // discriminator — see table below
    Interface string `json:"interface,omitempty"`
    Internal  bool   `json:"internal,omitempty"`
    Reason    string `json:"reason,omitempty"` // "closed" | "expired" (flow_purge only)
    Flow      any    `json:"flow"`             // polymorphic value, decoded by Type
}
```

`Flow` is decoded by a **custom `UnmarshalJSON`**: it reads `type`, then unmarshals `flow`
into the matching concrete struct. After decoding `Flow` holds a **value, not a pointer**:
always type-switch on `event.Flow.(type)`.

## Event types

| `type` wire value | Go constant | concrete struct | when emitted |
|---|---|---|---|
| `flow_dpi_complete` | `FlowTypeDpiComplete` | `FlowComplete` | DPI engine finished analysis; canonical record with full metadata |
| `flow_stats` | `FlowTypeStats` | `FlowStats` | periodic (~15 s) counters update for a live flow |
| `flow_purge` | `FlowTypePurge` | `FlowPurge` | flow removed from engine (TCP close / inactivity timeout) |

netifyd also emits `flow` and `flow_dpi_update` — these are intentionally **not handled**.
Unknown `type` → `ErrUnsupportedFlowType`. **Expected, not fatal**: callers check
`errors.Is(err, ErrUnsupportedFlowType)` and skip. Decode failures of a known type wrap
the cause: `fmt.Errorf("malformed %q flow: %w", type, err)`.

## Digest stability

The `digest` field is a 7-tuple hash (src/dst IP+port, IP proto, VLAN, interface). As DPI
refines the flow, the current `digest` can change — previous values accumulate in
`digest_prev`. **`digest_prev[0]` is always the stable 7-tuple digest** (unchanged over
the flow lifetime, Netify ≥ 5.2). The processor keys the in-memory store by the **current
`Digest`** from the Go struct.

## Counter semantics — critical

| field group | `flow_dpi_complete` | `flow_stats` | `flow_purge` |
|---|---|---|---|
| `local_bytes` / `other_bytes` | activity during detection window | **per-interval delta** (not cumulative) | activity in final interval |
| `total_bytes` / `total_packets` | cumulative to this point | **cumulative session total** | **final cumulative total** |
| `local_rate` / `other_rate` | burst rate in current lifecycle | **burst rate — not additive** | burst rate at close |

The processor **adds** `LocalBytes`/`OtherBytes` from `flow_stats` onto the stored
`FlowComplete` (per-interval deltas accumulate), **replaces** rates and `Total*` (already
cumulative). Do not sum rates across events.

## Key structs

```go
// Shared counter group — embedded in FlowComplete, FlowPurge, FlowStats
type Stats struct {
    LocalBytes   int64   `json:"local_bytes"`
    LocalPackets int     `json:"local_packets"`
    LocalRate    float64 `json:"local_rate"`
    OtherBytes   int64   `json:"other_bytes"`
    OtherPackets int     `json:"other_packets"`
    OtherRate    float64 `json:"other_rate"`
    TotalBytes   int64   `json:"total_bytes"`
    TotalPackets int     `json:"total_packets"`
}

// Unique flow key — embedded in all three concrete types
type FlowBase struct { Digest string `json:"digest"` }
```

Optional protocol blocks on `FlowComplete` are **pointer fields with `,omitempty`**
(`*Ssl`, `*Http`, `*Tcp`, `*Dhcp`, `*Ssh`, `*Ssdp`, `*Stun`, `*Bt`, `*Mdns`, `*Nfq`,
`*GtpFlow`, `*Category`). Nil-check before use; follow the same pattern for new ones.

Timestamps `FirstSeenAt` / `LastSeenAt` are **Unix milliseconds** (`int64`).
Always use `time.UnixMilli`, never `time.Unix`.

## In-memory store (`FlowProcessor`)

`map[string]FlowEvent` keyed by `Digest`, guarded by `sync.RWMutex`.

```go
type FlowAccessor interface { GetEvents() map[string]FlowEvent }
type FlowIngestor interface { Process(event FlowEvent) }
```

`Process` rules (write lock held throughout):

- **`FlowComplete`** → stored/replaced directly under its digest.
- **`FlowStats`** → looks up existing `FlowComplete`; if absent logs and returns.
  Adds `Local*`/`Other*` byte+packet deltas, replaces rates, `Total*`, and `LastSeenAt`.
- **`FlowPurge`** → looks up existing `FlowComplete`; replaces `TotalBytes`/`TotalPackets`
  with final values. If unknown, logs and returns.

`GetEvents()` returns a **copy** of the map — callers must not mutate it.

`PurgeFlowsOlderThan(d)` removes `FlowComplete` entries whose `LastSeenAt` is before
`now - d`. Driven by a 10-second ticker in `cmd/ns-flows/main.go`.

## Aggregator Type 3 (ns-stats input)

`ns-stats` ingests POST `/stats` payloads produced by the netifyd Aggregator plugin
configured to type 3. This produces per-hour, per-flow summaries keyed by
(application, protocol, interface, IP/MAC, direction). See
[references/aggregator-type3.md](references/aggregator-type3.md) for the full schema.

Key point: `detected_application_name` values from the Aggregator arrive as
`"netify.<tag>"` (e.g. `"netify.google-chat"`). Older netifyd versions may include a
numeric ID prefix (e.g. `"10910.netify.google-chat"`). The `stripAppNameID` function in
`stats/stats.go` normalises this by stripping any leading `<digits>.` prefix.

## Edit checklist

1. JSON tags: `snake_case`, `,omitempty` for optional fields; pointer for optional
   sub-objects; value embed for shared groups (`FlowBase`, `Stats`).
2. If a new field must accumulate across `flow_stats` intervals, add accumulation logic in
   `FlowProcessor.Process`; otherwise the field is silently overwritten on each update.
3. Add/update table-driven tests in [flows/parser_test.go](../../../flows/parser_test.go)
   and [flows/processor_test.go](../../../flows/processor_test.go). Use
   `go-playground/assert/v2`; mark shared helpers with `t.Helper()`.
4. If a field surfaces through the HTTP API, sync [openapi.yaml](../../../openapi.yaml).
5. Run `make format && make lint && make test`.
