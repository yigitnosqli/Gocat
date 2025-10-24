package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
)

// PrometheusMetrics implements a Prometheus-compatible metrics exporter
type PrometheusMetrics struct {
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
	mu         sync.RWMutex
	namespace  string
	subsystem  string
}

// Counter represents a Prometheus counter metric
type Counter struct {
	name   string
	help   string
	value  float64
	labels map[string]string
	mu     sync.Mutex
}

// Gauge represents a Prometheus gauge metric
type Gauge struct {
	name   string
	help   string
	value  float64
	labels map[string]string
	mu     sync.RWMutex
}

// Histogram represents a Prometheus histogram metric
type Histogram struct {
	name    string
	help    string
	buckets []float64
	counts  []uint64
	sum     float64
	count   uint64
	labels  map[string]string
	mu      sync.Mutex
}

// NewPrometheusMetrics creates a new Prometheus metrics collector
func NewPrometheusMetrics(namespace, subsystem string) *PrometheusMetrics {
	return &PrometheusMetrics{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		namespace:  namespace,
		subsystem:  subsystem,
	}
}

// IncrementCounter increments a counter metric
func (pm *PrometheusMetrics) IncrementCounter(name string, tags map[string]string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := pm.getMetricKey(name, tags)
	counter, exists := pm.counters[key]
	if !exists {
		counter = &Counter{
			name:   name,
			help:   fmt.Sprintf("Counter for %s", name),
			labels: tags,
		}
		pm.counters[key] = counter
	}

	counter.mu.Lock()
	counter.value++
	counter.mu.Unlock()
}

// RecordGauge records a gauge metric
func (pm *PrometheusMetrics) RecordGauge(name string, value float64, tags map[string]string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := pm.getMetricKey(name, tags)
	gauge, exists := pm.gauges[key]
	if !exists {
		gauge = &Gauge{
			name:   name,
			help:   fmt.Sprintf("Gauge for %s", name),
			labels: tags,
		}
		pm.gauges[key] = gauge
	}

	gauge.mu.Lock()
	gauge.value = value
	gauge.mu.Unlock()
}

// RecordHistogram records a histogram metric
func (pm *PrometheusMetrics) RecordHistogram(name string, value float64, tags map[string]string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := pm.getMetricKey(name, tags)
	histogram, exists := pm.histograms[key]
	if !exists {
		histogram = &Histogram{
			name:    name,
			help:    fmt.Sprintf("Histogram for %s", name),
			buckets: []float64{0.001, 0.01, 0.1, 0.5, 1, 5, 10, 30, 60},
			counts:  make([]uint64, 10), // buckets + 1 for +Inf
			labels:  tags,
		}
		pm.histograms[key] = histogram
	}

	histogram.mu.Lock()
	histogram.sum += value
	histogram.count++

	// Find appropriate bucket
	for i, bucket := range histogram.buckets {
		if value <= bucket {
			histogram.counts[i]++
		}
	}
	histogram.counts[len(histogram.counts)-1]++ // +Inf bucket
	histogram.mu.Unlock()
}

// RecordTimer records a timer metric (as histogram in milliseconds)
func (pm *PrometheusMetrics) RecordTimer(name string, duration time.Duration, tags map[string]string) {
	pm.RecordHistogram(name, duration.Seconds(), tags)
}

// ServeHTTP implements http.Handler for Prometheus /metrics endpoint
func (pm *PrometheusMetrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Export counters
	for _, counter := range pm.counters {
		counter.mu.Lock()
		metricName := pm.formatMetricName(counter.name)
		fmt.Fprintf(w, "# HELP %s %s\n", metricName, counter.help)
		fmt.Fprintf(w, "# TYPE %s counter\n", metricName)
		fmt.Fprintf(w, "%s%s %.0f\n", metricName, pm.formatLabels(counter.labels), counter.value)
		counter.mu.Unlock()
	}

	// Export gauges
	for _, gauge := range pm.gauges {
		gauge.mu.RLock()
		metricName := pm.formatMetricName(gauge.name)
		fmt.Fprintf(w, "# HELP %s %s\n", metricName, gauge.help)
		fmt.Fprintf(w, "# TYPE %s gauge\n", metricName)
		fmt.Fprintf(w, "%s%s %.2f\n", metricName, pm.formatLabels(gauge.labels), gauge.value)
		gauge.mu.RUnlock()
	}

	// Export histograms
	for _, histogram := range pm.histograms {
		histogram.mu.Lock()
		metricName := pm.formatMetricName(histogram.name)
		fmt.Fprintf(w, "# HELP %s %s\n", metricName, histogram.help)
		fmt.Fprintf(w, "# TYPE %s histogram\n", metricName)

		// Bucket counts
		var cumulativeCount uint64
		for i, bucket := range histogram.buckets {
			cumulativeCount += histogram.counts[i]
			labels := pm.copyLabels(histogram.labels)
			labels["le"] = fmt.Sprintf("%.3f", bucket)
			fmt.Fprintf(w, "%s_bucket%s %d\n", metricName, pm.formatLabels(labels), cumulativeCount)
		}

		// +Inf bucket
		labels := pm.copyLabels(histogram.labels)
		labels["le"] = "+Inf"
		fmt.Fprintf(w, "%s_bucket%s %d\n", metricName, pm.formatLabels(labels), histogram.count)

		// Sum and count
		fmt.Fprintf(w, "%s_sum%s %.6f\n", metricName, pm.formatLabels(histogram.labels), histogram.sum)
		fmt.Fprintf(w, "%s_count%s %d\n", metricName, pm.formatLabels(histogram.labels), histogram.count)
		histogram.mu.Unlock()
	}
}

// Helper functions

func (pm *PrometheusMetrics) getMetricKey(name string, tags map[string]string) string {
	key := name
	if len(tags) > 0 {
		key += "{"
		first := true
		for k, v := range tags {
			if !first {
				key += ","
			}
			key += fmt.Sprintf("%s=%s", k, v)
			first = false
		}
		key += "}"
	}
	return key
}

func (pm *PrometheusMetrics) formatMetricName(name string) string {
	if pm.namespace != "" && pm.subsystem != "" {
		return fmt.Sprintf("%s_%s_%s", pm.namespace, pm.subsystem, name)
	} else if pm.namespace != "" {
		return fmt.Sprintf("%s_%s", pm.namespace, name)
	}
	return name
}

func (pm *PrometheusMetrics) formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	result := "{"
	first := true
	for k, v := range labels {
		if !first {
			result += ","
		}
		result += fmt.Sprintf("%s=\"%s\"", k, v)
		first = false
	}
	result += "}"
	return result
}

func (pm *PrometheusMetrics) copyLabels(labels map[string]string) map[string]string {
	copy := make(map[string]string, len(labels))
	for k, v := range labels {
		copy[k] = v
	}
	return copy
}

// StartMetricsServer starts an HTTP server to expose Prometheus metrics
func StartMetricsServer(port string, metrics *PrometheusMetrics) error {
	http.Handle("/metrics", metrics)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK\n"))
	})

	logger.Info("Prometheus metrics server listening on :%s/metrics", port)
	return http.ListenAndServe(":"+port, nil)
}

// DefaultMetrics creates and returns default GoCat metrics
func DefaultMetrics() *PrometheusMetrics {
	pm := NewPrometheusMetrics("gocat", "network")
	
	// Initialize common metrics
	pm.RecordGauge("build_info", 1, map[string]string{
		"version": "dev",
	})
	pm.RecordGauge("start_time_seconds", float64(time.Now().Unix()), nil)
	
	return pm
}
