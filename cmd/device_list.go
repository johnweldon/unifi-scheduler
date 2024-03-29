package cmd

import (
	"github.com/spf13/cobra"
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

		for _, device := range devices {
			cmd.Printf("%s\n", device.String())
		}
	},
}

func init() { // nolint: gochecknoinits
	deviceCmd.AddCommand(listCmd)
}
