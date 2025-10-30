package cmd

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var (
	sniffInterface string
	sniffFilter    string
	sniffPromisc   bool
	sniffSnaplen   int32
	sniffTimeout   time.Duration
	sniffPackets   int
	sniffVerbose   bool
	sniffDecode    bool
	sniffSaveFile  string
)

// PacketStats holds packet statistics
type PacketStats struct {
	Total     int64
	TCP       int64
	UDP       int64
	ICMP      int64
	ARP       int64
	Other     int64
	Bytes     int64
	Errors    int64
	StartTime time.Time
	mu        sync.RWMutex
}

var packetStats = &PacketStats{StartTime: time.Now()}

var snifferCmd = &cobra.Command{
	Use:     "sniffer",
	Aliases: []string{"sniff", "capture"},
	Short:   "Network packet sniffer",
	Long: `Capture and analyze network packets with BPF filtering support.

Requires root/administrator privileges for packet capture.`,
	Example: `  # Sniff all packets on eth0
  sudo gocat sniffer -i eth0
  
  # Capture only HTTP traffic
  sudo gocat sniffer -i eth0 -f "tcp port 80"
  
  # Save packets to file
  sudo gocat sniffer -i eth0 -o capture.pcap
  
  # Verbose mode with packet decoding
  sudo gocat sniffer -i eth0 --decode -v`,
	Run: runSniffer,
}

func init() {
	rootCmd.AddCommand(snifferCmd)

	snifferCmd.Flags().StringVarP(&sniffInterface, "interface", "i", "", "Network interface to sniff")
	snifferCmd.Flags().StringVarP(&sniffFilter, "filter", "f", "", "BPF filter expression")
	snifferCmd.Flags().BoolVar(&sniffPromisc, "promisc", true, "Enable promiscuous mode")
	snifferCmd.Flags().Int32Var(&sniffSnaplen, "snaplen", 65535, "Snapshot length")
	snifferCmd.Flags().DurationVar(&sniffTimeout, "timeout", 30*time.Second, "Read timeout")
	snifferCmd.Flags().IntVarP(&sniffPackets, "count", "c", 0, "Number of packets to capture (0=unlimited)")
	snifferCmd.Flags().BoolVarP(&sniffVerbose, "verbose", "v", false, "Verbose output")
	snifferCmd.Flags().BoolVar(&sniffDecode, "decode", false, "Decode packet contents")
	snifferCmd.Flags().StringVarP(&sniffSaveFile, "output", "o", "", "Save packets to pcap file")
}

func runSniffer(cmd *cobra.Command, args []string) {
	// List interfaces if none specified
	if sniffInterface == "" {
		listInterfaces()
		return
	}

	// Open device for packet capture
	handle, err := pcap.OpenLive(sniffInterface, sniffSnaplen, sniffPromisc, sniffTimeout)
	if err != nil {
		logger.Fatal("Failed to open interface %s: %v", sniffInterface, err)
	}
	defer handle.Close()

	// Apply BPF filter if specified
	if sniffFilter != "" {
		if err := handle.SetBPFFilter(sniffFilter); err != nil {
			logger.Fatal("Failed to set BPF filter: %v", err)
		}
		logger.Info("Applied filter: %s", sniffFilter)
	}

	// Create pcap writer if saving to file
	var pcapWriter *os.File
	if sniffSaveFile != "" {
		pcapWriter, err = os.Create(sniffSaveFile)
		if err != nil {
			logger.Fatal("Failed to create pcap file: %v", err)
		}
		defer pcapWriter.Close()
		logger.Info("Saving packets to: %s", sniffSaveFile)
	}

	logger.Info("Starting packet capture on %s...", sniffInterface)
	color.Yellow("Press Ctrl+C to stop\n")

	// Start statistics reporter
	go reportPacketStats()

	// Create packet source
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packetCount := 0

	for packet := range packetSource.Packets() {
		processPacket(packet, pcapWriter)

		packetCount++
		if sniffPackets > 0 && packetCount >= sniffPackets {
			break
		}
	}

	// Print final statistics
	printFinalStats()
}

