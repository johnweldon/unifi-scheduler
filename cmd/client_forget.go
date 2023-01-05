package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
)

var forgetCmd = &cobra.Command{
	Use:     "forget",
	Aliases: []string{"f", "del"},
	Short:   "forget client",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		names, err := ses.GetNames()
		cobra.CheckErr(err)

		var all []unifi.MAC
		for _, victim := range args {
			if macs, ok := names[victim]; ok {
				for _, mac := range macs {
					fmt.Fprintf(cmd.OutOrStdout(), "forgetting %q (%s) ... \n", victim, mac)
					all = append(all, mac)
				}
			}
		}

		_, err = ses.Forget(all)
		cobra.CheckErr(err)
		fmt.Fprintf(cmd.OutOrStdout(), "ok\n")
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(forgetCmd)
}
