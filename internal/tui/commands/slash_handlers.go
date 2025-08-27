package commands

import (
	"fmt"
	"strings"
	"time"

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

	default:
		// Unknown command - show error
		ctx.LogStore.Add("system", "System", fmt.Sprintf("❌ Unknown command: %s", command), true)
		ctx.LogStore.Add("system", "System", "Available commands: /run, /restart, /stop, /clear, /show, /hide, /proxy, /toggle-proxy, /ai, /term, /docker-up, /docker-down, /docker-build, /compose-up, /compose-down, /help", false)
	}
}

func handleRunCommand(ctx *SlashCommandContext, parts []string) {
	if len(parts) < 2 {
		ctx.LogStore.Add("system", "System", "Error: /run command requires a script name", true)
		return
	}
	scriptName := parts[1]

	// Execute the script
	errorHandler := NewStandardErrorHandler(ctx.LogStore, ctx.UpdateChan)
	SafeGoroutine(
		fmt.Sprintf("start script '%s'", scriptName),
		func() error {
			_, err := ctx.ProcessManager.StartScript(scriptName)
			if err == nil {
				ctx.UpdateChan <- processUpdateMsg{}
			}
			return err
		},
		func(err error) {
			errorCtx := ScriptStartContext(scriptName, "Slash Command", ctx.LogStore, ctx.UpdateChan)
			errorHandler.HandleError(err, errorCtx)
		},
	)

	// Switch to logs view immediately
	*ctx.CurrentView = "logs"
}

func handleRestartCommand(ctx *SlashCommandContext, parts []string) {
	processName := "all"
	if len(parts) >= 2 {
		processName = parts[1]
	}

	if processName == "all" {
		// Restart all running processes
		SafeGoroutineNoError(
			"restart all processes",
			func() {
				processes := ctx.ProcessManager.GetAllProcesses()
				restarted := 0
				for _, proc := range processes {
					if proc.GetStatus() == process.StatusRunning {
						// Stop the process and wait for termination
						timeout := 5 * time.Second
						if err := ctx.ProcessManager.StopProcessAndWait(proc.ID, timeout); err != nil {
							ctx.LogStore.Add("system", "System", fmt.Sprintf("Error stopping process %s: %v", proc.Name, err), true)
							continue
						}
						// Start it again
						_, err := ctx.ProcessManager.StartScript(proc.Name)
						if err != nil {
							ctx.LogStore.Add("system", "System", fmt.Sprintf("Error restarting script %s: %v", proc.Name, err), true)
						} else {
							restarted++
						}
					}
				}
				ctx.LogStore.Add("system", "System", fmt.Sprintf("🔄 Restarted %d processes", restarted), false)
				ctx.UpdateChan <- processUpdateMsg{}
			},
			func(err error) {
				ctx.LogStore.Add("system", "System", fmt.Sprintf("Critical error during restart all: %v", err), true)
				ctx.UpdateChan <- logUpdateMsg{}
			},
		)
	} else {
		// Restart specific process
		SafeGoroutineNoError(
			fmt.Sprintf("restart process '%s'", processName),
			func() {
				// Find the process
				var targetProc *process.Process
				for _, proc := range ctx.ProcessManager.GetAllProcesses() {
					if proc.Name == processName && proc.GetStatus() == process.StatusRunning {
						targetProc = proc
						break
					}
				}

				if targetProc == nil {
					ctx.LogStore.Add("system", "System", fmt.Sprintf("Process '%s' is not running", processName), true)
					ctx.UpdateChan <- logUpdateMsg{}
					return
				}

				// Stop and restart the process
				timeout := 5 * time.Second
				if err := ctx.ProcessManager.StopProcessAndWait(targetProc.ID, timeout); err != nil {
					errorMsg := fmt.Sprintf("Error stopping process '%s' (ID: %s) during restart: %v", processName, targetProc.ID, err)
					ctx.LogStore.Add("system", "System", errorMsg, true)
					ctx.UpdateChan <- logUpdateMsg{}
					return
				}

				_, err := ctx.ProcessManager.StartScript(processName)
				if err != nil {
					errorMsg := fmt.Sprintf("Error restarting script '%s' after successful stop: %v", processName, err)
					ctx.LogStore.Add("system", "System", errorMsg, true)
				} else {
					ctx.LogStore.Add("system", "System", fmt.Sprintf("🔄 Restarted process: %s", processName), false)
				}
				ctx.UpdateChan <- processUpdateMsg{}
			},
			func(err error) {
				ctx.LogStore.Add("system", "System", fmt.Sprintf("Critical error during restart of '%s': %v", processName, err), true)
				ctx.UpdateChan <- logUpdateMsg{}
			},
		)
	}
	*ctx.CurrentView = "processes"
}

