package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/pkg/events"
)

// configAdapter implements aicoder.Config using the real config
type configAdapter struct {
	cfg *config.Config
}

func (c *configAdapter) GetAICoderConfig() aicoder.AICoderConfig {
	if c.cfg == nil || c.cfg.AICoders == nil {
		// Return defaults if no config
		return aicoder.AICoderConfig{
			MaxConcurrent:    3,
			WorkspaceBaseDir: filepath.Join(os.Getenv("HOME"), ".brummer", "ai-coders"),
			DefaultProvider:  "claude",
			TimeoutMinutes:   30,
		}
	}

	aiCfg := c.cfg.AICoders

	// Helper function to safely dereference pointers with defaults
	intVal := func(ptr *int, def int) int {
		if ptr != nil {
			return *ptr
		}
		return def
	}
	stringVal := func(ptr *string, def string) string {
		if ptr != nil {
			return *ptr
		}
		return def
	}

	return aicoder.AICoderConfig{
		MaxConcurrent:    intVal(aiCfg.MaxConcurrent, 3),
		WorkspaceBaseDir: stringVal(aiCfg.WorkspaceBaseDir, filepath.Join(os.Getenv("HOME"), ".brummer", "ai-coders")),
		DefaultProvider:  stringVal(aiCfg.DefaultProvider, "claude"),
		TimeoutMinutes:   intVal(aiCfg.TimeoutMinutes, 30),
	}
}

func (c *configAdapter) GetProviderConfigs() map[string]*aicoder.ProviderConfig {
	result := make(map[string]*aicoder.ProviderConfig)

	if c.cfg == nil || c.cfg.AICoders == nil || c.cfg.AICoders.Providers == nil {
		return result
	}

	// Helper function to safely dereference string pointers
	stringVal := func(ptr *string, def string) string {
		if ptr != nil {
			return *ptr
		}
		return def
	}

	// Convert from config.ProviderConfig to aicoder.ProviderConfig
	for name, provider := range c.cfg.AICoders.Providers {
		if provider == nil {
			continue
		}

		aiProvider := &aicoder.ProviderConfig{}

		// Convert CLI tool config if present
		if provider.CLITool != nil {
			aiProvider.CLITool = &aicoder.CLIToolConfig{
				Command:     stringVal(provider.CLITool.Command, ""),
				BaseArgs:    provider.CLITool.BaseArgs,
				FlagMapping: provider.CLITool.FlagMapping,
				WorkingDir:  stringVal(provider.CLITool.WorkingDir, ""),
				Environment: provider.CLITool.Environment,
			}
		}

		// Copy other fields with pointer dereferencing
		if provider.Model != nil {
			aiProvider.Model = *provider.Model
		}
		if provider.APIKeyEnv != nil {
			aiProvider.APIKeyEnv = *provider.APIKeyEnv
		}
		if provider.MaxTokens != nil {
			aiProvider.MaxTokens = *provider.MaxTokens
		}
		if provider.Temperature != nil {
			aiProvider.Temperature = *provider.Temperature
		}

		result[name] = aiProvider
	}

	return result
}

// eventBusWrapper wraps the Brummer EventBus to implement aicoder.EventBus
type eventBusWrapper struct {
	eventBus *events.EventBus
}

func (e *eventBusWrapper) Subscribe(eventType string, handler func(data map[string]interface{})) {
	// TODO: Convert aicoder events to Brummer events when event integration is ready
}

func (e *eventBusWrapper) Publish(eventType string, data map[string]interface{}) {
	// TODO: Convert aicoder events to Brummer events when event integration is ready
}

func (e *eventBusWrapper) Emit(eventType string, data interface{}) {
	// TODO: Convert aicoder events to Brummer events when event integration is ready
}

// windowSizeMsg represents a window size change for the PTY view
type windowSizeMsg tea.WindowSizeMsg

