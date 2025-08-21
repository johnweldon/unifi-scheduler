package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/pkg/nats"
)

var natsAgentCmd = &cobra.Command{
	Use:     "agent",
	Aliases: []string{"agt", "a"},
	Short:   "nats agent",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("Version: %s\n", Version)

		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		opts := []nats.ClientOpt{
			nats.OptNATSUrl(natsURL),
			nats.OptCreds(natsCreds),
			nats.OptConnectTimeout(natsConnTimeout),
			nats.OptWriteTimeout(natsConnTimeout),
			nats.OptOperationTimeout(natsOpTimeout),
			nats.OptStreamReplicas(streamReplicas),
			nats.OptKVReplicas(kvReplicas),
		}

		a := nats.NewAgent(ses, baseSubject, opts...)
		cobra.CheckErr(a.Start(ctx))

		markInterval := time.After(1 * time.Second)
		hourInterval := time.After(1 * time.Hour)

		for {
			select {
			case <-ctx.Done():
				cmd.Printf("quitting...\n")
				return
			case <-markInterval:
				markInterval = time.After(1 * time.Minute)
				cmd.Printf(".")
			case <-hourInterval:
				hourInterval = time.After(1 * time.Hour)
				cmd.Printf("H\n")
			}

			_ = os.Stdout.Sync()
		}
	},
}

func init() { // nolint: gochecknoinits
	natsCmd.AddCommand(natsAgentCmd)
}
