package cmd

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

var all bool

var clientListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "list clients",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		fetch := ses.GetClients
		if all {
			fetch = ses.GetUsers
		}

		clients, err := fetch()
		cobra.CheckErr(err)

		configs := []table.ColumnConfig{
			{Name: "Name", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight},
			{Name: "B"},
			{Name: "G"},
			{Name: "W"},
			{Name: "IP"},
			{Name: "Uptime"},
			{Name: "Rx", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight},
			{Name: "Tx", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight},
			{Name: "Rx Rate", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight},
			{Name: "Tx Rate", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight},
			{Name: "Link"},
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
		for _, client := range clients {
			t.AppendRow([]interface{}{
				client.DisplayName(),
				string(client.IsBlockedGlyph()),
				string(client.IsGuestGlyph()),
				string(client.IsWiredGlyph()),
				client.DisplayIP(),
				client.DisplayUptime(),
				client.DisplayReceivedBytes(),
				client.DisplaySentBytes(),
				client.DisplayReceiveRate(),
				client.DisplaySendRate(),
				client.DisplaySwitchName(),
			})
		}
		t.AppendFooter(table.Row{fmt.Sprintf("Total %d", len(clients))})

		t.Render()
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(clientListCmd)

	clientListCmd.Flags().BoolVar(&all, "all", all, "show all clients")
}
