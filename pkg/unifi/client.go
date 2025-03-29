package unifi

import (
	"fmt"
	"sort"
	"time"

	"github.com/dustin/go-humanize"
)

var (
	ClientBytesReceived = func(lhs, rhs *Client) bool { return lhs.BytesReceived < rhs.BytesReceived }
	ClientBytesSent     = func(lhs, rhs *Client) bool { return lhs.BytesSent < rhs.BytesSent }
	ClientConfidence    = func(lhs, rhs *Client) bool { return lhs.Confidence < rhs.Confidence }
	ClientFirstSeen     = func(lhs, rhs *Client) bool { return lhs.FirstSeen < rhs.FirstSeen }
	ClientIP            = func(lhs, rhs *Client) bool { return lhs.IP.Less(rhs.IP) }
	ClientIdle          = func(lhs, rhs *Client) bool { return lhs.IdleTime < rhs.IdleTime }
	ClientAuthorized    = func(lhs, rhs *Client) bool { return !lhs.IsAuthorized && rhs.IsAuthorized }
	ClientBlocked       = func(lhs, rhs *Client) bool { return !lhs.IsBlocked && rhs.IsBlocked }
	ClientGuest         = func(lhs, rhs *Client) bool { return !lhs.IsGuest && rhs.IsGuest }
	ClientWired         = func(lhs, rhs *Client) bool { return lhs.IsWired && !rhs.IsWired }
	ClientLastSeen      = func(lhs, rhs *Client) bool { return lhs.LastSeen < rhs.LastSeen }
	ClientName          = func(lhs, rhs *Client) bool { return lhs.DisplayName() < rhs.DisplayName() }
	ClientNetwork       = func(lhs, rhs *Client) bool { return lhs.Network < rhs.Network }
	ClientNoise         = func(lhs, rhs *Client) bool { return lhs.Noise < rhs.Noise }
	ClientSatisfaction  = func(lhs, rhs *Client) bool { return lhs.Satisfaction < rhs.Satisfaction }
	ClientScore         = func(lhs, rhs *Client) bool { return lhs.Score < rhs.Score }
	ClientSignal        = func(lhs, rhs *Client) bool { return lhs.Signal < rhs.Signal }
	ClientUptime        = func(lhs, rhs *Client) bool { return lhs.Uptime < rhs.Uptime }

	ClientDefault    = ClientOrderedBy(ClientWired, ClientIP)
	ClientHistorical = ClientOrderedBy(ClientLastSeen)

	ShowRate = false
)

