package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
)

// ImprovedStreamableServer fixes goroutine leaks and adds robustness
type ImprovedStreamableServer struct {
	*StreamableServer
	
	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	// Worker pools
	notificationWorkers *WorkerPool
	eventWorkers        *WorkerPool
	
	// Cleanup tracking
	cleanupMu      sync.Mutex
	cleanupFuncs   []func()
	eventSubIDs    map[events.EventType]events.HandlerID
	
	// Metrics
	activeGoroutines   atomic.Int64
	panicRecoveries    atomic.Int64
}

// WorkerPool manages a fixed number of worker goroutines
type WorkerPool struct {
	workers    int
	tasks      chan func()
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(ctx context.Context, workers int, bufferSize int) *WorkerPool {
	poolCtx, cancel := context.WithCancel(ctx)
	pool := &WorkerPool{
		workers: workers,
		tasks:   make(chan func(), bufferSize),
		ctx:     poolCtx,
		cancel:  cancel,
	}
	
	// Start workers
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}
	
	return pool
}

// worker processes tasks from the queue
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			// Execute task with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Log panic (in production, use proper logging)
						fmt.Printf("Worker %d recovered from panic: %v\n", id, r)
					}
				}()
				task()
			}()
		}
	}
}

// Submit adds a task to the pool
func (p *WorkerPool) Submit(task func()) error {
	select {
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	case p.tasks <- task:
		return nil
	default:
		return fmt.Errorf("worker pool queue is full")
	}
}

// Stop gracefully shuts down the pool
func (p *WorkerPool) Stop() {
	p.cancel()
	close(p.tasks)
	p.wg.Wait()
}

// NewImprovedStreamableServer creates a server with goroutine leak fixes
func NewImprovedStreamableServer(port int, pm *process.Manager, ls *logs.Store, ps *proxy.Server, eb *events.EventBus) *ImprovedStreamableServer {
	ctx, cancel := context.WithCancel(context.Background())
	
	s := &ImprovedStreamableServer{
		StreamableServer: NewStreamableServer(port, pm, ls, ps, eb),
		ctx:              ctx,
		cancel:           cancel,
		eventSubIDs:      make(map[events.EventType]events.HandlerID),
	}
	
	// Create worker pools
	s.notificationWorkers = NewWorkerPool(ctx, 10, 100)
	s.eventWorkers = NewWorkerPool(ctx, 5, 50)
	
	return s
}

// Start initializes the server with proper lifecycle management
func (s *ImprovedStreamableServer) Start() error {
	// Start base server
	if err := s.StreamableServer.Start(); err != nil {
		return err
	}
	
	// Setup improved event handlers
	s.setupImprovedEventHandlers()
	
	return nil
}

// Stop gracefully shuts down the server and all goroutines
func (s *ImprovedStreamableServer) Stop() error {
	// Cancel context to signal shutdown
	s.cancel()
	
	// Stop worker pools
	s.notificationWorkers.Stop()
	s.eventWorkers.Stop()
	
	// Unsubscribe from all events
	s.cleanupMu.Lock()
	for eventType, handlerID := range s.eventSubIDs {
		if improvedEB, ok := s.eventBus.(*events.ImprovedEventBus); ok {
			improvedEB.Unsubscribe(eventType, handlerID)
		}
	}
	
	// Run cleanup functions
	for _, cleanup := range s.cleanupFuncs {
		cleanup()
	}
	s.cleanupMu.Unlock()
	
	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// All goroutines finished
	case <-time.After(5 * time.Second):
		return fmt.Errorf("shutdown timed out waiting for goroutines")
	}
	
	// Stop base server
	return s.StreamableServer.Stop()
}

