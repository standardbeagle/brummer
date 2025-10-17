package components

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/test"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// TestViewComponent displays test results and manages test execution
// It receives test-related messages and updates the display accordingly.
//
// Design principles:
// - Read-only access to test manager for initial load
// - All updates via test event messages
// - Compact view with focus on failures and summary
// - Keyboard shortcuts for test execution (r, R, c)
type TestViewComponent struct {
	BaseComponent

	viewport     viewport.Model
	testMgr      *test.TestManager
	testExecutor *test.TestExecutor
	activeRuns   []*test.TestRun
	selectedRun  string
	selectedTab  TestTab

	// Cache for test results
	results    map[string]*test.TestResults
	lastUpdate time.Time

	// Current execution state
	isRunning  bool
	currentRun *test.TestRun

	// Styles
	headerStyle   lipgloss.Style
	successStyle  lipgloss.Style
	failureStyle  lipgloss.Style
	skippedStyle  lipgloss.Style
	runningStyle  lipgloss.Style
	normalStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	borderStyle   lipgloss.Style
}

// TestTab represents the active tab in the test view
type TestTab int

const (
	TestTabSummary TestTab = iota
	TestTabResults
	TestTabCoverage
	TestTabPerformance
)

var testTabNames = []string{"Summary", "Results", "Coverage", "Performance"}

// NewTestViewComponent creates a test view with initial data from test manager
func NewTestViewComponent(testMgr *test.TestManager, width, height int) *TestViewComponent {
	vp := viewport.New(width-4, height-6) // Account for borders and header
	vp.Style = lipgloss.NewStyle().Padding(0, 1)

	tv := &TestViewComponent{
		BaseComponent: BaseComponent{
			visible: true,
			width:   width,
			height:  height,
		},
		viewport:     vp,
		testMgr:      testMgr,
		testExecutor: test.NewTestExecutor(),
		results:      make(map[string]*test.TestResults),
		selectedTab:  TestTabSummary,
		isRunning:    false,
	}

	// Initialize styles
	tv.initStyles()

	// Load initial data
	if testMgr != nil {
		tv.activeRuns = testMgr.GetActiveRuns()
	}

	return tv
}

// initStyles initializes the styling for the test view
func (tv *TestViewComponent) initStyles() {
	tv.headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		MarginBottom(1)

	tv.successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4ADE80"))

	tv.failureStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F87171"))

	tv.skippedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FBBF24"))

	tv.runningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#60A5FA"))

	tv.normalStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB"))

	tv.selectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#374151")).
		Foreground(lipgloss.Color("#FFFFFF"))

	tv.borderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4B5563"))
}

// Init initializes the test view component
func (tv *TestViewComponent) Init() tea.Cmd {
	return nil
}

// Update handles updates to the test view component
func (tv *TestViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return tv.handleKeyMsg(msg)

	case messages.TestRunStartedMsg:
		tv.handleTestRunStarted(msg)
		return tv, nil

	case messages.TestRunCompletedMsg:
		tv.handleTestRunCompleted(msg)
		return tv, nil

	case messages.TestCaseStartedMsg:
		tv.handleTestCaseStarted(msg)
		return tv, nil

	case messages.TestCaseCompletedMsg:
		tv.handleTestCaseCompleted(msg)
		return tv, nil

	case messages.TestResultsClearedMsg:
		tv.handleTestResultsCleared(msg)
		return tv, nil

	case messages.TestCoverageUpdatedMsg:
		tv.handleTestCoverageUpdated(msg)
		return tv, nil

	case messages.TestPerformanceUpdatedMsg:
		tv.handleTestPerformanceUpdated(msg)
		return tv, nil
	}

	// Handle viewport updates
	var cmd tea.Cmd
	tv.viewport, cmd = tv.viewport.Update(msg)
	return tv, cmd
}

