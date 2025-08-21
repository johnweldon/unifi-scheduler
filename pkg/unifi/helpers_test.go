package unifi

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "first string non-empty",
			input:    []string{"hello", "world", "test"},
			expected: "hello",
		},
		{
			name:     "second string non-empty",
			input:    []string{"", "world", "test"},
			expected: "world",
		},
		{
			name:     "last string non-empty",
			input:    []string{"", "", "test"},
			expected: "test",
		},
		{
			name:     "all empty strings",
			input:    []string{"", "", ""},
			expected: "",
		},
		{
			name:     "no strings",
			input:    []string{},
			expected: "",
		},
		{
			name:     "single non-empty string",
			input:    []string{"only"},
			expected: "only",
		},
		{
			name:     "single empty string",
			input:    []string{""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := firstNonEmpty(tt.input...)
			if result != tt.expected {
				t.Errorf("firstNonEmpty(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatBytesSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{
			name:     "zero bytes",
			size:     0,
			expected: "",
		},
		{
			name:     "negative bytes",
			size:     -100,
			expected: "",
		},
		{
			name:     "small bytes",
			size:     512,
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			size:     1024,
			expected: "1.0 kB",
		},
		{
			name:     "megabytes",
			size:     1048576, // 1024 * 1024
			expected: "1.0 MB",
		},
		{
			name:     "gigabytes",
			size:     1073741824, // 1024 * 1024 * 1024
			expected: "1.1 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytesSize(tt.size)
			if result != tt.expected {
				t.Errorf("formatBytesSize(%d) = %q, want %q", tt.size, result, tt.expected)
			}
		})
	}
}

func TestMACString(t *testing.T) {
	tests := []struct {
		name     string
		mac      MAC
		expected string
	}{
		{
			name:     "valid MAC address",
			mac:      MAC("aa:bb:cc:dd:ee:ff"),
			expected: "aa:bb:cc:dd:ee:ff",
		},
		{
			name:     "empty MAC address",
			mac:      MAC(""),
			expected: "00:00:00:00:00:00",
		},
		{
			name:     "uppercase MAC address",
			mac:      MAC("AA:BB:CC:DD:EE:FF"),
			expected: "AA:BB:CC:DD:EE:FF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mac.String()
			if result != tt.expected {
				t.Errorf("MAC(%q).String() = %q, want %q", string(tt.mac), result, tt.expected)
			}
		})
	}
}

func TestIPLess(t *testing.T) {
	tests := []struct {
		name     string
		lhs      IP
		rhs      IP
		expected bool
	}{
		{
			name:     "lhs less than rhs",
			lhs:      IP("192.168.1.1"),
			rhs:      IP("192.168.1.2"),
			expected: true,
		},
		{
			name:     "lhs greater than rhs",
			lhs:      IP("192.168.1.2"),
			rhs:      IP("192.168.1.1"),
			expected: false,
		},
		{
			name:     "equal IPs",
			lhs:      IP("192.168.1.1"),
			rhs:      IP("192.168.1.1"),
			expected: false,
		},
		{
			name:     "lhs empty",
			lhs:      IP(""),
			rhs:      IP("192.168.1.1"),
			expected: true,
		},
		{
			name:     "rhs empty",
			lhs:      IP("192.168.1.1"),
			rhs:      IP(""),
			expected: false,
		},
		{
			name:     "both empty",
			lhs:      IP(""),
			rhs:      IP(""),
			expected: false,
		},
		{
			name:     "IPv4 vs IPv6",
			lhs:      IP("192.168.1.1"),
			rhs:      IP("::1"),
			expected: false, // IPv4 is not less than IPv6 in this implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.lhs.Less(tt.rhs)
			if result != tt.expected {
				t.Errorf("IP(%q).Less(%q) = %v, want %v", string(tt.lhs), string(tt.rhs), result, tt.expected)
			}
		})
	}
}

func TestNumberUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Number
		shouldError bool
	}{
		{
			name:     "valid number",
			input:    "42",
			expected: Number(42),
		},
		{
			name:     "valid string number",
			input:    `"123"`,
			expected: Number(123),
		},
		{
			name:     "zero",
			input:    "0",
			expected: Number(0),
		},
		{
			name:     "negative number",
			input:    "-99",
			expected: Number(-99),
		},
		{
			name:        "invalid string",
			input:       `"not-a-number"`,
			shouldError: true,
		},
		{
			name:        "invalid JSON",
			input:       `invalid`,
			shouldError: true,
		},
		{
			name:        "empty input",
			input:       "",
			shouldError: true, // Empty JSON input causes an error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n Number
			err := json.Unmarshal([]byte(tt.input), &n)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Number.UnmarshalJSON(%q) should have returned an error", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Number.UnmarshalJSON(%q) returned unexpected error: %v", tt.input, err)
				return
			}

			if n != tt.expected {
				t.Errorf("Number.UnmarshalJSON(%q) = %v, want %v", tt.input, n, tt.expected)
			}
		})
	}
}

func TestTimeStampString(t *testing.T) {
	// Test with a known timestamp (January 1, 2023 12:00:00 UTC)
	timestamp := TimeStamp(1672574400000) // milliseconds
	result := timestamp.String()

	// The result will be relative to current time, so we just check it's not empty
	if result == "" {
		t.Errorf("TimeStamp.String() returned empty string")
	}

	// Test ShortTime format
	shortTime := timestamp.ShortTime()
	expectedTime := time.UnixMilli(int64(timestamp)).Format("03:04:05PM")
	if shortTime != expectedTime {
		t.Errorf("TimeStamp.ShortTime() = %q, want %q", shortTime, expectedTime)
	}
}

func TestDurationString(t *testing.T) {
	// Test duration string formatting (relative to current time)
	duration := Duration(3600) // 1 hour in seconds
	result := duration.String()

	// The result will be relative to current time, so we just check it's not empty
	if result == "" {
		t.Errorf("Duration.String() returned empty string")
	}
}
