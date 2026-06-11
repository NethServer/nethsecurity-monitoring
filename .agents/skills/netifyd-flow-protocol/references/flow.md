# Flow Telemetry Reference (`flow_dpi_complete`)

Source: https://www.netify.ai/documentation/agent/v5/integrations/telemetry/flow

Emitted when the DPI engine completes analysis. Use for event-driven detection,
enrichment, and routing. For ongoing bandwidth use `flow_stats`; for final
session counters use `flow_purge`.

Go struct: `flows.FlowComplete` (embeds `FlowBase`, `Stats`).

## Detection lifecycle

netifyd emits three stages: `flow` → `flow_dpi_update` → `flow_dpi_complete`.
Only `flow_dpi_complete` is handled by this project. The first two are silently
ignored via `ErrUnsupportedFlowType`.

## Envelope fields

| field | type | notes |
|---|---|---|
| `type` | string | `flow`, `flow_dpi_update`, or `flow_dpi_complete` |
| `interface` | string | Interface name |
| `internal` | bool | Flow internal to local network |

## `flow` object fields

### Identification

| JSON field | Go field | type | description |
|---|---|---|---|
| `digest` | `Digest` | string | Current flow digest (7-tuple hash; may change as DPI refines) |
| `digest_prev` | `DigestPrev` | []string | Previous digests; `[0]` is always the stable 7-tuple digest (Netify ≥ 5.2) |
| `first_seen_at` | `FirstSeenAt` | int64 | Unix epoch **milliseconds** |
| `last_seen_at` | `LastSeenAt` | int64 | Unix epoch **milliseconds** |
| `vlan_id` | `VlanId` | int | Observed VLAN ID |

### Endpoints

| JSON field | Go field | type | description |
|---|---|---|---|
| `local_ip` | `LocalIp` | string | Local endpoint IP |
| `local_mac` | `LocalMac` | string | Local endpoint MAC |
| `local_port` | `LocalPort` | int | Local endpoint port |
| `local_origin` | `LocalOrigin` | bool | Local endpoint initiated the connection |
| `other_ip` | `OtherIp` | string | Remote endpoint IP |
| `other_mac` | `OtherMac` | string | Remote endpoint MAC |
| `other_port` | `OtherPort` | int | Remote endpoint port |
| `other_type` | `OtherType` | string | `local`, `remote`, `broadcast`, `multicast`, `unsupported`, `error`, `unknown` |
| `ip_protocol` | `IpProtocol` | int | IANA IP protocol number (6=TCP, 17=UDP, …) |
| `ip_version` | `IpVersion` | int | `4` or `6` |
| `ip_nat` | `IpNat` | bool | NAT detected |
| `ip_dscp` | `IpDscp` | int | DSCP value |

### DPI classification

| JSON field | Go field | type | description |
|---|---|---|---|
| `detected_application` | `DetectedApplication` | int | Netify application ID |
| `detected_application_name` | `DetectedApplicationName` | string | Netify application tag (e.g. `netify.tailscale`) |
| `detected_protocol` | `DetectedProtocol` | int | Netify protocol ID |
| `detected_protocol_name` | `DetectedProtocolName` | string | Protocol name from DPI driver (e.g. `HTTP/S`) |
| `detection_guessed` | `DetectionGuessed` | bool | Classification inferred by default port (not fully dissected) |
| `detection_packets` | `DetectionPackets` | int | Packets used to complete DPI |
| `detection_updated` | `DetectionUpdated` | bool | Additional packets updated earlier classification |
| `dhc_hit` | `DhcHit` | bool | Domain hint cache contributed to classification |
| `fhc_hit` | `FhcHit` | bool | Flow hash cache contributed to classification |
| `soft_dissector` | `SoftDissector` | bool | Soft dissector was used |
| `app_ip_override` | `AppIpOverride` | bool | Application IP override logic applied |
| `app_proto_twins` | `AppProtoTwins` | bool | Protocol twin detection applied |

### Hostnames

| JSON field | Go field | type | description |
|---|---|---|---|
| `dns_host_name` | `DnsHostName` | string | Hostname from a corresponding DNS query |
| `host_server_name` | `HostServerName` | string | Hostname from protocol metadata (TLS SNI or HTTP Host) |

### Counters (embedded `Stats`)

