package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

var interruptCount int32
var shutdownStarted int32

// setupSignalHandling sets up signal handling for Unix systems with double Ctrl+C to shutdown
func setupSignalHandling(sigChan chan os.Signal) {
	// Create a raw signal channel that we monitor directly
	rawSigChan := make(chan os.Signal, 1)
	signal.Notify(rawSigChan, os.Interrupt, syscall.SIGTERM)

	// Monitor signals in background with double-interrupt detection
	go func() {
		for sig := range rawSigChan {
			count := atomic.AddInt32(&interruptCount, 1)
			isShuttingDown := atomic.LoadInt32(&shutdownStarted) == 1

			if count == 1 && !isShuttingDown {
				// First interrupt: just show warning, DON'T shutdown yet
				fmt.Fprintln(os.Stderr, "\n⚠️  Interrupt received. Press Ctrl+C again within 2 seconds to exit.")

				// Reset counter after 2 seconds (user can continue working)
				time.AfterFunc(2*time.Second, func() {
					atomic.StoreInt32(&interruptCount, 0)
				})
			} else if count >= 2 && !isShuttingDown {
				// Second interrupt: start graceful shutdown
				atomic.StoreInt32(&shutdownStarted, 1)
				fmt.Fprintln(os.Stderr, "⏳ Shutting down gracefully... (Press Ctrl+C again to force quit)")
				sigChan <- sig
			} else if isShuttingDown {
				// Any interrupt during shutdown: force exit immediately
				fmt.Fprintln(os.Stderr, "\n🚨 Force quit! Exiting immediately...")
				os.Exit(130) // Exit code 130 is standard for Ctrl+C
			}
		}
	}()
}