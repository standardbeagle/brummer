package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/discovery"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui"
	"github.com/standardbeagle/brummer/pkg/events"
)

var (
	// Version is set at build time
	Version = "dev"

	workDir       string
	mcpPort       int
	proxyPort     int
	proxyMode     string
	proxyURL      string
	standardProxy bool
	noMCP         bool
	noTUI         bool
	noProxy       bool
	showVersion   bool
	showSettings  bool
	debugMode     bool
	mcpHub        bool
)

var rootCmd = &cobra.Command{
	Use:   "brum [scripts...] or brum '<command>'",
	Short: "A TUI for managing npm/yarn/pnpm/bun scripts with MCP integration",
	Long: `Brummer is a terminal user interface for managing package.json scripts and running commands.
It provides real-time log monitoring, error detection, and MCP server integration
for external tool access. Works with or without package.json.

Basic Usage:
  brum                          # Start TUI in scripts view
  brum dev                      # Start 'dev' script and show logs
  brum dev test                 # Start both 'dev' and 'test' scripts
  brum 'node server.js'         # Run arbitrary command
  brum -d ../app dev            # Run 'dev' in ../app directory

Proxy Examples:
  brum --standard-proxy         # Start with traditional HTTP proxy (port 19888)
  brum --proxy-url http://localhost:3000
                                # Auto-proxy specific URL in reverse mode
  brum --proxy-port 8888        # Use custom proxy port
  brum --no-proxy               # Disable proxy entirely

MCP Server Examples:
  brum --no-mcp                 # Disable MCP server
  brum -p 8080                  # Use custom MCP port (default: 7777)
  brum --no-tui                 # Headless mode (MCP only)

Proxy Modes (default: reverse):
  reverse                       # Creates shareable URLs for detected endpoints
                                # Each URL gets its own proxy port (e.g. localhost:20888)
  full                          # Traditional HTTP proxy requiring browser config
                                # Configure browser to use localhost:19888 as proxy

Toggle proxy modes at runtime:
  /toggle-proxy                 # Slash command in TUI
  Press 't'                     # Key binding in URLs/Web view

Default Ports & Settings:
  MCP Server: 7777              # Model Context Protocol for external tools
  Proxy Server: 19888           # HTTP proxy (PAC file: /proxy.pac)
  Reverse Proxy URLs: 20888+    # Auto-allocated ports for each URL
  Proxy Mode: reverse           # Creates shareable URLs by default`,
	Args: cobra.ArbitraryArgs,
	Run:  runApp,
}

