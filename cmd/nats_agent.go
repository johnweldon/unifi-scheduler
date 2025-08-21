package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/johnweldon/unifi-scheduler/pkg/nats"
)

var natsAgentCmd = &cobra.Command{
	Use:     "agent",
	Aliases: []string{"agt", "a"},
	Short:   "Run NATS caching agent (long-running service)",
	Long: `The NATS agent continuously polls the UniFi controller and caches data in NATS
key-value storage for fast access by other unifi-scheduler instances.

The agent:
  - Polls UniFi controller for active clients and events
  - Stores data in NATS JetStream key-value buckets
  - Maintains client name-to-MAC mappings
  - Provides timestamped data for monitoring
  - Runs indefinitely until interrupted (Ctrl+C)

Data stored:
  - 'unifi.detail.active' - Currently active clients
  - 'unifi.detail.events' - Recent connection events
  - 'unifi.by_mac.*' - Client names indexed by MAC address

This enables other instances to query cached data via 'nats clients' and
'nats connections' commands without hitting the UniFi controller directly.`,
	Example: `  # Run agent with default NATS server
  unifi-scheduler nats agent

  # Run agent with custom NATS server
  unifi-scheduler --nats_url nats://monitor-server:4222 nats agent

  # Run agent with authentication
  unifi-scheduler --nats_url nats://server:4222 --nats_creds /path/to/creds nats agent`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("Version: %s\n", Version)

		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		opts := []nats.ClientOpt{
			nats.OptNATSUrl(natsURL),
			nats.OptCreds(natsCreds),
			nats.OptConnectTimeout(natsConnTimeout),
			nats.OptWriteTimeout(natsConnTimeout),
			nats.OptOperationTimeout(natsOpTimeout),
			nats.OptStreamReplicas(streamReplicas),
			nats.OptKVReplicas(kvReplicas),
		}

		a := nats.NewAgent(ses, baseSubject, opts...)
		cobra.CheckErr(a.Start(ctx))

		markInterval := time.After(1 * time.Second)
		hourInterval := time.After(1 * time.Hour)

		for {
			select {
			case <-ctx.Done():
				cmd.Printf("quitting...\n")
				return
			case <-markInterval:
				markInterval = time.After(1 * time.Minute)
				cmd.Printf(".")
			case <-hourInterval:
				hourInterval = time.After(1 * time.Hour)
				cmd.Printf("H\n")
			}

			_ = os.Stdout.Sync()
		}
	},
}

func init() { // nolint: gochecknoinits
	natsCmd.AddCommand(natsAgentCmd)
}
