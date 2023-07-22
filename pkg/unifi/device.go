package unifi

import (
	"fmt"
	"sort"
	"strconv"
)

var (
	DeviceBytesReceived = func(lhs, rhs *Device) bool { return lhs.BytesReceived < rhs.BytesReceived }
	DeviceBytesSent     = func(lhs, rhs *Device) bool { return lhs.BytesSent < rhs.BytesSent }
	DeviceIP            = func(lhs, rhs *Device) bool { return lhs.IP.Less(rhs.IP) }

	DeviceDefault = DeviceOrderedBy(DeviceIP)
)

// Device describes unifi managed device.
type Device struct {
	ID string `json:"_id,omitempty"`

	AdoptIP                    IP                    `json:"adopt_ip,omitempty"`
	AdoptURL                   string                `json:"adopt_url,omitempty"`
	Anomalies                  int64                 `json:"anomalies,omitempty"`
	AnonID                     string                `json:"anon_id,omitempty"`
	Architecture               string                `json:"architecture,omitempty"`
	BoardRevision              int64                 `json:"board_rev,omitempty"`
	Bytes                      int64                 `json:"bytes,omitempty"`
	BytesReceived              int64                 `json:"rx_bytes,omitempty"`
	BytesSent                  int64                 `json:"tx_bytes,omitempty"`
	ConfigNetwork              ConfigNetwork         `json:"config_network,omitempty"`
	ConfigVersion              string                `json:"cfgversion,omitempty"`
	ConnectRequestIP           IP                    `json:"connect_request_ip,omitempty"`
	ConnectRequestPort         string                `json:"connect_request_port,omitempty"`
	ConnectedAt                TimeStamp             `json:"connected_at,omitempty"`
	ConsideredLostAt           TimeStamp             `json:"considered_lost_at,omitempty"`
	DHCPServerTable            []DHCPServer          `json:"dhcp_server_table,omitempty"`
	DeviceID                   string                `json:"device_id,omitempty"`
	DeviceType                 string                `json:"type,omitempty"`
	DiscoveredVia              string                `json:"discovered_via,omitempty"`
	DownlinkTable              []Downlink            `json:"downlink_table,omitempty"`
	EthernetTable              []EthernetDevice      `json:"ethernet_table,omitempty"`
	FirmwareCapabilities       int64                 `json:"fw_caps,omitempty"`
	GatewayMAC                 MAC                   `json:"gateway_mac,omitempty"`
	GeneralTemperature         int64                 `json:"general_temperature,omitempty"`
	GuestNumSTA                int64                 `json:"guest-num_sta,omitempty"`
	HardwareCapabilities       int64                 `json:"hw_caps,omitempty"`
	HasFan                     bool                  `json:"has_fan,omitempty"`
	HasInternet                bool                  `json:"internet,omitempty"`
	HasTemperature             bool                  `json:"has_temperature,omitempty"`
	HasTwoPhaseAdopt           bool                  `json:"two_phase_adopt,omitempty"`
	HashID                     string                `json:"hash_id,omitempty"`
	IP                         IP                    `json:"ip,omitempty"`
	InformIP                   IP                    `json:"inform_ip,omitempty"`
	InformURL                  string                `json:"inform_url,omitempty"`
	IsAdoptableWhenUpgraded    bool                  `json:"adoptable_when_upgraded,omitempty"`
	IsAdopted                  bool                  `json:"adopted,omitempty"`
	IsDefault                  bool                  `json:"default,omitempty"`
	IsDot1xPortCtrlEnabled     bool                  `json:"dot1x_portctrl_enabled,omitempty"`
	IsFlowCtrlEnabled          bool                  `json:"flowctrl_enabled,omitempty"`
	IsJumboFrameEnabled        bool                  `json:"jumboframe_enabled,omitempty"`
	IsLocating                 bool                  `json:"locating,omitempty"`
	IsModelInEol               bool                  `json:"model_in_eol,omitempty"`
	IsModelInLts               bool                  `json:"model_in_lts,omitempty"`
	IsModelIncompatible        bool                  `json:"model_incompatible,omitempty"`
	IsOverheating              bool                  `json:"overheating,omitempty"`
	IsPowerSourceCtrlEnabled   bool                  `json:"power_source_ctrl_enabled,omitempty"`
	IsUnsupported              bool                  `json:"unsupported,omitempty"`
	IsUpgradable               bool                  `json:"upgradable,omitempty"`
	IsXAESGCM                  bool                  `json:"x_aes_gcm,omitempty"`
	KernelVersion              string                `json:"kernel_version,omitempty"`
	KnownConfigVersion         string                `json:"known_cfgversion,omitempty"`
	LCMBrightnessOverride      bool                  `json:"lcm_brightness_override,omitempty"`
	LCMIdleTimeoutOverride     bool                  `json:"lcm_idle_timeout_override,omitempty"`
	LEDOverride                string                `json:"led_override,omitempty"`
	LEDOverrideColor           string                `json:"led_override_color,omitempty"`
	LEDOverrideColorBrightness int64                 `json:"led_override_color_brightness,omitempty"`
	LLDPTable                  []LLDP                `json:"lldp_table,omitempty"`
	LastSeen                   TimeStamp             `json:"last_seen,omitempty"`
	LastUplink                 UplinkSummary         `json:"last_uplink,omitempty"`
	LicenseState               string                `json:"license_state,omitempty"`
	MAC                        MAC                   `json:"mac,omitempty"`
	ManufacturerID             int64                 `json:"manufacturer_id,omitempty"`
	Model                      string                `json:"model,omitempty"`
	Name                       string                `json:"name,omitempty"`
	NextHeartbeatAt            TimeStamp             `json:"next_heartbeat_at,omitempty"`
	NextInterval               Duration              `json:"next_interval,omitempty"`
	NumSTA                     int64                 `json:"num_sta,omitempty"`
	OutdoorModeOverride        string                `json:"outdoor_mode_override,omitempty"`
	PortTable                  []Port                `json:"port_table,omitempty"`
	PreviousNonBusyState       int64                 `json:"prev_non_busy_state,omitempty"`
	ProvisionedAt              TimeStamp             `json:"provisioned_at,omitempty"`
	RequiredVersion            string                `json:"required_version,omitempty"`
	RollUpgrade                bool                  `json:"rollupgrade,omitempty"`
	SSHSessionTable            []SSHSession          `json:"ssh_session_table,omitempty"`
	STPPriority                Number                `json:"stp_priority,omitempty"`
	STPVersion                 string                `json:"stp_version,omitempty"`
	Satisfaction               int64                 `json:"satisfaction,omitempty"`
	Serial                     string                `json:"serial,omitempty"`
	SiteID                     string                `json:"site_id,omitempty"`
	StartConnected             TimeStampMilliseconds `json:"start_connected_millis,omitempty"`
	StartDisconnected          TimeStampMilliseconds `json:"start_disconnected_millis,omitempty"`
	Stat                       map[string]Stat       `json:"stat,omitempty"`
	State                      int64                 `json:"state,omitempty"`
	SwitchCapabilities         SwitchCapabilities    `json:"switch_caps,omitempty"`
	SysErrorCapabilities       int64                 `json:"sys_error_caps,omitempty"`
	SysStats                   SysStats              `json:"sys_stats,omitempty"`
	SyslogKey                  string                `json:"syslog_key,omitempty"`
	SystemStats                SystemStats           `json:"system-stats,omitempty"`
	TotalMaxPower              int64                 `json:"total_max_power,omitempty"`
	UnsupportedReason          int64                 `json:"unsupported_reason,omitempty"`
	UpgradeToFirmware          string                `json:"upgrade_to_firmware,omitempty"`
	Uplink                     Uplink                `json:"uplink,omitempty"`
	Uptime                     Duration              `json:"uptime,omitempty"`
	UserNumSTA                 int64                 `json:"user-num_sta,omitempty"`
	Version                    string                `json:"version,omitempty"`
	XAuthKey                   string                `json:"x_authkey,omitempty"`
	XFingerprint               string                `json:"x_fingerprint,omitempty"`
	XHasSSHHostKey             bool                  `json:"x_has_ssh_hostkey,omitempty"`
	XInformAuthKey             string                `json:"x_inform_authkey,omitempty"`
	XSSHHostKeyFingerprint     string                `json:"x_ssh_hostkey_fingerprint,omitempty"`
}

