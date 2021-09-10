package cmd

import (
	"github.com/spf13/cobra"
)

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "interact with device specific endpoints",
}

func init() { // nolint: gochecknoinits
	rootCmd.AddCommand(deviceCmd)
}
