package unifi

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	// ErrInvalidTLSConfig is returned when TLS configuration is invalid
	ErrInvalidTLSConfig = errors.New("invalid TLS configuration")
	// ErrCertificateNotFound is returned when a certificate file cannot be found
	ErrCertificateNotFound = errors.New("certificate file not found")
	// ErrInsecureTLS is returned when insecure TLS settings are detected
	ErrInsecureTLS = errors.New("insecure TLS configuration detected")
)

// TLSConfig holds TLS configuration options
type TLSConfig struct {
	// InsecureSkipVerify controls whether a client verifies the server's certificate chain and host name
	InsecureSkipVerify bool

	// MinVersion specifies the minimum SSL/TLS version that is acceptable
	MinVersion uint16

	// MaxVersion specifies the maximum SSL/TLS version that is acceptable
	MaxVersion uint16

	// CipherSuites specifies the cipher suites to use
	CipherSuites []uint16

	// ClientCertFile is the path to client certificate file for mutual TLS
	ClientCertFile string

	// ClientKeyFile is the path to client private key file for mutual TLS
	ClientKeyFile string

	// RootCAFile is the path to root CA certificate file
	RootCAFile string

	// RootCAData contains root CA certificate data directly
	RootCAData []byte

	// ServerName is used to verify the hostname on the returned certificates
	ServerName string

	// Timeout for TLS handshake
	HandshakeTimeout time.Duration

	// Enable strict certificate validation
	StrictValidation bool
}

// DefaultTLSConfig returns a secure default TLS configuration
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
		CipherSuites:       SecureCipherSuites(),
		HandshakeTimeout:   10 * time.Second,
		StrictValidation:   true,
	}
}

// InsecureTLSConfig returns a configuration that skips verification (for development/testing only)
func InsecureTLSConfig() *TLSConfig {
	config := DefaultTLSConfig()
	config.InsecureSkipVerify = true
	config.StrictValidation = false
	return config
}

// SecureCipherSuites returns a list of secure cipher suites
func SecureCipherSuites() []uint16 {
	return []uint16{
		// TLS 1.3 cipher suites (preferred)
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_CHACHA20_POLY1305_SHA256,

		// TLS 1.2 cipher suites (secure fallback)
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	}
}

// Validate validates the TLS configuration and returns errors for insecure settings
func (cfg *TLSConfig) Validate() error {
	if cfg == nil {
		return fmt.Errorf("%w: configuration is nil", ErrInvalidTLSConfig)
	}

	// Check for insecure settings in strict mode
	if cfg.StrictValidation {
		if cfg.InsecureSkipVerify {
			return fmt.Errorf("%w: certificate verification disabled", ErrInsecureTLS)
		}

		if cfg.MinVersion < tls.VersionTLS12 {
			return fmt.Errorf("%w: minimum TLS version below 1.2", ErrInsecureTLS)
		}
	}

	// Validate file paths if provided
	if cfg.ClientCertFile != "" {
		if _, err := os.Stat(cfg.ClientCertFile); err != nil {
			return fmt.Errorf("%w: client certificate file %s: %v", ErrCertificateNotFound, cfg.ClientCertFile, err)
		}
	}

	if cfg.ClientKeyFile != "" {
		if _, err := os.Stat(cfg.ClientKeyFile); err != nil {
			return fmt.Errorf("%w: client key file %s: %v", ErrCertificateNotFound, cfg.ClientKeyFile, err)
		}
	}

	if cfg.RootCAFile != "" {
		if _, err := os.Stat(cfg.RootCAFile); err != nil {
			return fmt.Errorf("%w: root CA file %s: %v", ErrCertificateNotFound, cfg.RootCAFile, err)
		}
	}

	// Validate that if client cert is specified, key is also specified
	if (cfg.ClientCertFile != "") != (cfg.ClientKeyFile != "") {
		return fmt.Errorf("%w: client certificate and key must both be specified", ErrInvalidTLSConfig)
	}

	return nil
}

// ToStandardTLSConfig converts to Go's standard tls.Config
func (cfg *TLSConfig) ToStandardTLSConfig() (*tls.Config, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		MinVersion:         cfg.MinVersion,
		MaxVersion:         cfg.MaxVersion,
		CipherSuites:       cfg.CipherSuites,
		ServerName:         cfg.ServerName,
	}

	// Load client certificate if specified
	if cfg.ClientCertFile != "" && cfg.ClientKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCertFile, cfg.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("loading client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load root CA if specified
	if cfg.RootCAFile != "" || len(cfg.RootCAData) > 0 {
		rootCAs := x509.NewCertPool()

		if cfg.RootCAFile != "" {
			caCert, err := os.ReadFile(cfg.RootCAFile)
			if err != nil {
				return nil, fmt.Errorf("reading root CA file: %w", err)
			}
			if !rootCAs.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse root CA certificate from file: %s", cfg.RootCAFile)
			}
		}

		if len(cfg.RootCAData) > 0 {
			if !rootCAs.AppendCertsFromPEM(cfg.RootCAData) {
				return nil, fmt.Errorf("failed to parse root CA certificate from data")
			}
		}

		tlsConfig.RootCAs = rootCAs
	}

	return tlsConfig, nil
}

