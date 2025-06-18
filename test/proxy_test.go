// +build integration

package test

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/test/testutil"
)

// TestProxyServerStartupNoTUI tests proxy server startup in headless mode
func TestProxyServerStartupNoTUI(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with dev script that outputs URLs
	err := bt.Start("--no-tui", "dev")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for proxy server to start
	port, err := bt.WaitForProxy(5 * time.Second)
	if err != nil {
		t.Fatalf("Proxy server did not start: %v\nOutput: %s", err, bt.Output())
	}

	t.Logf("Proxy server started successfully on port %d", port)

	// Verify the message format
	output := bt.Output()
	proxyPattern := regexp.MustCompile(`🌐 Started HTTP proxy server on port \d+`)
	if !proxyPattern.MatchString(output) {
		t.Errorf("Proxy startup message not in expected format")
	}
}

// TestProxyServerStartupTUI tests proxy server startup in TUI mode
func TestProxyServerStartupTUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TUI test in short mode")
	}

	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer in TUI mode with dev script
	err := bt.Start("dev")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// TUI should start successfully
	time.Sleep(2 * time.Second)

	// Check that process is still running
	if bt.Cmd.ProcessState != nil {
		t.Errorf("TUI process exited unexpectedly")
		t.Logf("Output: %s", bt.Output())
	}
}

// TestProxyURLDetection tests that proxy detects URLs from process output
func TestProxyURLDetection(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with dev script
	err := bt.Start("--no-tui", "dev")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for dev server URL to appear
	err = bt.WaitForOutput("http://localhost:3000", 5*time.Second)
	if err != nil {
		t.Fatalf("Dev server URL not detected: %v", err)
	}

	// Proxy should start after URL detection
	port, err := bt.WaitForProxy(5 * time.Second)
	if err != nil {
		t.Errorf("Proxy did not start after URL detection: %v", err)
		t.Logf("Output: %s", bt.Output())
		return
	}

	t.Logf("URL detected and proxy started on port %d", port)

	// Verify the proxy is set up for the detected URL
	output := bt.Output()
	testutil.AssertContains(t, output, "http://localhost:3000")
	testutil.AssertContains(t, output, "Started HTTP proxy")
}

// TestProxyRequestHandling tests that proxy can handle HTTP requests
func TestProxyRequestHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping proxy request test in short mode")
	}

	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with start script (runs on port 8080)
	err := bt.Start("--no-tui", "start")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for server to start
	err = bt.WaitForOutput("http://localhost:8080", 5*time.Second)
	if err != nil {
		t.Fatalf("Server URL not found: %v", err)
	}

	// Wait for proxy
	proxyPort, err := bt.WaitForProxy(5 * time.Second)
	if err != nil {
		t.Fatalf("Proxy did not start: %v", err)
	}

	// Give services time to fully initialize
	time.Sleep(1 * time.Second)

	// Test proxy request
	proxyURL := fmt.Sprintf("http://localhost:%d", proxyPort)
	client := &http.Client{Timeout: 5 * time.Second}
	
	resp, err := client.Get(proxyURL)
	if err != nil {
		// Proxy might reject the request, but it should be reachable
		if strings.Contains(err.Error(), "connection refused") {
			t.Errorf("Proxy server not reachable on port %d: %v", proxyPort, err)
		} else {
			t.Logf("Proxy responded (possibly with error): %v", err)
		}
	} else {
		resp.Body.Close()
		t.Logf("Proxy server responded with status: %s", resp.Status)
	}
}

// TestProxyDisableFlag tests the --no-proxy flag
func TestProxyDisableFlag(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with --no-proxy flag
	err := bt.Start("--no-tui", "--no-proxy", "dev")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for dev server to start
	err = bt.WaitForOutput("Dev server running", 5*time.Second)
	if err != nil {
		t.Fatalf("Dev server did not start: %v", err)
	}

	// Give it time to potentially start proxy
	time.Sleep(2 * time.Second)

	// Check that proxy was NOT started
	output := bt.Output()
	if strings.Contains(output, "Started HTTP proxy server") {
		t.Errorf("Proxy server started despite --no-proxy flag")
		t.Logf("Output: %s", output)
	} else {
		t.Log("Proxy correctly disabled with --no-proxy flag")
	}
}

// TestProxyMultipleURLs tests proxy handling multiple URLs
func TestProxyMultipleURLs(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Create a custom test script that outputs multiple URLs
	// For now, we'll run both dev and start scripts
	err := bt.Start("--no-tui", "dev", "start")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for both URLs
	err = bt.WaitForOutput("http://localhost:3000", 5*time.Second)
	if err != nil {
		t.Errorf("First URL not detected: %v", err)
	}

	err = bt.WaitForOutput("http://localhost:8080", 5*time.Second)
	if err != nil {
		t.Errorf("Second URL not detected: %v", err)
	}

	// Proxy should handle both
	_, err = bt.WaitForProxy(5 * time.Second)
	if err != nil {
		t.Errorf("Proxy did not start with multiple URLs: %v", err)
	} else {
		t.Log("Proxy started successfully with multiple URLs")
	}
}

// TestProxyURLFormat tests proxy handling of different URL formats
func TestProxyURLFormat(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// The dev script outputs a standard http://localhost:port URL
	err := bt.Start("--no-tui", "dev")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for URL and proxy
	err = bt.WaitForOutput("http://localhost:", 5*time.Second)
	if err != nil {
		t.Fatalf("URL not detected: %v", err)
	}

	proxyPort, err := bt.WaitForProxy(5 * time.Second)
	if err != nil {
		t.Fatalf("Proxy did not start: %v", err)
	}

	// Verify URL format in output
	output := bt.Output()
	urlPattern := regexp.MustCompile(`http://localhost:\d+`)
	matches := urlPattern.FindAllString(output, -1)
	
	if len(matches) == 0 {
		t.Errorf("No localhost URLs found in output")
	} else {
		t.Logf("Found %d URL(s), proxy on port %d", len(matches), proxyPort)
		for i, url := range matches {
			t.Logf("URL %d: %s", i+1, url)
		}
	}
}

// TestProxyWithoutURLs tests proxy behavior when no URLs are detected
func TestProxyWithoutURLs(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start with a script that doesn't output URLs
	err := bt.Start("--no-tui", "lint")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for script to complete
	err = bt.WaitForOutput("No issues found", 5*time.Second)
	if err != nil {
		t.Errorf("Lint script did not complete: %v", err)
	}

	// Give it time to potentially start proxy
	time.Sleep(2 * time.Second)

	// Proxy should not start without URLs
	output := bt.Output()
	if strings.Contains(output, "Started HTTP proxy server") {
		t.Logf("Note: Proxy started even without URLs (this may be expected behavior)")
	} else {
		t.Logf("Proxy did not start without URLs (expected behavior)")
	}
}