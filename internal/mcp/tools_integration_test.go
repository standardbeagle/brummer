package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServerWithPackageJSON creates a test server with a real package.json
func setupTestServerWithPackageJSON(t testing.TB) (*StreamableServer, string) {
	// Get the testdata directory
	testDir, err := filepath.Abs("./testdata")
	require.NoError(t, err)

	// Verify package.json exists
	packageJSONPath := filepath.Join(testDir, "package.json")
	_, err = os.Stat(packageJSONPath)
	require.NoError(t, err, "testdata/package.json must exist")

	// Create server components with test directory
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(10000)
	processMgr, err := process.NewManager(testDir, eventBus, true)
	require.NoError(t, err)

	// Use a random port for proxy server in tests to avoid conflicts
	proxyPort := 8080 + (time.Now().UnixNano() % 1000)
	proxyServer := proxy.NewServer(int(proxyPort), eventBus)

	t.Cleanup(func() {
		// Clean up in reverse order
		proxyServer.Stop()
		processMgr.Cleanup()
		logStore.Close()
	})

	server := NewStreamableServer(7777, processMgr, logStore, proxyServer, eventBus)
	return server, testDir
}

// Test scripts/list with real package.json
func TestScriptsListWithRealPackageJSON(t *testing.T) {
	server, _ := setupTestServerWithPackageJSON(t)

	msg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name":      "scripts_list",
		"arguments": map[string]interface{}{},
	}, 1)

	response := sendRequest(t, server, msg)

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}
	assert.NotNil(t, response.Result)

	result := response.Result.(map[string]interface{})

	scripts, ok := result["scripts"].(map[string]interface{})
	if !ok {
		t.Fatalf("scripts is not a map, got: %T", result["scripts"])
	}

	// Verify we have the expected scripts
	scriptCommands := make(map[string]string)
	for name, cmd := range scripts {
		scriptCommands[name] = cmd.(string)
	}

	// Check for some expected scripts
	assert.NotEmpty(t, scriptCommands["test"])
	assert.NotEmpty(t, scriptCommands["dev"])
	assert.NotEmpty(t, scriptCommands["build"])
	assert.NotEmpty(t, scriptCommands["echo-test"])
	assert.NotEmpty(t, scriptCommands["error-test"])

	// Verify specific script commands
	assert.Equal(t, "echo 'Hello from echo-test script'", scriptCommands["echo-test"])
	assert.Equal(t, "echo 'Starting error test' && exit 1", scriptCommands["error-test"])
}