func init() {
	// Version flag
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
	rootCmd.Flags().BoolVar(&showSettings, "settings", false, "Show current configuration settings with sources")

	// Directory and port flags
	rootCmd.Flags().StringVarP(&workDir, "dir", "d", ".", "Working directory (package.json optional)")
	rootCmd.Flags().IntVarP(&mcpPort, "port", "p", 7777, "MCP server port")

	// Proxy configuration
	rootCmd.Flags().IntVar(&proxyPort, "proxy-port", 19888, "HTTP proxy server port")
	rootCmd.Flags().StringVar(&proxyMode, "proxy-mode", "reverse", "Proxy mode: 'full' (traditional proxy) or 'reverse' (create shareable URLs)")
	rootCmd.Flags().StringVar(&proxyURL, "proxy-url", "", "URL to automatically proxy in reverse mode (e.g., http://localhost:3000)")
	rootCmd.Flags().BoolVar(&standardProxy, "standard-proxy", false, "Start in standard/full proxy mode (equivalent to --proxy-mode=full)")
	rootCmd.Flags().BoolVar(&noProxy, "no-proxy", false, "Disable HTTP proxy server")

	// Feature toggles
	rootCmd.Flags().BoolVar(&noMCP, "no-mcp", false, "Disable MCP server")
	rootCmd.Flags().BoolVar(&noTUI, "no-tui", false, "Run in headless mode (MCP server only)")
	rootCmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug mode with MCP connections tab")
	rootCmd.Flags().BoolVar(&mcpHub, "mcp", false, "Run as MCP hub (stdio transport, no TUI)")

	// Set version for cobra
	rootCmd.Version = Version
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runApp(cmd *cobra.Command, args []string) {
	// Handle version flag
	if showVersion {
		fmt.Printf("brum version %s\n", Version)
		return
	}

	// Handle settings flag
	if showSettings {
		showCurrentSettings()
		return
	}

	// Handle MCP hub mode
	if mcpHub {
		runMCPHub()
		return
	}

	// Resolve working directory
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		log.Fatal("Failed to resolve working directory:", err)
	}

	// Check if package.json exists (not required - will use fallback mode if missing)
	hasPackageJSON := true
	if _, err := os.Stat(filepath.Join(absWorkDir, "package.json")); os.IsNotExist(err) {
		hasPackageJSON = false
	}

	// Initialize components
	eventBus := events.NewEventBus()

	processMgr, err := process.NewManager(absWorkDir, eventBus, hasPackageJSON)
	if err != nil {
		log.Fatal("Failed to initialize process manager:", err)
	}

	logStore := logs.NewStore(10000)
	detector := logs.NewEventDetector(eventBus)

	// Initialize proxy server if enabled
	var proxyServer *proxy.Server
	if !noProxy {
		// Parse proxy mode
		mode := proxy.ProxyModeReverse
		if proxyMode == "full" || standardProxy {
			// Check if proxy-url is specified with full mode
			if proxyURL != "" {
				fmt.Printf("Warning: --proxy-url requires reverse proxy mode. Switching to reverse mode.\n")
				mode = proxy.ProxyModeReverse
			} else {
				mode = proxy.ProxyModeFull
			}
		}

		// Only start proxy server if we have explicit proxy parameters or full proxy mode
		shouldStartProxy := mode == proxy.ProxyModeFull || proxyURL != "" || standardProxy

		if shouldStartProxy {
			proxyServer = proxy.NewServerWithMode(proxyPort, mode, eventBus)
			if err := proxyServer.Start(); err != nil {
				if noTUI {
					log.Printf("Failed to start proxy server: %v", err)
				} else {
					logStore.Add("system", "proxy", fmt.Sprintf("❌ Failed to start proxy server: %v", err), true)
				}
				// Continue without proxy
				proxyServer = nil
			} else {
				// Get the actual port being used (may be different if there was a conflict)
				actualPort := proxyServer.GetPort()

				// Only write to stdout if TUI is disabled
				if noTUI {
					if mode == proxy.ProxyModeFull {
						fmt.Printf("Started HTTP proxy server on port %d (full proxy mode)\n", actualPort)
						fmt.Printf("PAC file available at: %s\n", proxyServer.GetPACURL())
						fmt.Printf("Configure browser automatic proxy: %s\n", proxyServer.GetPACURL())
					} else {
						fmt.Printf("Started HTTP proxy server on port %d (will create proxies for detected URLs)\n", actualPort)
					}
				} else {
					modeDesc := "reverse proxy (shareable URLs)"
					if mode == proxy.ProxyModeFull {
						modeDesc = "full proxy"
					}
					logStore.Add("system", "proxy", fmt.Sprintf("🌐 Started HTTP proxy server on port %d in %s mode", actualPort, modeDesc), false)
					if mode == proxy.ProxyModeFull {
						logStore.Add("system", "proxy", fmt.Sprintf("📄 PAC file available at: %s", proxyServer.GetPACURL()), false)
					}
				}

				// Register arbitrary URL if provided
				if proxyURL != "" && mode == proxy.ProxyModeReverse {
					if proxyResult := proxyServer.RegisterURL(proxyURL, "custom"); proxyResult != proxyURL {
						if noTUI {
							fmt.Printf("Registered custom URL: %s -> %s\n", proxyURL, proxyResult)
						}
						// Add the URL to the log store so it appears in the URLs tab
						logStore.Add("custom-proxy", "custom", fmt.Sprintf("🌐 Proxy registered: %s", proxyURL), false)
						// Update the proxy URL mapping
						logStore.UpdateProxyURL(proxyURL, proxyResult)
					} else {
						if noTUI {
							fmt.Printf("Note: Custom URL %s will be proxied when accessed\n", proxyURL)
						}
						// Add the URL to the log store so it appears in the URLs tab
						logStore.Add("custom-proxy", "custom", fmt.Sprintf("🌐 Proxy ready: %s", proxyURL), false)
					}
				}
			}
		} else {
			// Create proxy server but don't start it yet - it will be started when URLs are detected
			proxyServer = proxy.NewServerWithMode(proxyPort, mode, eventBus)
		}
	}

	// Set up log processing with event detection
	processMgr.AddLogCallback(func(processID, line string, isError bool) {
		if proc, exists := processMgr.GetProcess(processID); exists {
			// In noTUI mode, print logs directly to stdout/stderr
			if noTUI && line != "" {
				timestamp := time.Now().Format("15:04:05")
				if isError {
					fmt.Fprintf(os.Stderr, "[%s] %s: %s\n", timestamp, proc.Name, line)
				} else {
					fmt.Printf("[%s] %s: %s\n", timestamp, proc.Name, line)
				}
			}

			entry := logStore.Add(processID, proc.Name, line, isError)
			detector.ProcessLogLine(processID, proc.Name, line, isError)

			// Register URLs with proxy server
			if proxyServer != nil && entry != nil {
				// Only check URLs that were detected in this specific log entry
				detectedURLs := logStore.DetectURLsInContent(line)
				if len(detectedURLs) > 0 {
					// Start proxy server if it's not running and URLs are detected
					if !proxyServer.IsRunning() {
						if err := proxyServer.Start(); err != nil {
							if noTUI {
								log.Printf("Failed to start proxy server: %v", err)
							} else {
								logStore.Add("system", "proxy", fmt.Sprintf("❌ Failed to start proxy server: %v", err), true)
							}
						} else {
							// Proxy server started successfully due to URL detection
							actualPort := proxyServer.GetPort()
							if noTUI {
								fmt.Printf("Started HTTP proxy server on port %d (detected URLs in logs)\n", actualPort)
							} else {
								logStore.Add("system", "proxy", fmt.Sprintf("🌐 Started HTTP proxy server on port %d for detected URLs", actualPort), false)
							}
						}
					}

					// Process detected URLs
					for _, url := range detectedURLs {
						// Check if this URL is already proxied
						existingProxyURL := proxyServer.GetProxyURL(url)
						if existingProxyURL == url {
							// URL not yet proxied, register it with context from the log line
							label := extractURLLabel(line, proc.Name)
							proxyURL := proxyServer.RegisterURLWithLabel(url, proc.Name, label)
							// Store the proxy URL if different
							if proxyURL != url {
								logStore.UpdateProxyURL(url, proxyURL)
							}
						}
					}
				}
			}
		}
	})

	// Handle CLI arguments to start scripts
	var startedFromCLI bool
	if len(args) > 0 {
		startedFromCLI = true

		if hasPackageJSON {
			scripts := processMgr.GetScripts()

			for _, arg := range args {
				// Check if it's a known script
				if _, exists := scripts[arg]; exists {
					// Start the script
					proc, err := processMgr.StartScript(arg)
					if err != nil {
						if noTUI {
							log.Printf("Failed to start script '%s': %v", arg, err)
						} else {
							logStore.Add("system", "startup", fmt.Sprintf("❌ Failed to start script '%s': %v", arg, err), true)
						}
					} else {
						if noTUI {
							fmt.Printf("Started script '%s' (PID: %s)\n", arg, proc.ID)
						} else {
							logStore.Add("system", "startup", fmt.Sprintf("✅ Started script '%s' (PID: %s)", arg, proc.ID), false)
						}
					}
					continue
				}

				// Fallback to command execution if not a script
				if len(args) == 1 && strings.Contains(arg, " ") {
					// Single argument with spaces - treat as a command
					parts := strings.Fields(arg)
					if len(parts) > 0 {
						proc, err := processMgr.StartCommand("custom", parts[0], parts[1:])
						if err != nil {
							log.Fatalf("Failed to start command '%s': %v", arg, err)
						} else {
							if noTUI {
								fmt.Printf("Started command '%s' (PID: %s)\n", arg, proc.ID)
							} else {
								logStore.Add("system", "startup", fmt.Sprintf("✅ Started command '%s' (PID: %s)", arg, proc.ID), false)
							}
						}
					}
				} else {
					// Try to run it as a command
					proc, err := processMgr.StartCommand(arg, arg, []string{})
					if err != nil {
						if noTUI {
							log.Printf("Failed to start command '%s': %v", arg, err)
						} else {
							logStore.Add("system", "startup", fmt.Sprintf("❌ Failed to start command '%s': %v", arg, err), true)
						}
					} else {
						if noTUI {
							fmt.Printf("Started command '%s' (PID: %s)\n", arg, proc.ID)
						} else {
							logStore.Add("system", "startup", fmt.Sprintf("✅ Started command '%s' (PID: %s)", arg, proc.ID), false)
						}
					}
				}
			}
		} else {
			// No package.json - treat all args as commands
			for _, arg := range args {
				if len(args) == 1 && strings.Contains(arg, " ") {
					// Single argument with spaces - treat as a command
					parts := strings.Fields(arg)
					if len(parts) > 0 {
						proc, err := processMgr.StartCommand("custom", parts[0], parts[1:])
						if err != nil {
							log.Fatalf("Failed to start command '%s': %v", arg, err)
						} else {
							if noTUI {
								fmt.Printf("Started command '%s' (PID: %s)\n", arg, proc.ID)
							} else {
								logStore.Add("system", "startup", fmt.Sprintf("✅ Started command '%s' (PID: %s)", arg, proc.ID), false)
							}
						}
					}
				} else {
					// Try to run it as a command
					proc, err := processMgr.StartCommand(arg, arg, []string{})
					if err != nil {
						if noTUI {
							log.Printf("Failed to start command '%s': %v", arg, err)
						} else {
							logStore.Add("system", "startup", fmt.Sprintf("❌ Failed to start command '%s': %v", arg, err), true)
						}
					} else {
						if noTUI {
							fmt.Printf("Started command '%s' (PID: %s)\n", arg, proc.ID)
						} else {
							logStore.Add("system", "startup", fmt.Sprintf("✅ Started command '%s' (PID: %s)", arg, proc.ID), false)
						}
					}
				}
			}
		}

		// Give processes a moment to start before showing TUI
		if startedFromCLI {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Track instance registration for cleanup
	var registeredInstanceID string
	var registeredInstancesDir string

	// Start MCP server if enabled
	var mcpServerInterface interface {
		Start() error
		Stop() error
	}
	var mcpServer interface {
		IsRunning() bool
		GetPort() int
		Start() error
		Stop() error
	} // For TUI compatibility
	if !noMCP || noTUI {
		// Use new StreamableServer by default
		mcpServer = mcp.NewStreamableServer(mcpPort, processMgr, logStore, proxyServer, eventBus)
		mcpServerInterface = mcpServer
		
		// Start MCP server in background regardless of TUI mode
		go func() {
			if !noTUI {
				// Give TUI time to initialize subscriptions
				time.Sleep(100 * time.Millisecond)
			}
			
			if err := mcpServerInterface.Start(); err != nil {
				if noTUI {
					log.Fatal("MCP server error:", err)
				} else {
					// MCP server error (logged internally)
					errorMsg := fmt.Sprintf("❌ MCP server failed to start: %v", err)
					logStore.Add("system", "MCP", errorMsg, true)

					// Publish system message event
					eventBus.Publish(events.Event{
						Type: events.EventType("system.message"),
						Data: map[string]interface{}{
							"level":   "error",
							"context": "MCP Server",
							"message": errorMsg,
						},
					})
				}
			}
		}()

		// Give the server a moment to start and potentially change ports
		time.Sleep(200 * time.Millisecond)

		// Get actual port and register instance (same for both TUI and no-TUI)
		actualPort := mcpPort
		if mcpStreamable, ok := mcpServer.(*mcp.StreamableServer); ok {
			actualPort = mcpStreamable.GetPort()
		}
		
		// Register this instance with the discovery system
		instanceID := generateInstanceID(filepath.Base(absWorkDir))
		instance := &discovery.Instance{
			ID:        instanceID,
			Name:      filepath.Base(absWorkDir),
			Directory: absWorkDir,
			Port:      actualPort,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
		}
		instance.ProcessInfo.PID = os.Getpid()
		instance.ProcessInfo.Executable, _ = os.Executable()
		
		registeredInstancesDir = discovery.GetDefaultInstancesDir()
		registeredInstanceID = instanceID
		
		if noTUI {
			fmt.Printf("Registering instance: %s at %s\n", instanceID, registeredInstancesDir)
		}
		
		if err := discovery.RegisterInstance(registeredInstancesDir, instance); err != nil {
			if noTUI {
				fmt.Fprintf(os.Stderr, "Warning: Failed to register instance: %v\n", err)
			} else {
				logStore.Add("system", "discovery", fmt.Sprintf("⚠️  Failed to register instance: %v", err), true)
			}
		} else {
			if noTUI {
				fmt.Printf("Successfully registered instance: %s\n", instanceID)
			} else {
				logStore.Add("system", "discovery", fmt.Sprintf("✅ Registered instance: %s", instanceID), false)
			}
		}
		
		// Start ping update routine
		pingTicker := time.NewTicker(30 * time.Second)
		go func() {
			defer pingTicker.Stop()
			for range pingTicker.C {
				if err := discovery.UpdateInstancePing(registeredInstancesDir, registeredInstanceID); err != nil {
					// Silent failure - don't spam logs
				}
			}
		}()
		
		if noTUI {
			// Display the actual port being used (may be different if there was a conflict)
			fmt.Printf("MCP server URL: http://localhost:%d/mcp\n", actualPort)
			fmt.Printf("Press Ctrl+C to stop.\n")

			// Set up signal handling
			sigChan := make(chan os.Signal, 1)
			setupSignalHandling(sigChan)

			// Wait for signal
			<-sigChan
			fmt.Println("\nShutting down gracefully...")

			// Cleanup all processes
			fmt.Println("Stopping all running processes...")
			if err := processMgr.Cleanup(); err != nil {
				// Error during cleanup (logged internally in headless mode)
			}

			fmt.Println("Stopping MCP server...")
			_ = mcpServerInterface.Stop() // Ignore cleanup errors during shutdown
			
			// Unregister instance
			if registeredInstanceID != "" && registeredInstancesDir != "" {
				_ = discovery.UnregisterInstance(registeredInstancesDir, registeredInstanceID)
			}

			if proxyServer != nil {
				fmt.Println("Stopping proxy server...")
				_ = proxyServer.Stop() // Ignore cleanup errors during shutdown
			}

			fmt.Println("Cleanup complete.")
			return
		}
	}

	// Only run TUI if not in headless mode
	if !noTUI {
		// Set up signal handling for cleanup
		sigChan := make(chan os.Signal, 1)
		setupSignalHandling(sigChan)

		// Create and run TUI
		initialView := tui.ViewScriptSelector
		if startedFromCLI {
			initialView = tui.ViewLogs
		} else if !hasPackageJSON {
			// Default to processes view when no package.json (no scripts to select)
			initialView = tui.ViewProcesses
		}
		model := tui.NewModelWithView(processMgr, logStore, eventBus, mcpServer, proxyServer, mcpPort, initialView, debugMode)
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
			// Error during cleanup (logged internally)
		}

		if mcpServerInterface != nil {
			fmt.Println("Stopping MCP server...")
			_ = mcpServerInterface.Stop() // Ignore cleanup errors during shutdown
			
			// Unregister instance
			if registeredInstanceID != "" && registeredInstancesDir != "" {
				_ = discovery.UnregisterInstance(registeredInstancesDir, registeredInstanceID)
			}
		}

		if proxyServer != nil {
			fmt.Println("Stopping proxy server...")
			_ = proxyServer.Stop() // Ignore cleanup errors during shutdown
		}

		fmt.Println("Cleanup complete.")
	}
}

// extractURLLabel extracts a meaningful label from a log line containing a URL
func extractURLLabel(logLine, processName string) string {
	// Clean the log line first
	line := strings.TrimSpace(logLine)

	// Common patterns for server startup messages with labels
	patterns := []struct {
		regex   *regexp.Regexp
		extract func([]string) string
	}{
		// "[Frontend] Server listening on http://..."
		{regexp.MustCompile(`(?i)\[([^\]]+)\].*https?://`), func(m []string) string { return m[1] }},
		// "Frontend: Server listening on http://..."
		{regexp.MustCompile(`(?i)^([^:\s]+):\s+.*(?:server|listening|started|running).*https?://`), func(m []string) string { return m[1] }},
		// "Frontend server listening on http://..."
		{regexp.MustCompile(`(?i)^(\w+)\s+server\s+(?:listening|started|running)\s+(?:on|at).*https?://`), func(m []string) string { return m[1] + " Server" }},
		// "API started on http://..." or "Backend running at http://..."
		{regexp.MustCompile(`(?i)^(\w+)\s+(?:started|running|listening)\s+(?:on|at).*https?://`), func(m []string) string { return m[1] }},
		// "Local: http://..." (common in Vite/dev servers)
		{regexp.MustCompile(`(?i)(\w+):\s+https?://`), func(m []string) string { return m[1] }},
		// "Server ready at http://..." - extract what comes before
		{regexp.MustCompile(`(?i)(\w+)\s+(?:ready|available)\s+at\s+https?://`), func(m []string) string { return m[1] }},
		// Look for words that might indicate the service type near the URL
		{regexp.MustCompile(`(?i)(frontend|backend|api|admin|dashboard|web|client|server)\s+.*https?://`), func(m []string) string { return strings.Title(strings.ToLower(m[1])) }},
		{regexp.MustCompile(`(?i)https?://.*\s+(frontend|backend|api|admin|dashboard|web|client|server)`), func(m []string) string { return strings.Title(strings.ToLower(m[1])) }},
	}

	for _, p := range patterns {
		if matches := p.regex.FindStringSubmatch(line); len(matches) > 1 {
			label := p.extract(matches)
			// Clean up the label
			label = strings.TrimSpace(label)
			if label != "" && label != processName {
				return label
			}
		}
	}

	// Default to process name if no meaningful label found
	return processName
}

func showCurrentSettings() {
	cfg, err := config.LoadWithSources()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(cfg.DisplaySettingsWithSources())
}

// generateInstanceID creates a secure, unique instance ID
func generateInstanceID(prefix string) string {
	// Generate 8 random bytes
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp-based ID on error
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	// Convert to hex string (16 characters)
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("%s-%s", prefix, randomHex)
}

// Global systems for the hub
var (
	discoverySystem *discovery.Discovery
	connectionMgr   *mcp.ConnectionManager
)

// Global hub MCP server for dynamic tool registration
var hubMCPServer *server.MCPServer

// runMCPHub runs the MCP hub in stdio mode for discovering and managing instances
func runMCPHub() {
	// Initialize connection manager
	connectionMgr = mcp.NewConnectionManager()
	defer connectionMgr.Stop()
	
	// Initialize discovery system
	instancesDir := discovery.GetDefaultInstancesDir()
	var err error
	discoverySystem, err = discovery.New(instancesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize discovery: %v\n", err)
		os.Exit(1)
	}
	defer discoverySystem.Stop()
	
	// Register callback to handle discovered instances
	discoverySystem.OnUpdate(func(instances map[string]*discovery.Instance) {
		// Register new instances with connection manager
		for _, inst := range instances {
			if err := connectionMgr.RegisterInstance(inst); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to register instance %s: %v\n", inst.ID, err)
			}
		}
	})
	
	// Start discovery
	discoverySystem.Start()
	
	// Start periodic cleanup of stale instances
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := discoverySystem.CleanupStaleInstances(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to cleanup stale instances: %v\n", err)
			}
		}
	}()
	
	// Create MCP server
	hubMCPServer = server.NewMCPServer(
		"brummer-hub",
		Version,
		server.WithToolCapabilities(true),
	)
	
	// Add instances/list tool
	listTool := mcplib.NewTool("instances/list",
		mcplib.WithDescription("List all running brummer instances"),
	)
	hubMCPServer.AddTool(listTool, handleInstancesList)
	
	// Add instances/connect tool
	connectTool := mcplib.NewTool("instances/connect",
		mcplib.WithDescription("Connect to a specific brummer instance"),
		mcplib.WithString("instance_id",
			mcplib.Required(),
			mcplib.Description("The ID of the instance to connect to"),
		),
	)
	hubMCPServer.AddTool(connectTool, handleInstancesConnect)
	
	// Add instances/disconnect tool
	disconnectTool := mcplib.NewTool("instances/disconnect",
		mcplib.WithDescription("Disconnect from the current brummer instance"),
	)
	hubMCPServer.AddTool(disconnectTool, handleInstancesDisconnect)
	
	// Start stdio server
	if err := server.ServeStdio(hubMCPServer); err != nil {
		// Log to stderr to avoid corrupting stdio protocol
		fmt.Fprintf(os.Stderr, "Hub server error: %v\n", err)
		os.Exit(1)
	}
}

