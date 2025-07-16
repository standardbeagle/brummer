package testutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ErrorInjector provides systematic error injection capabilities for testing
type ErrorInjector struct {
	mu       sync.RWMutex
	failures map[string]*InjectionRule
}

// InjectionRule defines when and how to inject errors
type InjectionRule struct {
	FailCount    int32         // Number of times to fail (-1 for unlimited)
	FailureType  string        // Type of failure to inject
	ErrorMessage string        // Custom error message
	Delay        time.Duration // Delay before failure
}

// NewErrorInjector creates a new error injection system
func NewErrorInjector() *ErrorInjector {
	return &ErrorInjector{
		failures: make(map[string]*InjectionRule),
	}
}

// InjectFailure configures an error injection rule
func (ei *ErrorInjector) InjectFailure(operation string, rule *InjectionRule) {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	ei.failures[operation] = rule
}

// ShouldFail checks if an operation should fail and returns appropriate error
func (ei *ErrorInjector) ShouldFail(operation string) error {
	ei.mu.RLock()
	rule, exists := ei.failures[operation]
	ei.mu.RUnlock()

	if !exists {
		return nil
	}

	// Check if we should still fail
	if rule.FailCount == 0 {
		return nil
	}

	// Apply delay if specified
	if rule.Delay > 0 {
		time.Sleep(rule.Delay)
	}

	// Decrement failure count (if not unlimited)
	if rule.FailCount > 0 {
		atomic.AddInt32(&rule.FailCount, -1)
	}

	// Return appropriate error based on failure type
	switch rule.FailureType {
	case "network":
		return errors.New("network error: " + rule.ErrorMessage)
	case "timeout":
		return context.DeadlineExceeded
	case "resource":
		return errors.New("resource exhausted: " + rule.ErrorMessage)
	case "permission":
		return errors.New("permission denied: " + rule.ErrorMessage)
	case "io":
		return io.ErrUnexpectedEOF
	default:
		return errors.New(rule.ErrorMessage)
	}
}

// Reset clears all injection rules
func (ei *ErrorInjector) Reset() {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	ei.failures = make(map[string]*InjectionRule)
}

// NetworkErrorInjector creates HTTP servers that can fail on demand
type NetworkErrorInjector struct {
	server     *httptest.Server
	shouldFail atomic.Bool
	errorType  atomic.Value // stores string
	mu         sync.RWMutex
	responses  map[string]func(w http.ResponseWriter, r *http.Request)
}

// NewNetworkErrorInjector creates a controllable HTTP server for network failure testing
func NewNetworkErrorInjector(t *testing.T) *NetworkErrorInjector {
	nei := &NetworkErrorInjector{
		responses: make(map[string]func(w http.ResponseWriter, r *http.Request)),
	}

	nei.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if nei.shouldFail.Load() {
			errorType := nei.errorType.Load()
			if errorType == nil {
				errorType = "generic"
			}

			switch errorType.(string) {
			case "timeout":
				// Simulate timeout by delaying response
				time.Sleep(5 * time.Second)
			case "connection_refused":
				// Close connection immediately
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, _ := hj.Hijack()
					conn.Close()
				}
				return
			case "bad_response":
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
				return
			case "malformed_json":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("{invalid json"))
				return
			default:
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}

		// Normal response
		nei.mu.RLock()
		handler, exists := nei.responses[r.URL.Path]
		nei.mu.RUnlock()

		if exists {
			handler(w, r)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}))

	t.Cleanup(func() {
		nei.server.Close()
	})

	return nei
}

// URL returns the server URL
func (nei *NetworkErrorInjector) URL() string {
	return nei.server.URL
}

// InjectError configures the server to return errors
func (nei *NetworkErrorInjector) InjectError(errorType string) {
	nei.shouldFail.Store(true)
	nei.errorType.Store(errorType)
}

// StopInjection stops error injection
func (nei *NetworkErrorInjector) StopInjection() {
	nei.shouldFail.Store(false)
}

// SetResponse sets a custom response handler for a path
func (nei *NetworkErrorInjector) SetResponse(path string, handler func(w http.ResponseWriter, r *http.Request)) {
	nei.mu.Lock()
	defer nei.mu.Unlock()
	nei.responses[path] = handler
}

// ProcessKiller provides controlled process termination for testing
type ProcessKiller struct {
	processes map[int]context.CancelFunc
	mu        sync.RWMutex
}

// NewProcessKiller creates a new process killer
func NewProcessKiller() *ProcessKiller {
	return &ProcessKiller{
		processes: make(map[int]context.CancelFunc),
	}
}

// Register registers a process for potential killing
func (pk *ProcessKiller) Register(pid int, cancel context.CancelFunc) {
	pk.mu.Lock()
	defer pk.mu.Unlock()
	pk.processes[pid] = cancel
}

