package commands

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/domain/processes"
)

// SlashCommandContext contains only essential dependencies for slash commands
// SIMPLIFIED: Reduced from 14 fields to 3 fields
type SlashCommandContext struct {
	ProcessManager *processes.Manager
	LogStore       *logs.Store
	Model          interface{} // Access to Model for callbacks - typed as interface{} to avoid import cycle
}

// Command result messages for proper BubbleTea pattern
type CommandResultMsg struct {
	Command string
	Success bool
	Message string
	View    string // Optional view to switch to
}

// HandleSlashCommand processes slash commands and returns proper tea.Cmd
// NO BLOCKING: Returns tea.Cmd that BubbleTea runtime will execute
func HandleSlashCommand(ctx *SlashCommandContext, input string) tea.Cmd {
	// Parse the command
	input = strings.TrimSpace(input)

	// If the command doesn't start with /, add it
	if !strings.HasPrefix(input, "/") {
		input = "/" + input
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	command := parts[0]

	switch command {
	case "/run":
		return handleRun(ctx, parts)

	case "/restart":
		return handleRestart(ctx, parts)

	case "/stop":
		return handleStop(ctx, parts)

	case "/clear":
		return handleClear(ctx, parts)

	case "/show":
		return handleShow(ctx, parts)

	case "/hide":
		return handleHide(ctx, parts)

	case "/proxy":
		return handleProxy(ctx, parts)

	case "/toggle-proxy":
		return handleToggleProxy(ctx)

	case "/ai":
		return handleAI(ctx, parts)

	case "/term":
		return handleTerm(ctx)

	case "/docker-up":
		return handleDockerUp(ctx, parts)

	case "/docker-down":
		return handleDockerDown(ctx, parts)

	case "/docker-build":
		return handleDockerBuild(ctx, parts)

	case "/compose-up":
		return handleComposeUp(ctx, parts)

	case "/compose-down":
		return handleComposeDown(ctx, parts)

	case "/help":
		return func() tea.Msg {
			return CommandResultMsg{
				Command: "help",
				Success: true,
				View:    "help",
			}
		}

	case "/exit", "/quit":
		return func() tea.Msg {
			return tea.Quit()
		}

	default:
		return handleUnknown(ctx, command)
	}
}

// handleRun starts a script from package.json
func handleRun(ctx *SlashCommandContext, parts []string) tea.Cmd {
	if len(parts) < 2 {
		return logError(ctx, "run", "Command requires a script name")
	}

	scriptName := parts[1]

	return func() tea.Msg {
		process, err := ctx.ProcessManager.StartScript(scriptName)
		if err != nil {
			return logErrorMsg(ctx, "run", fmt.Sprintf("Failed to start script '%s': %v", scriptName, err))
		}

		return logSuccessMsg(ctx, "run", fmt.Sprintf("Started script '%s' (process ID: %s)", scriptName, process.ID), "logs")
	}
}

// handleRestart restarts one or all processes
func handleRestart(ctx *SlashCommandContext, parts []string) tea.Cmd {
	processName := "all"
	if len(parts) >= 2 {
		processName = parts[1]
	}

	return func() tea.Msg {
		if processName == "all" {
			return restartAllProcesses(ctx)
		}
		return restartProcess(ctx, processName)
	}
}

// restartAllProcesses restarts all running processes
func restartAllProcesses(ctx *SlashCommandContext) tea.Msg {
	// Get all running processes
	allProcesses := ctx.ProcessManager.GetAllProcesses()
	var scriptNames []string
	for _, p := range allProcesses {
		snapshot := p.GetSnapshot()
		if snapshot.IsRunning() {
			scriptNames = append(scriptNames, snapshot.Name)
		}
	}

	// Stop all processes
	if err := ctx.ProcessManager.StopAllProcesses(); err != nil {
		return logErrorMsg(ctx, "restart", fmt.Sprintf("Failed to stop all processes: %v", err))
	}

	ctx.LogStore.Add("system", "System", "✅ Stopped all processes", false)

	// Restart each script
	successCount := 0
	for _, scriptName := range scriptNames {
		if _, err := ctx.ProcessManager.StartScript(scriptName); err != nil {
			ctx.LogStore.Add("system", "System", fmt.Sprintf("❌ Failed to restart script '%s': %v", scriptName, err), true)
		} else {
			ctx.LogStore.Add("system", "System", fmt.Sprintf("✅ Restarted script '%s'", scriptName), false)
			successCount++
		}
	}

	return CommandResultMsg{
		Command: "restart",
		Success: true,
		Message: fmt.Sprintf("Restarted %d/%d processes", successCount, len(scriptNames)),
		View:    "processes",
	}
}

// restartProcess restarts a specific process
func restartProcess(ctx *SlashCommandContext, processName string) tea.Msg {
	// Find the process
	var foundProcess *processes.Process
	allProcesses := ctx.ProcessManager.GetAllProcesses()
	for _, p := range allProcesses {
		snapshot := p.GetSnapshot()
		if snapshot.Name == processName {
			foundProcess = p
			break
		}
	}

	if foundProcess == nil {
		return logErrorMsg(ctx, "restart", fmt.Sprintf("Process '%s' not found", processName))
	}

	// Stop the process
	if err := ctx.ProcessManager.StopProcess(foundProcess.ID); err != nil {
		return logErrorMsg(ctx, "restart", fmt.Sprintf("Failed to stop process '%s': %v", processName, err))
	}

	ctx.LogStore.Add("system", "System", fmt.Sprintf("✅ Stopped process '%s'", processName), false)

	// Restart the script
	if _, err := ctx.ProcessManager.StartScript(processName); err != nil {
		return logErrorMsg(ctx, "restart", fmt.Sprintf("Failed to restart script '%s': %v", processName, err))
	}

	return logSuccessMsg(ctx, "restart", fmt.Sprintf("Restarted script '%s'", processName), "processes")
}

// handleStop stops one or all processes
func handleStop(ctx *SlashCommandContext, parts []string) tea.Cmd {
	processName := "all"
	if len(parts) >= 2 {
		processName = parts[1]
	}

	return func() tea.Msg {
		if processName == "all" {
			if err := ctx.ProcessManager.StopAllProcesses(); err != nil {
				return logErrorMsg(ctx, "stop", fmt.Sprintf("Failed to stop all processes: %v", err))
			}
			return logSuccessMsg(ctx, "stop", "Stopped all processes", "processes")
		}

		// Find and stop specific process
		var foundProcess *processes.Process
		allProcesses := ctx.ProcessManager.GetAllProcesses()
		for _, p := range allProcesses {
			snapshot := p.GetSnapshot()
			if snapshot.Name == processName {
				foundProcess = p
				break
			}
		}

		if foundProcess == nil {
			return logErrorMsg(ctx, "stop", fmt.Sprintf("Process '%s' not found", processName))
		}

		if err := ctx.ProcessManager.StopProcess(foundProcess.ID); err != nil {
			return logErrorMsg(ctx, "stop", fmt.Sprintf("Failed to stop process '%s': %v", processName, err))
		}

		return logSuccessMsg(ctx, "stop", fmt.Sprintf("Stopped process '%s'", processName), "processes")
	}
}

// handleClear clears logs
func handleClear(ctx *SlashCommandContext, parts []string) tea.Cmd {
	target := "all"
	if len(parts) >= 2 {
		target = parts[1]
	}

	return func() tea.Msg {
		return CommandResultMsg{
			Command: "clear",
			Success: true,
			Message: target, // Model will handle the clearing
		}
	}
}

// handleShow sets a show filter pattern - returns message for Model to handle
func handleShow(ctx *SlashCommandContext, parts []string) tea.Cmd {
	if len(parts) < 2 {
		return logError(ctx, "show", "Command requires a pattern")
	}

	pattern := strings.Join(parts[1:], " ")

	return func() tea.Msg {
		return CommandResultMsg{
			Command: "show",
			Success: true,
			Message: pattern,
		}
	}
}

// handleHide sets a hide filter pattern - returns message for Model to handle
func handleHide(ctx *SlashCommandContext, parts []string) tea.Cmd {
	if len(parts) < 2 {
		return logError(ctx, "hide", "Command requires a pattern")
	}

	pattern := strings.Join(parts[1:], " ")

	return func() tea.Msg {
		return CommandResultMsg{
			Command: "hide",
			Success: true,
			Message: pattern,
		}
	}
}

// handleProxy manages proxy commands
func handleProxy(ctx *SlashCommandContext, parts []string) tea.Cmd {
	if len(parts) < 2 {
		ctx.LogStore.Add("system", "System", "📋 Proxy Commands:", false)
		ctx.LogStore.Add("system", "System", "  /proxy <URL>     - Register a URL for proxying", false)
		ctx.LogStore.Add("system", "System", "  /proxy list      - List all active proxy mappings", false)
		ctx.LogStore.Add("system", "System", "Example: /proxy https://stgeorge.sbdev.io", false)
		return nil
	}

	subcommand := parts[1]

	return func() tea.Msg {
		if subcommand == "list" {
			return CommandResultMsg{Command: "proxy", Success: true, Message: "list"}
		}

		// URL to register
		urlStr := strings.Join(parts[1:], " ")
		return CommandResultMsg{Command: "proxy", Success: true, Message: urlStr}
	}
}

// handleToggleProxy toggles proxy mode
func handleToggleProxy(ctx *SlashCommandContext) tea.Cmd {
	return func() tea.Msg {
		return CommandResultMsg{Command: "toggle-proxy", Success: true}
	}
}

// handleTerm shows terminal view
func handleTerm(ctx *SlashCommandContext) tea.Cmd {
	return func() tea.Msg {
		return CommandResultMsg{Command: "term", Success: true, View: "term"}
	}
}

// handleAI starts an AI coder session
func handleAI(ctx *SlashCommandContext, parts []string) tea.Cmd {
	if len(parts) < 2 {
		return logError(ctx, "ai", "Command requires a provider name (e.g., /ai claude, /ai aider)")
	}

	providerName := parts[1]

	return func() tea.Msg {
		return CommandResultMsg{
			Command: "ai",
			Success: true,
			Message: providerName,
			View:    "aicoders",
		}
	}
}

// handleDockerUp starts docker container
func handleDockerUp(ctx *SlashCommandContext, parts []string) tea.Cmd {
	container := ""
	if len(parts) >= 2 {
		container = parts[1]
	}

	return func() tea.Msg {
		var cmdArgs []string
		if container != "" {
			cmdArgs = []string{"start", container}
		} else {
			cmdArgs = []string{"start"}
		}

		process, err := ctx.ProcessManager.StartCommand(
			fmt.Sprintf("docker-up-%s", container),
			"docker",
			cmdArgs,
		)
		if err != nil {
			return logErrorMsg(ctx, "docker-up", fmt.Sprintf("Failed to start docker container: %v", err))
		}

		return logSuccessMsg(ctx, "docker-up", fmt.Sprintf("Started docker container (process ID: %s)", process.ID), "logs")
	}
}

// handleDockerDown stops docker container
func handleDockerDown(ctx *SlashCommandContext, parts []string) tea.Cmd {
	container := ""
	if len(parts) >= 2 {
		container = parts[1]
	}

	return func() tea.Msg {
		var cmdArgs []string
		if container != "" {
			cmdArgs = []string{"stop", container}
		} else {
			cmdArgs = []string{"stop"}
		}

		process, err := ctx.ProcessManager.StartCommand(
			fmt.Sprintf("docker-down-%s", container),
			"docker",
			cmdArgs,
		)
		if err != nil {
			return logErrorMsg(ctx, "docker-down", fmt.Sprintf("Failed to stop docker container: %v", err))
		}

		return logSuccessMsg(ctx, "docker-down", fmt.Sprintf("Stopping docker container (process ID: %s)", process.ID), "processes")
	}
}

// handleDockerBuild builds docker image
func handleDockerBuild(ctx *SlashCommandContext, parts []string) tea.Cmd {
	tag := "latest"
	if len(parts) >= 2 {
		tag = parts[1]
	}

	return func() tea.Msg {
		cmdArgs := []string{"build", "-t", tag, "."}

		process, err := ctx.ProcessManager.StartCommand(
			fmt.Sprintf("docker-build-%s", tag),
			"docker",
			cmdArgs,
		)
		if err != nil {
			return logErrorMsg(ctx, "docker-build", fmt.Sprintf("Failed to start docker build: %v", err))
		}

		return logSuccessMsg(ctx, "docker-build", fmt.Sprintf("Started docker build (process ID: %s)", process.ID), "logs")
	}
}

// handleComposeUp starts compose services
func handleComposeUp(ctx *SlashCommandContext, parts []string) tea.Cmd {
	service := ""
	if len(parts) >= 2 {
		service = parts[1]
	}

	return func() tea.Msg {
		var cmdArgs []string
		if service != "" {
			cmdArgs = []string{"up", "-d", service}
		} else {
			cmdArgs = []string{"up", "-d"}
		}

		process, err := ctx.ProcessManager.StartCommand(
			fmt.Sprintf("compose-up-%s", service),
			"docker-compose",
			cmdArgs,
		)
		if err != nil {
			return logErrorMsg(ctx, "compose-up", fmt.Sprintf("Failed to start compose: %v", err))
		}

		return logSuccessMsg(ctx, "compose-up", fmt.Sprintf("Started compose (process ID: %s)", process.ID), "logs")
	}
}

// handleComposeDown stops compose services
func handleComposeDown(ctx *SlashCommandContext, parts []string) tea.Cmd {
	service := ""
	if len(parts) >= 2 {
		service = parts[1]
	}

	return func() tea.Msg {
		var cmdArgs []string
		if service != "" {
			cmdArgs = []string{"down", service}
		} else {
			cmdArgs = []string{"down"}
		}

		process, err := ctx.ProcessManager.StartCommand(
			fmt.Sprintf("compose-down-%s", service),
			"docker-compose",
			cmdArgs,
		)
		if err != nil {
			return logErrorMsg(ctx, "compose-down", fmt.Sprintf("Failed to stop compose: %v", err))
		}

		return logSuccessMsg(ctx, "compose-down", fmt.Sprintf("Stopping compose (process ID: %s)", process.ID), "processes")
	}
}

// handleUnknown handles unknown commands
func handleUnknown(ctx *SlashCommandContext, command string) tea.Cmd {
	return func() tea.Msg {
		ctx.LogStore.Add("system", "System", fmt.Sprintf("❌ Unknown command: %s", command), true)
		ctx.LogStore.Add("system", "System", "Available commands: /run, /restart, /stop, /clear, /show, /hide, /proxy, /toggle-proxy, /ai, /term, /docker-up, /docker-down, /docker-build, /compose-up, /compose-down, /help, /exit", false)
		return nil
	}
}

// Helper functions for consistent error handling

func logError(ctx *SlashCommandContext, command, message string) tea.Cmd {
	return func() tea.Msg {
		ctx.LogStore.Add("system", "System", fmt.Sprintf("❌ %s: %s", command, message), true)
		return nil
	}
}

func logErrorMsg(ctx *SlashCommandContext, command, message string) CommandResultMsg {
	ctx.LogStore.Add("system", "System", fmt.Sprintf("❌ %s", message), true)
	return CommandResultMsg{
		Command: command,
		Success: false,
		Message: message,
	}
}

func logSuccessMsg(ctx *SlashCommandContext, command, message, view string) CommandResultMsg {
	ctx.LogStore.Add("system", "System", fmt.Sprintf("✅ %s", message), false)
	return CommandResultMsg{
		Command: command,
		Success: true,
		Message: message,
		View:    view,
	}
}
