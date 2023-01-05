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
		Jar:       jar,
		Timeout:   time.Minute * 1,
		Transport: transport.NewLoggingTransport(http.DefaultTransport, transport.LoggingOutput(nil)),
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
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := names[name]; !ok {
				names[name] = &stringset.OrderedStringSet{}
			}

			names[name].Add(string(device.MAC))
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
				names[name] = &stringset.OrderedStringSet{}
			}

			names[name].Add(string(user.MAC))
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
		return nil, fmt.Errorf("unmarshalling clients: %w", err)
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
		return nil, fmt.Errorf("unmarshalling devices: %w", err)
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
		return nil, fmt.Errorf("unmarshalling events: %w", err)
	}

	const maxRecent = 500
	if !all && eresp.Meta.Count > maxRecent {
		return eresp.Data[:maxRecent], nil
	}

	return eresp.Data, nil
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
func (s *Session) Kick(mac MAC) (string, error) { return s.macAction("kick-sta", mac) }

// Block prevents a specific client (identified by MAC) from connecting
// to the UniFi network.
func (s *Session) Block(mac MAC) (string, error) { return s.macAction("block-sta", mac) }

// Unblock re-enables a specific client.
func (s *Session) Unblock(mac MAC) (string, error) { return s.macAction("unblock-sta", mac) }

// Forget removes record of a specific list of MAC addresses.
func (s *Session) Forget(macs []MAC) (string, error) { return s.macsAction("forget-sta", macs) }

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

func (s *Session) ForgetFn(clients []Client, keys map[string]bool) {
	var macs []MAC
	for _, client := range clients {
		display := firstNonEmpty(client.Name, client.Hostname)
		if k, ok := keys[display]; ok && k {
			macs = append(macs, client.MAC)
		}
	}

	res, err := s.Forget(macs)
	if err != nil {
		fmt.Fprintf(s.errWriter, "%s\nerror forgetting: %v\n", res, err)

		return
	}

	fmt.Fprintf(s.outWriter, "%s\n", res)
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

	payload := fmt.Sprintf(`{"username":%q,"password":%q,"strict":"true","remember":"true"}`, s.Username, s.Password)

	respBody, err := s.post(u, bytes.NewBufferString(payload))
	if err == nil {
		s.login = func() (string, error) { return respBody, nil }
	}

	return respBody, err
}

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

func (s *Session) macAction(action string, mac MAC) (string, error) {
	payload := fmt.Sprintf(`{"cmd":%q,"mac":%q}`, action, mac)

	return s.action(http.MethodPost, "/cmd/stamgr", bytes.NewBufferString(payload))
}

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
