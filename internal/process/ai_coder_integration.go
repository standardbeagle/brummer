package process

import (
	"fmt"
	"strings"
	"time"

	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/pkg/events"
)

// AICoderIntegration manages integration between AI coder service and process manager
type AICoderIntegration struct {
	processMgr  *Manager
	aiCoderMgr  *aicoder.AICoderManager
	eventBridge *AICoderEventBridge
	eventBus    *events.EventBus
}

// NewAICoderIntegration creates a new AI coder integration
func NewAICoderIntegration(processMgr *Manager, eventBus *events.EventBus) *AICoderIntegration {
	return &AICoderIntegration{
		processMgr: processMgr,
		eventBus:   eventBus,
	}
}

// Initialize sets up integration with AI coder manager
func (integration *AICoderIntegration) Initialize(aiCoderMgr *aicoder.AICoderManager) error {
	integration.aiCoderMgr = aiCoderMgr

	// Set up AI coder manager in process manager
	integration.processMgr.SetAICoderManager(aiCoderMgr)

	// Create and start event bridge
	integration.eventBridge = NewAICoderEventBridge(
		integration.processMgr,
		aiCoderMgr,
		integration.eventBus,
	)
	integration.eventBridge.Start()

	return nil
}

// GetAICoderProcesses returns AI coder processes for display
func (integration *AICoderIntegration) GetAICoderProcesses() []*AICoderProcess {
	if integration.aiCoderMgr == nil {
		return nil
	}

	coders := integration.aiCoderMgr.ListCoders()
	processes := make([]*AICoderProcess, len(coders))

	for i, coder := range coders {
		processes[i] = NewAICoderProcess(coder)
	}

	return processes
}

// ControlAICoder controls AI coder through process interface
func (integration *AICoderIntegration) ControlAICoder(coderID, action string) error {
	if integration.aiCoderMgr == nil {
		return fmt.Errorf("AI coder manager not initialized")
	}

	switch action {
	case "start":
		return integration.aiCoderMgr.StartCoder(coderID)
	case "stop":
		return integration.aiCoderMgr.StopCoder(coderID)
	case "pause":
		return integration.aiCoderMgr.PauseCoder(coderID)
	case "resume":
		return integration.aiCoderMgr.ResumeCoder(coderID)
	default:
		return fmt.Errorf("unsupported action: %s", action)
	}
}

// AICoderProcessStatus represents AI coder process status for display
type AICoderProcessStatus struct {
	ID       string
	Name     string
	Status   ProcessStatus
	Progress float64
	Runtime  time.Duration
	Extra    map[string]interface{}
}

// GetProcessStatus gets AI coder process status for display
func (integration *AICoderIntegration) GetProcessStatus(processID string) (*AICoderProcessStatus, error) {
	if !strings.HasPrefix(processID, "ai-coder-") {
		return nil, fmt.Errorf("not an AI coder process")
	}

	coderID := strings.TrimPrefix(processID, "ai-coder-")

	if integration.aiCoderMgr == nil {
		return nil, fmt.Errorf("AI coder manager not initialized")
	}

	coder, exists := integration.aiCoderMgr.GetCoder(coderID)
	if !exists {
		return nil, fmt.Errorf("AI coder not found")
	}

	return &AICoderProcessStatus{
		ID:       processID,
		Name:     fmt.Sprintf("AI Coder: %s", coder.Name),
		Status:   mapAICoderStatus(coder.Status),
		Progress: coder.Progress,
		Runtime:  time.Since(coder.CreatedAt),
		Extra: map[string]interface{}{
			"provider":  coder.Provider,
			"workspace": coder.WorkspaceDir,
			"task":      coder.Task,
		},
	}, nil
}
