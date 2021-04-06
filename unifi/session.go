package unifi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
	ID                  string `json:"_id,omitempty"`
	MAC                 string `json:"mac,omitempty"`
	SiteID              string `json:"site_id,omitempty"`
	OUI                 string `json:"oui,omitempty"`
	NetworkID           string `json:"network_id,omitempty"`
	IP                  string `json:"ip,omitempty"`
	FixedIP             string `json:"fixed_ip,omitempty"`
	Hostname            string `json:"hostname,omitempty"`
	UsergroupID         string `json:"usergroup_id,omitempty"`
	Name                string `json:"name,omitempty"`
	FirstSeen           int64  `json:"first_seen,omitempty"`
	LastSeen            int64  `json:"last_seen,omitempty"`
	DeviceIDOverride    int    `json:"dev_id_override,omitempty"`
	FingerprintOverride bool   `json:"fingerprint_override,omitempty"`
	Blocked             bool   `json:"blocked,omitempty"`
	IsGuest             bool   `json:"is_guest,omitempty"`
	IsWired             bool   `json:"is_wired,omitempty"`
	Noted               bool   `json:"noted,omitempty"`
	UseFixedIP          bool   `json:"use_fixedip,omitempty"`
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
}

// Initialize prepares the session for use.
func (s *Session) Initialize() error {
	if s == nil {
		return ErrNilSession
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

// ListClients describes the known UniFi clients.
func (s *Session) ListClients() (string, error) {
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
				log.Printf("%s\nerror blocking: %v", res, err)

				return
			}

			log.Printf("%s\n", res)
		}
	}
}

func (s *Session) UnblockFn(clients []Client, keys map[string]bool) {
	for _, client := range clients {
		display := firstNonEmpty(client.Name, client.Hostname)
		if k, ok := keys[display]; ok && k {
			res, err := s.Unblock(client.MAC)
			if err != nil {
				log.Printf("%s\nerror unblocking: %v", res, err)

				return
			}

			log.Printf("%s\n", res)
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
			log.Printf("error encoding JSON: %v", err)
		}

		return
	}

	now := time.Now().Unix()

	sort.Slice(clients, func(i, j int) bool { return clients[i].LastSeen < clients[j].LastSeen })

	const cutOff = 60 * 60 * 12 // 12 hours
	for _, client := range clients {
		if (now - client.LastSeen) > cutOff {
			continue
		}

		display := firstNonEmpty(client.Name, client.Hostname, client.MAC, "-")
		ip := firstNonEmpty(client.IP, client.FixedIP)
		lastSeen := time.Unix(client.LastSeen, 0)

		guest := ""
		if client.IsGuest {
			guest = "✓"
		}

		log.Printf("%30s %2s %15s %s", display, guest, ip, lastSeen.Format(time.Kitchen))
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
