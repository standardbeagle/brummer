package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// HealthStatus represents the health status of an instance
type HealthStatus struct {
	InstanceID          string
	LastPing            time.Time
	LastSuccessfulPing  time.Time
	ConsecutiveFailures int
	ResponseTime        time.Duration
	IsHealthy           bool
	LastError           error
}

// HealthMonitor monitors the health of connected instances using MCP ping
type HealthMonitor struct {
	connMgr        *ConnectionManager
	healthStatuses map[string]*HealthStatus
	mu             sync.RWMutex

	// Configuration
	pingInterval time.Duration
	pingTimeout  time.Duration
	maxFailures  int

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Callbacks
	onUnhealthy func(instanceID string, status *HealthStatus)
	onRecovered func(instanceID string, status *HealthStatus)
	onDead      func(instanceID string, status *HealthStatus)
}

// HealthMonitorConfig configures the health monitor
type HealthMonitorConfig struct {
	PingInterval time.Duration
	PingTimeout  time.Duration
	MaxFailures  int
}

// DefaultHealthMonitorConfig provides sensible defaults
var DefaultHealthMonitorConfig = HealthMonitorConfig{
	PingInterval: 10 * time.Second,
	PingTimeout:  5 * time.Second,
	MaxFailures:  3,
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(connMgr *ConnectionManager, config *HealthMonitorConfig) *HealthMonitor {
	if config == nil {
		config = &DefaultHealthMonitorConfig
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &HealthMonitor{
		connMgr:        connMgr,
		healthStatuses: make(map[string]*HealthStatus),
		pingInterval:   config.PingInterval,
		pingTimeout:    config.PingTimeout,
		maxFailures:    config.MaxFailures,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start begins health monitoring
func (hm *HealthMonitor) Start() {
	hm.wg.Add(1)
	go hm.monitorLoop()
}

// Stop gracefully stops health monitoring
func (hm *HealthMonitor) Stop() {
	hm.cancel()
	hm.wg.Wait()
}

// monitorLoop is the main monitoring loop
func (hm *HealthMonitor) monitorLoop() {
	defer hm.wg.Done()

	ticker := time.NewTicker(hm.pingInterval)
	defer ticker.Stop()

	// Initial check
	hm.checkAllInstances()

	for {
		select {
		case <-hm.ctx.Done():
			return

		case <-ticker.C:
			hm.checkAllInstances()
		}
	}
}

// checkAllInstances checks the health of all active instances
func (hm *HealthMonitor) checkAllInstances() {
	instances := hm.connMgr.ListInstances()

	for _, info := range instances {
		// Only check active instances
		if info.State != StateActive {
			continue
		}

		hm.wg.Add(1)
		go func(instanceID string) {
			defer hm.wg.Done()
			hm.checkInstance(instanceID)
		}(info.InstanceID)
	}
}

// checkInstance performs a health check on a single instance
func (hm *HealthMonitor) checkInstance(instanceID string) {
	// Get or create health status
	hm.mu.Lock()
	status, exists := hm.healthStatuses[instanceID]
	if !exists {
		status = &HealthStatus{
			InstanceID: instanceID,
			IsHealthy:  true,
		}
		hm.healthStatuses[instanceID] = status
	}
	hm.mu.Unlock()

	// Get the client
	connections := hm.connMgr.ListInstances()
	var client HubClientInterface
	for _, conn := range connections {
		if conn.InstanceID == instanceID && conn.Client != nil {
			client = conn.Client
			break
		}
	}

	if client == nil {
		hm.recordFailure(status, fmt.Errorf("no client available"))
		return
	}

	// Perform ping with timeout
	ctx, cancel := context.WithTimeout(hm.ctx, hm.pingTimeout)
	defer cancel()

	startTime := time.Now()
	err := client.Ping(ctx)
	responseTime := time.Since(startTime)

	if err != nil {
		hm.recordFailure(status, err)
	} else {
		hm.recordSuccess(status, responseTime)
	}
}

// recordSuccess records a successful ping
func (hm *HealthMonitor) recordSuccess(status *HealthStatus, responseTime time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	wasUnhealthy := !status.IsHealthy

	status.LastPing = time.Now()
	status.LastSuccessfulPing = time.Now()
	status.ConsecutiveFailures = 0
	status.ResponseTime = responseTime
	status.IsHealthy = true
	status.LastError = nil

	// Update connection activity time
	if hm.connMgr.UpdateActivity(status.InstanceID) {
		log.Printf("Updated activity for instance %s (response time: %v)", status.InstanceID, responseTime)
	}

	// Trigger recovery callback if instance recovered
	if wasUnhealthy && hm.onRecovered != nil {
		hm.onRecovered(status.InstanceID, status)
	}
}

// recordFailure records a failed ping
func (hm *HealthMonitor) recordFailure(status *HealthStatus, err error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	wasHealthy := status.IsHealthy

	status.LastPing = time.Now()
	status.ConsecutiveFailures++
	status.LastError = err

	// Check if instance should be marked unhealthy
	if status.ConsecutiveFailures >= hm.maxFailures {
		status.IsHealthy = false

		// Update connection manager state
		if status.ConsecutiveFailures == hm.maxFailures {
			// First time becoming unhealthy
			hm.connMgr.updateStateWithReason(status.InstanceID, StateRetrying,
				fmt.Sprintf("Health check failed %d times: %v",
					status.ConsecutiveFailures, err))

			// Trigger unhealthy callback
			if wasHealthy && hm.onUnhealthy != nil {
				hm.onUnhealthy(status.InstanceID, status)
			}
		} else if status.ConsecutiveFailures > hm.maxFailures*2 {
			// Mark as dead after too many failures
			hm.connMgr.updateStateWithReason(status.InstanceID, StateDead,
				fmt.Sprintf("Failed %d consecutive health checks: %v",
					status.ConsecutiveFailures, status.LastError))

			// Trigger dead callback
			if hm.onDead != nil {
				hm.onDead(status.InstanceID, status)
			}
		}
	}

	log.Printf("Health check failed for %s: %v (failures: %d)",
		status.InstanceID, err, status.ConsecutiveFailures)
}

// GetHealthStatus returns the health status for an instance
func (hm *HealthMonitor) GetHealthStatus(instanceID string) (*HealthStatus, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	status, exists := hm.healthStatuses[instanceID]
	if !exists {
		return nil, fmt.Errorf("no health status for instance %s", instanceID)
	}

	// Return a copy to avoid races
	statusCopy := *status
	return &statusCopy, nil
}

// GetAllHealthStatuses returns all health statuses
func (hm *HealthMonitor) GetAllHealthStatuses() map[string]*HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	statuses := make(map[string]*HealthStatus)
	for id, status := range hm.healthStatuses {
		statusCopy := *status
		statuses[id] = &statusCopy
	}

	return statuses
}

// SetCallbacks sets the callback functions
func (hm *HealthMonitor) SetCallbacks(
	onUnhealthy func(instanceID string, status *HealthStatus),
	onRecovered func(instanceID string, status *HealthStatus),
	onDead func(instanceID string, status *HealthStatus),
) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.onUnhealthy = onUnhealthy
	hm.onRecovered = onRecovered
	hm.onDead = onDead
}

// ForceCheck forces an immediate health check of an instance
func (hm *HealthMonitor) ForceCheck(instanceID string) error {
	// Check if instance exists
	instances := hm.connMgr.ListInstances()
	found := false
	for _, info := range instances {
		if info.InstanceID == instanceID {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	// Perform check asynchronously
	go hm.checkInstance(instanceID)

	return nil
}

// GetMetrics returns health monitoring metrics
func (hm *HealthMonitor) GetMetrics() map[string]interface{} {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	healthy := 0
	unhealthy := 0
	totalResponseTime := time.Duration(0)
	count := 0

	for _, status := range hm.healthStatuses {
		if status.IsHealthy {
			healthy++
			if status.ResponseTime > 0 {
				totalResponseTime += status.ResponseTime
				count++
			}
		} else {
			unhealthy++
		}
	}

	avgResponseTime := time.Duration(0)
	if count > 0 {
		avgResponseTime = totalResponseTime / time.Duration(count)
	}

	return map[string]interface{}{
		"healthy_instances":   healthy,
		"unhealthy_instances": unhealthy,
		"total_instances":     len(hm.healthStatuses),
		"avg_response_time":   avgResponseTime,
		"ping_interval":       hm.pingInterval,
		"ping_timeout":        hm.pingTimeout,
		"max_failures":        hm.maxFailures,
	}
}
