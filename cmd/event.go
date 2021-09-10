package cmd

import (
	"github.com/spf13/cobra"
)

var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "interact with event specific endpoints",
}

func init() { // nolint: gochecknoinits
	rootCmd.AddCommand(eventCmd)
}
