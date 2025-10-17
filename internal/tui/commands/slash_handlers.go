package commands

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
)

// SlashCommandContext contains all the dependencies needed to handle slash commands
type SlashCommandContext struct {
	ProcessManager *process.Manager
	LogStore       *logs.Store
	UpdateChan     chan<- tea.Msg

	// Current state
	ShowPattern   *string
	HidePattern   *string
	SearchResults *[]logs.LogEntry
	CurrentView   *string

	// Callbacks
	UpdateLogsView func()
	ClearLogs      func(target string)
	SetProxyURL    func(urlStr string)
	ListProxies    func()
	ToggleProxy    func()
	StartAICoder   func(providerName string)
	ShowTerminal   func()
}

// HandleSlashCommand processes slash commands functionally
func HandleSlashCommand(ctx *SlashCommandContext, input string) {
	// Clear previous search results and filters
	*ctx.SearchResults = nil
	*ctx.ShowPattern = ""
	*ctx.HidePattern = ""

	// Parse the command
	input = strings.TrimSpace(input)

	// If the command doesn't start with /, add it
	if !strings.HasPrefix(input, "/") {
		input = "/" + input
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	command := parts[0]

	switch command {
	case "/show":
		if len(parts) < 2 {
			return
		}
		*ctx.ShowPattern = strings.Join(parts[1:], " ")

	case "/hide":
		if len(parts) < 2 {
			return
		}
		*ctx.HidePattern = strings.Join(parts[1:], " ")

	case "/run":
		handleRunCommand(ctx, parts)

	case "/restart":
		handleRestartCommand(ctx, parts)

	case "/stop":
		handleStopCommand(ctx, parts)

	case "/clear":
		target := "all"
		if len(parts) >= 2 {
			target = parts[1]
		}
		ctx.ClearLogs(target)

	case "/proxy":
		if len(parts) < 2 {
			// Show usage help
			ctx.LogStore.Add("system", "System", "📋 Proxy Commands:", false)
			ctx.LogStore.Add("system", "System", "  /proxy <URL>     - Register a URL for proxying", false)
			ctx.LogStore.Add("system", "System", "  /proxy list      - List all active proxy mappings", false)
			ctx.LogStore.Add("system", "System", "Example: /proxy https://stgeorge.sbdev.io", false)
			return
		}

		subcommand := parts[1]
		if subcommand == "list" {
			ctx.ListProxies()
		} else {
			// Treat the rest as a URL
			urlStr := strings.Join(parts[1:], " ")
			ctx.SetProxyURL(urlStr)
		}

	case "/toggle-proxy":
		ctx.ToggleProxy()

	case "/ai":
		if len(parts) < 2 {
			ctx.LogStore.Add("system", "System", "Error: /ai command requires a provider name", true)
			return
		}
		providerName := parts[1]
		ctx.StartAICoder(providerName)
		// Switch to AI Coders view
		*ctx.CurrentView = "ai-coders"

	case "/term":
		ctx.ShowTerminal()

	case "/docker-up":
		handleDockerUpCommand(ctx, parts)

	case "/docker-down":
		handleDockerDownCommand(ctx, parts)

	case "/docker-build":
		handleDockerBuildCommand(ctx, parts)

	case "/compose-up":
		handleComposeUpCommand(ctx, parts)

	case "/compose-down":
		handleComposeDownCommand(ctx, parts)

	case "/help":
		*ctx.CurrentView = "help"

	case "/exit", "/quit":
		// Send quit message to exit the TUI
		ctx.UpdateChan <- tea.Quit()
		ctx.LogStore.Add("system", "System", "👋 Exiting Brummer...", false)

	default:
		// Unknown command - show error
		ctx.LogStore.Add("system", "System", fmt.Sprintf("❌ Unknown command: %s", command), true)
		ctx.LogStore.Add("system", "System", "Available commands: /run, /restart, /stop, /clear, /show, /hide, /proxy, /toggle-proxy, /ai, /term, /docker-up, /docker-down, /docker-build, /compose-up, /compose-down, /help, /exit", false)
	}
}

func handleRunCommand(ctx *SlashCommandContext, parts []string) {
	if len(parts) < 2 {
		ctx.LogStore.Add("system", "System", "Error: /run command requires a script name", true)
		return
	}

	// Create async context for command handling
	asyncCtx := NewAsyncSlashCommandContext(
		ctx.ProcessManager,
		ctx.LogStore,
		ctx.UpdateChan,
		ctx.UpdateLogsView,
		ctx.ClearLogs,
		ctx.SetProxyURL,
		ctx.ListProxies,
		ctx.ToggleProxy,
		ctx.StartAICoder,
		ctx.ShowTerminal,
	)

	// Execute the script as command
	// TODO: Replace with proper command pattern for async execution
	// For now, execute synchronously to avoid goroutines
	cmd := handleRunCommandAsync(asyncCtx, parts)
	if cmd != nil {
		msg := cmd()
		ctx.UpdateChan <- msg
	}

	// Switch to logs view immediately
	*ctx.CurrentView = "logs"
}

func handleRestartCommand(ctx *SlashCommandContext, parts []string) {
	// Create async context for command handling
	asyncCtx := NewAsyncSlashCommandContext(
		ctx.ProcessManager,
		ctx.LogStore,
		ctx.UpdateChan,
		ctx.UpdateLogsView,
		ctx.ClearLogs,
		ctx.SetProxyURL,
		ctx.ListProxies,
		ctx.ToggleProxy,
		ctx.StartAICoder,
		ctx.ShowTerminal,
	)

	// Execute the restart as command
	// TODO: Replace with proper command pattern for async execution
	// For now, execute synchronously to avoid goroutines
	cmd := handleRestartCommandAsync(asyncCtx, parts)
	if cmd != nil {
		msg := cmd()
		ctx.UpdateChan <- msg
	}

	*ctx.CurrentView = "processes"
}

func handleStopCommand(ctx *SlashCommandContext, parts []string) {
	// Create async context for command handling
	asyncCtx := NewAsyncSlashCommandContext(
		ctx.ProcessManager,
		ctx.LogStore,
		ctx.UpdateChan,
		ctx.UpdateLogsView,
		ctx.ClearLogs,
		ctx.SetProxyURL,
		ctx.ListProxies,
		ctx.ToggleProxy,
		ctx.StartAICoder,
		ctx.ShowTerminal,
	)

	// Execute the stop as command
	// TODO: Replace with proper command pattern for async execution
	// For now, execute synchronously to avoid goroutines
	cmd := handleStopCommandAsync(asyncCtx, parts)
	if cmd != nil {
		msg := cmd()
		ctx.UpdateChan <- msg
	}

	*ctx.CurrentView = "processes"
}

func handleDockerUpCommand(ctx *SlashCommandContext, parts []string) {
	// Create async context for command handling
	asyncCtx := NewAsyncSlashCommandContext(
		ctx.ProcessManager,
		ctx.LogStore,
		ctx.UpdateChan,
		ctx.UpdateLogsView,
		ctx.ClearLogs,
		ctx.SetProxyURL,
		ctx.ListProxies,
		ctx.ToggleProxy,
		ctx.StartAICoder,
		ctx.ShowTerminal,
	)

	// Execute the docker up as command
	// TODO: Replace with proper command pattern for async execution
	// For now, execute synchronously to avoid goroutines
	cmd := handleDockerUpCommandAsync(asyncCtx, parts)
	if cmd != nil {
		msg := cmd()
		ctx.UpdateChan <- msg
	}

	*ctx.CurrentView = "logs"
}

func handleDockerDownCommand(ctx *SlashCommandContext, parts []string) {
	// Create async context for command handling
	asyncCtx := NewAsyncSlashCommandContext(
		ctx.ProcessManager,
		ctx.LogStore,
		ctx.UpdateChan,
		ctx.UpdateLogsView,
		ctx.ClearLogs,
		ctx.SetProxyURL,
		ctx.ListProxies,
		ctx.ToggleProxy,
		ctx.StartAICoder,
		ctx.ShowTerminal,
	)

	// Execute the docker down as command
	// TODO: Replace with proper command pattern for async execution
	// For now, execute synchronously to avoid goroutines
	cmd := handleDockerDownCommandAsync(asyncCtx, parts)
	if cmd != nil {
		msg := cmd()
		ctx.UpdateChan <- msg
	}

	*ctx.CurrentView = "processes"
}

func handleDockerBuildCommand(ctx *SlashCommandContext, parts []string) {
	// Create async context for command handling
	asyncCtx := NewAsyncSlashCommandContext(
		ctx.ProcessManager,
		ctx.LogStore,
		ctx.UpdateChan,
		ctx.UpdateLogsView,
		ctx.ClearLogs,
		ctx.SetProxyURL,
		ctx.ListProxies,
		ctx.ToggleProxy,
		ctx.StartAICoder,
		ctx.ShowTerminal,
	)

	// Execute the docker build as command
	// TODO: Replace with proper command pattern for async execution
	// For now, execute synchronously to avoid goroutines
	cmd := handleDockerBuildCommandAsync(asyncCtx, parts)
	if cmd != nil {
		msg := cmd()
		ctx.UpdateChan <- msg
	}

	*ctx.CurrentView = "logs"
}

func handleComposeUpCommand(ctx *SlashCommandContext, parts []string) {
	// Create async context for command handling
	asyncCtx := NewAsyncSlashCommandContext(
		ctx.ProcessManager,
		ctx.LogStore,
		ctx.UpdateChan,
		ctx.UpdateLogsView,
		ctx.ClearLogs,
		ctx.SetProxyURL,
		ctx.ListProxies,
		ctx.ToggleProxy,
		ctx.StartAICoder,
		ctx.ShowTerminal,
	)

	// Execute the compose up as command
	// TODO: Replace with proper command pattern for async execution
	// For now, execute synchronously to avoid goroutines
	cmd := handleComposeUpCommandAsync(asyncCtx, parts)
	if cmd != nil {
		msg := cmd()
		ctx.UpdateChan <- msg
	}

	*ctx.CurrentView = "logs"
}

func handleComposeDownCommand(ctx *SlashCommandContext, parts []string) {
	// Create async context for command handling
	asyncCtx := NewAsyncSlashCommandContext(
		ctx.ProcessManager,
		ctx.LogStore,
		ctx.UpdateChan,
		ctx.UpdateLogsView,
		ctx.ClearLogs,
		ctx.SetProxyURL,
		ctx.ListProxies,
		ctx.ToggleProxy,
		ctx.StartAICoder,
		ctx.ShowTerminal,
	)

	// Execute the compose down as command
	// TODO: Replace with proper command pattern for async execution
	// For now, execute synchronously to avoid goroutines
	cmd := handleComposeDownCommandAsync(asyncCtx, parts)
	if cmd != nil {
		msg := cmd()
		ctx.UpdateChan <- msg
	}

	*ctx.CurrentView = "processes"
}

