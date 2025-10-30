package nekodns

import (
	"sync"
)

// DNSResponse holds DNS response chunks
type DNSResponse struct {
	Chunks []string
	mu     sync.Mutex
}

// ActiveCommand represents the current command state
type ActiveCommand struct {
	Cmd                string
	Delivered          bool
	Chunks             []string
	UploadInProgress   bool
	FileChunksToSend   []string
	mu                 sync.Mutex
}

// RemoteInfo holds remote system information
type RemoteInfo struct {
	Whoami   string
	Hostname string
	Pwd      string
	mu       sync.RWMutex
}

// Lock locks the DNSResponse mutex
func (r *DNSResponse) Lock() {
	r.mu.Lock()
}

// Unlock unlocks the DNSResponse mutex
func (r *DNSResponse) Unlock() {
	r.mu.Unlock()
}

// Lock locks the ActiveCommand mutex
func (a *ActiveCommand) Lock() {
	a.mu.Lock()
}

// Unlock unlocks the ActiveCommand mutex
func (a *ActiveCommand) Unlock() {
	a.mu.Unlock()
}

// RLock read locks the RemoteInfo
func (r *RemoteInfo) RLock() {
	r.mu.RLock()
}

// RUnlock read unlocks the RemoteInfo
func (r *RemoteInfo) RUnlock() {
	r.mu.RUnlock()
}

// Lock write locks the RemoteInfo
func (r *RemoteInfo) Lock() {
	r.mu.Lock()
}

// Unlock write unlocks the RemoteInfo
func (r *RemoteInfo) Unlock() {
	r.mu.Unlock()
}
