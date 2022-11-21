package cmd

import (
	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:     "user",
	Aliases: []string{"users"},
	Short:   "interact with user specific endpoints",
}

func init() { // nolint: gochecknoinits
	rootCmd.AddCommand(userCmd)
}