func (d *Device) UniqueID() string { return d.ID }

func (d *Device) String() string {
	traffic := ""
	if d.BytesReceived+d.BytesSent > 0 {
		recvd := formatBytesSize(d.BytesReceived)
		sent := formatBytesSize(d.BytesSent)
		traffic = fmt.Sprintf("%10s ↓ / %10s ↑", recvd, sent)
	}

	temp := ""
	if d.HasTemperature {
		temp = fmt.Sprintf("%d°C", d.GeneralTemperature)
	}

	return fmt.Sprintf("%25s   %-15s %-4s %-35s %s", d.Name, d.IP, temp, d.SystemStats, traffic)
}

type ConfigNetwork struct {
	DNS1        string `json:"dns1,omitempty"`
	DNS2        string `json:"dns2,omitempty"`
	DNSSuffix   string `json:"dnssuffix,omitempty"`
	Gateway     string `json:"gateway,omitempty"`
	IP          IP     `json:"ip,omitempty"`
	Netmask     string `json:"netmask,omitempty"`
	NetworkType string `json:"type,omitempty"`
}

type EthernetDevice struct {
	MAC     MAC    `json:"mac,omitempty"`
	Name    string `json:"name,omitempty"`
	NumPort int64  `json:"num_port,omitempty"`
}

