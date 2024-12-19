package cmd

import (
	_ "net/http/pprof"

	"github.com/spf13/cobra"
)

var natsCmd = &cobra.Command{
	Use:     "nats",
	Aliases: []string{"nts", "n"},
	Short:   "nats tools",
}

const (
	baseSubject   = "unifi"
	natsURLFlag   = "nats_url"
	natsCredsFlag = "nats_creds"
)

var (
	natsURL   = "nats://localhost:4222"
	natsCreds = ""
)

func init() { // nolint: gochecknoinits
	pf := natsCmd.PersistentFlags()

	pf.StringVar(&natsURL, natsURLFlag, natsURL, "NATS URL")
	_ = cobra.MarkFlagRequired(pf, natsURLFlag)

	pf.StringVar(&natsCreds, natsCredsFlag, natsCreds, "NATS Credentials File")

	rootCmd.AddCommand(natsCmd)
}
