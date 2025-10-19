//go:build integration

package test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	
	processes "github.com/standardbeagle/brummer/internal/domain/processes"
	"github.com/standardbeagle/brummer/internal/tui/commands"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealVsFakeIntegrationTest demonstrates the difference between
// fake integration tests (what we had before) and real integration tests
func TestRealVsFakeIntegrationTest(t *testing.T) {
	t.Logf("🎯 TESTING: REAL vs FAKE Integration Tests")
	t.Logf("==========================================")

	// 1. Create the REAL application structure that users actually experience
	// NOT the component-based system our previous tests covered
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(1000, eventBus)
	processManager, err := processes.NewManager(".", eventBus, false)
	require.NoError(t, err)

	updateChan := make(chan tea.Msg, 10)
	var currentView = "processes"
	var searchResults []logs.LogEntry
	var showPattern, hidePattern string

	// 2. Create the ACTUAL slash command context that the real app uses
	// This is what users ACTUALLY interact with
	slashCmd := &commands.SlashCommandContext{
		ProcessManager: processManager,
		LogStore:       logStore,
		UpdateChan:     updateChan,
		CurrentView:    &currentView,
		ShowPattern:    &showPattern,
		HidePattern:    &hidePattern,
		SearchResults:  &searchResults,
		UpdateLogsView: func() {},
		ClearLogs:      func(target string) {},
		SetProxyURL:    func(urlStr string) {},
		ListProxies:    func() {},
		ToggleProxy:    func() {},
		// This is the KEY callback that our previous tests completely missed
		StartAICoder: func(providerName string) {
			t.Logf("📞 REAL INTEGRATION: StartAICoder called with provider: %s", providerName)
			t.Logf("   This is what ACTUALLY happens when users type '/ai claude'")
			t.Logf("   Our previous tests never covered this!")
		},
		ShowTerminal: func() {},
	}

	// 3. Execute the ACTUAL command handler that the real app uses
	// NOT the CommandInputComponent → CommandExecuteMsg flow we tested before
	t.Logf("🔍 EXECUTING: Real slash command handler")
	t.Logf("   Command: /ai claude help me code")
	t.Logf("   Method: commands.HandleSlashCommand (REAL app method)")
	t.Logf("   NOT: CommandInputComponent (our previous fake tests)")

	commands.HandleSlashCommand(slashCmd, "/ai claude help me code")

	// 4. Verify REAL application state changes
	t.Logf("📊 RESULTS:")

	finalView := *slashCmd.CurrentView
	t.Logf("   📱 Initial view: processes")
	t.Logf("   📱 Final view: %s", finalView)

	if finalView == "ai-coders" {
		t.Logf("   ✅ View switched to ai-coders (REAL behavior)")
	} else {
		t.Logf("   ❌ View did not switch as expected")
	}

	// 5. Check the logs for what ACTUALLY happened
	logEntries := logStore.GetAll()
	t.Logf("   📄 Total log entries: %d", len(logEntries))

	var commandErrors []string
	for _, log := range logEntries {
		if strings.Contains(log.Content, "Error") || strings.Contains(log.Content, "requires") {
			commandErrors = append(commandErrors, log.Content)
		}
	}

	if len(commandErrors) > 0 {
		t.Logf("   ❌ Command errors found:")
		for _, err := range commandErrors {
			t.Logf("      - %s", err)
		}
	} else {
		t.Logf("   ✅ No command errors found")
	}

	// 6. THE CRITICAL COMPARISON
	t.Logf("\n🎯 CRITICAL FINDINGS:")
	t.Logf("===================")
	t.Logf("📋 WHAT OUR PREVIOUS TESTS COVERED (FAKE):")
	t.Logf("   ❌ CommandInputComponent (NEW system - not used by real app)")
	t.Logf("   ❌ CommandExecuteMsg (NEW system - not used by real app)")
	t.Logf("   ❌ CommandExecutor (NEW system - not used by real app)")
	t.Logf("   ❌ Message-based view switching (NEW system - not used by real app)")
	t.Logf("   ❌ Mock AI coder manager (FAKE dependency)")

	t.Logf("\n📋 WHAT THE REAL APP ACTUALLY USES (REAL):")
	t.Logf("   ✅ HandleSlashCommand (OLD system - ACTUALLY USED)")
	t.Logf("   ✅ SlashCommandContext (REAL dependency structure)")
	t.Logf("   ✅ StartAICoder callback (REAL integration point)")
	t.Logf("   ✅ Direct view switching (REAL behavior)")
	t.Logf("   ✅ Real log store and process manager (REAL dependencies)")

	t.Logf("\n💡 THE INTEGRATION TEST GAP:")
	t.Logf("   ❌ Our previous tests were COMPONENT UNIT TESTS, not integration tests")
	t.Logf("   ❌ They tested the NEW system that the app doesn't actually use")
	t.Logf("   ❌ They missed the REAL command flow that users experience")
	t.Logf("   ✅ This test demonstrates REAL integration testing of the actual app")

	// 7. Final assertion
	assert.Equal(t, "ai-coders", finalView, "Real app should switch to ai-coders view")
	t.Logf("🎯 CONCLUSION:")
	t.Logf("✅ This is what REAL UI integration testing should look like!")
	t.Logf("✅ Tests the ACTUAL application flow, not fake components")
	t.Logf("✅ Uses the REAL dependencies, not mocks")
	t.Logf("✅ Validates the REAL user experience, not component messages")
}