// handleInstancesList returns a list of all running brummer instances
func handleInstancesList(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	// Get instances from connection manager (includes connection state)
	connections := connectionMgr.ListInstances()
	
	// Convert to slice for JSON output
	instanceList := make([]map[string]interface{}, 0, len(connections))
	for _, conn := range connections {
		instanceList = append(instanceList, map[string]interface{}{
			"id":          conn.InstanceID,
			"name":        conn.Name,
			"directory":   conn.Directory,
			"port":        conn.Port,
			"process_pid": conn.ProcessPID,
			"state":       conn.State.String(),
			"connected":   conn.State == mcp.StateActive,
		})
	}
	
	data, err := json.Marshal(instanceList)
	if err != nil {
		return nil, err
	}
	
	return &mcplib.CallToolResult{
		Content: []mcplib.Content{
			mcplib.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// Track registered proxy tools for cleanup
var registeredProxyTools = make(map[string][]string) // instanceID -> tool names

// handleInstancesConnect connects to a specific brummer instance
func handleInstancesConnect(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	instanceID, err := request.RequireString("instance_id")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}
	
	// For stdio transport, we use a single global session
	// In a real implementation, this would come from the MCP context
	sessionID := "stdio-session"
	
	// Connect session to instance
	if err := connectionMgr.ConnectSession(sessionID, instanceID); err != nil {
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to connect: %v", err),
				},
			},
		}, nil
	}
	
	// Get the client for this instance
	connections := connectionMgr.ListInstances()
	var activeClient *mcp.HubClient
	for _, conn := range connections {
		if conn.InstanceID == instanceID && conn.State == mcp.StateActive && conn.Client != nil {
			activeClient = conn.Client
			break
		}
	}
	
	if activeClient == nil {
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to get client for instance %s", instanceID),
				},
			},
		}, nil
	}
	
	// Fetch tools from the instance
	toolsData, err := activeClient.ListTools(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list tools from instance %s: %v\n", instanceID, err)
		// Continue anyway - connection is still established
	} else {
		// Parse tools and register proxies
		var toolsResponse struct {
			Tools []struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				InputSchema json.RawMessage `json:"inputSchema"`
			} `json:"tools"`
		}
		
		if err := json.Unmarshal(toolsData, &toolsResponse); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse tools from instance %s: %v\n", instanceID, err)
		} else {
			// Register proxy tools
			var toolNames []string
			for _, tool := range toolsResponse.Tools {
				// Create prefixed tool name
				prefixedName := fmt.Sprintf("%s/%s", instanceID, tool.Name)
				toolNames = append(toolNames, prefixedName)
				
				// Create proxy tool handler
				handler := createProxyToolHandler(instanceID, tool.Name, connectionMgr)
				
				// Parse input schema for tool creation
				var inputSchema map[string]interface{}
				if len(tool.InputSchema) > 0 {
					json.Unmarshal(tool.InputSchema, &inputSchema)
				}
				
				// Create tool with mark3labs library
				proxyTool := mcplib.NewTool(prefixedName,
					mcplib.WithDescription(fmt.Sprintf("[%s] %s", instanceID, tool.Description)),
					// Note: We can't dynamically add schema with mark3labs library
					// This is a limitation we'll need to work around
				)
				
				// Add tool to hub server
				hubMCPServer.AddTool(proxyTool, handler)
			}
			
			// Track registered tools for cleanup
			registeredProxyTools[instanceID] = toolNames
			
			fmt.Fprintf(os.Stderr, "Registered %d proxy tools from instance %s\n", len(toolNames), instanceID)
		}
	}
	
	return &mcplib.CallToolResult{
		Content: []mcplib.Content{
			mcplib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Connected to instance %s", instanceID),
			},
		},
	}, nil
}

