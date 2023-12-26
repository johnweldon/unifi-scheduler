package unifi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/jw4/x/stringset"
	"github.com/jw4/x/transport"
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

	nonUDMPro bool
	site      string

	outWriter io.Writer
	errWriter io.Writer
	dbgWriter io.Writer
}

// Option describes an option parameter.
type Option func(*Session)

func WithOut(o io.Writer) Option { return func(s *Session) { s.outWriter = o } }
func WithErr(e io.Writer) Option { return func(s *Session) { s.errWriter = e } }
func WithDbg(d io.Writer) Option { return func(s *Session) { s.dbgWriter = d } }

// Initialize prepares the session for use.
func (s *Session) Initialize(options ...Option) error {
	if s == nil {
		return ErrNilSession
	}

	s.outWriter = os.Stdout
	s.errWriter = os.Stderr

	for _, option := range options {
		option(s)
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
		Jar:       jar,
		Timeout:   time.Minute * 1,
		Transport: transport.NewLoggingTransport(http.DefaultTransport, transport.LoggingOutput(s.dbgWriter)),
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

// GetDevices looks up and returns known Devices.
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

// GetClients returns a list of connected Clients.
func (s *Session) GetClients(filters ...ClientFilter) ([]Client, error) {
	return s.getClients(false, filters...)
}

// GetAllClients returns all known Clients.
func (s *Session) GetAllClients(filters ...ClientFilter) ([]Client, error) {
	return s.getClients(true, filters...)
}

// GetAllEvents returns all events.
func (s *Session) GetAllEvents() ([]Event, error) {
	return s.getEvents(true)
}

// GetRecentEvents returns a list of "recent" events.
func (s *Session) GetRecentEvents() ([]Event, error) {
	return s.getEvents(false)
}

// GetMACs returns all known MAC addresses, and the associated names.
func (s *Session) GetMACs() (map[MAC][]string, error) {
	var (
		macs = map[MAC]*stringset.OrderedStringSet{}

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
			device.MAC.String(),
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := macs[device.MAC]; !ok {
				macs[device.MAC] = &stringset.OrderedStringSet{}
			}

			macs[device.MAC].Add(name)
		}
	}

	if users, err = s.GetAllClients(); err != nil {
		return nil, fmt.Errorf("getting users: %w", err)
	}

	for _, user := range users {
		for _, name := range []string{
			user.Name,
			user.Hostname,
			user.MAC.String(),
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := macs[user.MAC]; !ok {
				macs[user.MAC] = &stringset.OrderedStringSet{}
			}

			macs[user.MAC].Add(name)
		}
	}

	ret := map[MAC][]string{}
	for mac, m := range macs {
		ret[mac] = m.Values()
	}

	return ret, nil
}

// GetNames returns all known names, and the associated MAC addresses.
func (s *Session) GetNames() (map[string][]MAC, error) { // nolint:funlen
	var (
		names = map[string]*stringset.OrderedStringSet{}

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
			string(device.MAC),
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := names[name]; !ok {
				names[name] = &stringset.OrderedStringSet{}
			}

			names[name].Add(device.MAC.String())
		}
	}

	if clients, err = s.GetClients(); err != nil {
		return nil, fmt.Errorf("getting users: %w", err)
	}

	if users, err = s.GetAllClients(); err != nil {
		return nil, fmt.Errorf("getting users: %w", err)
	}

	for _, user := range append(clients, users...) {
		for _, name := range []string{
			user.Name,
			user.Hostname,
			user.DeviceName,
			string(user.IP),
			string(user.FixedIP),
			user.MAC.String(),
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := names[name]; !ok {
				names[name] = &stringset.OrderedStringSet{}
			}

			names[name].Add(user.MAC.String())
		}
	}

	ret := map[string][]MAC{}
	for name, m := range names {
		vals := m.Values()
		macs := make([]MAC, len(vals))
		for ix, val := range vals {
			macs[ix] = MAC(val)
		}

		ret[name] = macs
	}

	return ret, nil
}

func (s *Session) GetMACsBy(ids ...string) ([]MAC, error) {
	var (
		err     error
		allMACs []MAC
		names   map[string][]MAC
	)

	if names, err = s.GetNames(); err != nil {
		return nil, err
	}

	for _, id := range ids {
		if macs, ok := names[id]; ok {
			allMACs = append(allMACs, macs...)
		}
	}

	return allMACs, nil
}

// Raw executes arbitrary endpoints.
func (s *Session) Raw(method, path string, body io.Reader) (string, error) {
	return s.action(method, path, body)
}

// ListEvents describes the latest events.
func (s *Session) ListEvents() (string, error) { return s.action(http.MethodGet, "/stat/event", nil) }

// ListAllEvents describes all events.
func (s *Session) ListAllEvents() (string, error) {
	return s.action(http.MethodGet, "/rest/event", nil)
}

