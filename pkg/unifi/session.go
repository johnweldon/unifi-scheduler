// Package unifi provides a comprehensive client library for interacting with UniFi Network controllers.
//
// This package implements a session-based client that handles authentication, network operations,
// and data retrieval from UniFi controllers. It supports secure credential management, TLS
// configuration, circuit breaker patterns, and retry logic for robust network communication.
//
// Key features:
//   - Session-based authentication with automatic login
//   - Secure credential management with multiple input methods
//   - Comprehensive TLS configuration including mutual TLS
//   - Circuit breaker pattern for resilient network operations
//   - Exponential backoff retry logic
//   - Client, device, and event management
//   - Support for both active and historical data
//
// Basic usage:
//
//	// Create and initialize a session
//	session := &unifi.Session{
//	    Endpoint: "https://controller.example.com",
//	}
//
//	creds, err := unifi.NewCredentials("admin", "password")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	err = session.Initialize(
//	    unifi.WithCredentials(creds),
//	    unifi.WithTLSConfig(unifi.DefaultTLSConfig()),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get connected clients
//	clients, err := session.GetClients()
//	if err != nil {
//	    log.Fatal(err)
//	}
package unifi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/jw4/x/stringset"
	"github.com/jw4/x/transport"
)

// Session represents a persistent connection to a UniFi Network controller.
//
// A Session manages the authentication state, HTTP client configuration, and
// operational parameters for communicating with a UniFi controller. It provides
// a high-level interface for performing network operations while handling
// authentication, retries, and error recovery automatically.
//
// The Session is designed to be long-lived and reusable across multiple operations.
// It maintains cookies for authentication, handles CSRF tokens, and provides
// circuit breaker functionality for resilient network communication.
//
// Key responsibilities:
//   - Authentication and session management
//   - HTTP client configuration with TLS support
//   - Request/response handling with retries
//   - Error handling and recovery
//   - Output and logging management
//
// Example:
//
//	session := &unifi.Session{
//	    Endpoint: "https://controller.example.com",
//	}
//
//	err := session.Initialize(
//	    unifi.WithCredentials(creds),
//	    unifi.WithHTTPTimeout(30*time.Second),
//	)
type Session struct {
	// Endpoint is the base URL of the UniFi controller (e.g., "https://controller.example.com")
	Endpoint string

	// Username for authentication (deprecated: use Credentials instead)
	// Deprecated: Use the Credentials field for secure credential management
	Username string

	// Password for authentication (deprecated: use Credentials instead)
	// Deprecated: Use the Credentials field for secure credential management
	Password string

	// Credentials provides secure storage and management of authentication data
	Credentials *Credentials

	// Internal session state
	csrf   string                 // CSRF token for request authentication
	client *http.Client           // HTTP client for controller communication
	login  func() (string, error) // Login function for authentication
	err    error                  // Last error encountered during operations

	// Controller configuration
	nonUDMPro bool   // Flag indicating non-UDM Pro controller type
	site      string // Site identifier for multi-site controllers

	// Output and logging configuration
	outWriter io.Writer // Writer for standard output
	errWriter io.Writer // Writer for error output
	dbgWriter io.Writer // Writer for debug output

	// Network and reliability configuration
	httpTimeout    time.Duration   // Timeout for HTTP requests
	circuitBreaker *CircuitBreaker // Circuit breaker for resilient operations
	pathValidator  PathValidator   // Validator for API endpoint paths
	tlsConfig      *TLSConfig      // TLS configuration for secure connections
	backoffRetry   *BackoffRetry   // Retry logic with exponential backoff

	// Internal state management
	initialized bool // Flag to track whether the session has been initialized
}

// Option represents a configuration function for customizing Session behavior.
//
// Options follow the functional options pattern, allowing callers to configure
// specific aspects of a Session during initialization. This provides a clean,
// extensible API for session configuration without requiring large constructor
// parameter lists.
//
// Example usage:
//
//	session.Initialize(
//	    WithCredentials(creds),
//	    WithHTTPTimeout(30*time.Second),
//	    WithTLSConfig(tlsConfig),
//	)
type Option func(*Session)

// WithOut configures the standard output writer for the session.
// This writer will receive normal operational output from the session.
func WithOut(o io.Writer) Option { return func(s *Session) { s.outWriter = o } }

// WithErr configures the error output writer for the session.
// This writer will receive error messages and warnings from the session.
func WithErr(e io.Writer) Option { return func(s *Session) { s.errWriter = e } }

// WithDbg configures the debug output writer for the session.
// This writer will receive detailed debugging information when debug mode is enabled.
// Debug output includes HTTP request/response details and internal state information.
func WithDbg(d io.Writer) Option { return func(s *Session) { s.dbgWriter = d } }

