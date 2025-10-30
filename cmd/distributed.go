package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/gorilla/websocket"
	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var (
	distMode       string // master, worker, standalone
	distMasterAddr string
	distWorkerID   string
	distPort       int
	distToken      string
	distMaxWorkers int
)

// NodeType represents the type of distributed node
type NodeType string

const (
	MasterNode     NodeType = "master"
	WorkerNode     NodeType = "worker"
	StandaloneNode NodeType = "standalone"
)

// Node represents a distributed node
type Node struct {
	ID         string    `json:"id"`
	Type       NodeType  `json:"type"`
	Address    string    `json:"address"`
	Status     string    `json:"status"`
	Capacity   int       `json:"capacity"`
	Load       int       `json:"load"`
	LastSeen   time.Time `json:"last_seen"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Task represents a distributed task
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    int                    `json:"priority"`
	Payload     map[string]interface{} `json:"payload"`
	AssignedTo  string                 `json:"assigned_to"`
	Status      string                 `json:"status"`
	Result      interface{}            `json:"result"`
	Error       string                 `json:"error"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at"`
}

// DistributedManager manages distributed operations
type DistributedManager struct {
	nodeType   NodeType
	nodeID     string
	nodes      map[string]*Node
	tasks      map[string]*Task
	taskQueue  chan *Task
	resultChan chan *Task
	conn       *websocket.Conn
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

var distributedCmd = &cobra.Command{
	Use:   "distributed",
	Aliases: []string{"dist", "cluster"},
	Short: "Distributed mode for cluster operations",
	Long: `Run GoCat in distributed mode for cluster operations.

Supports master-worker architecture for distributed scanning, processing, and coordination.`,
	Example: `  # Start as master node
  gocat distributed --mode master --port 9090
  
  # Start as worker node
  gocat distributed --mode worker --master 192.168.1.100:9090 --id worker1
  
  # Start standalone with clustering capability
  gocat distributed --mode standalone --port 9090`,
	Run: runDistributed,
}

func init() {
	rootCmd.AddCommand(distributedCmd)

	distributedCmd.Flags().StringVar(&distMode, "mode", "standalone", "Node mode (master/worker/standalone)")
	distributedCmd.Flags().StringVar(&distMasterAddr, "master", "", "Master node address (for worker mode)")
	distributedCmd.Flags().StringVar(&distWorkerID, "id", "", "Worker node ID")
	distributedCmd.Flags().IntVar(&distPort, "port", 9090, "Port for distributed communication")
	distributedCmd.Flags().StringVar(&distToken, "token", "", "Authentication token")
	distributedCmd.Flags().IntVar(&distMaxWorkers, "max-workers", 100, "Maximum number of workers (master mode)")
}

func runDistributed(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dm := &DistributedManager{
		nodeType:   NodeType(distMode),
		nodes:      make(map[string]*Node),
		tasks:      make(map[string]*Task),
		taskQueue:  make(chan *Task, 1000),
		resultChan: make(chan *Task, 1000),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Generate node ID if not provided
	if distWorkerID == "" {
		hostname, _ := os.Hostname()
		dm.nodeID = fmt.Sprintf("%s-%d", hostname, time.Now().Unix())
	} else {
		dm.nodeID = distWorkerID
	}

	switch dm.nodeType {
	case MasterNode:
		if err := dm.startMaster(); err != nil {
			logger.Fatal("Failed to start master node: %v", err)
		}
	case WorkerNode:
		if distMasterAddr == "" {
			logger.Fatal("Master address required for worker mode")
		}
		if err := dm.startWorker(); err != nil {
			logger.Fatal("Failed to start worker node: %v", err)
		}
	case StandaloneNode:
		if err := dm.startStandalone(); err != nil {
			logger.Fatal("Failed to start standalone node: %v", err)
		}
	default:
		logger.Fatal("Invalid mode: %s", distMode)
	}
}

// startMaster starts the node as a master
func (dm *DistributedManager) startMaster() error {
	logger.Info("Starting as MASTER node on port %d", distPort)
	
	// Start WebSocket server for worker connections
	http.HandleFunc("/ws", dm.handleWorkerConnection)
	http.HandleFunc("/api/status", dm.handleAPIStatus)
	http.HandleFunc("/api/tasks", dm.handleAPITasks)
	http.HandleFunc("/api/nodes", dm.handleAPINodes)
	
	// Start task scheduler
	go dm.taskScheduler()
	
	// Start result processor
	go dm.resultProcessor()
	
	// Start monitoring
	go dm.monitorNodes()
	
	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", distPort),
		Handler: nil,
	}
	
	// Print master info
	dm.printMasterInfo()
	
	return server.ListenAndServe()
}

// startWorker starts the node as a worker
func (dm *DistributedManager) startWorker() error {
	logger.Info("Starting as WORKER node, connecting to master at %s", distMasterAddr)
	
	// Connect to master via WebSocket
	url := fmt.Sprintf("ws://%s/ws", distMasterAddr)
	
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to master: %v", err)
	}
	dm.conn = conn
	
	// Register with master
	if err := dm.registerWithMaster(); err != nil {
		return fmt.Errorf("failed to register with master: %v", err)
	}
	
	// Start task processor
	go dm.processWorkerTasks()
	
	// Start heartbeat
	go dm.sendHeartbeat()
	
	// Print worker info
	dm.printWorkerInfo()
	
	// Wait for tasks
	<-dm.ctx.Done()
	return nil
}

