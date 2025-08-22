package cmd

import (
	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/pkg/output"
	"github.com/johnweldon/unifi-scheduler/pkg/unifi/display"
)

var allClients bool

var clientListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List network clients",
	Long: `List clients connected to the UniFi network. By default, shows only currently
connected clients. Use --all flag to include historical/offline clients.

Output includes:
  - Client name and hostname
  - MAC address
  - IP address
  - Connection status
  - Network usage statistics
  - Device type and vendor`,
	Example: `  # List currently connected clients
  unifi-scheduler client list

  # List all clients (including offline)
  unifi-scheduler client list --all

  # List clients with aliases
  unifi-scheduler clients ls
  unifi-scheduler cl list`,
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		fetch := ses.GetClients
		if allClients {
			fetch = ses.GetAllClients
		}

		clients, err := fetch()
		cobra.CheckErr(err)

		// Get output options and create formatter
		opts, err := getOutputOptions(cmd)
		cobra.CheckErr(err)

		formatter := opts.CreateFormatter()

		// Handle different output formats
		switch opts.Format {
		case output.FormatTable:
			display.ClientsTable(cmd.OutOrStdout(), clients).Render()
		case output.FormatJSON, output.FormatYAML:
			err = formatter.Write(clients)
			cobra.CheckErr(err)
		}
	},
}

func init() { // nolint: gochecknoinits
	clientCmd.AddCommand(clientListCmd)

	clientListCmd.Flags().BoolVar(&allClients, "all", allClients, "show all clients (including offline/historical)")
}
