package cmd

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var (
	benchTarget      string
	benchPort        int
	benchConnections int
	benchDuration    time.Duration
	benchPacketSize  int
	benchProtocol    string
	benchRate        int
	benchVerbose     bool
)

// BenchmarkResults holds benchmark statistics
type BenchmarkResults struct {
	TotalConnections int64
	SuccessfulConns  int64
	FailedConns      int64
	TotalBytes       int64
	TotalPackets     int64
	MinLatency       time.Duration
	MaxLatency       time.Duration
	AvgLatency       time.Duration
	StartTime        time.Time
	EndTime          time.Time
	Errors           []string
	mu               sync.RWMutex
}

var benchResults = &BenchmarkResults{
	MinLatency: time.Hour, // Start with high value
}

var benchmarkCmd = &cobra.Command{
	Use:     "benchmark",
	Aliases: []string{"bench", "stress"},
	Short:   "Network performance benchmark and stress testing",
	Long: `Perform network performance testing and stress testing against a target.

WARNING: Only use against systems you own or have permission to test.`,
	Example: `  # Basic TCP benchmark
  gocat benchmark -t example.com -p 80 -c 100 -d 30s
  
  # UDP stress test
  gocat benchmark -t 192.168.1.1 -p 53 --protocol udp -c 1000
  
  # Rate-limited test
  gocat benchmark -t localhost -p 8080 --rate 100 --packet-size 1024`,
	Run: runBenchmark,
}

func init() {
	rootCmd.AddCommand(benchmarkCmd)

	benchmarkCmd.Flags().StringVar(&benchTarget, "target", "", "Target host to benchmark")
	benchmarkCmd.Flags().IntVar(&benchPort, "port", 80, "Target port")
	benchmarkCmd.Flags().IntVar(&benchConnections, "connections", 10, "Number of concurrent connections")
	benchmarkCmd.Flags().DurationVar(&benchDuration, "duration", 10*time.Second, "Test duration")
	benchmarkCmd.Flags().IntVar(&benchPacketSize, "packet-size", 64, "Size of test packets in bytes")
	benchmarkCmd.Flags().StringVar(&benchProtocol, "protocol", "tcp", "Protocol to use (tcp/udp)")
	benchmarkCmd.Flags().IntVar(&benchRate, "rate", 0, "Maximum requests per second (0=unlimited)")
	benchmarkCmd.Flags().BoolVar(&benchVerbose, "bench-verbose", false, "Verbose output")

	benchmarkCmd.MarkFlagRequired("target")
}

func runBenchmark(cmd *cobra.Command, args []string) {
	if benchTarget == "" {
		logger.Fatal("Target host is required")
	}

	logger.Warn("⚠️  Starting benchmark against %s:%d", benchTarget, benchPort)
	logger.Warn("⚠️  Only use this tool against systems you own or have permission to test")
	
	time.Sleep(3 * time.Second) // Give user time to cancel

	benchResults.StartTime = time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), benchDuration)
	defer cancel()

	// Start progress reporter
	go reportBenchProgress(ctx)

	// Rate limiter
	var rateLimiter <-chan time.Time
	if benchRate > 0 {
		ticker := time.NewTicker(time.Second / time.Duration(benchRate))
		defer ticker.Stop()
		rateLimiter = ticker.C
	}

	// Start workers
	var wg sync.WaitGroup
	workerChan := make(chan struct{}, benchConnections)

	logger.Info("Starting %d concurrent connections for %v", benchConnections, benchDuration)

	for i := 0; i < benchConnections; i++ {
		wg.Add(1)
		go benchmarkWorker(ctx, &wg, workerChan, rateLimiter, i)
	}

	// Wait for completion
	wg.Wait()
	benchResults.EndTime = time.Now()

	// Print final results
	printBenchmarkResults()
}