type Port struct {
	Name string `json:"name,omitempty"`

	Anomalies             int64   `json:"anomalies,omitempty"`
	Autonegotiate         bool    `json:"autoneg,omitempty"`
	BytesErr              float64 `json:"bytes-r,omitempty"`
	Dot1xMode             string  `json:"dot1x_mode,omitempty"`
	Dot1xStatus           string  `json:"dot1x_status,omitempty"`
	Enable                bool    `json:"enable,omitempty"`
	HasFlowcontrolReceive bool    `json:"flowctrl_rx,omitempty"`
	HasFlowcontrolSend    bool    `json:"flowctrl_tx,omitempty"`
	HasJumbo              bool    `json:"jumbo,omitempty"`
	IsAggregatedBy        bool    `json:"aggregated_by,omitempty"`
	IsFullDuplex          bool    `json:"full_duplex,omitempty"`
	IsMasked              bool    `json:"masked,omitempty"`
	IsPortPOE             bool    `json:"port_poe,omitempty"`
	IsUp                  bool    `json:"up,omitempty"`
	IsUplink              bool    `json:"is_uplink,omitempty"`
	Media                 string  `json:"media,omitempty"`
	OPMode                string  `json:"op_mode,omitempty"`
	POECapabilities       int64   `json:"poe_caps,omitempty"`
	POEClass              string  `json:"poe_class,omitempty"`
	POECurrent            string  `json:"poe_current,omitempty"`
	POEEnable             bool    `json:"poe_enable,omitempty"`
	POEGood               bool    `json:"poe_good,omitempty"`
	POEMode               string  `json:"poe_mode,omitempty"`
	POEPower              string  `json:"poe_power,omitempty"`
	POEVoltage            string  `json:"poe_voltage,omitempty"`
	PortConfigID          string  `json:"portconf_id,omitempty"`
	PortIndex             int64   `json:"port_idx,omitempty"`
	ReceiveBroadcast      int64   `json:"rx_broadcast,omitempty"`
	ReceiveBytes          int64   `json:"rx_bytes,omitempty"`
	ReceiveBytesErr       float64 `json:"rx_bytes-r,omitempty"`
	ReceiveDropped        int64   `json:"rx_dropped,omitempty"`
	ReceiveErrors         int64   `json:"rx_errors,omitempty"`
	ReceiveMulticast      int64   `json:"rx_multicast,omitempty"`
	ReceivePackets        int64   `json:"rx_packets,omitempty"`
	STPPathCost           int64   `json:"stp_pathcost,omitempty"`
	STPState              string  `json:"stp_state,omitempty"`
	Satisfaction          int64   `json:"satisfaction,omitempty"`
	SatisfactionReason    int64   `json:"satisfaction_reason,omitempty"`
	SendBytes             int64   `json:"tx_bytes,omitempty"`
	SendBytesErr          float64 `json:"tx_bytes-r,omitempty"`
	SendDropped           int64   `json:"tx_dropped,omitempty"`
	SendErrors            int64   `json:"tx_errors,omitempty"`
	SendMulticast         int64   `json:"tx_multicast,omitempty"`
	SendPackets           int64   `json:"tx_packets,omitempty"`
	Sendbroadcast         int64   `json:"tx_broadcast,omitempty"`
	Speed                 int64   `json:"speed,omitempty"`
	SpeedCapabilities     int64   `json:"speed_caps,omitempty"`
}

