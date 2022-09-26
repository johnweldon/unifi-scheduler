package main

import "github.com/johnweldon/unifi-scheduler/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
