package cmd

import (
	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/unifi/display"
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

		display.ClientsTable(cmd.OutOrStdout(), clients).Render()
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(clientListCmd)

	clientListCmd.Flags().BoolVar(&all, "all", all, "show all clients")
}
