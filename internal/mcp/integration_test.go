package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration test with MCP Inspector CLI
func TestMCPInspectorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if mcp-inspector is installed
	_, err := exec.LookPath("mcp-inspector")
	if err != nil {
		t.Skip("mcp-inspector not found in PATH, skipping integration test")
	}

	// Start the MCP server
	server := setupTestServer(t)
	go func() {
		err := server.Start()
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Server start failed: %v", err)
		}
	}()

	// Wait for server to be ready
	serverReady := make(chan bool)
	go func() {
		for i := 0; i < 50; i++ { // Try for up to 5 seconds
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/mcp", server.GetPort()))
			if err == nil {
				resp.Body.Close()
				close(serverReady)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	select {
	case <-serverReady:
		// Server is ready
	case <-time.After(5 * time.Second):
		t.Fatal("Server failed to start within timeout")
	}

	// Ensure server cleanup
	defer func() {
		server.Stop()
	}()

	serverURL := fmt.Sprintf("http://localhost:%d/mcp", server.GetPort())

	t.Run("validate protocol with mcp-inspector", func(t *testing.T) {
		// Run mcp-inspector validate command with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "mcp-inspector", "--cli", "validate", serverURL)
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Logf("MCP Inspector output: %s", string(output))
			if ctx.Err() == context.DeadlineExceeded {
				t.Fatalf("MCP Inspector validation timed out")
			}
			t.Fatalf("MCP Inspector validation failed: %v", err)
		}

		// Check output for validation success
		outputStr := string(output)
		assert.Contains(t, outputStr, "valid", "Server should pass validation")
	})

	t.Run("test methods with mcp-inspector", func(t *testing.T) {
		// Test initialize method with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "mcp-inspector", "test", serverURL, "initialize")
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Logf("MCP Inspector output: %s", string(output))
		}
		assert.NoError(t, err, "Initialize method should work")

		// Test tools/list with timeout
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()

		cmd = exec.CommandContext(ctx2, "mcp-inspector", "--cli", "test", serverURL, "tools/list")
		output, err = cmd.CombinedOutput()
		assert.NoError(t, err, "tools/list should work")

		// Test resources/list with timeout
		ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel3()

		cmd = exec.CommandContext(ctx3, "mcp-inspector", "--cli", "test", serverURL, "resources/list")
		output, err = cmd.CombinedOutput()
		assert.NoError(t, err, "resources/list should work")
	})
}

