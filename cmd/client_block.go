package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var blockCmd = &cobra.Command{
	Use:     "block",
	Aliases: []string{"blk", "bl"},
	Short:   "block client",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		macs, err := ses.GetMACsBy(args...)
		cobra.CheckErr(err)

		_, err = ses.Block(macs...)
		cobra.CheckErr(err)

		fmt.Fprintf(cmd.OutOrStdout(), "ok\n")
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(blockCmd)
}
