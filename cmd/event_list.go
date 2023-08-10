package cmd

import (
	"github.com/spf13/cobra"
)

var allEvents bool

var eventListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "list events",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		fetch := ses.GetRecentEvents
		if allEvents {
			fetch = ses.GetAllEvents
		}

		events, err := fetch()
		cobra.CheckErr(err)

		for _, event := range events {
			cmd.Printf("%s\n", event.String())
		}
	},
}

func init() { // nolint: gochecknoinits
	eventCmd.AddCommand(eventListCmd)

	eventListCmd.Flags().BoolVar(&allEvents, "all", allEvents, "show all events")
}
