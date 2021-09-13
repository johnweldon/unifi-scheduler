package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/unifi"
)

var eventConnectionsCmd = &cobra.Command{
	Use:     "connections",
	Aliases: []string{"conns"},
	Short:   "connections events",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		events, err := ses.GetAllEvents()
		cobra.CheckErr(err)

		macs, err := ses.GetMACs()
		cobra.CheckErr(err)

		for _, event := range events {
			var (
				names []string
				ok    bool
			)

			for _, mac := range []unifi.MAC{
				event.User,
				event.Client,
				event.Guest,
			} {
				if names, ok = macs[mac]; ok {
					name := string(mac)
					if len(names) > 0 {
						name = names[0]
					}

					evt := event.Key[7:]

					fmt.Fprintf(cmd.OutOrStdout(), "%20s  %-15s  %s\n", name, evt, event.TimeStamp)
				}
			}
		}
	},
}

func init() { // nolint: gochecknoinits
	eventCmd.AddCommand(eventConnectionsCmd)
}