// ListUsers describes the known UniFi clients.
func (s *Session) ListUsers() (string, error) { return s.action(http.MethodGet, "/rest/user", nil) }

// GetUser returns user info.
func (s *Session) GetUser(id string) (string, error) {
	return s.action(http.MethodGet, "/rest/user/"+id, nil)
}

// GetUserByMAC returns user info.
func (s *Session) GetUserByMAC(mac string) (string, error) {
	return s.action(http.MethodGet, "/rest/user/?mac="+mac, nil)
}

// SetUserDetails configures a friendly name and static ip assignation
// for a given MAC address.
func (s *Session) SetUserDetails(mac, name, ip string) (string, error) {
	user, err := s.getUserByMac(mac)
	if err != nil {
		return "", err
	}

	return s.setUserDetails(user.ID, name, ip)
}

// ListClients describes currently connected clients.
func (s *Session) ListClients() (string, error) { return s.action(http.MethodGet, "/stat/sta", nil) }

// ListDevices describes currently connected clients.
func (s *Session) ListDevices() (string, error) { return s.action(http.MethodGet, "/stat/device", nil) }

// Kick disconnects a connected client, identified by MAC address.
func (s *Session) Kick(macs ...MAC) (string, error) { return s.macsAction("kick-sta", macs) }

// Block prevents a specific client (identified by MAC) from connecting
// to the UniFi network.
func (s *Session) Block(macs ...MAC) (string, error) { return s.macsAction("block-sta", macs) }

// Unblock re-enables a specific client.
func (s *Session) Unblock(macs ...MAC) (string, error) { return s.macsAction("unblock-sta", macs) }

// Forget removes record of a specific list of MAC addresses.
func (s *Session) Forget(macs ...MAC) (string, error) { return s.macsAction("forget-sta", macs) }

// KickFn uses Clients to find MAC addresses to Kick.
func (s *Session) KickFn(clients []Client, keys map[string]bool) {
	s.clientsFn(s.Kick, keys, clients...)
}

// BlockFn uses Clients to find MAC addresses to Block.
func (s *Session) BlockFn(clients []Client, keys map[string]bool) {
	s.clientsFn(s.Block, keys, clients...)
}

// UnblockFn uses Clients to find MAC addresses to Unblock.
func (s *Session) UnblockFn(clients []Client, keys map[string]bool) {
	s.clientsFn(s.Unblock, keys, clients...)
}

// ForgetFn uses Clients to find MAC addresses to Forget.
func (s *Session) ForgetFn(clients []Client, keys map[string]bool) {
	s.clientsFn(s.Forget, keys, clients...)
}

// getUserByMac looks up a Client by the MAC address.
func (s *Session) getUserByMac(mac string) (*Client, error) {
	var (
		err  error
		data string
		resp ClientResponse
	)

	if data, err = s.GetUserByMAC(mac); err != nil {
		return nil, fmt.Errorf("retrieving user by mac: %w", err)
	}

	if err = json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling user: %w", err)
	}

	if len(resp.Data) < 1 {
		return nil, fmt.Errorf("zero results: %s", data)
	}

	return &resp.Data[0], nil
}

type ClientFilter func(Client) bool

func Not(filter ClientFilter) ClientFilter { return func(c Client) bool { return !filter(c) } }

func Blocked(c Client) bool    { return c.IsBlocked }
func Authorized(c Client) bool { return c.IsAuthorized }
func Guest(c Client) bool      { return c.IsGuest }
func Wired(c Client) bool      { return c.IsWired }

func passAll(client Client, filters ...ClientFilter) bool {
	for _, filter := range filters {
		if !filter(client) {
			return false
		}
	}

	return true
}

// getClients returns a list of clients.  If all is false, only the active
// clients will be returned, otherwise all the known clients will be returned.
func (s *Session) getClients(all bool, filters ...ClientFilter) ([]Client, error) {
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
		return nil, fmt.Errorf("unmarshalling clients: %w", err)
	}

	for _, client := range cresp.Data {
		if dev, ok := devices[client.UpstreamMAC()]; ok {
			client.UpstreamName = dev.Name
		}

		if passAll(client, filters...) {
			clients = append(clients, client)
		}
	}

	sorter.Sort(clients)

	return clients, nil
}

// getDevices returns all known devices mapped by name.
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
		return nil, fmt.Errorf("unmarshalling devices: %w", err)
	}

	for _, device := range dresp.Data {
		devices[device.MAC.String()] = device
	}

	return devices, nil
}

// getEvents returns a list of events. If all is true, then all known events
// will be returned, otherwise only the most recent ones will be returned.
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
		return nil, fmt.Errorf("unmarshalling events: %w", err)
	}

	events := eresp.Data

	DefaultEventSort.Sort(events)

	return events, nil
}

