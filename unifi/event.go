package unifi

import (
	"fmt"
	"time"
)

type EventType string

var (
	EventTypeAccessPointAutoReadopted        = EventType("EVT_AP_AutoReadopted")
	EventTypeAccessPointChannelChanged       = EventType("EVT_AP_ChannelChanged")
	EventTypeAccessPointConnected            = EventType("EVT_AP_Connected")
	EventTypeAccessPointDetectRogueAP        = EventType("EVT_AP_DetectRogueAP")
	EventTypeAccessPointIsolated             = EventType("EVT_AP_Isolated")
	EventTypeAccessPointLostContact          = EventType("EVT_AP_Lost_Contact")
	EventTypeAccessPointPossibleInterference = EventType("EVT_AP_PossibleInterference")
	EventTypeAccessPointRestarted            = EventType("EVT_AP_Restarted")
	EventTypeAccessPointRestartedUnknown     = EventType("EVT_AP_RestartedUnknown")
	EventTypeAccessPointUpgradeScheduled     = EventType("EVT_AP_UpgradeScheduled")
	EventTypeAccessPointUpgraded             = EventType("EVT_AP_Upgraded")
	EventTypeBridgeAutoReadopted             = EventType("EVT_BB_AutoReadopted")
	EventTypeBridgeChannelChanged            = EventType("EVT_BB_ChannelChanged")
	EventTypeBridgeConnected                 = EventType("EVT_BB_Connected")
	EventTypeBridgeLinkRadioChanged          = EventType("EVT_BB_LinkRadioChanged")
	EventTypeBridgeLostContact               = EventType("EVT_BB_Lost_Contact")
	EventTypeBridgeRestarted                 = EventType("EVT_BB_Restarted")
	EventTypeBridgeRestartedUnknown          = EventType("EVT_BB_RestartedUnknown")
	EventTypeDMConnected                     = EventType("EVT_DM_Connected")
	EventTypeDMUpgraded                      = EventType("EVT_DM_Upgraded")
	EventTypeLANClientBlocked                = EventType("EVT_LC_Blocked")
	EventTypeLANClientUnblocked              = EventType("EVT_LC_Unblocked")
	EventTypeLANGuestConnected               = EventType("EVT_LG_Connected")
	EventTypeLANGuestDisconnected            = EventType("EVT_LG_Disconnected")
	EventTypeLANUserConnected                = EventType("EVT_LU_Connected")
	EventTypeSwitchAutoReadopted             = EventType("EVT_SW_AutoReadopted")
	EventTypeSwitchConnected                 = EventType("EVT_SW_Connected")
	EventTypeSwitchDetectRogueDHCP           = EventType("EVT_SW_DetectRogueDHCP")
	EventTypeSwitchLostContact               = EventType("EVT_SW_Lost_Contact")
	EventTypeSwitchRestarted                 = EventType("EVT_SW_Restarted")
	EventTypeSwitchRestartedUnknown          = EventType("EVT_SW_RestartedUnknown")
	EventTypeSwitchUpgradeScheduled          = EventType("EVT_SW_UpgradeScheduled")
	EventTypeSwitchUpgraded                  = EventType("EVT_SW_Upgraded")
	EventTypeWirelessClientBlocked           = EventType("EVT_WC_Blocked")
	EventTypeWirelessClientUnblocked         = EventType("EVT_WC_Unblocked")
	EventTypeWirelessGuestDisconnected       = EventType("EVT_WG_Disconnected")
	EventTypeWirelessUserConnected           = EventType("EVT_WU_Connected")
	EventTypeWirelessUserDisconnected        = EventType("EVT_WU_Disconnected")
	EventTypeWirelessUserRoam                = EventType("EVT_WU_Roam")
	EventTypeWirelessUserRoamRadio           = EventType("EVT_WU_RoamRadio")
)

type Event struct {
	ID                 string    `json:"_id,omitempty"`
	Key                EventType `json:"key,omitempty"`
	AccessPoint        MAC       `json:"ap,omitempty"`
	AccessPointFrom    MAC       `json:"ap_from,omitempty"`
	AccessPointTo      MAC       `json:"ap_to,omitempty"`
	Bridge             MAC       `json:"bb,omitempty"`
	Client             MAC       `json:"client,omitempty"`
	DM                 MAC       `json:"dm,omitempty"`
	Gateway            MAC       `json:"gw,omitempty"`
	Guest              MAC       `json:"guest,omitempty"`
	MAC                MAC       `json:"mac,omitempty"`
	Switch             MAC       `json:"sw,omitempty"`
	User               MAC       `json:"user,omitempty"`
	IP                 IP        `json:"ip,omitempty"`
	DateTime           time.Time `json:"datetime,omitempty"`
	Duration           Duration  `json:"duration,omitempty"`
	TimeStamp          TimeStamp `json:"time,omitempty"`
	AccessPointDisplay string    `json:"ap_displayName,omitempty"`
	AccessPointModel   string    `json:"ap_model,omitempty"`
	AccessPointName    string    `json:"ap_name,omitempty"`
	Admin              string    `json:"admin,omitempty"`
	BridgeDisplay      string    `json:"bb_displayName,omitempty"`
	BridgeModel        string    `json:"bb_model,omitempty"`
	BridgeName         string    `json:"bb_name,omitempty"`
	Channel            string    `json:"channel,omitempty"`
	ChannelFrom        Number    `json:"channel_from,omitempty"`
	ChannelTo          Number    `json:"channel_to,omitempty"`
	DMDisplay          string    `json:"dm_displayName,omitempty"`
	DMModel            string    `json:"dm_model,omitempty"`
	DMName             string    `json:"dm_name,omitempty"`
	ESSID              string    `json:"essid,omitempty"`
	GatewayDisplay     string    `json:"gw_displayName,omitempty"`
	Hostname           string    `json:"hostname,omitempty"`
	Message            string    `json:"msg,omitempty"`
	Name               string    `json:"name,omitempty"`
	Network            string    `json:"network,omitempty"`
	Radio              string    `json:"radio,omitempty"`
	RadioFrom          string    `json:"radio_from,omitempty"`
	RadioTo            string    `json:"radio_to,omitempty"`
	RogueChannel       string    `json:"rogue_channel,omitempty"`
	SSID               string    `json:"ssid,omitempty"`
	SiteID             string    `json:"site_id,omitempty"`
	Subsystem          string    `json:"subsystem,omitempty"`
	SwitchDisplay      string    `json:"sw_displayName,omitempty"`
	SwitchModel        string    `json:"sw_model,omitempty"`
	SwitchName         string    `json:"sw_name,omitempty"`
	VersionFrom        string    `json:"version_from,omitempty"`
	VersionTo          string    `json:"version_to,omitempty"`
	IsAdmin            bool      `json:"is_admin,omitempty"`
	IsNegative         bool      `json:"is_negative,omitempty"`
	Bytes              int64     `json:"bytes,omitempty"`
	NumSta             int64     `json:"num_sta,omitempty"`
	Port               int64     `json:"port,omitempty"`
	SignalStrength     int64     `json:"signal_strength,omitempty"`
	VLAN               int64     `json:"vlan,omitempty"`
}

func (e Event) UniqueID() string { return e.ID }

func (e Event) String() string {
	const maxMsgLen = 100

	msg := e.Message
	if len(msg) > maxMsgLen {
		msg = msg[:maxMsgLen]
	}

	return fmt.Sprintf(
		"%25s %-30s %s",
		e.Key,
		e.DateTime,
		msg,
	)
}
