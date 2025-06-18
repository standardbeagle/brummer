package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers
func setupTestServer(t testing.TB) *StreamableServer {
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(10000)
	processMgr, _ := process.NewManager(".", eventBus, false)
	
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
	return server
}

func makeJSONRPCRequest(method string, params interface{}, id interface{}) JSONRPCMessage {
	paramsJSON, _ := json.Marshal(params)
	return JSONRPCMessage{
		Jsonrpc: "2.0",
		Method:  method,
		Params:  json.RawMessage(paramsJSON),
		ID:      id,
	}
}

func sendRequest(t *testing.T, server *StreamableServer, msg JSONRPCMessage) JSONRPCMessage {
	body, err := json.Marshal(msg)
	require.NoError(t, err)
	
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)
	
	var response JSONRPCMessage
	err = json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)
	
	return response
}

func sendBatchRequest(t *testing.T, server *StreamableServer, msgs []JSONRPCMessage) []JSONRPCMessage {
	body, err := json.Marshal(msgs)
	require.NoError(t, err)
	
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)
	
	var responses []JSONRPCMessage
	err = json.NewDecoder(rec.Body).Decode(&responses)
	require.NoError(t, err)
	
	return responses
}

// Test basic server setup
func TestNewStreamableServer(t *testing.T) {
	server := setupTestServer(t)
	assert.NotNil(t, server)
	assert.Equal(t, 7777, server.port)
	assert.NotNil(t, server.router)
	assert.NotNil(t, server.tools)
	assert.NotNil(t, server.resources)
	assert.NotNil(t, server.prompts)
}

// Test JSON-RPC protocol compliance
func TestJSONRPCProtocol(t *testing.T) {
	server := setupTestServer(t)
	
	t.Run("valid request", func(t *testing.T) {
		msg := makeJSONRPCRequest("initialize", nil, 1)
		response := sendRequest(t, server, msg)
		
		assert.Equal(t, "2.0", response.Jsonrpc)
		assert.Equal(t, float64(1), response.ID) // JSON numbers are decoded as float64
		assert.Nil(t, response.Error)
		assert.NotNil(t, response.Result)
	})
	
	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		
		var response JSONRPCMessage
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Equal(t, "2.0", response.Jsonrpc)
		assert.NotNil(t, response.Error)
		assert.Equal(t, -32700, response.Error.Code) // Parse error
	})
	
	t.Run("method not found", func(t *testing.T) {
		msg := makeJSONRPCRequest("nonexistent/method", nil, 2)
		response := sendRequest(t, server, msg)
		
		assert.Equal(t, "2.0", response.Jsonrpc)
		assert.Equal(t, float64(2), response.ID) // JSON numbers are decoded as float64
		assert.NotNil(t, response.Error)
		assert.Equal(t, -32601, response.Error.Code) // Method not found
	})
	
	t.Run("batch request", func(t *testing.T) {
		msgs := []JSONRPCMessage{
			makeJSONRPCRequest("initialize", nil, 1),
			makeJSONRPCRequest("tools/list", nil, 2),
			makeJSONRPCRequest("resources/list", nil, 3),
		}
		
		responses := sendBatchRequest(t, server, msgs)
		
		assert.Len(t, responses, 3)
		for i, resp := range responses {
			assert.Equal(t, "2.0", resp.Jsonrpc)
			assert.Equal(t, float64(i+1), resp.ID) // JSON numbers are decoded as float64
			assert.Nil(t, resp.Error)
			assert.NotNil(t, resp.Result)
		}
	})
	
	t.Run("notification (no ID)", func(t *testing.T) {
		// Notifications don't have an ID and don't expect a response
		msg := JSONRPCMessage{
			Jsonrpc: "2.0",
			Method:  "initialize",
		}
		
		body, _ := json.Marshal(msg)
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		
		// For notifications, we should get a valid response with the result
		responseBody := rec.Body.String()
		// Try to decode as single response first
		var singleResp JSONRPCMessage
		if err := json.Unmarshal([]byte(responseBody), &singleResp); err == nil {
			// Single response - notifications should still get a response for initialize
			assert.Equal(t, "2.0", singleResp.Jsonrpc)
			assert.NotNil(t, singleResp.Result)
		} else {
			// Could be array or empty
			assert.True(t, responseBody == "" || responseBody == "[]" || responseBody == "[]\n" || responseBody == "null\n")
		}
	})
}