// startStandalone starts the node in standalone mode
func (dm *DistributedManager) startStandalone() error {
	logger.Info("Starting in STANDALONE mode with clustering capability on port %d", distPort)
	
	// Can act as both master and worker
	go dm.startMaster()
	
	// Also process tasks locally
	go dm.processLocalTasks()
	
	// Print standalone info
	dm.printStandaloneInfo()
	
	<-dm.ctx.Done()
	return nil
}

// handleWorkerConnection handles WebSocket connections from workers
func (dm *DistributedManager) handleWorkerConnection(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()
	
	// Handle worker messages
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			logger.Error("Failed to read message: %v", err)
			break
		}
		
		dm.handleWorkerMessage(conn, msg)
	}
}

// taskScheduler schedules tasks to workers
func (dm *DistributedManager) taskScheduler() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-dm.ctx.Done():
			return
		case task := <-dm.taskQueue:
			// Find best worker for task
			worker := dm.findBestWorker()
			if worker != nil {
				dm.assignTask(task, worker)
			} else {
				// Re-queue task
				go func() {
					time.Sleep(5 * time.Second)
					dm.taskQueue <- task
				}()
			}
		case <-ticker.C:
			// Rebalance tasks if needed
			dm.rebalanceTasks()
		}
	}
}

// findBestWorker finds the best worker for a task
func (dm *DistributedManager) findBestWorker() *Node {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	var bestWorker *Node
	minLoad := int(^uint(0) >> 1) // Max int
	
	for _, node := range dm.nodes {
		if node.Type != WorkerNode {
			continue
		}
		if node.Status != "active" {
			continue
		}
		if node.Load < minLoad {
			bestWorker = node
			minLoad = node.Load
		}
	}
	
	return bestWorker
}

// printMasterInfo prints master node information
func (dm *DistributedManager) printMasterInfo() {
	fmt.Println()
	color.New(color.FgCyan, color.Bold).Println("╔══════════════════════════════════════════════╗")
	color.New(color.FgCyan, color.Bold).Println("║         🎛️  MASTER NODE ACTIVE              ║")
	color.New(color.FgCyan, color.Bold).Println("╚══════════════════════════════════════════════╝")
	fmt.Println()
	
	color.New(color.FgWhite, color.Bold).Print("📍 Node ID: ")
	color.Yellow("%s\n", dm.nodeID)
	
	color.New(color.FgWhite, color.Bold).Print("🌐 API Endpoint: ")
	color.Green("http://0.0.0.0:%d\n", distPort)
	
	color.New(color.FgWhite, color.Bold).Print("🔗 WebSocket: ")
	color.Green("ws://0.0.0.0:%d/ws\n", distPort)
	
	fmt.Println()
	color.New(color.FgWhite).Println("📊 Endpoints:")
	fmt.Println("  • /api/status - Cluster status")
	fmt.Println("  • /api/tasks  - Task management")
	fmt.Println("  • /api/nodes  - Node information")
	fmt.Println()
	
	color.Yellow("⏳ Waiting for worker connections...")
}

