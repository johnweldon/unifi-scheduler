package cmd

import (
	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/pkg/output"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "list devices",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		devices, err := ses.GetDevices()
		cobra.CheckErr(err)

		// Get output options and create formatter
		opts, err := getOutputOptions(cmd)
		cobra.CheckErr(err)

		formatter := opts.CreateFormatter()

		// Handle different output formats
		switch opts.Format {
		case output.FormatTable:
			for _, device := range devices {
				cmd.Printf("%s\n", device.String())
			}
		case output.FormatJSON, output.FormatYAML:
			err = formatter.Write(devices)
			cobra.CheckErr(err)
		}
	},
}

func init() { // nolint: gochecknoinits
	deviceCmd.AddCommand(listCmd)
}
