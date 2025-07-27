package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/standardbeagle/brummer/internal/discovery"
)

// Connection states
type ConnectionState int

const (
	StateDiscovered ConnectionState = iota // File found, not connected
	StateConnecting                        // Attempting connection
	StateActive                            // Connected and responsive
	StateRetrying                          // Connection lost, retrying
	StateDead                              // Given up
)

func (s ConnectionState) String() string {
	switch s {
	case StateDiscovered:
		return "discovered"
	case StateConnecting:
		return "connecting"
	case StateActive:
		return "active"
	case StateRetrying:
		return "retrying"
	case StateDead:
		return "dead"
	default:
		return "unknown"
	}
}

// StateTransition records a state change
type StateTransition struct {
	From      ConnectionState
	To        ConnectionState
	Timestamp time.Time
	Reason    string
}

// ConnectionInfo tracks instance connection
type ConnectionInfo struct {
	// Instance metadata
	InstanceID string
	Name       string
	Directory  string
	Port       int
	ProcessPID int

	// Connection state
	State        ConnectionState
	Client       HubClientInterface // HTTP client to instance (regular or persistent)
	LastActivity time.Time
	ConnectedAt  time.Time
	RetryCount   int

	// Session mapping
	Sessions map[string]bool // Active sessions for this instance

	// State timing tracking
	DiscoveredAt   time.Time
	StateChangedAt time.Time
	StateHistory   []StateTransition

	// Retry policy for robust connections
	RetryPolicy *RetryPolicy
	NextRetryAt time.Time
}

// Request types for channel operations
type registerRequest struct {
	instance *discovery.Instance
	response chan error
}

type connectRequest struct {
	instanceID string
	sessionID  string
	response   chan error
}

type disconnectRequest struct {
	sessionID string
	response  chan error
}

type ensureRequest struct {
	instanceID string
	response   chan bool
}

type stateChangeRequest struct {
	instanceID string
	newState   ConnectionState
	reason     string
	response   chan error
}

type listRequest struct {
	response chan []*ConnectionInfo
}

type getClientRequest struct {
	sessionID string
	response  chan HubClientInterface
}

type setClientRequest struct {
	instanceID string
	client     HubClientInterface
	response   chan error
}

// ConnectionManager manages all instance connections
type ConnectionManager struct {
	connections map[string]*ConnectionInfo
	sessions    map[string]string // sessionID -> instanceID

	// Channel operations
	registerChan   chan registerRequest
	connectChan    chan connectRequest
	disconnectChan chan disconnectRequest
	ensureChan     chan ensureRequest
	stateChan      chan stateChangeRequest
	listChan       chan listRequest
	getClientChan  chan getClientRequest
	setClientChan  chan setClientRequest

	// Control
	stopCh chan struct{}
	doneCh chan struct{}

	// Context management
	sessionManager *SessionManager

	// Network monitoring
	networkMonitor *NetworkMonitor
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	cm := &ConnectionManager{
		connections:    make(map[string]*ConnectionInfo),
		sessions:       make(map[string]string),
		registerChan:   make(chan registerRequest),
		connectChan:    make(chan connectRequest),
		disconnectChan: make(chan disconnectRequest),
		ensureChan:     make(chan ensureRequest),
		stateChan:      make(chan stateChangeRequest),
		listChan:       make(chan listRequest),
		getClientChan:  make(chan getClientRequest),
		setClientChan:  make(chan setClientRequest),
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
		sessionManager: NewSessionManager(),
		networkMonitor: NewNetworkMonitor(),
	}

	// Start session manager
	cm.sessionManager.Start()

	// Start network monitor
	if err := cm.networkMonitor.Start(); err != nil {
		debugLog("Failed to start network monitor: %v", err)
	}

	go cm.run()
	go cm.handleNetworkEvents()

	return cm
}

// run is the main event loop - owns all state
func (cm *ConnectionManager) run() {
	defer close(cm.doneCh)

	// Start connection monitor
	go cm.monitorConnections()

	for {
		select {
		case req := <-cm.registerChan:
			cm.handleRegister(req)

		case req := <-cm.connectChan:
			cm.handleConnect(req)

		case req := <-cm.disconnectChan:
			cm.handleDisconnect(req)

		case req := <-cm.ensureChan:
			cm.handleEnsure(req)

		case req := <-cm.stateChan:
			cm.handleStateChange(req)

		case req := <-cm.listChan:
			cm.handleList(req)

		case req := <-cm.getClientChan:
			cm.handleGetClient(req)

		case req := <-cm.setClientChan:
			cm.handleSetClient(req)

		case <-cm.stopCh:
			cm.cleanup()
			return
		}
	}
}

