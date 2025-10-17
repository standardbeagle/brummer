package commands

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
)

// AsyncSlashCommandContext contains all the dependencies needed to handle slash commands with async operations
type AsyncSlashCommandContext struct {
	ProcessManager *process.Manager
	LogStore       *logs.Store
	UpdateChan     chan<- tea.Msg
	CommandFactory  *SafeOperationCommands

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

// HandleAsyncSlashCommand processes slash commands functionally using commands
func HandleAsyncSlashCommand(ctx *AsyncSlashCommandContext, input string) tea.Cmd {
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
		return nil
	}

	command := parts[0]

	switch command {
	case "/run":
		return handleRunCommandAsync(ctx, parts)

	case "/restart":
		return handleRestartCommandAsync(ctx, parts)

	case "/stop":
		return handleStopCommandAsync(ctx, parts)

	case "/clear":
		return handleClearCommandAsync(ctx, parts)

	case "/proxy":
		return handleProxyCommandAsync(ctx, parts)

	case "/toggle-proxy":
		ctx.ToggleProxy()
		return nil

	case "/ai":
		return handleAICommandAsync(ctx, parts)

	case "/term":
		ctx.ShowTerminal()
		return nil

	case "/docker-up":
		return handleDockerUpCommandAsync(ctx, parts)

	case "/docker-down":
		return handleDockerDownCommandAsync(ctx, parts)

	case "/docker-build":
		return handleDockerBuildCommandAsync(ctx, parts)

	case "/compose-up":
		return handleComposeUpCommandAsync(ctx, parts)

	case "/compose-down":
		return handleComposeDownCommandAsync(ctx, parts)

	case "/help":
		*ctx.CurrentView = "help"
		return nil

	case "/exit", "/quit":
		// Send quit message to exit the TUI
		return func() tea.Msg {
			return tea.Quit()
		}

	default:
		// Unknown command - show error
		ctx.LogStore.Add("system", "System", fmt.Sprintf("❌ Unknown command: %s", command), true)
		ctx.LogStore.Add("system", "System", "Available commands: /run, /restart, /stop, /clear, /show, /hide, /proxy, /toggle-proxy, /ai, /term, /docker-up, /docker-down, /docker-build, /compose-up, /compose-down, /help, /exit", false)
		return nil
	}
}

func handleRunCommandAsync(ctx *AsyncSlashCommandContext, parts []string) tea.Cmd {
	if len(parts) < 2 {
		ctx.LogStore.Add("system", "System", "Error: /run command requires a script name", true)
		return nil
	}
	scriptName := parts[1]

	// Create an async command to start the script
	return ctx.CommandFactory.ExecuteWithPanicRecovery(
		fmt.Sprintf("run-script-%s", scriptName),
		fmt.Sprintf("start script '%s'", scriptName),
		false, // success = true
		nil,
	)
}

func handleRestartCommandAsync(ctx *AsyncSlashCommandContext, parts []string) tea.Cmd {
	processName := "all"
	if len(parts) >= 2 {
		processName = parts[1]
	}

	if processName == "all" {
		// Restart all running processes
		return ctx.CommandFactory.ExecuteNoErrorWithPanicRecovery(
			fmt.Sprintf("restart-all-%d", time.Now().UnixNano()),
			fmt.Sprintf("restart all processes"),
			false, // shouldPanic = false
			nil,
		)
	} else {
		// Restart specific process
		return ctx.CommandFactory.ExecuteNoErrorWithPanicRecovery(
			fmt.Sprintf("restart-%s-%d", processName, time.Now().UnixNano()),
			fmt.Sprintf("restart process '%s'", processName),
			false, // shouldPanic = false
			nil,
		)
	}
}

func handleStopCommandAsync(ctx *AsyncSlashCommandContext, parts []string) tea.Cmd {
	processName := "all"
	if len(parts) >= 2 {
		processName = parts[1]
	}

	if processName == "all" {
		// Stop all running processes
		return ctx.CommandFactory.ExecuteNoErrorWithPanicRecovery(
			fmt.Sprintf("stop-all-%d", time.Now().UnixNano()),
			fmt.Sprintf("stop all processes"),
			false, // shouldPanic = false
			nil,
		)
	} else {
		// Stop specific process
		return ctx.CommandFactory.ExecuteNoErrorWithPanicRecovery(
			fmt.Sprintf("stop-%s-%d", processName, time.Now().UnixNano()),
			fmt.Sprintf("stop process '%s'", processName),
			false, // shouldPanic = false
			nil,
		)
	}
}

func handleClearCommandAsync(ctx *AsyncSlashCommandContext, parts []string) tea.Cmd {
	target := "all"
	if len(parts) >= 2 {
		target = parts[1]
	}

	// Create an async command to clear logs
	return ctx.CommandFactory.SafeAsyncOperation(
		fmt.Sprintf("clear-%s-%d", target, time.Now().UnixNano()),
		fmt.Sprintf("clear logs for target '%s'", target),
		5*time.Second,
		nil,
	)
}

