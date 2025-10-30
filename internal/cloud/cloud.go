package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/ibrahmsql/gocat/internal/logger"
)

// CloudProvider represents a cloud service provider
type CloudProvider string

const (
	AWS          CloudProvider = "aws"
	Azure        CloudProvider = "azure"
	GCP          CloudProvider = "gcp"
	DigitalOcean CloudProvider = "digitalocean"
	Linode       CloudProvider = "linode"
	Custom       CloudProvider = "custom"
)

// CloudConfig holds cloud configuration
type CloudConfig struct {
	Provider    CloudProvider          `json:"provider"`
	Region      string                 `json:"region"`
	Credentials map[string]string      `json:"credentials"`
	Endpoints   map[string]string      `json:"endpoints"`
	Settings    map[string]interface{} `json:"settings"`
}

// CloudClient interface for cloud operations
type CloudClient interface {
	Connect() error
	Disconnect() error
	IsConnected() bool
	
	// Storage operations
	Upload(key string, data []byte) error
	Download(key string) ([]byte, error)
	Delete(key string) error
	List(prefix string) ([]string, error)
	
	// Compute operations
	StartInstance(config InstanceConfig) (string, error)
	StopInstance(instanceID string) error
	GetInstanceStatus(instanceID string) (InstanceStatus, error)
	ListInstances() ([]Instance, error)
	
	// Network operations
	CreateTunnel(config TunnelConfig) (string, error)
	CloseTunnel(tunnelID string) error
	ListTunnels() ([]Tunnel, error)
	
	// Metrics
	GetMetrics() (CloudMetrics, error)
}

// Instance represents a cloud compute instance
type Instance struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Status    InstanceStatus    `json:"status"`
	IP        string            `json:"ip"`
	Region    string            `json:"region"`
	Type      string            `json:"type"`
	CreatedAt time.Time         `json:"created_at"`
	Tags      map[string]string `json:"tags"`
}

// InstanceStatus represents the status of an instance
type InstanceStatus string

const (
	InstanceRunning InstanceStatus = "running"
	InstanceStopped InstanceStatus = "stopped"
	InstancePending InstanceStatus = "pending"
	InstanceError   InstanceStatus = "error"
)

// InstanceConfig holds configuration for creating an instance
type InstanceConfig struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Image    string            `json:"image"`
	Region   string            `json:"region"`
	SSHKey   string            `json:"ssh_key"`
	UserData string            `json:"user_data"`
	Tags     map[string]string `json:"tags"`
}

