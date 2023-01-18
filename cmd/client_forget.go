package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var forgetCmd = &cobra.Command{
	Use:     "forget",
	Aliases: []string{"f", "del"},
	Short:   "forget client",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		macs, err := ses.GetMACsBy(args...)
		cobra.CheckErr(err)

		_, err = ses.Forget(macs...)
		cobra.CheckErr(err)

		fmt.Fprintf(cmd.OutOrStdout(), "ok\n")
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(forgetCmd)
}
