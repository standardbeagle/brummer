package aicoder

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AICoderStatus represents the current state of an AI coder
type AICoderStatus string

const (
	StatusCreating  AICoderStatus = "creating"
	StatusRunning   AICoderStatus = "running"
	StatusPaused    AICoderStatus = "paused"
	StatusCompleted AICoderStatus = "completed"
	StatusFailed    AICoderStatus = "failed"
	StatusStopped   AICoderStatus = "stopped"
)

// AICoderProcess represents a single AI coder instance
type AICoderProcess struct {
	ID               string
	Name             string
	Provider         string
	WorkspaceDir     string
	Status           AICoderStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Task             string
	Progress         float64
	CurrentMessage   string
	SessionID        string // For tmux-style session management
	AttachedSessions int    // Number of attached UI sessions
	cancel           context.CancelFunc
	workspaceMgr     *WorkspaceManager // Reference to workspace manager
	mu               sync.RWMutex
}

// Thread-safe getters
func (p *AICoderProcess) GetStatus() AICoderStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Status
}

func (p *AICoderProcess) GetProgress() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Progress
}

// Thread-safe setters
func (p *AICoderProcess) SetStatus(status AICoderStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Status = status
	p.UpdatedAt = time.Now()
}

func (p *AICoderProcess) UpdateProgress(progress float64, message string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if progress < 0 || progress > 1 {
		return fmt.Errorf("progress must be between 0 and 1, got %f", progress)
	}

	p.Progress = progress
	p.CurrentMessage = message
	p.UpdatedAt = time.Now()
	return nil
}

// CreateCoderRequest represents a request to create a new AI coder
type CreateCoderRequest struct {
	Name           string
	Provider       string
	Task           string
	WorkspaceFiles []string
	Context        map[string]interface{}
}

// AICoderError represents an AI coder specific error
type AICoderError struct {
	CoderID string
	Op      string
	Err     error
}

func (e *AICoderError) Error() string {
	return fmt.Sprintf("ai coder %s: %s: %v", e.CoderID, e.Op, e.Err)
}

// Configuration interface for dependency injection
type Config interface {
	GetAICoderConfig() AICoderConfig
	GetProviderConfigs() map[string]*ProviderConfig
}

// AICoderConfig is a simplified version for the aicoder package
// The full configuration is in the config package
type AICoderConfig struct {
	MaxConcurrent    int
	WorkspaceBaseDir string
	DefaultProvider  string
	TimeoutMinutes   int
}

// ProviderConfig is a simplified provider configuration
type ProviderConfig struct {
	Model       string
	APIKeyEnv   string
	MaxTokens   int
	Temperature float64

	// CLI Tool specific configuration
	CLITool *CLIToolConfig
}

// CLIToolConfig represents configuration for CLI-based AI tools
type CLIToolConfig struct {
	Command     string
	BaseArgs    []string
	FlagMapping map[string]string
	WorkingDir  string
	Environment map[string]string
}

// Event bus interface for dependency injection
type EventBus interface {
	Emit(eventType string, data interface{})
}

// Event types for AI coder events
type AICoderEventType string

const (
	EventAICoderCreated   AICoderEventType = "ai_coder_created"
	EventAICoderStarted   AICoderEventType = "ai_coder_started"
	EventAICoderPaused    AICoderEventType = "ai_coder_paused"
	EventAICoderResumed   AICoderEventType = "ai_coder_resumed"
	EventAICoderCompleted AICoderEventType = "ai_coder_completed"
	EventAICoderFailed    AICoderEventType = "ai_coder_failed"
	EventAICoderStopped   AICoderEventType = "ai_coder_stopped"
	EventAICoderDeleted   AICoderEventType = "ai_coder_deleted"
	EventAICoderProgress  AICoderEventType = "ai_coder_progress"
)

// AICoderEvent represents an event emitted by the AI coder system
type AICoderEvent struct {
	Type      string                 `json:"type"`
	CoderID   string                 `json:"coder_id"`
	CoderName string                 `json:"coder_name"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Time      time.Time              `json:"time"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Workspace file operations
func (p *AICoderProcess) WriteFile(filename string, content []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.workspaceMgr == nil {
		return fmt.Errorf("workspace manager not initialized")
	}

	p.UpdatedAt = time.Now()
	return p.workspaceMgr.WriteFile(p.WorkspaceDir, filename, content)
}

func (p *AICoderProcess) ReadFile(filename string) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.workspaceMgr == nil {
		return nil, fmt.Errorf("workspace manager not initialized")
	}

	return p.workspaceMgr.ReadFile(p.WorkspaceDir, filename)
}

func (p *AICoderProcess) ListWorkspaceFiles() ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.workspaceMgr == nil {
		return nil, fmt.Errorf("workspace manager not initialized")
	}

	return p.workspaceMgr.ListFiles(p.WorkspaceDir)
}

// ReadWorkspaceFile reads a file from the AI coder's workspace
func (p *AICoderProcess) ReadWorkspaceFile(filePath string) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// This would be implemented by the workspace manager
	// For now, return a mock implementation
	content := fmt.Sprintf("// File: %s\n// AI Coder: %s\n// Task: %s\n", filePath, p.ID, p.Task)
	return []byte(content), nil
}