// WithHTTPTimeout configures the timeout for HTTP requests made by the session.
// This timeout applies to individual HTTP operations, not the overall session lifetime.
// A reasonable default is 30 seconds for most UniFi controller operations.
func WithHTTPTimeout(t time.Duration) Option { return func(s *Session) { s.httpTimeout = t } }

// WithCircuitBreakerConfig configures the circuit breaker for resilient network operations.
//
// The circuit breaker helps protect against cascading failures by temporarily
// failing fast when the UniFi controller is experiencing issues. It automatically
// reopens when the controller becomes healthy again.
//
// Parameters:
//   - config: Circuit breaker configuration including failure thresholds and timeouts
func WithCircuitBreakerConfig(config CircuitBreakerConfig) Option {
	return func(s *Session) {
		s.circuitBreaker = NewCircuitBreaker(config)
	}
}

// WithBackoffConfig configures the exponential backoff retry logic for failed operations.
//
// The backoff retry mechanism automatically retries failed operations with
// increasing delays between attempts. This helps handle temporary network
// issues and controller overload gracefully.
//
// Parameters:
//   - config: Backoff configuration including max retries, initial delay, and backoff multiplier
func WithBackoffConfig(config BackoffConfig) Option {
	return func(s *Session) {
		s.backoffRetry = NewBackoffRetry(config)
	}
}

// WithPathValidator configures a custom validator for API endpoint paths.
//
// The path validator provides security by validating that API requests only
// access authorized endpoints. This helps prevent path traversal attacks
// and ensures requests stay within the expected API boundaries.
//
// Parameters:
//   - validator: Path validator implementation to use for endpoint validation
func WithPathValidator(validator PathValidator) Option {
	return func(s *Session) {
		s.pathValidator = validator
	}
}

// WithTLSConfig configures custom TLS settings for secure controller communication.
//
// This option allows fine-grained control over TLS behavior, including:
//   - Certificate verification settings
//   - Minimum and maximum TLS versions
//   - Custom root CA certificates
//   - Client certificate authentication (mutual TLS)
//   - Cipher suite selection
//
// Parameters:
//   - config: TLS configuration specifying security requirements and certificates
func WithTLSConfig(config *TLSConfig) Option {
	return func(s *Session) {
		s.tlsConfig = config
	}
}

// WithInsecureTLS disables TLS certificate verification (for development only).
//
// WARNING: This option disables certificate verification and should NEVER be used
// in production environments. It makes the connection vulnerable to man-in-the-middle
// attacks. Only use this for development or testing with self-signed certificates.
//
// For production use with self-signed certificates, use WithTLSConfig with a
// custom root CA instead.
func WithInsecureTLS() Option {
	return func(s *Session) {
		s.tlsConfig = InsecureTLSConfig()
	}
}

// CircuitBreakerStats returns the current statistics for the session's circuit breaker.
//
// Circuit breaker statistics provide insight into the reliability of controller
// communication, including failure rates, current state, and timing information.
// This is useful for monitoring and debugging network reliability issues.
//
// Returns default statistics if no circuit breaker is configured.
func (s *Session) CircuitBreakerStats() CircuitBreakerStats {
	if s.circuitBreaker == nil {
		return CircuitBreakerStats{State: CircuitBreakerClosed}
	}
	return s.circuitBreaker.Stats()
}

// BackoffStats returns the current statistics for the session's retry backoff mechanism.
//
// Backoff statistics provide information about retry attempts, success rates,
// and timing patterns. This is valuable for understanding how often operations
// need to be retried and tuning backoff parameters.
//
// Returns default statistics if no backoff retry is configured.
func (s *Session) BackoffStats() BackoffStats {
	if s.backoffRetry == nil {
		return BackoffStats{Strategy: BackoffExponential}
	}
	return s.backoffRetry.Stats()
}

// WithCredentials configures secure credential storage for session authentication.
//
// This is the recommended way to provide authentication credentials to a session.
// The Credentials type provides secure storage and automatic memory clearing
// to protect sensitive authentication data.
//
// Parameters:
//   - creds: Secure credentials object containing username and password
func WithCredentials(creds *Credentials) Option {
	return func(s *Session) { s.Credentials = creds }
}

// WithSecureAuth creates secure credentials from a username and password.
//
// This is a convenience function that creates a Credentials object from
// plain text username and password. The credentials are automatically
// secured in memory and will be cleared when no longer needed.
//
// Parameters:
//   - username: UniFi controller username
//   - password: UniFi controller password
//
// Note: If credential creation fails, the session will have an error set
// that will be returned during initialization.
func WithSecureAuth(username, password string) Option {
	return func(s *Session) {
		creds, err := NewCredentials(username, password)
		if err != nil {
			s.setError(fmt.Errorf("failed to create secure credentials: %w", err))
			return
		}
		s.Credentials = creds
	}
}

