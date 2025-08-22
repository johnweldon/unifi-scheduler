package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	lnats "github.com/johnweldon/unifi-scheduler/pkg/nats"
	"github.com/johnweldon/unifi-scheduler/pkg/output"
	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
)

var (
	cfgFile  string
	debug    bool
	username string
	password string
	endpoint string

	// Output format options
	outputFormat string

	// Secure credential input options
	credentialFile  string
	useStdin        bool
	useKeychain     bool
	keychainService string
	keychainAccount string

	// TLS configuration options
	tlsInsecure       bool
	tlsMinVersion     string
	tlsMaxVersion     string
	tlsClientCertFile string
	tlsClientKeyFile  string
	tlsRootCAFile     string
	tlsServerName     string

	httpTimeout     time.Duration
	natsConnTimeout time.Duration
	natsOpTimeout   time.Duration
	streamReplicas  int
	kvReplicas      int

	Version string
)

var rootCmd = &cobra.Command{
	Use:     "unifi-scheduler",
	Aliases: []string{"ucli"},
	Short:   "A powerful CLI tool for managing UniFi network controllers",
	Long: `UniFi Scheduler provides comprehensive management of UniFi network controllers,
including client management, device monitoring, event tracking, and distributed
operations via NATS messaging.

For complete documentation and examples, visit:
https://github.com/johnweldon/unifi-scheduler`,
	Example: `  # List all connected clients (secure defaults)
  unifi-scheduler --endpoint https://controller --credential-file ~/.unifi-creds.json client list

  # Output in JSON format for automation
  unifi-scheduler --endpoint https://controller --output json client list

  # Output in YAML format
  unifi-scheduler --endpoint https://controller --output yaml device list

  # Connect with custom TLS settings
  unifi-scheduler --endpoint https://controller --tls-min-version 1.3 --keychain --keychain-account admin client list

  # Connect with custom root CA (for self-signed certificates)
  unifi-scheduler --endpoint https://controller --tls-root-ca /path/to/ca.pem --stdin client list

  # Connect with mutual TLS authentication
  unifi-scheduler --endpoint https://controller --tls-client-cert /path/to/cert.pem --tls-client-key /path/to/key.pem client list

  # Development mode (insecure - not for production)
  unifi-scheduler --endpoint https://controller --tls-insecure --username admin --password pass client list

  # Test TLS connection
  unifi-scheduler tls test --endpoint https://controller

  # Use configuration file
  unifi-scheduler --config ~/.unifi-scheduler.yaml client list`,
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"ver", "v"},
	Short:   "Display application version information",
	Long:    "Display the current version of unifi-scheduler.",
	Example: "  unifi-scheduler version",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("Version: %s\n", Version)
	},
}

func Execute(version string) {
	Version = version
	cobra.CheckErr(rootCmd.Execute())
}

func init() { // nolint: gochecknoinits
	cobra.OnInitialize(initConfig)

	pf := rootCmd.PersistentFlags()

	pf.StringVar(&cfgFile, "config", "", "config file (default is $HOME/.unifi-scheduler.yaml)")
	pf.BoolVar(&debug, "debug", debug, "debug output")
	pf.StringVar(&outputFormat, "output", "table", "output format (table, json, yaml)")

	pf.StringVar(&username, usernameFlag, username, "unifi username (optional if using secure credential input)")
	pf.StringVar(&password, passwordFlag, password, "unifi password (optional if using secure credential input)")
	pf.StringVar(&endpoint, endpointFlag, endpoint, "unifi endpoint")
	_ = cobra.MarkFlagRequired(pf, endpointFlag)

	// Secure credential input flags
	pf.StringVar(&credentialFile, "credential-file", "", "path to JSON credential file")
	pf.BoolVar(&useStdin, "stdin", false, "read credentials from stdin")
	pf.BoolVar(&useKeychain, "keychain", false, "read credentials from system keychain")
	pf.StringVar(&keychainService, "keychain-service", "unifi-scheduler", "keychain service name")
	pf.StringVar(&keychainAccount, "keychain-account", "", "keychain account/username")

	// TLS configuration flags
	pf.BoolVar(&tlsInsecure, "tls-insecure", false, "skip TLS certificate verification (not recommended)")
	pf.StringVar(&tlsMinVersion, "tls-min-version", "1.2", "minimum TLS version (1.0, 1.1, 1.2, 1.3)")
	pf.StringVar(&tlsMaxVersion, "tls-max-version", "1.3", "maximum TLS version (1.0, 1.1, 1.2, 1.3)")
	pf.StringVar(&tlsClientCertFile, "tls-client-cert", "", "client certificate file for mutual TLS")
	pf.StringVar(&tlsClientKeyFile, "tls-client-key", "", "client private key file for mutual TLS")
	pf.StringVar(&tlsRootCAFile, "tls-root-ca", "", "root CA certificate file")
	pf.StringVar(&tlsServerName, "tls-server-name", "", "server name for certificate verification")

	// Timeout configuration
	pf.DurationVar(&httpTimeout, "http-timeout", 2*time.Minute, "HTTP request timeout")
	pf.DurationVar(&natsConnTimeout, "nats-conn-timeout", 15*time.Second, "NATS connection timeout")
	pf.DurationVar(&natsOpTimeout, "nats-op-timeout", 30*time.Second, "NATS operation timeout")
	pf.IntVar(&streamReplicas, "stream-replicas", 1, "NATS stream replica count")
	pf.IntVar(&kvReplicas, "kv-replicas", 1, "NATS key-value replica count")

	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.SetConfigName(".unifi-scheduler")
		viper.SetEnvPrefix("unifi")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	if err := postInitConfig(rootCmd.Commands()); err != nil {
		fmt.Fprintf(os.Stderr, "Error during configuration: %v\n", err)
		os.Exit(1)
	}
}

