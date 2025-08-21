package cmd

import (
	"github.com/spf13/cobra"
)

var eventCmd = &cobra.Command{
	Use:     "event",
	Aliases: []string{"events", "evt", "evts"},
	Short:   "Monitor and query network events",
	Long: `Event commands allow you to monitor and query various network events from your UniFi controller.
This includes client connections/disconnections, device status changes, authentication events,
and other network activity.

Events provide insight into:
  - Client connect/disconnect events
  - Device status changes
  - Authentication failures
  - Network configuration changes
  - Security-related events`,
	Example: `  # List recent events
  unifi-scheduler event list

  # Monitor connection events
  unifi-scheduler event connections

  # Use aliases
  unifi-scheduler events list
  unifi-scheduler evt connections`,
}

func init() { // nolint: gochecknoinits
	rootCmd.AddCommand(eventCmd)
}