// handleKeyMsg handles keyboard input for the test view
func (tv *TestViewComponent) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Toggle selection if we have runs
		if len(tv.activeRuns) > 0 {
			if tv.selectedRun == "" {
				tv.selectedRun = tv.activeRuns[0].ID
			} else {
				tv.selectedRun = ""
			}
		}

	case tea.KeyUp:
		// Move selection up
		tv.moveSelection(-1)

	case tea.KeyDown:
		// Move selection down
		tv.moveSelection(1)

	case tea.KeyTab:
		// Switch tabs
		tv.selectedTab = (tv.selectedTab + 1) % TestTab(len(testTabNames))

	case tea.KeyShiftTab:
		// Switch tabs backwards
		tv.selectedTab = (tv.selectedTab - 1 + TestTab(len(testTabNames))) % TestTab(len(testTabNames))

	case 'r':
		// Run all tests
		if !tv.isRunning {
			tv.isRunning = true

			// Create a new test run
			run := &test.TestRun{
				ID:        fmt.Sprintf("run-%d", time.Now().Unix()),
				Framework: "auto",
				Pattern:   "",
				StartTime: time.Now(),
				Status:    test.TestRunStatusRunning,
				Options: test.TestOptions{
					Verbose: true,
					Timeout: 5 * time.Minute,
				},
			}

			tv.currentRun = run
			tv.activeRuns = append(tv.activeRuns, run)
			tv.lastUpdate = time.Now()

			// Execute tests using command-driven approach
			return tv, tv.executeTestsCommand(run)
		}

	case 'R':
		// Run specific test/file
		return tv, tea.Cmd(func() tea.Msg {
			// TODO: Show input prompt for test pattern
			return nil
		})

	case 'c':
		// Clear test results
		tv.handleTestResultsCleared(messages.TestResultsClearedMsg{})
		return tv, nil
	}

	return tv, nil
}

// moveSelection moves the selection up or down
func (tv *TestViewComponent) moveSelection(direction int) {
	if len(tv.activeRuns) == 0 {
		return
	}

	currentIndex := -1
	if tv.selectedRun != "" {
		for i, run := range tv.activeRuns {
			if run.ID == tv.selectedRun {
				currentIndex = i
				break
			}
		}
	}

	newIndex := currentIndex + direction
	if newIndex < 0 {
		newIndex = len(tv.activeRuns) - 1
	} else if newIndex >= len(tv.activeRuns) {
		newIndex = 0
	}

	tv.selectedRun = tv.activeRuns[newIndex].ID
}

// handleTestRunStarted handles test run started messages
func (tv *TestViewComponent) handleTestRunStarted(msg messages.TestRunStartedMsg) {
	// Refresh active runs from test manager
	if tv.testMgr != nil {
		tv.activeRuns = tv.testMgr.GetActiveRuns()
	}
	tv.lastUpdate = time.Now()
}

// handleTestRunCompleted handles test run completed messages
func (tv *TestViewComponent) handleTestRunCompleted(msg messages.TestRunCompletedMsg) {
	// This would typically be handled by updating the test manager
	// and then refreshing our view
	if tv.testMgr != nil {
		tv.activeRuns = tv.testMgr.GetActiveRuns()
	}
	tv.lastUpdate = time.Now()
}

// handleTestCaseStarted handles test case started messages
func (tv *TestViewComponent) handleTestCaseStarted(msg messages.TestCaseStartedMsg) {
	tv.lastUpdate = time.Now()
}

// handleTestCaseCompleted handles test case completed messages
func (tv *TestViewComponent) handleTestCaseCompleted(msg messages.TestCaseCompletedMsg) {
	tv.lastUpdate = time.Now()
}

// handleTestResultsCleared handles test results cleared messages
func (tv *TestViewComponent) handleTestResultsCleared(msg messages.TestResultsClearedMsg) {
	if msg.RunID == "" {
		tv.results = make(map[string]*test.TestResults)
	} else {
		delete(tv.results, msg.RunID)
	}
	tv.lastUpdate = time.Now()
}

// handleTestCoverageUpdated handles test coverage updated messages
func (tv *TestViewComponent) handleTestCoverageUpdated(msg messages.TestCoverageUpdatedMsg) {
	tv.lastUpdate = time.Now()
}