func handleStopCommand(ctx *SlashCommandContext, parts []string) {
	processName := "all"
	if len(parts) >= 2 {
		processName = parts[1]
	}

	if processName == "all" {
		// Stop all running processes
		SafeGoroutineNoError(
			"stop all processes",
			func() {
				processes := ctx.ProcessManager.GetAllProcesses()
				stopped := 0
				for _, proc := range processes {
					if proc.GetStatus() == process.StatusRunning {
						if err := ctx.ProcessManager.StopProcess(proc.ID); err != nil {
							ctx.LogStore.Add("system", "System", fmt.Sprintf("Error stopping process %s: %v", proc.Name, err), true)
						} else {
							stopped++
						}
					}
				}
				ctx.LogStore.Add("system", "System", fmt.Sprintf("⏹️ Stopped %d processes", stopped), false)
				ctx.UpdateChan <- processUpdateMsg{}
			},
			func(err error) {
				ctx.LogStore.Add("system", "System", fmt.Sprintf("Critical error during stop all: %v", err), true)
				ctx.UpdateChan <- logUpdateMsg{}
			},
		)
	} else {
		// Stop specific process
		SafeGoroutineNoError(
			fmt.Sprintf("stop process '%s'", processName),
			func() {
				// Find the process
				var targetProc *process.Process
				for _, proc := range ctx.ProcessManager.GetAllProcesses() {
					if proc.Name == processName && proc.GetStatus() == process.StatusRunning {
						targetProc = proc
						break
					}
				}

				if targetProc == nil {
					ctx.LogStore.Add("system", "System", fmt.Sprintf("Process '%s' is not running", processName), true)
					ctx.UpdateChan <- logUpdateMsg{}
					return
				}

				if err := ctx.ProcessManager.StopProcess(targetProc.ID); err != nil {
					ctx.LogStore.Add("system", "System", fmt.Sprintf("Error stopping process %s: %v", processName, err), true)
				} else {
					ctx.LogStore.Add("system", "System", fmt.Sprintf("⏹️ Stopped process: %s", processName), false)
				}
				ctx.UpdateChan <- processUpdateMsg{}
			},
			func(err error) {
				ctx.LogStore.Add("system", "System", fmt.Sprintf("Critical error during stop of '%s': %v", processName, err), true)
				ctx.UpdateChan <- logUpdateMsg{}
			},
		)
	}
	*ctx.CurrentView = "processes"
}

func handleDockerUpCommand(ctx *SlashCommandContext, parts []string) {
	container := ""
	if len(parts) >= 2 {
		container = parts[1]
	}

	errorHandler := NewStandardErrorHandler(ctx.LogStore, ctx.UpdateChan)
	SafeGoroutine(
		fmt.Sprintf("docker up '%s'", container),
		func() error {
			args := []string{"run", "-d"}
			if container != "" {
				args = append(args, container)
			} else {
				// Default behavior - try to find and run first available container
				args = append(args, "--rm", "-it", "ubuntu:latest")
			}
			_, err := ctx.ProcessManager.StartCommand("docker", "docker", args)
			if err == nil {
				ctx.UpdateChan <- processUpdateMsg{}
			}
			return err
		},
		func(err error) {
			errorCtx := ScriptStartContext("docker", "Docker Up", ctx.LogStore, ctx.UpdateChan)
			errorHandler.HandleError(err, errorCtx)
		},
	)

	*ctx.CurrentView = "logs"
}

