// +build integration

package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/test/testutil"
)

// TestMCPServerStartupNoTUI tests MCP server startup in headless mode
func TestMCPServerStartupNoTUI(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer in headless mode with debug (enables MCP)
	err := bt.Start("--no-tui", "--debug")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for MCP server to start
	port, err := bt.WaitForMCP(5 * time.Second)
	if err != nil {
		t.Fatalf("MCP server did not start: %v\nOutput: %s", err, bt.Output())
	}

	t.Logf("MCP server started successfully on port %d", port)

	// Verify the URL format in output
	output := bt.Output()
	re := regexp.MustCompile(`MCP server started on http://localhost:\d+/mcp`)
	if !re.MatchString(output) {
		t.Errorf("MCP URL not in expected format")
	}
}

// TestMCPServerStartupTUI tests MCP server startup in TUI mode
func TestMCPServerStartupTUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TUI test in short mode")
	}

	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer in TUI mode with debug
	err := bt.Start("--debug")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// TUI should start successfully
	time.Sleep(2 * time.Second)

	// Check that process is still running
	if bt.Cmd.ProcessState != nil {
		t.Errorf("TUI process exited unexpectedly")
		t.Logf("Output: %s", bt.Output())
	}
}

// TestMCPURLDisplay tests that MCP URL is displayed correctly
func TestMCPURLDisplay(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer
	err := bt.Start("--no-tui", "--debug")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for MCP URL to appear
	err = bt.WaitForOutput("✅ MCP server started on http://localhost:", 5*time.Second)
	if err != nil {
		t.Errorf("MCP URL not displayed: %v", err)
		t.Logf("Output: %s", bt.Output())
		return
	}

	// Extract and validate URL format
	output := bt.Output()
	urlPattern := regexp.MustCompile(`✅ MCP server started on http://localhost:(\d+)/mcp`)
	matches := urlPattern.FindStringSubmatch(output)
	if len(matches) < 2 {
		t.Errorf("Could not extract MCP port from URL display")
		return
	}

	t.Logf("MCP URL displayed correctly with port %s", matches[1])
}

// TestMCPJSONRPC tests MCP JSON-RPC functionality
func TestMCPJSONRPC(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start MCP server
	err := bt.Start("--no-tui", "--debug")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for MCP server to be ready
	port, err := bt.WaitForMCP(5 * time.Second)
	if err != nil {
		t.Fatalf("MCP server did not start: %v", err)
	}

	// Wait a bit more for full initialization
	time.Sleep(1 * time.Second)

	// Test JSON-RPC request
	t.Run("tools/list", func(t *testing.T) {
		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/list",
		}

		resp, err := makeJSONRPCRequest(t, port, request)
		if err != nil {
			t.Fatalf("JSON-RPC request failed: %v", err)
		}

		// Check response structure
		if resp["jsonrpc"] != "2.0" {
			t.Errorf("Expected jsonrpc 2.0, got %v", resp["jsonrpc"])
		}

		if resp["id"] != float64(1) { // JSON numbers are float64
			t.Errorf("Expected id 1, got %v", resp["id"])
		}

		// Check for result
		if result, ok := resp["result"].(map[string]interface{}); ok {
			if tools, ok := result["tools"].([]interface{}); ok {
				t.Logf("Found %d tools", len(tools))
				if len(tools) == 0 {
					t.Errorf("Expected at least one tool")
				}
			} else {
				t.Errorf("Result does not contain tools array")
			}
		} else {
			t.Errorf("Response does not contain result object")
		}
	})

	t.Run("resources/list", func(t *testing.T) {
		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "resources/list",
		}

		resp, err := makeJSONRPCRequest(t, port, request)
		if err != nil {
			t.Fatalf("JSON-RPC request failed: %v", err)
		}

		// Check for result
		if result, ok := resp["result"].(map[string]interface{}); ok {
			if resources, ok := result["resources"].([]interface{}); ok {
				t.Logf("Found %d resources", len(resources))
			}
		}
	})
}

// TestMCPBatchRequests tests batch JSON-RPC requests
func TestMCPBatchRequests(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start MCP server
	err := bt.Start("--no-tui", "--debug")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for MCP server
	port, err := bt.WaitForMCP(5 * time.Second)
	if err != nil {
		t.Fatalf("MCP server did not start: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Send batch request
	batch := []interface{}{
		map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/list",
		},
		map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "resources/list",
		},
	}

	requestBody, _ := json.Marshal(batch)
	url := fmt.Sprintf("http://localhost:%d/mcp", port)
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("Batch request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var responses []map[string]interface{}
	if err := json.Unmarshal(body, &responses); err != nil {
		t.Fatalf("Failed to parse batch response: %v", err)
	}

	if len(responses) != 2 {
		t.Errorf("Expected 2 responses, got %d", len(responses))
	}

	for i, resp := range responses {
		if resp["id"] != float64(i+1) {
			t.Errorf("Response %d has wrong id: %v", i, resp["id"])
		}
	}
}

// TestMCPServerSentEvents tests SSE support
func TestMCPServerSentEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSE test in short mode")
	}

	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start MCP server
	err := bt.Start("--no-tui", "--debug", "dev")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for MCP server
	port, err := bt.WaitForMCP(5 * time.Second)
	if err != nil {
		t.Fatalf("MCP server did not start: %v", err)
	}

	// Test SSE endpoint
	url := fmt.Sprintf("http://localhost:%d/mcp", port)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("SSE request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %s", resp.Header.Get("Content-Type"))
	}

	// Read a bit of the stream
	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	if n > 0 {
		t.Logf("SSE stream data: %s", string(buf[:n]))
	}
}

// TestMCPDebugMode tests that MCP is enabled in debug mode
func TestMCPDebugMode(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start without debug mode
	err := bt.Start("--no-tui")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// MCP should still start (it's always enabled now)
	_, err = bt.WaitForMCP(3 * time.Second)
	if err != nil {
		t.Logf("MCP not started without debug mode (this may be expected)")
	}

	// Stop and restart with debug mode
	bt.Stop()
	bt = testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	err = bt.Start("--no-tui", "--debug")
	if err != nil {
		t.Fatalf("failed to start brummer with debug: %v", err)
	}

	// MCP should definitely start in debug mode
	port, err := bt.WaitForMCP(5 * time.Second)
	if err != nil {
		t.Errorf("MCP did not start in debug mode: %v", err)
	} else {
		t.Logf("MCP started in debug mode on port %d", port)
	}
}

// TestMCPPortConfiguration tests custom MCP port
func TestMCPPortConfiguration(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Get a free port
	customPort, err := testutil.GetFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	// Start with custom port
	err = bt.Start("--no-tui", "--debug", "--port", fmt.Sprintf("%d", customPort))
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Check that MCP started on custom port
	err = bt.WaitForOutput(fmt.Sprintf("MCP server started on http://localhost:%d/mcp", customPort), 5*time.Second)
	if err != nil {
		t.Errorf("MCP did not start on custom port %d: %v", customPort, err)
		t.Logf("Output: %s", bt.Output())
	}
}

// Helper function to make JSON-RPC requests
func makeJSONRPCRequest(t *testing.T, port int, request map[string]interface{}) (map[string]interface{}, error) {
	t.Helper()

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://localhost:%d/mcp", port)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response, nil
}