// handleTestPerformanceUpdated handles test performance updated messages
func (tv *TestViewComponent) handleTestPerformanceUpdated(msg messages.TestPerformanceUpdatedMsg) {
	tv.lastUpdate = time.Now()
}

// View renders the test view component
func (tv *TestViewComponent) View() string {
	if !tv.visible {
		return ""
	}

	var content strings.Builder

	// Header with tabs
	content.WriteString(tv.renderHeader())

	// Main content based on selected tab
	switch tv.selectedTab {
	case TestTabSummary:
		content.WriteString(tv.renderSummaryView())
	case TestTabResults:
		content.WriteString(tv.renderResultsView())
	case TestTabCoverage:
		content.WriteString(tv.renderCoverageView())
	case TestTabPerformance:
		content.WriteString(tv.renderPerformanceView())
	}

	// Footer with keyboard shortcuts
	content.WriteString("\n")
	content.WriteString(tv.renderFooter())

	return tv.borderStyle.Render(content.String())
}

// renderHeader renders the header with tabs
func (tv *TestViewComponent) renderHeader() string {
	var tabs strings.Builder

	for i, tabName := range testTabNames {
		style := tv.normalStyle
		if i == int(tv.selectedTab) {
			style = tv.selectedStyle.Copy().Background(lipgloss.Color("#7D56F4"))
		}
		tabs.WriteString(style.Render(tabName))
		if i < len(testTabNames)-1 {
			tabs.WriteString(" ")
		}
	}

	header := tv.headerStyle.Render("Test Management")
	return header + "\n" + tabs.String()
}

// renderSummaryView renders the summary tab
func (tv *TestViewComponent) renderSummaryView() string {
	if len(tv.activeRuns) == 0 && !tv.isRunning {
		return tv.normalStyle.Render("No test runs found. Press 'r' to run tests.")
	}

	var summary strings.Builder
	summary.WriteString("\n")

	// Show running state if active
	if tv.isRunning && tv.currentRun != nil {
		summary.WriteString(tv.runningStyle.Render(fmt.Sprintf(
			"  ► Running %s tests... (%.1fs)\n",
			tv.currentRun.Framework,
			time.Since(tv.currentRun.StartTime).Seconds(),
		)))
		summary.WriteString("\n")
	}

	// Show recent test runs
	for i, run := range tv.activeRuns {
		if i >= 5 { // Show only recent 5 runs
			break
		}

		style := tv.normalStyle
		if run.ID == tv.selectedRun {
			style = tv.selectedStyle
		}

		statusText := tv.getRunStatusText(run)
		duration := tv.formatDuration(run.Duration)

		// Show special styling for currently running test
		if run.Status == test.TestRunStatusRunning {
			style = tv.runningStyle
			statusText = fmt.Sprintf("Running... (%.1fs)", time.Since(run.StartTime).Seconds())
		}

		line := fmt.Sprintf("  %s %s - %s (%s)\n",
			style.Render("•"),
			style.Render(run.Framework),
			style.Render(statusText),
			style.Render(duration),
		)
		summary.WriteString(line)
	}

	// Show current results if available
	if tv.selectedRun != "" {
		if results, exists := tv.results[tv.selectedRun]; exists {
			summary.WriteString("\n")
			summary.WriteString(tv.normalStyle.Render(fmt.Sprintf(
				"Results: %d total, %s%d%s passed, %s%d%s failed, %s%d%s skipped\n",
				results.TotalTests,
				tv.successStyle.Render(""), results.PassedTests, tv.successStyle.Render(""),
				tv.failureStyle.Render(""), results.FailedTests, tv.failureStyle.Render(""),
				tv.skippedStyle.Render(""), results.SkippedTests, tv.skippedStyle.Render(""),
			)))
		}
	}

	return summary.String()
}

