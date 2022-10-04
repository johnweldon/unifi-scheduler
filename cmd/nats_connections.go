package cmd

import (
	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/pkg/nats"
	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
	"github.com/johnweldon/unifi-scheduler/pkg/unifi/display"
)

var natsConnectionsCmd = &cobra.Command{
	Use:     "connections",
	Aliases: []string{"conn"},
	Short:   "show connection events",
	Run: func(cmd *cobra.Command, args []string) {
		opts := []nats.ClientOpt{nats.OptNATSUrl(natsURL)}
		s := nats.NewSubscriber(opts...)

		detailBucket := nats.DetailBucket(baseSubject)

		var into []unifi.Event
		cobra.CheckErr(s.Get(detailBucket, nats.EventsKey, &into))

		unifi.DefaultEventSort.Sort(into)

		macBucket := nats.ByMACBucket(baseSubject)
		cache := map[string]string{}
		checkGetName := func(mac unifi.MAC) (string, bool) {
			k := nats.NormalizeKey(string(mac))

			if v, ok := cache[k]; ok {
				return v, true
			}

			var names []string
			if err := s.Get(macBucket, k, &names); err != nil || len(names) < 1 {
				return string(mac), false
			}

			cache[k] = names[0]

			return names[0], true
		}

		display.EventsTable(cmd.OutOrStdout(), checkGetName, into).Render()
	},
}

func init() { // nolint: gochecknoinits
	natsCmd.AddCommand(natsConnectionsCmd)
}
