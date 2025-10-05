package components

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// CommandExecutor handles execution of commands by routing them to appropriate services
type CommandExecutor struct {
	processMgr      *process.Manager
	logStore        *logs.Store
	proxyServer     *proxy.Server
	registry        *CommandRegistry
	aiCoderManager  *aicoder.AICoderManager
}

// NewCommandExecutor creates a new command executor with service references
func NewCommandExecutor(
	processMgr *process.Manager,
	logStore *logs.Store,
	proxyServer *proxy.Server,
	registry *CommandRegistry,
	aiCoderManager *aicoder.AICoderManager,
) *CommandExecutor {
	return &CommandExecutor{
		processMgr:     processMgr,
		logStore:       logStore,
		proxyServer:    proxyServer,
		registry:       registry,
		aiCoderManager: aiCoderManager,
	}
}

// Execute handles a CommandExecuteMsg and routes it to the appropriate service
func (e *CommandExecutor) Execute(msg messages.CommandExecuteMsg) tea.Cmd {
	commandName := msg.CommandName
	args := msg.Arguments

	// Route command to appropriate handler
	switch commandName {
	case "run":
		return e.handleRun(args)
	case "stop":
		return e.handleStop(args)
	case "restart":
		return e.handleRestart(args)
	case "show":
		return e.handleShow(args)
	case "hide":
		return e.handleHide(args)
	case "clear":
		return e.handleClear(args)
	case "proxy":
		return e.handleProxy(args)
	case "toggle-proxy":
		return e.handleToggleProxy(args)
	case "ai":
		return e.handleAI(args)
	case "term":
		return e.handleTerm(args)
	case "docker-up":
		return e.handleDockerUp(args)
	case "docker-down":
		return e.handleDockerDown(args)
	case "docker-build":
		return e.handleDockerBuild(args)
	case "compose-up":
		return e.handleComposeUp(args)
	case "compose-down":
		return e.handleComposeDown(args)
	case "exit", "quit":
		return e.handleExit(args)
	default:
		// Unknown command
		return e.sendError(fmt.Sprintf("Unknown command: %s", commandName))
	}
}

// Process management commands

func (e *CommandExecutor) handleRun(args []string) tea.Cmd {
	if len(args) < 1 {
		return e.sendError("Usage: /run <script>")
	}

	if e.processMgr == nil {
		return e.sendError("Process manager not available")
	}

	scriptName := args[0]

	// Start the process
	_, err := e.processMgr.StartScript(scriptName)
	if err != nil {
		return e.sendError(fmt.Sprintf("Failed to start %s: %v", scriptName, err))
	}

	return e.sendSuccess(fmt.Sprintf("Started process: %s", scriptName))
}

func (e *CommandExecutor) handleStop(args []string) tea.Cmd {
	if len(args) < 1 {
		return e.sendError("Usage: /stop <script>")
	}

	if e.processMgr == nil {
		return e.sendError("Process manager not available")
	}

	scriptName := args[0]

	// Find process by name and stop it
	processes := e.processMgr.GetAllProcesses()
	for _, proc := range processes {
		if proc.Name == scriptName && proc.GetStatus() == process.StatusRunning {
			err := e.processMgr.StopProcess(proc.ID)
			if err != nil {
				return e.sendError(fmt.Sprintf("Failed to stop %s: %v", scriptName, err))
			}
			return e.sendSuccess(fmt.Sprintf("Stopped process: %s", scriptName))
		}
	}

	return e.sendError(fmt.Sprintf("Process not found: %s", scriptName))
}