// AICoderController manages AI Coder functionality and PTY view
type AICoderController struct {
	// Core AI Coder components
	aiCoderManager  *aicoder.AICoderManager
	ptyManager      *aicoder.PTYManager
	ptyDataProvider aicoder.BrummerDataProvider
	debugForwarder  *AICoderDebugForwarder
	ptyEventSub     chan aicoder.PTYEvent

	// PTY View management
	aiCoderPTYView *AICoderPTYView

	// Dependencies
	logStore      *logs.Store
	updateChan    chan tea.Msg
	width         int
	height        int
	headerHeight  int
	footerHeight  int
	contentHeight int // Pre-calculated content height

	// Configuration
	cfg     *config.Config
	mcpPort int

	// Initialization error (if any)
	initError error

	// Session creation state
	isCreatingSession atomic.Bool

	// Context for lifecycle management
	ctx    context.Context
	cancel context.CancelFunc

	// Goroutine tracking
	activeMonitors atomic.Int32
}

// NewAICoderController creates a new AI Coder controller
func NewAICoderController(cfg *config.Config, eventBus *events.EventBus, logStore *logs.Store, updateChan chan tea.Msg) *AICoderController {
	ctx, cancel := context.WithCancel(context.Background())

	// Get MCP port with default fallback
	mcpPort := 7777 // default
	if cfg != nil {
		mcpPort = cfg.GetMCPPort()
	}

	controller := &AICoderController{
		logStore:   logStore,
		updateChan: updateChan,
		cfg:        cfg,
		mcpPort:    mcpPort,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Initialize AI Coder configuration
	var aiCoderConfig aicoder.Config
	if cfg != nil {
		aiCoderConfig = &configAdapter{cfg: cfg}
	} else {
		// Fallback for tests or when no config is provided
		aiCoderConfig = &configAdapter{cfg: nil}
	}

	eventBusWrapper := &eventBusWrapper{eventBus: eventBus}

	// Create PTY data provider
	controller.ptyDataProvider = NewTUIDataProvider(nil) // We'll set the model reference later

	// Initialize AI Coder manager with PTY support
	var err error
	controller.aiCoderManager, err = aicoder.NewAICoderManagerWithPTY(aiCoderConfig, eventBusWrapper, controller.ptyDataProvider)
	if err != nil {
		// Log error but continue - AI Coder is optional
		fmt.Printf("Warning: Failed to initialize AI Coder manager: %v\n", err)
		logStore.Add("system", "System", fmt.Sprintf("AI Coder initialization failed: %v", err), true)
		// Store the error for display
		controller.initError = err
		return controller
	}

	// Get PTY manager from AI coder manager
	controller.ptyManager = controller.aiCoderManager.GetPTYManager()

	// Initialize debug forwarder with controller reference
	controller.debugForwarder = NewAICoderDebugForwarder(controller)

	// Subscribe to PTY events (commented out until PTY events are implemented)
	controller.ptyEventSub = make(chan aicoder.PTYEvent, 100)
	if controller.ptyManager != nil {
		// TODO: Implement PTY event subscription when available
		// controller.ptyManager.Subscribe(controller.ptyEventSub)

		// Initialize PTY view
		controller.aiCoderPTYView = NewAICoderPTYView(controller.ptyManager)
	}

	return controller
}

// SetModelReference sets the model reference for data provider and debug forwarder
func (c *AICoderController) SetModelReference(model interface{}) {
	if c.ptyDataProvider != nil {
		if dataProvider, ok := c.ptyDataProvider.(*TUIDataProvider); ok {
			if m, ok := model.(*Model); ok {
				dataProvider.SetModel(m)
			}
		}
	}

	if c.debugForwarder != nil {
		// TODO: Implement SetModel method on AICoderDebugForwarder when needed
		// c.debugForwarder.SetModel(model)
	}
}

// SetAICoderManager sets the AI coder manager (for testing)
func (c *AICoderController) SetAICoderManager(manager *aicoder.AICoderManager) {
	c.aiCoderManager = manager
	// Get PTY manager from the new AI coder manager if available
	if manager != nil {
		c.ptyManager = manager.GetPTYManager()
	}
}

// UpdateSize updates the controller and PTY view dimensions with pre-calculated content height
func (c *AICoderController) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	c.width = width
	c.height = height
	c.headerHeight = headerHeight
	c.footerHeight = footerHeight
	c.contentHeight = contentHeight

	// Update PTY view size if it exists
	if c.aiCoderPTYView != nil {
		// Directly set the dimensions on the PTY view
		c.aiCoderPTYView.width = width
		c.aiCoderPTYView.height = contentHeight

		// Also send the window size message for proper handling
		c.aiCoderPTYView.Update(tea.WindowSizeMsg{Width: width, Height: contentHeight})
	}
}

