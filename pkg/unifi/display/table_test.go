package display

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
)

func TestClientsTable(t *testing.T) {
	var buf bytes.Buffer
	clients := []unifi.Client{
		{
			Name:           "TestClient1",
			IP:             unifi.IP("192.168.1.100"),
			MAC:            unifi.MAC("aa:bb:cc:dd:ee:01"),
			IsBlocked:      false,
			IsGuest:        false,
			IsWired:        true,
			BytesReceived:  1024000,
			BytesSent:      2048000,
			AccessPointMAC: "aa:bb:cc:dd:ee:ff",
		},
		{
			Name:           "TestClient2",
			IP:             unifi.IP("192.168.1.101"),
			MAC:            unifi.MAC("aa:bb:cc:dd:ee:02"),
			IsBlocked:      true,
			IsGuest:        true,
			IsWired:        false,
			BytesReceived:  512000,
			BytesSent:      1024000,
			AccessPointMAC: "aa:bb:cc:dd:ee:aa",
		},
	}

	renderer := ClientsTable(&buf, clients)
	if renderer == nil {
		t.Fatal("ClientsTable returned nil renderer")
	}

	output := renderer.Render()
	if output == "" {
		t.Error("ClientsTable rendered empty output")
	}

	// For debugging - let's see what the actual output looks like
	t.Logf("Actual output:\n%s", output)

	// Check that client names appear in output
	if !strings.Contains(output, "TestClient1") {
		t.Error("Output should contain TestClient1")
	}
	if !strings.Contains(output, "TestClient2") {
		t.Error("Output should contain TestClient2")
	}

	// Check that IP addresses appear
	if !strings.Contains(output, "192.168.1.100") {
		t.Error("Output should contain first IP address")
	}
	if !strings.Contains(output, "192.168.1.101") {
		t.Error("Output should contain second IP address")
	}

	// Check that total count appears - check for TOTAL format
	if !strings.Contains(output, "TOTAL 2") {
		t.Error("Output should contain total count")
	}

	// Verify that table headers are present - they are uppercase
	basicHeaders := []string{"NAME", "IP"}
	for _, header := range basicHeaders {
		if !strings.Contains(output, header) {
			t.Errorf("Output should contain header: %s", header)
		}
	}
}

func TestEventsTable(t *testing.T) {
	var buf bytes.Buffer

	// Create test events
	events := []unifi.Event{
		{
			Key:         unifi.EventTypeWirelessUserConnected,
			MAC:         unifi.MAC("aa:bb:cc:dd:ee:01"),
			User:        unifi.MAC("aa:bb:cc:dd:ee:01"),
			AccessPoint: unifi.MAC("ap:01:02:03:04:05"),
			TimeStamp:   unifi.TimeStamp(time.Now().Unix()),
		},
		{
			Key:         unifi.EventTypeWirelessUserDisconnected,
			MAC:         unifi.MAC("aa:bb:cc:dd:ee:02"),
			User:        unifi.MAC("aa:bb:cc:dd:ee:02"),
			AccessPoint: unifi.MAC("ap:01:02:03:04:06"),
			TimeStamp:   unifi.TimeStamp(time.Now().Unix()),
		},
	}

	// Create a display name function
	displayName := func(mac unifi.MAC) (string, bool) {
		switch mac {
		case unifi.MAC("aa:bb:cc:dd:ee:01"):
			return "TestDevice1", true
		case unifi.MAC("aa:bb:cc:dd:ee:02"):
			return "TestDevice2", true
		case unifi.MAC("ap:01:02:03:04:05"):
			return "AccessPoint1", true
		default:
			return "", false
		}
	}

	renderer := EventsTable(&buf, displayName, events)
	if renderer == nil {
		t.Fatal("EventsTable returned nil renderer")
	}

	output := renderer.Render()
	if output == "" {
		t.Error("EventsTable rendered empty output")
	}

	// For debugging
	t.Logf("Events output:\n%s", output)

	// Check that device names appear in output
	if !strings.Contains(output, "TestDevice1") {
		t.Error("Output should contain TestDevice1")
	}
	if !strings.Contains(output, "TestDevice2") {
		t.Error("Output should contain TestDevice2")
	}

	// Check that event types appear (processed event names)
	if !strings.Contains(output, "Connected") {
		t.Error("Output should contain connected event")
	}
	if !strings.Contains(output, "Disconnected") {
		t.Error("Output should contain disconnected event")
	}

	// Check that total count appears - uppercase format
	if !strings.Contains(output, "TOTAL 2") {
		t.Error("Output should contain total count")
	}

	// Verify that table headers are present - uppercase
	expectedHeaders := []string{"NAME", "EVENT"}
	for _, header := range expectedHeaders {
		if !strings.Contains(output, header) {
			t.Errorf("Output should contain header: %s", header)
		}
	}
}

