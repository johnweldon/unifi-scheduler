package cmd

import (
	"fmt"
	"net"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/johnweldon/unifi-scheduler/pkg/nats"
)

var (
	prefixCheckHostname string
	prefixCheckSubject  string
	prefixCheckForce    bool
)

var natsPrefixCheckCmd = &cobra.Command{
	Use:     "prefix-check",
	Aliases: []string{"prefix", "pc"},
	Short:   "Check delegated IPv6 prefix and publish updates to NATS",
	Long: `Check the IPv6 delegated prefix from the UniFi gateway and compare it with
the prefix of an AAAA record for a specified hostname. If the prefixes differ,
publish a notification message with the updated prefix to a NATS subject.

Use --force to publish the message regardless of whether the prefix has changed.

This command is useful for monitoring IPv6 prefix changes and notifying other
systems when the delegated prefix has changed.`,
	Example: `  # Check prefix and publish updates
  unifi-scheduler --nats_url nats://server:4222 nats prefix-check \
    --hostname example.com --subject ipv6.prefix.update

  # Force publish even if prefix is unchanged
  unifi-scheduler --nats_url nats://server:4222 nats prefix-check \
    --hostname example.com --subject ipv6.prefix.update --force

  # Use environment variables
  export UNIFI_PREFIX_CHECK_HOSTNAME=example.com
  export UNIFI_PREFIX_CHECK_SUBJECT=ipv6.prefix.update
  unifi-scheduler --nats_url nats://server:4222 nats prefix-check`,
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize session to get delegated prefix
		ses, err := initSession(cmd)
		cobra.CheckErr(err)

		// Get devices to find the gateway
		devices, err := ses.GetDevices()
		cobra.CheckErr(err)

		// Find gateway device and extract delegated prefix
		var delegatedPrefix string
		for _, device := range devices {
			// Gateway devices typically have type "ugw" or "udm" (UDM Pro)
			if device.DeviceType == "ugw" || device.DeviceType == "udm" {
				prefix := device.GetIPv6DelegatedPrefix()
				if prefix != "" {
					delegatedPrefix = prefix
					break
				}
			}
		}

		if delegatedPrefix == "" {
			cobra.CheckErr(fmt.Errorf("no IPv6 delegated prefix found on gateway device"))
		}

		// Perform DNS AAAA lookup for the hostname
		ips, err := net.LookupIP(prefixCheckHostname)
		cobra.CheckErr(err)

		// Find the first IPv6 address
		var hostnameIPv6 net.IP
		for _, ip := range ips {
			if ip.To4() == nil && ip.To16() != nil {
				hostnameIPv6 = ip
				break
			}
		}

		if hostnameIPv6 == nil {
			cobra.CheckErr(fmt.Errorf("no AAAA record found for hostname %q", prefixCheckHostname))
		}

		// Extract the prefix from the hostname's IPv6 address
		// Use the same delegation size as detected from the gateway
		hostnamePrefix, err := extractPrefix(hostnameIPv6, delegatedPrefix)
		cobra.CheckErr(err)

		// Compare the prefixes
		prefixChanged := delegatedPrefix != hostnamePrefix
		if !prefixChanged && !prefixCheckForce {
			cmd.Printf("Prefix unchanged: %s\n", delegatedPrefix)
			return
		}

		// Publish notification to NATS
		if prefixChanged {
			cmd.Printf("Prefix changed: %s -> %s\n", hostnamePrefix, delegatedPrefix)
		} else {
			cmd.Printf("Publishing unchanged prefix: %s (forced)\n", delegatedPrefix)
		}

		opts := []nats.ClientOpt{nats.OptNATSUrl(natsURL), nats.OptCreds(natsCreds)}
		p := nats.NewPublisher(opts...)

		msg := map[string]interface{}{
			"hostname":   prefixCheckHostname,
			"old_prefix": hostnamePrefix,
			"new_prefix": delegatedPrefix,
			"changed":    prefixChanged,
		}

		err = p.Publish(prefixCheckSubject, msg)
		cobra.CheckErr(err)

		cmd.Printf("Published prefix update to subject %q\n", prefixCheckSubject)
	},
}

func extractPrefix(ip net.IP, delegatedPrefix string) (string, error) {
	// Parse the delegated prefix to determine the delegation size
	parts := strings.Split(delegatedPrefix, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid prefix format: %s", delegatedPrefix)
	}

	delegationSize := parts[1]

	// Convert IP to 16-byte representation
	ipBytes := ip.To16()
	if ipBytes == nil {
		return "", fmt.Errorf("invalid IPv6 address")
	}

	// Zero out the host portion based on delegation size
	var prefixBytes [16]byte
	copy(prefixBytes[:], ipBytes)

	switch delegationSize {
	case "48":
		// Zero out bytes 6-15
		for i := 6; i < 16; i++ {
			prefixBytes[i] = 0
		}
	case "56":
		// Zero out bytes 7-15
		for i := 7; i < 16; i++ {
			prefixBytes[i] = 0
		}
	case "64":
		// Zero out bytes 8-15
		for i := 8; i < 16; i++ {
			prefixBytes[i] = 0
		}
	default:
		return "", fmt.Errorf("unsupported delegation size: %s", delegationSize)
	}

	// Format as IPv6 prefix
	prefix := net.IP(prefixBytes[:]).String()
	return fmt.Sprintf("%s/%s", prefix, delegationSize), nil
}

func init() { // nolint: gochecknoinits
	natsCmd.AddCommand(natsPrefixCheckCmd)

	flags := natsPrefixCheckCmd.Flags()
	flags.StringVar(&prefixCheckHostname, "hostname", "", "hostname to query for AAAA record (env: UNIFI_PREFIX_CHECK_HOSTNAME)")
	flags.StringVar(&prefixCheckSubject, "subject", "", "NATS subject to publish prefix updates (env: UNIFI_PREFIX_CHECK_SUBJECT)")
	flags.BoolVar(&prefixCheckForce, "force", false, "publish message even if prefix is unchanged (env: UNIFI_PREFIX_CHECK_FORCE)")

	// Bind custom environment variable names to flags
	cobra.CheckErr(viper.BindEnv("hostname", "UNIFI_PREFIX_CHECK_HOSTNAME"))
	cobra.CheckErr(viper.BindEnv("subject", "UNIFI_PREFIX_CHECK_SUBJECT"))
	cobra.CheckErr(viper.BindEnv("force", "UNIFI_PREFIX_CHECK_FORCE"))

	cobra.CheckErr(natsPrefixCheckCmd.MarkFlagRequired("hostname"))
	cobra.CheckErr(natsPrefixCheckCmd.MarkFlagRequired("subject"))
}
