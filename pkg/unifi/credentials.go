package unifi

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
)

// SecureString represents a securely handled string credential
type SecureString struct {
	data []byte
	salt []byte
}

// NewSecureString creates a new SecureString from a plain string
func NewSecureString(plaintext string) (*SecureString, error) {
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("credential cannot be empty")
	}

	// Generate random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// XOR data with salt for basic obfuscation in memory
	data := make([]byte, len(plaintext))
	for i, b := range []byte(plaintext) {
		data[i] = b ^ salt[i%len(salt)]
	}

	return &SecureString{
		data: data,
		salt: salt,
	}, nil
}

// String returns the plain text value (use sparingly)
func (s *SecureString) String() string {
	if s == nil || len(s.data) == 0 {
		return ""
	}

	// XOR back with salt to get original
	result := make([]byte, len(s.data))
	for i, b := range s.data {
		result[i] = b ^ s.salt[i%len(s.salt)]
	}

	return string(result)
}

// IsEmpty returns true if the credential is empty
func (s *SecureString) IsEmpty() bool {
	return s == nil || len(s.data) == 0
}

// Equals compares two SecureString values using constant time comparison
func (s *SecureString) Equals(other *SecureString) bool {
	if s == nil && other == nil {
		return true
	}
	if s == nil || other == nil {
		return false
	}

	thisVal := s.String()
	otherVal := other.String()

	// Use constant time comparison to prevent timing attacks
	result := subtle.ConstantTimeCompare([]byte(thisVal), []byte(otherVal)) == 1

	// Clear the temporary strings
	clearString(thisVal)
	clearString(otherVal)

	return result
}

// Clear zeroes out the credential data
func (s *SecureString) Clear() {
	if s == nil {
		return
	}

	// Zero out the data and salt
	for i := range s.data {
		s.data[i] = 0
	}
	for i := range s.salt {
		s.salt[i] = 0
	}

	s.data = nil
	s.salt = nil
}

// Validate performs basic credential validation
func (s *SecureString) Validate() error {
	if s.IsEmpty() {
		return fmt.Errorf("credential is empty")
	}

	plaintext := s.String()
	defer clearString(plaintext)

	if len(plaintext) < 1 {
		return fmt.Errorf("credential too short")
	}

	if len(plaintext) > 256 {
		return fmt.Errorf("credential too long (max 256 characters)")
	}

	return nil
}

// clearString attempts to zero out a string in memory
func clearString(s string) {
	// Convert string to byte slice and zero it
	// Note: This may not be completely effective due to Go's string immutability,
	// but provides best-effort clearing
	b := []byte(s)
	for i := range b {
		b[i] = 0
	}
}

// Credentials holds secure credentials
type Credentials struct {
	Username string
	Password *SecureString
}

// NewCredentials creates new secure credentials
func NewCredentials(username, password string) (*Credentials, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	securePassword, err := NewSecureString(password)
	if err != nil {
		return nil, fmt.Errorf("failed to create secure password: %w", err)
	}

	creds := &Credentials{
		Username: username,
		Password: securePassword,
	}

	// Validate credentials
	if err := creds.Validate(); err != nil {
		creds.Clear()
		return nil, err
	}

	return creds, nil
}

// Validate validates the credentials
func (c *Credentials) Validate() error {
	if c == nil {
		return fmt.Errorf("credentials are nil")
	}

	if c.Username == "" {
		return fmt.Errorf("username is empty")
	}

	if len(c.Username) > 128 {
		return fmt.Errorf("username too long (max 128 characters)")
	}

	return c.Password.Validate()
}

// Clear securely clears the credentials
func (c *Credentials) Clear() {
	if c == nil {
		return
	}

	c.Username = ""
	if c.Password != nil {
		c.Password.Clear()
		c.Password = nil
	}
}