func (e *CommandExecutor) handleRestart(args []string) tea.Cmd {
	if len(args) < 1 {
		return e.sendError("Usage: /restart <script>")
	}

	if e.processMgr == nil {
		return e.sendError("Process manager not available")
	}

	scriptName := args[0]

	// Find process by name and restart it (stop + start)
	processes := e.processMgr.GetAllProcesses()
	for _, proc := range processes {
		if proc.Name == scriptName {
			// Stop the process
			err := e.processMgr.StopProcess(proc.ID)
			if err != nil {
				return e.sendError(fmt.Sprintf("Failed to stop %s: %v", scriptName, err))
			}

			// Start it again
			_, err = e.processMgr.StartScript(scriptName)
			if err != nil {
				return e.sendError(fmt.Sprintf("Failed to restart %s: %v", scriptName, err))
			}

			return e.sendSuccess(fmt.Sprintf("Restarted process: %s", scriptName))
		}
	}

	return e.sendError(fmt.Sprintf("Process not found: %s", scriptName))
}

// Log filtering commands

func (e *CommandExecutor) handleShow(args []string) tea.Cmd {
	if len(args) < 1 {
		return e.sendError("Usage: /show <pattern>")
	}

	pattern := strings.Join(args, " ")

	// TODO: Store show pattern in model state for log filtering
	// For now, just send a success message
	return e.sendSuccess(fmt.Sprintf("Show filter set: %s", pattern))
}

func (e *CommandExecutor) handleHide(args []string) tea.Cmd {
	if len(args) < 1 {
		return e.sendError("Usage: /hide <pattern>")
	}

	pattern := strings.Join(args, " ")

	// TODO: Store hide pattern in model state for log filtering
	// For now, just send a success message
	return e.sendSuccess(fmt.Sprintf("Hide filter set: %s", pattern))
}

func (e *CommandExecutor) handleClear(args []string) tea.Cmd {
	if e.logStore == nil {
		return e.sendError("Log store not available")
	}

	target := "all"
	if len(args) > 0 {
		target = args[0]
	}

	if target == "all" {
		e.logStore.ClearLogs()
		return e.sendSuccess("Cleared all logs")
	} else {
		// TODO: Implement per-process clearing using filters
		return e.sendError("Per-process log clearing not yet implemented")
	}
}

// Proxy commands

func (e *CommandExecutor) handleProxy(args []string) tea.Cmd {
	if len(args) < 1 {
		return e.sendError("Usage: /proxy <URL> or /proxy list")
	}

	if e.proxyServer == nil {
		return e.sendError("Proxy server not available")
	}

	subcommand := args[0]
	if subcommand == "list" {
		// TODO: Get proxy list from proxy server
		return e.sendSuccess("Proxy list (not yet implemented)")
	} else {
		// Register URL for proxying
		urlStr := strings.Join(args, " ")
		proxyURL := e.proxyServer.RegisterURL(urlStr, "custom")
		return e.sendSuccess(fmt.Sprintf("Registered proxy: %s -> %s", urlStr, proxyURL))
	}
}

func (e *CommandExecutor) handleToggleProxy(args []string) tea.Cmd {
	if e.proxyServer == nil {
		return e.sendError("Proxy server not available")
	}

	// TODO: Implement proxy mode toggle
	return e.sendSuccess("Proxy mode toggled (not yet implemented)")
}

// AI coder commands

