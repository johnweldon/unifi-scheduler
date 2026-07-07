package unifi

import (
	"encoding/json"
	"testing"
)

// TestNetworkTable_VLANUnmarshal verifies network_table.vlan unmarshals from
// both numeric (UniFi Network 9+/10+) and string (older firmware) JSON values.
func TestNetworkTable_VLANUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		json string
		want Number
	}{
		{
			name: "numeric vlan (current firmware)",
			json: `{"data":[{"mac":"11:22:33:44:55:66","network_table":[{"name":"iot","vlan":30}]}]}`,
			want: 30,
		},
		{
			name: "string vlan (legacy firmware)",
			json: `{"data":[{"mac":"11:22:33:44:55:66","network_table":[{"name":"iot","vlan":"30"}]}]}`,
			want: 30,
		},
		{
			name: "empty string vlan",
			json: `{"data":[{"mac":"11:22:33:44:55:66","network_table":[{"name":"lan","vlan":""}]}]}`,
			want: 0,
		},
		{
			name: "absent vlan",
			json: `{"data":[{"mac":"11:22:33:44:55:66","network_table":[{"name":"lan"}]}]}`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp DeviceResponse
			if err := json.Unmarshal([]byte(tt.json), &resp); err != nil {
				t.Fatalf("unmarshalling device response: %v", err)
			}

			if len(resp.Data) != 1 || len(resp.Data[0].NetworkTable) != 1 {
				t.Fatalf("unexpected response shape: %+v", resp)
			}

			if got := resp.Data[0].NetworkTable[0].VLAN; got != tt.want {
				t.Errorf("expected vlan %d, got %v", tt.want, got)
			}
		})
	}
}

// TestNumber_UnmarshalNull verifies JSON null does not fail Number fields.
func TestNumber_UnmarshalNull(t *testing.T) {
	var n Number
	if err := json.Unmarshal([]byte(`null`), &n); err != nil {
		t.Fatalf("unmarshalling null: %v", err)
	}

	if n != 0 {
		t.Errorf("expected 0 for null, got %v", n)
	}
}
