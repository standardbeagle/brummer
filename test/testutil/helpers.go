package testutil

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// BrummerTest provides a test harness for running brummer
type BrummerTest struct {
	t          *testing.T
	binaryPath string
	workDir    string
	Cmd        *exec.Cmd // Exported for tests to check process state
	ctx        context.Context
	cancel     context.CancelFunc
	output     strings.Builder
}

// NewBrummerTest creates a new test harness
func NewBrummerTest(t *testing.T) *BrummerTest {
	t.Helper()
	
	// Find brummer binary
	binaryPath := os.Getenv("BRUMMER_BINARY")
	projectRoot := ""
	if binaryPath == "" {
		// Try to find it in project root
		var err error
		projectRoot, err = filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), "../.."))
		if err != nil {
			t.Fatal(err)
		}
		binaryPath = filepath.Join(projectRoot, "brum")
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			t.Fatalf("brummer binary not found at %s. Set BRUMMER_BINARY env var or build with 'make build'", binaryPath)
		}
	} else {
		// Extract project root from binary path
		projectRoot = filepath.Dir(binaryPath)
	}
	
	// Create temp work directory
	workDir := t.TempDir()
	
	// Find fixtures directory
	// Try multiple paths to handle different execution contexts
	possiblePaths := []string{
		filepath.Join(projectRoot, "test", "fixtures"),
		filepath.Join(filepath.Dir(t.Name()), "fixtures"),
		"fixtures",
		"./fixtures",
		"../fixtures",
	}
	
	var fixturesDir string
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			fixturesDir = path
			break
		}
	}
	
	if fixturesDir == "" {
		t.Fatalf("fixtures directory not found, tried: %v", possiblePaths)
	}
	
	// Copy fixtures
	if err := copyFixtures(fixturesDir, workDir); err != nil {
		t.Fatalf("failed to copy fixtures from %s: %v", fixturesDir, err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &BrummerTest{
		t:          t,
		binaryPath: binaryPath,
		workDir:    workDir,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start runs brummer with the given arguments
func (bt *BrummerTest) Start(args ...string) error {
	bt.t.Helper()
	
	// Build command
	fullArgs := append([]string{"-d", bt.workDir}, args...)
	bt.Cmd = exec.CommandContext(bt.ctx, bt.binaryPath, fullArgs...)
	
	// Capture output
	bt.Cmd.Stdout = &bt.output
	bt.Cmd.Stderr = &bt.output
	
	// Start command
	if err := bt.Cmd.Start(); err != nil {
		return fmt.Errorf("failed to start brummer: %w", err)
	}
	
	// Give it time to start up
	time.Sleep(500 * time.Millisecond)
	
	return nil
}

// Stop gracefully stops brummer
func (bt *BrummerTest) Stop() {
	bt.t.Helper()
	
	if bt.Cmd != nil && bt.Cmd.Process != nil {
		// Try graceful shutdown first
		bt.Cmd.Process.Signal(os.Interrupt)
		
		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- bt.Cmd.Wait()
		}()
		
		select {
		case <-done:
			// Graceful shutdown successful
		case <-time.After(2 * time.Second):
			// Force kill
			bt.cancel()
			bt.Cmd.Process.Kill()
			<-done
		}
	}
}

// Output returns the captured output
func (bt *BrummerTest) Output() string {
	return bt.output.String()
}

// WaitForOutput waits for specific output to appear
func (bt *BrummerTest) WaitForOutput(substr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		if strings.Contains(bt.output.String(), substr) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	return fmt.Errorf("timeout waiting for output containing %q\nGot: %s", substr, bt.output.String())
}

// WaitForMCP waits for MCP server to be ready and returns the port
func (bt *BrummerTest) WaitForMCP(timeout time.Duration) (int, error) {
	err := bt.WaitForOutput("MCP server started on http://localhost:", timeout)
	if err != nil {
		return 0, err
	}
	
	// Extract port from output
	output := bt.output.String()
	start := strings.Index(output, "MCP server started on http://localhost:")
	if start == -1 {
		return 0, fmt.Errorf("MCP server URL not found in output")
	}
	
	start += len("MCP server started on http://localhost:")
	end := strings.IndexAny(output[start:], "/ \n")
	if end == -1 {
		end = len(output) - start
	}
	
	var port int
	_, err = fmt.Sscanf(output[start:start+end], "%d", &port)
	if err != nil {
		return 0, fmt.Errorf("failed to parse MCP port: %w", err)
	}
	
	return port, nil
}

// WaitForProxy waits for proxy server to be ready and returns the port
func (bt *BrummerTest) WaitForProxy(timeout time.Duration) (int, error) {
	err := bt.WaitForOutput("Started HTTP proxy server on port", timeout)
	if err != nil {
		return 0, err
	}
	
	// Extract port from output
	output := bt.output.String()
	start := strings.Index(output, "Started HTTP proxy server on port ")
	if start == -1 {
		return 0, fmt.Errorf("proxy server port not found in output")
	}
	
	start += len("Started HTTP proxy server on port ")
	end := strings.IndexAny(output[start:], " \n")
	if end == -1 {
		end = len(output) - start
	}
	
	var port int
	_, err = fmt.Sscanf(output[start:start+end], "%d", &port)
	if err != nil {
		return 0, fmt.Errorf("failed to parse proxy port: %w", err)
	}
	
	return port, nil
}

// Cleanup should be called with defer to ensure proper cleanup
func (bt *BrummerTest) Cleanup() {
	bt.Stop()
}

// copyFixtures copies test fixtures to the work directory
func copyFixtures(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		
		dstPath := filepath.Join(dst, relPath)
		
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		
		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()
		
		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}

// GetFreePort returns a free port
func GetFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// WaitForHTTP waits for an HTTP server to be ready
func WaitForHTTP(url string, timeout time.Duration) error {
	client := &http.Client{Timeout: 1 * time.Second}
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	return fmt.Errorf("timeout waiting for HTTP server at %s", url)
}

// AssertContains checks if a string contains a substring
func AssertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\nGot: %s", needle, haystack)
	}
}

// AssertNotContains checks if a string does not contain a substring
func AssertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("expected output to NOT contain %q\nGot: %s", needle, haystack)
	}
}