// renderResultsView renders the detailed results tab
func (tv *TestViewComponent) renderResultsView() string {
	if tv.selectedRun == "" {
		return tv.normalStyle.Render("\nSelect a test run to view detailed results.")
	}

	results, exists := tv.results[tv.selectedRun]
	if !exists {
		return tv.normalStyle.Render("\nNo results available for selected run.")
	}

	var content strings.Builder
	content.WriteString("\n")

	// Summary line
	content.WriteString(tv.normalStyle.Render(fmt.Sprintf(
		"Total: %d | Passed: %s%d%s | Failed: %s%d%s | Skipped: %s%d%s\n",
		results.TotalTests,
		tv.successStyle.Render(""), results.PassedTests, tv.successStyle.Render(""),
		tv.failureStyle.Render(""), results.FailedTests, tv.failureStyle.Render(""),
		tv.skippedStyle.Render(""), results.SkippedTests, tv.skippedStyle.Render(""),
	)))

	// Show failures first
	if len(results.TestCases) > 0 {
		content.WriteString("\n")
		content.WriteString(tv.normalStyle.Render("Test Cases:\n"))

		for _, tc := range results.TestCases {
			if len(content.String()) > 2000 { // Limit output
				break
			}

			statusStyle := tv.normalStyle
			statusChar := "✓"
			switch tc.Status {
			case test.TestCaseStatusPassed:
				statusStyle = tv.successStyle
				statusChar = "✓"
			case test.TestCaseStatusFailed:
				statusStyle = tv.failureStyle
				statusChar = "✗"
			case test.TestCaseStatusSkipped:
				statusStyle = tv.skippedStyle
				statusChar = "⊘"
			}

			line := fmt.Sprintf("  %s %s - %s\n",
				statusStyle.Render(statusChar),
				tc.Name,
				tv.formatDuration(tc.Duration),
			)
			content.WriteString(line)

			// Show failure info if available and compact display
			if tc.Status == test.TestCaseStatusFailed && tc.FailureInfo != nil {
				failureLine := fmt.Sprintf("    %s\n", tv.failureStyle.Render(tc.FailureInfo.Message))
				content.WriteString(failureLine)
			}
		}
	}

	return content.String()
}

// renderCoverageView renders the coverage tab
func (tv *TestViewComponent) renderCoverageView() string {
	if tv.selectedRun == "" {
		return tv.normalStyle.Render("\nSelect a test run to view coverage.")
	}

	results, exists := tv.results[tv.selectedRun]
	if !exists || results.Coverage == nil {
		return tv.normalStyle.Render("\nNo coverage data available for selected run.")
	}

	var content strings.Builder
	content.WriteString("\n")

	coverage := results.Coverage
	coverageStyle := tv.normalStyle
	if coverage.OverallPct >= 80 {
		coverageStyle = tv.successStyle
	} else if coverage.OverallPct >= 60 {
		coverageStyle = tv.skippedStyle
	} else {
		coverageStyle = tv.failureStyle
	}

	content.WriteString(tv.normalStyle.Render(fmt.Sprintf(
		"Overall Coverage: %s%.1f%%%s (%d/%d lines)\n",
		coverageStyle.Render(""),
		coverage.OverallPct,
		coverageStyle.Render(""),
		coverage.LineCoverage,
		coverage.TotalLines,
	)))

	if len(coverage.Files) > 0 {
		content.WriteString("\n")
		content.WriteString(tv.normalStyle.Render("File Coverage:\n"))

		for _, file := range coverage.Files {
			if len(content.String()) > 1500 { // Limit output
				break
			}

			fileStyle := tv.normalStyle
			if file.CoveragePct >= 80 {
				fileStyle = tv.successStyle
			} else if file.CoveragePct >= 60 {
				fileStyle = tv.skippedStyle
			} else {
				fileStyle = tv.failureStyle
			}

			line := fmt.Sprintf("  %s %s (%.1f%%)\n",
				fileStyle.Render(file.FilePath),
				tv.normalStyle.Render(fmt.Sprintf("%d/%d lines", file.LinesCovered, file.TotalLines)),
				file.CoveragePct,
			)
			content.WriteString(line)
		}
	}

	return content.String()
}

