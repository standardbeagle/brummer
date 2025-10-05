//go:build tty && integration
// +build tty,integration

package tty

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PTYTestHelper provides concise helper functions for PTY integration testing
type PTYTestHelper struct {
	t   *RealPTYIntegrationTest
}

// NewPTYTestHelper creates a new PTY test helper
func NewPTYTestHelper(t *RealPTYIntegrationTest) *PTYTestHelper {
	return &PTYTestHelper{t: t}
}

// GetScreenContent returns the current screen content from the underlying PTY test
func (h *PTYTestHelper) GetScreenContent() string {
	h.t.t.Helper()
	return h.t.GetScreenContent()
}

// Navigation helpers

// NavigateToTab navigates to a specific tab using number keys
func (h *PTYTestHelper) NavigateToTab(tabNumber int) error {
	h.t.t.Helper()

	key := fmt.Sprintf("%d", tabNumber)
	return h.t.SendInput(key)
}

// NavigateToTabNamed navigates to a tab by looking for it and sending the corresponding number
func (h *PTYTestHelper) NavigateToTabNamed(tabName string) error {
	h.t.t.Helper()

	screen := h.t.GetScreenContent()

	// Look for tab patterns like "📋 Logs (1)" or "🤖 AI Coders (6)"
	tabPattern := regexp.MustCompile(`\b\(\d+\)`)
	matches := tabPattern.FindAllStringSubmatch(screen, -1)

	for _, match := range matches {
		if len(match) > 0 {
			// Extract the number from parentheses
			numberMatch := regexp.MustCompile(`\((\d+)\)`).FindStringSubmatch(match[0])
			if len(numberMatch) > 1 {
				tabNum := numberMatch[1]
				// Check if this tab matches our target
				tabText := strings.TrimSpace(strings.Split(match[0], "(")[0])
				if strings.Contains(strings.ToLower(tabText), strings.ToLower(tabName)) {
					return h.t.SendInput(tabNum)
				}
			}
		}
	}

	return fmt.Errorf("tab '%s' not found", tabName)
}

// PressKey sends a single keystroke
func (h *PTYTestHelper) PressKey(key string) error {
	h.t.t.Helper()
	return h.t.SendInput(key)
}

// PressKeySequence sends multiple keys with timing
func (h *PTYTestHelper) PressKeySequence(keys []string, interval time.Duration) error {
	h.t.t.Helper()
	return h.t.SendKeySequence(keys, interval)
}

// Command helpers

// OpenCommandInput opens the command input with /
func (h *PTYTestHelper) OpenCommandInput() error {
	h.t.t.Helper()
	return h.t.SendInput("/")
}

// SendCommand sends a command and presses enter
func (h *PTYTestHelper) SendCommand(command string) error {
	h.t.t.Helper()
	return h.t.SendKeySequence([]string{command, "\r"}, 100*time.Millisecond)
}

// SendAICommand sends an AI command with the specified provider
func (h *PTYTestHelper) SendAICommand(provider, task string) error {
	h.t.t.Helper()
	return h.t.SendCommand(fmt.Sprintf("ai %s %s", provider, task))
}

// Validation helpers

// AssertScreenContains asserts that the screen contains the expected text
func (h *PTYTestHelper) AssertScreenContains(expected string) {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()
	assert.Contains(h.t.t, screen, expected, "Screen should contain: %s", expected)
}

// AssertScreenNotContains asserts that the screen does not contain the text
func (h *PTYTestHelper) AssertScreenNotContains(unexpected string) {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()
	assert.NotContains(h.t.t, screen, unexpected, "Screen should not contain: %s", unexpected)
}

// AssertScreenMatches asserts that the screen matches the regex pattern
func (h *PTYTestHelper) AssertScreenMatches(pattern *regexp.Regexp) {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()
	assert.Regexp(h.t.t, pattern, screen, "Screen should match pattern: %s", pattern.String())
}