// Initialize prepares the session for communication with the UniFi controller.
//
// This method configures the session with default values, applies the provided
// options, validates the configuration, and sets up the HTTP client for secure
// communication. It must be called before any other session operations.
//
// The method is idempotent - calling it multiple times will not override existing
// configuration, but will apply any new options provided. This allows safe
// reconfiguration and use in health check scenarios.
//
// Initialization process:
//  1. Sets default values for unconfigured fields (first call only)
//  2. Applies all provided configuration options
//  3. Validates credentials and endpoint configuration
//  4. Performs TLS endpoint validation
//  5. Creates and configures the HTTP client
//  6. Sets up authentication mechanisms
//
// Parameters:
//   - options: Zero or more Option functions to configure the session
//
// Returns an error if:
//   - The session is nil
//   - Required configuration is missing (endpoint, credentials)
//   - TLS configuration is invalid
//   - HTTP client creation fails
//   - Endpoint TLS validation fails
//
// Example:
//
//	session := &unifi.Session{Endpoint: "https://controller.example.com"}
//	err := session.Initialize(
//	    unifi.WithCredentials(creds),
//	    unifi.WithHTTPTimeout(30*time.Second),
//	    unifi.WithTLSConfig(tlsConfig),
//	)
//	if err != nil {
//	    log.Fatalf("Failed to initialize session: %v", err)
//	}
func (s *Session) Initialize(options ...Option) error {
	if s == nil {
		return ErrNilSession
	}

	// Only set defaults if this is the first initialization
	if !s.initialized {
		if s.outWriter == nil {
			s.outWriter = os.Stdout
		}
		if s.errWriter == nil {
			s.errWriter = os.Stderr
		}
		if s.httpTimeout == 0 {
			s.httpTimeout = 2 * time.Minute // Default timeout
		}
		if s.circuitBreaker == nil {
			s.circuitBreaker = NewCircuitBreaker(DefaultCircuitBreakerConfig())
		}
		if s.pathValidator == nil {
			s.pathValidator = NewDefaultPathValidator()
		}
		if s.tlsConfig == nil {
			s.tlsConfig = DefaultTLSConfig()
		}
		if s.backoffRetry == nil {
			s.backoffRetry = NewBackoffRetry(UniFiRetryConfig())
		}
		s.initialized = true
	}

	// Always apply new options (allows reconfiguration)
	for _, option := range options {
		option(s)
	}

	s.err = nil

	if len(s.Endpoint) == 0 {
		s.setErrorString("missing endpoint")
	}

	// Handle both legacy and secure credential formats
	if s.Credentials != nil {
		// Validate secure credentials
		if err := s.Credentials.Validate(); err != nil {
			s.setError(fmt.Errorf("invalid credentials: %w", err))
		}
	} else if len(s.Username) > 0 && len(s.Password) > 0 {
		// Migrate from legacy format to secure credentials
		creds, err := NewCredentials(s.Username, s.Password)
		if err != nil {
			s.setError(fmt.Errorf("failed to create secure credentials: %w", err))
		} else {
			s.Credentials = creds
			// Clear legacy password from memory
			s.Password = ""
		}
	} else {
		s.setErrorString("missing credentials")
	}

	// Validate endpoint supports HTTPS
	if err := ValidateEndpointTLS(s.Endpoint, s.tlsConfig); err != nil {
		s.setError(fmt.Errorf("endpoint TLS validation failed: %w", err))
	}

	// Create or update HTTP client if needed
	if s.client == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			s.setError(err)
		}

		// Create secure transport
		secureTransport, err := s.tlsConfig.CreateSecureTransport()
		if err != nil {
			s.setError(fmt.Errorf("creating secure transport: %w", err))
		}

		// Wrap with logging if debug is enabled
		var finalTransport http.RoundTripper = secureTransport
		if s.dbgWriter != nil {
			finalTransport = transport.NewLoggingTransport(secureTransport, transport.LoggingOutput(s.dbgWriter))
		}

		s.client = &http.Client{ // nolint:exhaustivestruct
			Jar:       jar,
			Timeout:   s.httpTimeout,
			Transport: finalTransport,
		}
	}

	if s.login == nil {
		s.login = s.webLogin
	}

	return s.err
}

