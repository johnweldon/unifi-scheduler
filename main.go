package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/johnweldon/unifi-scheduler/unifi"
)

// nolint: gochecknoglobals
var (
	useJSON    bool
	historical bool
)

// nolint
func init() {
	flag.BoolVar(&useJSON, "json", useJSON, "output as json")
	flag.BoolVar(&historical, "historical", historical, "use historical data")
}

func main() {
	flag.Parse()

	_, invocation := filepath.Split(os.Args[0])

	username := os.Getenv("UNIFI_USERNAME")
	password := os.Getenv("UNIFI_PASSWORD")
	endpoint := os.Getenv("UNIFI_ENDPOINT")
	ses := &unifi.Session{
		Username: username,
		Password: password,
		Endpoint: endpoint,
	}

	if err := ses.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error:\n%v\n", err)

		return
	}

	if msg, err := ses.Login(); err != nil {
		fmt.Fprintf(os.Stderr, "Login Error: (%s)\n%v\n", msg, err)

		return
	}

	devices, err := getDevices(ses)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error:\n%v\n", err)

		return
	}

	clients, err := getClients(ses, invocation)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error:\n%v\n", err)

		return
	}

	target := map[string]bool{}
	for _, name := range os.Args[1:] {
		target[name] = true
	}

	functions := map[string]func([]unifi.Client, map[string]bool){
		"list":    listFunction(devices, useJSON),
		"block":   ses.BlockFn,
		"unblock": ses.UnblockFn,
	}

	fn, ok := functions[invocation]
	if !ok {
		fn = functions["list"]
	}

	fn(clients, target)
}

func getClients(ses *unifi.Session, mode string) ([]unifi.Client, error) {
	sorter := unifi.ClientDefault
	fetch := ses.ListClients

	if historical || strings.Contains(mode, "block") {
		sorter = unifi.ClientHistorical
		fetch = ses.ListUsers
	}

	u, err := fetch()
	if err != nil {
		return nil, err
	}

	var users unifi.ClientResponse
	if err := json.Unmarshal([]byte(u), &users); err != nil {
		return nil, err // nolint: wrapcheck
	}

	sorter.Sort(users.Data)

	return users.Data, nil
}

func getDevices(ses *unifi.Session) (map[string]unifi.Device, error) {
	d, err := ses.ListDevices()
	if err != nil {
		return nil, err // nolint: wrapcheck
	}

	var devices unifi.DeviceResponse
	if err := json.Unmarshal([]byte(d), &devices); err != nil {
		return nil, err // nolint: wrapcheck
	}

	dmap := map[string]unifi.Device{}
	for _, device := range devices.Data {
		dmap[string(device.MAC)] = device
	}

	return dmap, nil
}

func listFunction(devices map[string]unifi.Device, useJSON bool) func([]unifi.Client, map[string]bool) {
	return func(clients []unifi.Client, _ map[string]bool) {
		if useJSON {
			if err := json.NewEncoder(os.Stdout).Encode(clients); err != nil {
				fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
			}

			return
		}

		for _, client := range clients {
			ap := ""
			if dev, ok := devices[client.AccessPointMAC]; ok {
				ap = fmt.Sprintf(" %s", dev.Name)
			}

			fmt.Fprintf(os.Stdout, "%-95s%s\n", client.String(), ap)
		}

		fmt.Fprintf(os.Stdout, "%d clients\n", len(clients))
	}
}