func handleDockerDownCommand(ctx *SlashCommandContext, parts []string) {
	container := ""
	if len(parts) >= 2 {
		container = parts[1]
	}

	errorHandler := NewStandardErrorHandler(ctx.LogStore, ctx.UpdateChan)
	SafeGoroutine(
		fmt.Sprintf("docker down '%s'", container),
		func() error {
			args := []string{"stop"}
			if container != "" {
				args = append(args, container)
			} else {
				// Stop all containers
				args = append(args, "$(docker ps -q)")
			}
			_, err := ctx.ProcessManager.StartCommand("docker", "docker", args)
			if err == nil {
				ctx.UpdateChan <- processUpdateMsg{}
			}
			return err
		},
		func(err error) {
			errorCtx := ScriptStartContext("docker", "Docker Down", ctx.LogStore, ctx.UpdateChan)
			errorHandler.HandleError(err, errorCtx)
		},
	)

	*ctx.CurrentView = "processes"
}

func handleDockerBuildCommand(ctx *SlashCommandContext, parts []string) {
	tag := "latest"
	if len(parts) >= 2 {
		tag = parts[1]
	}

	errorHandler := NewStandardErrorHandler(ctx.LogStore, ctx.UpdateChan)
	SafeGoroutine(
		fmt.Sprintf("docker build '%s'", tag),
		func() error {
			args := []string{"build", "-t", tag, "."}
			_, err := ctx.ProcessManager.StartCommand("docker-build", "docker", args)
			if err == nil {
				ctx.UpdateChan <- processUpdateMsg{}
			}
			return err
		},
		func(err error) {
			errorCtx := ScriptStartContext("docker-build", "Docker Build", ctx.LogStore, ctx.UpdateChan)
			errorHandler.HandleError(err, errorCtx)
		},
	)

	*ctx.CurrentView = "logs"
}

func handleComposeUpCommand(ctx *SlashCommandContext, parts []string) {
	service := ""
	if len(parts) >= 2 {
		service = parts[1]
	}

	errorHandler := NewStandardErrorHandler(ctx.LogStore, ctx.UpdateChan)
	SafeGoroutine(
		fmt.Sprintf("compose up '%s'", service),
		func() error {
			args := []string{"up", "-d"}
			if service != "" {
				args = append(args, service)
			}
			_, err := ctx.ProcessManager.StartCommand("docker-compose", "docker-compose", args)
			if err == nil {
				ctx.UpdateChan <- processUpdateMsg{}
			}
			return err
		},
		func(err error) {
			errorCtx := ScriptStartContext("docker-compose", "Compose Up", ctx.LogStore, ctx.UpdateChan)
			errorHandler.HandleError(err, errorCtx)
		},
	)

	*ctx.CurrentView = "logs"
}

func handleComposeDownCommand(ctx *SlashCommandContext, parts []string) {
	service := ""
	if len(parts) >= 2 {
		service = parts[1]
	}

	errorHandler := NewStandardErrorHandler(ctx.LogStore, ctx.UpdateChan)
	SafeGoroutine(
		fmt.Sprintf("compose down '%s'", service),
		func() error {
			args := []string{"down"}
			if service != "" {
				args = append(args, service)
			}
			_, err := ctx.ProcessManager.StartCommand("docker-compose", "docker-compose", args)
			if err == nil {
				ctx.UpdateChan <- processUpdateMsg{}
			}
			return err
		},
		func(err error) {
			errorCtx := ScriptStartContext("docker-compose", "Compose Down", ctx.LogStore, ctx.UpdateChan)
			errorHandler.HandleError(err, errorCtx)
		},
	)

	*ctx.CurrentView = "processes"
}

// Message types used for updates
type logUpdateMsg struct{}
type processUpdateMsg struct{}