// Login performs authentication with the UniFi controller and establishes a session.
//
// This method authenticates with the controller using the configured credentials
// and stores the necessary authentication tokens (cookies, CSRF tokens) for
// subsequent API requests. The session must be initialized before calling Login.
//
// The authentication process includes:
//   - Sending credentials to the controller's login endpoint
//   - Storing authentication cookies for session persistence
//   - Retrieving and storing CSRF tokens for API security
//   - Validating the authentication response
//
// Returns:
//   - string: Success message from the controller (usually empty for successful logins)
//   - error: Authentication error if login fails
//
// Common errors:
//   - ErrUninitializedSession: Session not initialized before login attempt
//   - Network errors: Connection issues with the controller
//   - Authentication errors: Invalid credentials or controller issues
//
// Example:
//
//	msg, err := session.Login()
//	if err != nil {
//	    log.Fatalf("Authentication failed: %v", err)
//	}
func (s *Session) Login() (string, error) {
	if s.login == nil {
		s.login = func() (string, error) {
			return "", ErrUninitializedSession
		}
	}

	return s.login()
}

// GetDevices retrieves all network devices managed by the UniFi controller.
//
// This method fetches information about all UniFi devices (access points, switches,
// gateways, etc.) connected to the controller. The devices are automatically sorted
// using the default device sorting criteria.
//
// Device information includes:
//   - Device identification (name, model, MAC address)
//   - Status and health information
//   - Network configuration and statistics
//   - Firmware version and adoption status
//   - Performance metrics and uptime
//
// Returns:
//   - []Device: Slice of device objects with complete device information
//   - error: Error if device retrieval fails
//
// The returned devices are sorted by the default criteria (typically by name).
// Use the Device.Sort methods for custom sorting if needed.
//
// Example:
//
//	devices, err := session.GetDevices()
//	if err != nil {
//	    log.Fatalf("Failed to get devices: %v", err)
//	}
//
//	for _, device := range devices {
//	    fmt.Printf("Device: %s (%s)\n", device.Name, device.Model)
//	}
func (s *Session) GetDevices() ([]Device, error) {
	var (
		devices []Device
		dmap    map[string]Device

		err error
	)

	if dmap, err = s.getDevices(); err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}

	for _, device := range dmap {
		devices = append(devices, device)
	}

	DeviceDefault.Sort(devices)

	return devices, nil
}

// GetClients retrieves currently connected network clients from the UniFi controller.
//
// This method fetches information about active clients currently connected to the
// network. It returns only clients that are currently online and associated with
// access points or switches managed by the controller.
//
// Client information includes:
//   - Device identification (hostname, MAC address, device name)
//   - Network information (IP address, connection type, speed)
//   - Connection statistics (data usage, connection time)
//   - Device classification and vendor information
//   - Quality metrics (signal strength for wireless clients)
//
// Parameters:
//   - filters: Optional ClientFilter functions to filter the results
//
// Returns:
//   - []Client: Slice of active client objects
//   - error: Error if client retrieval fails
//
// Example:
//
//	clients, err := session.GetClients()
//	if err != nil {
//	    log.Fatalf("Failed to get clients: %v", err)
//	}
//
//	fmt.Printf("Found %d active clients\n", len(clients))
//	for _, client := range clients {
//	    fmt.Printf("Client: %s at %s\n", client.Hostname, client.IP)
//	}
func (s *Session) GetClients(filters ...ClientFilter) ([]Client, error) {
	return s.getClients(false, filters...)
}

// GetAllClients retrieves all known network clients from the UniFi controller.
//
// This method fetches information about all clients that the controller has ever
// seen, including both currently connected clients and historical/offline clients.
// This provides a complete view of all devices that have connected to the network.
//
// The returned data includes the same information as GetClients, but covers:
//   - Currently connected clients
//   - Recently disconnected clients
//   - Historical client records
//   - Blocked or restricted clients
//
// Parameters:
//   - filters: Optional ClientFilter functions to filter the results
//
// Returns:
//   - []Client: Slice of all known client objects
//   - error: Error if client retrieval fails
//
// Note: This method may return a large dataset on busy networks with many
// historical client connections. Consider using GetClients for real-time
// monitoring or apply filters to limit the results.
//
// Example:
//
//	allClients, err := session.GetAllClients()
//	if err != nil {
//	    log.Fatalf("Failed to get all clients: %v", err)
//	}
//
//	active := 0
//	for _, client := range allClients {
//	    if client.IsConnected() {
//	        active++
//	    }
//	}
//	fmt.Printf("Total clients: %d, Active: %d\n", len(allClients), active)
func (s *Session) GetAllClients(filters ...ClientFilter) ([]Client, error) {
	return s.getClients(true, filters...)
}

// GetAllEvents returns all events.
func (s *Session) GetAllEvents() ([]Event, error) {
	return s.getEvents(true)
}

// GetRecentEvents returns a list of "recent" events.
func (s *Session) GetRecentEvents() ([]Event, error) {
	return s.getEvents(false)
}