// Handle operations (run in main goroutine)

func (cm *ConnectionManager) handleRegister(req registerRequest) {
	if req.instance == nil {
		req.response <- fmt.Errorf("nil instance")
		return
	}

	// Check if already registered
	if _, exists := cm.connections[req.instance.ID]; exists {
		req.response <- nil // Already registered
		return
	}

	// Create connection info
	now := time.Now()
	info := &ConnectionInfo{
		InstanceID:     req.instance.ID,
		Name:           req.instance.Name,
		Directory:      req.instance.Directory,
		Port:           req.instance.Port,
		ProcessPID:     req.instance.ProcessInfo.PID,
		State:          StateDiscovered,
		LastActivity:   now,
		Sessions:       make(map[string]bool),
		DiscoveredAt:   now,
		StateChangedAt: now,
		RetryPolicy:    NewRetryPolicy(10), // Max 10 retry attempts
		StateHistory:   []StateTransition{},
	}

	cm.connections[req.instance.ID] = info

	// Start connection attempt
	go cm.attemptConnection(req.instance.ID)

	req.response <- nil
}

func (cm *ConnectionManager) handleConnect(req connectRequest) {
	instanceID, exists := cm.sessions[req.sessionID]
	if exists && instanceID != req.instanceID {
		req.response <- fmt.Errorf("session already connected to different instance")
		return
	}

	info, exists := cm.connections[req.instanceID]
	if !exists {
		req.response <- fmt.Errorf("instance not found: %s", req.instanceID)
		return
	}

	if info.State != StateActive {
		req.response <- fmt.Errorf("instance not active: %s", info.State)
		return
	}

	// Map session to instance
	cm.sessions[req.sessionID] = req.instanceID
	info.Sessions[req.sessionID] = true

	req.response <- nil
}

func (cm *ConnectionManager) handleDisconnect(req disconnectRequest) {
	instanceID, exists := cm.sessions[req.sessionID]
	if !exists {
		req.response <- nil // Not connected
		return
	}

	// Remove session mapping
	delete(cm.sessions, req.sessionID)

	// Remove from instance sessions
	if info, exists := cm.connections[instanceID]; exists {
		delete(info.Sessions, req.sessionID)
	}

	req.response <- nil
}

func (cm *ConnectionManager) handleEnsure(req ensureRequest) {
	info, exists := cm.connections[req.instanceID]
	if !exists {
		req.response <- false
		return
	}

	info.LastActivity = time.Now()
	req.response <- info.State == StateActive
}

func (cm *ConnectionManager) handleStateChange(req stateChangeRequest) {
	info, exists := cm.connections[req.instanceID]
	if !exists {
		req.response <- fmt.Errorf("instance not found")
		return
	}

	oldState := info.State
	info.State = req.newState
	now := time.Now()
	info.StateChangedAt = now

	// Record state transition
	transition := StateTransition{
		From:      oldState,
		To:        req.newState,
		Timestamp: now,
		Reason:    req.reason,
	}
	info.StateHistory = append(info.StateHistory, transition)

	// Keep history size reasonable
	if len(info.StateHistory) > 100 {
		info.StateHistory = info.StateHistory[len(info.StateHistory)-50:]
	}

	debugLog("Instance %s: %s -> %s", req.instanceID, oldState, req.newState)

	req.response <- nil
}

func (cm *ConnectionManager) handleList(req listRequest) {
	var list []*ConnectionInfo

	for _, info := range cm.connections {
		if info.State != StateDead {
			// Make a copy to avoid races
			infoCopy := *info
			list = append(list, &infoCopy)
		}
	}

	req.response <- list
}

func (cm *ConnectionManager) handleGetClient(req getClientRequest) {
	instanceID, exists := cm.sessions[req.sessionID]
	if !exists {
		req.response <- nil
		return
	}

	info, exists := cm.connections[instanceID]
	if !exists || info.State != StateActive {
		req.response <- nil
		return
	}

	req.response <- info.Client
}