type SwitchCapabilities struct {
	FeatureCapabilities  int64 `json:"feature_caps,omitempty"`
	MaxAggregateSessions int64 `json:"max_aggregate_sessions,omitempty"`
	MaxL3Intf            int64 `json:"max_l3_intf,omitempty"`
	MaxMirrorSessions    int64 `json:"max_mirror_sessions,omitempty"`
	MaxReservedRoutes    int64 `json:"max_reserved_routes,omitempty"`
	MaxStaticRoutes      int64 `json:"max_static_routes,omitempty"`
}

type UplinkSummary struct {
	PortIndex        int64 `json:"port_idx,omitempty"`
	UplinkMAC        MAC   `json:"uplink_mac,omitempty"`
	UplinkRemotePort int64 `json:"uplink_remote_port,omitempty"`
}

type Uplink struct {
	UplinkSummary

	IP               IP      `json:"ip,omitempty"`
	IsFullDuplex     bool    `json:"full_duplex,omitempty"`
	IsUp             bool    `json:"up,omitempty"`
	MAC              MAC     `json:"mac,omitempty"`
	MaxSpeed         int64   `json:"max_speed,omitempty"`
	Media            string  `json:"media,omitempty"`
	Name             string  `json:"name,omitempty"`
	Netmask          string  `json:"netmask,omitempty"`
	NumPort          int64   `json:"num_port,omitempty"`
	PortIndex        int64   `json:"port_idx,omitempty"`
	ReceiveBytes     int64   `json:"rx_bytes,omitempty"`
	ReceiveBytesErr  float64 `json:"rx_bytes-r,omitempty"`
	ReceiveDropped   int64   `json:"rx_dropped,omitempty"`
	ReceiveErrors    int64   `json:"rx_errors,omitempty"`
	ReceiveMulticast int64   `json:"rx_multicast,omitempty"`
	ReceivePackets   int64   `json:"rx_packets,omitempty"`
	SendBytes        int64   `json:"tx_bytes,omitempty"`
	SendBytesErr     float64 `json:"tx_bytes-r,omitempty"`
	SendDropped      int64   `json:"tx_dropped,omitempty"`
	SendErrors       int64   `json:"tx_errors,omitempty"`
	SendPackets      int64   `json:"tx_packets,omitempty"`
	Speed            int64   `json:"speed,omitempty"`
	UplinkMAC        MAC     `json:"uplink_mac,omitempty"`
	UplinkRemotePort int64   `json:"uplink_remote_port,omitempty"`
	UplinkType       string  `json:"type,omitempty"`
}

