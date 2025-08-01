package logs

import (
	"testing"
	"time"
)

func TestDetectURLsInContent(t *testing.T) {
	store := NewStore(100, nil)
	defer store.Close()

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "Single URL",
			content:  "Server running at http://localhost:3000",
			expected: []string{"http://localhost:3000"},
		},
		{
			name:     "Multiple URLs",
			content:  "API at http://localhost:3001 and Admin at https://admin.example.com",
			expected: []string{"http://localhost:3001", "https://admin.example.com"},
		},
		{
			name:     "URL with path",
			content:  "Check http://localhost:8080/api/v1/health for status",
			expected: []string{"http://localhost:8080/api/v1/health"},
		},
		{
			name:     "No URLs",
			content:  "This is just a regular log message",
			expected: []string{},
		},
		{
			name:     "URL with trailing punctuation",
			content:  "Visit http://example.com.",
			expected: []string{"http://example.com"},
		},
		{
			name:     "Multiple same URLs",
			content:  "Starting http://localhost:3000, server at http://localhost:3000",
			expected: []string{"http://localhost:3000", "http://localhost:3000"},
		},
		{
			name:     "Incomplete URL with trailing colon",
			content:  "Starting https://localhost:",
			expected: []string{}, // Should not detect incomplete URLs
		},
		{
			name:     "URL with port followed by colon",
			content:  "Server at http://localhost:3000: ready",
			expected: []string{"http://localhost:3000"},
		},
		{
			name:     "HTTPS URL with no port",
			content:  "Secure server at https://example.com ready",
			expected: []string{"https://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls := store.DetectURLsInContent(tt.content)

			if len(urls) != len(tt.expected) {
				t.Errorf("DetectURLsInContent() returned %d URLs, want %d", len(urls), len(tt.expected))
				t.Errorf("Got: %v", urls)
				t.Errorf("Want: %v", tt.expected)
				t.Logf("Input content: %q", tt.content)
				return
			}

			for i, url := range urls {
				if url != tt.expected[i] {
					t.Errorf("DetectURLsInContent()[%d] = %q, want %q", i, url, tt.expected[i])
				}
			}
		})
	}
}

func TestUpdateProxyURL(t *testing.T) {
	store := NewStore(100, nil)
	defer store.Close()

	// Add a URL through normal log processing
	store.Add("test-process", "test", "Server at http://localhost:3000", false)

	// Wait for async processing to complete
	time.Sleep(10 * time.Millisecond)

	// Get the URL entry
	urls := store.GetURLs()
	if len(urls) != 1 {
		t.Fatalf("Expected 1 URL, got %d", len(urls))
	}

	originalURL := urls[0].URL
	proxyURL := "http://localhost:8888"

	// Update the proxy URL
	store.UpdateProxyURL(originalURL, proxyURL)

	// Verify the update
	urls = store.GetURLs()
	if len(urls) != 1 {
		t.Fatalf("Expected 1 URL after update, got %d", len(urls))
	}

	if urls[0].ProxyURL != proxyURL {
		t.Errorf("UpdateProxyURL() failed: got ProxyURL = %q, want %q", urls[0].ProxyURL, proxyURL)
	}

	if urls[0].URL != originalURL {
		t.Errorf("UpdateProxyURL() changed original URL: got %q, want %q", urls[0].URL, originalURL)
	}
}