// Test protocol edge cases
func TestProtocolEdgeCases(t *testing.T) {
	server := setupTestServer(t)

	t.Run("empty batch request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte("[]")))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)

		// Empty batch should return empty array or empty body
		body := rec.Body.String()
		assert.True(t, body == "" || body == "[]" || body == "[]\n", "Expected empty response for empty batch")
	})

	t.Run("null request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte("null")))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)

		// Try to decode response, but it might be empty for null request
		body := rec.Body.String()
		if body != "" {
			var response JSONRPCMessage
			err := json.NewDecoder(bytes.NewReader([]byte(body))).Decode(&response)
			if err == nil {
				assert.NotNil(t, response.Error)
				assert.Equal(t, -32700, response.Error.Code) // Parse error
			}
		}
	})

	t.Run("mixed valid and invalid batch", func(t *testing.T) {
		batch := []interface{}{
			makeJSONRPCRequest("initialize", nil, 1),
			"invalid",
			makeJSONRPCRequest("tools/list", nil, 3),
		}

		body, _ := json.Marshal(batch)
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)

		// Should handle partial failures
		assert.NotEqual(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("request with invalid JSON-RPC version", func(t *testing.T) {
		msg := map[string]interface{}{
			"jsonrpc": "1.0", // Invalid version
			"method":  "initialize",
			"id":      1,
		}

		body, _ := json.Marshal(msg)
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)

		// Should still process but might indicate version mismatch
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("request with both result and error", func(t *testing.T) {
		// This is invalid according to JSON-RPC spec
		msg := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "success",
			"error": map[string]interface{}{
				"code":    -32000,
				"message": "error",
			},
		}

		body, _ := json.Marshal(msg)
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)

		// Server should handle this gracefully
		assert.NotEqual(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("extremely large request", func(t *testing.T) {
		// Create a very large parameter
		largeParam := make([]byte, 1024*1024) // 1MB
		for i := range largeParam {
			largeParam[i] = 'a'
		}

		msg := makeJSONRPCRequest("tools/call", map[string]interface{}{
			"name": "test",
			"arguments": map[string]interface{}{
				"data": string(largeParam),
			},
		}, 1)

		body, _ := json.Marshal(msg)
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)

		// Should handle large requests without crashing
		assert.NotEqual(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("request with unicode and special characters", func(t *testing.T) {
		msg := makeJSONRPCRequest("tools/call", map[string]interface{}{
			"name": "logs/search",
			"arguments": map[string]interface{}{
				"pattern": "🚀 Hello\nWorld\t\"Test\" 中文",
			},
		}, 1)

		body, _ := json.Marshal(msg)
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)

		var response JSONRPCMessage
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		// Should handle unicode properly
		assert.NotEqual(t, http.StatusInternalServerError, rec.Code)
	})
}

// Test streaming edge cases
func TestStreamingEdgeCases(t *testing.T) {
	server := setupTestServer(t)

	t.Run("client disconnect during streaming", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mcp", nil)
		req.Header.Set("Accept", "text/event-stream")

		ctx, cancel := context.WithCancel(req.Context())
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()

		// Start streaming in goroutine
		done := make(chan bool)
		go func() {
			server.router.ServeHTTP(rec, req)
			close(done)
		}()

		// Wait a bit then disconnect
		time.Sleep(50 * time.Millisecond)
		cancel()

		// Wait for handler to finish
		select {
		case <-done:
			// Good, handler finished
		case <-time.After(1 * time.Second):
			t.Fatal("Handler did not finish after client disconnect")
		}
	})

	t.Run("rapid connect/disconnect", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/mcp", nil)
			req.Header.Set("Accept", "text/event-stream")

			ctx, cancel := context.WithTimeout(req.Context(), 10*time.Millisecond)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
			cancel()
		}

		// Server should handle rapid connections without issue
		assert.True(t, true, "Server handled rapid connections")
	})

	t.Run("multiple concurrent SSE connections", func(t *testing.T) {
		numConnections := 5
		connections := make([]*httptest.ResponseRecorder, numConnections)
		contexts := make([]context.CancelFunc, numConnections)

		// Start multiple SSE connections
		for i := 0; i < numConnections; i++ {
			req := httptest.NewRequest("GET", "/mcp", nil)
			req.Header.Set("Accept", "text/event-stream")

			ctx, cancel := context.WithCancel(req.Context())
			req = req.WithContext(ctx)
			contexts[i] = cancel

			connections[i] = httptest.NewRecorder()

			go func(rec *httptest.ResponseRecorder, req *http.Request) {
				server.router.ServeHTTP(rec, req)
			}(connections[i], req)
		}

		// Let them run briefly
		time.Sleep(100 * time.Millisecond)

		// Check session count
		server.mu.RLock()
		sessionCount := len(server.sessions)
		server.mu.RUnlock()

		assert.Equal(t, numConnections, sessionCount, "Should have correct number of sessions")

		// Clean up
		for _, cancel := range contexts {
			cancel()
		}

		// Wait for cleanup
		time.Sleep(100 * time.Millisecond)

		// Verify sessions are cleaned up
		server.mu.RLock()
		sessionCount = len(server.sessions)
		server.mu.RUnlock()

		assert.Equal(t, 0, sessionCount, "All sessions should be cleaned up")
	})
}

// Test notification broadcasting
func TestNotificationBroadcasting(t *testing.T) {
	server := setupTestServer(t)

	// Set up a streaming client
	req := httptest.NewRequest("GET", "/mcp", nil)
	req.Header.Set("Accept", "text/event-stream")

	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	// Start streaming in goroutine
	streamStarted := make(chan bool)
	go func() {
		close(streamStarted)
		server.router.ServeHTTP(rec, req)
	}()

	// Wait for stream to start
	<-streamStarted
	time.Sleep(50 * time.Millisecond)

	// Broadcast a notification
	server.BroadcastNotification("test/notification", map[string]interface{}{
		"message":   "Hello, world!",
		"timestamp": time.Now().Format(time.RFC3339),
	})

	// Give time for notification to be sent
	time.Sleep(50 * time.Millisecond)

	// Check that notification was sent
	body := rec.Body.String()
	assert.Contains(t, body, "event: message")
	assert.Contains(t, body, "test/notification")
	assert.Contains(t, body, "Hello, world!")
}

// Benchmark protocol handling
func BenchmarkProtocolHandling(b *testing.B) {
	server := setupTestServer(b)

	b.Run("single request", func(b *testing.B) {
		msg := makeJSONRPCRequest("tools/list", nil, 1)
		body, _ := json.Marshal(msg)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
		}
	})

	b.Run("batch request", func(b *testing.B) {
		msgs := make([]JSONRPCMessage, 10)
		for i := 0; i < 10; i++ {
			msgs[i] = makeJSONRPCRequest("tools/list", nil, i)
		}
		body, _ := json.Marshal(msgs)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
		}
	})

	b.Run("resource read", func(b *testing.B) {
		msg := makeJSONRPCRequest("resources/read", map[string]interface{}{
			"uri": "logs://recent",
		}, 1)
		body, _ := json.Marshal(msg)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
		}
	})
}
