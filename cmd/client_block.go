package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var blockCmd = &cobra.Command{
	Use:   "block",
	Short: "block client",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		names, err := ses.GetNames()
		cobra.CheckErr(err)

		for _, victim := range args {
			if mac, ok := names[victim]; ok {
				fmt.Fprintf(cmd.OutOrStdout(), "blocking %q (%s) ... ", victim, mac)
				_, err := ses.Block(mac)
				cobra.CheckErr(err)
				fmt.Fprintf(cmd.OutOrStdout(), "ok\n")
			}
		}
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(blockCmd)
}
