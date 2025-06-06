package logs

import (
	"regexp"
	"strings"

	"github.com/beagle/brummer/pkg/events"
)

type EventDetector struct {
	eventBus      *events.EventBus
	errorPatterns []*regexp.Regexp
	buildPatterns []*regexp.Regexp
	testPatterns  []*regexp.Regexp
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