package cmd

import (
	"fmt"
	"strings"

	"github.com/ibrahmsql/gocat/internal/nekodns"
	"github.com/spf13/cobra"
)

var (
	nekoDNSIP       string
	nekoDNSPort     int
	nekoDNSProtocol string
	nekoDNSSilent   bool
)

// nekoDNSCmd represents the nekodns command
var nekoDNSCmd = &cobra.Command{
	Use:   "nekodns",
	Short: "DNS-based reverse shell server",
	Long: `NekoDNS - Experimental Reverse DNS Shell

Leverages DNS resolutions to establish a Reverse Shell over DNS.
Communication is performed through DNS queries (AAAA/A records) that carry
commands and responses as fragmented and reversed hexadecimal data.

This is a Go implementation of the original NekoDNS by @JoelGMSec.`,
	Example: `  # Start UDP DNS server
  gocat nekodns --ip 0.0.0.0 --port 53 --protocol udp

  # Start TCP DNS server
  gocat nekodns --ip 0.0.0.0 --port 53 --protocol tcp

  # Silent mode (no banner)
  gocat nekodns --ip 0.0.0.0 --port 53 --protocol udp --silent
  
  # Custom port
  gocat nekodns --ip 0.0.0.0 --port 5353 --protocol udp`,
	RunE: runNekoDNS,
}

func init() {
	rootCmd.AddCommand(nekoDNSCmd)

	nekoDNSCmd.Flags().StringVar(&nekoDNSIP, "ip", "0.0.0.0", "IP address to listen on")
	nekoDNSCmd.Flags().IntVar(&nekoDNSPort, "port", 53, "Port to listen on")
	nekoDNSCmd.Flags().StringVar(&nekoDNSProtocol, "protocol", "udp", "Protocol to use (udp/tcp)")
	nekoDNSCmd.Flags().BoolVar(&nekoDNSSilent, "silent", false, "Silent mode (no banner)")
}

func runNekoDNS(cmd *cobra.Command, args []string) error {
	protocol := strings.ToLower(nekoDNSProtocol)
	if protocol != "udp" && protocol != "tcp" {
		return fmt.Errorf("invalid protocol: %s (use 'udp' or 'tcp')", nekoDNSProtocol)
	}

	// Create and start the NekoDNS server
	server := nekodns.NewServer(nekoDNSIP, nekoDNSPort, protocol, nekoDNSSilent)
	return server.Start()
}
