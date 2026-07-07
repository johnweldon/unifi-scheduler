package main

import (
	"runtime/debug"

	"github.com/johnweldon/unifi-scheduler/cmd"
)

// version is stamped by goreleaser via ldflags; go-install builds keep the
// default and fall back to the module version from build info.
var version = "dev"

func main() {
	info, ok := debug.ReadBuildInfo()
	cmd.Execute(resolveVersion(version, info, ok))
}

func resolveVersion(ldflagsVersion string, info *debug.BuildInfo, ok bool) string {
	if ldflagsVersion != "dev" {
		return ldflagsVersion
	}

	if ok && info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return ldflagsVersion
}
