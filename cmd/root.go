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
	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
)

var (
	cfgFile  string
	debug    bool
	username string
	password string
	endpoint string

	// Secure credential input options
	credentialFile  string
	useStdin        bool
	useKeychain     bool
	keychainService string
	keychainAccount string

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
	Example: `  # List all connected clients (using credential file)
  unifi-scheduler --endpoint https://controller --credential-file ~/.unifi-creds.json client list

  # Block a client by name (using keychain)
  unifi-scheduler --endpoint https://controller --keychain --keychain-account admin client block "Problem Device"

  # Monitor network events (using environment variables)
  UNIFI_USERNAME=admin UNIFI_PASSWORD=secret unifi-scheduler --endpoint https://controller event list

  # Interactive credential input
  unifi-scheduler --endpoint https://controller --stdin client list

  # Traditional username/password flags (less secure)
  unifi-scheduler --endpoint https://controller --username admin --password pass client list

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

	// Timeout configuration
	pf.DurationVar(&httpTimeout, "http-timeout", 2*time.Minute, "HTTP request timeout")
	pf.DurationVar(&natsConnTimeout, "nats-conn-timeout", 15*time.Second, "NATS connection timeout")
	pf.DurationVar(&natsOpTimeout, "nats-op-timeout", 30*time.Second, "NATS operation timeout")
	pf.IntVar(&streamReplicas, "stream-replicas", 3, "NATS stream replica count")
	pf.IntVar(&kvReplicas, "kv-replicas", 3, "NATS key-value replica count")

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

	options := []unifi.Option{
		unifi.WithOut(outio),
		unifi.WithErr(errio),
		unifi.WithHTTPTimeout(httpTimeout),
		unifi.WithCredentials(credentials),
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
