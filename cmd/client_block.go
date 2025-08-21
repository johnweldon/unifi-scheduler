package cmd

import (
	"github.com/spf13/cobra"
)

var blockCmd = &cobra.Command{
	Use:     "block <client>",
	Aliases: []string{"blk", "bl"},
	Short:   "Block network access for specified clients",
	Long: `Block network access for one or more clients. Blocked clients will be unable
to access the internet or local network resources (depending on controller settings).

Clients can be specified by:
  - Display name ("iPhone", "John's Laptop")
  - Hostname ("johns-macbook", "smart-tv")
  - MAC address ("aa:bb:cc:dd:ee:ff")
  - Partial matches of any of the above

Multiple clients can be blocked in a single command by providing multiple arguments.`,
	Example: `  # Block a client by display name
  unifi-scheduler client block "Problem Device"

  # Block a client by MAC address
  unifi-scheduler client block "aa:bb:cc:dd:ee:ff"

  # Block a client by hostname
  unifi-scheduler client block "suspicious-laptop"

  # Block multiple clients
  unifi-scheduler client block "Device1" "Device2" "aa:bb:cc:dd:ee:ff"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		macs, err := ses.GetMACsBy(args...)
		cobra.CheckErr(err)

		_, err = ses.Block(macs...)
		cobra.CheckErr(err)

		cmd.Printf("ok\n")
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(blockCmd)
}
