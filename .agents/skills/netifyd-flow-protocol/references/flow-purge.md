# Flow Purge Telemetry Reference (`flow_purge`)

Source: https://www.netify.ai/documentation/agent/v5/integrations/telemetry/flow-purge

Emitted when a tracked flow is removed from the engine — after TCP close,
inactivity timeout, or internal purge. Carries **final** counters and
end-state details. Use for post-session analysis, accounting, and flow
lifecycle validation.

Go struct: `flows.FlowPurge` (embeds `FlowBase`, `Stats`).

## Envelope fields

| field | type | description |
|---|---|---|
| `type` | string | always `flow_purge` |
| `interface` | string | interface name |
| `internal` | bool | internal to local network |
| `reason` | string | `closed` (TCP/normal close) or `expired` (inactivity timeout) |

The `reason` field maps to `FlowEvent.Reason` on the Go struct.

## `flow` object fields

| JSON field | Go field | type | description |
|---|---|---|---|
| `digest` | `Digest` | string | Current flow digest |
| `digest_prev` | `DigestPrev` | []string | Previous digests; `[0]` is stable 7-tuple |
| `last_seen_at` | `LastSeenAt` | int64 | Unix epoch **milliseconds** |
| `detection_packets` | `DetectionPackets` | int | Packets used for detection |
| `local_bytes` | `LocalBytes` | int64 | Bytes from local endpoint in final interval |
| `local_packets` | `LocalPackets` | int | Packets from local endpoint in final interval |
| `local_rate` | `LocalRate` | float64 | Burst rate at close |
| `other_bytes` | `OtherBytes` | int64 | Bytes from remote endpoint in final interval |
| `other_packets` | `OtherPackets` | int | Packets from remote endpoint in final interval |
| `other_rate` | `OtherRate` | float64 | Burst rate at close |
| `total_bytes` | `TotalBytes` | int64 | **Final cumulative** session total bytes |
| `total_packets` | `TotalPackets` | int | **Final cumulative** session total packets |
| `tcp` | `Tcp` | `*Tcp` | TCP counters — omitted for non-TCP flows |

`Tcp` sub-fields: `resets`, `retrans`, `seq_errors` (all int).

## Processor behaviour

`Process` for `FlowPurge`:
- Looks up the existing `FlowComplete` by digest; if absent, logs and returns.
- **Replaces** `TotalBytes` and `TotalPackets` only (final authoritative totals).
- The entry is retained in the store until `PurgeFlowsOlderThan` removes it.

```go
// What Process() does for FlowPurge:
toUpdateFlow.TotalBytes   = f.TotalBytes   // replace with final
toUpdateFlow.TotalPackets = f.TotalPackets // replace with final
```

## Example JSON

```json
{
  "flow": {
    "detection_packets": 12,
    "digest": "fb69e87ed3b...",
    "digest_prev": ["d7fddd35cc27..."],
    "last_seen_at": 1772665905392,
    "local_bytes": 0,
    "local_packets": 0,
    "local_rate": 387,
    "other_bytes": 0,
    "other_packets": 0,
    "other_rate": 300,
    "tcp": { "resets": 0, "retrans": 0, "seq_errors": 0 },
    "total_bytes": 13968,
    "total_packets": 51
  },
  "interface": "wlp3s0",
  "internal": true,
  "reason": "closed",
  "type": "flow_purge"
}
```
