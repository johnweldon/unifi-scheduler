package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/johnweldon/unifi-scheduler/unifi"
)

var useJSON bool // nolint

// nolint
func init() {
	flag.BoolVar(&useJSON, "json", useJSON, "output as json")
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
		log.Printf("Error:\n%v", err)

		return
	}

	fmt.Fprintf(os.Stderr, "Initialized...\n")

	if msg, err := ses.Login(); err != nil {
		log.Printf("Login Error: (%s)\n%v", msg, err)

		return
	}

	fmt.Fprintf(os.Stderr, "Logged in...\n")

	u, err := ses.ListClients()
	if err != nil {
		log.Printf("Error:\n%v", err)

		return
	}

	var users unifi.Response
	if err := json.Unmarshal([]byte(u), &users); err != nil {
		log.Printf("%s\n%v", u, err)

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

	fn(users.Data, target)
}
