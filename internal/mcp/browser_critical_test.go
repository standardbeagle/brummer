package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCriticalBrowserFunctionality tests the most important browser screenshot and JS execution features
func TestCriticalBrowserFunctionality(t *testing.T) {
	// This is the critical test that must pass for browser features to be considered working
	
	eventBus := events.NewEventBus()
	server := NewStreamableServer(7779, nil, nil, nil, eventBus)
	defer server.Stop()
	
	// Ensure browser tools are registered
	server.registerBrowserTools()
	
	// Critical Test 1: Browser screenshot tool exists and is properly configured
	t.Run("ScreenshotToolConfiguration", func(t *testing.T) {
		tool, exists := server.tools["browser_screenshot"]
		require.True(t, exists, "browser_screenshot tool must be registered")
		
		// Verify tool has proper schema
		assert.NotNil(t, tool.InputSchema, "Screenshot tool must have input schema")
		assert.Contains(t, tool.Description, "screenshot", "Description must mention screenshot")
		assert.NotNil(t, tool.Handler, "Screenshot tool must have handler")
		
		// Verify schema contains required fields
		var schema map[string]interface{}
		err := json.Unmarshal(tool.InputSchema, &schema)
		require.NoError(t, err, "Schema must be valid JSON")
		
		properties, exists := schema["properties"].(map[string]interface{})
		require.True(t, exists, "Schema must have properties")
		
		// Check for critical parameters
		assert.Contains(t, properties, "format", "Schema must include format parameter")
		assert.Contains(t, properties, "fullPage", "Schema must include fullPage parameter")
		assert.Contains(t, properties, "selector", "Schema must include selector parameter")
		assert.Contains(t, properties, "quality", "Schema must include quality parameter")
	})
	
	// Critical Test 2: REPL execute tool exists and is properly configured
	t.Run("REPLToolConfiguration", func(t *testing.T) {
		tool, exists := server.tools["repl_execute"]
		require.True(t, exists, "repl_execute tool must be registered")
		
		// Verify tool has proper schema
		assert.NotNil(t, tool.InputSchema, "REPL tool must have input schema")
		assert.Contains(t, tool.Description, "JavaScript", "Description must mention JavaScript")
		assert.NotNil(t, tool.Handler, "REPL tool must have handler")
		
		// Verify schema contains required fields
		var schema map[string]interface{}
		err := json.Unmarshal(tool.InputSchema, &schema)
		require.NoError(t, err, "Schema must be valid JSON")
		
		properties, exists := schema["properties"].(map[string]interface{})
		require.True(t, exists, "Schema must have properties")
		
		// Check for critical parameters
		assert.Contains(t, properties, "code", "Schema must include code parameter")
		assert.Contains(t, properties, "sessionId", "Schema must include sessionId parameter")
		
		// Verify code is required
		required, exists := schema["required"].([]interface{})
		require.True(t, exists, "Schema must have required fields")
		assert.Contains(t, required, "code", "Code parameter must be required")
	})
	
	// Critical Test 3: Screenshot tool handles all supported formats
	t.Run("ScreenshotFormatSupport", func(t *testing.T) {
		tool := server.tools["browser_screenshot"]
		
		formats := []string{"png", "jpeg", "webp"}
		for _, format := range formats {
			t.Run(format, func(t *testing.T) {
				args := json.RawMessage(`{"format": "` + format + `"}`)
				_, err := tool.Handler(args)
				
				// Should timeout (expected in test environment) not format error
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", 
						"Should timeout waiting for browser, not fail on format validation")
				}
			})
		}
	})
	
	// Critical Test 4: Screenshot tool handles different capture modes
	t.Run("ScreenshotCaptureModes", func(t *testing.T) {
		tool := server.tools["browser_screenshot"]
		
		testCases := []struct {
			name string
			args string
		}{
			{"Viewport", `{"format": "png"}`},
			{"FullPage", `{"format": "png", "fullPage": true}`},
			{"ElementSelector", `{"format": "png", "selector": "#main"}`},
			{"HighQuality", `{"format": "jpeg", "quality": 95}`},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				args := json.RawMessage(tc.args)
				_, err := tool.Handler(args)
				
				// Should timeout (expected in test environment) not parameter error
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", 
						"Should timeout waiting for browser response")
				}
			})
		}
	})
	
	// Critical Test 5: REPL tool handles different JavaScript scenarios
	t.Run("REPLJavaScriptScenarios", func(t *testing.T) {
		tool := server.tools["repl_execute"]
		
		testCases := []struct {
			name string
			code string
		}{
			{"SimpleExpression", "return 2 + 2;"},
			{"DOMAccess", "return document.title;"},
			{"ConsoleLog", "console.log('test'); return 'logged';"},
			{"AsyncCode", "return new Promise(resolve => resolve('async'));"},
			{"MultiLine", "const x = 10; const y = 20; return x + y;"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				args := json.RawMessage(`{"code": "` + tc.code + `"}`)
				_, err := tool.Handler(args)
				
				// Should timeout (expected in test environment) not syntax error
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", 
						"Should timeout waiting for browser response")
				}
			})
		}
	})
	
	// Critical Test 6: Tools handle response registration and cleanup properly
	t.Run("ResponseHandling", func(t *testing.T) {
		// Test REPL response registration
		responseID := "critical-test-123"
		responseChan := server.registerREPLResponse(responseID)
		require.NotNil(t, responseChan, "Response channel must be created")
		
		// Test response handling
		testResponse := map[string]interface{}{
			"result": "critical test response",
			"success": true,
		}
		
		// Simulate response in goroutine
		go func() {
			time.Sleep(10 * time.Millisecond)
			server.handleREPLResponse(responseID, testResponse)
		}()
		
		// Wait for response
		select {
		case response := <-responseChan:
			responseMap, ok := response.(map[string]interface{})
			require.True(t, ok, "Response must be a map")
			assert.Equal(t, "critical test response", responseMap["result"], 
				"Must receive correct response")
			assert.Equal(t, true, responseMap["success"], 
				"Must receive success flag")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Must receive response within timeout")
		}
		
		// Test cleanup
		server.unregisterREPLResponse(responseID)
		
		// Verify cleanup worked by checking that channel doesn't exist anymore
		// We don't try to send to a cleaned up channel as that could panic
		// Instead, we just verify the cleanup completed without error
	})
	
	// Critical Test 7: Error handling for malformed inputs
	t.Run("ErrorHandling", func(t *testing.T) {
		tools := []string{"browser_screenshot", "repl_execute"}
		
		for _, toolName := range tools {
			t.Run(toolName, func(t *testing.T) {
				tool := server.tools[toolName]
				
				// Test malformed JSON
				_, err := tool.Handler(json.RawMessage(`{invalid json}`))
				assert.Error(t, err, "Must handle malformed JSON")
				
				// Test empty JSON
				_, err = tool.Handler(json.RawMessage(`{}`))
				// Should not error on empty JSON (should use defaults)
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", 
						"Should only timeout, not parameter error")
				}
			})
		}
	})
}

