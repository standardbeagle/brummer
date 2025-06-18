//go:build windows
// +build windows

package main

import (
	"os"
	"os/signal"
)

// setupSignalHandling sets up signal handling for Windows
func setupSignalHandling(sigChan chan os.Signal) {
	// On Windows, only os.Interrupt (Ctrl+C) is reliably supported
	signal.Notify(sigChan, os.Interrupt)
}
