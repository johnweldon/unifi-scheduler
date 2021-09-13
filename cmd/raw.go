package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

// rawCmd represents the raw command
var rawCmd = &cobra.Command{
	Use:   "raw",
	Aliases: []string{"r"},
	Short: "issue raw API commands",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		var (
			path = args[0]
			body io.Reader
		)

		if len(args) == 2 {
			body = bytes.NewBufferString(args[1])
		}

		out, err := ses.Raw(method, path, body)
		cobra.CheckErr(err)

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", out)
	},
}

var method = http.MethodGet

func init() { // nolint: gochecknoinits
	rootCmd.AddCommand(rawCmd)
	rawCmd.Flags().StringVar(&method, "method", method, "http method")
}