// renderPerformanceView renders the performance tab
func (tv *TestViewComponent) renderPerformanceView() string {
	if tv.selectedRun == "" {
		return tv.normalStyle.Render("\nSelect a test run to view performance.")
	}

	results, exists := tv.results[tv.selectedRun]
	if !exists || results.Performance == nil {
		return tv.normalStyle.Render("\nNo performance data available for selected run.")
	}

	var content strings.Builder
	content.WriteString("\n")

	perf := results.Performance
	content.WriteString(tv.normalStyle.Render(fmt.Sprintf(
		"Total Duration: %s\n",
		tv.formatDuration(perf.TotalDuration),
	)))
	content.WriteString(tv.normalStyle.Render(fmt.Sprintf(
		"Average Duration: %s\n",
		tv.formatDuration(perf.AverageDuration),
	)))

	if len(perf.SlowTests) > 0 {
		content.WriteString("\n")
		content.WriteString(tv.failureStyle.Render(fmt.Sprintf(
			"Slow Tests (> %s):\n",
			tv.formatDuration(perf.SlowThreshold),
		)))

		for _, slowTest := range perf.SlowTests {
			if len(content.String()) > 1500 { // Limit output
				break
			}

			line := fmt.Sprintf("  %s - %s\n",
				slowTest.TestName,
				tv.formatDuration(slowTest.Duration),
			)
			content.WriteString(line)
		}
	}

	return content.String()
}

// renderFooter renders the footer with keyboard shortcuts
func (tv *TestViewComponent) renderFooter() string {
	shortcuts := "[r] Run all tests | [R] Run specific | [c] Clear | [Tab] Switch tabs | [↑/↓] Navigate | [Enter] Select"
	return tv.normalStyle.Render(shortcuts)
}

// getRunStatusText returns formatted status text for a test run
func (tv *TestViewComponent) getRunStatusText(run *test.TestRun) string {
	switch run.Status {
	case test.TestRunStatusPending:
		return "Pending"
	case test.TestRunStatusRunning:
		return "Running"
	case test.TestRunStatusCompleted:
		if run.Results != nil {
			return fmt.Sprintf("%d passed, %d failed", run.Results.PassedTests, run.Results.FailedTests)
		}
		return "Completed"
	case test.TestRunStatusFailed:
		return "Failed"
	case test.TestRunStatusCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// formatDuration formats a duration for display
func (tv *TestViewComponent) formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1000000.0)
	} else if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1000000.0)
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
}

// executeTestsCommand creates a command to execute tests without goroutines
func (tv *TestViewComponent) executeTestsCommand(run *test.TestRun) tea.Cmd {
	return func() tea.Msg {
		// NO MORE GOROUTINES - Test execution will be command-driven
		// through BubbleTea Update() cycle instead of background goroutine

		// Detect available frameworks
		detected := tv.testExecutor.DetectAvailableFrameworks()
		var frameworkName string

		if len(detected) == 0 {
			// Fallback to Node.js test runner
			frameworkName = "node"
		} else {
			frameworkName = detected[0].Name()
		}

		// Update run with detected framework
		run.Framework = frameworkName

		// Execute tests synchronously in the command
		results, err := tv.testExecutor.ExecuteTests(
			context.Background(),
			frameworkName,
			"",
			test.TestOptions{Verbose: true},
		)

		// Update run status and store results
		endTime := time.Now()
		run.EndTime = &endTime
		run.Duration = endTime.Sub(run.StartTime)

		if err != nil {
			run.Status = test.TestRunStatusFailed
			fmt.Printf("Test execution error: %v\n", err)
		} else {
			run.Status = test.TestRunStatusCompleted
			run.Results = results
		}

		// Return a message to update the UI
		msg := messages.TestRunCompletedMsg{
			RunID:   run.ID,
		}
		if results != nil {
			msg.TotalTests = results.TotalTests
			msg.PassedTests = results.PassedTests
			msg.FailedTests = results.FailedTests
			msg.SkippedTests = results.SkippedTests
		}
		return msg
	}
}

// SetSize updates the component size
func (tv *TestViewComponent) SetSize(width, height int) {
	tv.width = width
	tv.height = height
	tv.viewport.Width = width - 4
	tv.viewport.Height = height - 6
}
