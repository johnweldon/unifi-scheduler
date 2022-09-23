package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/nats"
)

var natsCmd = &cobra.Command{
	Use:     "nats",
	Aliases: []string{"nts", "n"},
	Short:   "nats tools",
}

var natsAgentCmd = &cobra.Command{
	Use:     "agent",
	Aliases: []string{"agt", "a"},
	Short:   "nats agent",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, _ := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)

		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		baseSubject := "unifi"

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
				fmt.Fprintf(cmd.OutOrStdout(), "\n")
			}
		}
	},
}

const natsURLFlag = "nats_url"

var natsURL = "nats://localhost:4222"

func init() { // nolint: gochecknoinits
	pf := natsCmd.PersistentFlags()

	pf.StringVar(&natsURL, natsURLFlag, natsURL, "NATS URL")
	_ = cobra.MarkFlagRequired(pf, natsURLFlag)

	natsCmd.AddCommand(natsAgentCmd)
	rootCmd.AddCommand(natsCmd)
}