func (cm *ConnectionManager) handleSetClient(req setClientRequest) {
	info, exists := cm.connections[req.instanceID]
	if !exists {
		req.response <- fmt.Errorf("instance not found")
		return
	}

	info.Client = req.client
	info.ConnectedAt = time.Now()
	info.LastActivity = time.Now()
	info.RetryCount = 0

	req.response <- nil
}

// Connection establishment (runs in separate goroutine)
func (cm *ConnectionManager) attemptConnection(instanceID string) {
	cm.attemptConnectionWithContext(context.Background(), instanceID)
}

// attemptConnectionWithContext establishes connection with proper context management
func (cm *ConnectionManager) attemptConnectionWithContext(parentCtx context.Context, instanceID string) {
	// Get instance info
	listResp := make(chan []*ConnectionInfo)
	cm.listChan <- listRequest{response: listResp}
	connections := <-listResp

	var info *ConnectionInfo
	for _, conn := range connections {
		if conn.InstanceID == instanceID {
			info = conn
			break
		}
	}

	if info == nil {
		return
	}

	// Update state to connecting
	cm.updateState(instanceID, StateConnecting)

	// Create connection context with timeout derived from parent context
	connCtx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	// Use retry policy for robust connection establishment
	err := info.RetryPolicy.ExecuteWithRetry(func() error {
		// Check if parent context is cancelled
		select {
		case <-parentCtx.Done():
			return fmt.Errorf("parent context cancelled: %w", parentCtx.Err())
		default:
		}

		// Create HTTP client (regular or persistent based on configuration)
		client, err := NewHubClientInterface(info.Port)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		// Test connection with initialize using connection context
		initCtx, initCancel := context.WithTimeout(connCtx, 10*time.Second)
		defer initCancel()

		if err := client.Initialize(initCtx); err != nil {
			return fmt.Errorf("failed to initialize connection: %w", err)
		}

		// Connection successful - update client through channel
		respChan := make(chan error)
		cm.setClientChan <- setClientRequest{
			instanceID: instanceID,
			client:     client,
			response:   respChan,
		}
		return <-respChan
	})

	if err != nil {
		// Check if this is a context cancellation
		select {
		case <-parentCtx.Done():
			debugLog("Connection attempt to %s cancelled: %v", instanceID, parentCtx.Err())
			cm.updateStateWithReason(instanceID, StateDiscovered,
				fmt.Sprintf("Connection cancelled: %v", parentCtx.Err()))
			return
		default:
		}

		debugLog("Failed to establish connection to %s after retries: %v", instanceID, err)

		// Check if this is a circuit breaker error
		if IsCircuitBreakerError(err) {
			cm.updateStateWithReason(instanceID, StateDead,
				fmt.Sprintf("Circuit breaker open: %v", err))
		} else {
			// Calculate next retry time with exponential backoff
			nextDelay := info.RetryPolicy.backoff.NextDelay()
			cm.scheduleRetry(instanceID, nextDelay)
		}
		return
	}

	// Success - reset retry policy and mark as active
	info.RetryPolicy.Reset()
	cm.updateState(instanceID, StateActive)
}

// scheduleRetry schedules a retry attempt after a delay
func (cm *ConnectionManager) scheduleRetry(instanceID string, delay time.Duration) {
	// Update the next retry time
	nextRetryTime := time.Now().Add(delay)

	// Update state to retrying with next retry time
	req := stateChangeRequest{
		instanceID: instanceID,
		newState:   StateRetrying,
		reason:     fmt.Sprintf("Retry scheduled in %v", delay),
		response:   make(chan error),
	}
	cm.stateChan <- req
	<-req.response

	// Update NextRetryAt field
	listResp := make(chan []*ConnectionInfo)
	cm.listChan <- listRequest{response: listResp}
	connections := <-listResp

	for _, conn := range connections {
		if conn.InstanceID == instanceID {
			conn.NextRetryAt = nextRetryTime
			break
		}
	}

	debugLog("Scheduled retry for instance %s in %v", instanceID, delay)
}

// Connection monitoring
func (cm *ConnectionManager) monitorConnections() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.checkConnections()

		case <-cm.stopCh:
			return
		}
	}
}

