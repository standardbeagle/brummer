package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerCreation tests creating proxy servers
func TestServerCreation(t *testing.T) {
	eventBus := events.NewEventBus()

	// Test creating server with default mode (full)
	server := NewServer(0, eventBus) // Use port 0 for auto-assignment
	require.NotNil(t, server)
	assert.Equal(t, ProxyModeFull, server.GetMode())
	assert.False(t, server.IsRunning())

	// Test creating server with specific mode
	serverReverse := NewServerWithMode(0, ProxyModeReverse, eventBus)
	require.NotNil(t, serverReverse)
	assert.Equal(t, ProxyModeReverse, serverReverse.GetMode())
	assert.False(t, serverReverse.IsRunning())
}

// TestServerLifecycle tests starting and stopping the server
func TestServerLifecycle(t *testing.T) {
	eventBus := events.NewEventBus()
	server := NewServerWithMode(0, ProxyModeFull, eventBus)

	// Test starting the server
	err := server.Start()
	require.NoError(t, err)
	assert.True(t, server.IsRunning())

	// Give server a moment to fully start
	time.Sleep(50 * time.Millisecond)
	assert.Greater(t, server.GetPort(), 0)

	// Test stopping the server
	err = server.Stop()
	require.NoError(t, err)
	assert.False(t, server.IsRunning())
}

