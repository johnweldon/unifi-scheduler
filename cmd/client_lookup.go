package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/unifi"
)

var lookupCmd = &cobra.Command{
	Use:   "lookup",
	Short: "lookup client",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		names, err := ses.GetNames()
		cobra.CheckErr(err)

		macs, err := ses.GetMACs()
		cobra.CheckErr(err)

		for _, victim := range args {
			if mac, ok := names[victim]; ok {
				fmt.Fprintf(cmd.OutOrStdout(), "%s %q\n", mac, victim)
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
