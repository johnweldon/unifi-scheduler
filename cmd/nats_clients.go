package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/pkg/nats"
	"github.com/johnweldon/unifi-scheduler/pkg/output"
	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
	"github.com/johnweldon/unifi-scheduler/pkg/unifi/display"
)

var natsClientsCmd = &cobra.Command{
	Use:     "clients",
	Aliases: []string{"client", "cl", "c"},
	Short:   "Display cached active clients from NATS storage",
	Long: `Display currently active UniFi clients from NATS cache. This data is populated
by a running 'unifi-scheduler nats agent' and provides fast access to client
information without querying the UniFi controller directly.

The data shown is the last cached snapshot of active clients, including:
  - Client names and hostnames
  - MAC addresses and IP addresses
  - Connection status and network usage
  - Device types and vendors

Data freshness depends on the agent polling interval.`,
	Example: `  # Show cached active clients
  unifi-scheduler --nats_url nats://server:4222 nats clients

  # Use alias
  unifi-scheduler --nats_url nats://server:4222 nats cl`,
	Run: func(cmd *cobra.Command, args []string) {
		opts := []nats.ClientOpt{nats.OptNATSUrl(natsURL), nats.OptCreds(natsCreds)}
		s := nats.NewSubscriber(opts...)

		var into []unifi.Client
		cobra.CheckErr(s.Get(nats.DetailBucket(baseSubject), nats.ActiveKey, &into))

		// Get output options and create formatter
		outputOpts, err := getOutputOptions(cmd)
		cobra.CheckErr(err)

		formatter := outputOpts.CreateFormatter()

		// Handle different output formats
		switch outputOpts.Format {
		case output.FormatTable:
			display.ClientsTable(cmd.OutOrStdout(), into).Render()
			fmt.Fprintf(cmd.OutOrStdout(), "%70s\n", time.Now().Format(time.RFC1123))
		case output.FormatJSON, output.FormatYAML:
			// For structured formats, include timestamp in the data
			result := map[string]interface{}{
				"clients":   into,
				"timestamp": time.Now().Format(time.RFC3339),
				"count":     len(into),
			}
			err = formatter.Write(result)
			cobra.CheckErr(err)
		}
	},
}

func init() { // nolint: gochecknoinits
	natsCmd.AddCommand(natsClientsCmd)
}
