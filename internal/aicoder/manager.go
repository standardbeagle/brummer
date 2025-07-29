package aicoder

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AICoderManager manages AI coder instances
type AICoderManager struct {
	coders         map[string]*AICoderProcess
	mu             sync.RWMutex
	eventBus       EventBus
	config         Config
	workspaceMgr   *WorkspaceManager
	providerReg    *ProviderRegistry
	processMgr     *ProcessManager
	
	// PTY support for interactive sessions
	ptyManager     *PTYManager
}

// NewAICoderManager creates a new AI coder manager
func NewAICoderManager(config Config, eventBus EventBus) (*AICoderManager, error) {
	aiConfig := config.GetAICoderConfig()
	
	// Create workspace manager
	workspaceMgr, err := NewWorkspaceManager(aiConfig.WorkspaceBaseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}
	
	// Create provider registry
	providerReg := NewProviderRegistry()
	
	// Register mock provider for testing
	mockProvider := NewMockProvider("mock")
	if err := providerReg.Register("mock", mockProvider); err != nil {
		return nil, fmt.Errorf("failed to register mock provider: %w", err)
	}
	
	// Register Claude provider
	claudeProvider := NewClaudeProvider("", "")
	if err := providerReg.Register("claude", claudeProvider); err != nil {
		return nil, fmt.Errorf("failed to register claude provider: %w", err)
	}
	
	// Register OpenAI provider
	openaiProvider := NewOpenAIProvider("", "")
	if err := providerReg.Register("openai", openaiProvider); err != nil {
		return nil, fmt.Errorf("failed to register openai provider: %w", err)
	}
	
	// Register Gemini provider
	geminiProvider := NewGeminiProvider("", "")
	if err := providerReg.Register("gemini", geminiProvider); err != nil {
		return nil, fmt.Errorf("failed to register gemini provider: %w", err)
	}
	
	// Register Terminal provider
	terminalProvider := NewTerminalProvider("")
	if err := providerReg.Register("terminal", terminalProvider); err != nil {
		return nil, fmt.Errorf("failed to register terminal provider: %w", err)
	}
	
	// Register CLI tool providers based on configuration
	if err := registerCLIToolProviders(providerReg, config); err != nil {
		return nil, fmt.Errorf("failed to register CLI tool providers: %w", err)
	}
	
	manager := &AICoderManager{
		coders:       make(map[string]*AICoderProcess),
		eventBus:     eventBus,
		config:       config,
		workspaceMgr: workspaceMgr,
		providerReg:  providerReg,
	}
	
	// Create process manager
	manager.processMgr = NewProcessManager(manager, workspaceMgr, providerReg)
	
	return manager, nil
}

// NewAICoderManagerWithPTY creates a new AI coder manager with PTY support
func NewAICoderManagerWithPTY(config Config, eventBus EventBus, dataProvider BrummerDataProvider) (*AICoderManager, error) {
	// Create the standard manager first
	manager, err := NewAICoderManager(config, eventBus)
	if err != nil {
		return nil, err
	}
	
	// Add PTY support
	manager.ptyManager = NewPTYManager(dataProvider, eventBus)
	
	return manager, nil
}

// NewAICoderManagerWithoutMockProvider creates a new AI coder manager without registering the mock provider
// This is useful for testing when you want to control provider registration
func NewAICoderManagerWithoutMockProvider(config Config, eventBus EventBus) (*AICoderManager, error) {
	aiConfig := config.GetAICoderConfig()
	
	// Create workspace manager
	workspaceMgr, err := NewWorkspaceManager(aiConfig.WorkspaceBaseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}
	
	// Create provider registry (no providers registered)
	providerReg := NewProviderRegistry()
	
	manager := &AICoderManager{
		coders:       make(map[string]*AICoderProcess),
		eventBus:     eventBus,
		config:       config,
		workspaceMgr: workspaceMgr,
		providerReg:  providerReg,
	}
	
	// Create process manager
	manager.processMgr = NewProcessManager(manager, workspaceMgr, providerReg)
	
	return manager, nil
}

