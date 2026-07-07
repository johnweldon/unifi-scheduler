package unifi

import "testing"

func TestSystemLogEntry_ToEvent_PopulatesClientFields(t *testing.T) {
	entry := SystemLogEntry{
		ID:         "6a4c474a964f169c297b71bd",
		Key:        "CLIENT_CONNECTED_WIRELESS_2",
		Category:   "CLIENT_DEVICES",
		MessageRaw: "{CLIENT} connected to {WLAN} on {DEVICE}.",
		Timestamp:  1783383882512,
		Parameters: map[string]SystemLogParam{
			"CLIENT": {
				ID:       "22:e2:16:26:09:80",
				Name:     "simeon-chromecast",
				Hostname: "customer.tmpeazx1.isp.starlink.com",
				IP:       "10.36.20.83",
			},
			"DEVICE": {
				ID:    "74:ac:b9:5a:a7:80",
				Name:  "unifi-ap-casita",
				IP:    "10.36.1.40",
				Model: "UAP-nanoHD",
			},
			"WLAN": {ID: "5c8329445c5bb60031432f29", Name: "weldon"},
		},
	}

	evt := entry.ToEvent()

	if evt.Name != "simeon-chromecast" {
		t.Errorf("Name = %q, want simeon-chromecast", evt.Name)
	}
	if evt.Client != MAC("22:e2:16:26:09:80") {
		t.Errorf("Client = %q, want 22:e2:16:26:09:80", evt.Client)
	}
	if evt.Device != MAC("74:ac:b9:5a:a7:80") {
		t.Errorf("Device = %q, want 74:ac:b9:5a:a7:80", evt.Device)
	}
	if evt.DeviceName != "unifi-ap-casita" {
		t.Errorf("DeviceName = %q, want unifi-ap-casita", evt.DeviceName)
	}
	if evt.ESSID != "weldon" {
		t.Errorf("ESSID = %q, want weldon", evt.ESSID)
	}
}

func TestSystemLogEntry_ToEvent_PopulatesRoamFromTo(t *testing.T) {
	entry := SystemLogEntry{
		Key:        "CLIENT_ROAMED_2",
		Category:   "CLIENT_DEVICES",
		MessageRaw: "{CLIENT} roamed from {DEVICE_FROM} to {DEVICE_TO}.",
		Timestamp:  1783383882512,
		Parameters: map[string]SystemLogParam{
			"CLIENT":      {ID: "2e:1a:fb:0e:20:d2", Name: "john-pixel-9"},
			"DEVICE_FROM": {ID: "70:a7:41:c5:1b:c8", Name: "unifi-ap-main-1"},
			"DEVICE_TO":   {ID: "f0:9f:c2:70:65:9a", Name: "unifi-ap-indoor"},
		},
	}

	evt := entry.ToEvent()

	if evt.DeviceFromName != "unifi-ap-main-1" {
		t.Errorf("DeviceFromName = %q, want unifi-ap-main-1", evt.DeviceFromName)
	}
	if evt.DeviceToName != "unifi-ap-indoor" {
		t.Errorf("DeviceToName = %q, want unifi-ap-indoor", evt.DeviceToName)
	}
}

func TestSystemLogEntry_ToEvent_PrefersDeviceWithPort(t *testing.T) {
	entry := SystemLogEntry{
		Key:        "CLIENT_DISCONNECTED_WIRED_2",
		Category:   "CLIENT_DEVICES",
		MessageRaw: "{CLIENT} disconnected from {NETWORK} on {DEVICE_WITH_PORT}.",
		Timestamp:  1783383882512,
		Parameters: map[string]SystemLogParam{
			"CLIENT":           {ID: "bc:24:11:a8:22:1b", Name: "WIN-NSC8FGN021O 22:1b"},
			"DEVICE":           {ID: "18:e8:29:29:0b:2e", Name: "unifi-switch-1"},
			"DEVICE_WITH_PORT": {ID: "18:e8:29:29:0b:2e", Name: "unifi-switch-1 Port 3"},
		},
	}

	evt := entry.ToEvent()

	if evt.DeviceName != "unifi-switch-1 Port 3" {
		t.Errorf("DeviceName = %q, want unifi-switch-1 Port 3", evt.DeviceName)
	}
	if evt.Device != MAC("18:e8:29:29:0b:2e") {
		t.Errorf("Device = %q, want 18:e8:29:29:0b:2e", evt.Device)
	}
}

func TestSystemLogEntry_ToEvent_PopulatesAdmin(t *testing.T) {
	entry := SystemLogEntry{
		Key:        "ADMIN_ACCESS",
		Category:   "AUDIT",
		MessageRaw: "{ADMIN} accessed UniFi Network using the {PLATFORM}.",
		Timestamp:  1783378233108,
		Parameters: map[string]SystemLogParam{
			"ADMIN": {ID: "64b081064ddfff787d033d53", Name: "John Weldon"},
		},
	}

	evt := entry.ToEvent()

	if evt.Admin != "John Weldon" {
		t.Errorf("Admin = %q, want John Weldon", evt.Admin)
	}
}