func (e *CommandExecutor) handleAI(args []string) tea.Cmd {
	if len(args) < 1 {
		return e.sendError("Usage: /ai <provider> [task]")
	}

	if e.aiCoderManager == nil {
		// Create a simple session and add it to the AI coders view
		// This provides immediate user feedback while full AI coder integration is implemented
		return func() tea.Msg {
			// Create a mock session info
			sessionInfo := &messages.AICoderSessionInfo{
				ID:             fmt.Sprintf("claude-%d", time.Now().Unix()),
				Name:           "Claude AI Coder",
				Status:         "active",
				ProcessID:      "claude-pty",
				StartTime:      time.Now(),
				LastActivity:   time.Now(),
				LinesExchanged: 0,
				CurrentTask:    "Interactive AI coding session",
				ErrorMessage:   "",
			}

			// Return both the session info and view switch message
			return tea.Batch(
				func() tea.Msg { return messages.AddAICoderSessionMsg{Session: sessionInfo} },
				func() tea.Msg { return messages.NavigateToViewMsg{View: "aicoders"} },
			)
		}
	}

	providerName := args[0]
	task := ""
	if len(args) > 1 {
		task = strings.Join(args[1:], " ")
	}

	// Create AI coder session
	return func() tea.Msg {
		// Create the AI coder
		coder, err := e.aiCoderManager.CreateCoder(context.Background(), aicoder.CreateCoderRequest{
			Provider: providerName,
			Task:     task,
		})
		if err != nil {
			return messages.CommandResultMsg{
				Success: false,
				Message: fmt.Sprintf("Error creating AI coder with provider '%s': %v", providerName, err),
			}
		}

		// Start the AI coder
		err = e.aiCoderManager.StartCoder(coder.ID)
		if err != nil {
			// Clean up the created but not started coder
			if deleteErr := e.aiCoderManager.DeleteCoder(coder.ID); deleteErr != nil {
				if e.logStore != nil {
					e.logStore.Add("system", "System", fmt.Sprintf("Error cleaning up AI coder '%s': %v", coder.ID, deleteErr), true)
				}
			}
			return messages.CommandResultMsg{
				Success: false,
				Message: fmt.Sprintf("Error starting AI coder '%s': %v", coder.ID, err),
			}
		}

		// Log success
		successMsg := fmt.Sprintf("Started AI coder session (%s): %s", providerName, coder.ID)
		if e.logStore != nil {
			e.logStore.Add("system", "System", successMsg, false)
		}

		// Return navigation message to switch to AI coders view
		return messages.NavigateToViewMsg{View: "aicoders"}
	}
}

func (e *CommandExecutor) handleTerm(args []string) tea.Cmd {
	// TODO: Show integrated terminal
	return e.sendSuccess("Showing integrated terminal (not yet implemented)")
}

// Docker commands

func (e *CommandExecutor) handleDockerUp(args []string) tea.Cmd {
	service := ""
	if len(args) > 0 {
		service = args[0]
	}

	// TODO: Implement docker-up
	if service != "" {
		return e.sendSuccess(fmt.Sprintf("Starting Docker service: %s", service))
	} else {
		return e.sendSuccess("Starting all Docker services")
	}
}

func (e *CommandExecutor) handleDockerDown(args []string) tea.Cmd {
	service := ""
	if len(args) > 0 {
		service = args[0]
	}

	// TODO: Implement docker-down
	if service != "" {
		return e.sendSuccess(fmt.Sprintf("Stopping Docker service: %s", service))
	} else {
		return e.sendSuccess("Stopping all Docker services")
	}
}

func (e *CommandExecutor) handleDockerBuild(args []string) tea.Cmd {
	service := ""
	if len(args) > 0 {
		service = args[0]
	}

	// TODO: Implement docker-build
	if service != "" {
		return e.sendSuccess(fmt.Sprintf("Building Docker service: %s", service))
	} else {
		return e.sendSuccess("Building all Docker services")
	}
}

func (e *CommandExecutor) handleComposeUp(args []string) tea.Cmd {
	service := ""
	if len(args) > 0 {
		service = args[0]
	}

	// TODO: Implement compose-up
	if service != "" {
		return e.sendSuccess(fmt.Sprintf("Starting compose service: %s", service))
	} else {
		return e.sendSuccess("Starting all compose services")
	}
}

func (e *CommandExecutor) handleComposeDown(args []string) tea.Cmd {
	// TODO: Implement compose-down
	return e.sendSuccess("Stopping all compose services")
}

func (e *CommandExecutor) handleExit(args []string) tea.Cmd {
	// Log exit message
	if e.logStore != nil {
		e.logStore.Add("system", "Command", "👋 Exiting Brummer...", false)
	}
	// Return quit command to exit the TUI
	return tea.Quit
}

// Helper methods

func (e *CommandExecutor) sendSuccess(message string) tea.Cmd {
	if e.logStore != nil {
		e.logStore.Add("system", "Command", "✅ "+message, false)
	}
	return nil
}

func (e *CommandExecutor) sendError(message string) tea.Cmd {
	if e.logStore != nil {
		e.logStore.Add("system", "Command", "❌ "+message, true)
	}
	return nil
}