func postInitConfig(commands []*cobra.Command) error {
	for _, cmd := range commands {
		if err := presetRequiredFlags(cmd); err != nil {
			return err
		}
		if cmd.HasSubCommands() {
			if err := postInitConfig(cmd.Commands()); err != nil {
				return err
			}
		}
	}
	return nil
}

func presetRequiredFlags(cmd *cobra.Command) error {
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return fmt.Errorf("binding flags for command %q: %w", cmd.Name(), err)
	}

	var flagError error
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if flagError != nil {
			return // Stop processing if we already have an error
		}
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			if err := cmd.Flags().Set(f.Name, viper.GetString(f.Name)); err != nil {
				flagError = fmt.Errorf("setting flag %q for command %q: %w", f.Name, cmd.Name(), err)
			}
		}
	})

	return flagError
}

const (
	usernameFlag = "username"
	passwordFlag = "password"
	endpointFlag = "endpoint"
)

func initSession(cmd *cobra.Command) (*unifi.Session, error) {
	// Get credentials using secure credential management
	credentials, err := getSecureCredentials(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain credentials: %w", err)
	}

	ses := &unifi.Session{
		Endpoint: endpoint,
	}

	var outio, errio io.Writer

	nc, err := nats.Connect(natsURL)
	if err != nil {
		// Continue without NATS logging
		if debug {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not connect to NATS (%v), continuing without NATS logging\n", err)
		}
		outio = cmd.OutOrStdout()
		errio = cmd.ErrOrStderr()
	} else {
		// NATS connection successful
		outio = io.MultiWriter(&lnats.Logger{
			Connection:     nc,
			PublishSubject: "log.info",
		}, cmd.OutOrStdout())

		errio = io.MultiWriter(&lnats.Logger{
			Connection:     nc,
			PublishSubject: "log.error",
		}, cmd.ErrOrStderr())
	}

	// Create TLS configuration
	tlsConfig, err := createTLSConfig(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS configuration: %w", err)
	}

	options := []unifi.Option{
		unifi.WithOut(outio),
		unifi.WithErr(errio),
		unifi.WithHTTPTimeout(httpTimeout),
		unifi.WithCredentials(credentials),
		unifi.WithTLSConfig(tlsConfig),
	}

	if debug {
		// Use secure logging to prevent credential leakage in debug output
		options = append(options, unifi.WithDbg(unifi.NewSecureWriter(cmd.OutOrStderr())))
	}

	if err := ses.Initialize(options...); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "error initializing: %v\n", err)

		return nil, err
	}

	if msg, err := ses.Login(); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "error logging in %q: %v\n", msg, err)

		return nil, err
	}

	return ses, nil
}

