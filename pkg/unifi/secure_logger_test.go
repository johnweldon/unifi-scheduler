package unifi

import (
	"bytes"
	"testing"
)

func TestSecureWriter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "JSON password field",
			input:    `{"username":"user","password":"secret123"}`,
			expected: `{"username":"user","password":[REDACTED]}`,
		},
		{
			name:     "URL with credentials",
			input:    "https://user:secret@example.com/api",
			expected: "https://[REDACTED]@example.com/api",
		},
		{
			name:     "Command line password flag",
			input:    "./app --username user --password secret123 --debug",
			expected: "./app --username user --password [REDACTED] --debug",
		},
		{
			name:     "Bearer token",
			input:    "Authorization: Bearer abc123def456",
			expected: "Authorization: Bearer [REDACTED]",
		},
		{
			name:     "API key",
			input:    `{"api_key": "sk-123456789"}`,
			expected: `{"api_key": "[REDACTED]"}`,
		},
		{
			name:     "Multiple sensitive fields",
			input:    `{"user":"admin","password":"pass123","api_key":"key456"}`,
			expected: `{"user":"admin","password":[REDACTED],"api_key":"[REDACTED]"}`,
		},
		{
			name:     "Non-sensitive data unchanged",
			input:    `{"username":"admin","timeout":30,"enabled":true}`,
			expected: `{"username":"admin","timeout":30,"enabled":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			sw := NewSecureWriter(&buf)

			n, err := sw.Write([]byte(tt.input))
			if err != nil {
				t.Errorf("SecureWriter.Write() error = %v", err)
				return
			}

			if n != len(tt.input) {
				t.Errorf("SecureWriter.Write() returned %d bytes, want %d", n, len(tt.input))
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("SecureWriter output = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRedactSensitiveFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "password field",
			input:    `password: mysecret`,
			expected: `password: [REDACTED]`,
		},
		{
			name:     "secret field",
			input:    `secret=topsecret`,
			expected: `secret=[REDACTED]`,
		},
		{
			name:     "case insensitive",
			input:    `PASSWORD: "secret123"`,
			expected: `PASSWORD: [REDACTED]`,
		},
		{
			name:     "JSON format",
			input:    `"token": "abc123"`,
			expected: `"token": [REDACTED]`,
		},
		{
			name:     "non-sensitive unchanged",
			input:    `username: admin, timeout: 30`,
			expected: `username: admin, timeout: 30`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSensitiveFields(tt.input)
			if result != tt.expected {
				t.Errorf("RedactSensitiveFields(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsLikelySensitive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "contains password",
			input:    "user password",
			expected: true,
		},
		{
			name:     "contains secret",
			input:    "API secret key",
			expected: true,
		},
		{
			name:     "contains token",
			input:    "auth token",
			expected: true,
		},
		{
			name:     "case insensitive",
			input:    "USER PASSWORD",
			expected: true,
		},
		{
			name:     "no sensitive terms",
			input:    "username and timeout",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLikelySensitive(tt.input)
			if result != tt.expected {
				t.Errorf("IsLikelySensitive(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSecureWriterCustomScrubber(t *testing.T) {
	var buf bytes.Buffer
	sw := NewSecureWriter(&buf)

	// Add custom scrubber to redact phone numbers
	sw.AddCustomScrubber(func(data string) string {
		// Simple phone number pattern
		return bytes.NewBufferString(data).String() // placeholder - would implement actual regex
	})

	input := "Contact: admin, phone: 555-1234"
	sw.Write([]byte(input))

	// Should still work with custom scrubber added
	result := buf.String()
	if result == "" {
		t.Error("SecureWriter with custom scrubber produced empty output")
	}
}

func TestScrubString(t *testing.T) {
	var buf bytes.Buffer
	sw := NewSecureWriter(&buf)

	input := `{"password":"secret","username":"admin"}`
	result := sw.ScrubString(input)

	expected := `{"password":[REDACTED],"username":"admin"}`
	if result != expected {
		t.Errorf("ScrubString(%q) = %q, want %q", input, result, expected)
	}
}
