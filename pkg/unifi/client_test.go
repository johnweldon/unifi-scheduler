package unifi

import (
	"testing"
)

func TestClientDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		client   Client
		expected string
	}{
		{
			name: "client with name",
			client: Client{
				Name:     "TestDevice",
				Hostname: "test-host",
				MAC:      MAC("aa:bb:cc:dd:ee:ff"),
			},
			expected: "TestDevice",
		},
		{
			name: "client with hostname only",
			client: Client{
				Name:     "",
				Hostname: "test-host",
				MAC:      MAC("aa:bb:cc:dd:ee:ff"),
			},
			expected: "test-host",
		},
		{
			name: "client with MAC only",
			client: Client{
				Name:     "",
				Hostname: "",
				MAC:      MAC("aa:bb:cc:dd:ee:ff"),
			},
			expected: "aa:bb:cc:dd:ee:ff",
		},
		{
			name: "empty client",
			client: Client{
				Name:     "",
				Hostname: "",
				MAC:      MAC(""),
			},
			expected: "-", // Final fallback value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.DisplayName()
			if result != tt.expected {
				t.Errorf("Client.DisplayName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestClientUpstreamMAC(t *testing.T) {
	tests := []struct {
		name     string
		client   Client
		expected string
	}{
		{
			name: "client with access point MAC",
			client: Client{
				AccessPointMAC: "aa:bb:cc:dd:ee:ff",
				GatewayMAC:     "11:22:33:44:55:66",
			},
			expected: "aa:bb:cc:dd:ee:ff",
		},
		{
			name: "client with gateway MAC only",
			client: Client{
				AccessPointMAC: "",
				GatewayMAC:     "11:22:33:44:55:66",
			},
			expected: "11:22:33:44:55:66",
		},
		{
			name: "client with no upstream MACs",
			client: Client{
				AccessPointMAC: "",
				GatewayMAC:     "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.UpstreamMAC()
			if result != tt.expected {
				t.Errorf("Client.UpstreamMAC() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestClientFilters(t *testing.T) {
	blockedClient := Client{IsBlocked: true}
	authorizedClient := Client{IsAuthorized: true}
	guestClient := Client{IsGuest: true}
	wiredClient := Client{IsWired: true}
	regularClient := Client{IsBlocked: false, IsAuthorized: false, IsGuest: false, IsWired: false}

	tests := []struct {
		name     string
		filter   ClientFilter
		client   Client
		expected bool
	}{
		// Blocked filter tests
		{"blocked client is blocked", Blocked, blockedClient, true},
		{"regular client is not blocked", Blocked, regularClient, false},

		// Authorized filter tests
		{"authorized client is authorized", Authorized, authorizedClient, true},
		{"regular client is not authorized", Authorized, regularClient, false},

		// Guest filter tests
		{"guest client is guest", Guest, guestClient, true},
		{"regular client is not guest", Guest, regularClient, false},

		// Wired filter tests
		{"wired client is wired", Wired, wiredClient, true},
		{"regular client is not wired", Wired, regularClient, false},

		// Not filter tests
		{"not blocked filter - blocked client", Not(Blocked), blockedClient, false},
		{"not blocked filter - regular client", Not(Blocked), regularClient, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter(tt.client)
			if result != tt.expected {
				t.Errorf("filter(%+v) = %v, want %v", tt.client, result, tt.expected)
			}
		})
	}
}

func TestClientSorting(t *testing.T) {
	client1 := Client{
		Name:          "Client1",
		IP:            IP("192.168.1.1"),
		BytesReceived: 1000,
		IsWired:       true,
	}
	client2 := Client{
		Name:          "Client2",
		IP:            IP("192.168.1.2"),
		BytesReceived: 2000,
		IsWired:       false,
	}

	tests := []struct {
		name     string
		sorter   func(*Client, *Client) bool
		lhs      *Client
		rhs      *Client
		expected bool
	}{
		{
			name:     "bytes received comparison",
			sorter:   ClientBytesReceived,
			lhs:      &client1,
			rhs:      &client2,
			expected: true, // client1 has fewer bytes
		},
		{
			name:     "IP comparison",
			sorter:   ClientIP,
			lhs:      &client1,
			rhs:      &client2,
			expected: true, // client1 IP is less
		},
		{
			name:     "name comparison",
			sorter:   ClientName,
			lhs:      &client1,
			rhs:      &client2,
			expected: true, // "Client1" < "Client2"
		},
		{
			name:     "wired comparison",
			sorter:   ClientWired,
			lhs:      &client1,
			rhs:      &client2,
			expected: true, // wired clients come first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sorter(tt.lhs, tt.rhs)
			if result != tt.expected {
				t.Errorf("sorter(%+v, %+v) = %v, want %v", tt.lhs, tt.rhs, result, tt.expected)
			}
		})
	}
}

func TestClientOrderedBySorting(t *testing.T) {
	clients := []Client{
		{Name: "Wireless1", IP: IP("192.168.1.3"), IsWired: false},
		{Name: "Wired1", IP: IP("192.168.1.1"), IsWired: true},
		{Name: "Wireless2", IP: IP("192.168.1.4"), IsWired: false},
		{Name: "Wired2", IP: IP("192.168.1.2"), IsWired: true},
	}

	// Test the default sorting (wired first, then by IP)
	ClientDefault.Sort(clients)

	// After sorting: wired clients should come first, sorted by IP
	expected := []string{"Wired1", "Wired2", "Wireless1", "Wireless2"}

	for i, expectedName := range expected {
		if clients[i].Name != expectedName {
			t.Errorf("After sorting, clients[%d].Name = %q, want %q", i, clients[i].Name, expectedName)
		}
	}
}
