package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui"
	"github.com/standardbeagle/brummer/pkg/events"
)

// Simple test model that just logs keyboard input
type testModel struct {
	logStore *logs.Store
}

func (m testModel) Init() tea.Cmd {
	return nil
}

func (m testModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		keyDesc := fmt.Sprintf("type=%v, runes=%v", msg.Type, msg.Runes)
		m.logStore.Add("test", "Keyboard", "🎹 KEY PRESSED: "+keyDesc, false)

		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m testModel) View() string {
	return "Keyboard Test - Press keys (Ctrl+C to quit)\nCheck logs for keyboard input"
}

func main() {
	fmt.Println("🧪 Simple keyboard input test")
	fmt.Println("This will test if BubbleTea receives keyboard input at all")
	fmt.Println("")

	// Create basic dependencies
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(eventBus)

	// Create minimal config
	cfg := &config.Config{}

	// Create process manager
	processMgr := process.NewManager(logStore, eventBus, cfg, ".")

	// Create MCP and proxy servers (minimal)
	mcpServer := &mcp.Server{} // minimal implementation
	proxyServer := &proxy.Server{} // minimal implementation

	// Create the actual Brummer model to test keyboard routing
	brummerModel := tui.NewModel(
		processMgr,
		logStore,
		eventBus,
		mcpServer,
		proxyServer,
		7777,
		cfg,
		".",
	)

	// Create test program
	p := tea.NewProgram(brummerModel)

	fmt.Println("🚀 Starting keyboard test (10 seconds)...")
	fmt.Println("Press various keys to test keyboard input")
	fmt.Println("Keys to try: 1, 2, 3, 4, 5, Tab, /, Esc, Enter")
	fmt.Println("")

	// Run with timeout
	go func() {
		time.Sleep(10 * time.Second)
		p.Quit()
	}()

	// Start the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Print logs
	fmt.Println("\n📋 Keyboard input received:")
	logs := logStore.GetLogs()
	if len(logs) == 0 {
		fmt.Println("❌ No keyboard input detected")
	} else {
		for _, log := range logs {
			if log.Source == "Keyboard" || log.Source == "Model" {
				fmt.Printf("  %s\n", log.Message)
			}
		}
	}

	fmt.Println("\n🔍 Checking for diagnostic logs...")
	fmt.Println("Looking for:")
	fmt.Println("- '⌨️  DIAGNOSTIC: Received keyboard message'")
	fmt.Println("- '🔄 Routing keyboard message to InputComponent'")
	fmt.Println("- '🎯 DIAGNOSTIC: Global keyboard handler executed'")

	// Count diagnostic messages
	diagnosticCount := 0
	for _, log := range logs {
		message := log.Message
		if message == "⌨️  DIAGNOSTIC: Received keyboard message" ||
		   message == "🔄 Routing keyboard message to InputComponent" ||
		   message[:len("🎯 DIAGNOSTIC:")] == "🎯 DIAGNOSTIC:" {
			diagnosticCount++
			fmt.Printf("  ✅ %s\n", message)
		}
	}

	if diagnosticCount == 0 {
		fmt.Println("  ❌ No diagnostic messages found - keyboard routing not working")
	}

	fmt.Printf("\n📊 Summary: %d keyboard events logged, %d diagnostic messages\n", len(logs), diagnosticCount)
}