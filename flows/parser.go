package flows

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	FlowTypeDpiComplete = "flow_dpi_complete"
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
	Id           int    `json:"id"`
	Mark         int    `json:"mark"`
	ReplyDstIp   string `json:"reply_dst_ip"`
	ReplyDstPort int    `json:"reply_dst_port"`
	ReplySrcIp   string `json:"reply_src_ip"`
	ReplySrcPort int    `json:"reply_src_port"`
}

type Category struct {
	Application  int `json:"application"`
	Domain       int `json:"domain"`
	LocalNetwork int `json:"local_network"`
	OtherNetwork int `json:"other_network"`
	Overlay      int `json:"overlay"`
	Protocol     int `json:"protocol"`
}

type Tcp struct {
	Resets    int `json:"resets"`
	Retrans   int `json:"retrans"`
	SeqErrors int `json:"seq_errors"`
}

type Bt struct {
	InfoHash string `json:"info_hash"`
}

type Dhcp struct {
	ClassIdent  string `json:"class_ident"`
	Fingerprint string `json:"fingerprint"`
}

type GtpFlow struct {
	IpDscp    int    `json:"ip_dscp"`
	IpVersion int    `json:"ip_version"`
	LocalIp   string `json:"local_ip"`
	LocalPort int    `json:"local_port"`
	LocalTeid int    `json:"local_teid"`
	OtherIp   string `json:"other_ip"`
	OtherPort int    `json:"other_port"`
	OtherTeid int    `json:"other_teid"`
	OtherType string `json:"other_type"`
	Version   int    `json:"version"`
}

type Http struct {
	Url       string `json:"url"`
	UserAgent string `json:"user_agent"`
}

type Mdns struct {
	Answer string `json:"answer"`
}

type Nfq struct {
	DstIface string `json:"dst_iface"`
	SrcIface string `json:"src_iface"`
}

type Ssh struct {
	Client string `json:"client"`
	Server string `json:"server"`
}

type Ssdp struct {
	UserAgent string `json:"user_agent"`
}

type Stun struct {
	Mapped   string `json:"mapped"`
	Other    string `json:"other"`
	Peer     string `json:"peer"`
	Relayed  string `json:"relayed"`
	Response string `json:"response"`
}

type Ssl struct {
	Alpn               []string `json:"alpn,omitempty"`
	AlpnServer         []string `json:"alpn_server,omitempty"`
	CipherSuite        string   `json:"cipher_suite"`
	ClientJa4          string   `json:"client_ja4"`
	ClientSni          string   `json:"client_sni,omitempty"`
	EncryptedChVersion string   `json:"encrypted_ch_version"`
	Fingerprint        string   `json:"fingerprint"`
	IssuerDn           string   `json:"issuer_dn"`
	ServerCn           string   `json:"server_cn"`
	SubjectDn          string   `json:"subject_dn"`
	Version            string   `json:"version"`
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

type FlowComplete struct {
	FlowBase
	AppIpOverride           bool      `json:"app_ip_override"`
	AppProtoTwins           bool      `json:"app_proto_twins"`
	Bt                      *Bt       `json:"bt,omitempty"`
	Category                *Category `json:"category,omitempty"`
	Conntrack               Conntrack `json:"conntrack"`
	DetectedApplication     int       `json:"detected_application"`
	DetectedApplicationName string    `json:"detected_application_name"`
	DetectedProtocol        int       `json:"detected_protocol"`
	DetectedProtocolName    string    `json:"detected_protocol_name"`
	DetectionGuessed        bool      `json:"detection_guessed"`
	DetectionPackets        int       `json:"detection_packets"`
	DetectionUpdated        bool      `json:"detection_updated"`
	DhcHit                  bool      `json:"dhc_hit"`
	Dhcp                    *Dhcp     `json:"dhcp,omitempty"`
	DigestPrev              []string  `json:"digest_prev,omitempty"`
	DnsHostName             string    `json:"dns_host_name,omitempty"`
	FhcHit                  bool      `json:"fhc_hit"`
	FirstSeenAt             int64     `json:"first_seen_at"`
	Gtp                     *GtpFlow  `json:"gtp,omitempty"`
	HostServerName          string    `json:"host_server_name,omitempty"`
	Http                    *Http     `json:"http,omitempty"`
	IpDscp                  int       `json:"ip_dscp"`
	IpNat                   bool      `json:"ip_nat"`
	IpProtocol              int       `json:"ip_protocol"`
	IpVersion               int       `json:"ip_version"`
	LastSeenAt              int64     `json:"last_seen_at"`
	LocalIp                 string    `json:"local_ip"`
	LocalMac                string    `json:"local_mac"`
	LocalOrigin             bool      `json:"local_origin"`
	LocalPort               int       `json:"local_port"`
	Mdns                    *Mdns     `json:"mdns,omitempty"`
	Nfq                     *Nfq      `json:"nfq,omitempty"`
	OtherIp                 string    `json:"other_ip"`
	OtherMac                string    `json:"other_mac"`
	OtherPort               int       `json:"other_port"`
	OtherType               string    `json:"other_type"`
	Risks                   struct {
		NdpiRiskScore       int   `json:"ndpi_risk_score"`
		NdpiRiskScoreClient int   `json:"ndpi_risk_score_client"`
		NdpiRiskScoreServer int   `json:"ndpi_risk_score_server"`
		Risks               []int `json:"risks,omitempty"`
	} `json:"risks"`
	SoftDissector bool  `json:"soft_dissector"`
	Ssh           *Ssh  `json:"ssh,omitempty"`
	Ssdp          *Ssdp `json:"ssdp,omitempty"`
	Ssl           *Ssl  `json:"ssl,omitempty"`
	Stats
	Stun   *Stun    `json:"stun,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Tcp    *Tcp     `json:"tcp,omitempty"`
	VlanId int      `json:"vlan_id"`
}

type FlowPurge struct {
	FlowBase
	DetectionPackets int      `json:"detection_packets"`
	DigestPrev       []string `json:"digest_prev,omitempty"`
	LastSeenAt       int64    `json:"last_seen_at"`
	Stats
	Tcp *Tcp `json:"tcp,omitempty"`
}

type FlowStats struct {
	FlowBase
	DetectionPackets int      `json:"detection_packets"`
	DigestPrev       []string `json:"digest_prev,omitempty"`
	LastSeenAt       int64    `json:"last_seen_at"`
	Stats
	Tcp *Tcp `json:"tcp,omitempty"`
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
