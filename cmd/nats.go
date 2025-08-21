package cmd

import (
	_ "net/http/pprof"

	"github.com/spf13/cobra"
)

var natsCmd = &cobra.Command{
	Use:     "nats",
	Aliases: []string{"nts", "n"},
	Short:   "Query cached UniFi data from NATS storage",
	Long: `NATS commands allow you to query UniFi network data that has been cached in NATS
key-value storage by a running unifi-scheduler nats agent.

The NATS agent continuously collects and caches:
  - Active client information in the 'unifi.detail.active' key
  - Network connection events in the 'unifi.detail.events' key
  - Client name mappings by MAC address in 'unifi.by_mac.*' keys

These commands provide fast access to cached data without directly querying
the UniFi controller, useful for monitoring and reporting.

Requires:
  1. A NATS server with JetStream enabled
  2. A running 'unifi-scheduler nats agent' to populate the cache`,
	Example: `  # View currently active clients from cache
  unifi-scheduler --nats_url nats://server:4222 nats clients

  # View recent connection events from cache
  unifi-scheduler --nats_url nats://server:4222 nats connections

  # Run agent to populate NATS cache (long-running)
  unifi-scheduler --nats_url nats://server:4222 nats agent`,
}

const (
	baseSubject   = "unifi"
	natsURLFlag   = "nats_url"
	natsCredsFlag = "nats_creds"
)

var (
	natsURL   = ""
	natsCreds = ""
)

func init() { // nolint: gochecknoinits
	pf := natsCmd.PersistentFlags()

	pf.StringVar(&natsURL, natsURLFlag, "nats://localhost:4222", "NATS server URL (e.g., nats://localhost:4222)")
	// NATS URL is required only for NATS commands
	_ = cobra.MarkFlagRequired(pf, natsURLFlag)

	pf.StringVar(&natsCreds, natsCredsFlag, natsCreds, "NATS credentials file path for authentication")

	rootCmd.AddCommand(natsCmd)
}
