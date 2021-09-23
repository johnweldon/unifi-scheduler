package cmd

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
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

		checkGetName := func(mac unifi.MAC) (string, bool) {
			if slice, ok := macs[mac]; ok && len(slice) > 0 {
				return slice[0], true
			}

			return string(mac), false
		}

		getName := func(mac unifi.MAC) string {
			name, _ := checkGetName(mac)
			return name
		}

		configs := []table.ColumnConfig{
			{Name: "Name", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight},
			{Name: "Event"},
			{Name: "To"},
			{Name: "From"},
			{Name: "When"},
		}

		headerRow := table.Row{}
		for _, c := range configs {
			headerRow = append(headerRow, c.Name)
		}

		t := table.NewWriter()
		t.SetStyle(table.StyleColoredDark)
		t.SetColumnConfigs(configs)
		t.SetOutputMirror(cmd.OutOrStdout())

		t.AppendHeader(headerRow)
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
				if name, ok = checkGetName(mac); ok {
					evt := event.Key[7:]

					to := "-"
					from := "-"
					switch event.Key {
					case unifi.EventTypeWirelessUserRoam:
						to = getName(event.AccessPointTo)
						from = getName(event.AccessPointFrom)
					case
						unifi.EventTypeWirelessGuestDisconnected,
						unifi.EventTypeWirelessUserDisconnected:
						from = getName(event.AccessPoint)
					case unifi.EventTypeWirelessUserConnected:
						to = getName(event.AccessPoint)
					case unifi.EventTypeWirelessUserRoamRadio:
						from = fmt.Sprintf("%s (%d)", event.RadioFrom, event.ChannelFrom)
						to = fmt.Sprintf("%s (%d)", event.RadioTo, event.ChannelTo)
					}

					t.AppendRow([]interface{}{
						name, evt, to, from, event.TimeStamp.String(),
					})
				}
			}
		}
		t.AppendFooter(table.Row{fmt.Sprintf("Total %d", t.Length())})

		t.Render()
	},
}

func init() { // nolint: gochecknoinits
	eventCmd.AddCommand(eventConnectionsCmd)
}
