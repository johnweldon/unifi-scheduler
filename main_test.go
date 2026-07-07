package main

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	cases := []struct {
		name     string
		ldflags  string
		info     *debug.BuildInfo
		ok       bool
		expected string
	}{
		{
			name:     "ldflags version wins",
			ldflags:  "0.4.5",
			info:     &debug.BuildInfo{Main: debug.Module{Version: "v0.9.9"}},
			ok:       true,
			expected: "0.4.5",
		},
		{
			name:     "go install falls back to module version",
			ldflags:  "dev",
			info:     &debug.BuildInfo{Main: debug.Module{Version: "v0.4.5"}},
			ok:       true,
			expected: "v0.4.5",
		},
		{
			name:     "devel build info stays dev",
			ldflags:  "dev",
			info:     &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			ok:       true,
			expected: "dev",
		},
		{
			name:     "missing build info stays dev",
			ldflags:  "dev",
			info:     nil,
			ok:       false,
			expected: "dev",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion(tc.ldflags, tc.info, tc.ok); got != tc.expected {
				t.Errorf("resolveVersion(%q, ...) = %q, want %q", tc.ldflags, got, tc.expected)
			}
		})
	}
}
