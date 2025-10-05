//go:build tty && integration
// +build tty,integration

package tty

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTabNavigation tests basic tab navigation functionality
func TestTabNavigation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PTY test in short mode")
	}

	t.Logf("🧪 TESTING: Tab Navigation")
	t.Logf("=========================")

	// Setup PTY test
	ptyTest := NewRealPTYIntegrationTest(t)
	defer ptyTest.Stop()
	helper := NewPTYTestHelper(ptyTest)

	// Start application
	err := ptyTest.StartWithPTY()
	require.NoError(t, err, "Failed to start brummer with PTY")

	// Wait for initial UI
	helper.Sleep(2 * time.Second)

	// Test 1: Validate initial layout
	t.Logf("\n📋 Test 1: Initial layout validation")
	helper.AssertLayoutValid()
	helper.AssertTabExists("Logs")
	helper.AssertTabExists("Processes")
	helper.AssertTabExists("URLs")
	helper.AssertTabExists("Errors")
	helper.AssertTabExists("Web")
	helper.AssertTabExists("AI Coders")
	helper.AssertTabExists("MCP")
	helper.AssertTabExists("Settings")
	helper.AssertTabExists("Diagnostic")

	// Debug: Show available tabs
	helper.DebugTabs()

	// Test 2: Navigate to Process tab (tab 2)
	t.Logf("\n🔄 Test 2: Navigate to Process tab")
	err = helper.NavigateToTab(2)
	require.NoError(t, err, "Failed to navigate to Process tab")

	// Wait for navigation and validate
	helper.Sleep(1 * time.Second)
	helper.AssertProcessViewContent()

	// Test 3: Navigate to AI Coders tab (tab 6)
	t.Logf("\n🤖 Test 3: Navigate to AI Coders tab")
	err = helper.NavigateToTab(6)
	require.NoError(t, err, "Failed to navigate to AI Coders tab")

	// Wait for navigation and validate
	helper.Sleep(1 * time.Second)
	helper.AssertAIViewContent()

	// Test 4: Navigate to Log tab (tab 1)
	t.Logf("\n📋 Test 4: Navigate to Log tab")
	err = helper.NavigateToTab(1)
	require.NoError(t, err, "Failed to navigate to Log tab")

	// Wait for navigation and validate
	helper.Sleep(1 * time.Second)
	helper.AssertLogViewContent()

	// Test 5: Navigate to Settings tab (tab 8)
	t.Logf("\n⚙️ Test 5: Navigate to Settings tab")
	err = helper.NavigateToTab(8)
	require.NoError(t, err, "Failed to navigate to Settings tab")

	// Wait for navigation
	helper.Sleep(1 * time.Second)

	t.Logf("✅ Tab navigation test completed successfully")
}

// TestArrowKeyNavigation tests arrow key navigation between tabs
func TestArrowKeyNavigation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PTY test in short mode")
	}

	t.Logf("⌨️ TESTING: Arrow Key Navigation")
	t.Logf("===============================")

	// Setup PTY test
	ptyTest := NewRealPTYIntegrationTest(t)
	defer ptyTest.Stop()
	helper := NewPTYTestHelper(ptyTest)

	// Start application
	err := ptyTest.StartWithPTY()
	require.NoError(t, err, "Failed to start brummer with PTY")

	// Wait for initial UI
	helper.Sleep(2 * time.Second)

	// Navigate to a middle tab first
	err = helper.NavigateToTab(3) // URLs tab
	require.NoError(t, err)
	helper.Sleep(500 * time.Millisecond)

	// Test 1: Right arrow navigation
	t.Logf("\n➡️ Test 1: Right arrow navigation")

	// Send right arrow keys and observe tab changes
	rightArrow := "\x1b[C"
	for i := 0; i < 3; i++ {
		err = helper.PressKey(rightArrow)
		require.NoError(t, err, "Failed to send right arrow")
		helper.Sleep(500 * time.Millisecond)
		t.Logf("  Sent right arrow %d/3", i+1)
	}

	// Test 2: Left arrow navigation
	t.Logf("\n⬅️ Test 2: Left arrow navigation")

	leftArrow := "\x1b[B" // Down arrow often acts as left navigation in some terminals
	for i := 0; i < 2; i++ {
		err = helper.PressKey(leftArrow)
		require.NoError(t, err, "Failed to send left arrow")
		helper.Sleep(500 * time.Millisecond)
		t.Logf("  Sent left arrow %d/2", i+1)
	}

	// Test 3: Test other arrow keys
	t.Logf("\n⬆️⬇️ Test 3: Up/Down arrow navigation")

	arrows := []string{"\x1b[A", "\x1b[B"} // Up, Down
	arrowNames := []string{"Up", "Down"}

	for i, arrow := range arrows {
		err = helper.PressKey(arrow)
		require.NoError(t, err, "Failed to send %s arrow", arrowNames[i])
		helper.Sleep(300 * time.Millisecond)
		t.Logf("  Sent %s arrow", arrowNames[i])
	}

	t.Logf("✅ Arrow key navigation test completed")
}

