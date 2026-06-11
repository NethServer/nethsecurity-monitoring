# Aggregator Telemetry — Type 3 Reference

Source: https://www.netify.ai/documentation/agent/v5/integrations/telemetry/aggregator-type3

The netifyd Aggregator plugin (configured to type 3) produces per-hour, per-flow
summaries. `ns-stats` ingests these via POST `/stats`.

Go struct: `stats.AggregatorPayload` wrapping `[]stats.AggregatorEntry`
(defined in [stats/stats.go](../../../../stats/stats.go)).

## Data profile

**Dimensions**: application, protocol, interface, IP protocol, IP version, local
origin, local IP, local MAC, other IP, other MAC, other type.

**Metrics**: download bytes, upload bytes, packets, flow digest list.

**Note**: Includes `other_port` (typically the remote/server port) but **omits**
`local_port` (ephemeral). Excludes NATed flows.

## Payload wrapper

```go
type AggregatorPayload struct {
    LogTimeEnd int64             `json:"log_time_end"` // Unix seconds marking hour end
    Stats      []AggregatorEntry `json:"stats"`
}
```

`LogTimeEnd` is used to key the hour bucket in the `aggregator_batches` table.

## `AggregatorEntry` fields

| JSON field | Go field | type | description |
|---|---|---|---|
| `detected_application` | `DetectedApplication` | int | Netify application ID |
| `detected_application_name` | `DetectedApplicationName` | string | Netify app tag (e.g. `netify.google-chat`) |
| `detected_protocol` | `DetectedProtocol` | int | Netify protocol ID |
| `detected_protocol_name` | `DetectedProtocolName` | string | Protocol name (e.g. `HTTP/S`, `QUIC`) |
| `interface` | `Interface` | string | Interface name |
| `ip_protocol` | `IpProtocol` | int | IANA protocol number (6=TCP, 17=UDP) |
| `ip_version` | `IpVersion` | int | `4` or `6` |
| `local_origin` | `LocalOrigin` | bool | Local endpoint originated the connection |
| `local_ip` | `LocalIp` | string | Local endpoint IP (omitted if privacy mode enabled) |
| `local_mac` | `LocalMac` | string | Local endpoint MAC (omitted if privacy mode enabled) |
| `other_ip` | `OtherIp` | string | Remote endpoint IP |
| `other_port` | `OtherPort` | int | Remote endpoint port |
| `other_type` | `OtherType` | string | `local`, `remote`, `broadcast`, `multicast`, `unsupported`, `error`, `unknown` |
| `local_bytes` | `LocalBytes` | int64 | Total bytes from local endpoint |
| `other_bytes` | `OtherBytes` | int64 | Total bytes from remote endpoint |
| `packets` | `Packets` | int | Total packet count |
| `digests` | `Digests` | []string | Flow digest identifiers in this bucket |

## Application name normalisation

`detected_application_name` arrives as `netify.<tag>` (e.g. `netify.google-chat`).
Older netifyd versions may include a leading numeric ID prefix
(e.g. `10910.netify.google-chat`). The `stripAppNameID` function in `stats/stats.go`
strips any leading `<digits>.` prefix before persistence.

## Database mapping

Each `AggregatorEntry` is inserted into the `aggregator_stats` table as one row.
The `other_ip` is also recorded without a resolved hostname initially
(`other_host IS NULL`); `reverse_dns` fills it in asynchronously.

See `stats/stats.go` for the full schema (`aggregator_batches`, `aggregator_stats`).

## Example flat-mode JSON (single entry)

```json
{
  "detected_application": 10033,
  "detected_application_name": "netify.netify",
  "detected_protocol": 196,
  "detected_protocol_name": "HTTP/S",
  "digests": ["5dd5bb2c827c677ee3f904d40ee0b0ce512234b8"],
  "interface": "wlp1s0",
  "internal": true,
  "ip_protocol": 6,
  "ip_version": 4,
  "local_bytes": 3095,
  "local_ip": "192.168.1.100",
  "local_mac": "00:00:00:00:00:00",
  "local_origin": true,
  "other_bytes": 457,
  "other_ip": "148.113.141.168",
  "other_port": 443,
  "other_type": "remote",
  "packets": 8
}
```

## Full payload wrapper example

```json
{
  "log_time_end": 1772668800,
  "stats": [
    { "...": "entry as above" }
  ]
}
```
