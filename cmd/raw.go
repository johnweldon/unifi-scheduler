package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
	"github.com/spf13/cobra"
)

// rawCmd represents the raw command
var rawCmd = &cobra.Command{
	Use:     "raw",
	Aliases: []string{"r"},
	Short:   "issue raw API commands",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		// Pre-validate inputs at CLI level for better error messages
		if err := validateRawCommandInputs(method, args); err != nil {
			cobra.CheckErr(fmt.Errorf("input validation failed: %w", err))
		}

		var (
			path = args[0]
			body io.Reader
		)

		if len(args) == 2 {
			body = bytes.NewBufferString(args[1])
		}

		out, err := ses.Raw(method, path, body)
		cobra.CheckErr(err)

		cmd.Printf("%s\n", out)
	},
}

var method = http.MethodGet

// validateRawCommandInputs performs pre-validation of raw command inputs
func validateRawCommandInputs(method string, args []string) error {
	// Validate method
	if err := unifi.ValidateHTTPMethod(method); err != nil {
		return fmt.Errorf("method validation: %w", err)
	}

	// Basic path validation
	if len(args) < 1 {
		return fmt.Errorf("path is required")
	}

	path := args[0]
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Basic payload validation if provided
	if len(args) > 1 {
		payload := args[1]
		if err := unifi.ValidatePayload([]byte(payload)); err != nil {
			return fmt.Errorf("payload validation: %w", err)
		}
	}

	return nil
}

func init() { // nolint: gochecknoinits
	rootCmd.AddCommand(rawCmd)
	rawCmd.Flags().StringVar(&method, "method", method, "http method")
}
