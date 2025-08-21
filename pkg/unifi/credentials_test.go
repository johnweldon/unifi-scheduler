package unifi

import (
	"strings"
	"testing"
)

func TestNewSecureString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "valid string",
			input:     "mypassword123",
			wantError: false,
		},
		{
			name:      "empty string",
			input:     "",
			wantError: true,
		},
		{
			name:      "single character",
			input:     "a",
			wantError: false,
		},
		{
			name:      "long string",
			input:     strings.Repeat("x", 200),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewSecureString(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewSecureString(%q) expected error but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("NewSecureString(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result == nil {
				t.Errorf("NewSecureString(%q) returned nil without error", tt.input)
				return
			}

			// Test that we can retrieve the original value
			if result.String() != tt.input {
				t.Errorf("SecureString.String() = %q, want %q", result.String(), tt.input)
			}

			// Clean up
			result.Clear()
		})
	}
}

func TestSecureStringIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *SecureString
		expected bool
	}{
		{
			name: "nil secure string",
			setup: func() *SecureString {
				return nil
			},
			expected: true,
		},
		{
			name: "empty secure string",
			setup: func() *SecureString {
				s := &SecureString{}
				return s
			},
			expected: true,
		},
		{
			name: "valid secure string",
			setup: func() *SecureString {
				s, _ := NewSecureString("test")
				return s
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.setup()
			result := s.IsEmpty()

			if result != tt.expected {
				t.Errorf("SecureString.IsEmpty() = %v, want %v", result, tt.expected)
			}

			if s != nil {
				s.Clear()
			}
		})
	}
}

func TestSecureStringEquals(t *testing.T) {
	s1, _ := NewSecureString("password123")
	s2, _ := NewSecureString("password123")
	s3, _ := NewSecureString("different")

	tests := []struct {
		name     string
		s1       *SecureString
		s2       *SecureString
		expected bool
	}{
		{
			name:     "equal strings",
			s1:       s1,
			s2:       s2,
			expected: true,
		},
		{
			name:     "different strings",
			s1:       s1,
			s2:       s3,
			expected: false,
		},
		{
			name:     "both nil",
			s1:       nil,
			s2:       nil,
			expected: true,
		},
		{
			name:     "first nil",
			s1:       nil,
			s2:       s1,
			expected: false,
		},
		{
			name:     "second nil",
			s1:       s1,
			s2:       nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.s1.Equals(tt.s2)
			if result != tt.expected {
				t.Errorf("SecureString.Equals() = %v, want %v", result, tt.expected)
			}
		})
	}

	// Clean up
	s1.Clear()
	s2.Clear()
	s3.Clear()
}

func TestSecureStringValidate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "valid password",
			input:     "validpassword",
			wantError: false,
		},
		{
			name:      "single character",
			input:     "a",
			wantError: false,
		},
		{
			name:      "too long password",
			input:     strings.Repeat("x", 300),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s *SecureString
			var err error

			if tt.input == "" {
				s = &SecureString{}
			} else {
				s, _ = NewSecureString(tt.input)
			}

			err = s.Validate()

			if tt.wantError {
				if err == nil {
					t.Errorf("SecureString.Validate() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("SecureString.Validate() unexpected error: %v", err)
				}
			}

			if s != nil {
				s.Clear()
			}
		})
	}
}

func TestNewCredentials(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		password  string
		wantError bool
	}{
		{
			name:      "valid credentials",
			username:  "testuser",
			password:  "testpassword",
			wantError: false,
		},
		{
			name:      "empty username",
			username:  "",
			password:  "testpassword",
			wantError: true,
		},
		{
			name:      "empty password",
			username:  "testuser",
			password:  "",
			wantError: true,
		},
		{
			name:      "username too long",
			username:  strings.Repeat("x", 200),
			password:  "testpassword",
			wantError: true,
		},
		{
			name:      "password too long",
			username:  "testuser",
			password:  strings.Repeat("x", 300),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := NewCredentials(tt.username, tt.password)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewCredentials() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("NewCredentials() unexpected error: %v", err)
				return
			}

			if creds == nil {
				t.Errorf("NewCredentials() returned nil without error")
				return
			}

			if creds.Username != tt.username {
				t.Errorf("Credentials.Username = %q, want %q", creds.Username, tt.username)
			}

			if creds.Password.String() != tt.password {
				t.Errorf("Credentials.Password = %q, want %q", creds.Password.String(), tt.password)
			}

			// Clean up
			creds.Clear()
		})
	}
}

func TestCredentialsClear(t *testing.T) {
	creds, _ := NewCredentials("testuser", "testpass")

	// Verify credentials are set
	if creds.Username != "testuser" {
		t.Errorf("Username not set correctly before clear")
	}
	if creds.Password.String() != "testpass" {
		t.Errorf("Password not set correctly before clear")
	}

	// Clear credentials
	creds.Clear()

	// Verify credentials are cleared
	if creds.Username != "" {
		t.Errorf("Username not cleared: %q", creds.Username)
	}
	if creds.Password != nil {
		t.Errorf("Password pointer not cleared")
	}
}

func TestSecureStringClear(t *testing.T) {
	s, _ := NewSecureString("sensitive")

	// Verify string is accessible
	if s.String() != "sensitive" {
		t.Errorf("String not accessible before clear")
	}

	// Clear the string
	s.Clear()

	// Verify string is cleared (should be empty)
	if s.String() != "" {
		t.Errorf("String not cleared properly")
	}
}

