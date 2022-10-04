package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
)

var lookupCmd = &cobra.Command{
	Use:     "lookup",
	Aliases: []string{"look"},
	Short:   "lookup client",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		names, err := ses.GetNames()
		cobra.CheckErr(err)

		macs, err := ses.GetMACs()
		cobra.CheckErr(err)

		for _, victim := range args {
			if macs, ok := names[victim]; ok {
				for _, mac := range macs {
					fmt.Fprintf(cmd.OutOrStdout(), "%s %q\n", mac, victim)
				}
			}

			if name, ok := macs[unifi.MAC(victim)]; ok {
				fmt.Fprintf(cmd.OutOrStdout(), "%s %q\n", name, victim)
			}
		}
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(lookupCmd)
}
