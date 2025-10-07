package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProxyServer creates a mock proxy server with telemetry endpoints
func MockProxyServer() *httptest.Server {
	mux := http.NewServeMux()

	// Mock telemetry endpoint
	mux.HandleFunc("/__brummer_telemetry__", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" && r.Method != "OPTIONS" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Read telemetry data
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request", http.StatusBadRequest)
			return
		}

		var telemetryData map[string]interface{}
		if err := json.Unmarshal(body, &telemetryData); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Echo back the telemetry data with a timestamp
		response := map[string]interface{}{
			"status": "received",
			"timestamp": time.Now().Format(time.RFC3339),
			"receivedData": telemetryData,
			"sessionId": telemetryData["sessionId"],
		}

		json.NewEncoder(w).Encode(response)
	})

	// Mock WebSocket telemetry endpoint
	mux.HandleFunc("/__brummer_ws__", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for testing
			},
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		// Send mock telemetry data
		telemetryEvents := []map[string]interface{}{
			{
				"type": "page_load",
				"url": "http://localhost:3000/",
				"timestamp": time.Now().Add(-5 * time.Second).Format(time.RFC3339),
				"metrics": map[string]interface{}{
					"loadTime": 1500,
					"domContentLoaded": 800,
					"firstPaint": 1200,
				},
			},
			{
				"type": "console_error",
				"url": "http://localhost:3000/",
				"timestamp": time.Now().Add(-3 * time.Second).Format(time.RFC3339),
				"error": map[string]interface{}{
					"message": "Cannot read property 'undefined' of undefined",
					"filename": "/src/main.js",
					"lineNumber": 42,
					"columnNumber": 15,
				},
			},
			{
				"type": "network_request",
				"url": "http://localhost:3000/api/data",
				"timestamp": time.Now().Add(-2 * time.Second).Format(time.RFC3339),
				"request": map[string]interface{}{
					"method": "GET",
					"status": 200,
					"duration": 45,
				},
			},
		}

		for _, event := range telemetryEvents {
			if err := conn.WriteJSON(event); err != nil {
				return
			}
			time.Sleep(100 * time.Millisecond) // Simulate real-time streaming
		}

		// Keep connection open for a bit to simulate real usage
		time.Sleep(1 * time.Second)
	})

	// Mock reverse proxy endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && r.Method == "GET" {
			// Mock root endpoint for reverse proxy info
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"message": "Reverse proxy mode active. Each URL gets its own port.",
				"mode": "reverse_proxy",
				"timestamp": time.Now().Format(time.RFC3339),
			}
			json.NewEncoder(w).Encode(response)
		} else {
			// Mock proxied request
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"proxied": true,
				"path": r.URL.Path,
				"method": r.Method,
				"headers": r.Header,
				"timestamp": time.Now().Format(time.RFC3339),
			}
			json.NewEncoder(w).Encode(response)
		}
	})

	return httptest.NewServer(mux)
}

// TestProxyTelemetryEndpoint tests the proxy telemetry collection
func TestProxyTelemetryEndpoint(t *testing.T) {
	server := MockProxyServer()
	defer server.Close()

	client := server.Client()

	t.Run("POST telemetry data", func(t *testing.T) {
		telemetryData := map[string]interface{}{
			"sessionId": "session-123",
			"url": "http://localhost:3000/",
			"type": "page_load",
			"metrics": map[string]interface{}{
				"loadTime": 1500,
				"domContentLoaded": 800,
				"firstPaint": 1200,
			},
			"timestamp": time.Now().Format(time.RFC3339),
		}

		jsonBody, err := json.Marshal(telemetryData)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/__brummer_telemetry__", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "received", response["status"])
		assert.Equal(t, "session-123", response["sessionId"])
		assert.Contains(t, response, "timestamp")
		assert.Contains(t, response, "receivedData")

		receivedData := response["receivedData"].(map[string]interface{})
		assert.Equal(t, "session-123", receivedData["sessionId"])
		assert.Equal(t, "page_load", receivedData["type"])

		t.Logf("✅ Telemetry data sent and received successfully")
	})

	t.Run("OPTIONS request for CORS", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", server.URL+"/__brummer_telemetry__", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "POST")

		t.Logf("✅ CORS preflight handled correctly")
	})

	t.Run("Invalid telemetry data", func(t *testing.T) {
		invalidJSON := `{ sessionId: "123", invalid }`

		req, err := http.NewRequest("POST", server.URL+"/__brummer_telemetry__", bytes.NewBufferString(invalidJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		t.Logf("✅ Invalid telemetry data properly rejected")
	})

	t.Run("Wrong HTTP method", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/__brummer_telemetry__", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

		t.Logf("✅ Wrong HTTP method properly rejected")
	})
}