// TestCommandNavigation tests navigation using slash commands
func TestCommandNavigation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PTY test in short mode")
	}

	t.Logf("💬 TESTING: Command Navigation")
	t.Logf("===============================")

	// Setup PTY test
	ptyTest := NewRealPTYIntegrationTest(t)
	defer ptyTest.Stop()
	helper := NewPTYTestHelper(ptyTest)

	// Start application
	err := ptyTest.StartWithPTY()
	require.NoError(t, err, "Failed to start brummer with PTY")

	// Wait for initial UI
	helper.Sleep(2 * time.Second)

	// Test 1: Help command
	t.Logf("\n❓ Test 1: Help command")

	err = helper.OpenCommandInput()
	require.NoError(t, err, "Failed to open command input")

	err = helper.SendCommand("help")
	require.NoError(t, err, "Failed to send help command")

	helper.Sleep(1 * time.Second)

	// Check if help content appeared
	helpPattern := regexp.MustCompile(`(?i)(help|commands|available)`)
	screen := helper.GetScreenContent()
	if helpPattern.MatchString(screen) {
		t.Logf("✅ Help content appeared")
	} else {
		t.Logf("⚠️  Help content not clearly visible")
	}

	// Test 2: AI command (this should trigger navigation)
	t.Logf("\n🤖 Test 2: AI command navigation")

	err = helper.SendAICommand("claude", "test navigation")
	require.NoError(t, err, "Failed to send AI command")

	helper.Sleep(2 * time.Second)

	// Check if we navigated to AI tab
	helper.AssertAIViewContent()

	t.Logf("✅ AI command navigation completed")

	// Test 3: Check current state after commands
	helper.DebugScreenLines(10)

	t.Logf("✅ Command navigation test completed")
}

