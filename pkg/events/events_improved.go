package events

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// HandlerID uniquely identifies a subscription
type HandlerID string

// HandlerOptions configures handler behavior
type HandlerOptions struct {
	Timeout      time.Duration
	MaxRetries   int
	RetryDelay   time.Duration
	NonBlocking  bool
	RecoverPanic bool
}

// DefaultHandlerOptions provides sensible defaults
var DefaultHandlerOptions = HandlerOptions{
	Timeout:      5 * time.Second,
	MaxRetries:   0,
	RetryDelay:   100 * time.Millisecond,
	NonBlocking:  true,
	RecoverPanic: true,
}

// ImprovedEventBus fixes race conditions and adds robustness
type ImprovedEventBus struct {
	handlers    map[EventType]map[HandlerID]*handlerInfo
	mu          sync.RWMutex
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	idCounter   atomic.Uint64
	maxHandlers int
	
	// Metrics for monitoring
	publishedEvents  atomic.Uint64
	failedHandlers   atomic.Uint64
	timedOutHandlers atomic.Uint64
}

type handlerInfo struct {
	id      HandlerID
	handler Handler
	options HandlerOptions
}

// NewImprovedEventBus creates a robust event bus
func NewImprovedEventBus(maxHandlers int) *ImprovedEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &ImprovedEventBus{
		handlers:    make(map[EventType]map[HandlerID]*handlerInfo),
		ctx:         ctx,
		cancel:      cancel,
		maxHandlers: maxHandlers,
	}
}

// Subscribe adds a handler with options and returns an ID for unsubscribing
func (eb *ImprovedEventBus) Subscribe(eventType EventType, handler Handler, opts *HandlerOptions) (HandlerID, error) {
	if opts == nil {
		opts = &DefaultHandlerOptions
	}
	
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	// Check if we've hit the max handlers limit
	totalHandlers := 0
	for _, handlers := range eb.handlers {
		totalHandlers += len(handlers)
	}
	if eb.maxHandlers > 0 && totalHandlers >= eb.maxHandlers {
		return "", fmt.Errorf("maximum number of handlers (%d) reached", eb.maxHandlers)
	}
	
	id := HandlerID(fmt.Sprintf("%s-%d", eventType, eb.idCounter.Add(1)))
	
	if eb.handlers[eventType] == nil {
		eb.handlers[eventType] = make(map[HandlerID]*handlerInfo)
	}
	
	eb.handlers[eventType][id] = &handlerInfo{
		id:      id,
		handler: handler,
		options: *opts,
	}
	
	return id, nil
}

// Unsubscribe removes a handler by ID
func (eb *ImprovedEventBus) Unsubscribe(eventType EventType, id HandlerID) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	handlers, exists := eb.handlers[eventType]
	if !exists {
		return fmt.Errorf("no handlers registered for event type %s", eventType)
	}
	
	if _, exists := handlers[id]; !exists {
		return fmt.Errorf("handler %s not found for event type %s", id, eventType)
	}
	
	delete(handlers, id)
	
	// Clean up empty maps
	if len(handlers) == 0 {
		delete(eb.handlers, eventType)
	}
	
	return nil
}

// UnsubscribeAll removes all handlers for an event type
func (eb *ImprovedEventBus) UnsubscribeAll(eventType EventType) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	delete(eb.handlers, eventType)
}

// Publish sends an event to all registered handlers
func (eb *ImprovedEventBus) Publish(event Event) {
	event.Timestamp = time.Now()
	event.ID = generateEventID()
	
	eb.publishedEvents.Add(1)
	
	eb.mu.RLock()
	handlers := make([]*handlerInfo, 0)
	if typeHandlers, exists := eb.handlers[event.Type]; exists {
		for _, info := range typeHandlers {
			handlers = append(handlers, info)
		}
	}
	eb.mu.RUnlock()
	
	for _, info := range handlers {
		if info.options.NonBlocking {
			eb.wg.Add(1)
			go eb.executeHandler(info, event)
		} else {
			eb.executeHandler(info, event)
		}
	}
}

// executeHandler runs a handler with all safety features
func (eb *ImprovedEventBus) executeHandler(info *handlerInfo, event Event) {
	if info.options.NonBlocking {
		defer eb.wg.Done()
	}
	
	// Setup panic recovery if enabled
	if info.options.RecoverPanic {
		defer func() {
			if r := recover(); r != nil {
				eb.failedHandlers.Add(1)
				// In production, log this error
				fmt.Printf("Handler %s panicked: %v\n", info.id, r)
			}
		}()
	}
	
	// Create context with timeout
	ctx := eb.ctx
	if info.options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, info.options.Timeout)
		defer cancel()
	}
	
	// Execute with retries
	attempts := 0
	maxAttempts := info.options.MaxRetries + 1
	
	for attempts < maxAttempts {
		attempts++
		
		// Run handler in a goroutine to respect timeout
		done := make(chan struct{})
		var handlerErr error
		
		go func() {
			defer close(done)
			
			// Create a wrapper that checks context
			wrappedHandler := func(e Event) {
				select {
				case <-ctx.Done():
					handlerErr = ctx.Err()
					return
				default:
					info.handler(e)
				}
			}
			
			wrappedHandler(event)
		}()
		
		select {
		case <-done:
			if handlerErr == nil {
				return // Success
			}
		case <-ctx.Done():
			eb.timedOutHandlers.Add(1)
			return
		}
		
		// Retry logic
		if attempts < maxAttempts && info.options.RetryDelay > 0 {
			select {
			case <-time.After(info.options.RetryDelay):
				// Continue to next attempt
			case <-ctx.Done():
				return
			}
		}
	}
	
	eb.failedHandlers.Add(1)
}

// Shutdown gracefully stops the event bus
func (eb *ImprovedEventBus) Shutdown(timeout time.Duration) error {
	eb.cancel()
	
	done := make(chan struct{})
	go func() {
		eb.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timed out after %v", timeout)
	}
}

// GetMetrics returns current metrics
func (eb *ImprovedEventBus) GetMetrics() map[string]uint64 {
	return map[string]uint64{
		"published_events":   eb.publishedEvents.Load(),
		"failed_handlers":    eb.failedHandlers.Load(),
		"timed_out_handlers": eb.timedOutHandlers.Load(),
		"active_handlers":    uint64(eb.getHandlerCount()),
	}
}

func (eb *ImprovedEventBus) getHandlerCount() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	count := 0
	for _, handlers := range eb.handlers {
		count += len(handlers)
	}
	return count
}