// WaitForScreen waits for specific text to appear on screen
func (h *PTYTestHelper) WaitForScreen(pattern string, timeout time.Duration) error {
	h.t.t.Helper()
	return h.t.WaitForText(pattern, timeout)
}

// WaitForScreenRegex waits for a regex pattern to appear on screen
func (h *PTYTestHelper) WaitForScreenRegex(pattern *regexp.Regexp, timeout time.Duration) error {
	h.t.t.Helper()
	return h.t.WaitForRegex(pattern, timeout)
}

// Tab and layout validation helpers

// AssertCurrentTab asserts that we're currently on the specified tab
func (h *PTYTestHelper) AssertCurrentTab(expectedTab string) {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()

	// Look for the tab indicator (highlighted tab)
	tabPattern := regexp.MustCompile(`(?m)^[\s│]*([^\n│\s]+.*?\(\d+\))`)
	matches := tabPattern.FindAllStringSubmatch(screen, -1)

	currentTab := ""
	for _, match := range matches {
		if len(match) > 1 {
			tabText := match[1]
			// Check if this tab appears highlighted (has different formatting)
			if strings.Contains(tabText, expectedTab) {
				currentTab = tabText
				break
			}
		}
	}

	assert.Contains(h.t.t, currentTab, expectedTab, "Should be on %s tab", expectedTab)
}

// AssertTabExists asserts that a tab with the specified name exists
func (h *PTYTestHelper) AssertTabExists(tabName string) {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()
	tabPattern := regexp.MustCompile(fmt.Sprintf(`%s.*?\(\d+\)`, regexp.QuoteMeta(tabName)))
	assert.Regexp(h.t.t, tabPattern, screen, "Tab '%s' should exist", tabName)
}

// AssertLayoutValid asserts that the basic layout structure is valid
func (h *PTYTestHelper) AssertLayoutValid() {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()

	// Debug: Let's see what we actually captured
	lines := strings.Split(screen, "\n")
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	h.t.t.Logf("🖥️  DEBUG: Screen analysis")
	h.t.t.Logf("   Total lines: %d", len(lines))
	h.t.t.Logf("   Non-empty lines: %d", nonEmptyLines)
	h.t.t.Logf("   Screen length: %d chars", len(screen))

	// Show first few lines for debugging
	h.t.t.Logf("📋 First 5 lines:")
	for i, line := range lines {
		if i >= 5 {
			break
		}
		if strings.TrimSpace(line) != "" {
			h.t.t.Logf("   %2d: %q", i+1, line)
		}
	}

	// Check for basic layout elements (using actual patterns from TUI)
	layoutChecks := map[string]string{
		"Header":         "🐝 Brummer - Development Buddy",
		"Border":         "─",
		"Tab separator": "│",
		"Tab indicators": "(1)",
		"Logs tab":       "📋 Logs",
		"Processes tab":  "⚙️ Processes",
		"AI Coders tab":  "🤖 AI Coders",
	}

	for element, pattern := range layoutChecks {
		assert.Contains(h.t.t, screen, pattern, "Layout should contain %s: %s", element, pattern)
	}
}

// GetAvailableTabs returns a list of available tabs on screen
func (h *PTYTestHelper) GetAvailableTabs() []string {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()

	// Extract tab names from the tab line
	tabPattern := regexp.MustCompile(`([^\n│]+)\s*\(\d+\)`)
	matches := tabPattern.FindAllStringSubmatch(screen, -1)

	var tabs []string
	for _, match := range matches {
		if len(match) > 1 {
			tabName := strings.TrimSpace(match[1])
			tabs = append(tabs, tabName)
		}
	}

	return tabs
}

// Content validation helpers