// Test initialize method
func TestInitialize(t *testing.T) {
	server := setupTestServer(t)
	
	msg := makeJSONRPCRequest("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "test-client",
			"version": "1.0.0",
		},
	}, 1)
	
	response := sendRequest(t, server, msg)
	
	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Result)
	
	result := response.Result.(map[string]interface{})
	assert.Equal(t, "2024-11-05", result["protocolVersion"])
	
	serverInfo := result["serverInfo"].(map[string]interface{})
	assert.Equal(t, "brummer-mcp", serverInfo["name"])
	assert.Equal(t, "2.0.0", serverInfo["version"])
	
	capabilities := result["capabilities"].(map[string]interface{})
	assert.NotNil(t, capabilities["tools"])
	assert.NotNil(t, capabilities["resources"])
	assert.NotNil(t, capabilities["prompts"])
}

// Test tools/list
func TestToolsList(t *testing.T) {
	server := setupTestServer(t)
	
	msg := makeJSONRPCRequest("tools/list", nil, 1)
	response := sendRequest(t, server, msg)
	
	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Result)
	
	result := response.Result.(map[string]interface{})
	tools := result["tools"].([]interface{})
	
	// Check that we have tools registered
	assert.Greater(t, len(tools), 0)
	
	// Verify tool structure
	for _, toolInterface := range tools {
		tool := toolInterface.(map[string]interface{})
		assert.NotEmpty(t, tool["name"])
		assert.NotEmpty(t, tool["description"])
		// inputSchema is optional but should be valid if present
		if schema, ok := tool["inputSchema"]; ok && schema != nil {
			assert.IsType(t, map[string]interface{}{}, schema)
		}
	}
}

// Test SSE streaming
func TestSSEStreaming(t *testing.T) {
	server := setupTestServer(t)
	
	t.Run("GET with SSE accept header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mcp", nil)
		req.Header.Set("Accept", "text/event-stream")
		
		rec := httptest.NewRecorder()
		
		// Use a channel to signal when to stop
		done := make(chan bool)
		go func() {
			time.Sleep(50 * time.Millisecond) // Reduced delay for faster tests
			close(done)
		}()
		
		// Create a context that will be cancelled
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		req = req.WithContext(ctx)
		
		// Start the handler in a goroutine
		go func() {
			server.router.ServeHTTP(rec, req)
		}()
		
		// Wait for initial response
		time.Sleep(20 * time.Millisecond) // Reduced delay
		
		// Check headers
		assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
		assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
		
		// Check that we got SSE comments
		body := rec.Body.String()
		assert.Contains(t, body, ": MCP Streamable HTTP Transport")
		assert.Contains(t, body, ": Session-Id:")
		
		cancel() // Stop the streaming
	})
	
	t.Run("POST with SSE accept header", func(t *testing.T) {
		msg := makeJSONRPCRequest("initialize", nil, 1)
		body, _ := json.Marshal(msg)
		
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		
		// Check headers
		assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
		
		// Parse SSE response
		responseBody := rec.Body.String()
		assert.Contains(t, responseBody, "data: ")
		
		// Extract JSON from SSE data
		lines := strings.Split(responseBody, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "data: ") {
				jsonData := strings.TrimPrefix(line, "data: ")
				var response JSONRPCMessage
				err := json.Unmarshal([]byte(jsonData), &response)
				assert.NoError(t, err)
				assert.Equal(t, "2.0", response.Jsonrpc)
				assert.Equal(t, float64(1), response.ID) // JSON numbers are decoded as float64
				break
			}
		}
	})
}

// Test session management
func TestSessionManagement(t *testing.T) {
	server := setupTestServer(t)
	
	t.Run("session creation with custom ID", func(t *testing.T) {
		sessionID := "test-session-123"
		req := httptest.NewRequest("GET", "/mcp", nil)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Mcp-Session-Id", sessionID)
		
		rec := httptest.NewRecorder()
		
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		req = req.WithContext(ctx)
		
		// Start streaming in goroutine
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			server.router.ServeHTTP(rec, req)
		}()
		
		// Wait for session to be established
		time.Sleep(50 * time.Millisecond)
		
		// Check session exists
		server.mu.RLock()
		session, exists := server.sessions[sessionID]
		server.mu.RUnlock()
		
		assert.True(t, exists)
		assert.Equal(t, sessionID, session.ID)
		assert.True(t, session.StreamingActive)
		
		// Clean up
		cancel()
		wg.Wait()
		
		// Verify session is cleaned up
		server.mu.RLock()
		_, exists = server.sessions[sessionID]
		server.mu.RUnlock()
		assert.False(t, exists)
	})
}

