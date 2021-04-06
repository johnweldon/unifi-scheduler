package unifi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sort"
	"time"
)

var (
	ErrNilSession           = errors.New("nil session")
	ErrUninitializedSession = errors.New("uninitialized session")
	ErrTooManyWriters       = errors.New("too many writers")
)

// Response encapsulates a UniFi http response.
type Response struct {
	Meta struct {
		RC string `json:"rc,omitempty"`
	}
	Data []Client `json:"data,omitempty"`
}

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

// Session wraps metadata to manage session state.
type Session struct {
	Endpoint string
	Username string
	Password string
	UseJSON  bool
	csrf     string
	client   *http.Client
	login    func() (string, error)
	err      error

	outWriter io.Writer
	errWriter io.Writer
}

// Initialize prepares the session for use.
func (s *Session) Initialize(writers ...io.Writer) error {
	if s == nil {
		return ErrNilSession
	}

	// nolint: gomnd
	switch len(writers) {
	case 2:
		s.outWriter = writers[0]
		s.errWriter = writers[1]
	case 1:
		s.outWriter = writers[0]
		s.errWriter = os.Stderr
	case 0:
		s.outWriter = os.Stdout
		s.errWriter = os.Stderr
	default:
		return ErrTooManyWriters
	}

	s.err = nil

	if len(s.Endpoint) == 0 {
		s.setErrorString("missing endpoint")
	}

	if len(s.Username) == 0 {
		s.setErrorString("missing username")
	}

	if len(s.Password) == 0 {
		s.setErrorString("missing password")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		s.setError(err)
	}

	s.client = &http.Client{ // nolint:exhaustivestruct
		Jar:     jar,
		Timeout: time.Minute * 1,
	}
	s.login = s.webLogin

	return s.err
}

// Login performs authentication with the UniFi server, and stores the
// http credentials.
func (s *Session) Login() (string, error) {
	if s.login == nil {
		s.login = func() (string, error) {
			return "", ErrUninitializedSession
		}
	}

	return s.login()
}

// ListUsers describes the known UniFi clients.
func (s *Session) ListUsers() (string, error) {
	if s.err != nil {
		return "", s.err
	}

	u, err := url.Parse(fmt.Sprintf("%s/proxy/network/api/s/default/rest/user", s.Endpoint))
	if err != nil {
		s.setError(err)

		return "", s.err
	}

	return s.get(u)
}

// ListClients describes currently connected clients.
func (s *Session) ListClients() (string, error) {
	if s.err != nil {
		return "", s.err
	}

	u, err := url.Parse(fmt.Sprintf("%s/proxy/network/api/s/default/stat/sta", s.Endpoint))
	if err != nil {
		s.setError(err)

		return "", s.err
	}

	return s.get(u)
}

// Kick disconnects a connected client, identified by MAC address.
func (s *Session) Kick(mac string) (string, error) {
	return s.macAction("kick-sta", mac)
}

// Block prevents a specific client (identified by MAC) from connecting
// to the UniFi network.
func (s *Session) Block(mac string) (string, error) {
	return s.macAction("block-sta", mac)
}

// Unblock re-enables a specific client.
func (s *Session) Unblock(mac string) (string, error) {
	return s.macAction("unblock-sta", mac)
}

func (s *Session) BlockFn(clients []Client, keys map[string]bool) {
	for _, client := range clients {
		display := firstNonEmpty(client.Name, client.Hostname)
		if k, ok := keys[display]; ok && k {
			res, err := s.Block(client.MAC)
			if err != nil {
				fmt.Fprintf(s.errWriter, "%s\nerror blocking: %v\n", res, err)

				return
			}

			fmt.Fprintf(s.outWriter, "%s\n", res)
		}
	}
}

func (s *Session) UnblockFn(clients []Client, keys map[string]bool) {
	for _, client := range clients {
		display := firstNonEmpty(client.Name, client.Hostname)
		if k, ok := keys[display]; ok && k {
			res, err := s.Unblock(client.MAC)
			if err != nil {
				fmt.Fprintf(s.errWriter, "%s\nerror unblocking: %v\n", res, err)

				return
			}

			fmt.Fprintf(s.outWriter, "%s\n", res)
		}
	}
}

func (s *Session) webLogin() (string, error) {
	if s.err != nil {
		return "", s.err
	}

	u, err := url.Parse(fmt.Sprintf("%s/api/auth/login", s.Endpoint))
	if err != nil {
		s.setError(err)

		return "", s.err
	}

	r := bytes.NewBufferString(
		fmt.Sprintf(
			`{"username":%q,"password":%q,"strict":"true","remember":"true"}`,
			s.Username, s.Password))

	respBody, err := s.post(u, r)
	if err == nil {
		s.login = func() (string, error) { return respBody, nil }
	}

	return respBody, err
}

func (s *Session) macAction(action string, mac string) (string, error) {
	if b, err := s.login(); err != nil {
		return b, err
	}

	u, err := url.Parse(fmt.Sprintf("%s/proxy/network/api/s/default/cmd/stamgr", s.Endpoint))
	if err != nil {
		s.setError(err)

		return "", s.err
	}

	r := bytes.NewBufferString(fmt.Sprintf(`{"cmd":%q,"mac":%q}`, action, mac))

	return s.post(u, r)
}

func (s *Session) get(u fmt.Stringer) (string, error) {
	return s.verb("GET", u, nil)
}

func (s *Session) post(u fmt.Stringer, body io.Reader) (string, error) {
	return s.verb("POST", u, body)
}

func (s *Session) verb(verb string, u fmt.Stringer, body io.Reader) (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), verb, u.String(), body)
	if err != nil {
		s.setError(err)

		return "", s.err
	}

	req.Header.Set("User-Agent", "unifibot 2.0")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", s.Endpoint)

	if s.csrf != "" {
		req.Header.Set("x-csrf-token", s.csrf)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.setError(err)

		return "", s.err
	}
	defer resp.Body.Close()

	if tok := resp.Header.Get("x-csrf-token"); tok != "" {
		s.csrf = tok
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.setError(err)

		return "", s.err
	}

	if resp.StatusCode < http.StatusOK || http.StatusBadRequest <= resp.StatusCode {
		s.setErrorString(http.StatusText(resp.StatusCode))
	}

	return string(respBody), s.err
}

func (s *Session) setError(e error) {
	if e == nil {
		return
	}

	if s.err == nil {
		s.err = fmt.Errorf("%w", e)
	} else {
		s.err = fmt.Errorf("%s\n%w", e, s.err)
	}
}

func (s *Session) setErrorString(e string) {
	if len(e) == 0 {
		return
	}

	if s.err == nil {
		s.err = fmt.Errorf("%s", e) // nolint:goerr113
	} else {
		s.err = fmt.Errorf("%s\n%w", e, s.err)
	}
}

func (s *Session) ListFn(clients []Client, _ map[string]bool) {
	if s.UseJSON {
		if err := json.NewEncoder(os.Stdout).Encode(clients); err != nil {
			fmt.Fprintf(s.errWriter, "error encoding JSON: %v\n", err)
		}

		return
	}

	sort.Slice(clients, func(i, j int) bool { return clients[i].LastSeen < clients[j].LastSeen })

	for _, client := range clients {
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

		fmt.Fprintf(s.outWriter, "%20s %-2s %-2s %-2s %-15s %s\n",
			display,
			blocked,
			guest,
			wired,
			ip,
			uptime,
		)
	}
}

func firstNonEmpty(s ...string) string {
	for _, candidate := range s {
		if len(candidate) > 0 {
			return candidate
		}
	}

	return ""
}
