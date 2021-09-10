package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var kickCmd = &cobra.Command{
	Use:   "kick",
	Short: "kick client",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		names, err := ses.GetNames()
		cobra.CheckErr(err)

		for _, victim := range args {
			if mac, ok := names[victim]; ok {
				fmt.Fprintf(cmd.OutOrStdout(), "kicking %q (%s) ... ", victim, mac)
				_, err := ses.Kick(mac)
				cobra.CheckErr(err)
				fmt.Fprintf(cmd.OutOrStdout(), "ok\n")
			}
		}
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(kickCmd)
}