// Kill terminates a registered process
func (pk *ProcessKiller) Kill(pid int) error {
	pk.mu.RLock()
	cancel, exists := pk.processes[pid]
	pk.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process %d not registered", pid)
	}

	cancel()
	return nil
}

// KillAll terminates all registered processes
func (pk *ProcessKiller) KillAll() {
	pk.mu.RLock()
	defer pk.mu.RUnlock()

	for _, cancel := range pk.processes {
		cancel()
	}
}

// ResourceExhauster simulates resource exhaustion scenarios
type ResourceExhauster struct {
	memoryLimit int64
	fdLimit     int
	mu          sync.RWMutex
}

// NewResourceExhauster creates a resource exhaustion simulator
func NewResourceExhauster() *ResourceExhauster {
	return &ResourceExhauster{
		memoryLimit: -1, // -1 means no limit
		fdLimit:     -1,
	}
}

// SetMemoryLimit sets the memory allocation limit in bytes
func (re *ResourceExhauster) SetMemoryLimit(bytes int64) {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.memoryLimit = bytes
}

// SetFDLimit sets the file descriptor limit
func (re *ResourceExhauster) SetFDLimit(count int) {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.fdLimit = count
}

// CheckMemoryAllocation checks if memory allocation should fail
func (re *ResourceExhauster) CheckMemoryAllocation(bytes int64) error {
	re.mu.RLock()
	defer re.mu.RUnlock()

	if re.memoryLimit >= 0 && bytes > re.memoryLimit {
		return errors.New("memory allocation failed: limit exceeded")
	}
	return nil
}

// CheckFDAllocation checks if file descriptor allocation should fail
func (re *ResourceExhauster) CheckFDAllocation() error {
	re.mu.RLock()
	defer re.mu.RUnlock()

	if re.fdLimit >= 0 && re.fdLimit <= 0 {
		return errors.New("file descriptor allocation failed: limit exceeded")
	}
	if re.fdLimit > 0 {
		re.fdLimit--
	}
	return nil
}

// NetworkPartitionSimulator simulates network partitions
type NetworkPartitionSimulator struct {
	partitioned atomic.Bool
	listener    net.Listener
	connections []net.Conn
	mu          sync.RWMutex
}

// NewNetworkPartitionSimulator creates a network partition simulator
func NewNetworkPartitionSimulator(t *testing.T, addr string) *NetworkPartitionSimulator {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	nps := &NetworkPartitionSimulator{
		listener:    listener,
		connections: make([]net.Conn, 0),
	}

	t.Cleanup(func() {
		nps.Stop()
	})

	return nps
}

// Start begins accepting connections
func (nps *NetworkPartitionSimulator) Start() {
	go func() {
		for {
			conn, err := nps.listener.Accept()
			if err != nil {
				return // Listener closed
			}

			nps.mu.Lock()
			nps.connections = append(nps.connections, conn)
			nps.mu.Unlock()

			go nps.handleConnection(conn)
		}
	}()
}

// Partition simulates a network partition by closing all connections
func (nps *NetworkPartitionSimulator) Partition() {
	nps.partitioned.Store(true)

	nps.mu.Lock()
	defer nps.mu.Unlock()

	for _, conn := range nps.connections {
		conn.Close()
	}
	nps.connections = nps.connections[:0]
}

// Repair repairs the network partition
func (nps *NetworkPartitionSimulator) Repair() {
	nps.partitioned.Store(false)
}

// Stop stops the simulator
func (nps *NetworkPartitionSimulator) Stop() {
	nps.listener.Close()
	nps.Partition()
}

func (nps *NetworkPartitionSimulator) handleConnection(conn net.Conn) {
	defer conn.Close()

	// If partitioned, close immediately
	if nps.partitioned.Load() {
		return
	}

	// Echo server for testing
	buffer := make([]byte, 1024)
	for {
		if nps.partitioned.Load() {
			return
		}

		n, err := conn.Read(buffer)
		if err != nil {
			return
		}

		_, err = conn.Write(buffer[:n])
		if err != nil {
			return
		}
	}
}

// TestScenario represents a complete test scenario with error injection
type TestScenario struct {
	Name        string
	Description string
	Setup       func(t *testing.T) interface{}
	Execute     func(t *testing.T, context interface{}) error
	Verify      func(t *testing.T, context interface{}, err error)
	Cleanup     func(t *testing.T, context interface{})
}

// RunTestScenarios executes a series of test scenarios
func RunTestScenarios(t *testing.T, scenarios []TestScenario) {
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var context interface{}
			if scenario.Setup != nil {
				context = scenario.Setup(t)
			}

			if scenario.Cleanup != nil {
				defer scenario.Cleanup(t, context)
			}

			err := scenario.Execute(t, context)
			scenario.Verify(t, context, err)
		})
	}
}
