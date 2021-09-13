package cmd

import (
	"github.com/spf13/cobra"
)

var eventCmd = &cobra.Command{
	Use:     "event",
	Aliases: []string{"events", "evt", "evts"},
	Short:   "interact with event specific endpoints",
}

func init() { // nolint: gochecknoinits
	rootCmd.AddCommand(eventCmd)
}