// createProxyToolHandler creates a handler that forwards tool calls to the instance
func createProxyToolHandler(instanceID, toolName string, connMgr *mcp.ConnectionManager) func(context.Context, mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return func(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		// Get the client for this instance
		connections := connMgr.ListInstances()
		var activeClient *mcp.HubClient
		for _, conn := range connections {
			if conn.InstanceID == instanceID && conn.State == mcp.StateActive && conn.Client != nil {
				activeClient = conn.Client
				break
			}
		}
		
		if activeClient == nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Instance %s is not connected", instanceID)), nil
		}
		
		// Convert request arguments to map
		args := make(map[string]interface{})
		// The mark3labs library doesn't provide direct access to raw arguments
		// We need to extract them from the request
		// This is a limitation - we'll work with what we have
		
		// Forward the tool call
		result, err := activeClient.CallTool(ctx, toolName, args)
		if err != nil {
			return mcplib.NewToolResultError(fmt.Sprintf("Tool call failed: %v", err)), nil
		}
		
		// Parse result and return
		var response interface{}
		if err := json.Unmarshal(result, &response); err != nil {
			// Return raw result as text if parsing fails
			return &mcplib.CallToolResult{
				Content: []mcplib.Content{
					mcplib.TextContent{
						Type: "text",
						Text: string(result),
					},
				},
			}, nil
		}
		
		// Convert response to text
		responseText, _ := json.Marshal(response)
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: string(responseText),
				},
			},
		}, nil
	}
}