type SysStats struct {
	LoadAvg1  string `json:"loadavg_1,omitempty"`
	LoadAvg15 string `json:"loadavg_15,omitempty"`
	LoadAvg5  string `json:"loadavg_5,omitempty"`
	MemBuffer int64  `json:"mem_buffer,omitempty"`
	MemTotal  int64  `json:"mem_total,omitempty"`
	MemUsed   int64  `json:"mem_used,omitempty"`
}

type SystemStats struct {
	CPU    string `json:"cpu,omitempty"`
	Mem    string `json:"mem,omitempty"`
	Uptime string `json:"uptime,omitempty"`
}

func (s SystemStats) String() string {
	if len(s.CPU)+len(s.Mem)+len(s.Uptime) == 0 {
		return ""
	}

	uptime := ""
	if u, err := strconv.ParseInt(s.Uptime, 10, 64); err == nil {
		uptime = Duration(u).String()
	}
	return fmt.Sprintf("%4s%% cpu / %-4s%% mem  %s", s.CPU, s.Mem, uptime)
}

type SSHSession struct{}

type DHCPServer struct{}

type LLDP struct {
	ChassisID      string `json:"chassis_id,omitempty"`
	IsWired        bool   `json:"is_wired,omitempty"`
	LocalPortIndex int64  `json:"local_port_idx,omitempty"`
	LocalPortName  string `json:"local_port_name,omitempty"`
	PortID         string `json:"port_id,omitempty"`
}

type Downlink struct {
	PortIndex    int64 `json:"port_idx,omitempty"`
	Speed        int64 `json:"speed,omitempty"`
	IsFullDuplex bool  `json:"full_duplex,omitempty"`
	MAC          MAC   `json:"mac,omitempty"`
}

