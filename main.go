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

// nolint:funlen
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
		UseJSON:  useJSON,
	}

	if err := ses.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error:\n%v\n", err)

		return
	}

	if msg, err := ses.Login(); err != nil {
		fmt.Fprintf(os.Stderr, "Login Error: (%s)\n%v\n", msg, err)

		return
	}

	sorter := unifi.ClientDefault
	fetch := ses.ListClients

	if historical || strings.Contains(invocation, "block") {
		sorter = unifi.ClientHistorical
		fetch = ses.ListUsers
	}

	u, err := fetch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error:\n%v\n", err)

		return
	}

	var users unifi.Response
	if err := json.Unmarshal([]byte(u), &users); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n%v\n", u, err)

		return
	}

	target := map[string]bool{}
	for _, name := range os.Args[1:] {
		target[name] = true
	}

	functions := map[string]func([]unifi.Client, map[string]bool){
		"list":    ses.ListFn,
		"block":   ses.BlockFn,
		"unblock": ses.UnblockFn,
	}

	fn, ok := functions[invocation]
	if !ok {
		fn = ses.ListFn
	}

	sorter.Sort(users.Data)

	fn(users.Data, target)
}
