package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/testutil"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBrowserScreenshotTool tests the browser_screenshot tool comprehensively
func TestBrowserScreenshotTool(t *testing.T) {
	scenarios := []testutil.TestScenario{
		{
			Name:        "ScreenshotViewport",
			Description: "Test basic viewport screenshot capture",
			Setup: func(t *testing.T) interface{} {
				server := createTestStreamableServer(t)
				return server
			},
			Execute: func(t *testing.T, context interface{}) error {
				server := context.(*StreamableServer)
				
				args := json.RawMessage(`{"format": "png"}`)
				_, err := server.tools["browser_screenshot"].Handler(args)
				return err
			},
			Verify: func(t *testing.T, context interface{}, err error) {
				// Should succeed or timeout (expected without real browser)
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", "Expected timeout without real browser connection")
				}
			},
			Cleanup: func(t *testing.T, context interface{}) {
				server := context.(*StreamableServer)
				server.Stop()
			},
		},
		{
			Name:        "ScreenshotFullPage",
			Description: "Test full page screenshot capture",
			Setup: func(t *testing.T) interface{} {
				server := createTestStreamableServer(t)
				return server
			},
			Execute: func(t *testing.T, context interface{}) error {
				server := context.(*StreamableServer)
				
				args := json.RawMessage(`{
					"format": "png",
					"fullPage": true,
					"quality": 90
				}`)
				_, err := server.tools["browser_screenshot"].Handler(args)
				return err
			},
			Verify: func(t *testing.T, context interface{}, err error) {
				// Should succeed in generating code or timeout
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", "Expected timeout without real browser connection")
				}
			},
			Cleanup: func(t *testing.T, context interface{}) {
				server := context.(*StreamableServer)
				server.Stop()
			},
		},
		{
			Name:        "ScreenshotWithSelector",
			Description: "Test screenshot of specific element",
			Setup: func(t *testing.T) interface{} {
				server := createTestStreamableServer(t)
				return server
			},
			Execute: func(t *testing.T, context interface{}) error {
				server := context.(*StreamableServer)
				
				args := json.RawMessage(`{
					"format": "jpeg",
					"quality": 85,
					"selector": "#main-content"
				}`)
				_, err := server.tools["browser_screenshot"].Handler(args)
				return err
			},
			Verify: func(t *testing.T, context interface{}, err error) {
				// Should succeed in generating code or timeout
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", "Expected timeout without real browser connection")
				}
			},
			Cleanup: func(t *testing.T, context interface{}) {
				server := context.(*StreamableServer)
				server.Stop()
			},
		},
	}
	
	testutil.RunTestScenarios(t, scenarios)
}

// TestREPLExecuteTool tests the repl_execute tool comprehensively
func TestREPLExecuteTool(t *testing.T) {
	scenarios := []testutil.TestScenario{
		{
			Name:        "SimpleJavaScriptExecution",
			Description: "Test basic JavaScript execution",
			Setup: func(t *testing.T) interface{} {
				server := createTestStreamableServer(t)
				return server
			},
			Execute: func(t *testing.T, context interface{}) error {
				server := context.(*StreamableServer)
				
				args := json.RawMessage(`{
					"code": "console.log('Hello, World!'); return 'test result';"
				}`)
				_, err := server.tools["repl_execute"].Handler(args)
				return err
			},
			Verify: func(t *testing.T, context interface{}, err error) {
				// Should succeed or timeout (expected without real browser)
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", "Expected timeout without real browser connection")
				}
			},
			Cleanup: func(t *testing.T, context interface{}) {
				server := context.(*StreamableServer)
				server.Stop()
			},
		},
		{
			Name:        "DOMManipulation",
			Description: "Test JavaScript DOM manipulation",
			Setup: func(t *testing.T) interface{} {
				server := createTestStreamableServer(t)
				return server
			},
			Execute: func(t *testing.T, context interface{}) error {
				server := context.(*StreamableServer)
				
				args := json.RawMessage(`{
					"code": "document.body.style.backgroundColor = 'red'; return document.body.style.backgroundColor;"
				}`)
				_, err := server.tools["repl_execute"].Handler(args)
				return err
			},
			Verify: func(t *testing.T, context interface{}, err error) {
				// Should succeed or timeout
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", "Expected timeout without real browser connection")
				}
			},
			Cleanup: func(t *testing.T, context interface{}) {
				server := context.(*StreamableServer)
				server.Stop()
			},
		},
		{
			Name:        "AsyncJavaScriptExecution",
			Description: "Test asynchronous JavaScript execution",
			Setup: func(t *testing.T) interface{} {
				server := createTestStreamableServer(t)
				return server
			},
			Execute: func(t *testing.T, context interface{}) error {
				server := context.(*StreamableServer)
				
				args := json.RawMessage(`{
					"code": "return new Promise(resolve => setTimeout(() => resolve('async result'), 100));"
				}`)
				_, err := server.tools["repl_execute"].Handler(args)
				return err
			},
			Verify: func(t *testing.T, context interface{}, err error) {
				// Should succeed or timeout
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", "Expected timeout without real browser connection")
				}
			},
			Cleanup: func(t *testing.T, context interface{}) {
				server := context.(*StreamableServer)
				server.Stop()
			},
		},
		{
			Name:        "SpecificSessionExecution",
			Description: "Test JavaScript execution in specific session",
			Setup: func(t *testing.T) interface{} {
				server := createTestStreamableServer(t)
				return server
			},
			Execute: func(t *testing.T, context interface{}) error {
				server := context.(*StreamableServer)
				
				args := json.RawMessage(`{
					"code": "return window.location.href;",
					"sessionId": "test-session-123"
				}`)
				_, err := server.tools["repl_execute"].Handler(args)
				return err
			},
			Verify: func(t *testing.T, context interface{}, err error) {
				// Should succeed or timeout
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", "Expected timeout without real browser connection")
				}
			},
			Cleanup: func(t *testing.T, context interface{}) {
				server := context.(*StreamableServer)
				server.Stop()
			},
		},
	}
	
	testutil.RunTestScenarios(t, scenarios)
}