// Client describes a UniFi network client.
type Client struct {
	ID string `json:"_id,omitempty"`

	AccessPointMAC           string  `json:"ap_mac,omitempty"`
	Anomalies                int64   `json:"anomalies,omitempty"`
	BSSID                    string  `json:"bssid,omitempty"`
	BytesError               float64 `json:"bytes-r,omitempty"`
	BytesReceived            int64   `json:"rx_bytes,omitempty"`
	BytesReceivedError       float64 `json:"rx_bytes-r,omitempty"`
	BytesSent                int64   `json:"tx_bytes,omitempty"`
	BytesSentError           float64 `json:"tx_bytes-r,omitempty"`
	CCQ                      int64   `json:"ccq,omitempty"`
	Channel                  int64   `json:"channel,omitempty"`
	Confidence               int64   `json:"confidence,omitempty"`
	DHCPEndTime              int64   `json:"dhcpend_time,omitempty"`
	DeviceCategory           int64   `json:"dev_cat,omitempty"`
	DeviceFamily             int64   `json:"dev_family,omitempty"`
	DeviceID                 int64   `json:"dev_id,omitempty"`
	DeviceIDOverride         int64   `json:"dev_id_override,omitempty"`
	DeviceName               string  `json:"device_name,omitempty"`
	DeviceVendor             int64   `json:"dev_vendor,omitempty"`
	ESSID                    string  `json:"essid,omitempty"`
	FingerprintEngineVersion string  `json:"fingerprint_engine_version,omitempty"`
	FingerprintSource        int64   `json:"fingerprint_source,omitempty"`
	FirmwareVersion          string  `json:"fw_version,omitempty"`
	FirstAssociatedAt        int64   `json:"assoc_time,omitempty"`
	FirstSeen                int64   `json:"first_seen,omitempty"`
	FixedIP                  IP      `json:"fixed_ip,omitempty"`
	GatewayMAC               string  `json:"gw_mac,omitempty"`
	HasFingerprintOverride   bool    `json:"fingerprint_override,omitempty"`
	HasQosApplied            bool    `json:"qos_policy_applied,omitempty"`
	Hostname                 string  `json:"hostname,omitempty"`
	IP                       IP      `json:"ip,omitempty"`
	IdleTime                 int64   `json:"idletime,omitempty"`
	Is11r                    bool    `json:"is_11r,omitempty"`
	IsAuthorized             bool    `json:"authorized,omitempty"`
	IsBlocked                bool    `json:"blocked,omitempty"`
	IsGuest                  bool    `json:"is_guest,omitempty"`
	IsNoted                  bool    `json:"noted,omitempty"`
	IsPowersaveEnabled       bool    `json:"powersave_enabled,omitempty"`
	IsUAPGuest               bool    `json:"_is_guest_by_uap,omitempty"`
	IsUGWGuest               bool    `json:"_is_guest_by_ugw,omitempty"`
	IsUSWGuest               bool    `json:"_is_guest_by_usw,omitempty"`
	IsWired                  bool    `json:"is_wired,omitempty"`
	LastAssociatedAt         int64   `json:"latest_assoc_time,omitempty"`
	LastSeen                 int64   `json:"last_seen,omitempty"`
	MAC                      MAC     `json:"mac,omitempty"`
	Name                     string  `json:"name,omitempty"`
	Network                  string  `json:"network,omitempty"`
	NetworkID                string  `json:"network_id,omitempty"`
	Noise                    int64   `json:"noise,omitempty"`
	Note                     string  `json:"note,omitempty"`
	OSName                   int64   `json:"os_name,omitempty"`
	OUI                      string  `json:"oui,omitempty"`
	PacketsReceived          int64   `json:"rx_packets,omitempty"`
	PacketsSent              int64   `json:"tx_packets,omitempty"`
	RSSI                     int64   `json:"rssi,omitempty"`
	Radio                    string  `json:"radio,omitempty"`
	RadioName                string  `json:"radio_name,omitempty"`
	RadioProto               string  `json:"radio_proto,omitempty"`
	ReceiveRate              int64   `json:"rx_rate,omitempty"`
	Retries                  int64   `json:"tx_retries,omitempty"`
	Satisfaction             int64   `json:"satisfaction,omitempty"`
	Score                    int64   `json:"score,omitempty"`
	Signal                   int64   `json:"signal,omitempty"`
	SiteID                   string  `json:"site_id,omitempty"`
	SwitchDepth              int64   `json:"sw_depth,omitempty"`
	SwitchMAC                string  `json:"sw_mac,omitempty"`
	SwitchPort               int64   `json:"sw_port,omitempty"`
	TransmitPower            int64   `json:"tx_power,omitempty"`
	TransmitRate             int64   `json:"tx_rate,omitempty"`
	UAPLastSeen              int64   `json:"_last_seen_by_uap,omitempty"`
	UAPUptime                int64   `json:"_uptime_by_uap,omitempty"`
	UGWLastSeen              int64   `json:"_last_seen_by_ugw,omitempty"`
	UGWUptime                int64   `json:"_uptime_by_ugw,omitempty"`
	USWLastSeen              int64   `json:"_last_seen_by_usw,omitempty"`
	USWUptime                int64   `json:"_uptime_by_usw,omitempty"`
	Uptime                   int64   `json:"uptime,omitempty"`
	UseFixedIP               bool    `json:"use_fixedip,omitempty"`
	UserGroupIDComputed      string  `json:"user_group_id_computed,omitempty"`
	UserID                   string  `json:"user_id,omitempty"`
	UsergroupID              string  `json:"usergroup_id,omitempty"`
	VLAN                     int64   `json:"vlan,omitempty"`
	WifiAttempts             int64   `json:"wifi_tx_attempts,omitempty"`
	WiredBytesReceived       int64   `json:"wired-rx_bytes,omitempty"`
	WiredBytesReceivedError  float64 `json:"wired-rx_bytes-r,omitempty"`
	WiredBytesSent           int64   `json:"wired-tx_bytes,omitempty"`
	WiredBytesSentError      float64 `json:"wired-tx_bytes-r,omitempty"`
	WiredPacketsReceived     int64   `json:"wired-rx_packets,omitempty"`
	WiredPacketsSent         int64   `json:"wired-tx_packets,omitempty"`
	WiredRateMBPS            int64   `json:"wired_rate_mbps,omitempty"`

	// Synthetic fields

	UpstreamName string `json:"upstream_name,omitempty"`
}

func (client *Client) IsBlockedGlyph() rune {
	if client.IsBlocked {
		return '✗'
	}

	return ' '
}

func (client *Client) IsGuestGlyph() rune {
	if client.IsGuest {
		return '✓'
	}

	return ' '
}

func (client *Client) IsWiredGlyph() rune {
	if client.IsWired {
		return '⌁'
	}

	return '⌔'
}

