//go:build integration
// +build integration

package integration

import (
	"time"
)

// collectOutput collects output from the output channel for a specified duration
func collectOutput(outputChan <-chan []byte, duration time.Duration) []byte {
	var output []byte
	timeout := time.After(duration)

	for {
		select {
		case data := <-outputChan:
			output = append(output, data...)
		case <-timeout:
			return output
		}
	}
}