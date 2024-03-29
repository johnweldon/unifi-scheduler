package cmd

import (
	"fmt"
	"io"
	"os"

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

	Version string
)

var rootCmd = &cobra.Command{
	Use:     "unifi-scheduler",
	Aliases: []string{"ucli"},
	Short:   "utility for interacting with unifi",
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"ver", "v"},
	Short:   "application version",
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

	postInitConfig(rootCmd.Commands())
}

func postInitConfig(commands []*cobra.Command) {
	for _, cmd := range commands {
		presetRequiredFlags(cmd)
		if cmd.HasSubCommands() {
			postInitConfig(cmd.Commands())
		}
	}
}

func presetRequiredFlags(cmd *cobra.Command) {
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		panic(err)
	}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			if err := cmd.Flags().Set(f.Name, viper.GetString(f.Name)); err != nil {
				panic(err)
			}
		}
	})
}

const (
	usernameFlag = "username"
	passwordFlag = "password"
	endpointFlag = "endpoint"
)

func initSession(cmd *cobra.Command) (*unifi.Session, error) {
	ses := &unifi.Session{
		Endpoint: endpoint,
		Username: username,
		Password: password,
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "error connecting to NATS: %v\n", err)

		return nil, err
	}

	outio := io.MultiWriter(&lnats.Logger{
		Connection:     nc,
		PublishSubject: "log.info",
	}, cmd.OutOrStdout())

	errio := io.MultiWriter(&lnats.Logger{
		Connection:     nc,
		PublishSubject: "log.error",
	}, cmd.ErrOrStderr())

	options := []unifi.Option{
		unifi.WithOut(outio),
		unifi.WithErr(errio),
	}

	if debug {
		options = append(options, unifi.WithDbg(cmd.OutOrStderr()))
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
