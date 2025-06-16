//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// setupSignalHandling sets up signal handling for Unix systems
func setupSignalHandling(sigChan chan os.Signal) {
	// On Unix, we can handle both SIGINT and SIGTERM
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
}