func listInterfaces() {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		logger.Fatal("Failed to list interfaces: %v", err)
	}

	color.Cyan("Available network interfaces:\n")
	for _, device := range devices {
		fmt.Printf("\n📡 %s", color.GreenString(device.Name))
		if device.Description != "" {
			fmt.Printf(" - %s", device.Description)
		}
		fmt.Println()

		for _, address := range device.Addresses {
			fmt.Printf("   IP: %s", address.IP)
			if address.Netmask != nil {
				fmt.Printf("/%s", address.Netmask)
			}
			fmt.Println()
		}
	}

	fmt.Println("\nUsage: gocat sniffer -i <interface>")
}

func processPacket(packet gopacket.Packet, writer *pcap.Writer) {
	// Update statistics
	updatePacketStats(func(s *PacketStats) {
		s.Total++
		s.Bytes += int64(len(packet.Data()))
	})

	// Save to file if writer exists
	if writer != nil {
		writer.WritePacket(packet.Metadata().CaptureInfo, packet.Data())
	}

	// Process layers
	var srcIP, dstIP net.IP
	var srcPort, dstPort uint16
	var protocol string

	// Network layer
	if netLayer := packet.NetworkLayer(); netLayer != nil {
		switch netLayer.LayerType() {
		case layers.LayerTypeIPv4:
			ipv4 := netLayer.(*layers.IPv4)
			srcIP = ipv4.SrcIP
			dstIP = ipv4.DstIP
			protocol = ipv4.Protocol.String()
		case layers.LayerTypeIPv6:
			ipv6 := netLayer.(*layers.IPv6)
			srcIP = ipv6.SrcIP
			dstIP = ipv6.DstIP
			protocol = ipv6.NextHeader.String()
		case layers.LayerTypeARP:
			updatePacketStats(func(s *PacketStats) { s.ARP++ })
			if sniffVerbose {
				arp := netLayer.(*layers.ARP)
				printARPPacket(arp)
			}
			return
		}
	}

	// Transport layer
	if transLayer := packet.TransportLayer(); transLayer != nil {
		switch transLayer.LayerType() {
		case layers.LayerTypeTCP:
			tcp := transLayer.(*layers.TCP)
			srcPort = uint16(tcp.SrcPort)
			dstPort = uint16(tcp.DstPort)
			protocol = "TCP"
			updatePacketStats(func(s *PacketStats) { s.TCP++ })

			if sniffVerbose {
				printTCPPacket(srcIP, dstIP, tcp)
			}

		case layers.LayerTypeUDP:
			udp := transLayer.(*layers.UDP)
			srcPort = uint16(udp.SrcPort)
			dstPort = uint16(udp.DstPort)
			protocol = "UDP"
			updatePacketStats(func(s *PacketStats) { s.UDP++ })

			if sniffVerbose {
				printUDPPacket(srcIP, dstIP, udp)
			}

		case layers.LayerTypeICMPv4, layers.LayerTypeICMPv6:
			protocol = "ICMP"
			updatePacketStats(func(s *PacketStats) { s.ICMP++ })

			if sniffVerbose {
				printICMPPacket(srcIP, dstIP, transLayer)
			}
		}
	} else {
		updatePacketStats(func(s *PacketStats) { s.Other++ })
	}

	// Application layer analysis
	if sniffDecode && packet.ApplicationLayer() != nil {
		analyzeApplicationLayer(packet.ApplicationLayer(), srcIP, dstIP, srcPort, dstPort)
	}

	// Default output for non-verbose mode
	if !sniffVerbose && srcIP != nil && dstIP != nil {
		timestamp := packet.Metadata().Timestamp.Format("15:04:05.000000")
		fmt.Printf("[%s] %s %s:%d -> %s:%d (%d bytes)\n",
			timestamp, protocol, srcIP, srcPort, dstIP, dstPort,
			len(packet.Data()))
	}
}

func printTCPPacket(srcIP, dstIP net.IP, tcp *layers.TCP) {
	flags := []string{}
	if tcp.SYN {
		flags = append(flags, "SYN")
	}
	if tcp.ACK {
		flags = append(flags, "ACK")
	}
	if tcp.FIN {
		flags = append(flags, "FIN")
	}
	if tcp.RST {
		flags = append(flags, "RST")
	}
	if tcp.PSH {
		flags = append(flags, "PSH")
	}

	color.Green("TCP %s:%d -> %s:%d [%s] Seq=%d Ack=%d Win=%d",
		srcIP, tcp.SrcPort, dstIP, tcp.DstPort,
		strings.Join(flags, ","), tcp.Seq, tcp.Ack, tcp.Window)
}

