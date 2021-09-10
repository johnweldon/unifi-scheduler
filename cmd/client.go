package cmd

import (
	"github.com/spf13/cobra"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "interact with client specific endpoints",
}

func init() { // nolint: gochecknoinits
	rootCmd.AddCommand(clientCmd)
}
