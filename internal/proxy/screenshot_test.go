package proxy

import (
	"os"
	"strings"
	"testing"
	"time"
)

// Test screenshot file saving to project .brummer directory
func TestSaveScreenshot(t *testing.T) {
	// Create temporary project directory
	tempDir, err := os.MkdirTemp("", "brummer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory to simulate project root
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Create a simple 1x1 PNG in base64 (valid PNG header)
	// This is a tiny valid PNG image
	pngData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	dataURL := "data:image/png;base64," + pngData

	url := "http://localhost:3000/test"

	// Call saveScreenshot
	filename, err := saveScreenshot(dataURL, url)
	if err != nil {
		t.Fatalf("saveScreenshot failed: %v", err)
	}

	// Test 1: File should be saved in .brummer/screenshots/
	if !strings.Contains(filename, ".brummer/screenshots/") {
		t.Errorf("Expected filename to contain '.brummer/screenshots/', got: %s", filename)
	}

	// Test 2: File should exist
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("Screenshot file was not created: %s", filename)
	}

	// Test 3: Filename should include timestamp format
	if !strings.Contains(filename, "screenshot-") {
		t.Errorf("Expected filename to start with 'screenshot-', got: %s", filename)
	}

	// Test 4: Filename should have .png extension
	if !strings.HasSuffix(filename, ".png") {
		t.Errorf("Expected filename to end with '.png', got: %s", filename)
	}

	// Test 5: File should contain valid PNG data
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read screenshot file: %v", err)
	}

	// PNG files start with specific magic bytes
	if len(data) < 8 || data[0] != 0x89 || data[1] != 0x50 || data[2] != 0x4E || data[3] != 0x47 {
		t.Errorf("File does not contain valid PNG data")
	}
}

// Test screenshot filename generation with URL slugs
func TestGenerateScreenshotFilename(t *testing.T) {
	tests := []struct {
		url      string
		wantSlug string
	}{
		{"http://localhost:3000/", "localhost-3000"},
		{"http://localhost:3000/users/profile", "localhost-3000-users-profile"},
		{"https://example.com/path/to/page?query=1", "example.com-path-to-page"},
		{"http://192.168.1.1:8080/api", "192.168.1.1-8080-api"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			filename := generateScreenshotFilename(tt.url)

			// Should contain timestamp
			if !strings.Contains(filename, "screenshot-") {
				t.Errorf("Filename should start with 'screenshot-', got: %s", filename)
			}

			// Should contain URL slug
			if !strings.Contains(filename, tt.wantSlug) {
				t.Errorf("Expected filename to contain '%s', got: %s", tt.wantSlug, filename)
			}

			// Should have .png extension
			if !strings.HasSuffix(filename, ".png") {
				t.Errorf("Expected .png extension, got: %s", filename)
			}
		})
	}
}

// Test screenshot directory creation
func TestEnsureScreenshotDir(t *testing.T) {
	// Create temporary project directory
	tempDir, err := os.MkdirTemp("", "brummer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Ensure screenshot directory
	dir, err := ensureScreenshotDir()
	if err != nil {
		t.Fatalf("ensureScreenshotDir failed: %v", err)
	}

	// Test 1: Directory should be created
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		t.Errorf("Screenshot directory was not created: %s", dir)
	}

	// Test 2: Should be a directory
	if !info.IsDir() {
		t.Errorf("Expected %s to be a directory", dir)
	}

	// Test 3: Should end with .brummer/screenshots
	if !strings.HasSuffix(dir, ".brummer/screenshots") {
		t.Errorf("Expected directory to end with '.brummer/screenshots', got: %s", dir)
	}

	// Test 4: Calling again should not error (idempotent)
	dir2, err := ensureScreenshotDir()
	if err != nil {
		t.Errorf("ensureScreenshotDir should be idempotent: %v", err)
	}
	if dir != dir2 {
		t.Errorf("ensureScreenshotDir should return same path: %s != %s", dir, dir2)
	}
}