// Tunnel represents a network tunnel
type Tunnel struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	LocalPort  int       `json:"local_port"`
	RemoteHost string    `json:"remote_host"`
	RemotePort int       `json:"remote_port"`
	Protocol   string    `json:"protocol"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// TunnelConfig holds configuration for creating a tunnel
type TunnelConfig struct {
	Name       string `json:"name"`
	LocalPort  int    `json:"local_port"`
	RemoteHost string `json:"remote_host"`
	RemotePort int    `json:"remote_port"`
	Protocol   string `json:"protocol"`
}

// CloudMetrics holds cloud usage metrics
type CloudMetrics struct {
	Storage   StorageMetrics   `json:"storage"`
	Compute   ComputeMetrics   `json:"compute"`
	Network   NetworkMetrics   `json:"network"`
	Costs     CostMetrics      `json:"costs"`
	Timestamp time.Time        `json:"timestamp"`
}

// StorageMetrics holds storage usage metrics
type StorageMetrics struct {
	UsedBytes      int64 `json:"used_bytes"`
	TotalBytes     int64 `json:"total_bytes"`
	ObjectCount    int   `json:"object_count"`
	BandwidthBytes int64 `json:"bandwidth_bytes"`
}

// ComputeMetrics holds compute usage metrics
type ComputeMetrics struct {
	InstanceCount int     `json:"instance_count"`
	CPUHours      float64 `json:"cpu_hours"`
	MemoryGB      float64 `json:"memory_gb"`
	DiskGB        float64 `json:"disk_gb"`
}

// NetworkMetrics holds network usage metrics
type NetworkMetrics struct {
	IngressBytes  int64 `json:"ingress_bytes"`
	EgressBytes   int64 `json:"egress_bytes"`
	TunnelCount   int   `json:"tunnel_count"`
	RequestCount  int64 `json:"request_count"`
}

// CostMetrics holds cost information
type CostMetrics struct {
	CurrentMonth float64 `json:"current_month"`
	LastMonth    float64 `json:"last_month"`
	Projected    float64 `json:"projected"`
	Currency     string  `json:"currency"`
}

// CloudManager manages cloud integrations
type CloudManager struct {
	clients   map[string]CloudClient
	configs   map[string]*CloudConfig
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewCloudManager creates a new cloud manager
func NewCloudManager() *CloudManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &CloudManager{
		clients: make(map[string]CloudClient),
		configs: make(map[string]*CloudConfig),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// AddProvider adds a cloud provider
func (cm *CloudManager) AddProvider(name string, config *CloudConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Create client based on provider
	var client CloudClient
	switch config.Provider {
	case AWS:
		client = NewAWSClient(config)
	case Azure:
		client = NewAzureClient(config)
	case GCP:
		client = NewGCPClient(config)
	case Custom:
		client = NewCustomClient(config)
	default:
		return fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	// Connect to provider
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to %s: %v", name, err)
	}

	cm.clients[name] = client
	cm.configs[name] = config

	logger.Info("Added cloud provider: %s (%s)", name, config.Provider)
	return nil
}

// RemoveProvider removes a cloud provider
func (cm *CloudManager) RemoveProvider(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	client, exists := cm.clients[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	// Disconnect
	if err := client.Disconnect(); err != nil {
		logger.Warn("Error disconnecting from %s: %v", name, err)
	}

	delete(cm.clients, name)
	delete(cm.configs, name)

	logger.Info("Removed cloud provider: %s", name)
	return nil
}

// GetClient returns a cloud client by name
func (cm *CloudManager) GetClient(name string) (CloudClient, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	client, exists := cm.clients[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return client, nil
}

// ListProviders lists all configured providers
func (cm *CloudManager) ListProviders() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	providers := make([]string, 0, len(cm.clients))
	for name := range cm.clients {
		providers = append(providers, name)
	}
	return providers
}

// PrintProviderInfo prints information about all providers
func (cm *CloudManager) PrintProviderInfo() {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if len(cm.clients) == 0 {
		color.Yellow("No cloud providers configured")
		return
	}

	fmt.Println()
	color.New(color.FgCyan, color.Bold).Println("╔══════════════════════════════════════════════╗")
	color.New(color.FgCyan, color.Bold).Println("║           ☁️  CLOUD PROVIDERS               ║")
	color.New(color.FgCyan, color.Bold).Println("╚══════════════════════════════════════════════╝")
	fmt.Println()

	for name, client := range cm.clients {
		config := cm.configs[name]
		
		status := "❌ Disconnected"
		statusColor := color.FgRed
		if client.IsConnected() {
			status = "✅ Connected"
			statusColor = color.FgGreen
		}

		color.New(color.FgWhite, color.Bold).Printf("☁️  %s", name)
		color.New(statusColor).Printf(" [%s]\n", status)
		fmt.Printf("   Provider: %s\n", config.Provider)
		fmt.Printf("   Region: %s\n", config.Region)
		
		// Get metrics if connected
		if client.IsConnected() {
			if metrics, err := client.GetMetrics(); err == nil {
				fmt.Printf("   Instances: %d\n", metrics.Compute.InstanceCount)
				fmt.Printf("   Storage: %s / %s\n", 
					formatBytes(metrics.Storage.UsedBytes),
					formatBytes(metrics.Storage.TotalBytes))
				fmt.Printf("   Cost (Month): %s%.2f\n", 
					metrics.Costs.Currency, metrics.Costs.CurrentMonth)
			}
		}
		fmt.Println()
	}
}

// SyncData syncs data across all providers
func (cm *CloudManager) SyncData(key string, data []byte) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var errors []error
	for name, client := range cm.clients {
		if !client.IsConnected() {
			continue
		}

		if err := client.Upload(key, data); err != nil {
			errors = append(errors, fmt.Errorf("%s: %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("sync errors: %v", errors)
	}

	return nil
}

// Shutdown shuts down the cloud manager
func (cm *CloudManager) Shutdown() error {
	cm.cancel()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	for name, client := range cm.clients {
		if err := client.Disconnect(); err != nil {
			logger.Warn("Error disconnecting from %s: %v", name, err)
		}
	}

	cm.clients = make(map[string]CloudClient)
	cm.configs = make(map[string]*CloudConfig)

	return nil
}

// BaseCloudClient provides common functionality for cloud clients
type BaseCloudClient struct {
	config    *CloudConfig
	connected bool
	mu        sync.RWMutex
}

// NewAWSClient creates a new AWS client (placeholder)
func NewAWSClient(config *CloudConfig) CloudClient {
	return &BaseCloudClient{config: config}
}

// NewAzureClient creates a new Azure client (placeholder)
func NewAzureClient(config *CloudConfig) CloudClient {
	return &BaseCloudClient{config: config}
}

// NewGCPClient creates a new GCP client (placeholder)
func NewGCPClient(config *CloudConfig) CloudClient {
	return &BaseCloudClient{config: config}
}

// NewCustomClient creates a new custom cloud client
func NewCustomClient(config *CloudConfig) CloudClient {
	return &CustomCloudClient{
		BaseCloudClient: BaseCloudClient{config: config},
		httpClient:      &http.Client{Timeout: 30 * time.Second},
	}
}

// CustomCloudClient implements a custom cloud client
type CustomCloudClient struct {
	BaseCloudClient
	httpClient *http.Client
}

// Connect connects to the cloud provider
func (c *CustomCloudClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate endpoints
	if c.config.Endpoints == nil || c.config.Endpoints["api"] == "" {
		return fmt.Errorf("API endpoint not configured")
	}

	// Test connection
	resp, err := c.httpClient.Get(c.config.Endpoints["api"] + "/health")
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	c.connected = true
	return nil
}

// Disconnect disconnects from the cloud provider
func (c *CustomCloudClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.connected = false
	return nil
}

// IsConnected checks if connected
func (c *CustomCloudClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Upload uploads data to cloud storage
func (c *CustomCloudClient) Upload(key string, data []byte) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}

	endpoint := c.config.Endpoints["api"] + "/storage/" + key
	resp, err := c.httpClient.Post(endpoint, "application/octet-stream", 
		io.NopCloser(io.Reader(io.MultiReader())))
	if err != nil {
		return fmt.Errorf("upload failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed: status %d", resp.StatusCode)
	}

	return nil
}

// Download downloads data from cloud storage
func (c *CustomCloudClient) Download(key string) ([]byte, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	endpoint := c.config.Endpoints["api"] + "/storage/" + key
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("download failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// Delete deletes data from cloud storage
func (c *CustomCloudClient) Delete(key string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}

	endpoint := c.config.Endpoints["api"] + "/storage/" + key
	req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("delete request failed: %v", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete failed: status %d", resp.StatusCode)
	}

	return nil
}

// List lists objects in cloud storage
func (c *CustomCloudClient) List(prefix string) ([]string, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	endpoint := c.config.Endpoints["api"] + "/storage?prefix=" + prefix
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("list failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list failed: status %d", resp.StatusCode)
	}

	var keys []string
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return keys, nil
}

// Placeholder implementations for BaseCloudClient
func (c *BaseCloudClient) Connect() error { return nil }
func (c *BaseCloudClient) Disconnect() error { return nil }
func (c *BaseCloudClient) IsConnected() bool { return c.connected }
func (c *BaseCloudClient) Upload(key string, data []byte) error { return fmt.Errorf("not implemented") }
func (c *BaseCloudClient) Download(key string) ([]byte, error) { return nil, fmt.Errorf("not implemented") }
func (c *BaseCloudClient) Delete(key string) error { return fmt.Errorf("not implemented") }
func (c *BaseCloudClient) List(prefix string) ([]string, error) { return nil, fmt.Errorf("not implemented") }
func (c *BaseCloudClient) StartInstance(config InstanceConfig) (string, error) { return "", fmt.Errorf("not implemented") }
func (c *BaseCloudClient) StopInstance(instanceID string) error { return fmt.Errorf("not implemented") }
func (c *BaseCloudClient) GetInstanceStatus(instanceID string) (InstanceStatus, error) { return "", fmt.Errorf("not implemented") }
func (c *BaseCloudClient) ListInstances() ([]Instance, error) { return nil, fmt.Errorf("not implemented") }
func (c *BaseCloudClient) CreateTunnel(config TunnelConfig) (string, error) { return "", fmt.Errorf("not implemented") }
func (c *BaseCloudClient) CloseTunnel(tunnelID string) error { return fmt.Errorf("not implemented") }
func (c *BaseCloudClient) ListTunnels() ([]Tunnel, error) { return nil, fmt.Errorf("not implemented") }
func (c *BaseCloudClient) GetMetrics() (CloudMetrics, error) { return CloudMetrics{}, fmt.Errorf("not implemented") }

// Additional implementations for CustomCloudClient
func (c *CustomCloudClient) StartInstance(config InstanceConfig) (string, error) { return "", fmt.Errorf("not implemented") }
func (c *CustomCloudClient) StopInstance(instanceID string) error { return fmt.Errorf("not implemented") }
func (c *CustomCloudClient) GetInstanceStatus(instanceID string) (InstanceStatus, error) { return "", fmt.Errorf("not implemented") }
func (c *CustomCloudClient) ListInstances() ([]Instance, error) { return nil, fmt.Errorf("not implemented") }
func (c *CustomCloudClient) CreateTunnel(config TunnelConfig) (string, error) { return "", fmt.Errorf("not implemented") }
func (c *CustomCloudClient) CloseTunnel(tunnelID string) error { return fmt.Errorf("not implemented") }
func (c *CustomCloudClient) ListTunnels() ([]Tunnel, error) { return nil, fmt.Errorf("not implemented") }
func (c *CustomCloudClient) GetMetrics() (CloudMetrics, error) { return CloudMetrics{}, fmt.Errorf("not implemented") }

// Helper function to format bytes
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
