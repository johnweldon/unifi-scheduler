package cmd

import (
	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/nats"
	"github.com/johnweldon/unifi-scheduler/unifi"
	"github.com/johnweldon/unifi-scheduler/unifi/display"
)

var natsClientsCmd = &cobra.Command{
	Use:     "clients",
	Aliases: []string{"client", "cl", "c"},
	Short:   "show active clients",
	Run: func(cmd *cobra.Command, args []string) {
		opts := []nats.ClientOpt{nats.OptNATSUrl(natsURL)}
		s := nats.NewSubscriber(opts...)

		var into []unifi.Client
		cobra.CheckErr(s.Get(nats.DetailBucket(baseSubject), nats.ActiveKey, &into))

		display.ClientsTable(cmd.OutOrStdout(), into).Render()
	},
}

func init() { // nolint: gochecknoinits
	natsCmd.AddCommand(natsClientsCmd)
}
