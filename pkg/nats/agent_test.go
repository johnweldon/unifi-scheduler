package nats

import (
	"testing"
)

func TestDetailBucket(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		expected string
	}{
		{
			name:     "simple base",
			base:     "unifi",
			expected: "unifi-details",
		},
		{
			name:     "empty base",
			base:     "",
			expected: "-details",
		},
		{
			name:     "complex base",
			base:     "my-app-v1",
			expected: "my-app-v1-details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetailBucket(tt.base)
			if result != tt.expected {
				t.Errorf("DetailBucket(%q) = %q, want %q", tt.base, result, tt.expected)
			}
		})
	}
}

func TestByMACBucket(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		expected string
	}{
		{
			name:     "simple base",
			base:     "unifi",
			expected: "unifi-bymac",
		},
		{
			name:     "empty base",
			base:     "",
			expected: "-bymac",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ByMACBucket(tt.base)
			if result != tt.expected {
				t.Errorf("ByMACBucket(%q) = %q, want %q", tt.base, result, tt.expected)
			}
		})
	}
}

func TestByNameBucket(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		expected string
	}{
		{
			name:     "simple base",
			base:     "unifi",
			expected: "unifi-byname",
		},
		{
			name:     "empty base",
			base:     "",
			expected: "-byname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ByNameBucket(tt.base)
			if result != tt.expected {
				t.Errorf("ByNameBucket(%q) = %q, want %q", tt.base, result, tt.expected)
			}
		})
	}
}

func TestEventStream(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		expected string
	}{
		{
			name:     "simple base",
			base:     "unifi",
			expected: "unifi-events",
		},
		{
			name:     "empty base",
			base:     "",
			expected: "-events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EventStream(tt.base)
			if result != tt.expected {
				t.Errorf("EventStream(%q) = %q, want %q", tt.base, result, tt.expected)
			}
		})
	}
}

func TestSubSubject(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		subject  string
		expected string
	}{
		{
			name:     "simple case",
			base:     "unifi",
			subject:  "clients",
			expected: "unifi.clients",
		},
		{
			name:     "empty base",
			base:     "",
			subject:  "clients",
			expected: ".clients",
		},
		{
			name:     "empty subject",
			base:     "unifi",
			subject:  "",
			expected: "unifi.",
		},
		{
			name:     "both empty",
			base:     "",
			subject:  "",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := subSubject(tt.base, tt.subject)
			if result != tt.expected {
				t.Errorf("subSubject(%q, %q) = %q, want %q", tt.base, tt.subject, result, tt.expected)
			}
		})
	}
}

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase letters and numbers",
			input:    "abc123",
			expected: "abc123",
		},
		{
			name:     "MAC address with colons",
			input:    "aa:bb:cc:dd:ee:ff",
			expected: "aa-bb-cc-dd-ee-ff",
		},
		{
			name:     "string with spaces",
			input:    "hello world",
			expected: "hello-world",
		},
		{
			name:     "mixed case letters",
			input:    "MyDevice123",
			expected: "mydevice123", // uppercase converted to lowercase
		},
		{
			name:     "dots and dashes preserved",
			input:    "device.sub-component",
			expected: "device.sub-component",
		},
		{
			name:     "special characters removed",
			input:    "device@#$%^&*()",
			expected: "device",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only invalid characters",
			input:    "@#$%",
			expected: "",
		},
		{
			name:     "combination test",
			input:    "Test Device 01:23:45:67:89:ab!@#",
			expected: "test-device-01-23-45-67-89-ab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeKey(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAgentConstants(t *testing.T) {
	// Test that constants are not empty and follow expected patterns
	if ActiveKey == "" {
		t.Error("ActiveKey should not be empty")
	}
	if DevicesKey == "" {
		t.Error("DevicesKey should not be empty")
	}
	if EventsKey == "" {
		t.Error("EventsKey should not be empty")
	}

	// Test expected values
	if ActiveKey != "active" {
		t.Errorf("ActiveKey = %q, want %q", ActiveKey, "active")
	}
	if DevicesKey != "devices" {
		t.Errorf("DevicesKey = %q, want %q", DevicesKey, "devices")
	}
	if EventsKey != "events" {
		t.Errorf("EventsKey = %q, want %q", EventsKey, "events")
	}
}

