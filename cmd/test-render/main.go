package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/pkg/events"
)

func main() {
	// Get the current directory
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting directory:", err)
		os.Exit(1)
	}

	// Check for package.json
	packagePath := filepath.Join(dir, "package.json")
	hasPackageJSON := false
	if _, err := os.Stat(packagePath); err == nil {
		hasPackageJSON = true
		fmt.Println("Found package.json in", dir)
	} else {
		// Use the test directory
		dir = "/tmp/test-brummer"
		packagePath = filepath.Join(dir, "package.json")
		if _, err := os.Stat(packagePath); err == nil {
			hasPackageJSON = true
			fmt.Println("Using test directory:", dir)
		}
	}

	// Create event bus
	eventBus := &events.EventBus{}

	// Create process manager
	processMgr, err := process.NewManager(dir, eventBus, hasPackageJSON)
	if err != nil {
		fmt.Println("Failed to create process manager:", err)
		os.Exit(1)
	}

	// Create the integrated shell with process view
	shell := components.NewIntegratedShell(processMgr)

	// Simulate window size
	shell.SetSize(80, 24)

	// Render the shell
	fmt.Println("=== Integrated Shell Render Test ===")
	fmt.Println(shell.View())
	fmt.Println("\n=== Process View Component Test ===")

	// Create standalone process view
	processView := components.NewProcessViewNew(processMgr)
	processView.SetSize(78, 20) // Account for borders

	fmt.Println(processView.View())

	fmt.Println("\n=== Test Complete ===")
	fmt.Println("The integrated shell and process view components compiled and rendered successfully!")
}
