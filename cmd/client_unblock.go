package cmd

import (
	"github.com/spf13/cobra"
)

var unblockCmd = &cobra.Command{
	Use:     "unblock",
	Aliases: []string{"ublock", "unblk", "u"},
	Short:   "unblock client",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		macs, err := ses.GetMACsBy(args...)
		cobra.CheckErr(err)

		_, err = ses.Unblock(macs...)
		cobra.CheckErr(err)

		cmd.Printf("ok\n")
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(unblockCmd)
}