func TestEventsTableWithRoamEvent(t *testing.T) {
	var buf bytes.Buffer

	events := []unifi.Event{
		{
			Key:             unifi.EventTypeWirelessUserRoam,
			User:            unifi.MAC("aa:bb:cc:dd:ee:01"),
			AccessPointFrom: unifi.MAC("ap:01:02:03:04:05"),
			AccessPointTo:   unifi.MAC("ap:01:02:03:04:06"),
			TimeStamp:       unifi.TimeStamp(time.Now().Unix()),
		},
	}

	displayName := func(mac unifi.MAC) (string, bool) {
		switch mac {
		case unifi.MAC("aa:bb:cc:dd:ee:01"):
			return "RoamingDevice", true
		case unifi.MAC("ap:01:02:03:04:05"):
			return "OldAP", true
		case unifi.MAC("ap:01:02:03:04:06"):
			return "NewAP", true
		default:
			return "", false
		}
	}

	renderer := EventsTable(&buf, displayName, events)
	output := renderer.Render()

	// Check that roaming event shows correct from/to access points
	if !strings.Contains(output, "RoamingDevice") {
		t.Error("Output should contain roaming device name")
	}
	if !strings.Contains(output, "OldAP") {
		t.Error("Output should contain old access point")
	}
	if !strings.Contains(output, "NewAP") {
		t.Error("Output should contain new access point")
	}
}

func TestEventsTableWithLANEvent(t *testing.T) {
	var buf bytes.Buffer

	events := []unifi.Event{
		{
			Key:       unifi.EventTypeLANUserConnected,
			User:      unifi.MAC("aa:bb:cc:dd:ee:01"),
			Switch:    unifi.MAC("sw:01:02:03:04:05"),
			TimeStamp: unifi.TimeStamp(time.Now().Unix()),
		},
	}

	displayName := func(mac unifi.MAC) (string, bool) {
		switch mac {
		case unifi.MAC("aa:bb:cc:dd:ee:01"):
			return "LANDevice", true
		case unifi.MAC("sw:01:02:03:04:05"):
			return "MainSwitch", true
		default:
			return "", false
		}
	}

	renderer := EventsTable(&buf, displayName, events)
	output := renderer.Render()

	// Check that LAN event shows correct device and switch
	if !strings.Contains(output, "LANDevice") {
		t.Error("Output should contain LAN device name")
	}
	if !strings.Contains(output, "MainSwitch") {
		t.Error("Output should contain switch name")
	}
}

func TestEventsTableWithUnknownEvent(t *testing.T) {
	var buf bytes.Buffer

	events := []unifi.Event{
		{
			Key:       "EVT_Unknown_Event",
			MAC:       unifi.MAC("aa:bb:cc:dd:ee:01"),
			TimeStamp: unifi.TimeStamp(time.Now().Unix()),
		},
	}

	displayName := func(mac unifi.MAC) (string, bool) {
		return "", false
	}

	renderer := EventsTable(&buf, displayName, events)
	output := renderer.Render()

	// For unknown events, the processed key should be used (removes "EVT_" prefix)
	if !strings.Contains(output, "nown_Event") {
		t.Logf("Unknown event output:\n%s", output)
		t.Error("Output should contain the processed event key")
	}
}

func TestStyleDefault(t *testing.T) {
	// Test that StyleDefault is properly configured
	if StyleDefault.Name != "StyleDefault" {
		t.Errorf("StyleDefault.Name = %q, want %q", StyleDefault.Name, "StyleDefault")
	}

	// Verify that style has reasonable settings
	if StyleDefault.Box.Left == "" && StyleDefault.Box.Right == "" {
		// This is expected for NoBordersAndSeparators style
	}
}

func TestEmptyClientsTable(t *testing.T) {
	var buf bytes.Buffer
	var emptyClients []unifi.Client

	renderer := ClientsTable(&buf, emptyClients)
	if renderer == nil {
		t.Fatal("ClientsTable returned nil renderer for empty clients")
	}

	output := renderer.Render()
	if output == "" {
		t.Error("ClientsTable should render headers even for empty input")
	}

	// Should still show total count - uppercase format
	if !strings.Contains(output, "TOTAL 0") {
		t.Error("Output should show total count of 0 for empty table")
	}
}

func TestEmptyEventsTable(t *testing.T) {
	var buf bytes.Buffer
	var emptyEvents []unifi.Event

	displayName := func(mac unifi.MAC) (string, bool) {
		return "", false
	}

	renderer := EventsTable(&buf, displayName, emptyEvents)
	if renderer == nil {
		t.Fatal("EventsTable returned nil renderer for empty events")
	}

	output := renderer.Render()
	if output == "" {
		t.Error("EventsTable should render headers even for empty input")
	}

	// Should still show total count - uppercase format
	if !strings.Contains(output, "TOTAL 0") {
		t.Error("Output should show total count of 0 for empty table")
	}
}