// CreateCoder creates a new AI coder instance
func (m *AICoderManager) CreateCoder(ctx context.Context, req CreateCoderRequest) (*AICoderProcess, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check max concurrent limit
	aiConfig := m.config.GetAICoderConfig()
	if m.processMgr.GetActiveCount() >= aiConfig.MaxConcurrent {
		return nil, fmt.Errorf("maximum concurrent AI coders limit reached (%d)", aiConfig.MaxConcurrent)
	}
	
	// Generate unique ID
	coderID := uuid.New().String()
	
	// Use default provider if not specified
	provider := req.Provider
	if provider == "" {
		provider = aiConfig.DefaultProvider
		if provider == "" {
			provider = "claude" // Default to Claude provider
		}
	}
	
	// Validate provider exists
	if _, err := m.providerReg.Get(provider); err != nil {
		return nil, fmt.Errorf("invalid provider %s: %w", provider, err)
	}
	
	// Create workspace
	workspaceDir, err := m.workspaceMgr.CreateWorkspace(coderID)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}
	
	// Create AI coder process
	now := time.Now()
	coder := &AICoderProcess{
		ID:           coderID,
		Name:         req.Name,
		Provider:     provider,
		WorkspaceDir: workspaceDir,
		Status:       StatusCreating,
		CreatedAt:    now,
		UpdatedAt:    now,
		Task:         req.Task,
		Progress:     0.0,
		SessionID:    fmt.Sprintf("ai-coder-%s", coderID[:8]), // Short session ID for tmux-style
		workspaceMgr: m.workspaceMgr, // Inject workspace manager
	}
	
	// Store the coder
	m.coders[coderID] = coder
	
	// Emit creation event
	if m.eventBus != nil {
		m.eventBus.Emit(string(EventAICoderCreated), AICoderEvent{
			Type:      string(EventAICoderCreated),
			CoderID:   coderID,
			CoderName: coder.Name,
			Status:    string(coder.Status),
			Message:   fmt.Sprintf("AI coder created: %s", coder.Name),
			Time:      now,
			Data: map[string]interface{}{
				"provider": provider,
				"task":     req.Task,
			},
		})
	}
	
	// Copy initial workspace files if provided
	if len(req.WorkspaceFiles) > 0 {
		for _, file := range req.WorkspaceFiles {
			// In a real implementation, this would copy files from the project
			// to the AI coder's workspace
			content := fmt.Sprintf("// Initial file: %s\n// Task: %s\n", file, req.Task)
			if err := m.workspaceMgr.WriteFile(workspaceDir, file, []byte(content)); err != nil {
				// Log error but don't fail creation
				fmt.Printf("Warning: failed to create initial file %s: %v\n", file, err)
			}
		}
	}
	
	return coder, nil
}

// GetCoder retrieves an AI coder by ID
func (m *AICoderManager) GetCoder(id string) (*AICoderProcess, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	coder, exists := m.coders[id]
	return coder, exists
}

// ListCoders returns all AI coder instances
func (m *AICoderManager) ListCoders() []*AICoderProcess {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	coders := make([]*AICoderProcess, 0, len(m.coders))
	for _, coder := range m.coders {
		coders = append(coders, coder)
	}
	return coders
}

// StartCoder starts an AI coder process
func (m *AICoderManager) StartCoder(coderID string) error {
	coder, exists := m.GetCoder(coderID)
	if !exists {
		return fmt.Errorf("AI coder %s not found", coderID)
	}
	
	// Check if coder can be started
	status := coder.GetStatus()
	if status != StatusCreating && status != StatusStopped && status != StatusPaused {
		return fmt.Errorf("AI coder %s cannot be started from status %s", coderID, status)
	}
	
	// Start the coder process
	if err := m.processMgr.StartCoder(coder); err != nil {
		return fmt.Errorf("failed to start AI coder: %w", err)
	}
	
	// Wait a brief moment to check for immediate failures
	time.Sleep(50 * time.Millisecond)
	if coder.GetStatus() == StatusFailed {
		return fmt.Errorf("AI coder failed to start: provider error")
	}
	
	// Emit start event
	if m.eventBus != nil {
		m.eventBus.Emit(string(EventAICoderStarted), AICoderEvent{
			Type:      string(EventAICoderStarted),
			CoderID:   coderID,
			CoderName: coder.Name,
			Status:    string(StatusRunning),
			Message:   "AI coder started",
			Time:      time.Now(),
		})
	}
	
	return nil
}

