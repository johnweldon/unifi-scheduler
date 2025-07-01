package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/pkg/nats"
	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
	"github.com/johnweldon/unifi-scheduler/pkg/unifi/display"
)

var natsClientsCmd = &cobra.Command{
	Use:     "clients",
	Aliases: []string{"client", "cl", "c"},
	Short:   "show active clients",
	Run: func(cmd *cobra.Command, args []string) {
		opts := []nats.ClientOpt{nats.OptNATSUrl(natsURL), nats.OptCreds(natsCreds)}
		s := nats.NewSubscriber(opts...)

		var into []unifi.Client
		cobra.CheckErr(s.Get(nats.DetailBucket(baseSubject), nats.ActiveKey, &into))

		display.ClientsTable(cmd.OutOrStdout(), into).Render()
		fmt.Fprintf(cmd.OutOrStdout(), "%70s\n", time.Now().Format(time.RFC1123))
	},
}

func init() { // nolint: gochecknoinits
	natsCmd.AddCommand(natsClientsCmd)
}