func (cm *ConnectionManager) checkConnections() {
	// Get current connections
	listResp := make(chan []*ConnectionInfo)
	cm.listChan <- listRequest{response: listResp}
	connections := <-listResp

	for _, info := range connections {
		switch info.State {
		case StateActive:
			// Check if still responsive
			if time.Since(info.LastActivity) > 20*time.Second {
				debugLog("Instance %s not responsive, marking as retrying", info.InstanceID)
				cm.updateStateWithReason(info.InstanceID, StateRetrying,
					fmt.Sprintf("No activity for %v", time.Since(info.LastActivity)))
			}

		case StateRetrying:
			// Check if it's time to retry based on exponential backoff
			if !info.NextRetryAt.IsZero() && time.Now().After(info.NextRetryAt) {
				debugLog("Attempting retry for instance %s (attempt %d)",
					info.InstanceID, info.RetryPolicy.backoff.GetAttemptCount()+1)

				// Clear the retry time and attempt connection
				info.NextRetryAt = time.Time{}
				go cm.attemptConnection(info.InstanceID)
			}

		case StateDiscovered:
			// Try initial connection
			go cm.attemptConnection(info.InstanceID)
		}
	}
}

// Public API

func (cm *ConnectionManager) RegisterInstance(instance *discovery.Instance) error {
	respChan := make(chan error)
	cm.registerChan <- registerRequest{
		instance: instance,
		response: respChan,
	}
	return <-respChan
}

func (cm *ConnectionManager) ConnectSession(sessionID, instanceID string) error {
	respChan := make(chan error)
	cm.connectChan <- connectRequest{
		sessionID:  sessionID,
		instanceID: instanceID,
		response:   respChan,
	}
	return <-respChan
}

func (cm *ConnectionManager) DisconnectSession(sessionID string) error {
	respChan := make(chan error)
	cm.disconnectChan <- disconnectRequest{
		sessionID: sessionID,
		response:  respChan,
	}
	return <-respChan
}

func (cm *ConnectionManager) GetClient(sessionID string) HubClientInterface {
	respChan := make(chan HubClientInterface)
	cm.getClientChan <- getClientRequest{
		sessionID: sessionID,
		response:  respChan,
	}
	return <-respChan
}

func (cm *ConnectionManager) ListInstances() []*ConnectionInfo {
	respChan := make(chan []*ConnectionInfo)
	cm.listChan <- listRequest{
		response: respChan,
	}
	return <-respChan
}

func (cm *ConnectionManager) UpdateActivity(instanceID string) bool {
	respChan := make(chan bool)
	cm.ensureChan <- ensureRequest{
		instanceID: instanceID,
		response:   respChan,
	}
	return <-respChan
}

// Helper to update state
func (cm *ConnectionManager) updateState(instanceID string, newState ConnectionState) error {
	return cm.updateStateWithReason(instanceID, newState, "")
}

// updateStateWithReason updates state with a reason
func (cm *ConnectionManager) updateStateWithReason(instanceID string, newState ConnectionState, reason string) error {
	respChan := make(chan error)
	cm.stateChan <- stateChangeRequest{
		instanceID: instanceID,
		newState:   newState,
		reason:     reason,
		response:   respChan,
	}
	return <-respChan
}

func (cm *ConnectionManager) cleanup() {
	// Close all client connections
	for _, info := range cm.connections {
		if info.Client != nil {
			info.Client.Close()
		}
	}
}

func (cm *ConnectionManager) Stop() {
	// Stop network monitor first
	if cm.networkMonitor != nil {
		cm.networkMonitor.Stop()
	}

	// Stop session manager
	cm.sessionManager.Stop()

	close(cm.stopCh)
	<-cm.doneCh
}

// Context-aware connection methods

// ConnectWithContext attempts to connect to an instance using the provided context
func (cm *ConnectionManager) ConnectWithContext(ctx context.Context, instanceID string) error {
	// Use context-aware connection attempt
	go cm.attemptConnectionWithContext(ctx, instanceID)
	return nil
}

// ConnectSessionToInstance connects a session to an instance with context management
func (cm *ConnectionManager) ConnectSessionToInstance(sessionID, instanceID string) error {
	// Create or get session context
	var session *SessionContext
	if existing, err := cm.sessionManager.GetSession(sessionID); err == nil {
		session = existing
	} else {
		session = cm.sessionManager.CreateSession(sessionID, nil)
	}

	// Create connection context for this instance
	connCtx := session.GetOrCreateConnectionContext(instanceID)

	// Attempt connection using the connection context
	go cm.attemptConnectionWithContext(connCtx.Context(), instanceID)

	return nil
}

// GetSessionManager returns the session manager for external use
func (cm *ConnectionManager) GetSessionManager() *SessionManager {
	return cm.sessionManager
}