// getSecureCredentials obtains credentials using secure methods with fallback
func getSecureCredentials(cmd *cobra.Command) (*unifi.Credentials, error) {
	manager := unifi.NewCredentialManager(debug)

	// Add credential sources in order of preference

	// 1. Command line flags (if both username and password provided)
	if username != "" && password != "" {
		if debug {
			fmt.Fprintf(cmd.ErrOrStderr(), "Using credentials from command line flags\n")
		}
		return unifi.NewCredentials(username, password)
	}

	// 2. Credential file (if specified)
	if credentialFile != "" {
		manager.AddSource(unifi.NewFileCredentialSource(credentialFile))
	}

	// 3. Environment variables
	manager.AddSource(unifi.NewEnvironmentCredentialSource("UNIFI_USERNAME", "UNIFI_PASSWORD"))

	// 4. Keychain (if enabled)
	if useKeychain {
		account := keychainAccount
		if account == "" {
			account = username // Use username from flags if available
		}
		if account == "" {
			return nil, fmt.Errorf("keychain account required when using keychain")
		}
		manager.AddSource(unifi.NewKeychainCredentialSource(keychainService, account))
	}

	// 5. Stdin (if enabled or no other sources available)
	if useStdin || (!useKeychain && credentialFile == "" && username == "" && password == "") {
		// Check if we can prompt for credentials
		if unifi.IsCredentialInputAvailable() || useStdin {
			manager.AddSource(unifi.NewStdinCredentialSource(cmd.InOrStdin(), cmd.ErrOrStderr(), true))
		}
	}

	// Try to get credentials from sources
	credentials, err := manager.GetCredentials()
	if err != nil {
		return nil, fmt.Errorf("unable to obtain credentials: %w\n\nAvailable options:\n"+
			"  --username and --password flags\n"+
			"  --credential-file path/to/creds.json\n"+
			"  --keychain with --keychain-account\n"+
			"  --stdin to read from stdin\n"+
			"  UNIFI_USERNAME and UNIFI_PASSWORD environment variables", err)
	}

	return credentials, nil
}

// createTLSConfig creates TLS configuration from command line flags
func createTLSConfig(cmd *cobra.Command) (*unifi.TLSConfig, error) {
	var opts []unifi.TLSConfigOption

	// Handle insecure mode
	if tlsInsecure {
		fmt.Fprintf(cmd.ErrOrStderr(), "WARNING: TLS certificate verification is disabled. This is insecure and not recommended for production.\n")
		opts = append(opts, unifi.WithInsecureSkipVerify(true))
	}

	// Parse TLS versions
	if tlsMinVersion != "" {
		minVer, err := parseTLSVersion(tlsMinVersion)
		if err != nil {
			return nil, fmt.Errorf("invalid minimum TLS version %q: %w", tlsMinVersion, err)
		}
		opts = append(opts, unifi.WithMinTLSVersion(minVer))
	}

	if tlsMaxVersion != "" {
		maxVer, err := parseTLSVersion(tlsMaxVersion)
		if err != nil {
			return nil, fmt.Errorf("invalid maximum TLS version %q: %w", tlsMaxVersion, err)
		}
		opts = append(opts, unifi.WithMaxTLSVersion(maxVer))
	}

	// Client certificate for mutual TLS
	if tlsClientCertFile != "" || tlsClientKeyFile != "" {
		if tlsClientCertFile == "" || tlsClientKeyFile == "" {
			return nil, fmt.Errorf("both --tls-client-cert and --tls-client-key must be specified for mutual TLS")
		}
		opts = append(opts, unifi.WithClientCertificate(tlsClientCertFile, tlsClientKeyFile))
	}

	// Root CA certificate
	if tlsRootCAFile != "" {
		opts = append(opts, unifi.WithRootCA(tlsRootCAFile))
	}

	// Server name override
	if tlsServerName != "" {
		opts = append(opts, unifi.WithServerName(tlsServerName))
	}

	// Create and validate configuration
	config := unifi.NewTLSConfig(opts...)
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("TLS configuration validation failed: %w", err)
	}

	return config, nil
}

// getOutputOptions creates output options from the global output format flag
func getOutputOptions(cmd *cobra.Command) (*output.OutputOptions, error) {
	return output.NewOutputOptions(outputFormat, cmd.OutOrStdout())
}

// parseTLSVersion converts string version to TLS constant
func parseTLSVersion(version string) (uint16, error) {
	switch version {
	case "1.0":
		return 0x0301, nil // tls.VersionTLS10
	case "1.1":
		return 0x0302, nil // tls.VersionTLS11
	case "1.2":
		return 0x0303, nil // tls.VersionTLS12
	case "1.3":
		return 0x0304, nil // tls.VersionTLS13
	default:
		return 0, fmt.Errorf("unsupported TLS version: %s (supported: 1.0, 1.1, 1.2, 1.3)", version)
	}
}
