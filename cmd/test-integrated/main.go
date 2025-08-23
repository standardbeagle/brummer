package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/pkg/events"
)

func main() {
	// Get the current directory
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Check for package.json
	packagePath := filepath.Join(dir, "package.json")
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		fmt.Println("No package.json found in current directory")
		fmt.Println("This test requires a Node.js project with npm scripts")
		os.Exit(1)
	}

	// Create event bus
	eventBus := &events.EventBus{}
	
	// Create process manager
	processMgr, err := process.NewManager(dir, eventBus, true)
	if err != nil {
		log.Fatal("Failed to create process manager:", err)
	}

	// Create the integrated shell with process view
	shell := components.NewIntegratedShell(processMgr)

	// Create and run the program
	p := tea.NewProgram(shell, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}