// GetMACs returns all known MAC addresses, and the associated names.
func (s *Session) GetMACs() (map[MAC][]string, error) {
	var (
		macs = map[MAC]*stringset.OrderedStringSet{}

		devices []Device
		users   []Client

		err error
	)

	if devices, err = s.GetDevices(); err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}

	for _, device := range devices {
		for _, name := range []string{
			device.Name,
			device.MAC.String(),
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := macs[device.MAC]; !ok {
				macs[device.MAC] = &stringset.OrderedStringSet{}
			}

			macs[device.MAC].Add(name)
		}
	}

	if users, err = s.GetAllClients(); err != nil {
		return nil, fmt.Errorf("getting users: %w", err)
	}

	for _, user := range users {
		for _, name := range []string{
			user.Name,
			user.Hostname,
			user.MAC.String(),
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := macs[user.MAC]; !ok {
				macs[user.MAC] = &stringset.OrderedStringSet{}
			}

			macs[user.MAC].Add(name)
		}
	}

	ret := map[MAC][]string{}
	for mac, m := range macs {
		ret[mac] = m.Values()
	}

	return ret, nil
}

// GetNames returns all known names, and the associated MAC addresses.
func (s *Session) GetNames() (map[string][]MAC, error) { // nolint:funlen
	var (
		names = map[string]*stringset.OrderedStringSet{}

		devices []Device
		clients []Client
		users   []Client

		err error
	)

	if devices, err = s.GetDevices(); err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}

	for _, device := range devices {
		for _, name := range []string{
			device.Name,
			string(device.IP),
			string(device.MAC),
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := names[name]; !ok {
				names[name] = &stringset.OrderedStringSet{}
			}

			names[name].Add(device.MAC.String())
		}
	}

	if clients, err = s.GetClients(); err != nil {
		return nil, fmt.Errorf("getting users: %w", err)
	}

	if users, err = s.GetAllClients(); err != nil {
		return nil, fmt.Errorf("getting users: %w", err)
	}

	for _, user := range append(clients, users...) {
		for _, name := range []string{
			user.Name,
			user.Hostname,
			user.DeviceName,
			string(user.IP),
			string(user.FixedIP),
			user.MAC.String(),
		} {
			if len(name) == 0 {
				continue
			}

			if _, ok := names[name]; !ok {
				names[name] = &stringset.OrderedStringSet{}
			}

			names[name].Add(user.MAC.String())
		}
	}

	ret := map[string][]MAC{}
	for name, m := range names {
		vals := m.Values()
		macs := make([]MAC, len(vals))
		for ix, val := range vals {
			macs[ix] = MAC(val)
		}

		ret[name] = macs
	}

	return ret, nil
}

func (s *Session) GetMACsBy(ids ...string) ([]MAC, error) {
	var (
		err     error
		allMACs []MAC
		names   map[string][]MAC
	)

	if names, err = s.GetNames(); err != nil {
		return nil, err
	}

	for _, id := range ids {
		if macs, ok := names[id]; ok {
			allMACs = append(allMACs, macs...)
		}
	}

	return allMACs, nil
}

// Raw executes arbitrary endpoints.
func (s *Session) Raw(method, path string, body io.Reader) (string, error) {
	// Validate HTTP method
	if err := ValidateHTTPMethod(method); err != nil {
		return "", fmt.Errorf("raw command validation failed: %w", err)
	}

	// Sanitize and validate path
	sanitizedPath := SanitizeInput(path)
	if err := s.pathValidator.ValidatePath(sanitizedPath); err != nil {
		return "", fmt.Errorf("raw command path validation failed: %w", err)
	}

	// Validate payload if present
	if body != nil {
		// Read the body to validate it
		var bodyBytes []byte
		var err error
		if bodyBytes, err = io.ReadAll(body); err != nil {
			return "", fmt.Errorf("raw command failed to read body: %w", err)
		}

		if err := ValidatePayload(bodyBytes); err != nil {
			return "", fmt.Errorf("raw command payload validation failed: %w", err)
		}

		// Recreate the reader with validated content
		body = bytes.NewReader(bodyBytes)
	}

	return s.action(method, sanitizedPath, body)
}

// ListEvents describes the latest events.
func (s *Session) ListEvents() (string, error) { return s.action(http.MethodGet, "/stat/event", nil) }

// ListAllEvents describes all events.
func (s *Session) ListAllEvents() (string, error) {
	return s.action(http.MethodGet, "/rest/event", nil)
}

// ListUsers describes the known UniFi clients.
func (s *Session) ListUsers() (string, error) { return s.action(http.MethodGet, "/rest/user", nil) }

// GetUser returns user info.
func (s *Session) GetUser(id string) (string, error) {
	return s.action(http.MethodGet, "/rest/user/"+id, nil)
}

