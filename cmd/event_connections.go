package cmd

import (
	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
	"github.com/johnweldon/unifi-scheduler/pkg/unifi/display"
)

var allConnections bool

var eventConnectionsCmd = &cobra.Command{
	Use:     "connections",
	Aliases: []string{"conns"},
	Short:   "connections events",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		fetch := ses.GetRecentEvents
		if allConnections {
			fetch = ses.GetAllEvents
		}

		events, err := fetch()
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

	eventCmd.PersistentFlags().BoolVar(&allConnections, "all", allConnections, "show all connection events")
}