// setupImprovedEventHandlers sets up event handlers with proper cleanup
func (s *ImprovedStreamableServer) setupImprovedEventHandlers() {
	// Use improved event bus if available
	improvedEB, isImproved := s.eventBus.(*events.ImprovedEventBus)
	
	// Subscribe to process events
	handler := func(event events.Event) {
		// Use worker pool for event processing
		s.eventWorkers.Submit(func() {
			s.handleEventWithRecovery(event)
		})
	}
	
	if isImproved {
		// Subscribe with improved event bus
		handlerOpts := &events.HandlerOptions{
			Timeout:      5 * time.Second,
			RecoverPanic: true,
			NonBlocking:  false, // We handle async in worker pool
		}
		
		if id, err := improvedEB.Subscribe(events.ProcessStarted, handler, handlerOpts); err == nil {
			s.eventSubIDs[events.ProcessStarted] = id
		}
		
		if id, err := improvedEB.Subscribe(events.ProcessExited, handler, handlerOpts); err == nil {
			s.eventSubIDs[events.ProcessExited] = id
		}
		
		if id, err := improvedEB.Subscribe(events.LogLine, handler, handlerOpts); err == nil {
			s.eventSubIDs[events.LogLine] = id
		}
		
		if id, err := improvedEB.Subscribe(events.ErrorDetected, handler, handlerOpts); err == nil {
			s.eventSubIDs[events.ErrorDetected] = id
		}
	} else {
		// Fallback to regular event bus
		s.eventBus.Subscribe(events.ProcessStarted, handler)
		s.eventBus.Subscribe(events.ProcessExited, handler)
		s.eventBus.Subscribe(events.LogLine, handler)
		s.eventBus.Subscribe(events.ErrorDetected, handler)
	}
}

// handleEventWithRecovery processes an event with panic recovery
func (s *ImprovedStreamableServer) handleEventWithRecovery(event events.Event) {
	defer func() {
		if r := recover(); r != nil {
			s.panicRecoveries.Add(1)
			s.logStore.Add("mcp-server", "ERROR", 
				fmt.Sprintf("Panic in event handler: %v", r), true)
		}
	}()
	
	// Convert to resource update based on event type
	var resourceURI string
	switch event.Type {
	case events.ProcessStarted, events.ProcessExited:
		resourceURI = "processes://active"
	case events.LogLine:
		resourceURI = "logs://recent"
	case events.ErrorDetected:
		resourceURI = "logs://errors"
	default:
		return
	}
	
	// Broadcast resource update
	s.BroadcastResourceUpdate(resourceURI)
}

// BroadcastNotification sends notifications with worker pool
func (s *ImprovedStreamableServer) BroadcastNotification(method string, params interface{}) {
	s.mu.RLock()
	sessions := make([]*ClientSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		if session.eventStream != nil {
			sessions = append(sessions, session)
		}
	}
	s.mu.RUnlock()
	
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	
	// Send to each session using worker pool
	for _, session := range sessions {
		session := session // Capture for closure
		err := s.notificationWorkers.Submit(func() {
			s.sendNotificationWithTimeout(session, notification)
		})
		if err != nil {
			// Worker pool is full or shutting down
			s.logStore.Add("mcp-server", "WARN", 
				fmt.Sprintf("Failed to queue notification: %v", err), false)
		}
	}
}

// sendNotificationWithTimeout sends a notification with timeout
func (s *ImprovedStreamableServer) sendNotificationWithTimeout(session *ClientSession, notification interface{}) {
	// Create timeout context
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()
	
	// Send in goroutine to respect timeout
	done := make(chan error, 1)
	go func() {
		done <- s.sendSSEEvent(session.eventStream, notification)
	}()
	
	select {
	case <-ctx.Done():
		// Timeout or shutdown
		s.logStore.Add("mcp-server", "WARN", 
			fmt.Sprintf("Notification timed out for session %s", session.ID), false)
	case err := <-done:
		if err != nil {
			s.logStore.Add("mcp-server", "ERROR", 
				fmt.Sprintf("Failed to send notification: %v", err), true)
		}
	}
}