// GetUserByMAC returns user info.
func (s *Session) GetUserByMAC(mac string) (string, error) {
	return s.action(http.MethodGet, "/rest/user/?mac="+mac, nil)
}

// SetUserDetails configures a friendly name and static ip assignation
// for a given MAC address.
func (s *Session) SetUserDetails(mac, name, ip string) (string, error) {
	user, err := s.getUserByMac(mac)
	if err != nil {
		return "", err
	}

	return s.setUserDetails(user.ID, name, ip)
}

// ListClients describes currently connected clients.
func (s *Session) ListClients() (string, error) { return s.action(http.MethodGet, "/stat/sta", nil) }

// ListDevices describes currently connected clients.
func (s *Session) ListDevices() (string, error) { return s.action(http.MethodGet, "/stat/device", nil) }

// Kick disconnects a connected client, identified by MAC address.
func (s *Session) Kick(macs ...MAC) (string, error) { return s.macsAction("kick-sta", macs) }

// Block prevents a specific client (identified by MAC) from connecting
// to the UniFi network.
func (s *Session) Block(macs ...MAC) (string, error) { return s.macsAction("block-sta", macs) }

// Unblock re-enables a specific client.
func (s *Session) Unblock(macs ...MAC) (string, error) { return s.macsAction("unblock-sta", macs) }

// Forget removes record of a specific list of MAC addresses.
func (s *Session) Forget(macs ...MAC) (string, error) { return s.macsAction("forget-sta", macs) }

// KickFn uses Clients to find MAC addresses to Kick.
func (s *Session) KickFn(clients []Client, keys map[string]bool) {
	s.clientsFn(s.Kick, keys, clients...)
}

// BlockFn uses Clients to find MAC addresses to Block.
func (s *Session) BlockFn(clients []Client, keys map[string]bool) {
	s.clientsFn(s.Block, keys, clients...)
}

// UnblockFn uses Clients to find MAC addresses to Unblock.
func (s *Session) UnblockFn(clients []Client, keys map[string]bool) {
	s.clientsFn(s.Unblock, keys, clients...)
}

// ForgetFn uses Clients to find MAC addresses to Forget.
func (s *Session) ForgetFn(clients []Client, keys map[string]bool) {
	s.clientsFn(s.Forget, keys, clients...)
}

// getUserByMac looks up a Client by the MAC address.
func (s *Session) getUserByMac(mac string) (*Client, error) {
	var (
		err  error
		data string
		resp ClientResponse
	)

	if data, err = s.GetUserByMAC(mac); err != nil {
		return nil, fmt.Errorf("retrieving user by mac: %w", err)
	}

	if err = json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, fmt.Errorf("unmarshalling user: %w", err)
	}

	if len(resp.Data) < 1 {
		return nil, fmt.Errorf("zero results: %s", data)
	}

	return &resp.Data[0], nil
}

type ClientFilter func(Client) bool

func Not(filter ClientFilter) ClientFilter { return func(c Client) bool { return !filter(c) } }

func Blocked(c Client) bool    { return c.IsBlocked }
func Authorized(c Client) bool { return c.IsAuthorized }
func Guest(c Client) bool      { return c.IsGuest }
func Wired(c Client) bool      { return c.IsWired }

func passAll(client Client, filters ...ClientFilter) bool {
	for _, filter := range filters {
		if !filter(client) {
			return false
		}
	}

	return true
}

// getClients returns a list of clients.  If all is false, only the active
// clients will be returned, otherwise all the known clients will be returned.
func (s *Session) getClients(all bool, filters ...ClientFilter) ([]Client, error) {
	var (
		devices map[string]Device

		clientsJSON string
		clients     []Client
		cresp       ClientResponse

		err error
	)

	sorter := ClientDefault
	fetch := s.ListClients

	if all {
		sorter = ClientHistorical
		fetch = s.ListUsers
	}

	if devices, err = s.getDevices(); err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}

	if clientsJSON, err = fetch(); err != nil {
		return nil, fmt.Errorf("listing clients: %w", err)
	}

	if err = json.Unmarshal([]byte(clientsJSON), &cresp); err != nil {
		return nil, fmt.Errorf("unmarshalling clients: %w", err)
	}

	for _, client := range cresp.Data {
		if dev, ok := devices[client.UpstreamMAC()]; ok {
			client.UpstreamName = dev.Name
		}

		if passAll(client, filters...) {
			clients = append(clients, client)
		}
	}

	sorter.Sort(clients)

	return clients, nil
}

