package cmd

import (
	"fmt"

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

		for _, client := range clients {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", client.String())
		}
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(clientListCmd)

	clientListCmd.Flags().BoolVar(&all, "all", all, "show all clients")
}
