package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/pkg/events"
)

func TestProxyServer(t *testing.T) {
	eventBus := events.NewEventBus()

	t.Run("NewServer", func(t *testing.T) {
		server := NewServer(0, eventBus) // Random port
		if server == nil {
			t.Fatal("Failed to create proxy server")
		}
		
		// Default mode should be reverse
		if server.mode != ProxyModeReverse {
			t.Errorf("Expected default mode to be reverse, got %v", server.mode)
		}
	})

	t.Run("StartStop", func(t *testing.T) {
		server := NewServer(0, eventBus)
		
		err := server.Start()
		if err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}
		
		if !server.IsRunning() {
			t.Error("Server should be running")
		}
		
		port := server.GetPort()
		if port == 0 {
			t.Error("Server port should be non-zero")
		}
		
		err = server.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
		
		if server.IsRunning() {
			t.Error("Server should not be running after stop")
		}
	})

	t.Run("ReverseProxyMode", func(t *testing.T) {
		// Create test backend server
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Hello from backend",
				"path":    r.URL.Path,
			})
		}))
		defer backend.Close()

		// Create proxy server in reverse mode
		server := NewServerWithMode(0, ProxyModeReverse, eventBus)
		err := server.Start()
		if err != nil {
			t.Fatalf("Failed to start proxy: %v", err)
		}
		defer server.Stop()

		// Register backend URL
		proxyURL := server.RegisterURL(backend.URL, "test-service")
		if proxyURL == backend.URL {
			t.Error("Expected proxy URL to be different from backend URL")
		}

		// Test proxied request
		resp, err := http.Get(proxyURL + "/test")
		if err != nil {
			t.Fatalf("Failed to make proxied request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]string
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["message"] != "Hello from backend" {
			t.Errorf("Expected message from backend, got %s", result["message"])
		}
	})

	t.Run("FullProxyMode", func(t *testing.T) {
		// Create test target server
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Target response"))
		}))
		defer target.Close()

		// Create proxy server in full mode
		server := NewServerWithMode(0, ProxyModeFull, eventBus)
		err := server.Start()
		if err != nil {
			t.Fatalf("Failed to start proxy: %v", err)
		}
		defer server.Stop()

		// Create HTTP client configured to use proxy
		proxyURL := fmt.Sprintf("http://localhost:%d", server.GetPort())
		client := &http.Client{
			Transport: &http.Transport{
				Proxy: func(req *http.Request) (*http.URL, error) {
					return http.ParseURL(proxyURL)
				},
			},
		}

		// Make request through proxy
		resp, err := client.Get(target.URL)
		if err != nil {
			t.Fatalf("Failed to make proxied request: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		if string(body) != "Target response" {
			t.Errorf("Expected 'Target response', got %s", string(body))
		}
	})

	t.Run("RequestCapture", func(t *testing.T) {
		// Create test backend
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		defer backend.Close()

		server := NewServerWithMode(0, ProxyModeReverse, eventBus)
		err := server.Start()
		if err != nil {
			t.Fatalf("Failed to start proxy: %v", err)
		}
		defer server.Stop()

		// Register and make request
		proxyURL := server.RegisterURL(backend.URL, "test")
		
		// Make POST request with body
		requestBody := `{"test": "data"}`
		resp, err := http.Post(proxyURL, "application/json", strings.NewReader(requestBody))
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		resp.Body.Close()

		// Give time for request to be captured
		time.Sleep(100 * time.Millisecond)

		// Check captured requests
		requests := server.GetCapturedRequests()
		if len(requests) == 0 {
			t.Error("No requests were captured")
		}

		// Verify request details
		capturedReq := requests[0]
		if capturedReq.Method != "POST" {
			t.Errorf("Expected POST method, got %s", capturedReq.Method)
		}
		if capturedReq.Body != requestBody {
			t.Errorf("Expected body %s, got %s", requestBody, capturedReq.Body)
		}
	})

	t.Run("PACFile", func(t *testing.T) {
		server := NewServerWithMode(0, ProxyModeFull, eventBus)
		err := server.Start()
		if err != nil {
			t.Fatalf("Failed to start proxy: %v", err)
		}
		defer server.Stop()

		// Get PAC file URL
		pacURL := server.GetPACURL()
		if pacURL == "" {
			t.Error("PAC URL should not be empty")
		}

		// Fetch PAC file
		resp, err := http.Get(pacURL)
		if err != nil {
			t.Fatalf("Failed to fetch PAC file: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for PAC file, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read PAC file: %v", err)
		}

		pacContent := string(body)
		if !strings.Contains(pacContent, "function FindProxyForURL") {
			t.Error("PAC file should contain FindProxyForURL function")
		}

		expectedProxy := fmt.Sprintf("PROXY localhost:%d", server.GetPort())
		if !strings.Contains(pacContent, expectedProxy) {
			t.Errorf("PAC file should contain proxy directive: %s", expectedProxy)
		}
	})

	t.Run("URLRegistration", func(t *testing.T) {
		server := NewServerWithMode(0, ProxyModeReverse, eventBus)
		err := server.Start()
		if err != nil {
			t.Fatalf("Failed to start proxy: %v", err)
		}
		defer server.Stop()

		// Register multiple URLs
		url1 := "http://localhost:3000"
		url2 := "http://localhost:4000"
		
		proxy1 := server.RegisterURL(url1, "service1")
		proxy2 := server.RegisterURL(url2, "service2")

		if proxy1 == proxy2 {
			t.Error("Different URLs should get different proxy URLs")
		}

		// Register same URL again
		proxy1Again := server.RegisterURL(url1, "service1")
		if proxy1 != proxy1Again {
			t.Error("Same URL should get same proxy URL")
		}

		// Check URL mapping
		retrievedURL := server.GetProxyURL(url1)
		if retrievedURL != proxy1 {
			t.Errorf("Expected proxy URL %s, got %s", proxy1, retrievedURL)
		}
	})

	t.Run("WebSocketSupport", func(t *testing.T) {
		// This would require websocket server setup
		// For now, just verify the proxy doesn't crash with upgrade headers
		server := NewServerWithMode(0, ProxyModeReverse, eventBus)
		err := server.Start()
		if err != nil {
			t.Fatalf("Failed to start proxy: %v", err)
		}
		defer server.Stop()

		// Create request with websocket upgrade headers
		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d", server.GetPort()), nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Key", "test-key")
		req.Header.Set("Sec-WebSocket-Version", "13")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make websocket request: %v", err)
		}
		defer resp.Body.Close()

		// Should handle gracefully (may return error, but shouldn't crash)
		// The exact response depends on the backend
	})

	t.Run("CORSHeaders", func(t *testing.T) {
		// Create backend that sets CORS headers
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		defer backend.Close()

		server := NewServerWithMode(0, ProxyModeReverse, eventBus)
		err := server.Start()
		if err != nil {
			t.Fatalf("Failed to start proxy: %v", err)
		}
		defer server.Stop()

		proxyURL := server.RegisterURL(backend.URL, "cors-service")

		// Make OPTIONS request
		req, _ := http.NewRequest("OPTIONS", proxyURL, nil)
		req.Header.Set("Origin", "http://localhost:3000")
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make OPTIONS request: %v", err)
		}
		defer resp.Body.Close()

		// Verify CORS headers are preserved
		allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
		if allowOrigin != "*" {
			t.Errorf("Expected CORS header to be preserved, got %s", allowOrigin)
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		server := NewServerWithMode(0, ProxyModeReverse, eventBus)
		err := server.Start()
		if err != nil {
			t.Fatalf("Failed to start proxy: %v", err)
		}
		defer server.Stop()

		// Register non-existent backend
		proxyURL := server.RegisterURL("http://localhost:99999", "non-existent")

		// Make request to non-existent backend
		resp, err := http.Get(proxyURL)
		if err != nil {
			t.Fatalf("Request should not fail at HTTP level: %v", err)
		}
		defer resp.Body.Close()

		// Should return 502 Bad Gateway or similar error
		if resp.StatusCode < 500 {
			t.Errorf("Expected 5xx error for non-existent backend, got %d", resp.StatusCode)
		}
	})

	t.Run("TelemetryIntegration", func(t *testing.T) {
		// Track events published by proxy
		var telemetryEvents []events.Event
		eventBus.Subscribe(events.EventType("proxy.request"), func(e events.Event) {
			telemetryEvents = append(telemetryEvents, e)
		})

		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		defer backend.Close()

		server := NewServerWithMode(0, ProxyModeReverse, eventBus)
		err := server.Start()
		if err != nil {
			t.Fatalf("Failed to start proxy: %v", err)
		}
		defer server.Stop()

		proxyURL := server.RegisterURL(backend.URL, "telemetry-test")

		// Make request
		resp, err := http.Get(proxyURL)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		resp.Body.Close()

		// Give time for events to be processed
		time.Sleep(100 * time.Millisecond)

		// Verify telemetry events were published
		// This depends on the actual implementation
		// For now, just verify the server handled the request
		requests := server.GetCapturedRequests()
		if len(requests) == 0 {
			t.Error("Expected at least one captured request")
		}
	})
}

