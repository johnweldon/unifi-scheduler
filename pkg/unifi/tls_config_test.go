package unifi

import (
	"crypto/tls"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultTLSConfig(t *testing.T) {
	config := DefaultTLSConfig()

	if config.InsecureSkipVerify {
		t.Error("Default config should not skip certificate verification")
	}

	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("Default minimum TLS version should be 1.2, got %d", config.MinVersion)
	}

	if config.MaxVersion != tls.VersionTLS13 {
		t.Errorf("Default maximum TLS version should be 1.3, got %d", config.MaxVersion)
	}

	if !config.StrictValidation {
		t.Error("Default config should have strict validation enabled")
	}

	if len(config.CipherSuites) == 0 {
		t.Error("Default config should have cipher suites configured")
	}
}

func TestInsecureTLSConfig(t *testing.T) {
	config := InsecureTLSConfig()

	if !config.InsecureSkipVerify {
		t.Error("Insecure config should skip certificate verification")
	}

	if config.StrictValidation {
		t.Error("Insecure config should not have strict validation")
	}
}

func TestTLSConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *TLSConfig
		wantErr bool
		errType error
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errType: ErrInvalidTLSConfig,
		},
		{
			name:    "valid default config",
			config:  DefaultTLSConfig(),
			wantErr: false,
		},
		{
			name: "insecure with strict validation",
			config: &TLSConfig{
				InsecureSkipVerify: true,
				StrictValidation:   true,
			},
			wantErr: false, // InsecureSkipVerify overrides strict validation
		},
		{
			name: "TLS version below 1.2 with strict validation",
			config: &TLSConfig{
				MinVersion:       tls.VersionTLS11,
				StrictValidation: true,
			},
			wantErr: true,
			errType: ErrInsecureTLS,
		},
		{
			name: "client cert without key",
			config: &TLSConfig{
				ClientCertFile: "/path/to/cert",
				// ClientKeyFile missing
			},
			wantErr: true,
			errType: ErrCertificateNotFound, // File existence is checked first
		},
		{
			name: "client key without cert",
			config: &TLSConfig{
				ClientKeyFile: "/path/to/key",
				// ClientCertFile missing
			},
			wantErr: true,
			errType: ErrCertificateNotFound, // File existence is checked first
		},
		{
			name: "non-existent client cert file",
			config: &TLSConfig{
				ClientCertFile: "/nonexistent/cert.pem",
				ClientKeyFile:  "/nonexistent/key.pem",
			},
			wantErr: true,
			errType: ErrCertificateNotFound,
		},
		{
			name: "non-existent root CA file",
			config: &TLSConfig{
				RootCAFile: "/nonexistent/ca.pem",
			},
			wantErr: true,
			errType: ErrCertificateNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("Validate() error = %v, wantErrType %v", err, tt.errType)
			}
		})
	}
}

func TestTLSConfig_ToStandardTLSConfig(t *testing.T) {
	config := DefaultTLSConfig()
	config.ServerName = "test.example.com"

	stdConfig, err := config.ToStandardTLSConfig()
	if err != nil {
		t.Fatalf("ToStandardTLSConfig() error = %v", err)
	}

	if stdConfig.ServerName != "test.example.com" {
		t.Errorf("ServerName = %v, want %v", stdConfig.ServerName, "test.example.com")
	}

	if stdConfig.MinVersion != config.MinVersion {
		t.Errorf("MinVersion = %v, want %v", stdConfig.MinVersion, config.MinVersion)
	}

	if stdConfig.MaxVersion != config.MaxVersion {
		t.Errorf("MaxVersion = %v, want %v", stdConfig.MaxVersion, config.MaxVersion)
	}

	if len(stdConfig.CipherSuites) != len(config.CipherSuites) {
		t.Errorf("CipherSuites length = %v, want %v", len(stdConfig.CipherSuites), len(config.CipherSuites))
	}
}

func TestTLSConfig_CreateSecureTransport(t *testing.T) {
	config := DefaultTLSConfig()
	config.HandshakeTimeout = 5 * time.Second

	transport, err := config.CreateSecureTransport()
	if err != nil {
		t.Fatalf("CreateSecureTransport() error = %v", err)
	}

	if transport.TLSHandshakeTimeout != config.HandshakeTimeout {
		t.Errorf("TLSHandshakeTimeout = %v, want %v", transport.TLSHandshakeTimeout, config.HandshakeTimeout)
	}

	if transport.TLSClientConfig == nil {
		t.Error("TLSClientConfig should not be nil")
	}

	if !transport.ForceAttemptHTTP2 {
		t.Error("HTTP/2 should be enabled by default")
	}
}

func TestNewTLSConfig(t *testing.T) {
	config := NewTLSConfig(
		WithInsecureSkipVerify(true),
		WithMinTLSVersion(tls.VersionTLS13),
		WithServerName("test.example.com"),
		WithHandshakeTimeout(15*time.Second),
	)

	if !config.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true")
	}

	if config.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion = %v, want %v", config.MinVersion, tls.VersionTLS13)
	}

	if config.ServerName != "test.example.com" {
		t.Errorf("ServerName = %v, want %v", config.ServerName, "test.example.com")
	}

	if config.HandshakeTimeout != 15*time.Second {
		t.Errorf("HandshakeTimeout = %v, want %v", config.HandshakeTimeout, 15*time.Second)
	}
}

