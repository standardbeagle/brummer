//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"time"
)

// setupSignalHandling sets up signal handling for Windows with double Ctrl+C force exit
func setupSignalHandling(sigChan chan os.Signal) {
	// Create a raw signal channel that we monitor directly
	rawSigChan := make(chan os.Signal, 1)
	// On Windows, only os.Interrupt (Ctrl+C) is reliably supported
	signal.Notify(rawSigChan, os.Interrupt)

	// Monitor signals in background with double-interrupt detection
	go func() {
		var interruptCount int32

		for sig := range rawSigChan {
			count := atomic.AddInt32(&interruptCount, 1)

			if count == 1 {
				// First interrupt: forward signal for graceful shutdown
				fmt.Fprintln(os.Stderr, "\n⚠️  Received interrupt signal (Press Ctrl+C again to force quit)")
				sigChan <- sig

				// Reset counter after 2 seconds (if no second interrupt, next one will be treated as first)
				time.AfterFunc(2*time.Second, func() {
					atomic.StoreInt32(&interruptCount, 0)
				})
			} else {
				// Second interrupt: force exit immediately
				fmt.Fprintln(os.Stderr, "\n🚨 Force quit! Exiting immediately...")
				os.Exit(130) // Exit code 130 is standard for Ctrl+C
			}
		}
	}()
}