func printUDPPacket(srcIP, dstIP net.IP, udp *layers.UDP) {
	color.Blue("UDP %s:%d -> %s:%d Len=%d",
		srcIP, udp.SrcPort, dstIP, udp.DstPort, udp.Length)
}

func printICMPPacket(srcIP, dstIP net.IP, layer gopacket.Layer) {
	switch layer.LayerType() {
	case layers.LayerTypeICMPv4:
		icmp := layer.(*layers.ICMPv4)
		color.Cyan("ICMPv4 %s -> %s Type=%d Code=%d",
			srcIP, dstIP, icmp.TypeCode.Type(), icmp.TypeCode.Code())
	case layers.LayerTypeICMPv6:
		icmp := layer.(*layers.ICMPv6)
		color.Cyan("ICMPv6 %s -> %s Type=%d Code=%d",
			srcIP, dstIP, icmp.TypeCode.Type(), icmp.TypeCode.Code())
	}
}

func printARPPacket(arp *layers.ARP) {
	op := "Unknown"
	switch arp.Operation {
	case layers.ARPRequest:
		op = "Request"
	case layers.ARPReply:
		op = "Reply"
	}

	color.Yellow("ARP %s: %s (%s) -> %s (%s)",
		op,
		net.IP(arp.SourceProtAddress),
		net.HardwareAddr(arp.SourceHwAddress),
		net.IP(arp.DstProtAddress),
		net.HardwareAddr(arp.DstHwAddress))
}

func analyzeApplicationLayer(appLayer gopacket.ApplicationLayer, srcIP, dstIP net.IP, srcPort, dstPort uint16) {
	payload := appLayer.Payload()
	if len(payload) == 0 {
		return
	}

	// Try to detect protocol
	if isHTTP(payload) {
		color.Magenta("🌐 HTTP Traffic detected: %s:%d -> %s:%d", srcIP, srcPort, dstIP, dstPort)
		printHTTPContent(payload)
	} else if isDNS(srcPort, dstPort) {
		color.Magenta("🔍 DNS Traffic detected")
	} else if isSSH(srcPort, dstPort) {
		color.Magenta("🔐 SSH Traffic detected")
	} else if isTLS(payload) {
		color.Magenta("🔒 TLS/SSL Traffic detected")
	}

	// Show hex dump for small payloads
	if sniffVerbose && len(payload) <= 256 {
		fmt.Println("Payload (hex):")
		fmt.Println(hex.Dump(payload))
	}
}

func isHTTP(payload []byte) bool {
	httpMethods := []string{"GET ", "POST ", "PUT ", "DELETE ", "HEAD ", "OPTIONS ", "HTTP/"}
	payloadStr := string(payload[:min(100, len(payload))])

	for _, method := range httpMethods {
		if strings.HasPrefix(payloadStr, method) {
			return true
		}
	}
	return false
}

func printHTTPContent(payload []byte) {
	lines := strings.Split(string(payload), "\r\n")
	for i, line := range lines {
		if i > 10 || line == "" {
			break
		}
		fmt.Printf("  %s\n", line)
	}
}

func isDNS(srcPort, dstPort uint16) bool {
	return srcPort == 53 || dstPort == 53
}

func isSSH(srcPort, dstPort uint16) bool {
	return srcPort == 22 || dstPort == 22
}

func isTLS(payload []byte) bool {
	if len(payload) < 6 {
		return false
	}
	// Check for TLS record header
	return payload[0] == 0x16 && payload[1] == 0x03
}

func updatePacketStats(fn func(*PacketStats)) {
	packetStats.mu.Lock()
	defer packetStats.mu.Unlock()
	fn(packetStats)
}

func reportPacketStats() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		packetStats.mu.RLock()
		duration := time.Since(packetStats.StartTime)
		pps := float64(packetStats.Total) / duration.Seconds()

		fmt.Printf("\n📊 Packet Statistics:\n")
		fmt.Printf("  Total: %d packets (%.1f pps)\n", packetStats.Total, pps)
		fmt.Printf("  TCP: %d | UDP: %d | ICMP: %d | ARP: %d | Other: %d\n",
			packetStats.TCP, packetStats.UDP, packetStats.ICMP,
			packetStats.ARP, packetStats.Other)
		fmt.Printf("  Data: %s\n", formatBytes(packetStats.Bytes))
		packetStats.mu.RUnlock()
	}
}