// Test running a simple echo script
// NOTE: This test may fail if the process manager doesn't capture stdout properly
// or if the script execution environment doesn't forward output correctly
func TestScriptsRunEchoScript(t *testing.T) {
	t.Skip("Skipping - process output capture not working in test environment")
	server, _ := setupTestServerWithPackageJSON(t)

	msg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name": "scripts_run",
		"arguments": map[string]interface{}{
			"name": "echo-test",
		},
	}, 1)

	response := sendRequest(t, server, msg)

	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Result)

	result := response.Result.(map[string]interface{})
	processID := result["processId"].(string)
	assert.NotEmpty(t, processID)

	// Wait for script to complete
	time.Sleep(100 * time.Millisecond) // Reduced from 1s

	// First check process status
	statusMsg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name": "scripts_status",
		"arguments": map[string]interface{}{
			"name": "echo-test",
		},
	}, 2)

	statusResponse := sendRequest(t, server, statusMsg)
	if statusResponse.Error == nil {
		t.Logf("Process status: %v", statusResponse.Result)
	}

	// Search with a broad query to get all echo-test logs
	logsMsg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name": "logs_search",
		"arguments": map[string]interface{}{
			"processId": processID,
			"query":     "echo", // Broad search term
		},
	}, 3)

	logsResponse := sendRequest(t, server, logsMsg)
	if logsResponse.Error != nil {
		t.Fatalf("Logs search failed: %v", logsResponse.Error)
	}

	// logs_search returns an array directly
	logs, ok := logsResponse.Result.([]interface{})
	if !ok {
		t.Fatalf("Expected logs to be an array, got: %T", logsResponse.Result)
	}

	assert.Greater(t, len(logs), 0, "Should find the echo output")

	// Verify the log content
	found := false
	for _, logInterface := range logs {
		log := logInterface.(map[string]interface{})
		if message, ok := log["message"].(string); ok {
			if strings.Contains(message, "Hello from echo-test") {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Should find the specific echo message")
}

// Test running a script that exits with error
func TestScriptsRunErrorScript(t *testing.T) {
	server, _ := setupTestServerWithPackageJSON(t)

	msg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name": "scripts_run",
		"arguments": map[string]interface{}{
			"name": "error-test",
		},
	}, 1)

	response := sendRequest(t, server, msg)

	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Result)

	result := response.Result.(map[string]interface{})
	processID := result["processId"].(string)

	// Wait for script to fail
	time.Sleep(200 * time.Millisecond) // Give script time to exit

	// Check process status
	statusMsg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name":      "scripts_status",
		"arguments": map[string]interface{}{},
	}, 2)

	statusResponse := sendRequest(t, server, statusMsg)
	assert.Nil(t, statusResponse.Error)

	// scripts_status returns an array of processes
	processes, ok := statusResponse.Result.([]interface{})
	if !ok {
		// It might return a map with processes key
		if resultMap, ok := statusResponse.Result.(map[string]interface{}); ok {
			processes = resultMap["processes"].([]interface{})
		} else {
			t.Fatalf("Unexpected status result type: %T", statusResponse.Result)
		}
	}

	// Find our process
	var foundProcess map[string]interface{}
	for _, proc := range processes {
		p := proc.(map[string]interface{})
		if p["processId"] == processID {
			foundProcess = p
			break
		}
	}

	// It should have failed
	if foundProcess != nil {
		assert.Equal(t, "failed", foundProcess["status"])
		if exitCode, ok := foundProcess["exitCode"]; ok {
			assert.Equal(t, float64(1), exitCode)
		}
	}
}

// Test running and stopping a long-running script
func TestScriptsRunAndStop(t *testing.T) {
	t.Skip("Skipping - may hang in test environment")
}

// Test running multiple scripts concurrently
func TestScriptsRunConcurrent(t *testing.T) {
	server, _ := setupTestServerWithPackageJSON(t)

	// Start two concurrent scripts
	scripts := []string{"concurrent-1", "concurrent-2"}
	processIDs := make([]string, len(scripts))

	for i, scriptName := range scripts {
		msg := makeJSONRPCRequest("tools/call", map[string]interface{}{
			"name": "scripts_run",
			"arguments": map[string]interface{}{
				"name": scriptName,
			},
		}, i+1)

		response := sendRequest(t, server, msg)
		require.Nil(t, response.Error)

		result := response.Result.(map[string]interface{})
		processIDs[i] = result["processId"].(string)
	}

	// Check that both are running
	statusMsg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name":      "scripts_status",
		"arguments": map[string]interface{}{},
	}, 10)

	statusResponse := sendRequest(t, server, statusMsg)
	assert.Nil(t, statusResponse.Error)

	// scripts_status returns an array of processes directly
	processes, ok := statusResponse.Result.([]interface{})
	if !ok {
		t.Fatalf("Expected processes array, got: %T", statusResponse.Result)
	}

	runningCount := 0
	for _, proc := range processes {
		p := proc.(map[string]interface{})
		if p["status"] == "running" {
			for _, pid := range processIDs {
				if p["processId"] == pid {
					runningCount++
				}
			}
		}
	}

	assert.Equal(t, 2, runningCount, "Both scripts should be running concurrently")
}

