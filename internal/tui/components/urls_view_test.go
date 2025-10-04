package components

import (
	"strings"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/messages"
)

func TestURLsViewComponent_NewComponent(t *testing.T) {
	component := NewURLsViewComponent()
	component.SetSize(80, 24)

	if len(component.urls) != 0 {
		t.Errorf("Expected 0 initial URLs, got %d", len(component.urls))
	}

	view := component.View()
	if !strings.Contains(view, "No URLs detected") {
		t.Error("View should show 'No URLs detected' when empty")
	}
}

func TestURLsViewComponent_URLDetectedMsg(t *testing.T) {
	component := NewURLsViewComponent()
	component.SetSize(80, 24)

	// Add a URL via message
	msg := messages.URLDetectedMsg{
		URL:       "http://localhost:3000",
		ProcessID: "dev-123",
		Type:      "http",
		Timestamp: time.Now(),
		HasProxy:  false,
	}

	component.Update(msg)

	// Verify URL was added
	if len(component.urls) != 1 {
		t.Fatalf("Expected 1 URL, got %d", len(component.urls))
	}

	info := component.urls[0]
	if info.URL != "http://localhost:3000" {
		t.Errorf("Expected URL 'http://localhost:3000', got '%s'", info.URL)
	}

	if info.ProcessID != "dev-123" {
		t.Errorf("Expected ProcessID 'dev-123', got '%s'", info.ProcessID)
	}

	// Verify view contains URL
	view := component.View()
	if !strings.Contains(view, "localhost:3000") {
		t.Error("View should contain 'localhost:3000'")
	}
}

func TestURLsViewComponent_DuplicateDetection(t *testing.T) {
	component := NewURLsViewComponent()
	component.SetSize(80, 24)

	// Add the same URL twice
	msg := messages.URLDetectedMsg{
		URL:       "http://localhost:3000",
		ProcessID: "dev-123",
		Type:      "http",
		Timestamp: time.Now(),
		HasProxy:  false,
	}

	component.Update(msg)
	component.Update(msg) // Duplicate

	// Verify only one URL was added
	if len(component.urls) != 1 {
		t.Errorf("Expected 1 URL (duplicate should be ignored), got %d", len(component.urls))
	}
}

func TestURLsViewComponent_MultipleURLs(t *testing.T) {
	component := NewURLsViewComponent()
	component.SetSize(80, 24)

	// Add multiple URLs
	urls := []string{
		"http://localhost:3000",
		"http://localhost:8080",
		"ws://localhost:9000",
	}

	for i, url := range urls {
		msg := messages.URLDetectedMsg{
			URL:       url,
			ProcessID: "proc-" + string(rune('1'+i)),
			Type:      "http",
			Timestamp: time.Now(),
			HasProxy:  false,
		}
		component.Update(msg)
	}

	// Verify all URLs were added
	if len(component.urls) != 3 {
		t.Errorf("Expected 3 URLs, got %d", len(component.urls))
	}

	// Verify view contains all URLs
	view := component.View()
	for _, url := range urls {
		if !strings.Contains(view, "localhost") {
			t.Errorf("View should contain URL '%s'", url)
		}
	}
}

func TestURLsViewComponent_ProxyDisplay(t *testing.T) {
	component := NewURLsViewComponent()
	component.SetSize(80, 24)

	// Add URL with proxy
	msg := messages.URLDetectedMsg{
		URL:        "http://localhost:3000",
		ProcessID:  "dev-123",
		Type:       "http",
		Timestamp:  time.Now(),
		HasProxy:   true,
		ProxyURL:   "http://localhost:7777",
		ProxyLabel: "proxy:3000",
	}

	component.Update(msg)

	// Verify URL has proxy info
	info := component.urls[0]
	if !info.HasProxy {
		t.Error("URL should have proxy enabled")
	}

	if info.ProxyURL != "http://localhost:7777" {
		t.Errorf("Expected ProxyURL 'http://localhost:7777', got '%s'", info.ProxyURL)
	}

	if info.ProxyLabel != "proxy:3000" {
		t.Errorf("Expected ProxyLabel 'proxy:3000', got '%s'", info.ProxyLabel)
	}

	// Verify view contains proxy info
	view := component.View()
	if !strings.Contains(view, "proxy:3000") {
		t.Error("View should contain proxy label 'proxy:3000'")
	}
}

func TestURLsViewComponent_SetSize(t *testing.T) {
	component := NewURLsViewComponent()

	// Set size
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

func TestURLsViewComponent_Visibility(t *testing.T) {
	component := NewURLsViewComponent()

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

func TestURLsViewComponent_URLTypes(t *testing.T) {
	component := NewURLsViewComponent()
	component.SetSize(80, 24)

	// Add URLs of different types
	types := []struct {
		url     string
		urlType string
	}{
		{"http://localhost:3000", "http"},
		{"https://localhost:3001", "https"},
		{"ws://localhost:3002", "ws"},
		{"wss://localhost:3003", "wss"},
	}

	for i, tc := range types {
		msg := messages.URLDetectedMsg{
			URL:       tc.url,
			ProcessID: "proc-" + string(rune('1'+i)),
			Type:      tc.urlType,
			Timestamp: time.Now(),
			HasProxy:  false,
		}
		component.Update(msg)
	}

	// Verify all types were recorded
	if len(component.urls) != 4 {
		t.Errorf("Expected 4 URLs, got %d", len(component.urls))
	}

	// Check each type
	for i, tc := range types {
		if component.urls[i].Type != tc.urlType {
			t.Errorf("Expected type '%s', got '%s'", tc.urlType, component.urls[i].Type)
		}
	}
}
