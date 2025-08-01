package events

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

type EventType string

const (
	ProcessStarted  EventType = "process.started"
	ProcessExited   EventType = "process.exited"
	LogLine         EventType = "log.line"
	ErrorDetected   EventType = "error.detected"
	BuildEvent      EventType = "build.event"
	TestFailed      EventType = "test.failed"
	TestPassed      EventType = "test.passed"
	MCPActivity     EventType = "mcp.activity"
	MCPConnected    EventType = "mcp.connected"
	MCPDisconnected EventType = "mcp.disconnected"
)

type Event struct {
	ID        string
	Type      EventType
	ProcessID string
	Timestamp time.Time
	Data      map[string]interface{}
}

type Handler func(event Event)

// WorkerPoolConfig holds configuration for the event bus worker pool
type WorkerPoolConfig struct {
	WorkerCount int // Number of worker goroutines (default: CPU cores * 2.5)
	BufferSize  int // Channel buffer size (default: 1000)
}

// DefaultWorkerPoolConfig returns the default configuration
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		WorkerCount: int(float64(runtime.NumCPU()) * 2.5),
		BufferSize:  1000,
	}
}

type eventTask struct {
	event   Event
	handler Handler
}

type EventBus struct {
	handlers   map[EventType][]Handler
	mu         sync.RWMutex
	workerPool chan eventTask
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	config     WorkerPoolConfig
}

func NewEventBus() *EventBus {
	return NewEventBusWithConfig(DefaultWorkerPoolConfig())
}

func NewEventBusWithConfig(config WorkerPoolConfig) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	eb := &EventBus{
		handlers:   make(map[EventType][]Handler),
		workerPool: make(chan eventTask, config.BufferSize),
		ctx:        ctx,
		cancel:     cancel,
		config:     config,
	}

	// Start worker goroutines
	for i := 0; i < config.WorkerCount; i++ {
		eb.wg.Add(1)
		go eb.worker()
	}

	return eb
}

// worker processes events from the worker pool
func (eb *EventBus) worker() {
	defer eb.wg.Done()

	for {
		select {
		case task := <-eb.workerPool:
			// Execute handler with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Log panic but continue processing (could add logging here)
						fmt.Printf("EventBus handler panic: %v\n", r)
					}
				}()
				task.handler(task.event)
			}()
		case <-eb.ctx.Done():
			return
		}
	}
}

func (eb *EventBus) Subscribe(eventType EventType, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

func (eb *EventBus) Publish(event Event) {
	event.Timestamp = time.Now()
	event.ID = generateEventID()

	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		task := eventTask{
			event:   event,
			handler: handler,
		}

		// Non-blocking send to worker pool
		select {
		case eb.workerPool <- task:
			// Task queued successfully
		default:
			// Worker pool full - execute synchronously as fallback
			go func(h Handler, e Event) {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("EventBus fallback handler panic: %v\n", r)
					}
				}()
				h(e)
			}(handler, event)
		}
	}
}

// Shutdown gracefully shuts down the EventBus worker pool
func (eb *EventBus) Shutdown() {
	eb.cancel()
	eb.wg.Wait()
}

func generateEventID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// AI Coder Event Integration

// RegisterAICoderAggregator creates and registers an AI coder event aggregator
func (eb *EventBus) RegisterAICoderAggregator(maxEvents int) *AICoderEventAggregator {
	return NewAICoderEventAggregator(eb, maxEvents)
}

// SubscribeAICoderEvents subscribes to all AI coder events for a specific coder
func (eb *EventBus) SubscribeAICoderEvents(coderID string, handler Handler) {
	// Subscribe to all AI coder events for specific coder
	eventTypes := []EventType{
		EventAICoderCreated, EventAICoderStarted, EventAICoderPaused,
		EventAICoderResumed, EventAICoderCompleted, EventAICoderFailed,
		EventAICoderStopped, EventAICoderDeleted, EventAICoderProgress,
	}

	for _, eventType := range eventTypes {
		eb.Subscribe(eventType, func(event Event) {
			// Check if event is for the specific coder
			if eventCoderID, ok := event.Data["coder_id"].(string); ok && eventCoderID == coderID {
				handler(event)
			}
		})
	}
}

// EmitAICoderEvent emits an AI coder event with validation
func (eb *EventBus) EmitAICoderEvent(eventType EventType, coderID, coderName string, data map[string]interface{}) {
	if string(eventType) == "" {
		log.Printf("Warning: AI coder event missing type")
		return
	}

	if coderID == "" {
		log.Printf("Warning: AI coder event missing coder ID")
		return
	}

	// Ensure data map exists
	if data == nil {
		data = make(map[string]interface{})
	}

	// Add coder information to data
	data["coder_id"] = coderID
	data["coder_name"] = coderName

	eb.Publish(Event{
		Type:      eventType,
		ProcessID: fmt.Sprintf("ai-coder-%s", coderID),
		Timestamp: time.Now(),
		Data:      data,
	})
}
