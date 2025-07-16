package events

import (
	"fmt"
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

type EventBus struct {
	handlers map[EventType][]Handler
	mu       sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]Handler),
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
		go handler(event)
	}
}

func generateEventID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
