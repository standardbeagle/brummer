package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/beagle/brummer/internal/logs"
	"github.com/beagle/brummer/internal/mcp"
	"github.com/beagle/brummer/internal/process"
	"github.com/beagle/brummer/internal/tui"
	"github.com/beagle/brummer/pkg/events"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	workDir  string
	mcpPort  int
	noMCP    bool
	noTUI    bool
)

var rootCmd = &cobra.Command{
	Use:   "brum [scripts...] or brum '<command>'",
	Short: "A TUI for managing npm/yarn/pnpm/bun scripts with MCP integration",
	Long: `Brummer is a terminal user interface for managing package.json scripts.
It provides real-time log monitoring, error detection, and MCP server integration
for external tool access.

Examples:
  brum                    # Start TUI in scripts view
  brum dev                # Start 'dev' script and show logs
  brum dev test           # Start both 'dev' and 'test' scripts
  brum 'node server.js'   # Run arbitrary command
  brum -d ../app dev      # Run 'dev' in ../app directory`,
	Args: cobra.ArbitraryArgs,
	Run: runApp,
}

func init() {
	rootCmd.Flags().StringVarP(&workDir, "dir", "d", ".", "Working directory containing package.json")
	rootCmd.Flags().IntVarP(&mcpPort, "port", "p", 7777, "MCP server port")
	rootCmd.Flags().BoolVar(&noMCP, "no-mcp", false, "Disable MCP server")
	rootCmd.Flags().BoolVar(&noTUI, "no-tui", false, "Run in headless mode (MCP server only)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runApp(cmd *cobra.Command, args []string) {
	// Resolve working directory
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		log.Fatal("Failed to resolve working directory:", err)
	}

	// Check if package.json exists
	if _, err := os.Stat(filepath.Join(absWorkDir, "package.json")); os.IsNotExist(err) {
		log.Fatal("No package.json found in", absWorkDir)
	}

	// Initialize components
	eventBus := events.NewEventBus()
	
	processMgr, err := process.NewManager(absWorkDir, eventBus)
	if err != nil {
		log.Fatal("Failed to initialize process manager:", err)
	}

	logStore := logs.NewStore(10000)
	detector := logs.NewEventDetector(eventBus)

	// Set up log processing with event detection
	processMgr.RegisterLogCallback(func(processID, line string, isError bool) {
		if proc, exists := processMgr.GetProcess(processID); exists {
			logStore.Add(processID, proc.Name, line, isError)
			detector.ProcessLogLine(processID, proc.Name, line, isError)
		}
	})

	// Handle CLI arguments to start scripts
	var startedFromCLI bool
	if len(args) > 0 {
		startedFromCLI = true
		scripts := processMgr.GetScripts()
		
		for _, arg := range args {
			// Check if it's a known script
			if _, exists := scripts[arg]; exists {
				// Start the script
				proc, err := processMgr.StartScript(arg)
				if err != nil {
					log.Printf("Failed to start script '%s': %v", arg, err)
				} else {
					fmt.Printf("Started script '%s' (PID: %s)\n", arg, proc.ID)
				}
			} else if len(args) == 1 && strings.Contains(arg, " ") {
				// Single argument with spaces - treat as a command
				parts := strings.Fields(arg)
				if len(parts) > 0 {
					proc, err := processMgr.StartCommand("custom", parts[0], parts[1:])
					if err != nil {
						log.Fatalf("Failed to start command '%s': %v", arg, err)
					} else {
						fmt.Printf("Started command '%s' (PID: %s)\n", arg, proc.ID)
					}
				}
			} else {
				// Try to run it as a command
				proc, err := processMgr.StartCommand(arg, arg, []string{})
				if err != nil {
					log.Printf("Failed to start command '%s': %v", arg, err)
				} else {
					fmt.Printf("Started command '%s' (PID: %s)\n", arg, proc.ID)
				}
			}
		}
		
		// Give processes a moment to start before showing TUI
		if startedFromCLI {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Start MCP server if enabled
	var mcpServer *mcp.Server
	if !noMCP || noTUI {
		mcpServer = mcp.NewServer(mcpPort, processMgr, logStore, eventBus)
		if noTUI {
			// In headless mode, run MCP server in foreground
			fmt.Printf("Starting MCP server on port %d (headless mode)...\n", mcpPort)
			fmt.Printf("Press Ctrl+C to stop.\n")
			
			// Set up signal handling
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			
			go func() {
				if err := mcpServer.Start(); err != nil {
					log.Fatal("MCP server error:", err)
				}
			}()
			
			// Wait for signal
			<-sigChan
			fmt.Println("\nShutting down gracefully...")
			
			// Cleanup all processes
			fmt.Println("Stopping all running processes...")
			if err := processMgr.Cleanup(); err != nil {
				log.Printf("Error during process cleanup: %v", err)
			}
			
			fmt.Println("Stopping MCP server...")
			mcpServer.Stop()
			fmt.Println("Cleanup complete.")
			return
		} else {
			// In TUI mode, run MCP server in background
			go func() {
				fmt.Printf("Starting MCP server on port %d...\n", mcpPort)
				if err := mcpServer.Start(); err != nil {
					log.Printf("MCP server error: %v", err)
				}
			}()
		}
	}

	// Only run TUI if not in headless mode
	if !noTUI {
		// Set up signal handling for cleanup
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		
		// Create and run TUI
		initialView := tui.ViewScripts
		if startedFromCLI {
			initialView = tui.ViewLogs
		}
		model := tui.NewModelWithView(processMgr, logStore, eventBus, mcpServer, initialView)
		p := tea.NewProgram(model, tea.WithAltScreen())

		// Run TUI in goroutine so we can handle signals
		done := make(chan error)
		go func() {
			_, err := p.Run()
			done <- err
		}()

		// Wait for either TUI to exit or signal
		select {
		case err := <-done:
			if err != nil {
				log.Fatal("Failed to run TUI:", err)
			}
		case <-sigChan:
			fmt.Println("\nShutting down gracefully...")
			p.Quit()
			<-done // Wait for TUI to actually exit
		}

		// Cleanup all processes and resources
		fmt.Println("Stopping all running processes...")
		if err := processMgr.Cleanup(); err != nil {
			log.Printf("Error during process cleanup: %v", err)
		}
		
		if mcpServer != nil {
			fmt.Println("Stopping MCP server...")
			mcpServer.Stop()
		}
		
		fmt.Println("Cleanup complete.")
	}
}