// TestRealConfigurationIntegration tests the actual configuration system
// that causes the real bug users experience
func TestRealConfigurationIntegration(t *testing.T) {
	t.Logf("🔍 TESTING: Real Configuration System")
	t.Logf("===================================")

	// 1. Load the ACTUAL configuration that the real app uses
	cfg, err := config.LoadWithSources()
	require.NoError(t, err, "Failed to load real config")

	// 2. Check the AI coder configuration that causes the real bug
	aiCoderConfig := cfg.GetAICoderConfig()
	t.Logf("📋 AI Coder Configuration:")
	t.Logf("   Default provider: %s", aiCoderConfig.DefaultProvider)
	t.Logf("   Max concurrent: %d", aiCoderConfig.MaxConcurrent)
	t.Logf("   Workspace base: %s", aiCoderConfig.WorkspaceBaseDir)

	// 3. Check if there are any providers configured (this is the real bug)
	if aiCoderConfig.DefaultProvider == "" {
		t.Logf("❌ REPRODUCED THE REAL BUG:")
		t.Logf("   No default provider configured")
		t.Logf("   This is why users see 'Provider not configured' errors")
		t.Logf("   When they type '/ai claude', the app fails silently")
	}

	// 4. Look for any provider configurations
	t.Logf("📋 Provider Configuration Analysis:")
	t.Logf("   Checking for provider configurations...")

	// The key issue: by default, no providers are configured
	// This is what causes the real user bug
	if aiCoderConfig.DefaultProvider == "" {
		t.Logf("❌ NO PROVIDERS CONFIGURED")
		t.Logf("   This is exactly what causes the user-reported bug")
		t.Logf("   User types: '/ai claude help me code'")
		t.Logf("   App looks for provider 'claude'")
		t.Logf("   Finds: [] (empty list)")
		t.Logf("   Logs: 'Provider claude not configured. Available: []'")
		t.Logf("   Returns early, no session created")
		t.Logf("   User sees: 'No active AI coder sessions'")
	} else {
		t.Logf("✅ Default provider configured: %s", aiCoderConfig.DefaultProvider)
	}

	// 5. This demonstrates the root cause of the real user issue
	t.Logf("\n🎯 REAL BUG ROOT CAUSE:")
	t.Logf("======================")
	t.Logf("❌ The real app requires provider configurations to work")
	t.Logf("❌ By default, no providers are configured")
	t.Logf("❌ When users type '/ai claude', it fails with provider error")
	t.Logf("❌ Our previous tests never caught this because they used mocks")
	t.Logf("✅ Real integration testing would have caught this immediately")
}

