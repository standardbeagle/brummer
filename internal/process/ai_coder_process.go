package process

import (
	"fmt"
	"sync"

	"github.com/standardbeagle/brummer/internal/aicoder"
)

// AICoderProcess wraps AI coder instances as processes
type AICoderProcess struct {
	*Process // Embed standard process
	aiCoder  *aicoder.AICoderProcess
	mu       sync.RWMutex
}

// NewAICoderProcess creates a new AI coder process wrapper
func NewAICoderProcess(coder *aicoder.AICoderProcess) *AICoderProcess {
	process := &Process{
		ID:        fmt.Sprintf("ai-coder-%s", coder.ID),
		Name:      fmt.Sprintf("AI Coder: %s", coder.Name),
		Script:    coder.Task,
		Status:    mapAICoderStatus(coder.Status),
		StartTime: coder.CreatedAt,
		// Cmd is nil for AI coder processes since they're not OS processes
		Cmd:    nil,
		cancel: nil, // AI coders use their own lifecycle management
	}

	return &AICoderProcess{
		Process: process,
		aiCoder: coder,
	}
}

// mapAICoderStatus maps AI coder status to process status
func mapAICoderStatus(status aicoder.AICoderStatus) ProcessStatus {
	switch status {
	case aicoder.StatusCreating:
		return StatusPending
	case aicoder.StatusRunning:
		return StatusRunning
	case aicoder.StatusPaused:
		return StatusStopped
	case aicoder.StatusCompleted:
		return StatusSuccess
	case aicoder.StatusFailed:
		return StatusFailed
	case aicoder.StatusStopped:
		return StatusStopped
	default:
		return StatusPending
	}
}

// updateFromAICoder updates process state from AI coder state
func (acp *AICoderProcess) updateFromAICoder(coder *aicoder.AICoderProcess) {
	acp.mu.Lock()
	defer acp.mu.Unlock()

	oldStatus := acp.Status
	newStatus := mapAICoderStatus(coder.Status)

	acp.Status = newStatus
	acp.aiCoder = coder

	// Update process-specific fields that don't exist in base Process
	// Progress tracking will be handled separately as it's not part of base Process

	// Status change detection for event emission (handled by manager)
	_ = oldStatus // Available for status change detection
}

// GetAICoderInfo returns AI coder specific information
func (acp *AICoderProcess) GetAICoderInfo() map[string]interface{} {
	acp.mu.RLock()
	defer acp.mu.RUnlock()

	if acp.aiCoder == nil {
		return nil
	}

	return map[string]interface{}{
		"provider":   acp.aiCoder.Provider,
		"workspace":  acp.aiCoder.WorkspaceDir,
		"task":       acp.aiCoder.Task,
		"progress":   acp.aiCoder.Progress,
		"created_at": acp.aiCoder.CreatedAt,
		"updated_at": acp.aiCoder.UpdatedAt,
	}
}

// GetProgress returns the AI coder's progress
func (acp *AICoderProcess) GetProgress() float64 {
	acp.mu.RLock()
	defer acp.mu.RUnlock()

	if acp.aiCoder == nil {
		return 0.0
	}

	return acp.aiCoder.Progress
}

// GetProvider returns the AI provider name
func (acp *AICoderProcess) GetProvider() string {
	acp.mu.RLock()
	defer acp.mu.RUnlock()

	if acp.aiCoder == nil {
		return ""
	}

	return acp.aiCoder.Provider
}

// GetWorkspaceDir returns the AI coder's workspace directory
func (acp *AICoderProcess) GetWorkspaceDir() string {
	acp.mu.RLock()
	defer acp.mu.RUnlock()

	if acp.aiCoder == nil {
		return ""
	}

	return acp.aiCoder.WorkspaceDir
}

// GetTask returns the AI coder's task description
func (acp *AICoderProcess) GetTask() string {
	acp.mu.RLock()
	defer acp.mu.RUnlock()

	if acp.aiCoder == nil {
		return ""
	}

	return acp.aiCoder.Task
}

// IsAICoder returns true indicating this is an AI coder process
func (acp *AICoderProcess) IsAICoder() bool {
	return true
}

// Control operations that delegate to AI coder manager
// These methods return errors directing users to use the AI coder manager directly

func (acp *AICoderProcess) Stop() error {
	return fmt.Errorf("AI coder stop operations must go through AI coder manager - use aiCoderManager.StopCoder()")
}

func (acp *AICoderProcess) Pause() error {
	return fmt.Errorf("AI coder pause operations must go through AI coder manager - use aiCoderManager.PauseCoder()")
}

func (acp *AICoderProcess) Resume() error {
	return fmt.Errorf("AI coder resume operations must go through AI coder manager - use aiCoderManager.ResumeCoder()")
}

// Kill is not applicable for AI coder processes
func (acp *AICoderProcess) Kill() error {
	return fmt.Errorf("AI coder processes cannot be killed - use Stop() instead")
}
