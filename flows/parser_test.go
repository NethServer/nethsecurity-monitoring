package flows

import (
	"encoding/json"
	"strings"
	"testing"
)

const DpiCompleteFlowExample = `
{
  "flow": {
    "app_ip_override": false,
    "app_proto_twins": false,
    "category": {
      "application": 14,
      "domain": 0,
      "local_network": 0,
      "other_network": 0,
      "overlay": 0,
      "protocol": 18
    },
    "conntrack": {
      "id": 163461572,
      "mark": 0,
      "reply_dst_ip": "192.168.1.100",
      "reply_dst_port": 44743,
      "reply_src_ip": "203.0.113.88",
      "reply_src_port": 443
    },
    "detected_application": 10733,
    "detected_application_name": "netify.cloud-service",
    "detected_protocol": 188,
    "detected_protocol_name": "QUIC",
    "detection_guessed": false,
    "detection_packets": 3,
    "detection_updated": true,
    "dhc_hit": false,
    "digest": "c3d4e5f6a7b890123456789012345678abcdef01",
    "digest_prev": [
      "0123456789abcdef0123456789abcdef01234567",
      "abcdef0123456789abcdef0123456789abcdef01"
    ],
    "dns_host_name": "analytics.example-cloud.io",
    "fhc_hit": false,
    "first_seen_at": 1765893622750,
    "host_server_name": "api.example-cloud.io",
    "ip_dscp": 0,
    "ip_nat": false,
    "ip_protocol": 17,
    "ip_version": 4,
    "last_seen_at": 1765893622753,
    "local_bytes": 6583,
    "local_ip": "192.168.1.100",
    "local_mac": "00:00:00:00:00:00",
    "local_origin": true,
    "local_packets": 6,
    "local_port": 44743,
    "local_rate": 6583.0,
    "nfq": {
      "dst_iface": "eth1",
      "src_iface": "wg1"
    },
    "other_bytes": 0,
    "other_ip": "203.0.113.88",
    "other_mac": "aa:bb:cc:dd:ee:03",
    "other_packets": 0,
    "other_port": 443,
    "other_rate": 0.0,
    "other_type": "remote",
    "risks": {
      "ndpi_risk_score": 110,
      "ndpi_risk_score_client": 95,
      "ndpi_risk_score_server": 15,
      "risks": [
        39,
        46
      ]
    },
    "soft_dissector": false,
    "ssl": {
      "alpn": [
        "h3"
      ],
      "alpn_server": [],
      "cipher_suite": "0x0000",
      "client_ja4": "t13d1817h2_e8a523a41297",
      "client_sni": "api.example-cloud.io",
      "encrypted_ch_version": "0xfe0d",
      "fingerprint": "sha1fingerprint",
      "issuer_dn": "Let's Encrypt",
      "server_cn": "example-cloud.io",
      "subject_dn": "example-cloud.io",
      "version": "0x0304"
    },
    "tcp": {
      "resets": 0,
      "retrans": 0,
      "seq_errors": 0
    },
    "total_bytes": 0,
    "total_packets": 0,
    "tags": ["test-tag"],
    "vlan_id": 0
  },
  "interface": "lan",
  "internal": true,
  "type": "flow_dpi_complete"
}`

const DpiPurgeFlowExample = `
{
  "flow": {
    "detection_packets": 2,
    "digest": "d4e5f6a7b8c901234567890123456789bcdef012",
    "digest_prev": [
      "23456789abcdef0123456789abcdef0123456789"
    ],
    "last_seen_at": 1765893726667,
    "local_bytes": 256,
    "local_packets": 1,
    "local_rate": 128.0,
    "other_bytes": 158,
    "other_packets": 1,
    "other_rate": 79.0,
    "tcp": {
      "resets": 0,
      "retrans": 0,
      "seq_errors": 0
    },
    "total_bytes": 414,
    "total_packets": 2
  },
  "interface": "lan",
  "internal": true,
  "reason": "expired",
  "type": "flow_purge"
}`

const DpiStatsFlowExample = `
{
  "flow": {
    "detection_packets": 32,
    "digest": "e5f6a7b8c9d012345678901234567890cdef0123",
    "digest_prev": [
      "3456789abcdef0123456789abcdef01234567890",
      "bcdef0123456789abcdef0123456789abcdef012"
    ],
    "last_seen_at": 1765893801667,
    "local_bytes": 2028,
    "local_packets": 34,
    "local_rate": 1510.6666259765625,
    "other_bytes": 29478,
    "other_packets": 37,
    "other_rate": 29478.0,
    "tcp": {
      "resets": 0,
      "retrans": 0,
      "seq_errors": 0
    },
    "total_bytes": 9404196,
    "total_packets": 18450
  },
  "interface": "lan",
  "internal": true,
  "type": "flow_stats"
}`