// Test resource subscriptions
func TestResourceSubscriptions(t *testing.T) {
	server := setupTestServer(t)
	
	t.Run("subscribe to resource", func(t *testing.T) {
		sessionID := "sub-test-123"
		
		// Subscribe to a resource
		msg := makeJSONRPCRequest("resources/subscribe", map[string]string{
			"uri": "logs://recent",
		}, 1)
		
		body, _ := json.Marshal(msg)
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Mcp-Session-Id", sessionID)
		
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		
		var response JSONRPCMessage
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Nil(t, response.Error)
		
		// Verify subscription is recorded
		server.subscriptionsMu.RLock()
		subs, exists := server.subscriptions[sessionID]
		server.subscriptionsMu.RUnlock()
		
		assert.True(t, exists)
		assert.True(t, subs["logs://recent"])
	})
	
	t.Run("unsubscribe from resource", func(t *testing.T) {
		sessionID := "unsub-test-123"
		
		// First subscribe
		server.subscriptionsMu.Lock()
		server.subscriptions[sessionID] = map[string]bool{
			"logs://recent": true,
		}
		server.subscriptionsMu.Unlock()
		
		// Unsubscribe
		msg := makeJSONRPCRequest("resources/unsubscribe", map[string]string{
			"uri": "logs://recent",
		}, 1)
		
		body, _ := json.Marshal(msg)
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Mcp-Session-Id", sessionID)
		
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		
		var response JSONRPCMessage
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Nil(t, response.Error)
		
		// Verify subscription is removed
		server.subscriptionsMu.RLock()
		subs := server.subscriptions[sessionID]
		server.subscriptionsMu.RUnlock()
		
		assert.False(t, subs["logs://recent"])
	})
}

// Test error handling
func TestErrorHandling(t *testing.T) {
	server := setupTestServer(t)
	
	testCases := []struct {
		name         string
		request      JSONRPCMessage
		expectedCode int
		expectedMsg  string
	}{
		{
			name: "invalid params",
			request: JSONRPCMessage{
				Jsonrpc: "2.0",
				Method:  "tools/call",
				Params:  json.RawMessage(`{"invalid": "params"}`),
				ID:      1,
			},
			expectedCode: -32602,
			expectedMsg:  "Invalid params",
		},
		{
			name: "tool not found",
			request: JSONRPCMessage{
				Jsonrpc: "2.0",
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name": "nonexistent", "arguments": {}}`),
				ID:      1,
			},
			expectedCode: -32602,
			expectedMsg:  "Tool not found",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := sendRequest(t, server, tc.request)
			
			assert.NotNil(t, response.Error)
			assert.Equal(t, tc.expectedCode, response.Error.Code)
			assert.Contains(t, response.Error.Message, tc.expectedMsg)
		})
	}
}

// Test concurrent requests
func TestConcurrentRequests(t *testing.T) {
	server := setupTestServer(t)
	
	numRequests := 50
	var wg sync.WaitGroup
	errors := make(chan error, numRequests)
	
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			msg := makeJSONRPCRequest("tools/list", nil, id)
			body, _ := json.Marshal(msg)
			
			req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			
			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
			
			var response JSONRPCMessage
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				errors <- err
				return
			}
			
			if response.Error != nil {
				errors <- fmt.Errorf("request %d failed: %v", id, response.Error)
				return
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent request failed: %v", err)
	}
}

// Test health endpoint
func TestHealthEndpoint(t *testing.T) {
	server := setupTestServer(t)
	
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	
	server.router.ServeHTTP(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	
	var health map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&health)
	require.NoError(t, err)
	
	assert.Equal(t, "healthy", health["status"])
	assert.Equal(t, "streamable", health["mode"])
	assert.Equal(t, float64(0), health["sessions"])
}

// Test CORS headers
func TestCORSHeaders(t *testing.T) {
	server := setupTestServer(t)
	
	// Note: corsMiddleware is applied in server.Start(), not in router setup
	// For testing, we need to apply it manually
	handler := corsMiddleware(server.router)
	
	req := httptest.NewRequest("OPTIONS", "/mcp", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	
	// Check CORS headers
	assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Headers"))
}

// Benchmark message processing
func BenchmarkMessageProcessing(b *testing.B) {
	server := setupTestServer(b)
	msg := makeJSONRPCRequest("tools/list", nil, 1)
	body, _ := json.Marshal(msg)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
	}
}

// Benchmark SSE streaming
func BenchmarkSSEStreaming(b *testing.B) {
	server := setupTestServer(b)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/mcp", nil)
		req.Header.Set("Accept", "text/event-stream")
		
		ctx, cancel := context.WithTimeout(req.Context(), 10*time.Millisecond)
		req = req.WithContext(ctx)
		
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		cancel()
	}
}