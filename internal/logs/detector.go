package logs

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/standardbeagle/brummer/pkg/events"
)

type EventDetector struct {
	eventBus       *events.EventBus
	errorPatterns  []*regexp.Regexp
	buildPatterns  []*regexp.Regexp
	testPatterns   []*regexp.Regexp
	urlPatterns    []*regexp.Regexp
	dockerPatterns []*regexp.Regexp
}

func NewEventDetector(eventBus *events.EventBus) *EventDetector {
	return &EventDetector{
		eventBus: eventBus,
		errorPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)error:`),
			regexp.MustCompile(`(?i)failed to`),
			regexp.MustCompile(`(?i)cannot find`),
			regexp.MustCompile(`(?i)undefined`),
			regexp.MustCompile(`(?i)exception:`),
			regexp.MustCompile(`(?i)fatal:`),
			regexp.MustCompile(`(?i)panic:`),
		},
		buildPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)build (completed|succeeded|failed)`),
			regexp.MustCompile(`(?i)compilation (completed|succeeded|failed)`),
			regexp.MustCompile(`(?i)webpack.*built`),
			regexp.MustCompile(`(?i)bundle.*generated`),
			regexp.MustCompile(`(?i)compiled successfully`),
			regexp.MustCompile(`(?i)build error`),
		},
		testPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(\d+) (test|tests?) (passed|failed)`),
			regexp.MustCompile(`(?i)✓|✗|✔|✘`),
			regexp.MustCompile(`(?i)PASS:|FAIL:`),
			regexp.MustCompile(`(?i)test suite (passed|failed)`),
			regexp.MustCompile(`(?i)(\d+) passing`),
			regexp.MustCompile(`(?i)(\d+) failing`),
		},
		urlPatterns: []*regexp.Regexp{
			regexp.MustCompile(`https?://(?:localhost|127\.0\.0\.1|0\.0\.0\.0):(\d+)(?:[/?#][^\s]*)?`),
			regexp.MustCompile(`(?i)server.*running.*(?:on|at).*https?://(?:localhost|127\.0\.0\.1|0\.0\.0\.0):(\d+)`),
			regexp.MustCompile(`(?i)listening.*(?:on|at).*(?:port\s+)?(\d+)`),
			regexp.MustCompile(`(?i)dev.*server.*(?:http://)?(?:localhost|127\.0\.0\.1):(\d+)`),
		},
		dockerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)docker.*run.*-p\s+(\d+):(\d+)`),
			regexp.MustCompile(`(?i)port.*(\d+)/tcp.*->.*(\d+)/tcp`),
			regexp.MustCompile(`(?i)container.*(?:listening|running).*(?:on|port).*(\d+)`),
			regexp.MustCompile(`(?i)exposed.*port.*(\d+)`),
		},
	}
}

func (d *EventDetector) ProcessLogLine(processID, processName, content string, isError bool) {
	lower := strings.ToLower(content)

	if d.isError(content, isError) {
		d.eventBus.Publish(events.Event{
			Type:      events.ErrorDetected,
			ProcessID: processID,
			Data: map[string]interface{}{
				"processName": processName,
				"content":     content,
				"severity":    d.getErrorSeverity(content),
			},
		})
	}

	if d.isBuildEvent(content) {
		d.eventBus.Publish(events.Event{
			Type:      events.BuildEvent,
			ProcessID: processID,
			Data: map[string]interface{}{
				"processName": processName,
				"content":     content,
				"success":     strings.Contains(lower, "success") || strings.Contains(lower, "completed"),
			},
		})
	}

	if testResult := d.detectTestResult(content); testResult != nil {
		eventType := events.TestPassed
		if testResult["failed"].(bool) {
			eventType = events.TestFailed
		}

		d.eventBus.Publish(events.Event{
			Type:      eventType,
			ProcessID: processID,
			Data: map[string]interface{}{
				"processName": processName,
				"content":     content,
				"details":     testResult,
			},
		})
	}

	// Detect URLs for automatic proxy setup
	if urls := d.detectURLs(content); len(urls) > 0 {
		for _, url := range urls {
			d.eventBus.Publish(events.Event{
				Type:      events.URLDetected,
				ProcessID: processID,
				Data: map[string]interface{}{
					"processName": processName,
					"url":         url.URL,
					"port":        url.Port,
					"label":       url.Label,
					"context":     content,
				},
			})
		}
	}

	// Detect Docker port mappings
	if dockerPorts := d.detectDockerPorts(content); len(dockerPorts) > 0 {
		for _, mapping := range dockerPorts {
			d.eventBus.Publish(events.Event{
				Type:      events.DockerPortDetected,
				ProcessID: processID,
				Data: map[string]interface{}{
					"processName":   processName,
					"hostPort":      mapping.HostPort,
					"containerPort": mapping.ContainerPort,
					"context":       content,
				},
			})
		}
	}
}

func (d *EventDetector) isError(content string, isError bool) bool {
	if isError {
		return true
	}

	for _, pattern := range d.errorPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}

	return false
}

func (d *EventDetector) getErrorSeverity(content string) string {
	lower := strings.ToLower(content)

	if strings.Contains(lower, "fatal") || strings.Contains(lower, "panic") {
		return "critical"
	}
	if strings.Contains(lower, "error") || strings.Contains(lower, "failed") {
		return "error"
	}
	if strings.Contains(lower, "warning") || strings.Contains(lower, "warn") {
		return "warning"
	}

	return "info"
}

func (d *EventDetector) isBuildEvent(content string) bool {
	for _, pattern := range d.buildPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

func (d *EventDetector) detectTestResult(content string) map[string]interface{} {
	for _, pattern := range d.testPatterns {
		if matches := pattern.FindStringSubmatch(content); matches != nil {
			lower := strings.ToLower(content)
			failed := strings.Contains(lower, "fail") || strings.Contains(lower, "✗") || strings.Contains(lower, "✘")

			result := map[string]interface{}{
				"failed": failed,
				"line":   content,
			}

			passMatch := regexp.MustCompile(`(\d+)\s*(passing|passed|✓)`).FindStringSubmatch(content)
			if passMatch != nil {
				result["passed"] = passMatch[1]
			}

			failMatch := regexp.MustCompile(`(\d+)\s*(failing|failed|✗)`).FindStringSubmatch(content)
			if failMatch != nil {
				result["failed_count"] = failMatch[1]
			}

			return result
		}
	}

	return nil
}

// DetectedURL represents a URL found in log output
type DetectedURL struct {
	URL   string
	Port  string
	Label string
}

// DockerPortMapping represents a Docker port mapping
type DockerPortMapping struct {
	HostPort      string
	ContainerPort string
}

func (d *EventDetector) detectURLs(content string) []DetectedURL {
	var urls []DetectedURL

	for _, pattern := range d.urlPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				port := match[1]
				url := fmt.Sprintf("http://localhost:%s", port)

				// Try to extract a meaningful label from context
				label := d.extractURLLabel(content, port)

				urls = append(urls, DetectedURL{
					URL:   url,
					Port:  port,
					Label: label,
				})
			}
		}
	}

	return urls
}

func (d *EventDetector) detectDockerPorts(content string) []DockerPortMapping {
	var mappings []DockerPortMapping

	for _, pattern := range d.dockerPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				// Docker port mapping format: hostPort:containerPort
				mappings = append(mappings, DockerPortMapping{
					HostPort:      match[1],
					ContainerPort: match[2],
				})
			} else if len(match) >= 2 {
				// Single port exposure
				port := match[1]
				mappings = append(mappings, DockerPortMapping{
					HostPort:      port,
					ContainerPort: port,
				})
			}
		}
	}

	return mappings
}

func (d *EventDetector) extractURLLabel(content, port string) string {
	lower := strings.ToLower(content)

	// Common service patterns
	if strings.Contains(lower, "frontend") || strings.Contains(lower, "client") {
		return "Frontend"
	}
	if strings.Contains(lower, "backend") || strings.Contains(lower, "server") || strings.Contains(lower, "api") {
		return "Backend"
	}
	if strings.Contains(lower, "admin") {
		return "Admin"
	}
	if strings.Contains(lower, "docs") || strings.Contains(lower, "documentation") {
		return "Docs"
	}
	if strings.Contains(lower, "storybook") {
		return "Storybook"
	}
	if strings.Contains(lower, "webpack") || strings.Contains(lower, "vite") || strings.Contains(lower, "dev server") {
		return "Dev Server"
	}

	// Default label with port
	return fmt.Sprintf("Port %s", port)
}