func TestParsingFlow(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantType      string
		wantInterface string
		wantInternal  bool
		wantReason    string
		checkFlow     func(t *testing.T, flow any)
	}{
		{
			name:          "DPI complete flow",
			input:         DpiCompleteFlowExample,
			wantType:      FlowTypeDpiComplete,
			wantInterface: "lan",
			wantInternal:  true,
			checkFlow: func(t *testing.T, flow any) {
				f := flow.(FlowComplete)
				// Base and core fields
				assertEqual(t, f.Digest, "c3d4e5f6a7b890123456789012345678abcdef01", "Digest")
				assertEqual(t, f.Conntrack.Id, 163461572, "Conntrack.Id")
				assertEqual(t, f.Conntrack.Mark, 0, "Conntrack.Mark")
				assertEqual(t, f.Conntrack.ReplyDstIp, "192.168.1.100", "Conntrack.ReplyDstIp")
				assertEqual(t, f.Conntrack.ReplyDstPort, 44743, "Conntrack.ReplyDstPort")
				assertEqual(t, f.Conntrack.ReplySrcIp, "203.0.113.88", "Conntrack.ReplySrcIp")
				assertEqual(t, f.Conntrack.ReplySrcPort, 443, "Conntrack.ReplySrcPort")

				// Detection fields
				assertEqual(
					t,
					f.DetectedApplicationName,
					"netify.cloud-service",
					"DetectedApplicationName",
				)
				assertEqual(t, f.DetectedApplication, 10733, "DetectedApplication")
				assertEqual(t, f.DetectedProtocol, 188, "DetectedProtocol")
				assertEqual(t, f.DetectedProtocolName, "QUIC", "DetectedProtocolName")
				assertEqual(t, f.DetectionGuessed, false, "DetectionGuessed")
				assertEqual(t, f.DetectionPackets, 3, "DetectionPackets")
				assertEqual(t, f.DetectionUpdated, true, "DetectionUpdated")

				// Timestamps
				assertEqual(t, f.FirstSeenAt, int64(1765893622750), "FirstSeenAt")
				assertEqual(t, f.LastSeenAt, int64(1765893622753), "LastSeenAt")

				// IP and network fields
				assertEqual(t, f.LocalIp, "192.168.1.100", "LocalIp")
				assertEqual(t, f.LocalMac, "00:00:00:00:00:00", "LocalMac")
				assertEqual(t, f.LocalOrigin, true, "LocalOrigin")
				assertEqual(t, f.LocalPort, 44743, "LocalPort")
				assertEqual(t, f.OtherIp, "203.0.113.88", "OtherIp")
				assertEqual(t, f.OtherMac, "aa:bb:cc:dd:ee:03", "OtherMac")
				assertEqual(t, f.OtherPort, 443, "OtherPort")
				assertEqual(t, f.OtherType, "remote", "OtherType")

				// DNS and hostname fields
				assertEqual(t, f.DnsHostName, "analytics.example-cloud.io", "DnsHostName")
				assertEqual(t, f.HostServerName, "api.example-cloud.io", "HostServerName")

				// Stats
				assertEqual(t, f.LocalBytes, int64(6583), "LocalBytes")
				assertEqual(t, f.LocalPackets, 6, "LocalPackets")
				assertEqual(t, f.LocalRate, 6583.0, "LocalRate")
				assertEqual(t, f.OtherBytes, int64(0), "OtherBytes")
				assertEqual(t, f.OtherPackets, 0, "OtherPackets")
				assertEqual(t, f.OtherRate, 0.0, "OtherRate")
				assertEqual(t, f.TotalBytes, int64(0), "TotalBytes")
				assertEqual(t, f.TotalPackets, 0, "TotalPackets")

				// Risk scores
				assertEqual(t, f.Risks.NdpiRiskScore, 110, "Risks.NdpiRiskScore")
				assertEqual(t, f.Risks.NdpiRiskScoreClient, 95, "Risks.NdpiRiskScoreClient")
				assertEqual(t, f.Risks.NdpiRiskScoreServer, 15, "Risks.NdpiRiskScoreServer")
				assertSliceEqual(t, f.Risks.Risks, []int{39, 46}, "Risks.Risks")

				// New fields
				assertEqual(t, f.AppIpOverride, false, "AppIpOverride")
				assertEqual(t, f.AppProtoTwins, false, "AppProtoTwins")
				assertEqual(t, f.DhcHit, false, "DhcHit")
				assertEqual(t, f.FhcHit, false, "FhcHit")
				assertEqual(t, f.IpDscp, 0, "IpDscp")
				assertEqual(t, f.IpNat, false, "IpNat")
				assertEqual(t, f.IpProtocol, 17, "IpProtocol")
				assertEqual(t, f.IpVersion, 4, "IpVersion")
				assertEqual(t, f.SoftDissector, false, "SoftDissector")
				assertEqual(t, f.VlanId, 0, "VlanId")
				assertSliceEqual(
					t,
					f.DigestPrev,
					[]string{
						"0123456789abcdef0123456789abcdef01234567",
						"abcdef0123456789abcdef0123456789abcdef01",
					},
					"DigestPrev",
				)
				assertSliceEqual(t, f.Tags, []string{"test-tag"}, "Tags")

				// Category
				if f.Category != nil {
					assertEqual(t, f.Category.Application, 14, "Category.Application")
					assertEqual(t, f.Category.Domain, 0, "Category.Domain")
					assertEqual(t, f.Category.LocalNetwork, 0, "Category.LocalNetwork")
					assertEqual(t, f.Category.OtherNetwork, 0, "Category.OtherNetwork")
					assertEqual(t, f.Category.Overlay, 0, "Category.Overlay")
					assertEqual(t, f.Category.Protocol, 18, "Category.Protocol")
				}

				// Nfq
				if f.Nfq != nil {
					assertEqual(t, f.Nfq.DstIface, "eth1", "Nfq.DstIface")
					assertEqual(t, f.Nfq.SrcIface, "wg1", "Nfq.SrcIface")
				}

				// SSL
				if f.Ssl != nil {
					assertSliceEqual(t, f.Ssl.Alpn, []string{"h3"}, "Ssl.Alpn")
					assertEqual(t, f.Ssl.CipherSuite, "0x0000", "Ssl.CipherSuite")
					assertEqual(t, f.Ssl.ClientJa4, "t13d1817h2_e8a523a41297", "Ssl.ClientJa4")
					assertEqual(t, f.Ssl.ClientSni, "api.example-cloud.io", "Ssl.ClientSni")
					assertEqual(t, f.Ssl.EncryptedChVersion, "0xfe0d", "Ssl.EncryptedChVersion")
					assertEqual(t, f.Ssl.Version, "0x0304", "Ssl.Version")
				}

				// TCP
				if f.Tcp != nil {
					assertEqual(t, f.Tcp.Resets, 0, "Tcp.Resets")
					assertEqual(t, f.Tcp.Retrans, 0, "Tcp.Retrans")
					assertEqual(t, f.Tcp.SeqErrors, 0, "Tcp.SeqErrors")
				}
			},
		},
		{
			name:          "DPI purge flow",
			input:         DpiPurgeFlowExample,
			wantType:      FlowTypePurge,
			wantInterface: "lan",
			wantInternal:  true,
			wantReason:    "expired",
			checkFlow: func(t *testing.T, flow any) {
				f := flow.(FlowPurge)
				assertEqual(t, f.Digest, "d4e5f6a7b8c901234567890123456789bcdef012", "Digest")
				assertEqual(t, f.DetectionPackets, 2, "DetectionPackets")
				assertSliceEqual(
					t,
					f.DigestPrev,
					[]string{"23456789abcdef0123456789abcdef0123456789"},
					"DigestPrev",
				)
				assertEqual(t, f.LastSeenAt, int64(1765893726667), "LastSeenAt")
				assertEqual(t, f.LocalBytes, int64(256), "LocalBytes")
				assertEqual(t, f.LocalPackets, 1, "LocalPackets")
				assertEqual(t, f.LocalRate, 128.0, "LocalRate")
				assertEqual(t, f.OtherBytes, int64(158), "OtherBytes")
				assertEqual(t, f.OtherPackets, 1, "OtherPackets")
				assertEqual(t, f.OtherRate, 79.0, "OtherRate")
				assertEqual(t, f.TotalBytes, int64(414), "TotalBytes")
				assertEqual(t, f.TotalPackets, 2, "TotalPackets")
				if f.Tcp != nil {
					assertEqual(t, f.Tcp.Resets, 0, "Tcp.Resets")
					assertEqual(t, f.Tcp.Retrans, 0, "Tcp.Retrans")
					assertEqual(t, f.Tcp.SeqErrors, 0, "Tcp.SeqErrors")
				}
			},
		},
		{
			name:          "DPI stats flow",
			input:         DpiStatsFlowExample,
			wantType:      FlowTypeStats,
			wantInterface: "lan",
			wantInternal:  true,
			checkFlow: func(t *testing.T, flow any) {
				f := flow.(FlowStats)
				assertEqual(t, f.Digest, "e5f6a7b8c9d012345678901234567890cdef0123", "Digest")
				assertEqual(t, f.DetectionPackets, 32, "DetectionPackets")
				assertSliceEqual(
					t,
					f.DigestPrev,
					[]string{
						"3456789abcdef0123456789abcdef01234567890",
						"bcdef0123456789abcdef0123456789abcdef012",
					},
					"DigestPrev",
				)
				assertEqual(t, f.LastSeenAt, int64(1765893801667), "LastSeenAt")
				assertEqual(t, f.LocalBytes, int64(2028), "LocalBytes")
				assertEqual(t, f.LocalPackets, 34, "LocalPackets")
				assertEqual(t, f.LocalRate, 1510.6666259765625, "LocalRate")
				assertEqual(t, f.OtherBytes, int64(29478), "OtherBytes")
				assertEqual(t, f.OtherPackets, 37, "OtherPackets")
				assertEqual(t, f.OtherRate, 29478.0, "OtherRate")
				assertEqual(t, f.TotalBytes, int64(9404196), "TotalBytes")
				assertEqual(t, f.TotalPackets, 18450, "TotalPackets")
				if f.Tcp != nil {
					assertEqual(t, f.Tcp.Resets, 0, "Tcp.Resets")
					assertEqual(t, f.Tcp.Retrans, 0, "Tcp.Retrans")
					assertEqual(t, f.Tcp.SeqErrors, 0, "Tcp.SeqErrors")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event FlowEvent
			if err := json.Unmarshal([]byte(tt.input), &event); err != nil {
				t.Fatalf("Failed to unmarshal flow event: %v", err)
			}

			assertEqual(t, event.Type, tt.wantType, "Type")
			assertEqual(t, event.Interface, tt.wantInterface, "Interface")
			assertEqual(t, event.Internal, tt.wantInternal, "Internal")
			if tt.wantReason != "" {
				assertEqual(t, event.Reason, tt.wantReason, "Reason")
			}

			if tt.checkFlow != nil {
				tt.checkFlow(t, event.Flow)
			}
		})
	}
}

