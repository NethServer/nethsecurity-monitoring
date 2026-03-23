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
			name:          "DPI complete flow",
			input:         DpiCompleteFlowExample,
			wantType:      FlowTypeDpiComplete,
			wantInterface: "lan",
			wantInternal:  true,
			checkFlow: func(t *testing.T, flow any) {
				f := flow.(FlowComplete)
				assertEqual(t, f.Conntrack.Id, 163461572, "Conntrack.Id")
				assertEqual(t, f.Digest, "c3d4e5f6a7b890123456789012345678abcdef01", "Digest")
				assertEqual(
					t,
					f.DetectedApplicationName,
					"netify.cloud-service",
					"DetectedApplicationName",
				)
				assertEqual(t, f.DetectedApplication, 10733, "DetectedApplication")
				assertEqual(t, f.DetectedProtocol, 188, "DetectedProtocol")
				assertEqual(t, f.DetectedProtocolName, "QUIC", "DetectedProtocolName")
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
