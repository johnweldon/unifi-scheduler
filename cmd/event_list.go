package cmd

import (
	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/pkg/output"
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

		// Get output options and create formatter
		opts, err := getOutputOptions(cmd)
		cobra.CheckErr(err)

		formatter := opts.CreateFormatter()

		// Handle different output formats
		switch opts.Format {
		case output.FormatTable:
			for _, event := range events {
				cmd.Printf("%s\n", event.String())
			}
		case output.FormatJSON, output.FormatYAML:
			err = formatter.Write(events)
			cobra.CheckErr(err)
		}
	},
}

func init() { // nolint: gochecknoinits
	eventCmd.AddCommand(eventListCmd)

	eventListCmd.Flags().BoolVar(&allEvents, "all", allEvents, "show all events")
}
