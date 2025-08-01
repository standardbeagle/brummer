package events

import (
	"time"
)

// AI Coder Event Types
const (
	// Lifecycle events
	EventAICoderCreated   EventType = "ai_coder_created"
	EventAICoderStarted   EventType = "ai_coder_started"
	EventAICoderPaused    EventType = "ai_coder_paused"
	EventAICoderResumed   EventType = "ai_coder_resumed"
	EventAICoderCompleted EventType = "ai_coder_completed"
	EventAICoderFailed    EventType = "ai_coder_failed"
	EventAICoderStopped   EventType = "ai_coder_stopped"
	EventAICoderDeleted   EventType = "ai_coder_deleted"

	// Progress events
	EventAICoderProgress  EventType = "ai_coder_progress"
	EventAICoderMilestone EventType = "ai_coder_milestone"
	EventAICoderOutput    EventType = "ai_coder_output"

	// Workspace events
	EventAICoderFileCreated   EventType = "ai_coder_file_created"
	EventAICoderFileModified  EventType = "ai_coder_file_modified"
	EventAICoderFileDeleted   EventType = "ai_coder_file_deleted"
	EventAICoderWorkspaceSync EventType = "ai_coder_workspace_sync"

	// Provider events
	EventAICoderAPICall   EventType = "ai_coder_api_call"
	EventAICoderAPIError  EventType = "ai_coder_api_error"
	EventAICoderRateLimit EventType = "ai_coder_rate_limit"

	// Resource events
	EventAICoderResourceUsage EventType = "ai_coder_resource_usage"
	EventAICoderResourceLimit EventType = "ai_coder_resource_limit"
)

// Core AI Coder Event Structure
type AICoderEvent struct {
	Type      string                 `json:"type"`
	CoderID   string                 `json:"coder_id"`
	CoderName string                 `json:"coder_name"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Lifecycle Event Data
type AICoderLifecycleEvent struct {
	AICoderEvent
	PreviousStatus string `json:"previous_status"`
	CurrentStatus  string `json:"current_status"`
	Reason         string `json:"reason,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// Progress Event Data
type AICoderProgressEvent struct {
	AICoderEvent
	Progress    float64 `json:"progress"`
	Stage       string  `json:"stage"`
	Description string  `json:"description"`
	Milestone   string  `json:"milestone,omitempty"`
}

// Workspace Event Data
type AICoderWorkspaceEvent struct {
	AICoderEvent
	Operation   string `json:"operation"`
	FilePath    string `json:"file_path"`
	FileSize    int64  `json:"file_size,omitempty"`
	ContentHash string `json:"content_hash,omitempty"`
}

// Provider Event Data
type AICoderProviderEvent struct {
	AICoderEvent
	Provider     string        `json:"provider"`
	Model        string        `json:"model"`
	TokensUsed   int           `json:"tokens_used,omitempty"`
	Duration     time.Duration `json:"duration,omitempty"`
	ErrorCode    string        `json:"error_code,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// Resource Event Data
type AICoderResourceEvent struct {
	AICoderEvent
	MemoryMB     int64   `json:"memory_mb"`
	CPUPercent   float64 `json:"cpu_percent"`
	DiskUsageMB  int64   `json:"disk_usage_mb"`
	NetworkBytes int64   `json:"network_bytes"`
	FileCount    int     `json:"file_count"`
}

// Event Factory Functions
func NewAICoderLifecycleEvent(coderID, coderName, eventType, prevStatus, currStatus, reason string) *AICoderLifecycleEvent {
	return &AICoderLifecycleEvent{
		AICoderEvent: AICoderEvent{
			Type:      eventType,
			CoderID:   coderID,
			CoderName: coderName,
			Timestamp: time.Now(),
			Data:      make(map[string]interface{}),
		},
		PreviousStatus: prevStatus,
		CurrentStatus:  currStatus,
		Reason:         reason,
	}
}

func NewAICoderProgressEvent(coderID, coderName string, progress float64, stage, description string) *AICoderProgressEvent {
	return &AICoderProgressEvent{
		AICoderEvent: AICoderEvent{
			Type:      string(EventAICoderProgress),
			CoderID:   coderID,
			CoderName: coderName,
			Timestamp: time.Now(),
			Data:      make(map[string]interface{}),
		},
		Progress:    progress,
		Stage:       stage,
		Description: description,
	}
}

func NewAICoderWorkspaceEvent(coderID, coderName, operation, filePath string) *AICoderWorkspaceEvent {
	return &AICoderWorkspaceEvent{
		AICoderEvent: AICoderEvent{
			Type:      string(getWorkspaceEventType(operation)),
			CoderID:   coderID,
			CoderName: coderName,
			Timestamp: time.Now(),
			Data:      make(map[string]interface{}),
		},
		Operation: operation,
		FilePath:  filePath,
	}
}

func NewAICoderProviderEvent(coderID, coderName, provider, model string) *AICoderProviderEvent {
	return &AICoderProviderEvent{
		AICoderEvent: AICoderEvent{
			Type:      string(EventAICoderAPICall),
			CoderID:   coderID,
			CoderName: coderName,
			Timestamp: time.Now(),
			Data:      make(map[string]interface{}),
		},
		Provider: provider,
		Model:    model,
	}
}

func NewAICoderResourceEvent(coderID, coderName string, memMB int64, cpuPercent float64, diskMB int64) *AICoderResourceEvent {
	return &AICoderResourceEvent{
		AICoderEvent: AICoderEvent{
			Type:      string(EventAICoderResourceUsage),
			CoderID:   coderID,
			CoderName: coderName,
			Timestamp: time.Now(),
			Data:      make(map[string]interface{}),
		},
		MemoryMB:    memMB,
		CPUPercent:  cpuPercent,
		DiskUsageMB: diskMB,
	}
}

// Helper functions
func getWorkspaceEventType(operation string) EventType {
	switch operation {
	case "create":
		return EventAICoderFileCreated
	case "modify":
		return EventAICoderFileModified
	case "delete":
		return EventAICoderFileDeleted
	default:
		return EventAICoderWorkspaceSync
	}
}