func benchmarkWorker(ctx context.Context, wg *sync.WaitGroup, _ chan struct{}, 
	rateLimiter <-chan time.Time, workerID int) {
	defer wg.Done()

	testData := make([]byte, benchPacketSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Rate limiting
			if rateLimiter != nil {
				select {
				case <-rateLimiter:
				case <-ctx.Done():
					return
				}
			}

			// Perform benchmark operation
			switch benchProtocol {
			case "tcp":
				benchmarkTCP(testData, workerID)
			case "udp":
				benchmarkUDP(testData, workerID)
			default:
				logger.Error("Unknown protocol: %s", benchProtocol)
				return
			}
		}
	}
}

func benchmarkTCP(data []byte, workerID int) {
	startTime := time.Now()
	
	// Connect
	conn, err := net.DialTimeout("tcp", 
		fmt.Sprintf("%s:%d", benchTarget, benchPort), 
		5*time.Second)
	
	if err != nil {
		atomic.AddInt64(&benchResults.FailedConns, 1)
		addError(fmt.Sprintf("Worker %d: Connection failed: %v", workerID, err))
		return
	}
	defer conn.Close()

	atomic.AddInt64(&benchResults.SuccessfulConns, 1)
	atomic.AddInt64(&benchResults.TotalConnections, 1)

	// Send data
	n, err := conn.Write(data)
	if err != nil {
		atomic.AddInt64(&benchResults.FailedConns, 1)
		addError(fmt.Sprintf("Worker %d: Write failed: %v", workerID, err))
		return
	}

	atomic.AddInt64(&benchResults.TotalBytes, int64(n))
	atomic.AddInt64(&benchResults.TotalPackets, 1)

	// Read response (if any)
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	buffer := make([]byte, 1024)
	if n, err := conn.Read(buffer); err == nil && n > 0 {
		atomic.AddInt64(&benchResults.TotalBytes, int64(n))
	}

	// Calculate latency
	latency := time.Since(startTime)
	updateLatency(latency)

	if benchVerbose {
		logger.Debug("Worker %d: Connection successful, latency: %v", workerID, latency)
	}
}

func benchmarkUDP(data []byte, workerID int) {
	startTime := time.Now()

	// Resolve address
	addr, err := net.ResolveUDPAddr("udp", 
		fmt.Sprintf("%s:%d", benchTarget, benchPort))
	if err != nil {
		atomic.AddInt64(&benchResults.FailedConns, 1)
		addError(fmt.Sprintf("Worker %d: Address resolution failed: %v", workerID, err))
		return
	}

	// Create connection
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		atomic.AddInt64(&benchResults.FailedConns, 1)
		addError(fmt.Sprintf("Worker %d: UDP dial failed: %v", workerID, err))
		return
	}
	defer conn.Close()

	atomic.AddInt64(&benchResults.TotalConnections, 1)

	// Send data
	n, err := conn.Write(data)
	if err != nil {
		atomic.AddInt64(&benchResults.FailedConns, 1)
		addError(fmt.Sprintf("Worker %d: UDP write failed: %v", workerID, err))
		return
	}

	atomic.AddInt64(&benchResults.SuccessfulConns, 1)
	atomic.AddInt64(&benchResults.TotalBytes, int64(n))
	atomic.AddInt64(&benchResults.TotalPackets, 1)

	// Try to read response
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	buffer := make([]byte, 1024)
	if n, err := conn.Read(buffer); err == nil && n > 0 {
		atomic.AddInt64(&benchResults.TotalBytes, int64(n))
	}

	// Calculate latency
	latency := time.Since(startTime)
	updateLatency(latency)

	if benchVerbose {
		logger.Debug("Worker %d: UDP packet sent, latency: %v", workerID, latency)
	}
}

func updateLatency(latency time.Duration) {
	benchResults.mu.Lock()
	defer benchResults.mu.Unlock()

	if latency < benchResults.MinLatency {
		benchResults.MinLatency = latency
	}
	if latency > benchResults.MaxLatency {
		benchResults.MaxLatency = latency
	}

	// Simple moving average
	totalConns := benchResults.SuccessfulConns
	if totalConns > 0 {
		currentAvg := benchResults.AvgLatency
		benchResults.AvgLatency = (currentAvg*time.Duration(totalConns-1) + latency) / time.Duration(totalConns)
	}
}