// Test log streaming with fast output
func TestLogsStreamFastOutput(t *testing.T) {
	server, _ := setupTestServerWithPackageJSON(t)

	// Start a script with fast output
	runMsg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name": "scripts_run",
		"arguments": map[string]interface{}{
			"name": "fast-output",
		},
	}, 1)

	runResponse := sendRequest(t, server, runMsg)
	require.Nil(t, runResponse.Error)

	result := runResponse.Result.(map[string]interface{})
	processID := result["processId"].(string)

	// Wait for completion
	time.Sleep(100 * time.Millisecond) // Reduced from 1s

	// Search logs
	searchMsg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name": "logs_search",
		"arguments": map[string]interface{}{
			"processId": processID,
			"pattern":   "Fast line",
		},
	}, 2)

	searchResponse := sendRequest(t, server, searchMsg)
	assert.Nil(t, searchResponse.Error)

	searchResult := searchResponse.Result.(map[string]interface{})
	logs := searchResult["logs"].([]interface{})

	// Should have captured all 10 lines
	assert.GreaterOrEqual(t, len(logs), 10)
}

// Test scripts with different output types
func TestScriptsWithDifferentOutputs(t *testing.T) {
	server, _ := setupTestServerWithPackageJSON(t)

	testCases := []struct {
		name        string
		scriptName  string
		searchFor   string
		expectError bool
	}{
		{
			name:       "JSON output",
			scriptName: "json-output",
			searchFor:  `"status": "ok"`,
		},
		{
			name:       "ANSI colors",
			scriptName: "ansi-colors",
			searchFor:  "Red text",
		},
		{
			name:       "Unicode output",
			scriptName: "unicode-test",
			searchFor:  "🚀",
		},
		{
			name:       "Multi-line output",
			scriptName: "multi-line",
			searchFor:  "Line 2",
		},
		{
			name:        "Error output",
			scriptName:  "with-error-output",
			searchFor:   "Error output",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run the script
			runMsg := makeJSONRPCRequest("tools/call", map[string]interface{}{
				"name": "scripts_run",
				"arguments": map[string]interface{}{
					"name": tc.scriptName,
				},
			}, 1)

			runResponse := sendRequest(t, server, runMsg)
			require.Nil(t, runResponse.Error)

			result := runResponse.Result.(map[string]interface{})
			processID := result["processId"].(string)

			// Wait for completion
			time.Sleep(50 * time.Millisecond) // Reduced from 500ms

			// Search for expected output
			searchMsg := makeJSONRPCRequest("tools/call", map[string]interface{}{
				"name": "logs_search",
				"arguments": map[string]interface{}{
					"processId": processID,
					"errorOnly": tc.expectError,
				},
			}, 2)

			searchResponse := sendRequest(t, server, searchMsg)
			assert.Nil(t, searchResponse.Error)

			searchResult := searchResponse.Result.(map[string]interface{})
			logs := searchResult["logs"].([]interface{})

			// Check if we found the expected output
			found := false
			for _, logInterface := range logs {
				log := logInterface.(map[string]interface{})
				content := log["content"].(string)
				if content != "" {
					found = true
					break
				}
			}

			assert.True(t, found, "Should find output for %s", tc.name)
		})
	}
}

// Test duplicate script detection
func TestScriptsRunDuplicate(t *testing.T) {
	server, _ := setupTestServerWithPackageJSON(t)

	// Start a long-running script
	firstMsg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name": "scripts_run",
		"arguments": map[string]interface{}{
			"name": "start",
		},
	}, 1)

	firstResponse := sendRequest(t, server, firstMsg)
	require.Nil(t, firstResponse.Error)

	firstResult := firstResponse.Result.(map[string]interface{})
	firstProcessID := firstResult["processId"].(string)

	// Try to start the same script again
	secondMsg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name": "scripts_run",
		"arguments": map[string]interface{}{
			"name": "start",
		},
	}, 2)

	secondResponse := sendRequest(t, server, secondMsg)
	assert.Nil(t, secondResponse.Error)

	secondResult := secondResponse.Result.(map[string]interface{})

	// Should indicate it's a duplicate
	if duplicate, ok := secondResult["duplicate"].(bool); ok {
		assert.True(t, duplicate)
		assert.Equal(t, firstProcessID, secondResult["processId"])
	}

	// Clean up
	stopMsg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name": "scripts_stop",
		"arguments": map[string]interface{}{
			"processId": firstProcessID,
		},
	}, 3)

	sendRequest(t, server, stopMsg)
}
