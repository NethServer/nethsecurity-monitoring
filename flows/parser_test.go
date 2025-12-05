package flows

import (
	"encoding/json"
	"strings"
	"testing"
)

const StartFlowExample = `
{
  "flow": {
    "app_ip_override": false,
    "category": {
      "application": 28,
      "domain": 0,
      "network": 0,
      "protocol": 18,
      "tag": 0
    },
    "conntrack": {
      "id": 2246304405,
      "mark": 0,
      "reply_dst_ip": "192.168.1.100",
      "reply_dst_port": 41014,
      "reply_src_ip": "203.0.113.42",
      "reply_src_port": 443
    },
    "detected_application": 11153,
    "detected_application_name": "netify.example-app",
    "detected_protocol": 196,
    "detected_protocol_name": "HTTP/S",
    "detection_guessed": false,
    "detection_updated": false,
    "dhc_hit": false,
    "digest": "ff77f56330b3037b0d4925047789bb77cadbd4bc",
    "digest_prev": [
      "87246f24a768e89d3e50fb6113dec51833e500ff"
    ],
    "dns_host_name": "cdn-example.test-domain.com",
    "fhc_hit": false,
    "first_seen_at": 1765892953385,
    "host_server_name": "api.example-service.com",
    "ip_dscp": 0,
    "ip_nat": false,
    "ip_protocol": 6,
    "ip_version": 4,
    "last_seen_at": 1765892953405,
    "local_ip": "192.168.1.100",
    "local_mac": "00:00:00:00:00:00",
    "local_origin": true,
    "local_port": 41014,
    "nfq": {
      "dst_iface": "eth1",
      "src_iface": "wg1"
    },
    "other_ip": "203.0.113.42",
    "other_mac": "aa:bb:cc:dd:ee:01",
    "other_port": 443,
    "other_type": "remote",
    "risks": {
      "ndpi_risk_score": 0,
      "ndpi_risk_score_client": 0,
      "ndpi_risk_score_server": 0
    },
    "soft_dissector": false,
    "ssl": {
      "cipher_suite": "0x0000",
      "client_ja4": "t13d371300_db35923f8641_867a32efce91",
      "client_sni": "api.example-service.com",
      "version": "0x0303"
    },
    "vlan_id": 0
  },
  "interface": "lan",
  "internal": true,
  "type": "flow"
}`

const DpiUpdateFlowExample = `
{
  "flow": {
    "app_ip_override": false,
    "category": {
      "application": 27,
      "domain": 0,
      "network": 0,
      "protocol": 18,
      "tag": 0
    },
    "conntrack": {
      "id": 2200126535,
      "mark": 0,
      "reply_dst_ip": "192.168.1.100",
      "reply_dst_port": 56953,
      "reply_src_ip": "198.51.100.25",
      "reply_src_port": 443
    },
    "detected_application": 124,
    "detected_application_name": "netify.video-streaming",
    "detected_protocol": 188,
    "detected_protocol_name": "QUIC",
    "detection_guessed": false,
    "detection_packets": 2,
    "detection_updated": true,
    "dhc_hit": false,
    "digest": "6036f2d3ecb8217f4f3ed8eb23ed295c54c628d1",
    "digest_prev": [
      "d052bdc731dfb63e8c82c37a807b45c8bbae1636",
      "0d50d9ae3e9e9a25f9d8cdb12288070358f9c7d3"
    ],
    "dns_host_name": "video-cdn.example-media.net",
    "fhc_hit": false,
    "first_seen_at": 1765893713856,
    "host_server_name": "www.example-media.net",
    "ip_dscp": 0,
    "ip_nat": false,
    "ip_protocol": 17,
    "ip_version": 4,
    "last_seen_at": 1765893713860,
    "local_bytes": 5246,
    "local_ip": "192.168.1.100",
    "local_mac": "00:00:00:00:00:00",
    "local_origin": true,
    "local_packets": 4,
    "local_port": 56953,
    "local_rate": 5246.0,
    "nfq": {
      "dst_iface": "eth1",
      "src_iface": "wg1"
    },
    "other_bytes": 0,
    "other_ip": "198.51.100.25",
    "other_mac": "aa:bb:cc:dd:ee:02",
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
      "cipher_suite": "0x0000",
      "client_sni": "www.example-media.net",
      "encrypted_ch_version": "0xfe0d",
      "version": "0x0304"
    },
    "total_bytes": 0,
    "total_packets": 0,
    "vlan_id": 0
  },
  "interface": "lan",
  "internal": true,
  "type": "flow_dpi_update"
}`