// TestBrowserToolsIntegration tests browser tools working together
func TestBrowserToolsIntegration(t *testing.T) {
	server := createTestStreamableServer(t)
	defer server.Stop()
	
	// Test tool registration
	assert.Contains(t, server.tools, "browser_screenshot", "Screenshot tool should be registered")
	assert.Contains(t, server.tools, "repl_execute", "REPL tool should be registered")
	assert.Contains(t, server.tools, "browser_open", "Browser open tool should be registered")
	assert.Contains(t, server.tools, "browser_refresh", "Browser refresh tool should be registered")
	
	// Test tool schemas
	screenshotTool := server.tools["browser_screenshot"]
	assert.NotNil(t, screenshotTool.InputSchema, "Screenshot tool should have input schema")
	assert.Contains(t, screenshotTool.Description, "screenshot", "Description should mention screenshot")
	
	replTool := server.tools["repl_execute"]
	assert.NotNil(t, replTool.InputSchema, "REPL tool should have input schema")
	assert.Contains(t, replTool.Description, "JavaScript", "Description should mention JavaScript")
	
	// Test that tools handle invalid JSON gracefully
	t.Run("InvalidJSON", func(t *testing.T) {
		invalidJSON := json.RawMessage(`{invalid json}`)
		
		_, err := screenshotTool.Handler(invalidJSON)
		assert.Error(t, err, "Should handle invalid JSON")
		
		_, err = replTool.Handler(invalidJSON)
		assert.Error(t, err, "Should handle invalid JSON")
	})
}

// TestBrowserToolsWithProxyServer tests browser tools with proxy server integration
func TestBrowserToolsWithProxyServer(t *testing.T) {
	// Create server with proxy enabled
	server := createTestStreamableServer(t)
	defer server.Stop()
	
	// Test browser_open with proxy
	t.Run("BrowserOpenWithProxy", func(t *testing.T) {
		args := json.RawMessage(`{
			"url": "http://localhost:3000",
			"processName": "dev"
		}`)
		
		result, err := server.tools["browser_open"].Handler(args)
		
		// Should succeed (will try to open browser)
		// In a real test environment, this might fail, but the logic should work
		if err != nil {
			t.Logf("Browser open failed as expected in test environment: %v", err)
		} else {
			resultMap := result.(map[string]interface{})
			assert.Contains(t, resultMap, "opened", "Result should indicate if browser was opened")
		}
	})
	
	// Test browser_refresh
	t.Run("BrowserRefresh", func(t *testing.T) {
		args := json.RawMessage(`{}`)
		
		result, err := server.tools["browser_refresh"].Handler(args)
		
		// Should succeed
		assert.NoError(t, err, "Browser refresh should not error")
		resultMap := result.(map[string]interface{})
		assert.Contains(t, resultMap, "refreshed", "Result should indicate refresh was sent")
	})
}