type Stat struct {
	SiteID           string                `json:"site_id,omitempty"`
	O                string                `json:"o,omitempty"`
	OID              string                `json:"oid,omitempty"`
	SW               string                `json:"sw,omitempty"`
	Time             TimeStampMilliseconds `json:"time,omitempty"`
	Datetime         string                `json:"datetime,omitempty"`
	ReceivePackets   float64               `json:"rx_packets,omitempty"`
	ReceiveBytes     float64               `json:"rx_bytes,omitempty"`
	ReceiveErrors    float64               `json:"rx_errors,omitempty"`
	ReceiveDropped   float64               `json:"rx_dropped,omitempty"`
	ReceiveCrypts    float64               `json:"rx_crypts,omitempty"`
	ReceiveFrags     float64               `json:"rx_frags,omitempty"`
	SendPackets      float64               `json:"tx_packets,omitempty"`
	SendBytes        float64               `json:"tx_bytes,omitempty"`
	SendErrors       float64               `json:"tx_errors,omitempty"`
	SendDropped      float64               `json:"tx_dropped,omitempty"`
	SendRetries      float64               `json:"tx_retries,omitempty"`
	ReceiveMulticast float64               `json:"rx_multicast,omitempty"`
	ReceiveBroadcast float64               `json:"rx_broadcast,omitempty"`
	SendMulticast    float64               `json:"tx_multicast,omitempty"`
	SendBroadcast    float64               `json:"tx_broadcast,omitempty"`
	Bytes            float64               `json:"bytes,omitempty"`
	Duration         float64               `json:"duration,omitempty"`
	/*
	   port_1-rx_packets int64 `json:"port_1-rx_packets,omitempty"`
	   port_1-rx_bytes int64 `json:"port_1-rx_bytes,omitempty"`
	   port_1-rx_dropped int64 `json:"port_1-rx_dropped,omitempty"`
	   port_1-tx_packets int64 `json:"port_1-tx_packets,omitempty"`
	   port_1-tx_bytes int64 `json:"port_1-tx_bytes,omitempty"`
	   port_1-rx_multicast int64 `json:"port_1-rx_multicast,omitempty"`
	   port_1-tx_multicast int64 `json:"port_1-tx_multicast,omitempty"`
	   port_1-tx_broadcast int64 `json:"port_1-tx_broadcast,omitempty"`
	   port_2-rx_packets int64 `json:"port_2-rx_packets,omitempty"`
	   port_2-rx_bytes int64 `json:"port_2-rx_bytes,omitempty"`
	   port_2-rx_dropped int64 `json:"port_2-rx_dropped,omitempty"`
	   port_2-tx_packets int64 `json:"port_2-tx_packets,omitempty"`
	   port_2-tx_bytes int64 `json:"port_2-tx_bytes,omitempty"`
	   port_2-rx_multicast int64 `json:"port_2-rx_multicast,omitempty"`
	   port_2-rx_broadcast int64 `json:"port_2-rx_broadcast,omitempty"`
	   port_2-tx_multicast int64 `json:"port_2-tx_multicast,omitempty"`
	   port_2-tx_broadcast int64 `json:"port_2-tx_broadcast,omitempty"`
	   port_4-tx_packets int64 `json:"port_4-tx_packets,omitempty"`
	   port_4-tx_bytes int64 `json:"port_4-tx_bytes,omitempty"`
	   port_4-tx_multicast int64 `json:"port_4-tx_multicast,omitempty"`
	   port_4-tx_broadcast int64 `json:"port_4-tx_broadcast,omitempty"`
	   port_8-rx_packets int64 `json:"port_8-rx_packets,omitempty"`
	   port_8-rx_bytes int64 `json:"port_8-rx_bytes,omitempty"`
	   port_8-rx_dropped int64 `json:"port_8-rx_dropped,omitempty"`
	   port_8-tx_packets int64 `json:"port_8-tx_packets,omitempty"`
	   port_8-tx_bytes int64 `json:"port_8-tx_bytes,omitempty"`
	   port_8-rx_multicast int64 `json:"port_8-rx_multicast,omitempty"`
	   port_8-rx_broadcast int64 `json:"port_8-rx_broadcast,omitempty"`
	   port_8-tx_multicast int64 `json:"port_8-tx_multicast,omitempty"`
	   port_8-tx_broadcast int64 `json:"port_8-tx_broadcast,omitempty"`
	   port_1-rx_broadcast int64 `json:"port_1-rx_broadcast,omitempty"`
	   port_4-rx_packets int64 `json:"port_4-rx_packets,omitempty"`
	   port_4-rx_bytes int64 `json:"port_4-rx_bytes,omitempty"`
	   port_4-rx_multicast int64 `json:"port_4-rx_multicast,omitempty"`
	   port_4-rx_broadcast int64 `json:"port_4-rx_broadcast,omitempty"`
	*/
}

// DeviceOrderedBy returns a DeviceSorter that sorts by the provided less functions.
func DeviceOrderedBy(less ...DeviceLessFn) *DeviceSorter {
	return &DeviceSorter{less: less}
}

// DeviceLessFn describes a less function for a Device.
type DeviceLessFn func(lhs, rhs *Device) bool

// DeviceSorter is a multisorter for sorting slices of Device.
type DeviceSorter struct {
	devices []Device
	less    []DeviceLessFn
}

// Sort applies the configured less functions in order.
func (s *DeviceSorter) Sort(clients []Device) {
	s.devices = clients
	sort.Sort(s)
}

func (s *DeviceSorter) Len() int      { return len(s.devices) }
func (s *DeviceSorter) Swap(i, j int) { s.devices[i], s.devices[j] = s.devices[j], s.devices[i] }
func (s *DeviceSorter) Less(i, j int) bool {
	lhs, rhs := &s.devices[i], &s.devices[j]
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
