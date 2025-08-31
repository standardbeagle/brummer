package components

import (
	"regexp"
	"strings"
)

// URL detection regex patterns
var (
	// Match URLs with protocols
	urlRegex = regexp.MustCompile(`(?i)\b((?:https?|wss?|ftp)://[\w\-._~:/?#[\]@!$&'()*+,;=%]+)\b`)
	
	// Match localhost URLs with ports (common in dev)
	localhostRegex = regexp.MustCompile(`(?i)\b(localhost|127\.0\.0\.1|0\.0\.0\.0|::1)(?::(\d{1,5}))?\b`)
)

// extractURLs extracts URLs from a log line
func extractURLs(line string) []string {
	urls := []string{}
	seen := make(map[string]bool)

	// Find standard URLs with protocols
	matches := urlRegex.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 1 {
			url := match[1]
			// Clean up common artifacts
			url = strings.TrimSuffix(url, ".")
			url = strings.TrimSuffix(url, ",")
			url = strings.TrimSuffix(url, ";")
			url = strings.TrimSuffix(url, ")")
			url = strings.TrimSuffix(url, "]")
			
			if !seen[url] {
				urls = append(urls, url)
				seen[url] = true
			}
		}
	}

	// Find localhost URLs (common in development)
	localhostMatches := localhostRegex.FindAllStringSubmatch(line, -1)
	for _, match := range localhostMatches {
		if len(match) > 2 && match[2] != "" { // Has port
			// Check if line contains "listening" or "server" or "running"
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "listen") ||
				strings.Contains(lowerLine, "server") ||
				strings.Contains(lowerLine, "running") ||
				strings.Contains(lowerLine, "started") ||
				strings.Contains(lowerLine, "ready") {
				
				host := match[1]
				port := match[2]
				
				// Normalize localhost variations
				if host == "0.0.0.0" || host == "::" || host == "::1" {
					host = "localhost"
				}
				
				// Assume HTTP for common web ports
				protocol := "http"
				if port == "443" || port == "8443" || port == "4443" {
					protocol = "https"
				}
				
				url := protocol + "://" + host + ":" + port
				if !seen[url] {
					urls = append(urls, url)
					seen[url] = true
				}
			}
		}
	}

	// Look for common development server patterns
	if strings.Contains(line, "webpack") || strings.Contains(line, "vite") || strings.Contains(line, "next") {
		// Extract port from patterns like "Local:   http://localhost:3000"
		if strings.Contains(line, "Local:") || strings.Contains(line, "Network:") {
			// Already handled by URL regex above
		}
	}

	return urls
}

// getURLType determines the type of URL (http, https, ws, etc.)
func getURLType(url string) string {
	url = strings.ToLower(url)
	
	if strings.HasPrefix(url, "https://") {
		return "https"
	} else if strings.HasPrefix(url, "http://") {
		return "http"
	} else if strings.HasPrefix(url, "wss://") {
		return "wss"
	} else if strings.HasPrefix(url, "ws://") {
		return "ws"
	} else if strings.HasPrefix(url, "ftp://") {
		return "ftp"
	}
	
	return "unknown"
}

// isLocalURL checks if a URL is local (localhost, 127.0.0.1, etc.)
func isLocalURL(url string) bool {
	return strings.Contains(url, "localhost") ||
		strings.Contains(url, "127.0.0.1") ||
		strings.Contains(url, "0.0.0.0") ||
		strings.Contains(url, "::1")
}