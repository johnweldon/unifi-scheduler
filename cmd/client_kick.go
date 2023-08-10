package cmd

import (
	"github.com/spf13/cobra"
)

var kickCmd = &cobra.Command{
	Use:     "kick",
	Aliases: []string{"k"},
	Short:   "kick client",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		macs, err := ses.GetMACsBy(args...)
		cobra.CheckErr(err)

		_, err = ses.Kick(macs...)
		cobra.CheckErr(err)

		cmd.Printf("ok\n")
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(kickCmd)
}
