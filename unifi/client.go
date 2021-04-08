package unifi

import (
	"fmt"
	"sort"
	"time"
)

var (
	BytesReceived = func(lhs, rhs *Client) bool { return lhs.BytesReceived < rhs.BytesReceived }
	BytesSent     = func(lhs, rhs *Client) bool { return lhs.BytesSent < rhs.BytesSent }
	Confidence    = func(lhs, rhs *Client) bool { return lhs.Confidence < rhs.Confidence }
	FirstSeen     = func(lhs, rhs *Client) bool { return lhs.FirstSeen < rhs.FirstSeen }
	IPAddress     = func(lhs, rhs *Client) bool { return lhs.IP < rhs.IP }
	IdleTime      = func(lhs, rhs *Client) bool { return lhs.IdleTime < rhs.IdleTime }
	IsAuthorized  = func(lhs, rhs *Client) bool { return !lhs.IsAuthorized && rhs.IsAuthorized }
	IsBlocked     = func(lhs, rhs *Client) bool { return !lhs.IsBlocked && rhs.IsBlocked }
	IsGuest       = func(lhs, rhs *Client) bool { return !lhs.IsGuest && rhs.IsGuest }
	IsWired       = func(lhs, rhs *Client) bool { return !lhs.IsWired && rhs.IsWired }
	LastSeen      = func(lhs, rhs *Client) bool { return lhs.LastSeen < rhs.LastSeen }
	Name          = func(lhs, rhs *Client) bool { return lhs.Name < rhs.Name }
	Network       = func(lhs, rhs *Client) bool { return lhs.Network < rhs.Network }
	Noise         = func(lhs, rhs *Client) bool { return lhs.Noise < rhs.Noise }
	Satisfaction  = func(lhs, rhs *Client) bool { return lhs.Satisfaction < rhs.Satisfaction }
	Score         = func(lhs, rhs *Client) bool { return lhs.Score < rhs.Score }
	Signal        = func(lhs, rhs *Client) bool { return lhs.Signal < rhs.Signal }
	Uptime        = func(lhs, rhs *Client) bool { return lhs.Uptime < rhs.Uptime }

	ClientDefault    = OrderedBy(IsAuthorized, IsGuest, IsWired, IPAddress)
	ClientHistorical = OrderedBy(Name, LastSeen)
)

