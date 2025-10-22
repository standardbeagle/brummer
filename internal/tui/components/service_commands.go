package components

import (
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/domain/processes"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// GetProcessCommands returns commands exported by ProcessManager
func GetProcessCommands(processMgr *processes.Manager) []messages.Command {
	if processMgr == nil {
		return nil
	}

	return []messages.Command{
		{
			Name:        "run",
			Description: "Start a process from package.json scripts",
			Usage:       "/run <script>",
			Category:    "Process",
			Arguments: []CommandArgument{
				{
					Name:        "script",
					Description: "Script name from package.json",
					Required:    true,
					Suggestions: getScriptNames(processMgr),
				},
			},
		},
		{
			Name:        "stop",
			Description: "Stop a running process",
			Usage:       "/stop <script>",
			Category:    "Process",
			Arguments: []CommandArgument{
				{
					Name:        "script",
					Description: "Process to stop",
					Required:    true,
					Suggestions: getRunningProcessNames(processMgr),
				},
			},
		},
		{
			Name:        "restart",
			Description: "Restart a running process",
			Usage:       "/restart <script>",
			Category:    "Process",
			Arguments: []CommandArgument{
				{
					Name:        "script",
					Description: "Process to restart",
					Required:    true,
					Suggestions: getRunningProcessNames(processMgr),
				},
			},
		},
	}
}

// GetLogCommands returns commands exported by LogStore
func GetLogCommands(logStore *logs.Store) []messages.Command {
	if logStore == nil {
		return nil
	}

	return []messages.Command{
		{
			Name:        "show",
			Description: "Filter logs to show only lines matching pattern",
			Usage:       "/show <pattern>",
			Category:    "Logs",
			Arguments: []CommandArgument{
				{
					Name:        "pattern",
					Description: "Pattern to match (supports regex)",
					Required:    true,
					Suggestions: nil, // User enters custom pattern
				},
			},
		},
		{
			Name:        "hide",
			Description: "Filter logs to hide lines matching pattern",
			Usage:       "/hide <pattern>",
			Category:    "Logs",
			Arguments: []CommandArgument{
				{
					Name:        "pattern",
					Description: "Pattern to match (supports regex)",
					Required:    true,
					Suggestions: nil,
				},
			},
		},
		{
			Name:        "clear",
			Description: "Clear all logs or logs from specific process",
			Usage:       "/clear [process]",
			Category:    "Logs",
			Arguments: []CommandArgument{
				{
					Name:        "process",
					Description: "Process name (or 'all' for everything)",
					Required:    false,
					Suggestions: []string{"all"},
				},
			},
		},
	}
}

// GetProxyCommands returns commands exported by ProxyServer
func GetProxyCommands(proxyServer *proxy.Server) []messages.Command {
	if proxyServer == nil {
		return nil
	}

	return []messages.Command{
		{
			Name:        "proxy",
			Description: "Register a URL for proxying or list active proxies",
			Usage:       "/proxy <URL> or /proxy list",
			Category:    "Proxy",
			Arguments: []CommandArgument{
				{
					Name:        "url",
					Description: "URL to proxy or 'list' to show all",
					Required:    true,
					Suggestions: []string{"list"},
				},
			},
		},
		{
			Name:        "toggle-proxy",
			Description: "Toggle proxy mode between reverse and full",
			Usage:       "/toggle-proxy",
			Category:    "Proxy",
			Arguments:   nil,
		},
	}
}

// GetAICommands returns commands for AI coder sessions
// Note: AI coder manager is not directly passed as it's internal to TUI
func GetAICommands(aiProviders []string) []messages.Command {
	return []messages.Command{
		{
			Name:        "ai",
			Description: "Start an AI coder session",
			Usage:       "/ai <provider> [task]",
			Category:    "AI",
			Arguments: []CommandArgument{
				{
					Name:        "provider",
					Description: "AI provider name",
					Required:    true,
					Suggestions: aiProviders,
				},
				{
					Name:        "task",
					Description: "Optional task description",
					Required:    false,
					Suggestions: nil,
				},
			},
		},
	}
}

// GetDockerCommands returns Docker-related commands
func GetDockerCommands() []messages.Command {
	return []messages.Command{
		{
			Name:        "docker-up",
			Description: "Start Docker containers",
			Usage:       "/docker-up [service]",
			Category:    "Docker",
			Arguments: []CommandArgument{
				{
					Name:        "service",
					Description: "Service name (optional)",
					Required:    false,
					Suggestions: nil,
				},
			},
		},
		{
			Name:        "docker-down",
			Description: "Stop Docker containers",
			Usage:       "/docker-down [service]",
			Category:    "Docker",
			Arguments: []CommandArgument{
				{
					Name:        "service",
					Description: "Service name (optional)",
					Required:    false,
					Suggestions: nil,
				},
			},
		},
		{
			Name:        "docker-build",
			Description: "Build Docker images",
			Usage:       "/docker-build [service]",
			Category:    "Docker",
			Arguments: []CommandArgument{
				{
					Name:        "service",
					Description: "Service name (optional)",
					Required:    false,
					Suggestions: nil,
				},
			},
		},
		{
			Name:        "compose-up",
			Description: "Start docker-compose services",
			Usage:       "/compose-up [service]",
			Category:    "Docker",
			Arguments: []CommandArgument{
				{
					Name:        "service",
					Description: "Service name (optional)",
					Required:    false,
					Suggestions: nil,
				},
			},
		},
		{
			Name:        "compose-down",
			Description: "Stop docker-compose services",
			Usage:       "/compose-down",
			Category:    "Docker",
			Arguments:   nil,
		},
	}
}

// Helper type for command arguments (local to this file)
type CommandArgument = messages.CommandArgument

// getScriptNames returns available script names from ProcessManager
func getScriptNames(processMgr *processes.Manager) []string {
	scripts := processMgr.GetScripts()
	names := make([]string, 0, len(scripts))
	for name := range scripts {
		names = append(names, name)
	}
	return names
}

// getRunningProcessNames returns names of currently running processes
func getRunningProcessNames(processMgr *processes.Manager) []string {
	allProcesses := processMgr.GetAllProcesses()
	names := make([]string, 0, len(allProcesses))
	for _, proc := range allProcesses {
		if proc.GetStatus() == processes.StatusRunning {
			names = append(names, proc.Name)
		}
	}
	return names
}