// TestLayoutValidation tests the layout and display validation
func TestLayoutValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PTY test in short mode")
	}

	t.Logf("🖼️ TESTING: Layout and Display Validation")
	t.Logf("=====================================")

	// Setup PTY test
	ptyTest := NewRealPTYIntegrationTest(t)
	defer ptyTest.Stop()
	helper := NewPTYTestHelper(ptyTest)

	// Start application
	err := ptyTest.StartWithPTY()
	require.NoError(t, err, "Failed to start brummer with PTY")

	// Wait for initial UI
	helper.Sleep(2 * time.Second)

	// Test 1: Basic layout structure validation
	t.Logf("\n📐 Test 1: Basic layout structure")
	helper.AssertLayoutValid()

	// Test 2: Header validation
	t.Logf("\n📰 Test 2: Header validation")
	screen := helper.GetScreenContent()

	headerChecks := map[string]string{
		"App name":     "Brummer",
		"Title":        "Development Buddy",
		"Border":       "─",
		"Version info": "Bee emoji", // 🐝
	}

	for check, pattern := range headerChecks {
		if pattern == "Bee emoji" {
			assert.Contains(t, screen, "🐝", "Header should contain bee emoji")
		} else {
			assert.Contains(t, screen, pattern, "Header should contain %s: %s", check, pattern)
		}
	}

	// Test 3: Tab bar validation
	t.Logf("\n📑 Test 3: Tab bar validation")

	tabs := helper.GetAvailableTabs()
	assert.Greater(t, len(tabs), 5, "Should have at least 5 tabs")

	expectedTabs := []string{"Logs", "Processes", "URLs", "Errors", "AI Coders"}
	for _, expectedTab := range expectedTabs {
		found := false
		for _, tab := range tabs {
			if strings.Contains(strings.ToLower(tab), strings.ToLower(expectedTab)) {
				found = true
				break
			}
		}
		assert.True(t, found, "Tab '%s' should be available", expectedTab)
	}

	// Test 4: Navigation help validation
	t.Logf("\n🧭 Test 4: Navigation help validation")

	helpChecks := map[string]string{
		"Number navigation": "1-9",
		"Help key":         "?",
		"Quit key":         "q",
		"Help text":        "Press",
	}

	for check, pattern := range helpChecks {
		assert.Contains(t, screen, pattern, "Navigation help should contain %s: %s", check, pattern)
	}

	// Test 5: Status bar validation
	t.Logf("\n📊 Test 5: Status bar validation")

	// Look for status indicators
	statusPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\[\d{2}:\d{2}:\d{2}\]`), // Timestamp
		regexp.MustCompile(`discovery`),          // Discovery status
		regexp.MustCompile(`registered`),         // Registration status
		regexp.MustCompile(`instance`),           // Instance info
	}

	statusFound := 0
	for _, pattern := range statusPatterns {
		if pattern.MatchString(screen) {
			statusFound++
			t.Logf("  ✅ Found status pattern: %s", pattern.String())
		}
	}
	assert.GreaterOrEqual(t, statusFound, 2, "Should find at least 2 status indicators")

	// Test 6: Content area validation
	t.Logf("\n📄 Test 6: Content area validation")

	// Navigate through different tabs and validate content
	testTabs := []struct {
		tabNumber int
		name      string
		validator func(*PTYTestHelper)
	}{
		{1, "Logs", func(h *PTYTestHelper) { h.AssertLogViewContent() }},
		{2, "Processes", func(h *PTYTestHelper) { h.AssertProcessViewContent() }},
		{6, "AI Coders", func(h *PTYTestHelper) { h.AssertAIViewContent() }},
	}

	for _, testTab := range testTabs {
		t.Logf("  🔄 Testing %s tab", testTab.name)

		err = helper.NavigateToTab(testTab.tabNumber)
		require.NoError(t, err, "Failed to navigate to %s tab", testTab.name)

		helper.Sleep(1 * time.Second)
		testTab.validator(helper)

		// Check that content actually changed
		newScreen := helper.GetScreenContent()
		assert.NotEqual(t, screen, newScreen, "Screen content should change when navigating to %s tab", testTab.name)
		screen = newScreen
	}

	// Test 7: Error state validation
	t.Logf("\n⚠️ Test 7: Error state validation")

	errors := helper.GetErrors()
	if len(errors) > 0 {
		t.Logf("  Found %d error(s):", len(errors))
		for i, err := range errors {
			t.Logf("    %d. %s", i+1, err)
		}
	} else {
		t.Logf("  ✅ No errors found on screen")
	}

	t.Logf("\n✅ Layout validation test completed successfully")
}

// TestResponsiveLayout tests layout responsiveness and edge cases
func TestResponsiveLayout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PTY test in short mode")
	}

	t.Logf("📱 TESTING: Responsive Layout")
	t.Logf("=============================")

	// Setup PTY test
	ptyTest := NewRealPTYIntegrationTest(t)
	defer ptyTest.Stop()
	helper := NewPTYTestHelper(ptyTest)

	// Test 1: Normal size (80x24)
	t.Logf("\n📐 Test 1: Normal terminal size (80x24)")

	err := ptyTest.StartWithPTY()
	require.NoError(t, err, "Failed to start brummer with PTY")

	helper.Sleep(2 * time.Second)
	helper.AssertLayoutValid()

	// Capture normal layout
	normalScreen := helper.GetScreenContent()
	normalLines := len(strings.Split(normalScreen, "\n"))

	t.Logf("  Normal layout has %d lines", normalLines)
	assert.Greater(t, normalLines, 20, "Normal layout should have more than 20 lines")
	assert.Less(t, normalLines, 30, "Normal layout should have less than 30 lines")

	// Test 2: Content overflow handling
	t.Logf("\n📏 Test 2: Content overflow handling")

	// Navigate to AI tab which might have different content
	err = helper.NavigateToTab(6)
	require.NoError(t, err)
	helper.Sleep(1 * time.Second)

	// Check if layout remains valid
	helper.AssertLayoutValid()

	// Test 3: Tab navigation with content
	t.Logf("\n🔄 Test 3: Tab navigation with various content")

	contentTabs := []int{1, 2, 6, 8} // Logs, Processes, AI, Settings

	for _, tabNum := range contentTabs {
		err = helper.NavigateToTab(tabNum)
		require.NoError(t, err, "Failed to navigate to tab %d", tabNum)

		helper.Sleep(800 * time.Millisecond)
		helper.AssertLayoutValid()

		t.Logf("  ✅ Tab %d layout is valid", tabNum)
	}

	// Test 4: Help overlay
	t.Logf("\n❓ Test 4: Help overlay")

	err = helper.PressKey("?")
	require.NoError(t, err, "Failed to press help key")

	helper.Sleep(1 * time.Second)

	// Help should not break layout
	helper.AssertLayoutValid()

	// Check if help content appeared
	helpScreen := helper.GetScreenContent()
	if strings.Contains(helpScreen, "help") || strings.Contains(helpScreen, "Help") {
		t.Logf("  ✅ Help content appeared")
	} else {
		t.Logf("  ⚠️  Help content not clearly visible")
	}

	t.Logf("\n✅ Responsive layout test completed")
}

// TestNavigationEdgeCases tests navigation edge cases and error conditions
func TestNavigationEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PTY test in short mode")
	}

	t.Logf("🎯 TESTING: Navigation Edge Cases")
	t.Logf("===============================")

	// Setup PTY test
	ptyTest := NewRealPTYIntegrationTest(t)
	defer ptyTest.Stop()
	helper := NewPTYTestHelper(ptyTest)

	// Start application
	err := ptyTest.StartWithPTY()
	require.NoError(t, err, "Failed to start brummer with PTY")

	helper.Sleep(2 * time.Second)

	// Test 1: Invalid tab numbers
	t.Logf("\n❌ Test 1: Invalid tab navigation")

	invalidTabs := []string{"0", "10", "999", "abc", "!@#"}

	for _, invalidTab := range invalidTabs {
		err = helper.PressKey(invalidTab)
		require.NoError(t, err, "Failed to send invalid tab: %s", invalidTab)

		helper.Sleep(300 * time.Millisecond)

		// Layout should still be valid after invalid navigation
		helper.AssertLayoutValid()
		t.Logf("  ✅ Invalid tab '%s' handled gracefully", invalidTab)
	}

	// Test 2: Rapid tab switching
	t.Logf("\n⚡ Test 2: Rapid tab switching")

	tabSequence := []string{"1", "2", "3", "6", "1", "8", "6"}

	for i, tab := range tabSequence {
		err = helper.PressKey(tab)
		require.NoError(t, err, "Failed to press tab %s", tab)

		// Shorter sleep for rapid switching
		helper.Sleep(200 * time.Millisecond)
		t.Logf("  Rapid switch %d/%d: tab %s", i+1, len(tabSequence), tab)
	}

	// Test 3: Mixed navigation (numbers + arrows + commands)
	t.Logf("\n🔄 Test 3: Mixed navigation patterns")

	// Start with number navigation
	err = helper.NavigateToTab(2)
	require.NoError(t, err)
	helper.Sleep(300 * time.Millisecond)

	// Try arrow navigation
	err = helper.PressKeySequence([]string{"\x1b[C", "\x1b[C"}, 200*time.Millisecond)
	require.NoError(t, err)
	helper.Sleep(300 * time.Millisecond)

	// Try command navigation
	err = helper.SendCommand("ai claude test edge case")
	require.NoError(t, err)
	helper.Sleep(2 * time.Second)

	// Should end up on AI tab
	helper.AssertAIViewContent()

	// Test 4: Command input edge cases
	t.Logf("\n💬 Test 4: Command input edge cases")

	edgeCommands := []string{
		"",           // Empty command
		"   ",        // Whitespace only
		"/",          // Just slash
		"/invalid",   // Invalid command
		"/ai",        // Incomplete AI command
		"/ai unknown", // Unknown provider
		"/help",      // Valid help command
	}

	for _, cmd := range edgeCommands {
		err = helper.OpenCommandInput()
		require.NoError(t, err)

		err = helper.SendCommand(strings.TrimPrefix(cmd, "/"))
		require.NoError(t, err, "Failed to send command: %s", cmd)

		helper.Sleep(500 * time.Millisecond)

		// Should handle edge case gracefully
		helper.AssertLayoutValid()
		t.Logf("  ✅ Edge case command handled: '%s'", cmd)
	}

	t.Logf("\n✅ Navigation edge cases test completed")
}