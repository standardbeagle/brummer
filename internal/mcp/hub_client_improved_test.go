package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImprovedHubClient_Timeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "delayed",
		})
	}))
	defer server.Close()
	
	// Extract port from test server
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port := 0
	fmt.Sscanf(portStr, "%d", &port)
	
	// Create client with short timeout
	config := DefaultClientConfig
	config.RequestTimeout = 100 * time.Millisecond
	client, err := NewImprovedHubClient(port, &config)
	require.NoError(t, err)
	defer client.Close()
	
	// Make request that will timeout
	ctx := context.Background()
	err = client.Initialize(ctx)
	assert.Error(t, err)
	
	// Check metrics
	metrics := client.GetMetrics()
	assert.GreaterOrEqual(t, metrics["timeout_requests"].(uint64), uint64(1))
}

func TestImprovedHubClient_Retry(t *testing.T) {
	var attempts atomic.Int32
	
	// Create a test server that fails first 2 times
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "success",
		})
	}))
	defer server.Close()
	
	// Extract port
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port := 0
	fmt.Sscanf(portStr, "%d", &port)
	
	// Create client with retries
	config := DefaultClientConfig
	config.MaxRetries = 3
	config.RetryDelay = 10 * time.Millisecond
	client, err := NewImprovedHubClient(port, &config)
	require.NoError(t, err)
	defer client.Close()
	
	// Make request that will succeed on 3rd attempt
	ctx := context.Background()
	err = client.Initialize(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestImprovedHubClient_CircuitBreaker(t *testing.T) {
	// Create a test server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	
	// Extract port
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port := 0
	fmt.Sscanf(portStr, "%d", &port)
	
	// Create client with circuit breaker
	config := DefaultClientConfig
	config.FailureThreshold = 3
	config.RecoveryTimeout = 100 * time.Millisecond
	config.MaxRetries = 0 // No retries to test circuit breaker faster
	client, err := NewImprovedHubClient(port, &config)
	require.NoError(t, err)
	defer client.Close()
	
	ctx := context.Background()
	
	// Make requests until circuit opens
	for i := 0; i < 3; i++ {
		err = client.Ping(ctx)
		assert.Error(t, err)
	}
	
	// Circuit should be open now
	metrics := client.GetMetrics()
	assert.Equal(t, "open", metrics["circuit_state"])
	
	// Next request should fail immediately
	start := time.Now()
	err = client.Ping(ctx)
	duration := time.Since(start)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
	assert.Less(t, duration, 10*time.Millisecond) // Should fail fast
	
	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)
	
	// Circuit should transition to half-open
	err = client.Ping(ctx)
	assert.Error(t, err) // Still fails, but circuit is half-open
	
	metrics = client.GetMetrics()
	assert.Equal(t, "half-open", metrics["circuit_state"])
}

func TestImprovedHubClient_ConcurrentRequests(t *testing.T) {
	var requestCount atomic.Int32
	
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		
		// Parse request
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		
		// Echo back the ID
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result": map[string]interface{}{
				"tools": []interface{}{},
			},
		})
	}))
	defer server.Close()
	
	// Extract port
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port := 0
	fmt.Sscanf(portStr, "%d", &port)
	
	// Create client
	client, err := NewImprovedHubClient(port, nil)
	require.NoError(t, err)
	defer client.Close()
	
	// Make concurrent requests
	var wg sync.WaitGroup
	const numRequests = 100
	
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			_, err := client.ListTools(ctx)
			assert.NoError(t, err)
		}()
	}
	
	wg.Wait()
	assert.Equal(t, int32(numRequests), requestCount.Load())
}

func TestImprovedHubClient_ContextCancellation(t *testing.T) {
	// Create a test server that delays
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "too late",
		})
	}))
	defer server.Close()
	
	// Extract port
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port := 0
	fmt.Sscanf(portStr, "%d", &port)
	
	// Create client
	client, err := NewImprovedHubClient(port, nil)
	require.NoError(t, err)
	defer client.Close()
	
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start request
	errCh := make(chan error)
	go func() {
		errCh <- client.Ping(ctx)
	}()
	
	// Cancel after short delay
	time.Sleep(50 * time.Millisecond)
	cancel()
	
	// Should get context error
	err = <-errCh
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestImprovedHubClient_LargeResponse(t *testing.T) {
	// Create large response data
	largeData := make([]interface{}, 1000)
	for i := range largeData {
		largeData[i] = map[string]interface{}{
			"name":        fmt.Sprintf("tool_%d", i),
			"description": "A test tool with a reasonably long description",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"param1": map[string]string{"type": "string"},
					"param2": map[string]string{"type": "number"},
				},
			},
		}
	}
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"tools": largeData,
			},
		})
	}))
	defer server.Close()
	
	// Extract port
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port := 0
	fmt.Sscanf(portStr, "%d", &port)
	
	// Create client
	client, err := NewImprovedHubClient(port, nil)
	require.NoError(t, err)
	defer client.Close()
	
	// Make request
	ctx := context.Background()
	result, err := client.ListTools(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	
	// Verify we got the data
	var parsed map[string]interface{}
	err = json.Unmarshal(result, &parsed)
	assert.NoError(t, err)
	tools := parsed["tools"].([]interface{})
	assert.Len(t, tools, 1000)
}

func TestImprovedHubClient_ConnectionPooling(t *testing.T) {
	var (
		connectionCount atomic.Int32
		activeConns     atomic.Int32
	)
	
	// Create test server that tracks connections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connectionCount.Add(1)
		activeConns.Add(1)
		defer activeConns.Add(-1)
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "ok",
		})
	}))
	defer server.Close()
	
	// Extract port
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port := 0
	fmt.Sscanf(portStr, "%d", &port)
	
	// Create client with connection pooling
	config := DefaultClientConfig
	config.MaxConnsPerHost = 2
	client, err := NewImprovedHubClient(port, &config)
	require.NoError(t, err)
	defer client.Close()
	
	// Make sequential requests (should reuse connection)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		err := client.Ping(ctx)
		assert.NoError(t, err)
	}
	
	// Should have reused connections
	assert.Less(t, connectionCount.Load(), int32(5))
}

func TestImprovedHubClient_RPCError(t *testing.T) {
	// Create test server that returns RPC error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
				"data":    "The method 'unknown' does not exist",
			},
		})
	}))
	defer server.Close()
	
	// Extract port
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port := 0
	fmt.Sscanf(portStr, "%d", &port)
	
	// Create client
	client, err := NewImprovedHubClient(port, nil)
	require.NoError(t, err)
	defer client.Close()
	
	// Make request
	ctx := context.Background()
	_, err = client.CallTool(ctx, "unknown", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RPC error")
	assert.Contains(t, err.Error(), "Method not found")
}

// Benchmark connection reuse
func BenchmarkImprovedHubClient_ConnectionReuse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "ok",
		})
	}))
	defer server.Close()
	
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port := 0
	fmt.Sscanf(portStr, "%d", &port)
	
	client, _ := NewImprovedHubClient(port, nil)
	defer client.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client.Ping(ctx)
		}
	})
}