func TestProxyModes(t *testing.T) {
	eventBus := events.NewEventBus()

	t.Run("ModeString", func(t *testing.T) {
		tests := []struct {
			mode     ProxyMode
			expected string
		}{
			{ProxyModeReverse, "reverse"},
			{ProxyModeFull, "full"},
		}

		for _, tt := range tests {
			if tt.mode.String() != tt.expected {
				t.Errorf("Expected mode string %s, got %s", tt.expected, tt.mode.String())
			}
		}
	})

	t.Run("NewServerWithMode", func(t *testing.T) {
		server := NewServerWithMode(0, ProxyModeFull, eventBus)
		if server.mode != ProxyModeFull {
			t.Errorf("Expected mode %v, got %v", ProxyModeFull, server.mode)
		}
	})
}

func BenchmarkProxyServer(b *testing.B) {
	eventBus := events.NewEventBus()
	
	// Create simple backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	server := NewServerWithMode(0, ProxyModeReverse, eventBus)
	err := server.Start()
	if err != nil {
		b.Fatalf("Failed to start proxy: %v", err)
	}
	defer server.Stop()

	proxyURL := server.RegisterURL(backend.URL, "bench")
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		client := &http.Client{}
		for pb.Next() {
			resp, err := client.Get(proxyURL)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			resp.Body.Close()
		}
	})
}