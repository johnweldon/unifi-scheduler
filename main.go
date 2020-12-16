package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var useJSON bool

func init() {
	flag.BoolVar(&useJSON, "json", useJSON, "output as json")
}

func main() {
	flag.Parse()

	_, invocation := filepath.Split(os.Args[0])

	username := os.Getenv("UNIFI_USERNAME")
	password := os.Getenv("UNIFI_PASSWORD")
	endpoint := os.Getenv("UNIFI_ENDPOINT")
	ses := &UnifiSession{
		Username: username,
		Password: password,
		Endpoint: endpoint,
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

	users := &Response{}
	if err := json.Unmarshal([]byte(u), users); err != nil {
		log.Printf("%s\n%v", u, err)
		return
	}

	target := map[string]bool{}
	for _, name := range os.Args[1:] {
		target[name] = true
	}

	functions := map[string]func([]Client, map[string]bool){
		"list":    listFn,
		"block":   ses.blockFn,
		"unblock": ses.unblockFn,
	}

	fn, ok := functions[invocation]
	if !ok {
		fn = listFn
	}

	fn(users.Data, target)
}

func listFn(clients []Client, keys map[string]bool) {
	if useJSON {
		if err := json.NewEncoder(os.Stdout).Encode(clients); err != nil {
			log.Printf("error encoding JSON: %v", err)
		}
		return
	}

	for _, client := range clients {
		display := firstNonEmpty(client.Name, client.Hostname, "-")
		ip := firstNonEmpty(client.FixedIP, client.IP)
		log.Printf("%30s %s %s", display, client.MAC, ip)
	}
}

func (s *UnifiSession) blockFn(clients []Client, keys map[string]bool) {
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

func (s *UnifiSession) unblockFn(clients []Client, keys map[string]bool) {
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

func firstNonEmpty(s ...string) string {
	for _, candidate := range s {
		if len(candidate) > 0 {
			return candidate
		}
	}

	return ""
}
