package components

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/messages"
)

func TestWebViewComponent_NewComponent(t *testing.T) {
	component := NewWebViewComponent(nil)
	component.SetSize(80, 24)

	if len(component.requests) != 0 {
		t.Errorf("Expected 0 initial requests, got %d", len(component.requests))
	}

	view := component.View()
	if !strings.Contains(view, "No HTTP requests captured") {
		t.Error("View should show 'No HTTP requests captured' when empty")
	}
}

func TestWebViewComponent_ProxyRequestMsg(t *testing.T) {
	component := NewWebViewComponent(nil)
	component.SetSize(80, 24)

	// Add a request via message
	msg := messages.ProxyRequestMsg{
		RequestID:   "req-1",
		Method:      "GET",
		URL:         "http://localhost:3000/api/users",
		Path:        "/api/users",
		StatusCode:  200,
		Duration:    50 * time.Millisecond,
		ProcessName: "dev",
		Timestamp:   time.Now(),
	}

	component.Update(msg)

	// Verify request was added
	if len(component.requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(component.requests))
	}

	info := component.requests[0]
	if info.Method != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", info.Method)
	}
	if info.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", info.StatusCode)
	}

	// Verify view contains request
	view := component.View()
	if !strings.Contains(view, "GET") {
		t.Error("View should contain method 'GET'")
	}
	if !strings.Contains(view, "/api/users") {
		t.Error("View should contain URL path")
	}
}

func TestWebViewComponent_MultipleRequests(t *testing.T) {
	component := NewWebViewComponent(nil)
	component.SetSize(80, 24)

	// Add multiple requests
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for i, method := range methods {
		msg := messages.ProxyRequestMsg{
			RequestID:   fmt.Sprintf("req-%d", i),
			Method:      method,
			URL:         fmt.Sprintf("http://localhost:3000/api/%d", i),
			Path:        fmt.Sprintf("/api/%d", i),
			StatusCode:  200,
			Duration:    10 * time.Millisecond,
			ProcessName: "dev",
			Timestamp:   time.Now(),
		}
		component.Update(msg)
	}

	// Verify all requests were added
	if len(component.requests) != 4 {
		t.Errorf("Expected 4 requests, got %d", len(component.requests))
	}

	// Verify view contains all methods
	view := component.View()
	for _, method := range methods {
		if !strings.Contains(view, method) {
			t.Errorf("View should contain method '%s'", method)
		}
	}
}

func TestWebViewComponent_StatusCodeColors(t *testing.T) {
	component := NewWebViewComponent(nil)
	component.SetSize(80, 24)

	// Test different status code ranges
	testCases := []struct {
		statusCode int
		desc       string
	}{
		{200, "success"},
		{301, "redirect"},
		{404, "client error"},
		{500, "server error"},
	}

	for _, tc := range testCases {
		info := &RequestInfo{
			Method:     "GET",
			URL:        "http://localhost:3000",
			StatusCode: tc.statusCode,
			Duration:   10 * time.Millisecond,
		}

		// Should not crash with different status codes
		_ = component.formatRequestRow(info)
	}
}

func TestWebViewComponent_DurationFormatting(t *testing.T) {
	component := NewWebViewComponent(nil)
	component.SetSize(80, 24)

	testCases := []struct {
		duration time.Duration
		expected string // Just verify it formats
	}{
		{500 * time.Microsecond, "μs"},
		{50 * time.Millisecond, "ms"},
		{2 * time.Second, "s"},
	}

	for _, tc := range testCases {
		info := &RequestInfo{
			Method:     "GET",
			URL:        "http://localhost:3000",
			StatusCode: 200,
			Duration:   tc.duration,
		}

		formatted := component.formatRequestRow(info)
		// Just verify it doesn't crash and contains some time unit
		if formatted == "" {
			t.Error("Formatted row should not be empty")
		}
	}
}

func TestWebViewComponent_SetSize(t *testing.T) {
	component := NewWebViewComponent(nil)

	component.SetSize(100, 50)

	if component.Width() != 100 {
		t.Errorf("Expected width 100, got %d", component.Width())
	}
	if component.Height() != 50 {
		t.Errorf("Expected height 50, got %d", component.Height())
	}
	if component.viewport.Width != 100 {
		t.Errorf("Expected viewport width 100, got %d", component.viewport.Width)
	}
	if component.viewport.Height != 50 {
		t.Errorf("Expected viewport height 50, got %d", component.viewport.Height)
	}
}

func TestWebViewComponent_Visibility(t *testing.T) {
	component := NewWebViewComponent(nil)

	// Default is visible
	if !component.IsVisible() {
		t.Error("Component should be visible by default")
	}

	// Test hiding
	component.SetVisible(false)
	if component.IsVisible() {
		t.Error("Component should be hidden")
	}

	// View should be empty when hidden
	view := component.View()
	if view != "" {
		t.Error("Hidden component should return empty view")
	}
}

func TestWebViewComponent_URLTruncation(t *testing.T) {
	component := NewWebViewComponent(nil)
	component.SetSize(80, 24)

	// Test with very long URL
	longURL := "http://localhost:3000/api/v1/users/123/posts/456/comments/789/replies/abc/likes"
	info := &RequestInfo{
		Method:     "GET",
		URL:        longURL,
		StatusCode: 200,
		Duration:   10 * time.Millisecond,
	}

	formatted := component.formatRequestRow(info)

	// Should be truncated with "..."
	if !strings.Contains(formatted, "...") {
		t.Error("Long URL should be truncated with ellipsis")
	}
}