// CreateSecureTransport creates an HTTP transport with secure TLS configuration
func (cfg *TLSConfig) CreateSecureTransport() (*http.Transport, error) {
	tlsConfig, err := cfg.ToStandardTLSConfig()
	if err != nil {
		return nil, err
	}

	baseTransport := http.DefaultTransport.(*http.Transport).Clone()

	// Apply secure defaults
	baseTransport.TLSClientConfig = tlsConfig
	baseTransport.TLSHandshakeTimeout = cfg.HandshakeTimeout
	baseTransport.DisableCompression = false // Enable compression for efficiency
	baseTransport.ForceAttemptHTTP2 = true   // Prefer HTTP/2 when available

	// Security-focused timeouts
	baseTransport.ResponseHeaderTimeout = 30 * time.Second
	baseTransport.IdleConnTimeout = 90 * time.Second
	baseTransport.ExpectContinueTimeout = 1 * time.Second

	return baseTransport, nil
}

// TLSConfigOption defines a function for configuring TLS settings
type TLSConfigOption func(*TLSConfig)

// WithInsecureSkipVerify allows skipping certificate verification (not recommended for production)
func WithInsecureSkipVerify(skip bool) TLSConfigOption {
	return func(cfg *TLSConfig) {
		cfg.InsecureSkipVerify = skip
		if skip {
			cfg.StrictValidation = false
		}
	}
}

// WithMinTLSVersion sets the minimum TLS version
func WithMinTLSVersion(version uint16) TLSConfigOption {
	return func(cfg *TLSConfig) {
		cfg.MinVersion = version
	}
}

// WithMaxTLSVersion sets the maximum TLS version
func WithMaxTLSVersion(version uint16) TLSConfigOption {
	return func(cfg *TLSConfig) {
		cfg.MaxVersion = version
	}
}

// WithClientCertificate sets client certificate files for mutual TLS
func WithClientCertificate(certFile, keyFile string) TLSConfigOption {
	return func(cfg *TLSConfig) {
		cfg.ClientCertFile = certFile
		cfg.ClientKeyFile = keyFile
	}
}

// WithRootCA sets the root CA certificate file
func WithRootCA(caFile string) TLSConfigOption {
	return func(cfg *TLSConfig) {
		cfg.RootCAFile = caFile
	}
}

// WithRootCAData sets the root CA certificate data directly
func WithRootCAData(caData []byte) TLSConfigOption {
	return func(cfg *TLSConfig) {
		cfg.RootCAData = caData
	}
}

// WithServerName sets the server name for certificate verification
func WithServerName(serverName string) TLSConfigOption {
	return func(cfg *TLSConfig) {
		cfg.ServerName = serverName
	}
}

// WithHandshakeTimeout sets the TLS handshake timeout
func WithHandshakeTimeout(timeout time.Duration) TLSConfigOption {
	return func(cfg *TLSConfig) {
		cfg.HandshakeTimeout = timeout
	}
}

// WithStrictValidation enables or disables strict TLS validation
func WithStrictValidation(strict bool) TLSConfigOption {
	return func(cfg *TLSConfig) {
		cfg.StrictValidation = strict
	}
}

// WithCipherSuites sets custom cipher suites
func WithCipherSuites(suites []uint16) TLSConfigOption {
	return func(cfg *TLSConfig) {
		cfg.CipherSuites = suites
	}
}

// NewTLSConfig creates a new TLS configuration with the given options
func NewTLSConfig(opts ...TLSConfigOption) *TLSConfig {
	cfg := DefaultTLSConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// TLSVersionString returns a human-readable string for TLS version constants
func TLSVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

// CipherSuiteName returns a human-readable name for cipher suite constants
func CipherSuiteName(cs uint16) string {
	switch cs {
	case tls.TLS_AES_128_GCM_SHA256:
		return "TLS_AES_128_GCM_SHA256"
	case tls.TLS_AES_256_GCM_SHA384:
		return "TLS_AES_256_GCM_SHA384"
	case tls.TLS_CHACHA20_POLY1305_SHA256:
		return "TLS_CHACHA20_POLY1305_SHA256"
	case tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:
		return "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"
	case tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:
		return "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"
	case tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:
		return "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
	case tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:
		return "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
	case tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305:
		return "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305"
	case tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305:
		return "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", cs)
	}
}

// ValidateEndpointTLS validates that an endpoint supports secure TLS
func ValidateEndpointTLS(endpoint string, tlsConfig *TLSConfig) error {
	if !strings.HasPrefix(endpoint, "https://") {
		return fmt.Errorf("endpoint must use HTTPS for secure communication: %s", endpoint)
	}

	// Create a temporary client to test TLS connection
	transport, err := tlsConfig.CreateSecureTransport()
	if err != nil {
		return fmt.Errorf("creating secure transport: %w", err)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   tlsConfig.HandshakeTimeout + 5*time.Second,
	}

	// Test connection with HEAD request to avoid unnecessary data transfer
	resp, err := client.Head(endpoint)
	if err != nil {
		return fmt.Errorf("TLS connection test failed: %w", err)
	}
	resp.Body.Close()

	return nil
}
