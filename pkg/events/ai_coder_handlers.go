package events

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// AI Coder Event Aggregator
type AICoderEventAggregator struct {
	events    []AICoderEvent
	mu        sync.RWMutex
	maxEvents int
	eventBus  *EventBus
	
	// Event statistics
	stats   AICoderEventStats
	statsMu sync.RWMutex
}

type AICoderEventStats struct {
	TotalEvents     int64            `json:"total_events"`
	EventsByType    map[string]int64 `json:"events_by_type"`
	EventsByCoder   map[string]int64 `json:"events_by_coder"`
	LastEvent       time.Time        `json:"last_event"`
	EventsPerMinute float64          `json:"events_per_minute"`
}

func NewAICoderEventAggregator(eventBus *EventBus, maxEvents int) *AICoderEventAggregator {
	aggregator := &AICoderEventAggregator{
		events:    make([]AICoderEvent, 0, maxEvents),
		maxEvents: maxEvents,
		eventBus:  eventBus,
		stats: AICoderEventStats{
			EventsByType:  make(map[string]int64),
			EventsByCoder: make(map[string]int64),
		},
	}
	
	// Register handlers for all AI coder event types
	aggregator.registerHandlers()
	
	return aggregator
}

// Register event handlers
func (a *AICoderEventAggregator) registerHandlers() {
	eventTypes := []EventType{
		EventAICoderCreated, EventAICoderStarted, EventAICoderPaused,
		EventAICoderResumed, EventAICoderCompleted, EventAICoderFailed,
		EventAICoderStopped, EventAICoderDeleted, EventAICoderProgress,
		EventAICoderMilestone, EventAICoderOutput, EventAICoderFileCreated,
		EventAICoderFileModified, EventAICoderFileDeleted, EventAICoderWorkspaceSync,
		EventAICoderAPICall, EventAICoderAPIError, EventAICoderRateLimit,
		EventAICoderResourceUsage, EventAICoderResourceLimit,
	}
	
	for _, eventType := range eventTypes {
		a.eventBus.Subscribe(eventType, a.handleAICoderEvent)
	}
}

// Handle AI coder events
func (a *AICoderEventAggregator) handleAICoderEvent(event Event) {
	// Convert Event to AICoderEvent
	aiEvent := a.eventToAICoderEvent(event)
	if aiEvent.CoderID == "" {
		log.Printf("Warning: AI coder event missing coder ID")
		return
	}
	
	// Add to event history
	a.addEvent(aiEvent)
	
	// Update statistics
	a.updateStats(aiEvent)
	
	// Handle specialized processing
	a.processSpecializedEvent(aiEvent)
}

// Convert Event to AICoderEvent
func (a *AICoderEventAggregator) eventToAICoderEvent(event Event) AICoderEvent {
	aiEvent := AICoderEvent{
		Type:      string(event.Type),
		Timestamp: event.Timestamp,
		Data:      event.Data,
	}
	
	// Extract coder ID and name from event data
	if coderID, ok := event.Data["coder_id"].(string); ok {
		aiEvent.CoderID = coderID
	}
	
	if coderName, ok := event.Data["coder_name"].(string); ok {
		aiEvent.CoderName = coderName
	} else if name, ok := event.Data["name"].(string); ok {
		aiEvent.CoderName = name
	}
	
	return aiEvent
}

// Add event to history with size limit
func (a *AICoderEventAggregator) addEvent(event AICoderEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	// Add new event
	a.events = append(a.events, event)
	
	// Trim if over limit
	if len(a.events) > a.maxEvents {
		// Remove oldest events
		a.events = a.events[len(a.events)-a.maxEvents:]
	}
}

// Update event statistics
func (a *AICoderEventAggregator) updateStats(event AICoderEvent) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	
	a.stats.TotalEvents++
	a.stats.EventsByType[event.Type]++
	a.stats.EventsByCoder[event.CoderID]++
	a.stats.LastEvent = event.Timestamp
	
	// Calculate events per minute (simple moving average)
	if a.stats.TotalEvents > 1 {
		duration := event.Timestamp.Sub(time.Now().Add(-time.Minute))
		if duration > 0 {
			a.stats.EventsPerMinute = float64(a.stats.TotalEvents) / duration.Minutes()
		}
	}
}

// Process specialized event handling
func (a *AICoderEventAggregator) processSpecializedEvent(event AICoderEvent) {
	switch EventType(event.Type) {
	case EventAICoderFailed:
		a.handleFailureEvent(event)
	case EventAICoderCompleted:
		a.handleCompletionEvent(event)
	case EventAICoderRateLimit:
		a.handleRateLimitEvent(event)
	case EventAICoderResourceLimit:
		a.handleResourceLimitEvent(event)
	}
}