// Client describes a UniFi network client.
type Client struct {
	ID string `json:"_id,omitempty"`

	AccessPointMAC  string `json:"ap_mac,omitempty"`
	BSSID           string `json:"bssid,omitempty"`
	DeviceName      string `json:"device_name,omitempty"`
	ESSID           string `json:"essid,omitempty"`
	FirmwareVersion string `json:"fw_version,omitempty"`
	FixedIP         string `json:"fixed_ip,omitempty"`
	GatewayMAC      string `json:"gw_mac,omitempty"`
	Hostname        string `json:"hostname,omitempty"`
	IP              string `json:"ip,omitempty"`
	MAC             string `json:"mac,omitempty"`
	Name            string `json:"name,omitempty"`
	Network         string `json:"network,omitempty"`
	NetworkID       string `json:"network_id,omitempty"`
	Note            string `json:"note,omitempty"`
	OUI             string `json:"oui,omitempty"`
	Radio           string `json:"radio,omitempty"`
	RadioName       string `json:"radio_name,omitempty"`
	RadioProto      string `json:"radio_proto,omitempty"`
	SiteID          string `json:"site_id,omitempty"`
	SwitchMAC       string `json:"sw_mac,omitempty"`
	UserID          string `json:"user_id,omitempty"`
	UsergroupID     string `json:"usergroup_id,omitempty"`

	FingerprintEngineVersion string `json:"fingerprint_engine_version,omitempty"`
	UserGroupIDComputed      string `json:"user_group_id_computed,omitempty"`

	FirstAssociatedAt int64 `json:"assoc_time,omitempty"`
	FirstSeen         int64 `json:"first_seen,omitempty"`
	IdleTime          int64 `json:"idletime,omitempty"`
	LastAssociatedAt  int64 `json:"latest_assoc_time,omitempty"`
	LastSeen          int64 `json:"last_seen,omitempty"`
	UAPLastSeen       int64 `json:"_last_seen_by_uap,omitempty"`
	UAPUptime         int64 `json:"_uptime_by_uap,omitempty"`
	UGWLastSeen       int64 `json:"_last_seen_by_ugw,omitempty"`
	UGWUptime         int64 `json:"_uptime_by_ugw,omitempty"`
	USWLastSeen       int64 `json:"_last_seen_by_usw,omitempty"`
	USWUptime         int64 `json:"_uptime_by_usw,omitempty"`
	Uptime            int64 `json:"uptime,omitempty"`

	ReceiveRate   int64 `json:"rx_rate,omitempty"`
	TransmitRate  int64 `json:"tx_rate,omitempty"`
	WiredRateMBPS int64 `json:"wired_rate_mbps,omitempty"`

	BytesError         int64 `json:"bytes-r,omitempty"`
	BytesReceived      int64 `json:"rx_bytes,omitempty"`
	BytesReceivedError int64 `json:"rx_bytes-r,omitempty"`
	BytesSent          int64 `json:"tx_bytes,omitempty"`
	BytesSentError     int64 `json:"tx_bytes-r,omitempty"`
	PacketsReceived    int64 `json:"rx_packets,omitempty"`
	PacketsSent        int64 `json:"tx_packets,omitempty"`

	WiredBytesReceived      int64 `json:"wired-rx_bytes,omitempty"`
	WiredBytesReceivedError int64 `json:"wired-rx_bytes-r,omitempty"`
	WiredBytesSent          int64 `json:"wired-tx_bytes,omitempty"`
	WiredBytesSentError     int64 `json:"wired-tx_bytes-r,omitempty"`
	WiredPacketsReceived    int64 `json:"wired-rx_packets,omitempty"`
	WiredPacketsSent        int64 `json:"wired-tx_packets,omitempty"`

	Anomalies        int `json:"anomalies,omitempty"`
	Confidence       int `json:"confidence,omitempty"`
	DeviceCategory   int `json:"dev_cat,omitempty"`
	DeviceFamily     int `json:"dev_family,omitempty"`
	DeviceID         int `json:"dev_id,omitempty"`
	DeviceIDOverride int `json:"dev_id_override,omitempty"`
	DeviceVendor     int `json:"dev_vendor,omitempty"`
	OSName           int `json:"os_name,omitempty"`
	Retries          int `json:"tx_retries,omitempty"`
	Satisfaction     int `json:"satisfaction,omitempty"`
	Score            int `json:"score,omitempty"`
	SwitchDepth      int `json:"sw_depth,omitempty"`
	SwitchPort       int `json:"sw_port,omitempty"`
	WifiAttempts     int `json:"wifi_tx_attempts,omitempty"`

	CCQ           int `json:"ccq,omitempty"`
	Channel       int `json:"channel,omitempty"`
	DHCPEndTime   int `json:"dhcpend_time,omitempty"`
	Noise         int `json:"noise,omitempty"`
	RSSI          int `json:"rssi,omitempty"`
	Signal        int `json:"signal,omitempty"`
	TransmitPower int `json:"tx_power,omitempty"`
	VLAN          int `json:"vlan,omitempty"`

	FingerprintSource      int  `json:"fingerprint_source,omitempty"`
	HasFingerprintOverride bool `json:"fingerprint_override,omitempty"`

	HasQosApplied      bool `json:"qos_policy_applied,omitempty"`
	Is11r              bool `json:"is_11r,omitempty"`
	IsAuthorized       bool `json:"authorized,omitempty"`
	IsBlocked          bool `json:"blocked,omitempty"`
	IsGuest            bool `json:"is_guest,omitempty"`
	IsNoted            bool `json:"noted,omitempty"`
	IsPowersaveEnabled bool `json:"powersave_enabled,omitempty"`
	IsUAPGuest         bool `json:"_is_guest_by_uap,omitempty"`
	IsUGWGuest         bool `json:"_is_guest_by_ugw,omitempty"`
	IsUSWGuest         bool `json:"_is_guest_by_usw,omitempty"`
	IsWired            bool `json:"is_wired,omitempty"`
	UseFixedIP         bool `json:"use_fixedip,omitempty"`
}

func (client *Client) String() string {
	display := firstNonEmpty(client.Name, client.Hostname, client.DeviceName, client.OUI, client.MAC, "-")
	ip := firstNonEmpty(client.IP, client.FixedIP)

	blocked := ""
	if client.IsBlocked {
		blocked = "✗"
	}

	guest := ""
	if client.IsGuest {
		guest = "✓"
	}

	wired := "⌔"
	if client.IsWired {
		wired = "⌁"
	}

	uptime := (time.Duration(client.Uptime) * time.Second).String()
	if client.Uptime == 0 {
		uptime = time.Unix(client.LastSeen, 0).Format(time.RFC3339)
	}

	traffic := ""
	if client.BytesReceived+client.BytesSent > 0 {
		recvd := formatBytesSize(client.BytesReceived)
		sent := formatBytesSize(client.BytesSent)
		traffic = fmt.Sprintf("%10s ↓ / %10s ↑", recvd, sent)
	}

	return fmt.Sprintf("%25s %-2s %-2s %-2s %-15s %-10s %s",
		display,
		blocked,
		guest,
		wired,
		ip,
		uptime,
		traffic,
	)
}

// OrderedBy returns a ClientSorter that sorts by the provided less functions.
func OrderedBy(less ...ClientLessFn) *ClientSorter {
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
