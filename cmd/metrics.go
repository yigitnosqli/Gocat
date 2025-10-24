package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/ibrahmsql/gocat/internal/metrics"
	"github.com/spf13/cobra"
)

var (
	metricsPort      string
	metricsNamespace string
	metricsSubsystem string
	metricsInterval  time.Duration
)

// metricsCmd represents the metrics command
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Start Prometheus metrics exporter",
	Long: `Start an HTTP server that exposes Prometheus-compatible metrics.

The metrics endpoint will be available at http://localhost:<port>/metrics
and can be scraped by Prometheus for monitoring and alerting.

Available metrics include:
  - Connection statistics (total, active, failed)
  - Bytes transferred (sent, received)
  - Error counts
  - Request durations
  - System metrics (CPU, memory, goroutines)

Examples:
  # Start metrics server on default port 9090
  gocat metrics

  # Start on custom port
  gocat metrics --port 8080

  # With custom namespace
  gocat metrics --namespace myapp --subsystem network`,
	RunE: runMetrics,
}

func init() {
	rootCmd.AddCommand(metricsCmd)

	metricsCmd.Flags().StringVar(&metricsPort, "port", "9090", "Port to expose metrics on")
	metricsCmd.Flags().StringVar(&metricsNamespace, "namespace", "gocat", "Metrics namespace")
	metricsCmd.Flags().StringVar(&metricsSubsystem, "subsystem", "network", "Metrics subsystem")
	metricsCmd.Flags().DurationVar(&metricsInterval, "interval", 15*time.Second, "System metrics collection interval")
}

func runMetrics(cmd *cobra.Command, args []string) error {
	logger.Info("Starting Prometheus metrics exporter")
	logger.Info("Metrics endpoint: http://localhost:%s/metrics", metricsPort)
	logger.Info("Health endpoint: http://localhost:%s/health", metricsPort)

	// Create metrics collector
	pm := metrics.NewPrometheusMetrics(metricsNamespace, metricsSubsystem)

	// Add build info
	pm.RecordGauge("build_info", 1, map[string]string{
		"version":    version,
		"git_commit": gitCommit,
		"git_branch": gitBranch,
		"go_version": runtime.Version(),
	})

	// Add start time
	pm.RecordGauge("start_time_seconds", float64(time.Now().Unix()), nil)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start system metrics collector
	go collectSystemMetrics(ctx, pm, metricsInterval)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start metrics server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := metrics.StartMetricsServer(metricsPort, pm); err != nil {
			errChan <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		logger.Info("Received shutdown signal, stopping metrics server...")
		cancel()
		return nil
	case err := <-errChan:
		cancel()
		return err
	}
}

func collectSystemMetrics(ctx context.Context, pm *metrics.PrometheusMetrics, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var memStats runtime.MemStats

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Collect memory statistics
			runtime.ReadMemStats(&memStats)

			// Memory metrics
			pm.RecordGauge("memory_alloc_bytes", float64(memStats.Alloc), nil)
			pm.RecordGauge("memory_total_alloc_bytes", float64(memStats.TotalAlloc), nil)
			pm.RecordGauge("memory_sys_bytes", float64(memStats.Sys), nil)
			pm.RecordGauge("memory_heap_alloc_bytes", float64(memStats.HeapAlloc), nil)
			pm.RecordGauge("memory_heap_sys_bytes", float64(memStats.HeapSys), nil)
			pm.RecordGauge("memory_heap_idle_bytes", float64(memStats.HeapIdle), nil)
			pm.RecordGauge("memory_heap_inuse_bytes", float64(memStats.HeapInuse), nil)
			pm.RecordGauge("memory_stack_inuse_bytes", float64(memStats.StackInuse), nil)
			pm.RecordGauge("memory_stack_sys_bytes", float64(memStats.StackSys), nil)

			// GC metrics
			pm.RecordGauge("gc_runs_total", float64(memStats.NumGC), nil)
			pm.RecordGauge("gc_pause_ns", float64(memStats.PauseNs[(memStats.NumGC+255)%256]), nil)
			pm.RecordGauge("gc_cpu_fraction", memStats.GCCPUFraction, nil)

			// Goroutine metrics
			pm.RecordGauge("goroutines", float64(runtime.NumGoroutine()), nil)

			// CPU metrics
			pm.RecordGauge("cpu_cores", float64(runtime.NumCPU()), nil)

			logger.Debug("System metrics collected - Goroutines: %d, Memory: %.2f MB",
				runtime.NumGoroutine(),
				float64(memStats.Alloc)/(1024*1024))
		}
	}
}
