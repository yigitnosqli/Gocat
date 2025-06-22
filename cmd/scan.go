package cmd

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
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

		logger.Info("Starting port scan", "host", host, "ports", ports)

		portList, err := parsePortRange(ports)
		if err != nil {
			logger.Error("Invalid port range", "error", err)
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
	conn.Close()
	return true
}

func printResult(host string, port int, isOpen bool) {
	if isOpen {
		color.Green("[+] %s:%d - OPEN", host, port)
	} else if verboseOutput {
		color.Red("[-] %s:%d - CLOSED", host, port)
	}
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().DurationVarP(&scanTimeout, "timeout", "t", 3*time.Second, "Connection timeout for each port")
	scanCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 100, "Number of concurrent scans")
	scanCmd.Flags().StringVarP(&portRange, "ports", "p", "", "Port range to scan (e.g., 1-1000, 22,80,443)")
	scanCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false, "Show closed ports as well")
	scanCmd.Flags().BoolVarP(&onlyOpen, "open", "o", true, "Show only open ports")
	scanCmd.Flags().BoolVarP(&useUDPScan, "udp", "u", false, "Use UDP instead of TCP")
	scanCmd.Flags().BoolVarP(&forceIPv6Scan, "ipv6", "6", false, "Force IPv6")
	scanCmd.Flags().BoolVarP(&forceIPv4Scan, "ipv4", "4", false, "Force IPv4")

	// Mark conflicting flags
	scanCmd.MarkFlagsMutuallyExclusive("ipv4", "ipv6")
	scanCmd.MarkFlagsMutuallyExclusive("verbose", "open")
}