// Handle failure events
func (a *AICoderEventAggregator) handleFailureEvent(event AICoderEvent) {
	// Log failure for debugging
	log.Printf("AI Coder %s failed: %v", event.CoderID, event.Data)
	
	// Emit aggregated failure alert if multiple failures
	failureCount := a.getRecentFailureCount(event.CoderID)
	if failureCount >= 3 {
		a.eventBus.Publish(Event{
			Type:      "ai_coder_failure_alert",
			ProcessID: fmt.Sprintf("ai-coder-%s", event.CoderID),
			Timestamp: event.Timestamp,
			Data: map[string]interface{}{
				"coder_id":      event.CoderID,
				"failure_count": failureCount,
				"time":          event.Timestamp,
			},
		})
	}
}

// Handle completion events
func (a *AICoderEventAggregator) handleCompletionEvent(event AICoderEvent) {
	// Calculate completion time from creation
	creationTime := a.getCreationTime(event.CoderID)
	if !creationTime.IsZero() {
		duration := event.Timestamp.Sub(creationTime)
		
		// Emit completion metrics
		a.eventBus.Publish(Event{
			Type:      "ai_coder_completion_metrics",
			ProcessID: fmt.Sprintf("ai-coder-%s", event.CoderID),
			Timestamp: event.Timestamp,
			Data: map[string]interface{}{
				"coder_id": event.CoderID,
				"duration": duration,
				"time":     event.Timestamp,
			},
		})
	}
}

// Handle rate limit events
func (a *AICoderEventAggregator) handleRateLimitEvent(event AICoderEvent) {
	// Emit system-wide rate limit warning
	a.eventBus.Publish(Event{
		Type:      "ai_coder_system_warning",
		ProcessID: fmt.Sprintf("ai-coder-%s", event.CoderID),
		Timestamp: event.Timestamp,
		Data: map[string]interface{}{
			"type":     "rate_limit",
			"coder_id": event.CoderID,
			"message":  "AI provider rate limit reached",
			"time":     event.Timestamp,
		},
	})
}

// Handle resource limit events
func (a *AICoderEventAggregator) handleResourceLimitEvent(event AICoderEvent) {
	// Emit resource warning
	a.eventBus.Publish(Event{
		Type:      "ai_coder_system_warning",
		ProcessID: fmt.Sprintf("ai-coder-%s", event.CoderID),
		Timestamp: event.Timestamp,
		Data: map[string]interface{}{
			"type":     "resource_limit",
			"coder_id": event.CoderID,
			"message":  "AI coder resource limit exceeded",
			"time":     event.Timestamp,
		},
	})
}

// Query methods for event history
func (a *AICoderEventAggregator) GetEvents(filter AICoderEventFilter) []AICoderEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	var filtered []AICoderEvent
	for _, event := range a.events {
		if filter.matches(event) {
			filtered = append(filtered, event)
		}
	}
	
	return filtered
}

func (a *AICoderEventAggregator) GetStats() AICoderEventStats {
	a.statsMu.RLock()
	defer a.statsMu.RUnlock()
	
	return a.stats
}

// Event Filter
type AICoderEventFilter struct {
	CoderID   string
	EventType string
	Since     time.Time
	Until     time.Time
}

func (f AICoderEventFilter) matches(event AICoderEvent) bool {
	if f.CoderID != "" && event.CoderID != f.CoderID {
		return false
	}
	
	if f.EventType != "" && event.Type != f.EventType {
		return false
	}
	
	if !f.Since.IsZero() && event.Timestamp.Before(f.Since) {
		return false
	}
	
	if !f.Until.IsZero() && event.Timestamp.After(f.Until) {
		return false
	}
	
	return true
}

// Helper methods
func (a *AICoderEventAggregator) getRecentFailureCount(coderID string) int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	count := 0
	cutoff := time.Now().Add(-10 * time.Minute) // Recent = last 10 minutes
	
	for _, event := range a.events {
		if event.CoderID == coderID && 
		   event.Type == string(EventAICoderFailed) && 
		   event.Timestamp.After(cutoff) {
			count++
		}
	}
	
	return count
}

func (a *AICoderEventAggregator) getCreationTime(coderID string) time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	for _, event := range a.events {
		if event.CoderID == coderID && event.Type == string(EventAICoderCreated) {
			return event.Timestamp
		}
	}
	
	return time.Time{}
}