func printFinalStats() {
	packetStats.mu.RLock()
	defer packetStats.mu.RUnlock()

	duration := time.Since(packetStats.StartTime)

	// Modern header with box drawing
	fmt.Println()
	color.New(color.FgCyan, color.Bold).Println("╔══════════════════════════════════════════════╗")
	color.New(color.FgCyan, color.Bold).Println("║          📊 CAPTURE STATISTICS              ║")
	color.New(color.FgCyan, color.Bold).Println("╚══════════════════════════════════════════════╝")

	// Time and volume stats with icons
	fmt.Println()
	color.New(color.FgWhite, color.Bold).Print("⏱️  Capture Duration: ")
	color.Yellow("%v\n", duration)

	color.New(color.FgWhite, color.Bold).Print("📦 Total Packets: ")
	color.Green("%s\n", formatNumber(packetStats.Total))

	color.New(color.FgWhite, color.Bold).Print("💾 Total Data: ")
	color.Blue("%s\n", formatBytes(packetStats.Bytes))

	// Protocol distribution with progress bars
	fmt.Println()
	color.New(color.FgWhite, color.Bold).Println("📡 Protocol Distribution:")
	fmt.Println(strings.Repeat("─", 48))

	// TCP with progress bar
	tcpPercent := percent(packetStats.TCP, packetStats.Total)
	fmt.Printf("  TCP   %s %s %s (%.1f%%)\n",
		color.CyanString("█"),
		drawProgressBar(tcpPercent, 20, color.FgCyan),
		color.WhiteString("%6d", packetStats.TCP),
		tcpPercent)

	// UDP with progress bar
	udpPercent := percent(packetStats.UDP, packetStats.Total)
	fmt.Printf("  UDP   %s %s %s (%.1f%%)\n",
		color.GreenString("█"),
		drawProgressBar(udpPercent, 20, color.FgGreen),
		color.WhiteString("%6d", packetStats.UDP),
		udpPercent)

	// ICMP with progress bar
	icmpPercent := percent(packetStats.ICMP, packetStats.Total)
	fmt.Printf("  ICMP  %s %s %s (%.1f%%)\n",
		color.YellowString("█"),
		drawProgressBar(icmpPercent, 20, color.FgYellow),
		color.WhiteString("%6d", packetStats.ICMP),
		icmpPercent)

	// ARP with progress bar
	arpPercent := percent(packetStats.ARP, packetStats.Total)
	fmt.Printf("  ARP   %s %s %s (%.1f%%)\n",
		color.MagentaString("█"),
		drawProgressBar(arpPercent, 20, color.FgMagenta),
		color.WhiteString("%6d", packetStats.ARP),
		arpPercent)

	// Other with progress bar
	otherPercent := percent(packetStats.Other, packetStats.Total)
	fmt.Printf("  Other %s %s %s (%.1f%%)\n",
		color.RedString("█"),
		drawProgressBar(otherPercent, 20, color.FgRed),
		color.WhiteString("%6d", packetStats.Other),
		otherPercent)

	fmt.Println(strings.Repeat("─", 48))

	// Performance metrics
	if duration.Seconds() > 0 {
		pps := float64(packetStats.Total) / duration.Seconds()
		bps := float64(packetStats.Bytes) / duration.Seconds()

		fmt.Println()
		color.New(color.FgWhite, color.Bold).Println("⚡ Performance Metrics:")
		fmt.Printf("  Packets/sec: %s\n", color.YellowString("%.2f", pps))
		fmt.Printf("  Throughput:  %s/s\n", color.GreenString("%s", formatBytes(int64(bps))))
	}
}

// Helper function to draw progress bars
func drawProgressBar(percent float64, width int, colorAttr color.Attribute) string {
	filled := int(percent * float64(width) / 100)
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return color.New(colorAttr).Sprint(bar)
}

// Helper function to format numbers with thousand separators
func formatNumber(n int64) string {
	str := fmt.Sprintf("%d", n)
	var result []rune
	for i, r := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, r)
	}
	return string(result)
}

func percent(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) * 100 / float64(total)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
