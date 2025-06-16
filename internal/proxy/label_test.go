package proxy

import (
	"regexp"
	"strings"
	"testing"
)

// Copy of the extractURLLabel function for testing
func extractURLLabel(logLine, processName string) string {
	line := strings.TrimSpace(logLine)

	patterns := []struct {
		regex   *regexp.Regexp
		extract func([]string) string
	}{
		{regexp.MustCompile(`(?i)\[([^\]]+)\].*https?://`), func(m []string) string { return m[1] }},
		{regexp.MustCompile(`(?i)^([^:\s]+):\s+.*(?:server|listening|started|running).*https?://`), func(m []string) string { return m[1] }},
		{regexp.MustCompile(`(?i)^(\w+)\s+server\s+(?:listening|started|running)\s+(?:on|at).*https?://`), func(m []string) string { return m[1] + " Server" }},
		{regexp.MustCompile(`(?i)^(\w+)\s+(?:started|running|listening)\s+(?:on|at).*https?://`), func(m []string) string { return m[1] }},
		{regexp.MustCompile(`(?i)(\w+):\s+https?://`), func(m []string) string { return m[1] }},
		{regexp.MustCompile(`(?i)(\w+)\s+(?:ready|available)\s+at\s+https?://`), func(m []string) string { return m[1] }},
		{regexp.MustCompile(`(?i)(frontend|backend|api|admin|dashboard|web|client|server)\s+.*https?://`), func(m []string) string { return strings.Title(strings.ToLower(m[1])) }},
		{regexp.MustCompile(`(?i)https?://.*\s+(frontend|backend|api|admin|dashboard|web|client|server)`), func(m []string) string { return strings.Title(strings.ToLower(m[1])) }},
	}

	for _, p := range patterns {
		if matches := p.regex.FindStringSubmatch(line); len(matches) > 1 {
			label := p.extract(matches)
			label = strings.TrimSpace(label)
			if label != "" && label != processName {
				return label
			}
		}
	}

	return processName
}

func TestExtractURLLabel(t *testing.T) {
	tests := []struct {
		name        string
		logLine     string
		processName string
		expected    string
	}{
		{
			name:        "Bracketed label",
			logLine:     "[Frontend] Server listening on http://localhost:3000",
			processName: "dev",
			expected:    "Frontend",
		},
		{
			name:        "Process name with colon",
			logLine:     "API: Server started on http://localhost:8080",
			processName: "backend",
			expected:    "API",
		},
		{
			name:        "Service server pattern",
			logLine:     "Frontend server listening on http://localhost:3000",
			processName: "web",
			expected:    "Frontend Server",
		},
		{
			name:        "Started on pattern",
			logLine:     "Backend started on http://localhost:8080",
			processName: "api",
			expected:    "Backend",
		},
		{
			name:        "Simple colon pattern",
			logLine:     "Local: http://localhost:5173/",
			processName: "vite",
			expected:    "Local",
		},
		{
			name:        "Ready at pattern",
			logLine:     "Server ready at http://localhost:3000",
			processName: "app",
			expected:    "Server",
		},
		{
			name:        "Frontend keyword",
			logLine:     "Frontend is running at http://localhost:3000",
			processName: "dev",
			expected:    "Frontend",
		},
		{
			name:        "API keyword after URL",
			logLine:     "Listening on http://localhost:8080 for API requests",
			processName: "server",
			expected:    "Api",
		},
		{
			name:        "No extractable label",
			logLine:     "Listening on http://localhost:3000",
			processName: "myapp",
			expected:    "myapp",
		},
		{
			name:        "Complex Vite log",
			logLine:     "  ➜  Local:   http://localhost:5173/",
			processName: "dev",
			expected:    "Local",
		},
		{
			name:        "Next.js style",
			logLine:     "- Local:        http://localhost:3000",
			processName: "nextjs",
			expected:    "Local",
		},
		{
			name:        "Express style",
			logLine:     "App listening on http://localhost:8000",
			processName: "server",
			expected:    "App",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractURLLabel(tt.logLine, tt.processName)
			if result != tt.expected {
				t.Errorf("extractURLLabel(%q, %q) = %q, want %q",
					tt.logLine, tt.processName, result, tt.expected)
			}
		})
	}
}

func TestExtractURLLabelEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		logLine     string
		processName string
		expected    string
	}{
		{
			name:        "Extract more descriptive label",
			logLine:     "dev: Server running on http://localhost:3000",
			processName: "dev",
			expected:    "Server", // Should extract "Server" as it's more descriptive than "dev"
		},
		{
			name:        "Empty line",
			logLine:     "",
			processName: "test",
			expected:    "test",
		},
		{
			name:        "No URL in line",
			logLine:     "Starting application...",
			processName: "app",
			expected:    "app",
		},
		{
			name:        "Multiple URLs",
			logLine:     "Frontend: http://localhost:3000 and http://localhost:3001",
			processName: "multi",
			expected:    "Frontend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractURLLabel(tt.logLine, tt.processName)
			if result != tt.expected {
				t.Errorf("extractURLLabel(%q, %q) = %q, want %q",
					tt.logLine, tt.processName, result, tt.expected)
			}
		})
	}
}
