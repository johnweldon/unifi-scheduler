package unifi

import (
	"bytes"
	"context"
	"encoding/json"
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

// Session wraps metadata to manage session state.
type Session struct {
	Endpoint string
	Username string
	Password string

	csrf   string
	client *http.Client
	login  func() (string, error)
	err    error

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

func (s *Session) GetDevices() ([]Device, error) {
	var (
		devices []Device
		dmap    map[string]Device

		err error
	)

	if dmap, err = s.getDevices(); err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}

	for _, device := range dmap {
		devices = append(devices, device)
	}

	DeviceDefault.Sort(devices)

	return devices, nil
}

func (s *Session) GetClients() ([]Client, error) {
	return s.getClients(false)
}

func (s *Session) GetUsers() ([]Client, error) {
	return s.getClients(true)
}

func (s *Session) GetAllEvents() ([]Event, error) {
	return s.getEvents(true)
}

func (s *Session) GetRecentEvents() ([]Event, error) {
	return s.getEvents(false)
}

func (s *Session) GetMACs() (map[MAC][]string, error) {
	var (
		macs = map[MAC]map[string]string{}

		devices []Device
		users   []Client

		err error
	)

	if devices, err = s.GetDevices(); err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}

	for _, device := range devices {
		for _, name := range []string{
			device.Name,
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := macs[device.MAC]; !ok {
				macs[device.MAC] = map[string]string{}
			}

			macs[device.MAC][name] = device.ID
		}
	}

	if users, err = s.GetUsers(); err != nil {
		return nil, fmt.Errorf("getting users: %w", err)
	}

	for _, user := range users {
		for _, name := range []string{
			user.Name,
			user.Hostname,
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := macs[user.MAC]; !ok {
				macs[user.MAC] = map[string]string{}
			}

			macs[user.MAC][name] = user.ID
		}
	}

	ret := map[MAC][]string{}
	for mac, m := range macs {
		for name := range m {
			ret[mac] = append(ret[mac], name)
		}

		sort.Stable(sort.Reverse(sort.StringSlice(ret[mac])))
	}

	return ret, nil
}

func (s *Session) GetNames() (map[string][]MAC, error) { // nolint:funlen
	var (
		names = map[string]map[MAC]string{}

		devices []Device
		clients []Client
		users   []Client

		err error
	)

	if devices, err = s.GetDevices(); err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}

	for _, device := range devices {
		for _, name := range []string{
			device.Name,
			string(device.IP),
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := names[name]; !ok {
				names[name] = map[MAC]string{}
			}

			names[name][device.MAC] = device.ID
		}
	}

	if clients, err = s.GetClients(); err != nil {
		return nil, fmt.Errorf("getting users: %w", err)
	}

	if users, err = s.GetUsers(); err != nil {
		return nil, fmt.Errorf("getting users: %w", err)
	}

	for _, user := range append(clients, users...) {
		for _, name := range []string{
			user.Name,
			user.Hostname,
			user.DeviceName,
			string(user.IP),
			string(user.FixedIP),
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := names[name]; !ok {
				names[name] = map[MAC]string{}
			}

			names[name][user.MAC] = user.ID
		}
	}

	ret := map[string][]MAC{}
	for name, m := range names {
		for mac := range m {
			ret[name] = append(ret[name], mac)
		}
	}

	return ret, nil
}

func (s *Session) getClients(all bool) ([]Client, error) {
	var (
		devices map[string]Device

		clientsJSON string
		clients     []Client
		cresp       ClientResponse

		err error
	)

	sorter := ClientDefault
	fetch := s.ListClients

	if all {
		sorter = ClientHistorical
		fetch = s.ListUsers
	}

	if devices, err = s.getDevices(); err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}

	if clientsJSON, err = fetch(); err != nil {
		return nil, fmt.Errorf("listing clients: %w", err)
	}

	if err = json.Unmarshal([]byte(clientsJSON), &cresp); err != nil {
		return nil, fmt.Errorf("unmarshaling clients: %w", err)
	}

	for _, client := range cresp.Data {
		if dev, ok := devices[client.UpstreamMAC()]; ok {
			client.UpstreamName = dev.Name
		}

		clients = append(clients, client)
	}

	sorter.Sort(clients)

	return clients, nil
}

func (s *Session) getDevices() (map[string]Device, error) {
	var (
		devicesJSON string
		devices     = map[string]Device{}
		dresp       DeviceResponse

		err error
	)

	if devicesJSON, err = s.ListDevices(); err != nil {
		return nil, fmt.Errorf("listing devices: %w", err)
	}

	if err = json.Unmarshal([]byte(devicesJSON), &dresp); err != nil {
		return nil, fmt.Errorf("unmarshaling devices: %w", err)
	}

	for _, device := range dresp.Data {
		devices[string(device.MAC)] = device
	}

	return devices, nil
}

func (s *Session) getEvents(all bool) ([]Event, error) {
	var (
		eventsJSON string
		eresp      EventResponse

		err error
	)

	fetch := s.ListEvents
	if all {
		fetch = s.ListAllEvents
	}

	if eventsJSON, err = fetch(); err != nil {
		return nil, fmt.Errorf("fetching events: %w", err)
	}

	if err = json.Unmarshal([]byte(eventsJSON), &eresp); err != nil {
		return nil, fmt.Errorf("unmarshaling events: %w", err)
	}

	return eresp.Data, nil
}

// ListEvents describes the latest events.
func (s *Session) ListEvents() (string, error) {
	if s.err != nil {
		return "", s.err
	}

	u, err := url.Parse(fmt.Sprintf("%s/proxy/network/api/s/default/stat/event", s.Endpoint))
	if err != nil {
		s.setError(err)

		return "", s.err
	}

	return s.get(u)
}

// ListAllEvents describes all events.
func (s *Session) ListAllEvents() (string, error) {
	if s.err != nil {
		return "", s.err
	}

	u, err := url.Parse(fmt.Sprintf("%s/proxy/network/api/s/default/rest/event", s.Endpoint))
	if err != nil {
		s.setError(err)

		return "", s.err
	}

	return s.get(u)
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
func (s *Session) Kick(mac MAC) (string, error) {
	return s.macAction("kick-sta", mac)
}

// Block prevents a specific client (identified by MAC) from connecting
// to the UniFi network.
func (s *Session) Block(mac MAC) (string, error) {
	return s.macAction("block-sta", mac)
}

// Unblock re-enables a specific client.
func (s *Session) Unblock(mac MAC) (string, error) {
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

func (s *Session) macAction(action, mac MAC) (string, error) {
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
