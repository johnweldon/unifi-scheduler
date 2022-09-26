package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/nats"
)

var natsAgentCmd = &cobra.Command{
	Use:     "agent",
	Aliases: []string{"agt", "a"},
	Short:   "nats agent",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\n", Version)

		ctx, _ := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)

		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		opts := []nats.ClientOpt{nats.OptNATSUrl(natsURL)}

		a := nats.NewAgent(ses, baseSubject, opts...)
		cobra.CheckErr(a.Start(ctx))

		markInterval := time.After(1 * time.Second)
		nlInterval := time.After(2 * time.Second)

		for {
			select {
			case <-ctx.Done():
				fmt.Fprintf(cmd.OutOrStdout(), "quitting...\n")
				return
			case <-markInterval:
				markInterval = time.After(1 * time.Minute)
				fmt.Fprintf(cmd.OutOrStdout(), ".")
			case <-nlInterval:
				nlInterval = time.After(1 * time.Hour)
				fmt.Fprintf(cmd.OutOrStdout(), "H\n")
			}
		}
	},
}

func init() { // nolint: gochecknoinits
	natsCmd.AddCommand(natsAgentCmd)
}
