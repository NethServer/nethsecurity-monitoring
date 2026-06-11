# Flow Stats Telemetry Reference (`flow_stats`)

Source: https://www.netify.ai/documentation/agent/v5/integrations/telemetry/flow-stats

Emitted periodically (~every 15 seconds) for active flows. Provides ongoing
packet, byte, and rate counters while a flow remains open. Most useful for
near-real-time monitoring of long-lived sessions (streaming, VPN, persistent services).

Go struct: `flows.FlowStats` (embeds `FlowBase`, `Stats`).

## Critical: byte counter semantics

`local_bytes` / `other_bytes` in `flow_stats` are **per-interval deltas**, not
cumulative. The processor **adds** them onto the stored `FlowComplete`.

`total_bytes` / `total_packets` are **cumulative session totals**. The processor
**replaces** the stored values with these on each stats event.

`local_rate` / `other_rate` are **burst rates** (measured over the flow's active
duration within the interval, not the full 15-second window). They are **not additive**
across flows or events. The processor replaces them.

> Example: a 1-second flow within a 15-second interval reports its full 1-second
> bandwidth as the rate, not 1/15th of it.

## Envelope fields

| field | type | value |
|---|---|---|
| `type` | string | always `flow_stats` |
| `interface` | string | interface name |
| `internal` | bool | internal to local network |

## `flow` object fields

| JSON field | Go field | type | description |
|---|---|---|---|
| `digest` | `Digest` | string | Current flow digest |
| `digest_prev` | `DigestPrev` | []string | Previous digests; `[0]` is stable 7-tuple |
| `last_seen_at` | `LastSeenAt` | int64 | Unix epoch **milliseconds** |
| `detection_packets` | `DetectionPackets` | int | Packets used for detection |
| `local_bytes` | `LocalBytes` | int64 | **Per-interval delta** bytes from local endpoint |
| `local_packets` | `LocalPackets` | int | Per-interval packet delta, local |
| `local_rate` | `LocalRate` | float64 | Burst rate, local (not additive) |
| `other_bytes` | `OtherBytes` | int64 | **Per-interval delta** bytes from remote endpoint |
| `other_packets` | `OtherPackets` | int | Per-interval packet delta, remote |
| `other_rate` | `OtherRate` | float64 | Burst rate, remote (not additive) |
| `total_bytes` | `TotalBytes` | int64 | **Cumulative** session total bytes |
| `total_packets` | `TotalPackets` | int | **Cumulative** session total packets |
| `tcp` | `Tcp` | `*Tcp` | TCP counters — omitted for non-TCP flows |

`Tcp` sub-fields: `resets`, `retrans`, `seq_errors` (all int).

## Processor accumulation rules

```go
// What Process() does for FlowStats:
toUpdateFlow.LastSeenAt    = f.LastSeenAt        // replace
toUpdateFlow.LocalBytes   += f.LocalBytes        // ADD delta
toUpdateFlow.LocalPackets += f.LocalPackets      // ADD delta
toUpdateFlow.LocalRate     = f.LocalRate         // replace (burst)
toUpdateFlow.OtherBytes   += f.OtherBytes        // ADD delta
toUpdateFlow.OtherPackets += f.OtherPackets      // ADD delta
toUpdateFlow.OtherRate     = f.OtherRate         // replace (burst)
toUpdateFlow.TotalBytes    = f.TotalBytes        // replace (cumulative)
toUpdateFlow.TotalPackets  = f.TotalPackets      // replace (cumulative)
```

## Example JSON

```json
{
  "flow": {
    "detection_packets": 2,
    "digest": "c3086c57745b...",
    "digest_prev": ["fcd1061d4c2ba844..."],
    "last_seen_at": 1772665318561,
    "local_bytes": 5559,
    "local_packets": 6,
    "local_rate": 5559,
    "other_bytes": 913,
    "other_packets": 6,
    "other_rate": 913,
    "tcp": { "resets": 0, "retrans": 0, "seq_errors": 0 },
    "total_bytes": 613175,
    "total_packets": 1064
  },
  "interface": "wlp3s0",
  "internal": true,
  "type": "flow_stats"
}
```