// registerCLIToolProviders registers CLI tool providers based on auto-detection
func registerCLIToolProviders(providerReg *ProviderRegistry, config Config) error {
	// Check if aider is available and register it
	if _, err := exec.LookPath("aider"); err == nil {
		aiderConfig := CLIToolConfig{
			Command:  "aider",
			BaseArgs: []string{"--yes"},
			FlagMapping: map[string]string{
				"model":   "--model",
				"message": "--message",
			},
			WorkingDir:  ".",
			Environment: make(map[string]string),
		}
		
		aiderProvider := NewCLIToolProvider("aider", aiderConfig)
		if err := providerReg.Register("aider", aiderProvider); err != nil {
			return fmt.Errorf("failed to register aider provider: %w", err)
		}
	}
	
	// Check if claude CLI is available and register it
	if _, err := exec.LookPath("claude"); err == nil {
		claudeConfig := CLIToolConfig{
			Command:  "claude",
			BaseArgs: []string{"--print"}, // Use --print for non-interactive output
			FlagMapping: map[string]string{
				"model":         "--model",
				"output_format": "--output-format",
				"debug":         "--debug",
			},
			WorkingDir:  ".",
			Environment: make(map[string]string),
		}
		
		claudeProvider := NewCLIToolProvider("claude-cli", claudeConfig)
		if err := providerReg.Register("claude-cli", claudeProvider); err != nil {
			return fmt.Errorf("failed to register claude CLI provider: %w", err)
		}
	}
	
	// Check if opencode CLI is available and register it
	if _, err := exec.LookPath("opencode"); err == nil {
		opencodeConfig := CLIToolConfig{
			Command:  "opencode",
			BaseArgs: []string{"run"}, // Use 'run' subcommand for non-interactive
			FlagMapping: map[string]string{
				"model":  "--model",
				"prompt": "--prompt",
				"debug":  "--log-level",
			},
			WorkingDir:  ".",
			Environment: make(map[string]string),
		}
		
		opencodeProvider := NewCLIToolProvider("opencode-cli", opencodeConfig)
		if err := providerReg.Register("opencode-cli", opencodeProvider); err != nil {
			return fmt.Errorf("failed to register opencode CLI provider: %w", err)
		}
	}
	
	// Check if gemini CLI is available and register it
	if _, err := exec.LookPath("gemini"); err == nil {
		geminiConfig := CLIToolConfig{
			Command:  "gemini",
			BaseArgs: []string{}, // No special base args needed
			FlagMapping: map[string]string{
				"model":  "--model",
				"prompt": "--prompt",
				"debug":  "--debug",
			},
			WorkingDir:  ".",
			Environment: make(map[string]string),
		}
		
		geminiProvider := NewCLIToolProvider("gemini-cli", geminiConfig)
		if err := providerReg.Register("gemini-cli", geminiProvider); err != nil {
			return fmt.Errorf("failed to register gemini CLI provider: %w", err)
		}
	}
	
	return nil
}

// StopCoder stops an AI coder process
func (m *AICoderManager) StopCoder(coderID string) error {
	coder, exists := m.GetCoder(coderID)
	if !exists {
		return fmt.Errorf("AI coder %s not found", coderID)
	}
	
	// Stop the coder process
	if err := m.processMgr.StopCoder(coderID); err != nil {
		return fmt.Errorf("failed to stop AI coder: %w", err)
	}
	
	// Emit stop event
	if m.eventBus != nil {
		m.eventBus.Emit(string(EventAICoderStopped), AICoderEvent{
			Type:      string(EventAICoderStopped),
			CoderID:   coderID,
			CoderName: coder.Name,
			Status:    string(StatusStopped),
			Message:   "AI coder stopped",
			Time:      time.Now(),
		})
	}
	
	return nil
}