// Update handles messages for the AI Coder controller
func (c *AICoderController) Update(msg tea.Msg) tea.Cmd {
	if c.aiCoderPTYView == nil {
		return nil
	}

	// Handle PTY events
	select {
	case event := <-c.ptyEventSub:
		c.aiCoderPTYView.Update(PTYEventMsg{Event: event})
	default:
		// No event to process
	}

	var cmd tea.Cmd
	_, cmd = c.aiCoderPTYView.Update(msg)
	return cmd
}

// Render renders the AI Coder PTY view
func (c *AICoderController) Render() string {
	if c.initError != nil {
		return fmt.Sprintf("AI Coder initialization failed:\n\n%v\n\nPlease check your AI Coder configuration in .brum.toml", c.initError)
	}

	if c.aiCoderPTYView == nil {
		return "AI Coder PTY view not initialized"
	}

	// Ensure the PTY view has valid dimensions
	if c.aiCoderPTYView.width <= 0 || c.aiCoderPTYView.height <= 0 {
		// Force an update with current dimensions
		if c.width > 0 && c.height > 0 && c.contentHeight > 0 {
			c.aiCoderPTYView.Update(windowSizeMsg{Width: c.width, Height: c.contentHeight})
		}
	}

	return c.aiCoderPTYView.View()
}

// GetRawOutput returns the raw output for full screen mode
func (c *AICoderController) GetRawOutput() string {
	if c.aiCoderPTYView == nil {
		return "AI Coder PTY view not initialized"
	}
	return c.aiCoderPTYView.GetRawOutput()
}

// IsFullScreen returns whether the PTY view is in full screen mode
func (c *AICoderController) IsFullScreen() bool {
	if c.aiCoderPTYView == nil {
		return false
	}
	return c.aiCoderPTYView.isFullScreen
}

// IsTerminalFocused returns whether the terminal is currently focused
func (c *AICoderController) IsTerminalFocused() bool {
	if c.aiCoderPTYView == nil {
		return false
	}
	return c.aiCoderPTYView.IsTerminalFocused()
}

// ShouldInterceptSlashCommand determines if "/" should open Brummer command palette
func (c *AICoderController) ShouldInterceptSlashCommand() bool {
	if c.aiCoderPTYView == nil {
		return true // Default to allowing Brummer commands
	}
	return c.aiCoderPTYView.ShouldInterceptSlashCommand()
}

// GetProviders returns the list of available AI providers
func (c *AICoderController) GetProviders() []string {
	if c.aiCoderManager == nil {
		return nil
	}
	return c.aiCoderManager.GetProviders()
}

// GetStatusInfo returns status information for view display
func (c *AICoderController) GetStatusInfo() (int, int) {
	if c.aiCoderManager == nil {
		return 0, 0
	}

	coders := c.aiCoderManager.ListCoders()
	running := 0
	for _, coder := range coders {
		if coder.GetStatus() == aicoder.StatusRunning {
			running++
		}
	}
	return running, len(coders)
}

