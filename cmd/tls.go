package cmd

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
	"github.com/spf13/cobra"
)

// tlsCmd represents the tls command
var tlsCmd = &cobra.Command{
	Use:   "tls",
	Short: "TLS configuration and diagnostic tools",
	Long: `TLS configuration and diagnostic tools for secure connections to UniFi controllers.

This command provides utilities to test and validate TLS configurations, 
diagnose connection issues, and verify certificate settings.`,
	Example: `  # Test TLS connection to controller
  unifi-scheduler tls test --endpoint https://controller.local

  # Test with custom TLS settings
  unifi-scheduler tls test --endpoint https://controller.local --tls-min-version 1.3

  # Show TLS configuration
  unifi-scheduler tls config --endpoint https://controller.local

  # Test with insecure settings (development only)
  unifi-scheduler tls test --endpoint https://controller.local --tls-insecure`,
}

var tlsTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test TLS connection to UniFi controller",
	Long: `Test TLS connection to a UniFi controller endpoint.

This command establishes a TLS connection to the specified endpoint and reports
on the connection details, certificate information, and any issues encountered.`,
	Example: `  # Basic TLS test
  unifi-scheduler tls test --endpoint https://controller.local

  # Test with custom root CA
  unifi-scheduler tls test --endpoint https://controller.local --tls-root-ca /path/to/ca.pem

  # Test with client certificate (mutual TLS)
  unifi-scheduler tls test --endpoint https://controller.local \
    --tls-client-cert /path/to/cert.pem --tls-client-key /path/to/key.pem

  # Test with insecure settings
  unifi-scheduler tls test --endpoint https://controller.local --tls-insecure`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if endpoint == "" {
			return fmt.Errorf("endpoint is required for TLS testing")
		}

		// Create TLS configuration
		tlsConfig, err := createTLSConfig(cmd)
		if err != nil {
			return fmt.Errorf("failed to create TLS configuration: %w", err)
		}

		cmd.Printf("Testing TLS connection to: %s\n", endpoint)
		cmd.Printf("TLS Configuration:\n")
		cmd.Printf("  Min Version: %s\n", unifi.TLSVersionString(tlsConfig.MinVersion))
		cmd.Printf("  Max Version: %s\n", unifi.TLSVersionString(tlsConfig.MaxVersion))
		cmd.Printf("  Skip Verify: %v\n", tlsConfig.InsecureSkipVerify)
		cmd.Printf("  Strict Mode: %v\n", tlsConfig.StrictValidation)

		if tlsConfig.ServerName != "" {
			cmd.Printf("  Server Name: %s\n", tlsConfig.ServerName)
		}
		if tlsConfig.ClientCertFile != "" {
			cmd.Printf("  Client Cert: %s\n", tlsConfig.ClientCertFile)
		}
		if tlsConfig.RootCAFile != "" {
			cmd.Printf("  Root CA: %s\n", tlsConfig.RootCAFile)
		}

		cmd.Printf("\n")

		// Test the connection
		if err := testTLSConnection(cmd, endpoint, tlsConfig); err != nil {
			cmd.Printf("❌ TLS test failed: %v\n", err)
			return err
		}

		cmd.Printf("✅ TLS connection successful\n")
		return nil
	},
}

var tlsConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current TLS configuration",
	Long: `Display the current TLS configuration that would be used for connections.

This command shows the effective TLS settings based on command line flags,
configuration files, and defaults.`,
	Example: `  # Show default TLS configuration
  unifi-scheduler tls config

  # Show configuration with custom settings
  unifi-scheduler tls config --tls-min-version 1.3 --tls-insecure`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create TLS configuration
		tlsConfig, err := createTLSConfig(cmd)
		if err != nil {
			return fmt.Errorf("failed to create TLS configuration: %w", err)
		}

		cmd.Printf("Current TLS Configuration:\n")
		cmd.Printf("========================\n\n")

		// Basic settings
		cmd.Printf("Protocol Settings:\n")
		cmd.Printf("  Minimum TLS Version: %s (0x%04x)\n",
			unifi.TLSVersionString(tlsConfig.MinVersion), tlsConfig.MinVersion)
		cmd.Printf("  Maximum TLS Version: %s (0x%04x)\n",
			unifi.TLSVersionString(tlsConfig.MaxVersion), tlsConfig.MaxVersion)
		cmd.Printf("  Handshake Timeout: %v\n", tlsConfig.HandshakeTimeout)
		cmd.Printf("\n")

		// Security settings
		cmd.Printf("Security Settings:\n")
		cmd.Printf("  Certificate Verification: %s\n",
			map[bool]string{true: "❌ DISABLED (INSECURE)", false: "✅ Enabled"}[tlsConfig.InsecureSkipVerify])
		cmd.Printf("  Strict Validation: %s\n",
			map[bool]string{true: "✅ Enabled", false: "❌ Disabled"}[tlsConfig.StrictValidation])
		cmd.Printf("\n")

		// Certificate settings
		cmd.Printf("Certificate Settings:\n")
		if tlsConfig.ServerName != "" {
			cmd.Printf("  Server Name Override: %s\n", tlsConfig.ServerName)
		} else {
			cmd.Printf("  Server Name Override: (none - use endpoint hostname)\n")
		}

		if tlsConfig.ClientCertFile != "" {
			cmd.Printf("  Client Certificate: %s\n", tlsConfig.ClientCertFile)
			cmd.Printf("  Client Private Key: %s\n", tlsConfig.ClientKeyFile)
		} else {
			cmd.Printf("  Client Certificate: (none - server authentication only)\n")
		}

		if tlsConfig.RootCAFile != "" {
			cmd.Printf("  Custom Root CA: %s\n", tlsConfig.RootCAFile)
		} else if len(tlsConfig.RootCAData) > 0 {
			cmd.Printf("  Custom Root CA: (provided as data)\n")
		} else {
			cmd.Printf("  Root CA: (system default certificate store)\n")
		}
		cmd.Printf("\n")

		// Cipher suites
		cmd.Printf("Cipher Suites (%d configured):\n", len(tlsConfig.CipherSuites))
		for i, suite := range tlsConfig.CipherSuites {
			cmd.Printf("  %d. %s (0x%04x)\n", i+1, unifi.CipherSuiteName(suite), suite)
		}
		cmd.Printf("\n")

		// Validation status
		cmd.Printf("Configuration Status:\n")
		if err := tlsConfig.Validate(); err != nil {
			cmd.Printf("  ❌ Validation: FAILED\n")
			cmd.Printf("     Error: %v\n", err)
		} else {
			cmd.Printf("  ✅ Validation: PASSED\n")
		}

		// Security warnings
		if tlsConfig.InsecureSkipVerify {
			cmd.Printf("\n⚠️  WARNING: Certificate verification is disabled!\n")
			cmd.Printf("   This makes connections vulnerable to man-in-the-middle attacks.\n")
			cmd.Printf("   Only use this setting in development environments.\n")
		}

		if tlsConfig.MinVersion < tls.VersionTLS12 {
			cmd.Printf("\n⚠️  WARNING: Minimum TLS version is below 1.2!\n")
			cmd.Printf("   TLS versions below 1.2 have known security vulnerabilities.\n")
		}

		return nil
	},
}

func testTLSConnection(cmd *cobra.Command, endpoint string, tlsConfig *unifi.TLSConfig) error {
	// Parse endpoint to get hostname for connection testing
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}

	if u.Scheme != "https" {
		return fmt.Errorf("endpoint must use HTTPS protocol")
	}

	// Create transport and test connection
	transport, err := tlsConfig.CreateSecureTransport()
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}

	// Get the actual TLS config for connection details
	stdTLSConfig := transport.TLSClientConfig

	// Establish TLS connection for detailed inspection
	conn, err := tls.Dial("tcp", u.Host, stdTLSConfig)
	if err != nil {
		return fmt.Errorf("TLS connection failed: %w", err)
	}
	defer conn.Close()

	// Get connection state
	state := conn.ConnectionState()

	cmd.Printf("Connection Details:\n")
	cmd.Printf("  Protocol Version: %s\n", unifi.TLSVersionString(state.Version))
	cmd.Printf("  Cipher Suite: %s\n", unifi.CipherSuiteName(state.CipherSuite))
	cmd.Printf("  Server Name: %s\n", state.ServerName)
	cmd.Printf("  Handshake Complete: %v\n", state.HandshakeComplete)
	cmd.Printf("  Mutual TLS: %v\n", len(state.PeerCertificates) > 0 && len(state.OCSPResponse) > 0)
	cmd.Printf("\n")

	// Certificate information
	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		cmd.Printf("Server Certificate:\n")
		cmd.Printf("  Subject: %s\n", cert.Subject.String())
		cmd.Printf("  Issuer: %s\n", cert.Issuer.String())
		cmd.Printf("  Serial: %s\n", cert.SerialNumber.String())
		cmd.Printf("  Not Before: %s\n", cert.NotBefore.Format("2006-01-02 15:04:05 MST"))
		cmd.Printf("  Not After: %s\n", cert.NotAfter.Format("2006-01-02 15:04:05 MST"))

		// DNS names
		if len(cert.DNSNames) > 0 {
			cmd.Printf("  DNS Names: %s\n", strings.Join(cert.DNSNames, ", "))
		}

		// Check if certificate is valid
		opts := []string{}
		if cert.NotAfter.Before(cert.NotBefore) {
			opts = append(opts, "❌ Invalid time range")
		}
		if len(opts) > 0 {
			cmd.Printf("  Issues: %s\n", strings.Join(opts, ", "))
		}
		cmd.Printf("\n")
	}

	return nil
}

func init() { // nolint: gochecknoinits
	rootCmd.AddCommand(tlsCmd)

	// Add subcommands
	tlsCmd.AddCommand(tlsTestCmd)
	tlsCmd.AddCommand(tlsConfigCmd)

	// These commands inherit the TLS flags from root command
	// No additional flags needed as they use the global TLS configuration
}
