package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/unifi"
)

var eventConnectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "connections events",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		events, err := ses.GetAllEvents()
		cobra.CheckErr(err)

		macs, err := ses.GetMACs()
		cobra.CheckErr(err)

		for _, event := range events {
			var (
				name string
				ok   bool
			)

			for _, mac := range []unifi.MAC{
				event.User,
				event.Client,
				event.Guest,
			} {
				if name, ok = macs[mac]; ok {
					fmt.Fprintf(cmd.OutOrStdout(), "%40s  %-25s  %s\n", name, event.Key, event.TimeStamp)
				}
			}
		}
	},
}

func init() { // nolint: gochecknoinits
	eventCmd.AddCommand(eventConnectionsCmd)
}
