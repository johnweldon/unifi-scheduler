package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var setUserCmd = &cobra.Command{
	Use:     "set",
	Short:   "set user details",
	Example: "<mac> <name> <ip>",
	Args:    cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		check(err, ses)

		out, err := ses.SetUserDetails(args[0], args[1], args[2])
		check(err, out)

		cmd.Printf("%s\n", out)
	},
}

func init() { // nolint: gochecknoinits
	userCmd.AddCommand(setUserCmd)
}

func check(err error, args ...any) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		for _, arg := range args {
			fmt.Fprintf(os.Stderr, " - %v\n", arg)
		}
		os.Exit(1)
	}
}
