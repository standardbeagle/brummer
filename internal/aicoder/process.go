package aicoder

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProcessManager handles AI coder process lifecycle operations
type ProcessManager struct {
	manager      *AICoderManager
	workspaceMgr *WorkspaceManager
	providerReg  *ProviderRegistry
	activeCoders map[string]*processContext
	mu           sync.RWMutex
}

// processContext holds the context for a running AI coder
type processContext struct {
	coder    *AICoderProcess
	provider AIProvider
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewProcessManager creates a new process manager
func NewProcessManager(manager *AICoderManager, workspaceMgr *WorkspaceManager, providerReg *ProviderRegistry) *ProcessManager {
	return &ProcessManager{
		manager:      manager,
		workspaceMgr: workspaceMgr,
		providerReg:  providerReg,
		activeCoders: make(map[string]*processContext),
	}
}

// StartCoder starts an AI coder process
func (pm *ProcessManager) StartCoder(coder *AICoderProcess) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if already running
	if _, exists := pm.activeCoders[coder.ID]; exists {
		return fmt.Errorf("coder %s is already running", coder.ID)
	}

	// Get provider
	provider, err := pm.providerReg.Get(coder.Provider)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	// Create context for the process
	ctx, cancel := context.WithCancel(context.Background())

	// Store process context
	pm.activeCoders[coder.ID] = &processContext{
		coder:    coder,
		provider: provider,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Update coder status
	coder.SetStatus(StatusRunning)

	// Start the AI coder process in a goroutine
	go pm.runCoder(coder.ID)

	return nil
}

// StopCoder stops an AI coder process
func (pm *ProcessManager) StopCoder(coderID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	procCtx, exists := pm.activeCoders[coderID]
	if !exists {
		return fmt.Errorf("coder %s is not running", coderID)
	}

	// Cancel the context to stop the process
	procCtx.cancel()

	// Update status
	procCtx.coder.SetStatus(StatusStopped)

	// Remove from active coders
	delete(pm.activeCoders, coderID)

	return nil
}

// PauseCoder pauses an AI coder process
func (pm *ProcessManager) PauseCoder(coderID string) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	procCtx, exists := pm.activeCoders[coderID]
	if !exists {
		return fmt.Errorf("coder %s is not running", coderID)
	}

	// Update status to paused
	procCtx.coder.SetStatus(StatusPaused)

	// Note: Actual pause logic would depend on the provider implementation
	// For now, we just update the status

	return nil
}

// ResumeCoder resumes a paused AI coder process
func (pm *ProcessManager) ResumeCoder(coderID string) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	procCtx, exists := pm.activeCoders[coderID]
	if !exists {
		return fmt.Errorf("coder %s is not running", coderID)
	}

	if procCtx.coder.GetStatus() != StatusPaused {
		return fmt.Errorf("coder %s is not paused", coderID)
	}

	// Update status to running
	procCtx.coder.SetStatus(StatusRunning)

	return nil
}

// runCoder is the main loop for an AI coder process
func (pm *ProcessManager) runCoder(coderID string) {
	pm.mu.RLock()
	procCtx, exists := pm.activeCoders[coderID]
	pm.mu.RUnlock()

	if !exists {
		return
	}

	defer func() {
		// Cleanup on exit
		pm.mu.Lock()
		delete(pm.activeCoders, coderID)
		pm.mu.Unlock()

		// Update final status if not already set
		if status := procCtx.coder.GetStatus(); status == StatusRunning || status == StatusPaused {
			procCtx.coder.SetStatus(StatusCompleted)
		}
	}()

	// Try to use the provider to validate it works
	// This tests provider functionality early and handles failures gracefully
	generateOptions := GenerateOptions{
		Model:       "test-model",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	// Test the provider with a simple request
	_, err := procCtx.provider.GenerateCode(procCtx.ctx, "test prompt", generateOptions)
	if err != nil {
		// Provider failed, mark as failed and exit
		procCtx.coder.SetStatus(StatusFailed)
		return
	}

	// Provider works, continue with simulation
	for i := 0; i <= 100; i += 10 {
		select {
		case <-procCtx.ctx.Done():
			// Context cancelled, stop processing
			return
		case <-time.After(time.Second):
			// Check if paused
			if procCtx.coder.GetStatus() == StatusPaused {
				// Wait while paused
				for procCtx.coder.GetStatus() == StatusPaused {
					select {
					case <-procCtx.ctx.Done():
						return
					case <-time.After(100 * time.Millisecond):
						// Check status periodically
					}
				}
			}

			// Update progress
			progress := float64(i) / 100.0
			procCtx.coder.UpdateProgress(progress, fmt.Sprintf("Processing... %d%%", i))

			// Emit progress event
			if pm.manager.eventBus != nil {
				pm.manager.eventBus.Emit(string(EventAICoderProgress), AICoderEvent{
					Type:      string(EventAICoderProgress),
					CoderID:   coderID,
					CoderName: procCtx.coder.Name,
					Status:    string(procCtx.coder.GetStatus()),
					Message:   fmt.Sprintf("Progress: %d%%", i),
					Time:      time.Now(),
					Data: map[string]interface{}{
						"progress": progress,
					},
				})
			}
		}
	}

	// Mark as completed
	procCtx.coder.SetStatus(StatusCompleted)
}

// GetActiveCount returns the number of active AI coders
func (pm *ProcessManager) GetActiveCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.activeCoders)
}

// IsActive checks if a coder is currently active
func (pm *ProcessManager) IsActive(coderID string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	_, exists := pm.activeCoders[coderID]
	return exists
}

// GetActiveCoders returns a list of active coder IDs
func (pm *ProcessManager) GetActiveCoders() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	ids := make([]string, 0, len(pm.activeCoders))
	for id := range pm.activeCoders {
		ids = append(ids, id)
	}
	return ids
}