// printWorkerInfo prints worker node information
func (dm *DistributedManager) printWorkerInfo() {
	fmt.Println()
	color.New(color.FgGreen, color.Bold).Println("╔══════════════════════════════════════════════╗")
	color.New(color.FgGreen, color.Bold).Println("║         ⚙️  WORKER NODE ACTIVE               ║")
	color.New(color.FgGreen, color.Bold).Println("╚══════════════════════════════════════════════╝")
	fmt.Println()
	
	color.New(color.FgWhite, color.Bold).Print("📍 Worker ID: ")
	color.Yellow("%s\n", dm.nodeID)
	
	color.New(color.FgWhite, color.Bold).Print("🎛️  Master: ")
	color.Green("%s\n", distMasterAddr)
	
	color.New(color.FgWhite, color.Bold).Print("📊 Status: ")
	color.Green("Connected\n")
	
	fmt.Println()
	color.Yellow("⏳ Waiting for tasks...")
}

// printStandaloneInfo prints standalone node information
func (dm *DistributedManager) printStandaloneInfo() {
	fmt.Println()
	color.New(color.FgMagenta, color.Bold).Println("╔══════════════════════════════════════════════╗")
	color.New(color.FgMagenta, color.Bold).Println("║       🔄 STANDALONE NODE WITH CLUSTERING    ║")
	color.New(color.FgMagenta, color.Bold).Println("╚══════════════════════════════════════════════╝")
	fmt.Println()
	
	color.New(color.FgWhite, color.Bold).Print("📍 Node ID: ")
	color.Yellow("%s\n", dm.nodeID)
	
	color.New(color.FgWhite, color.Bold).Print("🌐 API Endpoint: ")
	color.Green("http://0.0.0.0:%d\n", distPort)
	
	color.New(color.FgWhite, color.Bold).Print("📊 Mode: ")
	color.Magenta("Master + Worker\n")
	
	fmt.Println()
	color.New(color.FgWhite).Println("✨ Capabilities:")
	fmt.Println("  • Can accept worker connections")
	fmt.Println("  • Can process tasks locally")
	fmt.Println("  • Can distribute tasks to workers")
	fmt.Println()
}

// monitorNodes monitors connected nodes
func (dm *DistributedManager) monitorNodes() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-ticker.C:
			dm.mu.Lock()
			now := time.Now()
			for id, node := range dm.nodes {
				if now.Sub(node.LastSeen) > 60*time.Second {
					node.Status = "inactive"
					logger.Warn("Node %s marked as inactive", id)
				}
			}
			dm.mu.Unlock()
			
			// Print cluster status
			dm.printClusterStatus()
		}
	}
}

// printClusterStatus prints the current cluster status
func (dm *DistributedManager) printClusterStatus() {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	activeNodes := 0
	totalCapacity := 0
	totalLoad := 0
	
	for _, node := range dm.nodes {
		if node.Status == "active" {
			activeNodes++
			totalCapacity += node.Capacity
			totalLoad += node.Load
		}
	}
	
	pendingTasks := 0
	completedTasks := 0
	failedTasks := 0
	
	for _, task := range dm.tasks {
		switch task.Status {
		case "pending":
			pendingTasks++
		case "completed":
			completedTasks++
		case "failed":
			failedTasks++
		}
	}
	
	fmt.Println()
	color.New(color.FgCyan).Println("📊 Cluster Status Update:")
	fmt.Printf("  Nodes: %d active | Capacity: %d | Load: %d (%.1f%%)\n",
		activeNodes, totalCapacity, totalLoad,
		float64(totalLoad)*100/float64(totalCapacity))
	fmt.Printf("  Tasks: %d pending | %d completed | %d failed\n",
		pendingTasks, completedTasks, failedTasks)
}

// Placeholder implementations for missing functions
func (dm *DistributedManager) handleWorkerMessage(conn *websocket.Conn, msg map[string]interface{}) {}
func (dm *DistributedManager) registerWithMaster() error { return nil }
func (dm *DistributedManager) processWorkerTasks() {}
func (dm *DistributedManager) processLocalTasks() {}
func (dm *DistributedManager) sendHeartbeat() {}
func (dm *DistributedManager) assignTask(task *Task, worker *Node) {}
func (dm *DistributedManager) rebalanceTasks() {}
func (dm *DistributedManager) resultProcessor() {}
func (dm *DistributedManager) handleAPIStatus(w http.ResponseWriter, r *http.Request) {}
func (dm *DistributedManager) handleAPITasks(w http.ResponseWriter, r *http.Request) {}
func (dm *DistributedManager) handleAPINodes(w http.ResponseWriter, r *http.Request) {}