func (client *Client) DisplayName() string {
	return firstNonEmpty(client.Name, client.Hostname, client.DeviceName, client.OUI, string(client.MAC), "-")
}

func (client *Client) DisplayIP() string {
	return firstNonEmpty(string(client.FixedIP), string(client.IP), string(client.MAC))
}

func (client *Client) DisplayLastAssociated() string {
	return humanize.Time(time.Unix(client.LastAssociatedAt, 0))
}

func (client *Client) DisplayUptime() string {
	if client.Uptime == 0 {
		return humanize.Time(time.Unix(client.LastAssociatedAt, 0))
	}

	return humanize.Time(time.Now().Add(time.Duration(client.Uptime) * -time.Second))
}

func (client *Client) DisplayReceivedBytes() string {
	if client.IsWired {
		return formatBytesSize(client.WiredBytesReceived)
	}

	return formatBytesSize(client.BytesReceived)
}

func (client *Client) DisplaySentBytes() string {
	if client.IsWired {
		return formatBytesSize(client.WiredBytesSent)
	}

	return formatBytesSize(client.BytesSent)
}

func (client *Client) DisplayReceiveRate() string {
	rate := client.DisplayWiredRate()
	if len(rate) > 0 {
		return rate
	}

	return formatBytesSize(client.ReceiveRate)
}

func (client *Client) DisplaySendRate() string {
	rate := client.DisplayWiredRate()
	if len(rate) > 0 {
		return rate
	}

	return formatBytesSize(client.TransmitRate)
}

func (client *Client) DisplayWiredRate() string {
	if client.IsWired {
		switch client.WiredRateMBPS {
		case 0:
			return ""
		case 100:
			return "FE"
		case 1000:
			return "GbE"
		}

		return humanize.Bytes(uint64(client.WiredRateMBPS * 1000000))
	}

	return ""
}

func (client *Client) DisplayConnectionRate() string {
	if client.IsWired {
		if client.WiredRateMBPS == 0 {
			return ""
		}

		return humanize.Bytes(uint64(client.WiredRateMBPS * 1000000))
	}

	return fmt.Sprintf("%11s↓ %11s↑", client.DisplayReceiveRate(), client.DisplaySendRate())
}

func (client *Client) DisplaySwitchName() string {
	if len(client.UpstreamName) > 0 {
		return client.UpstreamName
	}

	return client.UpstreamMAC()
}

func (client *Client) String() string {
	rate := ""
	if ShowRate {
		rate = fmt.Sprintf("%25s", client.DisplayConnectionRate())
	}

	return fmt.Sprintf("%25s %-2s%-2s%-2s %-15s %-14s %-25s %s %s",
		client.DisplayName(),
		string(client.IsBlockedGlyph()),
		string(client.IsGuestGlyph()),
		string(client.IsWiredGlyph()),
		client.DisplayIP(),
		client.DisplayUptime(),
		fmt.Sprintf("%11s↓ %11s↑", client.DisplayReceivedBytes(), client.DisplaySentBytes()),
		rate,
		client.DisplaySwitchName(),
	)
}

func (client *Client) UpstreamMAC() string {
	return firstNonEmpty(client.AccessPointMAC, client.SwitchMAC, client.GatewayMAC)
}

// ToMACs converts a slice of Client to a slice of the corresponding MACs.
func ToMACs(clients []Client) []MAC {
	var macs []MAC

	for _, client := range clients {
		macs = append(macs, client.MAC)
	}

	return macs
}

// ClientOrderedBy returns a ClientSorter that sorts by the provided less functions.
func ClientOrderedBy(less ...ClientLessFn) *ClientSorter {
	return &ClientSorter{less: less}
}

// ClientLessFn describes a less function for a Client.
type ClientLessFn func(lhs, rhs *Client) bool

// ClientSorter is a multisorter for sorting slices of Client.
type ClientSorter struct {
	clients []Client
	less    []ClientLessFn
}

// Sort applies the configured less functions in order.
func (s *ClientSorter) Sort(clients []Client) {
	s.clients = clients
	sort.Sort(s)
}

func (s *ClientSorter) Len() int      { return len(s.clients) }
func (s *ClientSorter) Swap(i, j int) { s.clients[i], s.clients[j] = s.clients[j], s.clients[i] }
func (s *ClientSorter) Less(i, j int) bool {
	lhs, rhs := &s.clients[i], &s.clients[j]
	var k int
	for k = 0; k < len(s.less)-1; k++ {
		less := s.less[k]
		switch {
		case less(lhs, rhs):
			return true
		case less(rhs, lhs):
			return false
		}
	}
	return s.less[k](lhs, rhs)
}
