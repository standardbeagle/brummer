package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// TestMCPHubInitialization tests that the MCP hub server initializes properly
func TestMCPHubInitialization(t *testing.T) {
	// Create a pipe to simulate stdio communication
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()

	// Mock stdio
	oldStdin := getStdin
	oldStdout := getStdout
	defer func() {
		getStdin = oldStdin
		getStdout = oldStdout
	}()

	getStdin = func() io.Reader { return reader }
	
	var output bytes.Buffer
	getStdout = func() io.Writer { return &output }

	// Send initialize request
	go func() {
		initReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]interface{}{
				"protocolVersion": "1.0",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
			},
		}
		data, _ := json.Marshal(initReq)
		writer.Write(data)
		writer.Write([]byte("\n"))
		writer.Close()
	}()

	// Run the hub in a test context
	done := make(chan bool)
	go func() {
		runMCPHub()
		done <- true
	}()

	// Wait for output or timeout
	select {
	case <-done:
		// Check the output
		outputStr := output.String()
		if !strings.Contains(outputStr, "brummer-hub") {
			t.Errorf("Expected output to contain 'brummer-hub', got: %s", outputStr)
		}
		if !strings.Contains(outputStr, "2025-03-26") {
			t.Errorf("Expected output to contain protocol version '2025-03-26', got: %s", outputStr)
		}
	}
}

// TestMCPHubTools tests that the hub exposes the correct tools
func TestMCPHubTools(t *testing.T) {
	expectedTools := []string{"instances/list", "instances/connect"}
	
	// This is a placeholder for now since we need to refactor
	// the hub code to be more testable
	t.Logf("Hub should expose tools: %v", expectedTools)
}

// Helper functions for testing
var (
	getStdin  = func() io.Reader { return nil }
	getStdout = func() io.Writer { return nil }
)