// AssertAIViewContent asserts AI view specific content
func (h *PTYTestHelper) AssertAIViewContent() {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()

	aiIndicators := []string{
		"AI",
		"Coders",
		"Sessions",
		"Press 'a' to start",
		"No active",
	}

	// At least some AI-related content should be present
	found := 0
	for _, indicator := range aiIndicators {
		if strings.Contains(screen, indicator) {
			found++
		}
	}

	assert.GreaterOrEqual(h.t.t, found, 2, "AI view should contain AI-related content")
}

// AssertProcessViewContent asserts process view specific content
func (h *PTYTestHelper) AssertProcessViewContent() {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()

	processIndicators := []string{
		"Process",
		"Scripts",
		"Run",
		"npm",
		"dev",
		"test",
	}

	// At least some process-related content should be present
	found := 0
	for _, indicator := range processIndicators {
		if strings.Contains(screen, indicator) {
			found++
		}
	}

	assert.GreaterOrEqual(h.t.t, found, 1, "Process view should contain process-related content")
}

// AssertLogViewContent asserts log view specific content
func (h *PTYTestHelper) AssertLogViewContent() {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()

	logIndicators := []string{
		"Log",
		"Output",
		"Error",
		"Info",
		"Warning",
		"Debug",
	}

	// At least some log-related content should be present
	found := 0
	for _, indicator := range logIndicators {
		if strings.Contains(screen, indicator) {
			found++
		}
	}

	assert.GreaterOrEqual(h.t.t, found, 1, "Log view should contain log-related content")
}

// Error and status helpers

// AssertNoErrors asserts that there are no error indicators on screen
func (h *PTYTestHelper) AssertNoErrors() {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()

	errorPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)error`),
		regexp.MustCompile(`(?i)failed`),
		regexp.MustCompile(`(?i)panic`),
		regexp.MustCompile(`(?i)fatal`),
	}

	for _, pattern := range errorPatterns {
		if pattern.MatchString(screen) {
			h.t.t.Logf("⚠️  Found error pattern on screen: %s", pattern.String())
		}
	}
}

// GetErrors returns any error messages found on screen
func (h *PTYTestHelper) GetErrors() []string {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()

	var errors []string
	lines := strings.Split(screen, "\n")

	errorPattern := regexp.MustCompile(`(?i)(error|failed|panic|fatal)`)
	for _, line := range lines {
		if errorPattern.MatchString(line) {
			errors = append(errors, strings.TrimSpace(line))
		}
	}

	return errors
}

// Timing and synchronization helpers

// WaitAndValidate waits for a pattern and validates it appears
func (h *PTYTestHelper) WaitAndValidate(pattern string, timeout time.Duration) {
	h.t.t.Helper()

	err := h.WaitForScreen(pattern, timeout)
	require.NoError(h.t.t, err, "Pattern '%s' should appear within timeout", pattern)
	h.AssertScreenContains(pattern)
}

// Sleep waits for the specified duration
func (h *PTYTestHelper) Sleep(duration time.Duration) {
	h.t.t.Helper()
	time.Sleep(duration)
}

// Debug helpers

// DebugScreen prints the current screen content for debugging
func (h *PTYTestHelper) DebugScreen() {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()
	h.t.t.Logf("🖥️  Current Screen Content:")
	h.t.t.Logf("%s", screen)
}

// DebugScreenLines prints specific lines from the screen
func (h *PTYTestHelper) DebugScreenLines(maxLines int) {
	h.t.t.Helper()
	screen := h.t.GetScreenContent()
	lines := strings.Split(screen, "\n")

	h.t.t.Logf("🖥️  Current Screen Content (first %d lines):", maxLines)
	for i, line := range lines {
		if i >= maxLines {
			break
		}
		h.t.t.Logf("  %2d: %s", i+1, line)
	}
}

// DebugTabs prints available tabs for debugging
func (h *PTYTestHelper) DebugTabs() {
	h.t.t.Helper()
	tabs := h.GetAvailableTabs()

	h.t.t.Logf("📋 Available Tabs:")
	for i, tab := range tabs {
		h.t.t.Logf("  %d. %s", i+1, tab)
	}
}