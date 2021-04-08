package unifi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"
)

// Session wraps metadata to manage session state.
type Session struct {
	Endpoint string
	Username string
	Password string
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

// ListDevices describes currently connected clients.
func (s *Session) ListDevices() (string, error) {
	if s.err != nil {
		return "", s.err
	}

	u, err := url.Parse(fmt.Sprintf("%s/proxy/network/api/s/default/stat/device", s.Endpoint))
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

func (s *Session) macAction(action, mac string) (string, error) {
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