// webLogin performs the authentication for this session.
func (s *Session) webLogin() (string, error) {
	if s.err != nil {
		return "", s.err
	}

	u, err := url.Parse(fmt.Sprintf("%s/api/auth/login", s.Endpoint))
	if err != nil {
		s.setError(err)

		return "", s.err
	}

	payload := fmt.Sprintf(`{"username":%q,"password":%q,"strict":"true","remember":"true"}`, s.Username, s.Password)

	respBody, err := s.post(u, bytes.NewBufferString(payload))
	if err == nil {
		s.login = func() (string, error) { return respBody, nil }
	}

	return respBody, err
}

// buildURL generates the endpoint URL relevant to the configured
// version of UniFi.
func (s *Session) buildURL(path string) (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}

	pathPrefix := "/proxy/network"
	if s.nonUDMPro {
		pathPrefix = ""
	}

	site := "default"
	if len(s.site) > 0 {
		site = s.site
	}

	return url.Parse(fmt.Sprintf("%s%s/api/s/%s%s", s.Endpoint, pathPrefix, site, path))
}

// macAction applies an action to a single MAC.
func (s *Session) macAction(action string, mac MAC) (string, error) {
	payload := fmt.Sprintf(`{"cmd":%q,"mac":%q}`, action, mac)

	return s.action(http.MethodPost, "/cmd/stamgr", bytes.NewBufferString(payload))
}

// macsAction applies a function to multiple MACs.
func (s *Session) macsAction(action string, macs []MAC) (string, error) {
	if len(macs) == 0 {
		return "", nil
	}

	var allmacs []string
	for _, mac := range macs {
		allmacs = append(allmacs, fmt.Sprintf("%q", mac))
	}

	payload := fmt.Sprintf(`{"cmd":%q,"macs":[%s]}`, action, strings.Join(allmacs, ","))

	return s.action(http.MethodPost, "/cmd/stamgr", bytes.NewBufferString(payload))
}

func (s *Session) clientsFn(action func(...MAC) (string, error), keys map[string]bool, clients ...Client) {
	var macs []MAC
	for _, client := range clients {
		display := firstNonEmpty(client.Name, client.Hostname)
		if k, ok := keys[display]; ok && k {
			macs = append(macs, client.MAC)
		}
	}

	res, err := action(macs...)
	if err != nil {
		fmt.Fprintf(s.errWriter, "%s\nerror: %v\n", res, err)

		return
	}

	fmt.Fprintf(s.outWriter, "%s\n", res)
}

func (s *Session) setUserDetails(id, name, ip string) (string, error) {
	if len(id) == 0 {
		return "", fmt.Errorf("missing user id")
	}

	tmpl, err := template.New("").Parse(`{ {{- /**/ -}}
  "local_dns_record_enabled":false,{{- /**/ -}}
  "local_dns_record":"",{{- /**/ -}}
  "name":"{{ with .Name }}{{ . }}{{ end }}",{{- /**/ -}}
  "usergroup_id":"{{ with .UsergroupID }}{{ . }}{{ end }}",{{- /**/ -}}
  "use_fixedip":{{ with .IP }}true{{ else }}false{{ end }},{{- /**/ -}}
  "network_id":"{{ with .NetworkID }}{{ . }}{{ end }}",{{- /**/ -}}
  "fixed_ip":"{{ with .IP }}{{ . }}{{ end }}"{{- /**/ -}}
}`)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, map[string]string{"Name": name, "IP": ip, "NetworkID": "5c82f1ce2679fb00116fb58e"}); err != nil {
		return "", err
	}

	return s.action(http.MethodPut, "/rest/user/"+id, &buf)
}

func (s *Session) action(method, path string, body io.Reader) (string, error) {
	if s.err != nil {
		return "", s.err
	}

	u, err := s.buildURL(path)
	if err != nil {
		s.setError(err)

		return "", s.err
	}

	switch method {
	case http.MethodGet:
		return s.get(u)
	case http.MethodPost:
		return s.post(u, body)
	case http.MethodPut:
		return s.put(u, body)
	default:
		return "", fmt.Errorf("unconfigured method: %q", method)
	}
}

func (s *Session) get(u fmt.Stringer) (string, error) {
	return s.verb("GET", u, nil)
}

func (s *Session) post(u fmt.Stringer, body io.Reader) (string, error) {
	return s.verb("POST", u, body)
}

func (s *Session) put(u fmt.Stringer, body io.Reader) (string, error) {
	return s.verb("PUT", u, body)
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.setError(err)

		return "", s.err
	}

	if resp.StatusCode < http.StatusOK || http.StatusBadRequest <= resp.StatusCode {
		if resp.StatusCode == http.StatusUnauthorized {
			fmt.Fprintf(s.errWriter, "\nlogged out; re-authenticating\n")
			s.login = s.webLogin
			if r, err := s.login(); err != nil {
				s.setError(err)
				return r, fmt.Errorf("login attempt failed: %w", err)
			}
		} else {
			return string(respBody), fmt.Errorf("http error: %s", resp.Status)
		}
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