Bytes/packets reflect activity during the detection window, **not** session totals.
See byte-counter semantics in SKILL.md.

| JSON field | Go field | type |
|---|---|---|
| `local_bytes` | `LocalBytes` | int64 |
| `local_packets` | `LocalPackets` | int |
| `local_rate` | `LocalRate` | float64 (burst rate) |
| `other_bytes` | `OtherBytes` | int64 |
| `other_packets` | `OtherPackets` | int |
| `other_rate` | `OtherRate` | float64 (burst rate) |
| `total_bytes` | `TotalBytes` | int64 |
| `total_packets` | `TotalPackets` | int |

### Tags & risks

| JSON field | Go field | type | description |
|---|---|---|---|
| `tags` | `Tags` | []string | Tags associated with the flow |
| `risks.ndpi_risk_score` | `Risks.NdpiRiskScore` | int | Aggregate nDPI risk score |
| `risks.ndpi_risk_score_client` | `Risks.NdpiRiskScoreClient` | int | Client-side risk score |
| `risks.ndpi_risk_score_server` | `Risks.NdpiRiskScoreServer` | int | Server-side risk score |
| `risks.risks` | `Risks.Risks` | []int | nDPI risk identifiers triggered |

### Optional protocol sub-objects (all pointer fields, `,omitempty`)

| JSON key | Go field | struct | key fields |
|---|---|---|---|
| `ssl` | `Ssl` | `*Ssl` | `client_sni`, `server_cn`, `fingerprint`, `cipher_suite`, `client_ja4`, `version`, `alpn`, `alpn_server`, `issuer_dn`, `subject_dn`, `encrypted_ch_version` |
| `http` | `Http` | `*Http` | `url`, `user_agent` |
| `tcp` | `Tcp` | `*Tcp` | `resets`, `retrans`, `seq_errors` |
| `ssh` | `Ssh` | `*Ssh` | `client`, `server` |
| `dhcp` | `Dhcp` | `*Dhcp` | `fingerprint`, `class_ident` |
| `bt` | `Bt` | `*Bt` | `info_hash` |
| `mdns` | `Mdns` | `*Mdns` | `answer` |
| `ssdp` | `Ssdp` | `*Ssdp` | `user_agent` |
| `stun` | `Stun` | `*Stun` | `mapped`, `other`, `peer`, `relayed`, `response` |
| `nfq` | `Nfq` | `*Nfq` | `src_iface`, `dst_iface` |
| `gtp` | `Gtp` | `*GtpFlow` | `local_ip`, `local_port`, `local_teid`, `other_ip`, `other_port`, `other_teid`, `version`, `ip_version`, `ip_dscp`, `other_type` |
| `category` | `Category` | `*Category` | `application`, `domain`, `local_network`, `other_network`, `overlay`, `protocol` |

## Example JSON

```json
{
  "flow": {
    "digest": "c4c07ca55baa19a7fe3652bcd356765a7...",
    "digest_prev": ["463c53093403fcce8eeb01df5b5125df66a0f53b"],
    "first_seen_at": 1772738467573,
    "last_seen_at": 1772738467684,
    "local_ip": "192.168.4.44",
    "local_mac": "f8:e9:03:01:69:13",
    "local_port": 35636,
    "local_origin": true,
    "other_ip": "192.200.0.102",
    "other_mac": "3c:7c:3f:a1:ed:58",
    "other_port": 443,
    "other_type": "remote",
    "ip_protocol": 6,
    "ip_version": 4,
    "detected_application": 11354,
    "detected_application_name": "netify.tailscale",
    "detected_protocol": 196,
    "detected_protocol_name": "HTTP/S",
    "dns_host_name": "login.tailscale.com",
    "host_server_name": "login.tailscale.com",
    "ssl": {
      "client_sni": "login.tailscale.com",
      "alpn": ["h2", "http/1.1"],
      "cipher_suite": "0x0000",
      "client_ja4": "t13d1817h2_e8a523a41297_...",
      "encrypted_ch_version": "0xfe0d",
      "version": "0x0303"
    },
    "risks": { "ndpi_risk_score": 0, "ndpi_risk_score_client": 0, "ndpi_risk_score_server": 0, "risks": [] }
  },
  "interface": "wlp3s0",
  "internal": true,
  "type": "flow_dpi_complete"
}
```