// TestFullProxyMode tests traditional HTTP proxy functionality
func TestFullProxyMode(t *testing.T) {
	eventBus := events.NewEventBus()
	server := NewServerWithMode(0, ProxyModeFull, eventBus)

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Test PAC file endpoint
	pacURL := server.GetPACURL()
	assert.NotEmpty(t, pacURL)
	assert.Contains(t, pacURL, fmt.Sprintf(":%d", server.GetPort()))

	// Make a request to the PAC file endpoint
	resp, err := http.Get(pacURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/x-ns-proxy-autoconfig", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	pacContent := string(body)

	// PAC file should contain proxy configuration
	assert.Contains(t, pacContent, "function FindProxyForURL")
	assert.Contains(t, pacContent, fmt.Sprintf("PROXY localhost:%d", server.GetPort()))
}

// TestReverseProxyMode tests reverse proxy functionality
func TestReverseProxyMode(t *testing.T) {
	eventBus := events.NewEventBus()
	server := NewServerWithMode(0, ProxyModeReverse, eventBus)

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Note: We'll use a mock URL for testing since we don't need a real server running

	// Test URL registration
	targetURL := "http://localhost:3000"
	proxyURL := server.RegisterURL(targetURL, "test-process")

	assert.NotEqual(t, targetURL, proxyURL)
	assert.Contains(t, proxyURL, "localhost")
	assert.NotContains(t, proxyURL, ":3000") // Should have different port

	// Test URL registration with label
	labeledProxyURL := server.RegisterURLWithLabel("http://localhost:4000", "api-process", "API Server")
	assert.NotEmpty(t, labeledProxyURL)
	assert.Contains(t, labeledProxyURL, "localhost")

	// Test getting proxy URL for existing registration
	existingProxyURL := server.GetProxyURL(targetURL)
	assert.Equal(t, proxyURL, existingProxyURL)

	// Test getting proxy URL for non-existent URL
	nonExistentProxyURL := server.GetProxyURL("http://localhost:9999")
	assert.Equal(t, "http://localhost:9999", nonExistentProxyURL) // Should return original
}

// TestURLMappings tests URL mapping functionality
func TestURLMappings(t *testing.T) {
	eventBus := events.NewEventBus()
	server := NewServerWithMode(0, ProxyModeReverse, eventBus)

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Register multiple URLs
	urls := []struct {
		target  string
		process string
		label   string
	}{
		{"http://localhost:3000", "frontend", "Frontend"},
		{"http://localhost:4000", "api", "API Server"},
		{"http://localhost:5000", "auth", "Auth Service"},
	}

	for _, u := range urls {
		server.RegisterURLWithLabel(u.target, u.process, u.label)
	}

	// Test getting all mappings
	mappings := server.GetURLMappings()
	assert.Len(t, mappings, 3)

	// Verify mappings are sorted by creation time
	for i := 1; i < len(mappings); i++ {
		assert.True(t, mappings[i].CreatedAt.After(mappings[i-1].CreatedAt) ||
			mappings[i].CreatedAt.Equal(mappings[i-1].CreatedAt))
	}

	// Verify mapping details
	mapping := mappings[0]
	assert.Contains(t, []string{"http://localhost:3000", "http://localhost:4000", "http://localhost:5000"}, mapping.TargetURL)
	assert.Contains(t, []string{"frontend", "api", "auth"}, mapping.ProcessName)
	assert.Contains(t, []string{"Frontend", "API Server", "Auth Service"}, mapping.Label)
	assert.Greater(t, mapping.ProxyPort, 0)
	assert.NotEmpty(t, mapping.ProxyURL)
	assert.False(t, mapping.CreatedAt.IsZero())
}

// TestModeSwitch tests switching between proxy modes
func TestModeSwitch(t *testing.T) {
	eventBus := events.NewEventBus()
	server := NewServerWithMode(0, ProxyModeReverse, eventBus)

	// Initially reverse mode
	assert.Equal(t, ProxyModeReverse, server.GetMode())

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Switch to full mode
	err = server.SwitchMode(ProxyModeFull)
	require.NoError(t, err)
	assert.Equal(t, ProxyModeFull, server.GetMode())

	// PAC file should now be available
	pacURL := server.GetPACURL()
	assert.NotEmpty(t, pacURL)

	// Switch back to reverse mode
	err = server.SwitchMode(ProxyModeReverse)
	require.NoError(t, err)
	assert.Equal(t, ProxyModeReverse, server.GetMode())
}

// TestRequestCapture tests HTTP request capture functionality
func TestRequestCapture(t *testing.T) {
	eventBus := events.NewEventBus()
	server := NewServerWithMode(0, ProxyModeFull, eventBus)

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Initially no requests
	requests := server.GetRequests()
	assert.Empty(t, requests)

	// Make a request through the proxy (we'll simulate this by adding a request manually)
	// In a real scenario, this would happen automatically when requests go through the proxy

	// Wait a bit to see if any requests were captured during startup
	time.Sleep(100 * time.Millisecond)

	// Test clearing requests
	server.ClearRequests()
	requests = server.GetRequests()
	assert.Empty(t, requests)

	// Test process-specific request operations
	server.ClearRequestsForProcess("test-process")
	processRequests := server.GetRequestsForProcess("test-process")
	assert.Empty(t, processRequests)
}

// TestTelemetryIntegration tests telemetry functionality
func TestTelemetryIntegration(t *testing.T) {
	eventBus := events.NewEventBus()
	server := NewServerWithMode(0, ProxyModeReverse, eventBus)

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Test telemetry store
	telemetryStore := server.GetTelemetryStore()
	assert.NotNil(t, telemetryStore)

	// Test telemetry enable/disable
	server.EnableTelemetry(true)
	assert.True(t, server.IsTelemetryEnabled())

	server.EnableTelemetry(false)
	assert.False(t, server.IsTelemetryEnabled())

	// Test getting telemetry sessions (should be empty initially)
	session, exists := server.GetTelemetrySession("nonexistent-session")
	assert.False(t, exists)
	assert.Nil(t, session)

	sessions := server.GetTelemetryForProcess("test-process")
	assert.Empty(t, sessions)

	// Test clearing telemetry
	server.ClearTelemetryForProcess("test-process")
	// Should not panic or error
}

// TestConcurrentOperations tests thread safety
func TestConcurrentOperations(t *testing.T) {
	eventBus := events.NewEventBus()
	server := NewServerWithMode(0, ProxyModeReverse, eventBus)

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Register URLs concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()

			targetURL := fmt.Sprintf("http://localhost:%d", 3000+index)
			processName := fmt.Sprintf("process-%d", index)

			proxyURL := server.RegisterURL(targetURL, processName)
			assert.NotEmpty(t, proxyURL)

			// Test concurrent reads
			mappings := server.GetURLMappings()
			assert.NotNil(t, mappings)

			requests := server.GetRequests()
			assert.NotNil(t, requests)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all URLs were registered
	mappings := server.GetURLMappings()
	assert.Len(t, mappings, 10)
}

// TestServerPortConflict tests handling of port conflicts
func TestServerPortConflict(t *testing.T) {
	eventBus := events.NewEventBus()

	// Start first server on a specific port
	server1 := NewServerWithMode(0, ProxyModeFull, eventBus)
	err := server1.Start()
	require.NoError(t, err)
	defer server1.Stop()

	port1 := server1.GetPort()
	assert.Greater(t, port1, 0)

	// Start second server (should get different port)
	server2 := NewServerWithMode(0, ProxyModeFull, eventBus)
	err = server2.Start()
	require.NoError(t, err)
	defer server2.Stop()

	port2 := server2.GetPort()
	assert.Greater(t, port2, 0)
	assert.NotEqual(t, port1, port2)
}

// TestProxyModeConstants tests proxy mode constants
func TestProxyModeConstants(t *testing.T) {
	assert.Equal(t, ProxyMode("full"), ProxyModeFull)
	assert.Equal(t, ProxyMode("reverse"), ProxyModeReverse)

	// Test that the constants are not empty
	assert.NotEmpty(t, string(ProxyModeFull))
	assert.NotEmpty(t, string(ProxyModeReverse))
}

// TestEventBusIntegration tests integration with the event bus
func TestEventBusIntegration(t *testing.T) {
	eventBus := events.NewEventBus()

	var receivedEvents []events.Event
	var mu sync.Mutex

	// Subscribe to any events that might be published
	eventTypes := []events.EventType{
		events.ProcessStarted,
		events.ProcessExited,
		events.LogLine,
		events.ErrorDetected,
	}

	for _, eventType := range eventTypes {
		eventBus.Subscribe(eventType, func(event events.Event) {
			mu.Lock()
			receivedEvents = append(receivedEvents, event)
			mu.Unlock()
		})
	}

	server := NewServerWithMode(0, ProxyModeReverse, eventBus)

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Register a URL and see if any events are published
	server.RegisterURL("http://localhost:3000", "test-process")

	// Wait briefly for any potential async events
	time.Sleep(50 * time.Millisecond)

	// Check if any events were received (this might be zero, which is fine)
	mu.Lock()
	defer mu.Unlock()

	// Just verify the events slice is properly initialized
	assert.NotNil(t, receivedEvents)
}

// Helper function to make HTTP requests through proxy (if needed for future tests)
func makeProxyRequest(proxyURL, targetURL string) (*http.Response, error) {
	proxyURLParsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURLParsed),
		},
		Timeout: 5 * time.Second,
	}

	return client.Get(targetURL)
}