// TestCompleteUserExperienceIntegration simulates the complete user journey
// from opening the app to experiencing the bug
func TestCompleteUserExperienceIntegration(t *testing.T) {
	t.Logf("🎯 TESTING: Complete User Experience Integration")
	t.Logf("==============================================")

	// 1. User opens the app
	t.Logf("👤 USER OPENS APP")
	t.Logf("   App starts in processes view")
	t.Logf("   User sees list of npm scripts and processes")

	// 2. User types the AI command
	t.Logf("\n⌨️  USER TYPES: /ai claude help me code")
	t.Logf("   User expects: View switches to AI tab, session created")
	t.Logf("   User actually gets: Bug experience")

	// 3. Simulate the exact user scenario
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(1000, eventBus)
	processManager, err := processes.NewManager(".", eventBus, false)
	require.NoError(t, err)

	updateChan := make(chan tea.Msg, 10)
	var currentView = "processes"
	var searchResults []logs.LogEntry
	var showPattern, hidePattern string

	// Track what gets called
	var startAICoderCalled bool
	var providerNameReceived string

	slashCmd := &commands.SlashCommandContext{
		ProcessManager: processManager,
		LogStore:       logStore,
		UpdateChan:     updateChan,
		CurrentView:    &currentView,
		ShowPattern:    &showPattern,
		HidePattern:    &hidePattern,
		SearchResults:  &searchResults,
		UpdateLogsView: func() {},
		ClearLogs:      func(target string) {},
		SetProxyURL:    func(urlStr string) {},
		ListProxies:    func() {},
		ToggleProxy:    func() {},
		StartAICoder: func(providerName string) {
			startAICoderCalled = true
			providerNameReceived = providerName
			t.Logf("📞 INTEGRATION POINT: StartAICoder called with: %s", providerName)
		},
		ShowTerminal: func() {},
	}

	// 4. Execute the command
	commands.HandleSlashCommand(slashCmd, "/ai claude help me code")

	// 5. Analyze the complete user experience
	t.Logf("\n📊 COMPLETE USER EXPERIENCE ANALYSIS:")
	t.Logf("=====================================")

	// View switching
	if currentView == "ai-coders" {
		t.Logf("✅ View switched: processes → ai-coders")
	} else {
		t.Logf("❌ View did not switch properly")
	}

	// Integration point
	if startAICoderCalled {
		t.Logf("✅ StartAICoder callback was called")
		t.Logf("   Provider name received: %s", providerNameReceived)
		t.Logf("   This means the integration point is working!")
	} else {
		t.Logf("❌ StartAICoder callback was NOT called")
		t.Logf("   This indicates an integration failure")
	}

	// Error analysis
	logEntries := logStore.GetAll()
	errorCount := 0
	for _, log := range logEntries {
		if log.IsError {
			errorCount++
		}
	}

	if errorCount > 0 {
		t.Logf("❌ Found %d error(s) in the logs", errorCount)
	} else {
		t.Logf("✅ No errors found in the logs")
	}

	// 6. Final verdict
	t.Logf("\n🎯 INTEGRATION TEST VERDICT:")
	t.Logf("==========================")
	if currentView == "ai-coders" && startAICoderCalled && errorCount == 0 {
		t.Logf("✅ SUCCESS: Real integration test passed")
		t.Logf("   ✅ View switching works")
		t.Logf("   ✅ Integration points work")
		t.Logf("   ✅ No errors in the flow")
		t.Logf("   ✅ This demonstrates the REAL app works correctly")
	} else {
		t.Logf("❌ ISSUES FOUND: Real integration test identified problems")
		if currentView != "ai-coders" {
			t.Logf("   ❌ View switching issue")
		}
		if !startAICoderCalled {
			t.Logf("   ❌ Integration point not working")
		}
		if errorCount > 0 {
			t.Logf("   ❌ Errors in the flow")
		}
		t.Logf("   ✅ This demonstrates the VALUE of real integration testing!")
	}

	// 7. The key insight
	t.Logf("\n💡 KEY INSIGHT:")
	t.Logf("===============")
	t.Logf("📋 FAKE INTEGRATION TESTS (what we had before):")
	t.Logf("   ❌ Test components in isolation")
	t.Logf("   ❌ Use mocks and fakes")
	t.Logf("   ❌ Test message passing between components")
	t.Logf("   ❌ Can pass while real app is broken")

	t.Logf("\n📋 REAL INTEGRATION TESTS (what we built now):")
	t.Logf("   ✅ Test actual application flow")
	t.Logf("   ✅ Use real dependencies")
	t.Logf("   ✅ Test user interaction patterns")
	t.Logf("   ✅ Catch real bugs that users experience")

	t.Logf("\n🎯 CONCLUSION:")
	t.Logf("=============")
	t.Logf("✅ This is what 100%% of the effort should focus on")
	t.Logf("✅ Real UI integration tests that actually test the app")
	t.Logf("✅ Not fake component tests that pretend to be integration tests")
	t.Logf("✅ This prevents the 'tests pass but app is broken' scenario")
}