// TestProxyWebSocketTelemetry tests the WebSocket telemetry streaming
func TestProxyWebSocketTelemetry(t *testing.T) {
	server := MockProxyServer()
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + server.URL[4:] + "/__brummer_ws__"

	t.Run("WebSocket telemetry streaming", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Set read deadline to prevent hanging
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		var events []map[string]interface{}

		// Read multiple telemetry events
		for i := 0; i < 3; i++ {
			var event map[string]interface{}
			err := conn.ReadJSON(&event)
			require.NoError(t, err)
			events = append(events, event)
		}

		assert.Len(t, events, 3)

		// Verify first event (page_load)
		pageLoadEvent := events[0]
		assert.Equal(t, "page_load", pageLoadEvent["type"])
		assert.Equal(t, "http://localhost:3000/", pageLoadEvent["url"])
		assert.Contains(t, pageLoadEvent, "metrics")

		metrics := pageLoadEvent["metrics"].(map[string]interface{})
		assert.Contains(t, metrics, "loadTime")
		assert.Equal(t, float64(1500), metrics["loadTime"])

		// Verify second event (console_error)
		errorEvent := events[1]
		assert.Equal(t, "console_error", errorEvent["type"])
		assert.Contains(t, errorEvent, "error")

		error := errorEvent["error"].(map[string]interface{})
		assert.Equal(t, "Cannot read property 'undefined' of undefined", error["message"])
		assert.Equal(t, "/src/main.js", error["filename"])
		assert.Equal(t, float64(42), error["lineNumber"])

		// Verify third event (network_request)
		networkEvent := events[2]
		assert.Equal(t, "network_request", networkEvent["type"])
		assert.Equal(t, "http://localhost:3000/api/data", networkEvent["url"])

		request := networkEvent["request"].(map[string]interface{})
		assert.Equal(t, "GET", request["method"])
		assert.Equal(t, float64(200), request["status"])

		t.Logf("✅ WebSocket telemetry streaming works correctly")
	})

	t.Run("WebSocket connection timeout", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Set a short read deadline to test timeout behavior
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))

		// Try to read after the server closes the connection
		var event map[string]interface{}
		err = conn.ReadJSON(&event)
		// Should get an error due to connection closing or timeout
		assert.Error(t, err)

		t.Logf("✅ WebSocket timeout handling works correctly")
	})
}

// TestProxyReverseMode tests the reverse proxy functionality
func TestProxyReverseMode(t *testing.T) {
	server := MockProxyServer()
	defer server.Close()

	client := server.Client()

	t.Run("Root endpoint info", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "Reverse proxy mode active. Each URL gets its own port.", response["message"])
		assert.Equal(t, "reverse_proxy", response["mode"])
		assert.Contains(t, response, "timestamp")

		t.Logf("✅ Root endpoint shows proxy info correctly")
	})

	t.Run("Mock proxied request", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/api/users", nil)
		require.NoError(t, err)
		req.Header.Set("X-Test-Header", "test-value")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, true, response["proxied"])
		assert.Equal(t, "/api/users", response["path"])
		assert.Equal(t, "GET", response["method"])
		assert.Contains(t, response, "headers")
		assert.Contains(t, response, "timestamp")

		t.Logf("✅ Reverse proxy handles requests correctly")
	})

	t.Run("Different HTTP methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

		for _, method := range methods {
			req, err := http.NewRequest(method, server.URL+"/test", nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			assert.Equal(t, method, response["method"])
		}

		t.Logf("✅ All HTTP methods handled correctly")
	})
}

// TestProxyPerformance tests proxy server performance
func TestProxyPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := MockProxyServer()
	defer server.Close()

	client := server.Client()

	t.Run("Telemetry endpoint performance", func(t *testing.T) {
		telemetryData := map[string]interface{}{
			"sessionId": "perf-test",
			"type": "performance_test",
			"data": make([]byte, 1024), // 1KB of data
		}

		jsonBody, err := json.Marshal(telemetryData)
		require.NoError(t, err)

		const numRequests = 100
		start := time.Now()

		for i := 0; i < numRequests; i++ {
			req, err := http.NewRequest("POST", server.URL+"/__brummer_telemetry__", bytes.NewBuffer(jsonBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			resp.Body.Close()
		}

		duration := time.Since(start)
		avgDuration := duration / numRequests

		t.Logf("📊 Telemetry Performance: %d requests in %v (%.2fms per request)",
			numRequests, duration, avgDuration.Seconds()*1000)

		// Should handle requests quickly (less than 10ms per request on average)
		assert.Less(t, avgDuration, 10*time.Millisecond, "Telemetry endpoint should be fast")
	})

	t.Run("Concurrent telemetry requests", func(t *testing.T) {
		const numConcurrent = 20
		const requestsPerGoroutine = 5

		errChan := make(chan error, numConcurrent)
		start := time.Now()

		for i := 0; i < numConcurrent; i++ {
			go func(goroutineID int) {
				for j := 0; j < requestsPerGoroutine; j++ {
					telemetryData := map[string]interface{}{
						"sessionId": fmt.Sprintf("concurrent-test-%d", goroutineID),
						"type": "concurrent_test",
						"iteration": j,
					}

					jsonBody, err := json.Marshal(telemetryData)
					if err != nil {
						errChan <- err
						return
					}

					req, err := http.NewRequest("POST", server.URL+"/__brummer_telemetry__", bytes.NewBuffer(jsonBody))
					if err != nil {
						errChan <- err
						return
					}
					req.Header.Set("Content-Type", "application/json")

					resp, err := client.Do(req)
					if err != nil {
						errChan <- err
						return
					}
					resp.Body.Close()
				}
				errChan <- nil
			}(i)
		}

		// Wait for all goroutines to complete
		var errors []error
		for i := 0; i < numConcurrent; i++ {
			if err := <-errChan; err != nil {
				errors = append(errors, err)
			}
		}

		duration := time.Since(start)
		totalRequests := numConcurrent * requestsPerGoroutine

		if len(errors) > 0 {
			t.Logf("❌ %d concurrent requests failed", len(errors))
			for _, err := range errors {
				t.Logf("  Error: %v", err)
			}
		} else {
			t.Logf("✅ All %d concurrent telemetry requests completed successfully in %v (%.2fms per request)",
				totalRequests, duration, float64(duration.Nanoseconds())/float64(totalRequests)/1000000)
		}

		assert.Empty(t, errors, "No concurrent requests should fail")
		assert.Less(t, duration, 5*time.Second, "Concurrent requests should complete quickly")
	})
}