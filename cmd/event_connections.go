package cmd

import (
	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/unifi"
	"github.com/johnweldon/unifi-scheduler/unifi/display"
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

		checkGetName := func(mac unifi.MAC) (string, bool) {
			if slice, ok := macs[mac]; ok && len(slice) > 0 {
				return slice[0], true
			}

			return string(mac), false
		}

		display.EventsTable(cmd.OutOrStdout(), checkGetName, events).Render()
	},
}

func init() { // nolint: gochecknoinits
	eventCmd.AddCommand(eventConnectionsCmd)
}
