package services

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ServiceCommandExecutor executes commands using service interfaces
type ServiceCommandExecutor struct {
	processService ProcessService
	logsService    LogsService
	proxyService   ProxyService
}

// NewServiceCommandExecutor creates an executor backed by services
func NewServiceCommandExecutor(
	processService ProcessService,
	logsService LogsService,
	proxyService ProxyService,
) *ServiceCommandExecutor {
	return &ServiceCommandExecutor{
		processService: processService,
		logsService:    logsService,
		proxyService:   proxyService,
	}
}

// Execute runs a command using the appropriate service
func (e *ServiceCommandExecutor) Execute(command string) tea.Cmd {
	// Remove leading slash if present
	cmd := strings.TrimPrefix(command, "/")
	parts := strings.Fields(cmd)
	
	if len(parts) == 0 {
		return nil
	}
	
	action := parts[0]
	args := parts[1:]
	
	switch action {
	case "start":
		if len(args) > 0 && e.processService != nil {
			return e.processService.Start(args[0])
		}
		return e.errorCmd("Specify script to start")
		
	case "stop":
		if len(args) > 0 && e.processService != nil {
			return e.processService.Stop(args[0])
		}
		return e.errorCmd("Specify process to stop")
		
	case "restart":
		if len(args) > 0 && e.processService != nil {
			return e.processService.Restart(args[0])
		}
		return e.errorCmd("Specify process to restart")
		
	case "proxy":
		if e.proxyService == nil {
			return e.errorCmd("Proxy service not available")
		}
		
		if len(args) == 0 {
			return e.proxyService.Toggle()
		}
		
		switch args[0] {
		case "on":
			return e.proxyService.SetEnabled(true)
		case "off":
			return e.proxyService.SetEnabled(false)
		case "toggle":
			return e.proxyService.Toggle()
		default:
			return e.errorCmd("Unknown proxy command")
		}
		
	case "clear":
		if e.logsService != nil {
			processID := ""
			if len(args) > 0 {
				processID = args[0]
			}
			return e.logsService.Clear(processID)
		}
		return e.errorCmd("Logs service not available")
		
	case "logs", "processes", "errors", "urls", "web", "ai", "settings":
		// Navigation commands - return navigation message
		return func() tea.Msg {
			return NavigateToViewMsg{View: action}
		}
		
	case "quit", "exit":
		return tea.Quit
		
	case "help", "?":
		return func() tea.Msg {
			return ShowHelpMsg{}
		}
		
	default:
		// Try to start as a script/command
		if e.processService != nil {
			return e.processService.Start(action)
		}
		return e.errorCmd("Unknown command: " + action)
	}
}

func (e *ServiceCommandExecutor) errorCmd(message string) tea.Cmd {
	return func() tea.Msg {
		return CommandResultMsg{
			Success: false,
			Message: message,
		}
	}
}

// Message types used by command executor
type NavigateToViewMsg struct {
	View string
}

type ShowHelpMsg struct{}

type CommandResultMsg struct {
	Success bool
	Message string
}