// PauseCoder pauses an AI coder process
func (m *AICoderManager) PauseCoder(coderID string) error {
	coder, exists := m.GetCoder(coderID)
	if !exists {
		return fmt.Errorf("AI coder %s not found", coderID)
	}
	
	if err := m.processMgr.PauseCoder(coderID); err != nil {
		return fmt.Errorf("failed to pause AI coder: %w", err)
	}
	
	// Emit pause event
	if m.eventBus != nil {
		m.eventBus.Emit(string(EventAICoderPaused), AICoderEvent{
			Type:      string(EventAICoderPaused),
			CoderID:   coderID,
			CoderName: coder.Name,
			Status:    string(StatusPaused),
			Message:   "AI coder paused",
			Time:      time.Now(),
		})
	}
	
	return nil
}

// ResumeCoder resumes a paused AI coder process
func (m *AICoderManager) ResumeCoder(coderID string) error {
	coder, exists := m.GetCoder(coderID)
	if !exists {
		return fmt.Errorf("AI coder %s not found", coderID)
	}
	
	if err := m.processMgr.ResumeCoder(coderID); err != nil {
		return fmt.Errorf("failed to resume AI coder: %w", err)
	}
	
	// Emit resume event
	if m.eventBus != nil {
		m.eventBus.Emit(string(EventAICoderResumed), AICoderEvent{
			Type:      string(EventAICoderResumed),
			CoderID:   coderID,
			CoderName: coder.Name,
			Status:    string(StatusRunning),
			Message:   "AI coder resumed",
			Time:      time.Now(),
		})
	}
	
	return nil
}

// DeleteCoder deletes an AI coder instance
func (m *AICoderManager) DeleteCoder(coderID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	coder, exists := m.coders[coderID]
	if !exists {
		return fmt.Errorf("AI coder %s not found", coderID)
	}
	
	// Stop if running
	if m.processMgr.IsActive(coderID) {
		if err := m.processMgr.StopCoder(coderID); err != nil {
			return fmt.Errorf("failed to stop AI coder before deletion: %w", err)
		}
	}
	
	// Clean up workspace
	if err := m.workspaceMgr.CleanupWorkspace(coder.WorkspaceDir); err != nil {
		// Log error but continue with deletion
		fmt.Printf("Warning: failed to cleanup workspace: %v\n", err)
	}
	
	// Remove from map
	delete(m.coders, coderID)
	
	// Emit deletion event
	if m.eventBus != nil {
		m.eventBus.Emit(string(EventAICoderDeleted), AICoderEvent{
			Type:      string(EventAICoderDeleted),
			CoderID:   coderID,
			CoderName: coder.Name,
			Status:    string(StatusStopped),
			Message:   "AI coder deleted",
			Time:      time.Now(),
		})
	}
	
	return nil
}

// UpdateCoderTask updates the task description for an AI coder
func (m *AICoderManager) UpdateCoderTask(coderID string, task string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	coder, exists := m.coders[coderID]
	if !exists {
		return fmt.Errorf("AI coder %s not found", coderID)
	}
	
	coder.Task = task
	coder.UpdatedAt = time.Now()
	
	return nil
}

// RegisterProvider registers a new AI provider
func (m *AICoderManager) RegisterProvider(name string, provider AIProvider) error {
	return m.providerReg.Register(name, provider)
}

// GetProviders returns a list of available providers
func (m *AICoderManager) GetProviders() []string {
	return m.providerReg.List()
}

// GetCoderWorkspace returns the workspace directory for an AI coder
func (m *AICoderManager) GetCoderWorkspace(coderID string) (string, error) {
	coder, exists := m.GetCoder(coderID)
	if !exists {
		return "", fmt.Errorf("AI coder %s not found", coderID)
	}
	return coder.WorkspaceDir, nil
}