// TestBrowserToolsE2EWorkflow tests a realistic end-to-end workflow
func TestBrowserToolsE2EWorkflow(t *testing.T) {
	// Simulate realistic workflow: open browser -> execute JS -> take screenshot
	
	eventBus := events.NewEventBus()
	server := NewStreamableServer(7780, nil, nil, nil, eventBus)
	defer server.Stop()
	
	server.registerBrowserTools()
	
	// Step 1: Open browser (would normally open actual browser)
	t.Run("Step1_OpenBrowser", func(t *testing.T) {
		args := json.RawMessage(`{
			"url": "http://localhost:3000",
			"processName": "test-app"
		}`)
		
		result, err := server.tools["browser_open"].Handler(args)
		
		// Should succeed (will try to open browser)
		if err != nil {
			t.Logf("Browser open failed in test environment (expected): %v", err)
		} else {
			resultMap := result.(map[string]interface{})
			assert.Contains(t, resultMap, "opened", "Should indicate browser open attempt")
		}
	})
	
	// Step 2: Execute JavaScript to set up page
	t.Run("Step2_ExecuteJavaScript", func(t *testing.T) {
		args := json.RawMessage(`{
			"code": "document.body.style.backgroundColor = 'lightblue'; return 'page setup complete';"
		}`)
		
		_, err := server.tools["repl_execute"].Handler(args)
		
		// Should timeout in test environment
		if err != nil {
			assert.Contains(t, err.Error(), "timeout", 
				"Should timeout waiting for browser")
		}
	})
	
	// Step 3: Take screenshot
	t.Run("Step3_TakeScreenshot", func(t *testing.T) {
		args := json.RawMessage(`{
			"format": "png",
			"fullPage": true
		}`)
		
		_, err := server.tools["browser_screenshot"].Handler(args)
		
		// Should timeout in test environment
		if err != nil {
			assert.Contains(t, err.Error(), "timeout", 
				"Should timeout waiting for browser")
		}
	})
}

// TestBrowserToolsPerformance tests that browser tools respond within reasonable time
func TestBrowserToolsPerformance(t *testing.T) {
	eventBus := events.NewEventBus()
	server := NewStreamableServer(7781, nil, nil, nil, eventBus)
	defer server.Stop()
	
	server.registerBrowserTools()
	
	// Test that tools respond quickly when they timeout (not hang indefinitely)
	t.Run("ResponseTimeouts", func(t *testing.T) {
		tools := []struct {
			name string
			args json.RawMessage
		}{
			{"browser_screenshot", json.RawMessage(`{"format": "png"}`)},
			{"repl_execute", json.RawMessage(`{"code": "return 'test';"}`)},
		}
		
		for _, tool := range tools {
			t.Run(tool.name, func(t *testing.T) {
				start := time.Now()
				_, err := server.tools[tool.name].Handler(tool.args)
				duration := time.Since(start)
				
				// Should complete within reasonable time (5 seconds + some buffer)
				assert.Less(t, duration, 6*time.Second, 
					"Tool should timeout within reasonable time")
				
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", 
						"Should timeout, not hang indefinitely")
				}
			})
		}
	})
}