func handleProxyCommandAsync(ctx *AsyncSlashCommandContext, parts []string) tea.Cmd {
	if len(parts) < 2 {
		// Show usage help
		ctx.LogStore.Add("system", "System", "📋 Proxy Commands:", false)
		ctx.LogStore.Add("system", "System", "  /proxy <URL>     - Register a URL for proxying", false)
		ctx.LogStore.Add("system", "System", "  /proxy list      - List all active proxy mappings", false)
		ctx.LogStore.Add("system", "System", "Example: /proxy https://stgeorge.sbdev.io", false)
		return nil
	}

	subcommand := parts[1]
	if subcommand == "list" {
		ctx.ListProxies()
		return nil
	} else {
		// Treat the rest as a URL
		urlStr := strings.Join(parts[1:], " ")
		ctx.SetProxyURL(urlStr)
		return nil
	}
}

func handleAICommandAsync(ctx *AsyncSlashCommandContext, parts []string) tea.Cmd {
	if len(parts) < 2 {
		ctx.LogStore.Add("system", "System", "Error: /ai command requires a provider name", true)
		return nil
	}
	providerName := parts[1]
	ctx.StartAICoder(providerName)
	// Switch to AI Coders view
	*ctx.CurrentView = "ai-coders"

	return nil
}

func handleDockerUpCommandAsync(ctx *AsyncSlashCommandContext, parts []string) tea.Cmd {
	container := ""
	if len(parts) >= 2 {
		container = parts[1]
	}

	// Create an async command to run docker up
	return ctx.CommandFactory.ExecuteWithPanicRecovery(
		fmt.Sprintf("docker-up-%s-%d", container, time.Now().UnixNano()),
		fmt.Sprintf("docker up '%s'", container),
		false, // success = true
		nil,
	)
}

func handleDockerDownCommandAsync(ctx *AsyncSlashCommandContext, parts []string) tea.Cmd {
	container := ""
	if len(parts) >= 2 {
		container = parts[1]
	}

	// Create an async command to run docker down
	return ctx.CommandFactory.ExecuteWithPanicRecovery(
		fmt.Sprintf("docker-down-%s-%d", container, time.Now().UnixNano()),
		fmt.Sprintf("docker down '%s'", container),
		false, // success = true
		nil,
	)
}

func handleDockerBuildCommandAsync(ctx *AsyncSlashCommandContext, parts []string) tea.Cmd {
	tag := "latest"
	if len(parts) >= 2 {
		tag = parts[1]
	}

	// Create an async command to run docker build
	return ctx.CommandFactory.ExecuteWithPanicRecovery(
		fmt.Sprintf("docker-build-%s-%d", tag, time.Now().UnixNano()),
		fmt.Sprintf("docker build '%s'", tag),
		false, // success = true
		nil,
	)
}

func handleComposeUpCommandAsync(ctx *AsyncSlashCommandContext, parts []string) tea.Cmd {
	service := ""
	if len(parts) >= 2 {
		service = parts[1]
	}

	// Create an async command to run compose up
	return ctx.CommandFactory.ExecuteWithPanicRecovery(
		fmt.Sprintf("compose-up-%s-%d", service, time.Now().UnixNano()),
		fmt.Sprintf("compose up '%s'", service),
		false, // success = true
		nil,
	)
}

func handleComposeDownCommandAsync(ctx *AsyncSlashCommandContext, parts []string) tea.Cmd {
	service := ""
	if len(parts) >= 2 {
		service = parts[1]
	}

	// Create an async command to run compose down
	return ctx.CommandFactory.ExecuteWithPanicRecovery(
		fmt.Sprintf("compose-down-%s-%d", service, time.Now().UnixNano()),
		fmt.Sprintf("compose down '%s'", service),
		false, // success = true
		nil,
	)
}

// Helper function to create async slash command context
func NewAsyncSlashCommandContext(
	processManager *process.Manager,
	logStore *logs.Store,
	updateChan chan<- tea.Msg,
	updateLogsView func(),
	clearLogs func(target string),
	setProxyURL func(urlStr string),
	listProxies func(),
	toggleProxy func(),
	startAICoder func(providerName string),
	showTerminal func(),
) *AsyncSlashCommandContext {
	return &AsyncSlashCommandContext{
		ProcessManager: processManager,
		LogStore:       logStore,
		UpdateChan:     updateChan,
		CommandFactory:  NewSafeOperationCommands(),
		ShowPattern:   new(string),
		HidePattern:   new(string),
		SearchResults: new([]logs.LogEntry),
		CurrentView:   new(string),
		UpdateLogsView: updateLogsView,
		ClearLogs:      clearLogs,
		SetProxyURL:    setProxyURL,
		ListProxies:    listProxies,
		ToggleProxy:    toggleProxy,
		StartAICoder:   startAICoder,
		ShowTerminal:   showTerminal,
	}
}