func addError(errMsg string) {
	benchResults.mu.Lock()
	defer benchResults.mu.Unlock()
	
	benchResults.Errors = append(benchResults.Errors, errMsg)
	
	// Keep only last 10 errors
	if len(benchResults.Errors) > 10 {
		benchResults.Errors = benchResults.Errors[len(benchResults.Errors)-10:]
	}
}

func reportBenchProgress(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(benchResults.StartTime)
			connsPerSec := float64(atomic.LoadInt64(&benchResults.TotalConnections)) / elapsed.Seconds()
			bytesPerSec := float64(atomic.LoadInt64(&benchResults.TotalBytes)) / elapsed.Seconds()

			fmt.Printf("\r⚡ Connections: %d | Success: %d | Failed: %d | %.1f conn/s | %s/s",
				atomic.LoadInt64(&benchResults.TotalConnections),
				atomic.LoadInt64(&benchResults.SuccessfulConns),
				atomic.LoadInt64(&benchResults.FailedConns),
				connsPerSec,
				formatBytes(int64(bytesPerSec)))
		}
	}
}

func printBenchmarkResults() {
	duration := benchResults.EndTime.Sub(benchResults.StartTime)
	
	color.Cyan("\n\n=== Benchmark Results ===")
	fmt.Printf("Target: %s:%d (%s)\n", benchTarget, benchPort, benchProtocol)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Concurrent Connections: %d\n", benchConnections)
	
	if benchRate > 0 {
		fmt.Printf("Rate Limit: %d req/s\n", benchRate)
	}
	
	fmt.Println("\n📊 Connection Statistics:")
	fmt.Printf("  Total Attempts: %d\n", benchResults.TotalConnections)
	fmt.Printf("  Successful: %d (%.1f%%)\n", 
		benchResults.SuccessfulConns,
		float64(benchResults.SuccessfulConns)*100/float64(benchResults.TotalConnections))
	fmt.Printf("  Failed: %d (%.1f%%)\n",
		benchResults.FailedConns,
		float64(benchResults.FailedConns)*100/float64(benchResults.TotalConnections))
	
	fmt.Println("\n📈 Performance Metrics:")
	fmt.Printf("  Connections/sec: %.2f\n", 
		float64(benchResults.TotalConnections)/duration.Seconds())
	fmt.Printf("  Data transferred: %s\n", formatBytes(benchResults.TotalBytes))
	fmt.Printf("  Throughput: %s/s\n", 
		formatBytes(int64(float64(benchResults.TotalBytes)/duration.Seconds())))
	fmt.Printf("  Packets sent: %d\n", benchResults.TotalPackets)
	
	if benchResults.SuccessfulConns > 0 {
		fmt.Println("\n⏱️  Latency Statistics:")
		fmt.Printf("  Min: %v\n", benchResults.MinLatency)
		fmt.Printf("  Max: %v\n", benchResults.MaxLatency)
		fmt.Printf("  Avg: %v\n", benchResults.AvgLatency)
	}
	
	if len(benchResults.Errors) > 0 {
		fmt.Println("\n❌ Recent Errors:")
		for _, err := range benchResults.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}
	
	// Performance grade
	successRate := float64(benchResults.SuccessfulConns) * 100 / float64(benchResults.TotalConnections)
	grade := getPerformanceGrade(successRate)
	
	fmt.Printf("\n🏆 Performance Grade: %s\n", grade)
}

func getPerformanceGrade(successRate float64) string {
	switch {
	case successRate >= 99:
		return color.GreenString("A+ (Excellent)")
	case successRate >= 95:
		return color.GreenString("A (Very Good)")
	case successRate >= 90:
		return color.YellowString("B (Good)")
	case successRate >= 80:
		return color.YellowString("C (Fair)")
	case successRate >= 70:
		return color.RedString("D (Poor)")
	default:
		return color.RedString("F (Failed)")
	}
}
