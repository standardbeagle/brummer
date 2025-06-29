package mcp

import (
	"context"
	"fmt"
	"log"
	"time"
	
	"github.com/standardbeagle/brummer/internal/discovery"
)

// Connection states
type ConnectionState int

const (
	StateDiscovered ConnectionState = iota  // File found, not connected
	StateConnecting                         // Attempting connection
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
	InstanceID     string
	Name           string
	Directory      string
	Port           int
	ProcessPID     int
	
	// Connection state
	State          ConnectionState
	Client         *HubClient      // HTTP client to instance
	LastActivity   time.Time
	ConnectedAt    time.Time
	RetryCount     int
	
	// Session mapping
	Sessions       map[string]bool // Active sessions for this instance
	
	// State timing tracking
	DiscoveredAt   time.Time
	StateChangedAt time.Time
	StateHistory   []StateTransition
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
	response  chan *HubClient
}

type setClientRequest struct {
	instanceID string
	client     *HubClient
	response   chan error
}

// ConnectionManager manages all instance connections
type ConnectionManager struct {
	connections map[string]*ConnectionInfo
	sessions    map[string]string  // sessionID -> instanceID
	
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
	}
	
	go cm.run()
	
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
	
	log.Printf("Instance %s: %s -> %s", req.instanceID, oldState, req.newState)
	
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
	
	// Create HTTP client
	client, err := NewHubClient(info.Port)
	if err != nil {
		log.Printf("Failed to create client for %s: %v", instanceID, err)
		cm.updateState(instanceID, StateRetrying)
		return
	}
	
	// Test connection with initialize
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Initialize(ctx); err != nil {
		log.Printf("Failed to initialize connection to %s: %v", instanceID, err)
		cm.updateState(instanceID, StateRetrying)
		return
	}
	
	// Connection successful - update client through channel
	respChan := make(chan error)
	cm.setClientChan <- setClientRequest{
		instanceID: instanceID,
		client:     client,
		response:   respChan,
	}
	<-respChan
	
	cm.updateState(instanceID, StateActive)
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
				log.Printf("Instance %s not responsive, marking as retrying", info.InstanceID)
				cm.updateStateWithReason(info.InstanceID, StateRetrying, 
					fmt.Sprintf("No activity for %v", time.Since(info.LastActivity)))
			}
			
		case StateRetrying:
			// Implement retry logic
			if info.RetryCount < 3 {
				info.RetryCount++
				go cm.attemptConnection(info.InstanceID)
			} else {
				cm.updateState(info.InstanceID, StateDead)
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

func (cm *ConnectionManager) GetClient(sessionID string) *HubClient {
	respChan := make(chan *HubClient)
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
	close(cm.stopCh)
	<-cm.doneCh
}