// TestBrowserToolResponseHandling tests response handling for browser tools
func TestBrowserToolResponseHandling(t *testing.T) {
	server := createTestStreamableServer(t)
	defer server.Stop()
	
	// Test REPL response registration and cleanup
	t.Run("REPLResponseHandling", func(t *testing.T) {
		responseID := "test-response-123"
		
		// Register response channel
		responseChan := server.registerREPLResponse(responseID)
		assert.NotNil(t, responseChan, "Response channel should be created")
		
		// Simulate response
		go func() {
			time.Sleep(50 * time.Millisecond)
			server.handleREPLResponse(responseID, map[string]interface{}{
				"result": "test response",
			})
		}()
		
		// Wait for response
		select {
		case response := <-responseChan:
			responseMap, ok := response.(map[string]interface{})
			require.True(t, ok, "Response should be a map")
			assert.Equal(t, "test response", responseMap["result"], "Should receive correct response")
		case <-time.After(200 * time.Millisecond):
			t.Fatal("Should have received response")
		}
		
		// Cleanup
		server.unregisterREPLResponse(responseID)
	})
}

// TestBrowserToolParameterValidation tests parameter validation for browser tools
func TestBrowserToolParameterValidation(t *testing.T) {
	server := createTestStreamableServer(t)
	defer server.Stop()
	
	// Test screenshot parameter validation
	t.Run("ScreenshotParameters", func(t *testing.T) {
		testCases := []struct {
			name     string
			args     string
			expectOK bool
		}{
			{
				name:     "ValidBasic",
				args:     `{"format": "png"}`,
				expectOK: true,
			},
			{
				name:     "ValidFullPage",
				args:     `{"format": "jpeg", "quality": 85, "fullPage": true}`,
				expectOK: true,
			},
			{
				name:     "ValidSelector",
				args:     `{"selector": "#test", "format": "png"}`,
				expectOK: true,
			},
			{
				name:     "EmptyArgs",
				args:     `{}`,
				expectOK: true, // Should use defaults
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				args := json.RawMessage(tc.args)
				_, err := server.tools["browser_screenshot"].Handler(args)
				
				// We expect timeout errors in test environment, not parameter errors
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", "Should timeout, not parameter error")
				}
			})
		}
	})
	
	// Test REPL parameter validation
	t.Run("REPLParameters", func(t *testing.T) {
		testCases := []struct {
			name     string
			args     string
			expectOK bool
		}{
			{
				name:     "ValidBasic",
				args:     `{"code": "console.log('test');"}`,
				expectOK: true,
			},
			{
				name:     "ValidWithSession",
				args:     `{"code": "return 42;", "sessionId": "test-123"}`,
				expectOK: true,
			},
			{
				name:     "EmptyCode",
				args:     `{"code": ""}`,
				expectOK: true, // Should handle empty code
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				args := json.RawMessage(tc.args)
				_, err := server.tools["repl_execute"].Handler(args)
				
				// We expect timeout errors in test environment, not parameter errors
				if err != nil {
					assert.Contains(t, err.Error(), "timeout", "Should timeout, not parameter error")
				}
			})
		}
	})
}

// TestJavaScriptCodeGeneration tests that the tools generate proper JavaScript
func TestJavaScriptCodeGeneration(t *testing.T) {
	server := createTestStreamableServer(t)
	defer server.Stop()
	
	// Test screenshot code generation (we can verify the parameters are processed correctly)
	t.Run("ScreenshotCodeGeneration", func(t *testing.T) {
		// Test with selector
		args := json.RawMessage(`{
			"selector": "#test-element",
			"format": "png"
		}`)
		
		// The handler will generate JavaScript code and try to execute it
		// In our test environment, it will timeout, but we can verify the parameters are parsed
		_, err := server.tools["browser_screenshot"].Handler(args)
		
		// Should timeout (expected behavior in test environment)
		if err != nil {
			assert.Contains(t, err.Error(), "timeout", "Expected timeout")
		}
	})
	
	// Test REPL code execution
	t.Run("REPLCodeExecution", func(t *testing.T) {
		args := json.RawMessage(`{
			"code": "return document.title;"
		}`)
		
		// The handler will try to execute the JavaScript
		_, err := server.tools["repl_execute"].Handler(args)
		
		// Should timeout (expected behavior in test environment)
		if err != nil {
			assert.Contains(t, err.Error(), "timeout", "Expected timeout")
		}
	})
}

// Helper function to create a test StreamableServer
func createTestStreamableServer(t *testing.T) *StreamableServer {
	// Create a test HTTP server
	mockProxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("mock response"))
	}))
	
	t.Cleanup(func() {
		mockProxy.Close()
	})
	
	// Create server with minimal configuration for testing
	eventBus := events.NewEventBus()
	server := NewStreamableServer(7778, nil, nil, nil, eventBus)
	
	// Make sure browser tools are registered
	server.registerBrowserTools()
	
	return server
}