func assertEqual[T comparable](t *testing.T, got, want T, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", field, got, want)
	}
}

func assertSliceEqual[T comparable](t *testing.T, got, want []T, field string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got length %d, want %d", field, len(got), len(want))
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d]: got %v, want %v", field, i, got[i], want[i])
		}
	}
}

func TestParsingFlowErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		errorContains string
	}{
		{
			name:          "type missing",
			input:         `{"flow": {}}`,
			errorContains: `flow type "" not supported`,
		},
		{
			name:          "type not supported",
			input:         `{"type": "hello", "flow": {}}`,
			errorContains: `flow type "hello" not supported`,
		},
		{
			name:          "flow malformed",
			input:         `{"type": {}, "internal": 3, "flow": 123}`,
			errorContains: "malformed flow event:",
		},
		{
			name:          "malformed flow_dpi_complete",
			input:         `{ "type": "flow_dpi_complete", "flow": { "other_packets": "not-an-int" } }`,
			errorContains: `malformed "flow_dpi_complete" flow:`,
		},
		{
			name:          "malformed flow_purge",
			input:         `{ "type": "flow_purge", "flow": { "digest": 123 } }`,
			errorContains: `malformed "flow_purge" flow:`,
		},
		{
			name:          "malformed flow_stats",
			input:         `{ "type": "flow_stats", "flow": { "last_seen_at": "not-an-int" } }`,
			errorContains: `malformed "flow_stats" flow:`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event FlowEvent
			err := json.Unmarshal([]byte(tt.input), &event)
			if err == nil {
				t.Errorf("expected error but got none")
			} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q but got %q", tt.errorContains, err.Error())
			}
		})
	}
}