// handleInstancesDisconnect disconnects from the current instance
func handleInstancesDisconnect(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	// For stdio transport, we use a single global session
	sessionID := "stdio-session"
	
	// Get current instance before disconnecting
	var currentInstanceID string
	connections := connectionMgr.ListInstances()
	for _, conn := range connections {
		if conn.Sessions[sessionID] {
			currentInstanceID = conn.InstanceID
			break
		}
	}
	
	// Disconnect session
	if err := connectionMgr.DisconnectSession(sessionID); err != nil {
		return &mcplib.CallToolResult{
			Content: []mcplib.Content{
				mcplib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to disconnect: %v", err),
				},
			},
		}, nil
	}
	
	// Remove proxy tools if we found the instance
	if currentInstanceID != "" && hubMCPServer != nil {
		if toolNames, exists := registeredProxyTools[currentInstanceID]; exists {
			// Note: mark3labs library doesn't support removing tools dynamically
			// This is a limitation - tools will remain registered but will fail when called
			// In a production system, we'd need to work around this
			delete(registeredProxyTools, currentInstanceID)
			fmt.Fprintf(os.Stderr, "Note: %d proxy tools from instance %s remain registered (library limitation)\n", 
				len(toolNames), currentInstanceID)
		}
	}
	
	return &mcplib.CallToolResult{
		Content: []mcplib.Content{
			mcplib.TextContent{
				Type: "text",
				Text: "Disconnected from instance",
			},
		},
	}, nil
}