// HandleAICommand handles starting an AI coder with the specified provider
func (c *AICoderController) HandleAICommand(providerName string) {
	if c.aiCoderManager == nil {
		c.logStore.Add("system", "System", "AI coder feature is not enabled", true)
		return
	}

	// Check if a session is already being created
	if !c.isCreatingSession.CompareAndSwap(false, true) {
		c.logStore.Add("system", "System", "An AI coder session is already being created. Please wait a few seconds and try again.", true)
		return
	}

	// Log successful acquisition of creation flag
	c.logStore.Add("system", "System", fmt.Sprintf("Starting AI coder session creation for provider: %s", providerName), false)

	// Create and start the AI coder in a goroutine
	SafeGoroutineNoError(
		fmt.Sprintf("create AI coder session with provider '%s'", providerName),
		func() {
			// Ensure we reset the flag when done
			defer c.isCreatingSession.Store(false)

			// Create the AI coder with proper context
			coder, err := c.aiCoderManager.CreateCoder(c.ctx, aicoder.CreateCoderRequest{
				Provider: providerName,
				Task:     "Interactive AI coding session",
			})
			if err != nil {
				errorMsg := fmt.Sprintf("Error creating AI coder with provider '%s': %v", providerName, err)
				c.logStore.Add("system", "System", errorMsg, true)
				c.updateChan <- logUpdateMsg{}
				return
			}

			// Track coder ID for cleanup
			coderID := coder.ID

			// Start the AI coder
			if err := c.aiCoderManager.StartCoder(coderID); err != nil {
				errorMsg := fmt.Sprintf("Error starting AI coder '%s' (provider: %s): %v", coderID, providerName, err)
				c.logStore.Add("system", "System", errorMsg, true)
				// Clean up the created but not started coder
				if deleteErr := c.aiCoderManager.DeleteCoder(coderID); deleteErr != nil {
					cleanupError := fmt.Sprintf("Error cleaning up AI coder '%s' after start failure: %v", coderID, deleteErr)
					c.logStore.Add("system", "System", cleanupError, true)
				}
				c.updateChan <- logUpdateMsg{}
				return
			}

			// Create a PTY session for the AI coder
			if c.ptyManager != nil {
				// Get provider configuration through the adapter
				adapter := &configAdapter{cfg: c.cfg}
				providerConfigs := adapter.GetProviderConfigs()

				providerConfig, exists := providerConfigs[providerName]
				if !exists {
					// Get available provider names for error message
					availableProviders := make([]string, 0, len(providerConfigs))
					for name := range providerConfigs {
						availableProviders = append(availableProviders, name)
					}
					c.logStore.Add("system", "System", fmt.Sprintf("Provider %s not configured. Available: %v", providerName, availableProviders), true)
					c.updateChan <- logUpdateMsg{}
					return
				}

				// Create PTY session with the provider's CLI tool
				if providerConfig.CLITool != nil {
					sessionName := fmt.Sprintf("%s AI Coder", providerName)
					command := providerConfig.CLITool.Command
					args := providerConfig.CLITool.BaseArgs
					if args == nil {
						args = []string{}
					}

					// Special handling for terminal provider
					if providerName == "terminal" {
						// Use bash in interactive mode
						command = "/bin/bash"
						args = []string{"-i"}
					} else {
						// For other providers, expand environment variables in args
						mcpURL := fmt.Sprintf("http://localhost:%d/mcp", c.mcpPort)
						expandedArgs := make([]string, len(args))
						for i, arg := range args {
							// Replace ${BRUMMER_MCP_URL} with actual URL
							expandedArg := strings.ReplaceAll(arg, "${BRUMMER_MCP_URL}", mcpURL)
							expandedArgs[i] = expandedArg
						}
						args = expandedArgs
					}

					// Set up environment variables
					envMap := make(map[string]string)
					envMap["BRUMMER_MCP_URL"] = fmt.Sprintf("http://localhost:%d/mcp", c.mcpPort)
					envMap["BRUMMER_MCP_PORT"] = fmt.Sprintf("%d", c.mcpPort)

					// Add provider-specific environment variables
					if providerConfig.CLITool.Environment != nil {
						for k, v := range providerConfig.CLITool.Environment {
							envMap[k] = v
						}
					}

					session, err := c.ptyManager.CreateSessionWithEnv(sessionName, command, args, envMap)
					if err != nil {
						// Provide more helpful error messages
						if strings.Contains(err.Error(), "executable file not found") {
							c.logStore.Add("system", "System", fmt.Sprintf("Command '%s' not found. Please install the %s CLI tool.", command, providerName), true)
						} else {
							c.logStore.Add("system", "System", fmt.Sprintf("Error creating PTY session: %v", err), true)
						}
						c.updateChan <- logUpdateMsg{}
						return
					}

					// Set the current session in the PTY view
					if c.aiCoderPTYView != nil {
						c.aiCoderPTYView.SetCurrentSession(session)
						c.logStore.Add("system", "System", fmt.Sprintf("Started %s AI coder session", providerName), false)

						// Ensure dimensions are set before switching views
						if c.width > 0 && c.height > 0 && c.contentHeight > 0 {
							c.aiCoderPTYView.Update(windowSizeMsg{Width: c.width, Height: c.contentHeight})
						}

						// Start monitoring PTY output
						go c.monitorPTYOutput(session)

						// Switch to AI Coders view to show the session
						c.updateChan <- switchToAICodersMsg{}
						// Trigger immediate update to show the session
						c.updateChan <- processUpdateMsg{}
					}
				} else {
					c.logStore.Add("system", "System", fmt.Sprintf("Provider %s does not have CLI tool configured", providerName), true)
				}
			}

			c.updateChan <- processUpdateMsg{}
		},
		func(err error) {
			c.isCreatingSession.Store(false) // Ensure flag is reset on panic
			errorMsg := fmt.Sprintf("Critical error during AI coder session creation: %v", err)
			c.logStore.Add("system", "System", errorMsg, true)
			c.updateChan <- logUpdateMsg{}
		},
	)
}

