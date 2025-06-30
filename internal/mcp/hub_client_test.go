package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHubClientInitialization tests client creation with various configurations
func TestHubClientInitialization(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid port", 8080, false},
		{"high port", 65535, false},
		{"zero port", 0, true},
		{"negative port", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewHubClient(tt.port)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, fmt.Sprintf("http://localhost:%d/mcp", tt.port), client.baseURL)
			}
		})
	}
}

// TestHubClientInitialize tests the Initialize method with various server responses
func TestHubClientInitialize(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContains string
	}{
		{
			name: "successful initialization",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/mcp", r.URL.Path)
				
				var req JSONRPCMessage
				json.NewDecoder(r.Body).Decode(&req)
				assert.Equal(t, "initialize", req.Method)
				
				resp := JSONRPCMessage{
					Jsonrpc: "2.0",
					ID:      req.ID,
					Result: map[string]interface{}{
						"protocolVersion": "1.0",
						"serverInfo": map[string]interface{}{
							"name":    "test-instance",
							"version": "1.0.0",
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
		},
		{
			name: "server returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				var req JSONRPCMessage
				json.NewDecoder(r.Body).Decode(&req)
				
				resp := JSONRPCMessage{
					Jsonrpc: "2.0",
					ID:      req.ID,
					Error: &JSONRPCError{
						Code:    -32000,
						Message: "Server not ready",
					},
				}
				json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "Server not ready",
		},
		{
			name: "server timeout",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Simulate timeout by not responding
				time.Sleep(100 * time.Millisecond)
			},
			wantErr:     true,
			errContains: "context deadline exceeded",
		},
		{
			name: "malformed response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("not json"))
			},
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name: "server unavailable",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantErr:     true,
			errContains: "HTTP 503",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := &HubClient{
				baseURL:    server.URL + "/mcp",
				httpClient: &http.Client{Timeout: 50 * time.Millisecond},
			}

			ctx := context.Background()
			if tt.name == "server timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer cancel()
			}

			err := client.Initialize(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestHubClientCallTool tests the CallTool method
func TestHubClientCallTool(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		args        map[string]interface{}
		handler     http.HandlerFunc
		wantResult  string
		wantErr     bool
		errContains string
	}{
		{
			name:     "successful tool call",
			toolName: "test_tool",
			args:     map[string]interface{}{"param": "value"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				var req JSONRPCMessage
				json.NewDecoder(r.Body).Decode(&req)
				assert.Equal(t, "tools/call", req.Method)
				
				var params map[string]interface{}
				json.Unmarshal(req.Params, &params)
				assert.Equal(t, "test_tool", params["name"])
				
				resp := JSONRPCMessage{
					Jsonrpc: "2.0",
					ID:      req.ID,
					Result: map[string]interface{}{
						"content": []map[string]interface{}{
							{"type": "text", "text": "tool executed successfully"},
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			},
			wantResult: `{"content":[{"text":"tool executed successfully","type":"text"}]}`,
			wantErr:    false,
		},
		{
			name:     "tool not found",
			toolName: "unknown_tool",
			args:     nil,
			handler: func(w http.ResponseWriter, r *http.Request) {
				var req JSONRPCMessage
				json.NewDecoder(r.Body).Decode(&req)
				
				resp := JSONRPCMessage{
					Jsonrpc: "2.0",
					ID:      req.ID,
					Error: &JSONRPCError{
						Code:    -32601,
						Message: "Tool not found: unknown_tool",
					},
				}
				json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "Tool not found",
		},
		{
			name:     "context cancelled during call",
			toolName: "slow_tool",
			args:     nil,
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Simulate slow response
				time.Sleep(100 * time.Millisecond)
			},
			wantErr:     true,
			errContains: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := &HubClient{
				baseURL: server.URL + "/mcp",
				httpClient: &http.Client{Timeout: 200 * time.Millisecond},
			}

			ctx := context.Background()
			if tt.name == "context cancelled during call" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer cancel()
			}

			result, err := client.CallTool(ctx, tt.toolName, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, string(result))
			}
		})
	}
}

// TestHubClientConcurrentRequests tests concurrent requests to the same instance
func TestHubClientConcurrentRequests(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var req JSONRPCMessage
		json.NewDecoder(r.Body).Decode(&req)
		
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		
		resp := JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("response %d", requestCount)},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &HubClient{
		baseURL: server.URL + "/mcp",
		httpClient: &http.Client{},
	}

	// Launch 5 concurrent requests (realistic scenario)
	results := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func(i int) {
			_, err := client.CallTool(context.Background(), fmt.Sprintf("tool_%d", i), nil)
			results <- err
		}(i)
	}

	// Collect results
	for i := 0; i < 5; i++ {
		err := <-results
		assert.NoError(t, err)
	}

	assert.Equal(t, 5, requestCount)
}

