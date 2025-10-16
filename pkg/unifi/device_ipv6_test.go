package unifi

import "testing"

func TestDeviceGetIPv6DelegatedPrefix(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		want   string
	}{
		{
			name: "single network /56",
			device: Device{
				NetworkTable: []NetworkTable{
					{
						IPv6Subnets: []string{"2605:59c0:40a6:e800::1/64"},
					},
				},
			},
			want: "2605:59c0:40a6:e800::/56",
		},
		{
			name: "multiple networks /56",
			device: Device{
				NetworkTable: []NetworkTable{
					{
						IPv6Subnets: []string{"2605:59c0:40a6:e800::1/64"},
					},
					{
						IPv6Subnets: []string{"2605:59c0:40a6:e801::1/64"},
					},
				},
			},
			want: "2605:59c0:40a6:e800::/56",
		},
		{
			name:   "no networks",
			device: Device{},
			want:   "",
		},
		{
			name: "network with no IPv6",
			device: Device{
				NetworkTable: []NetworkTable{
					{
						IPv6Subnets: []string{},
					},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.GetIPv6DelegatedPrefix()
			if got != tt.want {
				t.Errorf("GetIPv6DelegatedPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}