// GetNetworkMonitor returns the network monitor for external use
func (cm *ConnectionManager) GetNetworkMonitor() *NetworkMonitor {
	return cm.networkMonitor
}

// handleNetworkEvents processes network state changes and sleep/wake events
func (cm *ConnectionManager) handleNetworkEvents() {
	for {
		select {
		case event := <-cm.networkMonitor.SleepWakeEvents():
			cm.handleSleepWakeEvent(event)

		case event := <-cm.networkMonitor.NetworkEvents():
			cm.handleNetworkEvent(event)

		case <-cm.stopCh:
			return
		}
	}
}

// handleSleepWakeEvent processes system sleep/wake events
func (cm *ConnectionManager) handleSleepWakeEvent(event SleepWakeEvent) {
	debugLog("Sleep/wake event: %s at %v (%s)", event.Type, event.Timestamp, event.Reason)

	switch event.Type {
	case "wake", "suspected_wake":
		// System woke up - aggressively reconnect all instances
		cm.reconnectAllInstances("System wake detected")

	case "sleep":
		// System going to sleep - mark connections as suspicious
		cm.markAllConnectionsSuspect("System sleep detected")
	}
}

// handleNetworkEvent processes network connectivity changes
func (cm *ConnectionManager) handleNetworkEvent(event NetworkEvent) {
	debugLog("Network event: %s at %v (%s)", event.State, event.Timestamp, event.Reason)

	switch event.State {
	case NetworkStateConnected:
		// Network is back online - validate all connections
		cm.validateAllConnections("Network connectivity restored")

	case NetworkStateDisconnected:
		// Network is offline - mark connections as suspect and pause aggressive retries
		cm.markAllConnectionsSuspect("Network disconnected")

	case NetworkStateSuspicious:
		// Network state is uncertain - trigger connectivity checks
		cm.validateAllConnections("Network state suspicious")
	}
}

// reconnectAllInstances attempts to reconnect all instances
func (cm *ConnectionManager) reconnectAllInstances(reason string) {
	// Get current connections
	listResp := make(chan []*ConnectionInfo)
	select {
	case cm.listChan <- listRequest{response: listResp}:
		connections := <-listResp

		debugLog("Reconnecting %d instances due to: %s", len(connections), reason)

		for _, info := range connections {
			// Force reconnection for all instances regardless of current state
			if info.State == StateActive || info.State == StateRetrying {
				cm.updateStateWithReason(info.InstanceID, StateDiscovered,
					fmt.Sprintf("Network reconnection: %s", reason))

				// Reset retry policy to start fresh
				if info.RetryPolicy != nil {
					info.RetryPolicy.Reset()
				}

				// Trigger immediate reconnection attempt
				go cm.attemptConnection(info.InstanceID)
			}
		}
	case <-time.After(1 * time.Second):
		debugLog("Timeout getting connections for reconnection")
	}
}

// markAllConnectionsSuspect marks all active connections as potentially problematic
func (cm *ConnectionManager) markAllConnectionsSuspect(reason string) {
	// Get current connections
	listResp := make(chan []*ConnectionInfo)
	select {
	case cm.listChan <- listRequest{response: listResp}:
		connections := <-listResp

		debugLog("Marking %d connections as suspect due to: %s", len(connections), reason)

		for _, info := range connections {
			if info.State == StateActive {
				cm.updateStateWithReason(info.InstanceID, StateRetrying,
					fmt.Sprintf("Network issue: %s", reason))
			}
		}
	case <-time.After(1 * time.Second):
		debugLog("Timeout getting connections for marking suspect")
	}
}

// validateAllConnections triggers health checks for all connections
func (cm *ConnectionManager) validateAllConnections(reason string) {
	// Get current connections
	listResp := make(chan []*ConnectionInfo)
	select {
	case cm.listChan <- listRequest{response: listResp}:
		connections := <-listResp

		debugLog("Validating %d connections due to: %s", len(connections), reason)

		for _, info := range connections {
			// Trigger immediate connection attempt for non-active instances
			if info.State != StateActive {
				// Reset retry policy to avoid exponential backoff delays
				if info.RetryPolicy != nil {
					info.RetryPolicy.Reset()
				}

				go cm.attemptConnection(info.InstanceID)
			}
		}
	case <-time.After(1 * time.Second):
		debugLog("Timeout getting connections for validation")
	}
}
