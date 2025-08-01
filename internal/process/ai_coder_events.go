package process

import (
	"fmt"
	"strings"
	"time"

	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/pkg/events"
)

// AICoderEventBridge coordinates events between AI coder service and process manager
type AICoderEventBridge struct {
	processMgr *Manager
	aiCoderMgr *aicoder.AICoderManager
	eventBus   *events.EventBus
}

// NewAICoderEventBridge creates a new event bridge
func NewAICoderEventBridge(processMgr *Manager, aiCoderMgr *aicoder.AICoderManager, eventBus *events.EventBus) *AICoderEventBridge {
	return &AICoderEventBridge{
		processMgr: processMgr,
		aiCoderMgr: aiCoderMgr,
		eventBus:   eventBus,
	}
}

// Start begins event bridging between systems
func (bridge *AICoderEventBridge) Start() {
	// Subscribe to all AI coder events to translate them to process events
	aiCoderEventTypes := []events.EventType{
		events.EventAICoderCreated, events.EventAICoderStarted, events.EventAICoderPaused,
		events.EventAICoderResumed, events.EventAICoderCompleted, events.EventAICoderFailed,
		events.EventAICoderStopped, events.EventAICoderDeleted, events.EventAICoderProgress,
		events.EventAICoderMilestone, events.EventAICoderOutput,
	}

	for _, eventType := range aiCoderEventTypes {
		bridge.eventBus.Subscribe(eventType, bridge.handleAICoderEvent)
	}

	// Listen for process control events that target AI coders
	bridge.eventBus.Subscribe("process_control", bridge.handleProcessControlEvent)
}

// handleProcessControlEvent handles process control events for AI coders
func (bridge *AICoderEventBridge) handleProcessControlEvent(event events.Event) {
	// Only handle events for AI coder processes
	if !strings.HasPrefix(event.ProcessID, "ai-coder-") {
		return
	}

	// Extract AI coder ID from process ID
	coderID := strings.TrimPrefix(event.ProcessID, "ai-coder-")

	// Extract action from event data
	action, ok := event.Data["action"].(string)
	if !ok {
		return
	}

	// Execute action through AI coder manager
	var err error
	switch action {
	case "stop":
		err = bridge.aiCoderMgr.StopCoder(coderID)
	case "pause":
		err = bridge.aiCoderMgr.PauseCoder(coderID)
	case "resume":
		err = bridge.aiCoderMgr.ResumeCoder(coderID)
	case "start":
		err = bridge.aiCoderMgr.StartCoder(coderID)
	default:
		// Unknown action, ignore
		return
	}

	// Emit result event
	if err != nil {
		bridge.eventBus.Publish(events.Event{
			Type:      "process.control.error",
			ProcessID: event.ProcessID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"action": action,
				"error":  err.Error(),
			},
		})
	} else {
		bridge.eventBus.Publish(events.Event{
			Type:      "process.control.success",
			ProcessID: event.ProcessID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"action": action,
			},
		})
	}
}

// handleAICoderEvent handles AI coder events and translates to process events
func (bridge *AICoderEventBridge) handleAICoderEvent(event events.Event) {
	// Extract AI coder event data from standard event structure
	coderID, hasCoderID := event.Data["coder_id"].(string)
	if !hasCoderID {
		return // Not an AI coder event
	}

	processID := fmt.Sprintf("ai-coder-%s", coderID)

	var eventType events.EventType
	switch event.Type {
	case events.EventAICoderCreated:
		eventType = events.ProcessStarted
	case events.EventAICoderStarted:
		eventType = events.ProcessStarted
	case events.EventAICoderCompleted:
		eventType = events.ProcessExited
	case events.EventAICoderFailed:
		eventType = events.ProcessExited
	case events.EventAICoderStopped:
		eventType = events.ProcessExited
	case events.EventAICoderDeleted:
		eventType = events.ProcessExited
	default:
		// For other events like progress, just pass through
		return
	}

	// Create process event with AI coder context
	processEventData := make(map[string]interface{})
	for k, v := range event.Data {
		processEventData[k] = v
	}
	processEventData["process_type"] = "ai-coder"

	bridge.eventBus.Publish(events.Event{
		Type:      eventType,
		ProcessID: processID,
		Timestamp: time.Now(),
		Data:      processEventData,
	})
}

// emitAICoderProcessEvent emits a process event for an AI coder
func (bridge *AICoderEventBridge) emitAICoderProcessEvent(eventType events.EventType, coderID string, data map[string]interface{}) {
	processID := fmt.Sprintf("ai-coder-%s", coderID)

	// Add AI coder type marker
	if data == nil {
		data = make(map[string]interface{})
	}
	data["process_type"] = "ai-coder"

	bridge.eventBus.Publish(events.Event{
		Type:      eventType,
		ProcessID: processID,
		Timestamp: time.Now(),
		Data:      data,
	})
}

// EmitAICoderStarted emits event when AI coder starts
func (bridge *AICoderEventBridge) EmitAICoderStarted(coderID, name string) {
	bridge.emitAICoderProcessEvent(events.ProcessStarted, coderID, map[string]interface{}{
		"name": fmt.Sprintf("AI Coder: %s", name),
	})
}

// EmitAICoderStopped emits event when AI coder stops
func (bridge *AICoderEventBridge) EmitAICoderStopped(coderID, name string, exitCode int) {
	bridge.emitAICoderProcessEvent(events.ProcessExited, coderID, map[string]interface{}{
		"name":      fmt.Sprintf("AI Coder: %s", name),
		"exit_code": exitCode,
	})
}

// EmitAICoderStatusChanged emits event when AI coder status changes
func (bridge *AICoderEventBridge) EmitAICoderStatusChanged(coderID, name, status string) {
	bridge.emitAICoderProcessEvent("process.status.changed", coderID, map[string]interface{}{
		"name":   fmt.Sprintf("AI Coder: %s", name),
		"status": status,
	})
}