// handleSSEImproved handles SSE connections with proper cleanup
func (s *ImprovedStreamableServer) handleSSEImproved(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	
	// Get session ID
	sessionID := r.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		sessionID = generateSessionID()
	}
	
	// Create or get session
	s.mu.Lock()
	session, exists := s.sessions[sessionID]
	if !exists {
		session = &ClientSession{
			ID:              sessionID,
			ConnectedAt:     time.Now(),
			LastActivity:    time.Now(),
			eventStream:     w,
			resourceUpdates: make(chan string, 100),
			replResponses:   make(map[string]chan json.RawMessage),
		}
		s.sessions[sessionID] = session
	} else {
		session.eventStream = w
	}
	s.mu.Unlock()
	
	// Track cleanup
	defer func() {
		s.mu.Lock()
		session.eventStream = nil
		// Clean up channels
		close(session.resourceUpdates)
		for _, ch := range session.replResponses {
			close(ch)
		}
		s.mu.Unlock()
	}()
	
	// Setup context for this connection
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	
	// Heartbeat ticker
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()
	
	// Resource update aggregator
	updateAggregator := NewUpdateAggregator(100 * time.Millisecond)
	defer updateAggregator.Stop()
	
	// Main event loop
	for {
		select {
		case <-ctx.Done():
			return
			
		case <-s.ctx.Done():
			// Server shutting down
			return
			
		case <-heartbeat.C:
			// Send ping with timeout
			pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
			err := s.sendSSEPing(w, pingCtx)
			pingCancel()
			if err != nil {
				return
			}
			
		case update := <-session.resourceUpdates:
			updateAggregator.Add(update)
			
		case updates := <-updateAggregator.Updates():
			// Send aggregated updates
			for _, uri := range updates {
				notification := map[string]interface{}{
					"jsonrpc": "2.0",
					"method":  "notifications/resources/updated",
					"params": map[string]interface{}{
						"uri": uri,
					},
				}
				if err := s.sendSSEEvent(w, notification); err != nil {
					return
				}
			}
		}
	}
}

// UpdateAggregator batches resource updates to avoid flooding
type UpdateAggregator struct {
	pending  map[string]struct{}
	mu       sync.Mutex
	interval time.Duration
	updates  chan []string
	stop     chan struct{}
}

func NewUpdateAggregator(interval time.Duration) *UpdateAggregator {
	a := &UpdateAggregator{
		pending:  make(map[string]struct{}),
		interval: interval,
		updates:  make(chan []string),
		stop:     make(chan struct{}),
	}
	go a.run()
	return a
}

func (a *UpdateAggregator) run() {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-a.stop:
			return
		case <-ticker.C:
			a.flush()
		}
	}
}

func (a *UpdateAggregator) Add(uri string) {
	a.mu.Lock()
	a.pending[uri] = struct{}{}
	a.mu.Unlock()
}

func (a *UpdateAggregator) flush() {
	a.mu.Lock()
	if len(a.pending) == 0 {
		a.mu.Unlock()
		return
	}
	
	updates := make([]string, 0, len(a.pending))
	for uri := range a.pending {
		updates = append(updates, uri)
	}
	a.pending = make(map[string]struct{})
	a.mu.Unlock()
	
	select {
	case a.updates <- updates:
	case <-time.After(100 * time.Millisecond):
		// Don't block if no one is reading
	}
}

func (a *UpdateAggregator) Updates() <-chan []string {
	return a.updates
}

func (a *UpdateAggregator) Stop() {
	close(a.stop)
}

// sendSSEPing sends a ping event with context
func (s *ImprovedStreamableServer) sendSSEPing(w http.ResponseWriter, ctx context.Context) error {
	done := make(chan error, 1)
	go func() {
		_, err := fmt.Fprintf(w, "event: ping\ndata: %d\n\n", time.Now().Unix())
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		done <- err
	}()
	
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// GetMetrics returns server metrics
func (s *ImprovedStreamableServer) GetMetrics() map[string]interface{} {
	baseMetrics := s.StreamableServer.GetMetrics()
	
	// Add improved server metrics
	baseMetrics["active_goroutines"] = s.activeGoroutines.Load()
	baseMetrics["panic_recoveries"] = s.panicRecoveries.Load()
	baseMetrics["notification_queue_size"] = len(s.notificationWorkers.tasks)
	baseMetrics["event_queue_size"] = len(s.eventWorkers.tasks)
	
	return baseMetrics
}