// PTY Support Methods

// GetPTYManager returns the PTY manager
func (m *AICoderManager) GetPTYManager() *PTYManager {
	return m.ptyManager
}

// CreatePTYSession creates a new PTY session for an AI coder
func (m *AICoderManager) CreatePTYSession(name, command string, args []string) (*PTYSession, error) {
	if m.ptyManager == nil {
		return nil, fmt.Errorf("PTY support not initialized")
	}
	
	return m.ptyManager.CreateSession(name, command, args)
}

// CreateInteractiveCLISession creates a PTY session for interactive CLI usage
func (m *AICoderManager) CreateInteractiveCLISession(configKey string) (*PTYSession, error) {
	if m.ptyManager == nil {
		return nil, fmt.Errorf("PTY support not initialized")
	}
	
	// Get CLI command configuration
	command, args, err := m.getCLICommandFromConfig(configKey, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get CLI configuration: %w", err)
	}
	
	sessionName := fmt.Sprintf("%s-interactive", configKey)
	return m.ptyManager.CreateSession(sessionName, command, args)
}

// CreateTaskCLISession creates a PTY session for CLI with a specific task
func (m *AICoderManager) CreateTaskCLISession(configKey, task string) (*PTYSession, error) {
	if m.ptyManager == nil {
		return nil, fmt.Errorf("PTY support not initialized")
	}
	
	// Get CLI command configuration with task
	command, args, err := m.getCLICommandFromConfig(configKey, task)
	if err != nil {
		return nil, fmt.Errorf("failed to get CLI configuration: %w", err)
	}
	
	sessionName := fmt.Sprintf("%s-task", configKey)
	return m.ptyManager.CreateSession(sessionName, command, args)
}

// getCLICommandFromConfig retrieves CLI command and args from configuration
// This is similar to the one in model.go but belongs in the AI coder manager
func (m *AICoderManager) getCLICommandFromConfig(configKey string, task string) (string, []string, error) {
	// For now, use the auto-detected CLI tool mappings
	// This would be enhanced to use actual configuration
	
	cliMappings := map[string]struct {
		command string
		baseArgs []string
		taskFlag string
	}{
		"claude": {
			command: "claude",
			baseArgs: []string{},
			taskFlag: "", // Claude CLI accepts task as direct input
		},
		"sonnet": {
			command: "claude",
			baseArgs: []string{"--model", "sonnet"},
			taskFlag: "",
		},
		"opus": {
			command: "claude",
			baseArgs: []string{"--model", "opus"},
			taskFlag: "",
		},
		"aider": {
			command: "aider",
			baseArgs: []string{"--yes"},
			taskFlag: "--message",
		},
		"opencode": {
			command: "opencode",
			baseArgs: []string{"run"},
			taskFlag: "--prompt",
		},
		"gemini": {
			command: "gemini",
			baseArgs: []string{},
			taskFlag: "--prompt",
		},
	}
	
	mapping, exists := cliMappings[configKey]
	if !exists {
		return "", nil, fmt.Errorf("CLI configuration not found for '%s'", configKey)
	}
	
	args := make([]string, len(mapping.baseArgs))
	copy(args, mapping.baseArgs)
	
	// If task is provided, add it based on the tool's requirements
	if task != "" {
		if mapping.taskFlag != "" {
			// Tools that use flags for tasks (aider, opencode, gemini)
			args = append(args, mapping.taskFlag, task)
		} else {
			// Tools that accept task as direct input (claude)
			// For non-interactive mode, use structured output
			if configKey == "claude" || configKey == "sonnet" || configKey == "opus" {
				args = append(args, "--print", "--verbose", "--output-format", "stream-json")
			}
			args = append(args, task)
		}
	}
	// Note: For interactive mode (no task), no special flags are added
	// This gives us the full SSH-like experience
	
	return mapping.command, args, nil
}
