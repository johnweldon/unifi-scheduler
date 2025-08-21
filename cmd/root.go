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
	Example: `  # List all connected clients
  unifi-scheduler --endpoint https://controller --username admin --password pass client list

  # Block a client by name
  unifi-scheduler --endpoint https://controller --username admin --password pass client block "Problem Device"

  # Monitor network events
  unifi-scheduler --endpoint https://controller --username admin --password pass event list

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

	pf.StringVar(&username, usernameFlag, username, "unifi username")
	_ = cobra.MarkFlagRequired(pf, usernameFlag)

	pf.StringVar(&password, passwordFlag, password, "unifi password")
	_ = cobra.MarkFlagRequired(pf, passwordFlag)

	pf.StringVar(&endpoint, endpointFlag, endpoint, "unifi endpoint")
	_ = cobra.MarkFlagRequired(pf, endpointFlag)

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
		unifi.WithSecureAuth(username, password),
	}

	if debug {
		// Use secure logging to prevent credential leakage in debug output
		options = append(options, unifi.SecureLogOption(cmd.OutOrStderr()))
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