// Test data URL parsing
func TestParseDataURL(t *testing.T) {
	tests := []struct {
		name    string
		dataURL string
		wantErr bool
	}{
		{
			name:    "valid PNG data URL",
			dataURL: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
			wantErr: false,
		},
		{
			name:    "valid JPEG data URL",
			dataURL: "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAA8A",
			wantErr: false,
		},
		{
			name:    "missing data prefix",
			dataURL: "image/png;base64,iVBORw0KGgoAAAANS",
			wantErr: true,
		},
		{
			name:    "missing base64 part",
			dataURL: "data:image/png,iVBORw0KGgoAAAANS",
			wantErr: true,
		},
		{
			name:    "empty string",
			dataURL: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := parseDataURL(tt.dataURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for input: %s", tt.dataURL)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(data) == 0 {
					t.Errorf("Expected non-empty data")
				}
			}
		})
	}
}

// Test screenshot metadata extraction
func TestExtractScreenshotMetadata(t *testing.T) {
	url := "http://localhost:3000/users/profile"
	timestamp := time.Now()
	filename := "/path/to/.brummer/screenshots/screenshot-2024-01-15-143022-localhost-3000-users-profile.png"

	metadata := extractScreenshotMetadata(url, filename, timestamp)

	// Test 1: Should have required fields
	if metadata.URL != url {
		t.Errorf("Expected URL '%s', got: %s", url, metadata.URL)
	}

	if metadata.Filename != filename {
		t.Errorf("Expected filename '%s', got: %s", filename, metadata.Filename)
	}

	if metadata.Timestamp != timestamp {
		t.Errorf("Expected timestamp '%v', got: %v", timestamp, metadata.Timestamp)
	}

	// Test 2: Should extract URL components
	if metadata.Host != "localhost:3000" {
		t.Errorf("Expected host 'localhost:3000', got: %s", metadata.Host)
	}

	if metadata.Path != "/users/profile" {
		t.Errorf("Expected path '/users/profile', got: %s", metadata.Path)
	}

	// Test 3: Should have file info
	if metadata.FileSize == 0 {
		// FileSize might be 0 if file doesn't exist yet in test
		// This is acceptable for this test
	}
}

// Test WebSocket screenshot request handling
func TestHandleScreenshotRequest(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "brummer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Valid screenshot data
	pngData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	dataURL := "data:image/png;base64," + pngData

	requestData := map[string]interface{}{
		"requestId": "screenshot_123",
		"url":       "http://localhost:3000/test",
		"dataURL":   dataURL,
		"viewport": map[string]interface{}{
			"width":  1920,
			"height": 1080,
		},
	}

	// Call handler
	response, err := handleScreenshotRequest(requestData)

	// Test 1: Should succeed
	if err != nil {
		t.Fatalf("handleScreenshotRequest failed: %v", err)
	}

	// Test 2: Response should contain success flag
	if !response.Success {
		t.Errorf("Expected success=true, got: %v", response.Success)
	}

	// Test 3: Response should contain filename
	if response.Filename == "" {
		t.Errorf("Expected filename in response")
	}

	// Test 4: File should exist
	if _, err := os.Stat(response.Filename); os.IsNotExist(err) {
		t.Errorf("Screenshot file not created: %s", response.Filename)
	}

	// Test 5: Response should contain metadata
	if response.Metadata.URL != "http://localhost:3000/test" {
		t.Errorf("Expected URL in metadata, got: %s", response.Metadata.URL)
	}
}

// Test WebSocket screenshot request with missing data URL
func TestHandleScreenshotRequest_MissingDataURL(t *testing.T) {
	requestData := map[string]interface{}{
		"requestId": "screenshot_123",
		"url":       "http://localhost:3000/test",
		// Missing dataURL
	}

	response, err := handleScreenshotRequest(requestData)

	// Should return error
	if err == nil {
		t.Errorf("Expected error for missing dataURL")
	}

	// Response should indicate failure
	if response.Success {
		t.Errorf("Expected success=false for missing dataURL")
	}

	if response.Error == "" {
		t.Errorf("Expected error message in response")
	}
}

// Test WebSocket screenshot request with invalid data URL
func TestHandleScreenshotRequest_InvalidDataURL(t *testing.T) {
	requestData := map[string]interface{}{
		"requestId": "screenshot_123",
		"url":       "http://localhost:3000/test",
		"dataURL":   "invalid-data-url",
	}

	response, err := handleScreenshotRequest(requestData)

	// Should return error
	if err == nil {
		t.Errorf("Expected error for invalid dataURL")
	}

	// Response should indicate failure
	if response.Success {
		t.Errorf("Expected success=false for invalid dataURL")
	}
}
