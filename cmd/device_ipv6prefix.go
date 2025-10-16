package cmd

import (
	"github.com/spf13/cobra"
)

var ipv6PrefixCmd = &cobra.Command{
	Use:     "ipv6-prefix",
	Aliases: []string{"ipv6prefix", "ipv6"},
	Short:   "display the IPv6 delegated prefix from the gateway device",
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		devices, err := ses.GetDevices()
		cobra.CheckErr(err)

		// Find gateway devices with IPv6 prefix
		found := false
		for _, device := range devices {
			// Gateway devices typically have type "ugw" or "udm" (UDM Pro)
			if device.DeviceType == "ugw" || device.DeviceType == "udm" {
				prefix := device.GetIPv6DelegatedPrefix()
				if prefix != "" {
					cmd.OutOrStdout().Write([]byte(prefix + "\n"))
					found = true
				}
			}
		}

		if !found {
			cmd.PrintErrln("No gateway device with IPv6 prefix found")
		}
	},
}

func init() { // nolint: gochecknoinits
	deviceCmd.AddCommand(ipv6PrefixCmd)
}