// getDevices returns all known devices mapped by name.
func (s *Session) getDevices() (map[string]Device, error) {
	var (
		devicesJSON string
		devices     = map[string]Device{}
		dresp       DeviceResponse

		err error
	)

	if devicesJSON, err = s.ListDevices(); err != nil {
		return nil, fmt.Errorf("listing devices: %w", err)
	}

	if err = json.Unmarshal([]byte(devicesJSON), &dresp); err != nil {
		return nil, fmt.Errorf("unmarshalling devices: %w", err)
	}

	for _, device := range dresp.Data {
		devices[device.MAC.String()] = device
	}

	return devices, nil
}

// getEvents returns a list of events. If all is true, then all known events
// will be returned, otherwise only the most recent ones will be returned.
func (s *Session) getEvents(all bool) ([]Event, error) {
	var (
		eventsJSON string
		eresp      EventResponse

		err error
	)

	fetch := s.ListEvents
	if all {
		fetch = s.ListAllEvents
	}

	if eventsJSON, err = fetch(); err != nil {
		return nil, fmt.Errorf("fetching events: %w", err)
	}

	if err = json.Unmarshal([]byte(eventsJSON), &eresp); err != nil {
		return nil, fmt.Errorf("unmarshalling events: %w", err)
	}

	events := eresp.Data

	DefaultEventSort.Sort(events)

	return events, nil
}

// webLogin performs the authentication for this session.
func (s *Session) webLogin() (string, error) {
	if s.err != nil {
		return "", s.err
	}

	if s.Credentials == nil {
		return "", fmt.Errorf("no credentials available for login")
	}

	u, err := url.Parse(fmt.Sprintf("%s/api/auth/login", s.Endpoint))
	if err != nil {
		s.setError(err)
		return "", s.err
	}

	// Build secure payload
	password := s.Credentials.Password.String()
	payload := fmt.Sprintf(`{"username":%q,"password":%q,"strict":"true","remember":"true"}`, s.Credentials.Username, password)

	// Clear password from memory immediately after use
	clearString(password)

	respBody, err := s.post(u, bytes.NewBufferString(payload))

	// Clear payload from memory
	clearString(payload)

	if err == nil {
		s.login = func() (string, error) { return respBody, nil }
	}

	return respBody, err
}

// buildURL generates the endpoint URL relevant to the configured
// version of UniFi.
func (s *Session) buildURL(path string) (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}

	pathPrefix := "/proxy/network"
	if s.nonUDMPro {
		pathPrefix = ""
	}

	site := "default"
	if len(s.site) > 0 {
		site = s.site
	}

	return url.Parse(fmt.Sprintf("%s%s/api/s/%s%s", s.Endpoint, pathPrefix, site, path))
}

// macAction applies an action to a single MAC.
func (s *Session) macAction(action string, mac MAC) (string, error) {
	payload := fmt.Sprintf(`{"cmd":%q,"mac":%q}`, action, mac)

	return s.action(http.MethodPost, "/cmd/stamgr", bytes.NewBufferString(payload))
}

// macsAction applies a function to multiple MACs.
func (s *Session) macsAction(action string, macs []MAC) (string, error) {
	if len(macs) == 0 {
		return "", nil
	}

	var allmacs []string
	for _, mac := range macs {
		allmacs = append(allmacs, fmt.Sprintf("%q", mac))
	}

	payload := fmt.Sprintf(`{"cmd":%q,"macs":[%s]}`, action, strings.Join(allmacs, ","))

	return s.action(http.MethodPost, "/cmd/stamgr", bytes.NewBufferString(payload))
}

func (s *Session) clientsFn(action func(...MAC) (string, error), keys map[string]bool, clients ...Client) {
	var macs []MAC
	for _, client := range clients {
		display := firstNonEmpty(client.Name, client.Hostname)
		if k, ok := keys[display]; ok && k {
			macs = append(macs, client.MAC)
		}
	}

	res, err := action(macs...)
	if err != nil {
		fmt.Fprintf(s.errWriter, "%s\nerror: %v\n", res, err)

		return
	}

	fmt.Fprintf(s.outWriter, "%s\n", res)
}

func (s *Session) setUserDetails(id, name, ip string) (string, error) {
	if len(id) == 0 {
		return "", fmt.Errorf("missing user id")
	}

	tmpl, err := template.New("").Parse(`{ {{- /**/ -}}
  "local_dns_record_enabled":false,{{- /**/ -}}
  "local_dns_record":"",{{- /**/ -}}
  "name":"{{ with .Name }}{{ . }}{{ end }}",{{- /**/ -}}
  "usergroup_id":"{{ with .UsergroupID }}{{ . }}{{ end }}",{{- /**/ -}}
  "use_fixedip":{{ with .IP }}true{{ else }}false{{ end }},{{- /**/ -}}
  "network_id":"{{ with .NetworkID }}{{ . }}{{ end }}",{{- /**/ -}}
  "fixed_ip":"{{ with .IP }}{{ . }}{{ end }}"{{- /**/ -}}
}`)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	if err = tmpl.Execute(&buf, map[string]string{"Name": name, "IP": ip, "NetworkID": "5c82f1ce2679fb00116fb58e"}); err != nil {
		return "", err
	}

	return s.action(http.MethodPut, "/rest/user/"+id, &buf)
}