// TestHubClientReconnection tests client behavior during instance restart
func TestHubClientReconnection(t *testing.T) {
	var serverRunning bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !serverRunning {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		
		var req JSONRPCMessage
		json.NewDecoder(r.Body).Decode(&req)
		
		resp := JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{"status": "ok"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &HubClient{
		baseURL: server.URL + "/mcp",
		httpClient: &http.Client{Timeout: 100 * time.Millisecond},
	}

	// Instance is initially down
	serverRunning = false
	err := client.Initialize(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "503")

	// Instance comes up
	serverRunning = true
	err = client.Initialize(context.Background())
	assert.NoError(t, err)

	// Instance goes down again
	serverRunning = false
	_, err = client.CallTool(context.Background(), "test", nil)
	assert.Error(t, err)

	// Instance comes back up
	serverRunning = true
	_, err = client.CallTool(context.Background(), "test", nil)
	assert.NoError(t, err)
}

// TestHubClientListMethods tests all list methods (tools, resources, prompts)
func TestHubClientListMethods(t *testing.T) {
	methods := []struct {
		method   string
		callFunc func(*HubClient, context.Context) (json.RawMessage, error)
	}{
		{"tools/list", func(c *HubClient, ctx context.Context) (json.RawMessage, error) {
			return c.ListTools(ctx)
		}},
		{"resources/list", func(c *HubClient, ctx context.Context) (json.RawMessage, error) {
			return c.ListResources(ctx)
		}},
		{"prompts/list", func(c *HubClient, ctx context.Context) (json.RawMessage, error) {
			return c.ListPrompts(ctx)
		}},
	}

	for _, m := range methods {
		t.Run(m.method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req JSONRPCMessage
				json.NewDecoder(r.Body).Decode(&req)
				assert.Equal(t, m.method, req.Method)
				
				resp := JSONRPCMessage{
					Jsonrpc: "2.0",
					ID:      req.ID,
					Result:  map[string]interface{}{"items": []interface{}{}},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := &HubClient{
				baseURL: server.URL + "/mcp",
				httpClient: &http.Client{},
			}

			result, err := m.callFunc(client, context.Background())
			require.NoError(t, err)
			assert.Contains(t, string(result), "items")
		})
	}
}

// TestHubClientErrorPropagation tests that errors from instances are properly propagated
func TestHubClientErrorPropagation(t *testing.T) {
	errorCodes := []struct {
		code    int
		message string
		data    interface{}
	}{
		{-32700, "Parse error", nil},
		{-32600, "Invalid Request", nil},
		{-32601, "Method not found", nil},
		{-32602, "Invalid params", map[string]string{"param": "value"}},
		{-32603, "Internal error", nil},
		{-32000, "Custom error", "additional info"},
	}

	for _, ec := range errorCodes {
		t.Run(fmt.Sprintf("error_%d", ec.code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req JSONRPCMessage
				json.NewDecoder(r.Body).Decode(&req)
				
				resp := JSONRPCMessage{
					Jsonrpc: "2.0",
					ID:      req.ID,
					Error: &JSONRPCError{
						Code:    ec.code,
						Message: ec.message,
						Data:    ec.data,
					},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := &HubClient{
				baseURL: server.URL + "/mcp",
				httpClient: &http.Client{},
			}

			_, err := client.CallTool(context.Background(), "test", nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), ec.message)
			// The error format is "RPC error -32XXX: message"
		})
	}
}