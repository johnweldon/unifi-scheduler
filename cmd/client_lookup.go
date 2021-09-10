package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var lookupCmd = &cobra.Command{
	Use:   "lookup",
	Short: "lookup client",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		names, err := ses.GetNames()
		cobra.CheckErr(err)

		for _, victim := range args {
			if mac, ok := names[victim]; ok {
				fmt.Fprintf(cmd.OutOrStdout(), "%s %q\n", mac, victim)
			}
		}
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(lookupCmd)
}