// GetAICoderManager returns the AI coder manager instance
func (c *AICoderController) GetAICoderManager() *aicoder.AICoderManager {
	return c.aiCoderManager
}

// GetPTYView returns the PTY view for direct access
func (c *AICoderController) GetPTYView() *AICoderPTYView {
	return c.aiCoderPTYView
}

// IsInitialized returns whether the controller is properly initialized
func (c *AICoderController) IsInitialized() bool {
	return c.aiCoderManager != nil && c.ptyManager != nil && c.aiCoderPTYView != nil
}

// ListSessions returns a list of active PTY sessions
func (c *AICoderController) ListSessions() []*aicoder.PTYSession {
	if c.ptyManager == nil {
		return nil
	}
	return c.ptyManager.ListSessions()
}

// AttachToSession attaches the view to a specific session
func (c *AICoderController) AttachToSession(sessionID string) error {
	if c.aiCoderPTYView == nil {
		return fmt.Errorf("PTY view not initialized")
	}
	return c.aiCoderPTYView.AttachToSession(sessionID)
}

// CreateTerminalSession creates a simple terminal session
func (c *AICoderController) CreateTerminalSession() error {
	if c.ptyManager == nil {
		return fmt.Errorf("PTY manager not initialized")
	}

	// Create a basic terminal session
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	session, err := c.ptyManager.CreateSession("Terminal", shell, []string{"-i"})
	if err != nil {
		return err
	}

	// Attach to the new session
	if c.aiCoderPTYView != nil {
		c.aiCoderPTYView.SetCurrentSession(session)
		// Start monitoring PTY output
		go c.monitorPTYOutput(session)
	}

	return nil
}

// monitorPTYOutput monitors PTY output and triggers view updates
func (c *AICoderController) monitorPTYOutput(session *aicoder.PTYSession) {
	// Track this goroutine
	c.activeMonitors.Add(1)
	defer c.activeMonitors.Add(-1)

	// Log monitor start for debugging
	c.logStore.Add("system", "System", fmt.Sprintf("Started PTY output monitor for session %s (active monitors: %d)",
		session.ID, c.activeMonitors.Load()), false)

	defer func() {
		c.logStore.Add("system", "System", fmt.Sprintf("Stopped PTY output monitor for session %s (active monitors: %d)",
			session.ID, c.activeMonitors.Load()), false)
	}()

	for {
		select {
		case <-c.ctx.Done():
			// Controller is shutting down
			return

		case output, ok := <-session.OutputChan:
			if !ok {
				// Channel closed, session ended
				return
			}

			// Trigger a view update
			c.updateChan <- PTYOutputMsg{
				SessionID: session.ID,
				Data:      output,
			}

		case event := <-session.EventChan:
			// Handle PTY events
			switch event.Type {
			case aicoder.PTYEventClose:
				// Send event to PTY view
				c.updateChan <- PTYEventMsg{Event: event}
				// Exit monitoring
				return
			case aicoder.PTYEventResize:
				// Send event to PTY view
				c.updateChan <- PTYEventMsg{Event: event}
			}
			continue
		}
	}
}
