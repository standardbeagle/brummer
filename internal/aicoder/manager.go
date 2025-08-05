package aicoder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AICoderManager manages AI coder instances
type AICoderManager struct {
	coders       map[string]*AICoderProcess
	mu           sync.RWMutex
	eventBus     EventBus
	config       Config
	workspaceMgr *WorkspaceManager
	providerReg  *ProviderRegistry
	processMgr   *ProcessManager

	// PTY support for interactive sessions
	ptyManager *PTYManager
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

	// Get provider configurations to check which ones have CLI tools
	providerConfigs := config.GetProviderConfigs()

	// Register built-in providers only if they don't have CLI tool configurations
	if _, hasCLI := providerConfigs["claude"]; !hasCLI {
		claudeProvider := NewClaudeProvider("", "")
		if err := providerReg.Register("claude", claudeProvider); err != nil {
			return nil, fmt.Errorf("failed to register claude provider: %w", err)
		}
	}

	// Register OpenAI provider (always register as no CLI config exists)
	openaiProvider := NewOpenAIProvider("", "")
	if err := providerReg.Register("openai", openaiProvider); err != nil {
		return nil, fmt.Errorf("failed to register openai provider: %w", err)
	}

	if _, hasCLI := providerConfigs["gemini"]; !hasCLI {
		geminiProvider := NewGeminiProvider("", "")
		if err := providerReg.Register("gemini", geminiProvider); err != nil {
			return nil, fmt.Errorf("failed to register gemini provider: %w", err)
		}
	}

	if _, hasCLI := providerConfigs["terminal"]; !hasCLI {
		terminalProvider := NewTerminalProvider("")
		if err := providerReg.Register("terminal", terminalProvider); err != nil {
			return nil, fmt.Errorf("failed to register terminal provider: %w", err)
		}
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
		workspaceMgr: m.workspaceMgr,                          // Inject workspace manager
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

// registerCLIToolProviders registers CLI tool providers based on configuration and installation check
func registerCLIToolProviders(providerReg *ProviderRegistry, config Config) error {
	// Get provider configurations
	providerConfigs := config.GetProviderConfigs()

	// Go through all configured providers
	for name, providerConfig := range providerConfigs {
		// Skip if no CLI tool configuration
		if providerConfig == nil || providerConfig.CLITool == nil {
			continue
		}

		cliToolConfig := providerConfig.CLITool

		// Get the command name
		command := cliToolConfig.Command
		if command == "" {
			command = name // default to provider name
		}

		// Check if the command is installed
		if _, err := exec.LookPath(command); err != nil {
			// Command not found, skip this provider
			continue
		}

		// Create and register the provider
		cliProvider := NewCLIToolProvider(name, *cliToolConfig)
		if err := providerReg.Register(name, cliProvider); err != nil {
			// Log error but continue with other providers
			fmt.Printf("Warning: failed to register %s provider: %v\n", name, err)
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
	return m.CreateInteractiveCLISessionWithEnv(configKey, nil)
}

// CreateInteractiveCLISessionWithEnv creates a PTY session for interactive CLI usage with environment variables
func (m *AICoderManager) CreateInteractiveCLISessionWithEnv(configKey string, extraEnv map[string]string) (*PTYSession, error) {
	if m.ptyManager == nil {
		return nil, fmt.Errorf("PTY support not initialized")
	}

	// Get CLI command configuration
	command, args, err := m.getCLICommandFromConfig(configKey, "", extraEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to get CLI configuration: %w", err)
	}

	sessionName := fmt.Sprintf("%s-interactive", configKey)
	return m.ptyManager.CreateSessionWithEnv(sessionName, command, args, extraEnv)
}

// CreateTaskCLISession creates a PTY session for CLI with a specific task
func (m *AICoderManager) CreateTaskCLISession(configKey, task string) (*PTYSession, error) {
	return m.CreateTaskCLISessionWithEnv(configKey, task, nil)
}

// CreateTaskCLISessionWithEnv creates a PTY session for CLI with a specific task and environment variables
func (m *AICoderManager) CreateTaskCLISessionWithEnv(configKey, task string, extraEnv map[string]string) (*PTYSession, error) {
	if m.ptyManager == nil {
		return nil, fmt.Errorf("PTY support not initialized")
	}

	// Get CLI command configuration with task
	command, args, err := m.getCLICommandFromConfig(configKey, task, extraEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to get CLI configuration: %w", err)
	}

	sessionName := fmt.Sprintf("%s-task", configKey)
	return m.ptyManager.CreateSessionWithEnv(sessionName, command, args, extraEnv)
}

// getCLICommandFromConfig retrieves CLI command and args from configuration
func (m *AICoderManager) getCLICommandFromConfig(configKey string, task string, extraEnv map[string]string) (string, []string, error) {
	// Get provider configurations
	providerConfigs := m.config.GetProviderConfigs()

	// Look for provider configuration
	providerConfig, exists := providerConfigs[configKey]
	if !exists || providerConfig == nil || providerConfig.CLITool == nil {
		return "", nil, fmt.Errorf("CLI tool configuration not found for '%s'", configKey)
	}

	cliTool := providerConfig.CLITool

	// Get command
	command := cliTool.Command
	if command == "" {
		command = configKey // default to provider name
	}

	// Start with base args
	args := make([]string, len(cliTool.BaseArgs))
	copy(args, cliTool.BaseArgs)

	// Perform variable replacement in args
	args = m.replaceVariables(args, extraEnv)

	// If task is provided, add it based on the tool's flag mapping
	if task != "" {
		// Check if there's a specific task/message/prompt flag in the mapping
		taskFlag := ""
		for _, key := range []string{"task", "message", "prompt"} {
			if flag, exists := cliTool.FlagMapping[key]; exists {
				taskFlag = flag
				break
			}
		}

		if taskFlag != "" {
			// Tools that use flags for tasks
			args = append(args, taskFlag, task)
		} else {
			// Tools that accept task as direct input (like claude)
			// For non-interactive mode with claude, use structured output
			if command == "claude" {
				args = append(args, "--print", "--verbose", "--output-format", "stream-json")
			}
			args = append(args, task)
		}
	}
	// Note: For interactive mode (no task), no special flags are added
	// This gives us the full SSH-like experience

	return command, args, nil
}

// replaceVariables replaces environment variable placeholders in arguments
func (m *AICoderManager) replaceVariables(args []string, extraEnv map[string]string) []string {
	result := make([]string, len(args))

	for i, arg := range args {
		// Replace ${VAR_NAME} patterns with environment variable values
		replaced := arg

		// First check extraEnv (has priority)
		if extraEnv != nil {
			for key, value := range extraEnv {
				placeholder := "${" + key + "}"
				replaced = strings.ReplaceAll(replaced, placeholder, value)
			}
		}

		// Then check system environment variables
		for _, env := range os.Environ() {
			pair := strings.SplitN(env, "=", 2)
			if len(pair) == 2 {
				placeholder := "${" + pair[0] + "}"
				replaced = strings.ReplaceAll(replaced, placeholder, pair[1])
			}
		}

		result[i] = replaced
	}

	return result
}
