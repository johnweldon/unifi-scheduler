package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var unblockCmd = &cobra.Command{
	Use:     "unblock",
	Aliases: []string{"ublock", "unblk", "u"},
	Short:   "unblock client",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		names, err := ses.GetNames()
		cobra.CheckErr(err)

		for _, victim := range args {
			if macs, ok := names[victim]; ok {
				for _, mac := range macs {
					fmt.Fprintf(cmd.OutOrStdout(), "unblocking %q (%s) ... ", victim, mac)
					_, err := ses.Unblock(mac)
					cobra.CheckErr(err)
					fmt.Fprintf(cmd.OutOrStdout(), "ok\n")
				}
			}
		}
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(unblockCmd)
}
