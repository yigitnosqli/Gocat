package cmd

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var (
	scanTimeout    time.Duration
	concurrency    int
	portRange      string
	verboseOutput  bool
	onlyOpen       bool
	useUDPScan     bool
	forceIPv6Scan  bool
	forceIPv4Scan  bool
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan [host] [ports]",
	Short: "Port scanner for network reconnaissance",
	Long: `A fast and efficient port scanner that can scan single ports, port ranges,
or common ports on target hosts. Supports both TCP and UDP scanning with
configurable concurrency and timeout settings.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host := args[0]
		ports := "1-1000"
		if len(args) > 1 {
			ports = args[1]
		}

		if portRange != "" {
			ports = portRange
		}

		logger.Info("Starting port scan on %s for ports %s", host, ports)

		portList, err := parsePortRange(ports)
		if err != nil {
			logger.Error("Invalid port range: %v", err)
			return
		}

		scanPorts(host, portList)
	},
}

func parsePortRange(portStr string) ([]int, error) {
	var ports []int

	if strings.Contains(portStr, ",") {
		// Handle comma-separated ports: 22,80,443
		portStrs := strings.Split(portStr, ",")
		for _, p := range portStrs {
			port, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", p)
			}
			ports = append(ports, port)
		}
	} else if strings.Contains(portStr, "-") {
		// Handle port range: 1-1000
		parts := strings.Split(portStr, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid port range format")
		}
		start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid start port: %s", parts[0])
		}
		end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid end port: %s", parts[1])
		}
		for i := start; i <= end; i++ {
			ports = append(ports, i)
		}
	} else {
		// Single port
		port, err := strconv.Atoi(strings.TrimSpace(portStr))
		if err != nil {
			return nil, fmt.Errorf("invalid port: %s", portStr)
		}
		ports = append(ports, port)
	}

	return ports, nil
}

func scanPorts(host string, ports []int) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	for _, port := range ports {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			isOpen := scanPort(host, p)
			if isOpen || !onlyOpen {
				printResult(host, p, isOpen)
			}
		}(port)
	}

	wg.Wait()
}

func scanPort(host string, port int) bool {
	network := "tcp"
	if useUDPScan {
		network = "udp"
	}

	if forceIPv6Scan {
		network += "6"
	} else if forceIPv4Scan {
		network += "4"
	}

	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout(network, address, scanTimeout)
	if err != nil {
		return false
	}
	if err := conn.Close(); err != nil {
		// Log close error but don't fail the scan
		logger.Warn("Failed to close connection: %v", err)
	}
	return true
}

func printResult(host string, port int, isOpen bool) {
	theme := logger.GetCurrentTheme()
	if isOpen {
		if _, err := theme.Success.Printf("[+] %s:%d - OPEN\n", host, port); err != nil {
			log.Printf("Error printing success message: %v", err)
		}
	} else if verboseOutput {
		if _, err := theme.Error.Printf("[-] %s:%d - CLOSED\n", host, port); err != nil {
			log.Printf("Error printing error message: %v", err)
		}
	}
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().DurationVar(&scanTimeout, "scan-timeout", 3*time.Second, "Connection timeout for each port")
	scanCmd.Flags().IntVar(&concurrency, "concurrency", 100, "Number of concurrent scans")
	scanCmd.Flags().StringVar(&portRange, "ports", "", "Port range to scan (e.g., 1-1000, 22,80,443)")
	scanCmd.Flags().BoolVar(&verboseOutput, "scan-verbose", false, "Show closed ports as well")
	scanCmd.Flags().BoolVar(&onlyOpen, "open", true, "Show only open ports")
	scanCmd.Flags().BoolVar(&useUDPScan, "scan-udp", false, "Use UDP instead of TCP for scanning")
	scanCmd.Flags().BoolVar(&forceIPv6Scan, "scan-ipv6", false, "Force IPv6 for scanning")
	scanCmd.Flags().BoolVar(&forceIPv4Scan, "scan-ipv4", false, "Force IPv4 for scanning")

	// Mark conflicting flags
	scanCmd.MarkFlagsMutuallyExclusive("scan-ipv4", "scan-ipv6")
	scanCmd.MarkFlagsMutuallyExclusive("scan-verbose", "open")
}