package flows

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	FlowTypeBegin       = "flow"
	FlowTypeDpiComplete = "flow_dpi_complete"
	FlowTypeDpiUpdate   = "flow_dpi_update"
	FlowTypePurge       = "flow_purge"
	FlowTypeStats       = "flow_stats"
)

type FlowEvent struct {
	Type      string `json:"type"`
	Interface string `json:"interface,omitempty"`
	Internal  bool   `json:"internal,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Flow      any    `json:"flow"`
}

type Conntrack struct {
	Id int `json:"id"`
}

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

type FlowBase struct {
	Digest string `json:"digest"`
}

type FlowStart struct {
	FlowBase
	Conntrack           Conntrack `json:"conntrack"`
	DetectedApplication string    `json:"detected_application_name"`
	DetectedProtocol    string    `json:"detected_protocol_name"`
	FirstSeenAt         int64     `json:"first_seen_at"`
	LastSeenAt          int64     `json:"last_seen_at"`
	LocalIp             string    `json:"local_ip"`
	LocalMac            string    `json:"local_mac"`
	LocalOrigin         bool      `json:"local_origin"`
	LocalPort           int       `json:"local_port"`
	OtherIp             string    `json:"other_ip"`
	OtherMac            string    `json:"other_mac"`
	OtherPort           int       `json:"other_port"`
	OtherType           string    `json:"other_type"`
	HostServerName      string    `json:"host_server_name,omitempty"`
	DnsHostName         string    `json:"dns_host_name,omitempty"`
	Risks               struct {
		NdpiRiskScore       int   `json:"ndpi_risk_score"`
		NdpiRiskScoreClient int   `json:"ndpi_risk_score_client"`
		NdpiRiskScoreServer int   `json:"ndpi_risk_score_server"`
		Risks               []int `json:"risks,omitempty"`
	} `json:"risks"`
}

type FlowComplete struct {
	FlowStart
	Stats
	DetectionGuessed bool `json:"detection_guessed,omitempty"`
	DetectionPackets int  `json:"detection_packets,omitempty"`
	Ssl              struct {
		ClientSni string `json:"client_sni,omitempty"`
	} `json:"ssl,omitempty"`
}

type FlowPurge struct {
	FlowBase
}

type FlowStats struct {
	FlowBase
	Stats
	LastSeenAt int64 `json:"last_seen_at"`
}

func (f *FlowEvent) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Type      string          `json:"type"`
		Interface string          `json:"interface,omitempty"`
		Internal  bool            `json:"internal,omitempty"`
		Reason    string          `json:"reason,omitempty"`
		Flow      json.RawMessage `json:"flow"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return errors.New("malformed flow event: " + err.Error())
	}

	f.Type = tmp.Type
	f.Interface = tmp.Interface
	f.Internal = tmp.Internal
	f.Reason = tmp.Reason

	switch tmp.Type {
	case FlowTypeBegin:
		var flow FlowStart
		if err := json.Unmarshal(tmp.Flow, &flow); err != nil {
			return fmt.Errorf("malformed %q flow: %w", tmp.Type, err)
		}
		f.Flow = flow
	case FlowTypeDpiUpdate:
		fallthrough
	case FlowTypeDpiComplete:
		var flow FlowComplete
		if err := json.Unmarshal(tmp.Flow, &flow); err != nil {
			return fmt.Errorf("malformed %q flow: %w", tmp.Type, err)
		}
		f.Flow = flow
	case FlowTypePurge:
		var flow FlowPurge
		if err := json.Unmarshal(tmp.Flow, &flow); err != nil {
			return fmt.Errorf("malformed %q flow: %w", tmp.Type, err)
		}
		f.Flow = flow
	case FlowTypeStats:
		var flow FlowStats
		if err := json.Unmarshal(tmp.Flow, &flow); err != nil {
			return fmt.Errorf("malformed %q flow: %w", tmp.Type, err)
		}
		f.Flow = flow
	default:
		return fmt.Errorf("flow type %q not supported", tmp.Type)
	}
	return nil
}
