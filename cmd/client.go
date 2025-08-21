package cmd

import (
	"github.com/spf13/cobra"
)

var clientCmd = &cobra.Command{
	Use:     "client",
	Aliases: []string{"cl", "clients"},
	Short:   "Manage network clients (devices connected to the network)",
	Long: `Client management commands allow you to interact with devices connected to your UniFi network.
You can list, block, unblock, kick, forget, and lookup clients by name, hostname, or MAC address.

Clients can be identified by:
  - Display name ("iPhone", "John's Laptop")
  - Hostname ("johns-macbook", "smart-tv")
  - MAC address ("aa:bb:cc:dd:ee:ff")
  - Partial matches of any of the above`,
	Example: `  # List all connected clients
  unifi-scheduler client list

  # List all clients including offline/historical
  unifi-scheduler client list --all

  # Block a client by name
  unifi-scheduler client block "Problem Device"

  # Block a client by MAC address
  unifi-scheduler client block "aa:bb:cc:dd:ee:ff"

  # Unblock a client
  unifi-scheduler client unblock "iPhone"

  # Kick (disconnect) a client
  unifi-scheduler client kick "johns-laptop"`,
}

func init() { // nolint: gochecknoinits
	rootCmd.AddCommand(clientCmd)
}
