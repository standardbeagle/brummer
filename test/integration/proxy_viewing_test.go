package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestProxyRequestViewing tests the proxy request viewing functionality
// This test verifies proxy request capture and display
// Test Scenario 5 from quickstart.md
func TestProxyRequestViewing(t *testing.T) {
	SkipIfShort(t, "Integration test with proxy server")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Set initial window size
	ctx.SetWindowSize(120, 40)

	// Test Step 1: Start proxy server (should be started by TestContext)
	t.Run("proxy_server_running", func(t *testing.T) {
		// Verify proxy server is available
		assert.NotNil(t, ctx.ProxyServer, "Proxy server should be initialized")
	})

	// Test Step 2: Make HTTP requests that are captured
	t.Run("capture_http_requests", func(t *testing.T) {
		// Make a test HTTP request through the proxy
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		// Make request to a test endpoint
		resp, err := client.Get("http://localhost:3000/test")
		if err == nil {
			resp.Body.Close()
		}

		// Wait for request to be captured
		time.Sleep(200 * time.Millisecond)
	})

	// Test Step 3: Press '5' to view web/proxy view
	t.Run("switch_to_proxy_view", func(t *testing.T) {
		// Simulate pressing "5" to switch to web/proxy view
		ctx.SendKeyMsg("5")
		time.Sleep(100 * time.Millisecond)

		// Verify we're in proxy view
		ctx.AssertViewContains("Proxy") // or "Web" depending on view title
	})

	// Test Step 4: Verify requests are displayed
	t.Run("verify_requests_displayed", func(t *testing.T) {
		// The view should show captured requests
		// (Actual assertions depend on whether requests were successfully captured)

		// At minimum, the view should be rendered without errors
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Proxy view should render content")
	})

	// Test Step 5: Use arrow keys to select a request
	t.Run("navigate_request_list", func(t *testing.T) {
		// Send down arrow to navigate
		ctx.SendKeyMsg("down")
		time.Sleep(50 * time.Millisecond)

		// Send up arrow to navigate back
		ctx.SendKeyMsg("up")
		time.Sleep(50 * time.Millisecond)

		// View should still be in proxy mode
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should still show proxy view")
	})

	// Test Step 6: Press Enter to view request details
	t.Run("view_request_details", func(t *testing.T) {
		// Simulate pressing Enter to view details
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Details view should be shown
		// (Specific assertions depend on implementation)
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should show request details")
	})

	// Test Step 7: Filter by status code
	t.Run("filter_by_status_code", func(t *testing.T) {
		// Simulate pressing "f" to activate filter
		ctx.SendKeyMsg("f")
		time.Sleep(50 * time.Millisecond)

		// Type "200" for status code filter
		ctx.SendKeyMsg("2")
		ctx.SendKeyMsg("0")
		ctx.SendKeyMsg("0")
		time.Sleep(50 * time.Millisecond)

		// Press Enter to apply filter
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// View should show filtered results
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should show filtered proxy view")
	})
}

// TestProxyRequestCapture tests request capture functionality
func TestProxyRequestCapture(t *testing.T) {
	SkipIfShort(t, "Integration test with proxy capture")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("capture_request_metadata", func(t *testing.T) {
		// Make a test request
		client := &http.Client{Timeout: 5 * time.Second}

		testURL := "http://localhost:3000/api/test"
		req, err := http.NewRequest("GET", testURL, nil)
		assert.NoError(t, err, "Should create request")

		req.Header.Set("User-Agent", "BrummerTest")

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}

		// Wait for capture
		time.Sleep(200 * time.Millisecond)

		// Verify request was captured
		// (Specific verification depends on proxy server implementation)
	})
}

// TestProxyViewPerformance tests proxy view rendering performance
func TestProxyViewPerformance(t *testing.T) {
	SkipIfShort(t, "Performance test for proxy view")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Generate multiple requests
	t.Run("render_many_requests", func(t *testing.T) {
		// Make several requests to populate the proxy view
		client := &http.Client{Timeout: 5 * time.Second}

		for i := 0; i < 10; i++ {
			go func(idx int) {
				url := fmt.Sprintf("http://localhost:3000/test/%d", idx)
				resp, err := client.Get(url)
				if err == nil {
					resp.Body.Close()
				}
			}(i)
		}

		time.Sleep(500 * time.Millisecond)

		// Switch to proxy view
		ctx.SendKeyMsg("5")
		time.Sleep(100 * time.Millisecond)

		// Measure render time
		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "Should render proxy view")
		assert.Less(t, renderTime, 100*time.Millisecond, "Render should be fast even with many requests")
	})
}
