package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
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
	ID          string `json:"_id,omitempty"`
	MAC         string `json:"mac,omitempty"`
	SiteID      string `json:"site_id,omitempty"`
	OUI         string `json:"oui,omitempty"`
	IsGuest     bool   `json:"is_guest,omitempty"`
	FirstSeen   int    `json:"first_seen,omitempty"`
	LastSeen    int    `json:"last_seen,omitempty"`
	IsWired     bool   `json:"is_wired,omitempty"`
	UsergroupID string `json:"usergroup_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Noted       bool   `json:"noted,omitempty"`
	UseFixedIP  bool   `json:"use_fixedip,omitempty"`
	NetworkID   string `json:"network_id,omitempty"`
	IP          string `json:"ip,omitempty"`
	FixedIP     string `json:"fixed_ip,omitempty"`
	Hostname    string `json:"hostname,omitempty"`
}

// UnifiSession wraps metadata to manage session state.
type UnifiSession struct {
	Endpoint string
	Username string
	Password string
	client   *http.Client
	login    func() (string, error)
	err      error
}

// Initialize prepares the session for use.
func (s *UnifiSession) Initialize() error {
	if s == nil {
		return errors.New("nil session")
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
	s.client = &http.Client{
		Jar:     jar,
		Timeout: time.Minute * 1,
	}
	s.login = s.webLogin
	return s.err
}

// Login performs authentication with the UniFi server, and stores the
// http credentials.
func (s *UnifiSession) Login() (string, error) {
	if s.login == nil {
		s.login = func() (string, error) {
			return "", errors.New("uninitialized session")
		}
	}
	return s.login()
}

// ListClients describes the known UniFi clients.
func (s *UnifiSession) ListClients() (string, error) {
	if s.err != nil {
		return "", s.err
	}
	u, err := url.Parse(fmt.Sprintf("%s/api/s/default/rest/user", s.Endpoint))
	if err != nil {
		s.setError(err)
		return "", s.err
	}
	return s.get(u)
}

// Kick disconnects a connected client, identified by MAC address.
func (s *UnifiSession) Kick(mac string) (string, error) {
	return s.macAction("kick-sta", mac)
}

// Block prevents a specific client (identified by MAC) from connecting
// to the UniFi network.
func (s *UnifiSession) Block(mac string) (string, error) {
	return s.macAction("block-sta", mac)
}

// Unblock re-enables a specific client.
func (s *UnifiSession) Unblock(mac string) (string, error) {
	return s.macAction("unblock-sta", mac)
}

func (s *UnifiSession) webLogin() (string, error) {
	if s.err != nil {
		return "", s.err
	}
	u, err := url.Parse(fmt.Sprintf("%s/api/login", s.Endpoint))
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
		s.login = func() (string, error) { return string(respBody), nil }
	}
	return respBody, err
}

func (s *UnifiSession) macAction(action string, mac string) (string, error) {
	if b, err := s.login(); err != nil {
		return b, err
	}
	u, err := url.Parse(fmt.Sprintf("%s/api/s/default/cmd/stamgr", s.Endpoint))
	if err != nil {
		s.setError(err)
		return "", s.err
	}
	r := bytes.NewBufferString(fmt.Sprintf(`{"cmd":%q,"mac":%q}`, action, mac))
	return s.post(u, r)
}

func (s *UnifiSession) get(u *url.URL) (string, error) {
	return s.verb("GET", u, nil)
}

func (s *UnifiSession) post(u *url.URL, body io.Reader) (string, error) {
	return s.verb("POST", u, body)
}

func (s *UnifiSession) verb(verb string, u *url.URL, body io.Reader) (string, error) {
	req, err := http.NewRequest(verb, u.String(), body)
	if err != nil {
		s.setError(err)
		return "", s.err
	}
	req.Header.Set("User-Agent", "unifibot 2.0")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", s.Endpoint)
	for _, cookie := range s.client.Jar.Cookies(u) {
		if cookie.Name == "csrf_token" {
			req.Header.Set("X-CSRF-Token", cookie.Value)
		}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		s.setError(err)
		return "", s.err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.setError(err)
		return "", s.err
	}
	//log.Printf("Debug:\n<<<<<\n%s\n>>>>>\n", respBody)
	if resp.StatusCode < http.StatusOK || http.StatusBadRequest <= resp.StatusCode {
		s.setErrorString(http.StatusText(resp.StatusCode))
	}
	return string(respBody), s.err
}

func (s *UnifiSession) setError(e error) {
	if e == nil {
		return
	}
	if s.err == nil {
		s.err = fmt.Errorf("%w", e)
	} else {
		s.err = fmt.Errorf("%w\n%w", e, s.err)
	}
}

func (s *UnifiSession) setErrorString(e string) {
	if len(e) == 0 {
		return
	}
	if s.err == nil {
		s.err = fmt.Errorf("%s", e)
	} else {
		s.err = fmt.Errorf("%s\n%w", e, s.err)
	}
}