func (s *Session) action(method, path string, body io.Reader) (string, error) {
	if s.err != nil {
		return "", s.err
	}

	u, err := s.buildURL(path)
	if err != nil {
		s.setError(err)

		return "", s.err
	}

	switch method {
	case http.MethodGet:
		return s.get(u)
	case http.MethodPost:
		return s.post(u, body)
	case http.MethodPut:
		return s.put(u, body)
	default:
		return "", fmt.Errorf("unconfigured method: %q", method)
	}
}

func (s *Session) get(u fmt.Stringer) (string, error) {
	return s.verb("GET", u, nil)
}

func (s *Session) post(u fmt.Stringer, body io.Reader) (string, error) {
	return s.verb("POST", u, body)
}

func (s *Session) put(u fmt.Stringer, body io.Reader) (string, error) {
	return s.verb("PUT", u, body)
}

func (s *Session) verb(verb string, u fmt.Stringer, body io.Reader) (string, error) {
	var result string
	var responseError error

	// Create a context for the entire operation with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.httpTimeout)
	defer cancel()

	// Execute the HTTP request through backoff retry which wraps the circuit breaker
	err := s.backoffRetry.Retry(ctx, func() error {
		// Copy the body if it's provided, since it may need to be read multiple times on retry
		var requestBody io.Reader
		if body != nil {
			if bodyBytes, readErr := io.ReadAll(body); readErr == nil {
				requestBody = bytes.NewReader(bodyBytes)
			} else {
				return fmt.Errorf("failed to read request body: %w", readErr)
			}
		}

		// Execute the HTTP request through the circuit breaker
		return s.circuitBreaker.Execute(func() error {
			req, err := http.NewRequestWithContext(ctx, verb, u.String(), requestBody)
			if err != nil {
				return err
			}

			req.Header.Set("User-Agent", "unifibot 2.0")
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Origin", s.Endpoint)

			if s.csrf != "" {
				req.Header.Set("x-csrf-token", s.csrf)
			}

			resp, err := s.client.Do(req)
			if err != nil {
				// Check if this is a retryable network error
				if IsRetryableNetworkError(err) {
					return fmt.Errorf("%w: %v", ErrRetryableHTTP, err)
				}
				return err
			}
			defer resp.Body.Close()

			if tok := resp.Header.Get("x-csrf-token"); tok != "" {
				s.csrf = tok
			}

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || http.StatusBadRequest <= resp.StatusCode {
				if resp.StatusCode == http.StatusUnauthorized {
					fmt.Fprintf(s.errWriter, "\nlogged out; re-authenticating\n")
					s.login = s.webLogin
					if _, err := s.login(); err != nil {
						return fmt.Errorf("login attempt failed: %w", err)
					}
					// Return retryable error for re-authentication
					responseError = fmt.Errorf("re-authentication required")
					result = string(respBody)
					return fmt.Errorf("%w: re-authentication required", ErrRetryableHTTP)
				} else if IsRetryableHTTPError(resp.StatusCode) {
					// Return retryable error for HTTP errors that should be retried
					return fmt.Errorf("%w: http error: %s", ErrRetryableHTTP, resp.Status)
				} else {
					// Non-retryable HTTP error
					return fmt.Errorf("http error: %s", resp.Status)
				}
			}

			result = string(respBody)
			return nil
		})
	})
	// Handle errors from retry/circuit breaker
	if err != nil {
		if errors.Is(err, ErrCircuitBreakerOpen) {
			s.setError(err)
			return "", fmt.Errorf("circuit breaker open, service unavailable: %w", err)
		}
		s.setError(err)
		return "", s.err
	}

	// Handle authentication errors that succeeded with retry
	if responseError != nil {
		s.setError(responseError)
		return result, s.err
	}

	return result, s.err
}

func (s *Session) setError(e error) {
	if e == nil {
		return
	}

	if s.err == nil {
		s.err = fmt.Errorf("%w", e)
	} else {
		s.err = fmt.Errorf("%s\n%w", e, s.err)
	}
}

func (s *Session) setErrorString(e string) {
	if len(e) == 0 {
		return
	}

	if s.err == nil {
		s.err = fmt.Errorf("%s", e) // nolint:goerr113
	} else {
		s.err = fmt.Errorf("%s\n%w", e, s.err)
	}
}
