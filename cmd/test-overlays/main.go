package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/pkg/events"
)

type model struct {
	shell          *components.IntegratedShell
	width          int
	height         int
	lastCommand    string
	lastCommandMsg messages.CommandExecuteMsg
}

func initialModel() model {
	// Create event bus for service coordination
	eventBus := events.NewEventBus()

	// Create real services for testing
	processMgr, err := process.NewManager(".", eventBus, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create process manager: %v\n", err)
		processMgr = nil
	}

	logStore := logs.NewStore(10000, eventBus)
	proxyServer := proxy.NewServer(20000, eventBus)

	// Create IntegratedShell
	shell := components.NewIntegratedShell()

	// Create and set a logs view that uses the real log store
	logsView := components.NewLogsView(logStore)
	shell.SetLogsView(logsView)

	// Register commands with real service references
	registry := shell.GetCommandRegistry()
	registry.RegisterMultiple(components.GetProcessCommands(processMgr))
	registry.RegisterMultiple(components.GetLogCommands(logStore))
	registry.RegisterMultiple(components.GetProxyCommands(proxyServer))
	registry.RegisterMultiple(components.GetAICommands([]string{"claude", "gpt4", "mock"}))
	registry.RegisterMultiple(components.GetDockerCommands())

	// Create command executor with real services (aiCoderManager can be nil for testing)
	executor := components.NewCommandExecutor(processMgr, logStore, proxyServer, registry, nil)
	shell.SetCommandExecutor(executor)

	// Create and set command input
	commandInput := components.NewCommandInputComponent(registry)
	shell.SetCommandInput(commandInput)

	// Create and set help overlay
	helpOverlay := components.NewHelpOverlayComponent()
	shell.SetHelpOverlay(helpOverlay)

	return model{
		shell: shell,
	}
}

func (m model) Init() tea.Cmd {
	return m.shell.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "q" && !isOverlayActive(m.shell) {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case messages.CommandExecuteMsg:
		// Capture executed commands for display
		m.lastCommandMsg = msg
		args := strings.Join(msg.Arguments, " ")
		if args != "" {
			m.lastCommand = fmt.Sprintf("/%s %s", msg.CommandName, args)
		} else {
			m.lastCommand = fmt.Sprintf("/%s", msg.CommandName)
		}
	}

	// Forward to shell
	_, cmd := m.shell.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	view := m.shell.View()

	// Add instructions and last command at the bottom
	var statusLine string
	if m.lastCommand != "" {
		statusLine = fmt.Sprintf("\nLast command: %s", m.lastCommand)
	}

	instructions := "\n\n[/] Commands | [?] Help | [Ctrl+C] Quit" + statusLine

	return view + instructions
}

func isOverlayActive(shell *components.IntegratedShell) bool {
	// This is a helper to check if we should allow 'q' to quit
	// We don't have direct access, so we'll just return false for now
	return false
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
