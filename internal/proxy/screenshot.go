package proxy

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ScreenshotMetadata contains metadata about a captured screenshot
type ScreenshotMetadata struct {
	URL       string    `json:"url"`
	Filename  string    `json:"filename"`
	Timestamp time.Time `json:"timestamp"`
	Host      string    `json:"host"`
	Path      string    `json:"path"`
	FileSize  int64     `json:"fileSize"`
}

// saveScreenshot saves a screenshot data URL to a file in the project .brummer directory
func saveScreenshot(dataURL, pageURL string) (string, error) {
	// Parse the data URL to extract image data
	imageData, err := parseDataURL(dataURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse data URL: %w", err)
	}

	// Ensure screenshot directory exists
	screenshotDir, err := ensureScreenshotDir()
	if err != nil {
		return "", fmt.Errorf("failed to create screenshot directory: %w", err)
	}

	// Generate filename with timestamp and URL slug
	filename := generateScreenshotFilename(pageURL)
	fullPath := filepath.Join(screenshotDir, filename)

	// Write image data to file
	if err := os.WriteFile(fullPath, imageData, 0644); err != nil {
		return "", fmt.Errorf("failed to write screenshot file: %w", err)
	}

	return fullPath, nil
}

// parseDataURL extracts the base64 image data from a data URL
func parseDataURL(dataURL string) ([]byte, error) {
	// Expected format: data:image/png;base64,<base64-data>
	if !strings.HasPrefix(dataURL, "data:") {
		return nil, fmt.Errorf("invalid data URL: missing 'data:' prefix")
	}

	// Find the base64 data part
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid data URL format: missing comma separator")
	}

	// Check if it's base64 encoded
	if !strings.Contains(parts[0], "base64") {
		return nil, fmt.Errorf("data URL must be base64 encoded")
	}

	// Decode base64 data
	imageData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 data: %w", err)
	}

	if len(imageData) == 0 {
		return nil, fmt.Errorf("decoded image data is empty")
	}

	return imageData, nil
}

// generateScreenshotFilename generates a filename with timestamp and URL slug
func generateScreenshotFilename(pageURL string) string {
	// Generate timestamp in format: YYYY-MM-DD-HHmmss
	timestamp := time.Now().Format("2006-01-02-150405")

	// Create URL slug from the page URL
	urlSlug := createURLSlug(pageURL)

	// Format: screenshot-YYYY-MM-DD-HHmmss-url-slug.png
	return fmt.Sprintf("screenshot-%s-%s.png", timestamp, urlSlug)
}

// createURLSlug creates a filesystem-safe slug from a URL
func createURLSlug(pageURL string) string {
	// Parse URL
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		// If parsing fails, use a safe default
		return "page"
	}

	// Build slug from host and path
	slug := parsedURL.Host

	// Add path if present (remove leading slash and query params)
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		path := strings.Trim(parsedURL.Path, "/")
		slug = slug + "-" + path
	}

	// Replace unsafe characters with hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-.]`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from ends
	slug = strings.Trim(slug, "-")

	// Limit length to 50 characters
	if len(slug) > 50 {
		slug = slug[:50]
	}

	// If slug is empty after sanitization, use default
	if slug == "" {
		slug = "page"
	}

	return slug
}

// ensureScreenshotDir ensures the .brummer/screenshots directory exists
func ensureScreenshotDir() (string, error) {
	// Get current working directory (project root)
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create .brummer/screenshots directory
	screenshotDir := filepath.Join(cwd, ".brummer", "screenshots")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	return screenshotDir, nil
}

// extractScreenshotMetadata extracts metadata from a screenshot
func extractScreenshotMetadata(pageURL, filename string, timestamp time.Time) ScreenshotMetadata {
	metadata := ScreenshotMetadata{
		URL:       pageURL,
		Filename:  filename,
		Timestamp: timestamp,
	}

	// Parse URL to extract host and path
	if parsedURL, err := url.Parse(pageURL); err == nil {
		metadata.Host = parsedURL.Host
		metadata.Path = parsedURL.Path
	}

	// Get file size if file exists
	if info, err := os.Stat(filename); err == nil {
		metadata.FileSize = info.Size()
	}

	return metadata
}

// ScreenshotResponse represents the response to a screenshot request
type ScreenshotResponse struct {
	Success   bool               `json:"success"`
	Filename  string             `json:"filename,omitempty"`
	Metadata  ScreenshotMetadata `json:"metadata,omitempty"`
	Error     string             `json:"error,omitempty"`
	RequestID string             `json:"requestId"`
}

// handleScreenshotRequest processes a screenshot request from the browser
func handleScreenshotRequest(requestData map[string]interface{}) (ScreenshotResponse, error) {
	// Extract request ID
	requestID, _ := requestData["requestId"].(string)

	response := ScreenshotResponse{
		RequestID: requestID,
		Success:   false,
	}

	// Extract data URL
	dataURL, ok := requestData["dataURL"].(string)
	if !ok || dataURL == "" {
		err := fmt.Errorf("missing or invalid dataURL in request")
		response.Error = err.Error()
		return response, err
	}

	// Extract page URL
	pageURL, ok := requestData["url"].(string)
	if !ok || pageURL == "" {
		pageURL = "unknown"
	}

	// Save screenshot to file
	filename, err := saveScreenshot(dataURL, pageURL)
	if err != nil {
		response.Error = fmt.Sprintf("failed to save screenshot: %v", err)
		return response, err
	}

	// Extract metadata
	metadata := extractScreenshotMetadata(pageURL, filename, time.Now())

	// Build success response
	response.Success = true
	response.Filename = filename
	response.Metadata = metadata

	return response, nil
}

// formatAICoderMessage formats a user message for the AI Coder with context
func formatAICoderMessage(message, pageURL, screenshotPath string, data map[string]interface{}) string {
	var formatted strings.Builder

	// Start with the user's message
	formatted.WriteString(message)
	formatted.WriteString("\n\n")

	// Add context section
	formatted.WriteString("## Context from Browser\n\n")
	formatted.WriteString(fmt.Sprintf("**Page URL:** %s\n", pageURL))

	// Extract page title if available
	if context, ok := data["context"].(map[string]interface{}); ok {
		if title, ok := context["pageTitle"].(string); ok && title != "" {
			formatted.WriteString(fmt.Sprintf("**Page Title:** %s\n", title))
		}

		// Add viewport info
		if viewport, ok := context["viewport"].(map[string]interface{}); ok {
			width, _ := viewport["width"].(float64)
			height, _ := viewport["height"].(float64)
			if width > 0 && height > 0 {
				formatted.WriteString(fmt.Sprintf("**Viewport:** %dx%d\n", int(width), int(height)))
			}
		}
	}

	// Add screenshot reference if provided
	if screenshotPath != "" {
		formatted.WriteString(fmt.Sprintf("\n**Screenshot saved:** `%s`\n", screenshotPath))
		formatted.WriteString("*The screenshot shows the current state of the page when this message was sent.*\n")
	}

	// Add timestamp
	formatted.WriteString(fmt.Sprintf("\n**Sent:** %s\n", time.Now().Format("2006-01-02 15:04:05")))

	return formatted.String()
}
