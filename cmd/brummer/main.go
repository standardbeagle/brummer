package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

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
	Use:   "brum",
	Short: "A TUI for managing npm/yarn/pnpm/bun scripts with MCP integration",
	Long: `Brummer is a terminal user interface for managing package.json scripts.
It provides real-time log monitoring, error detection, and MCP server integration
for external tool access.`,
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
		model := tui.NewModel(processMgr, logStore, eventBus, mcpServer)
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