const DpiCompleteFlowExample = `
{
  "flow": {
    "app_ip_override": false,
    "category": {
      "application": 14,
      "domain": 0,
      "network": 0,
      "protocol": 18,
      "tag": 0
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
      "cipher_suite": "0x0000",
      "client_sni": "api.example-cloud.io",
      "encrypted_ch_version": "0xfe0d",
      "version": "0x0304"
    },
    "total_bytes": 0,
    "total_packets": 0,
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
    "local_bytes": 0,
    "local_packets": 0,
    "local_rate": 128.0,
    "other_bytes": 0,
    "other_packets": 0,
    "other_rate": 79.0,
    "total_bytes": 193,
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
			name:          "initial flow",
			input:         StartFlowExample,
			wantType:      FlowTypeBegin,
			wantInterface: "lan",
			wantInternal:  true,
			checkFlow: func(t *testing.T, flow any) {
				f := flow.(FlowStart)
				assertEqual(t, f.Conntrack.Id, 2246304405, "Conntrack.Id")
				assertEqual(t, f.Digest, "ff77f56330b3037b0d4925047789bb77cadbd4bc", "Digest")
				assertEqual(t, f.DetectedApplication, "netify.example-app", "DetectedApplication")
				assertEqual(t, f.DetectedProtocol, "HTTP/S", "DetectedProtocol")
				assertEqual(t, f.FirstSeenAt, int64(1765892953385), "FirstSeenAt")
				assertEqual(t, f.LastSeenAt, int64(1765892953405), "LastSeenAt")
				assertEqual(t, f.LocalIp, "192.168.1.100", "LocalIp")
				assertEqual(t, f.LocalMac, "00:00:00:00:00:00", "LocalMac")
				assertEqual(t, f.LocalOrigin, true, "LocalOrigin")
				assertEqual(t, f.LocalPort, 41014, "LocalPort")
				assertEqual(t, f.OtherIp, "203.0.113.42", "OtherIp")
				assertEqual(t, f.OtherMac, "aa:bb:cc:dd:ee:01", "OtherMac")
				assertEqual(t, f.OtherPort, 443, "OtherPort")
				assertEqual(t, f.OtherType, "remote", "OtherType")
				assertEqual(t, f.HostServerName, "api.example-service.com", "HostServerName")
				assertEqual(t, f.DnsHostName, "cdn-example.test-domain.com", "DnsHostName")
				assertEqual(t, f.Risks.NdpiRiskScore, 0, "Risks.NdpiRiskScore")
				assertEqual(t, f.Risks.NdpiRiskScoreClient, 0, "Risks.NdpiRiskScoreClient")
				assertEqual(t, f.Risks.NdpiRiskScoreServer, 0, "Risks.NdpiRiskScoreServer")
			},
		},
		{
			name:          "DPI update flow",
			input:         DpiUpdateFlowExample,
			wantType:      FlowTypeDpiUpdate,
			wantInterface: "lan",
			wantInternal:  true,
			checkFlow: func(t *testing.T, flow any) {
				f := flow.(FlowComplete)
				assertEqual(t, f.Conntrack.Id, 2200126535, "Conntrack.Id")
				assertEqual(t, f.Digest, "6036f2d3ecb8217f4f3ed8eb23ed295c54c628d1", "Digest")
				assertEqual(
					t,
					f.DetectedApplication,
					"netify.video-streaming",
					"DetectedApplication",
				)
				assertEqual(t, f.DetectedProtocol, "QUIC", "DetectedProtocol")
				assertEqual(t, f.FirstSeenAt, int64(1765893713856), "FirstSeenAt")
				assertEqual(t, f.LastSeenAt, int64(1765893713860), "LastSeenAt")
				assertEqual(t, f.LocalIp, "192.168.1.100", "LocalIp")
				assertEqual(t, f.LocalMac, "00:00:00:00:00:00", "LocalMac")
				assertEqual(t, f.LocalOrigin, true, "LocalOrigin")
				assertEqual(t, f.LocalPort, 56953, "LocalPort")
				assertEqual(t, f.OtherIp, "198.51.100.25", "OtherIp")
				assertEqual(t, f.OtherMac, "aa:bb:cc:dd:ee:02", "OtherMac")
				assertEqual(t, f.OtherPort, 443, "OtherPort")
				assertEqual(t, f.OtherType, "remote", "OtherType")
				assertEqual(t, f.LocalBytes, int64(5246), "LocalBytes")
				assertEqual(t, f.LocalPackets, 4, "LocalPackets")
				assertEqual(t, f.LocalRate, 5246.0, "LocalRate")
				assertEqual(t, f.OtherBytes, int64(0), "OtherBytes")
				assertEqual(t, f.OtherPackets, 0, "OtherPackets")
				assertEqual(t, f.OtherRate, 0.0, "OtherRate")
				assertEqual(t, f.TotalBytes, int64(0), "TotalBytes")
				assertEqual(t, f.TotalPackets, 0, "TotalPackets")
				assertEqual(t, f.HostServerName, "www.example-media.net", "HostServerName")
				assertEqual(t, f.DnsHostName, "video-cdn.example-media.net", "DnsHostName")
				assertEqual(t, f.Risks.NdpiRiskScore, 110, "Risks.NdpiRiskScore")
				assertEqual(t, f.Risks.NdpiRiskScoreClient, 95, "Risks.NdpiRiskScoreClient")
				assertEqual(t, f.Risks.NdpiRiskScoreServer, 15, "Risks.NdpiRiskScoreServer")
				assertSliceEqual(t, f.Risks.Risks, []int{39, 46}, "Risks.Risks")
			},
		},
		{
			name:          "DPI complete flow",
			input:         DpiCompleteFlowExample,
			wantType:      FlowTypeDpiComplete,
			wantInterface: "lan",
			wantInternal:  true,
			checkFlow: func(t *testing.T, flow any) {
				f := flow.(FlowComplete)
				assertEqual(t, f.Conntrack.Id, 163461572, "Conntrack.Id")
				assertEqual(t, f.Digest, "c3d4e5f6a7b890123456789012345678abcdef01", "Digest")
				assertEqual(t, f.DetectedApplication, "netify.cloud-service", "DetectedApplication")
				assertEqual(t, f.DetectedProtocol, "QUIC", "DetectedProtocol")
				assertEqual(t, f.FirstSeenAt, int64(1765893622750), "FirstSeenAt")
				assertEqual(t, f.LastSeenAt, int64(1765893622753), "LastSeenAt")
				assertEqual(t, f.LocalIp, "192.168.1.100", "LocalIp")
				assertEqual(t, f.LocalMac, "00:00:00:00:00:00", "LocalMac")
				assertEqual(t, f.LocalOrigin, true, "LocalOrigin")
				assertEqual(t, f.LocalPort, 44743, "LocalPort")
				assertEqual(t, f.OtherIp, "203.0.113.88", "OtherIp")
				assertEqual(t, f.OtherMac, "aa:bb:cc:dd:ee:03", "OtherMac")
				assertEqual(t, f.OtherPort, 443, "OtherPort")
				assertEqual(t, f.OtherType, "remote", "OtherType")
				assertEqual(t, f.LocalBytes, int64(6583), "LocalBytes")
				assertEqual(t, f.LocalPackets, 6, "LocalPackets")
				assertEqual(t, f.LocalRate, 6583.0, "LocalRate")
				assertEqual(t, f.OtherBytes, int64(0), "OtherBytes")
				assertEqual(t, f.OtherPackets, 0, "OtherPackets")
				assertEqual(t, f.OtherRate, 0.0, "OtherRate")
				assertEqual(t, f.TotalBytes, int64(0), "TotalBytes")
				assertEqual(t, f.TotalPackets, 0, "TotalPackets")
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
				assertEqual(t, f.LastSeenAt, int64(1765893801667), "LastSeenAt")
				assertEqual(t, f.LocalBytes, int64(2028), "LocalBytes")
				assertEqual(t, f.LocalPackets, 34, "LocalPackets")
				assertEqual(t, f.LocalRate, 1510.6666259765625, "LocalRate")
				assertEqual(t, f.OtherBytes, int64(29478), "OtherBytes")
				assertEqual(t, f.OtherPackets, 37, "OtherPackets")
				assertEqual(t, f.OtherRate, 29478.0, "OtherRate")
				assertEqual(t, f.TotalBytes, int64(9404196), "TotalBytes")
				assertEqual(t, f.TotalPackets, 18450, "TotalPackets")
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
			name:          "malformed flow start",
			input:         `{ "type": "flow", "flow": { "first_seen_at": "not-an-int" } }`,
			errorContains: `malformed "flow" flow:`,
		},
		{
			name:          "malformed flow_dpi_update",
			input:         `{ "type": "flow_dpi_update", "flow": { "local_bytes": "not-an-int" } }`,
			errorContains: `malformed "flow_dpi_update" flow:`,
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
