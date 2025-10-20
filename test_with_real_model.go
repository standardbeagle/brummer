package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/domain/processes"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui"
	"github.com/standardbeagle/brummer/pkg/events"
)

func main() {
	// Clear debug log
	os.Remove("/tmp/brum-debug.log")

	// Create minimal dependencies
	fmt.Println("Step 1: Creating EventBus...")
	eventBus := events.NewEventBus()

	fmt.Println("Step 2: Creating ProcessManager...")
	processMgr, _ := processes.NewManager(".", eventBus, false)

	fmt.Println("Step 3: Creating LogStore...")
	logStore := logs.NewStore(1000, eventBus)

	var proxyServer *proxy.Server = nil
	cfg := &config.Config{}

	// Create the actual Model
	fmt.Println("Step 4: Creating Model...")
	model := tui.NewModel(processMgr, logStore, eventBus, nil, proxyServer, 7777, cfg, ".")

	fmt.Println("Step 5: Model created successfully")
	fmt.Println("Step 6: Starting BubbleTea...")
	p := tea.NewProgram(model) // NO options, like minimal test

	fmt.Println("Step 7: Calling p.Run()...")
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Program exited")
}