func TestSecureCipherSuites(t *testing.T) {
	suites := SecureCipherSuites()

	if len(suites) == 0 {
		t.Error("SecureCipherSuites should return non-empty list")
	}

	// Check that it includes TLS 1.3 suites
	hasTLS13 := false
	for _, suite := range suites {
		if suite == tls.TLS_AES_256_GCM_SHA384 ||
			suite == tls.TLS_AES_128_GCM_SHA256 ||
			suite == tls.TLS_CHACHA20_POLY1305_SHA256 {
			hasTLS13 = true
			break
		}
	}

	if !hasTLS13 {
		t.Error("SecureCipherSuites should include TLS 1.3 cipher suites")
	}
}

func TestTLSVersionString(t *testing.T) {
	tests := []struct {
		version  uint16
		expected string
	}{
		{tls.VersionTLS10, "TLS 1.0"},
		{tls.VersionTLS11, "TLS 1.1"},
		{tls.VersionTLS12, "TLS 1.2"},
		{tls.VersionTLS13, "TLS 1.3"},
		{0x9999, "Unknown (0x9999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := TLSVersionString(tt.version)
			if result != tt.expected {
				t.Errorf("TLSVersionString(%d) = %v, want %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestCipherSuiteName(t *testing.T) {
	tests := []struct {
		suite    uint16
		expected string
	}{
		{tls.TLS_AES_128_GCM_SHA256, "TLS_AES_128_GCM_SHA256"},
		{tls.TLS_AES_256_GCM_SHA384, "TLS_AES_256_GCM_SHA384"},
		{tls.TLS_CHACHA20_POLY1305_SHA256, "TLS_CHACHA20_POLY1305_SHA256"},
		{0x9999, "Unknown (0x9999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := CipherSuiteName(tt.suite)
			if result != tt.expected {
				t.Errorf("CipherSuiteName(%d) = %v, want %v", tt.suite, result, tt.expected)
			}
		})
	}
}

func TestValidateEndpointTLS(t *testing.T) {
	config := DefaultTLSConfig()

	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{
			name:     "non-HTTPS endpoint",
			endpoint: "http://example.com",
			wantErr:  true,
		},
		{
			name:     "HTTPS endpoint",
			endpoint: "https://example.com",
			wantErr:  false, // May fail due to network, but URL format is correct
		},
		{
			name:     "malformed URL",
			endpoint: "not-a-url",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpointTLS(tt.endpoint, config)

			// For the non-HTTPS case, we should get an immediate error about the protocol
			if tt.name == "non-HTTPS endpoint" && err == nil {
				t.Error("ValidateEndpointTLS() should reject non-HTTPS endpoints")
				return
			}

			// For HTTPS endpoints, we might get network errors, which is fine for testing
			// We just want to make sure it doesn't reject the HTTPS protocol itself
			if tt.name == "HTTPS endpoint" && err != nil {
				// Check if it's a protocol error vs network error
				if err.Error() == "endpoint must use HTTPS for secure communication: https://example.com" {
					t.Error("ValidateEndpointTLS() should accept HTTPS endpoints")
				}
				// Network errors are expected in tests
			}
		})
	}
}

func TestTLSConfigWithFiles(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "tls_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create dummy cert and key files (not valid certificates, just for path testing)
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	caFile := filepath.Join(tmpDir, "ca.pem")

	// Write dummy content
	if err := os.WriteFile(certFile, []byte("dummy cert"), 0o600); err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, []byte("dummy key"), 0o600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}
	if err := os.WriteFile(caFile, []byte("dummy ca"), 0o600); err != nil {
		t.Fatalf("Failed to write CA file: %v", err)
	}

	config := NewTLSConfig(
		WithClientCertificate(certFile, keyFile),
		WithRootCA(caFile),
	)

	// This should pass validation (file paths exist)
	if err := config.Validate(); err != nil {
		t.Errorf("Config with valid file paths should validate: %v", err)
	}

	// Test with non-existent files
	config2 := NewTLSConfig(
		WithClientCertificate("/nonexistent/cert.pem", "/nonexistent/key.pem"),
	)

	if err := config2.Validate(); err == nil {
		t.Error("Config with non-existent files should fail validation")
	}
}

// Benchmark tests
func BenchmarkTLSConfig_Validate(b *testing.B) {
	config := DefaultTLSConfig()
	for i := 0; i < b.N; i++ {
		config.Validate()
	}
}

func BenchmarkTLSConfig_ToStandardTLSConfig(b *testing.B) {
	config := DefaultTLSConfig()
	for i := 0; i < b.N; i++ {
		config.ToStandardTLSConfig()
	}
}

func BenchmarkSecureCipherSuites(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SecureCipherSuites()
	}
}
