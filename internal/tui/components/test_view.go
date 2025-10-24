package components

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/state"
)

// TestViewComponentNew is a stateless BubbleTea component for displaying test results and managing test execution
// Architecture: Stateless rendering with external state (Redux/Elm compliant)
type TestViewComponentNew struct {
	BaseComponent

	// UI configuration only (no state)
	viewport viewport.Model

	// Styles
	headerStyle    lipgloss.Style
	successStyle   lipgloss.Style
	failureStyle   lipgloss.Style
	skippedStyle   lipgloss.Style
	runningStyle   lipgloss.Style
	normalStyle    lipgloss.Style
	selectedStyle  lipgloss.Style
	borderStyle    lipgloss.Style
	tabStyle       lipgloss.Style
	activeTabStyle lipgloss.Style
	statsStyle     lipgloss.Style
}

// Test tab names
var testTabNames = []string{"Summary", "Results", "Coverage", "Performance"}

// NewTestViewComponentNew creates a new stateless test view component
func NewTestViewComponentNew() *TestViewComponentNew {
	vp := viewport.New(80, 24)
	vp.Style = lipgloss.NewStyle().Padding(0, 1)

	return &TestViewComponentNew{
		BaseComponent: NewBaseComponent(),
		viewport:      vp,

		// Initialize styles
		headerStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).  // Bright Blue
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1),

		successStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("10")), // Bright Green
		failureStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("9")),  // Bright Red
		skippedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("11")), // Bright Yellow
		runningStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("14")), // Bright Cyan
		normalStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("15")), // White

		selectedStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("15")).

			Bold(true),

		borderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")), // Gray

		tabStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Padding(0, 2),

		activeTabStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("12")).
			Bold(true).
			Padding(0, 2),

		statsStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")), // Gray
	}
}

// ViewWithState renders the test view using external state (Redux/Elm compliant)
func (c *TestViewComponentNew) ViewWithState(masterData *state.MasterData) string {
	if !c.IsVisible() {
		return ""
	}

	if masterData == nil || masterData.TestViewState == nil {
		return c.renderErrorState("Test view state not available")
	}

	testViewState := masterData.TestViewState

	if testViewState.GetLoading() {
		return c.renderLoadingState()
	}

	if error := testViewState.GetError(); error != "" {
		return c.renderErrorState(error)
	}

	// Update viewport with current state
	c.updateViewportFromState(testViewState)

	// Render based on selected tab
	switch testViewState.GetSelectedTab() {
	case state.TestTabSummary:
		return c.renderSummaryTab(testViewState)
	case state.TestTabResults:
		return c.renderResultsTab(testViewState)
	case state.TestTabCoverage:
		return c.renderCoverageTab(testViewState)
	case state.TestTabPerformance:
		return c.renderPerformanceTab(testViewState)
	default:
		return c.renderSummaryTab(testViewState)
	}
}

// renderSummaryTab renders the summary tab
func (c *TestViewComponentNew) renderSummaryTab(testViewState *state.TestViewState) string {
	var content strings.Builder

	// Header with tabs
	content.WriteString(c.renderTabs(testViewState))
	content.WriteString("\n")

	// Statistics overview
	activeRuns := testViewState.GetActiveRuns()
	runsByStatus := testViewState.RunsByStatus

	stats := []string{
		fmt.Sprintf("Total Runs: %d", len(activeRuns)),
	}

	// Sort status keys for stable display order (prevent bouncing)
	statusKeys := make([]string, 0, len(runsByStatus))
	for status := range runsByStatus {
		statusKeys = append(statusKeys, status)
	}
	sort.Strings(statusKeys)
	for _, status := range statusKeys {
		stats = append(stats, fmt.Sprintf("%s: %d", status, runsByStatus[status]))
	}

	if testViewState.IsRunning() {
		stats = append(stats, "Status: Running")
	}

	content.WriteString(c.headerStyle.Render("Test Summary"))
	content.WriteString("\n")
	content.WriteString(strings.Join(stats, " | "))
	content.WriteString("\n\n")

	// Recent runs
	if len(activeRuns) > 0 {
		content.WriteString(c.headerStyle.Render("Recent Test Runs"))
		content.WriteString("\n")
		content.WriteString(strings.Repeat("─", 80))
		content.WriteString("\n")

		for i, run := range activeRuns {
			if i >= 10 { // Show only 10 most recent
				break
			}

			statusStyle := c.getStatusStyle(run.Status)
			runLine := fmt.Sprintf("%s %s - %s (%s) - %d/%d passed",
				statusStyle.Render(c.getStatusIcon(run.Status)),
				run.Name,
				c.formatDuration(run.Duration),
				run.Status,
				run.PassedTests,
				run.TotalTests,
			)

			if run.ID == testViewState.GetSelectedRunID() {
				runLine = c.selectedStyle.Render(runLine)
			}

			content.WriteString(runLine)
			content.WriteString("\n")
		}
	} else {
		content.WriteString(c.statsStyle.Render("No test runs yet. Press 'r' to run tests."))
	}

	// Help text
	content.WriteString("\n\n")
	content.WriteString(c.statsStyle.Render("Commands: r=Run tests, R=Run with coverage, c=Clear results, 1-4=Switch tabs, ↑↓=Select run"))

	// Set viewport content
	c.viewport.SetContent(content.String())
	return c.viewport.View()
}

// renderResultsTab renders the detailed results tab
func (c *TestViewComponentNew) renderResultsTab(testViewState *state.TestViewState) string {
	var content strings.Builder

	// Header with tabs
	content.WriteString(c.renderTabs(testViewState))
	content.WriteString("\n")

	selectedRun := testViewState.GetSelectedRun()
	if selectedRun == nil {
		content.WriteString(c.statsStyle.Render("Select a test run to view results"))
		c.viewport.SetContent(content.String())
		return c.viewport.View()
	}

	// Run details
	content.WriteString(c.headerStyle.Render(fmt.Sprintf("Test Results: %s", selectedRun.Name)))
	content.WriteString("\n")

	// Filter status
	filterStatus := testViewState.GetFilterStatus()
	if filterStatus != "all" {
		content.WriteString(fmt.Sprintf("Filter: %s", filterStatus))
		content.WriteString("\n\n")
	}

	// Test results
	results := testViewState.GetFilteredResults(selectedRun.ID)
	if len(results) == 0 {
		content.WriteString(c.statsStyle.Render("No test results to display"))
	} else {
		for _, result := range results {
			statusStyle := c.getStatusStyle(result.Status)
			resultLine := fmt.Sprintf("%s %s - %s",
				statusStyle.Render(c.getStatusIcon(result.Status)),
				result.Name,
				c.formatDuration(result.Duration),
			)

			if result.ErrorMessage != "" {
				resultLine += fmt.Sprintf(" - %s", result.ErrorMessage)
			}

			content.WriteString(resultLine)
			content.WriteString("\n")
		}
	}

	// Set viewport content
	c.viewport.SetContent(content.String())
	return c.viewport.View()
}

// renderCoverageTab renders the coverage tab
func (c *TestViewComponentNew) renderCoverageTab(testViewState *state.TestViewState) string {
	var content strings.Builder

	// Header with tabs
	content.WriteString(c.renderTabs(testViewState))
	content.WriteString("\n")

	content.WriteString(c.headerStyle.Render("Code Coverage"))
	content.WriteString("\n")

	selectedRun := testViewState.GetSelectedRun()
	if selectedRun == nil {
		content.WriteString(c.statsStyle.Render("Select a test run to view coverage"))
	} else if selectedRun.Coverage > 0 {
		content.WriteString(fmt.Sprintf("Coverage: %.1f%%\n", selectedRun.Coverage))
		content.WriteString(c.renderCoverageBar(selectedRun.Coverage))
	} else {
		content.WriteString(c.statsStyle.Render("No coverage data available. Run tests with coverage enabled (press 'R')."))
	}

	// Set viewport content
	c.viewport.SetContent(content.String())
	return c.viewport.View()
}

// renderPerformanceTab renders the performance tab
func (c *TestViewComponentNew) renderPerformanceTab(testViewState *state.TestViewState) string {
	var content strings.Builder

	// Header with tabs
	content.WriteString(c.renderTabs(testViewState))
	content.WriteString("\n")

	content.WriteString(c.headerStyle.Render("Test Performance"))
	content.WriteString("\n")

	selectedRun := testViewState.GetSelectedRun()
	if selectedRun == nil {
		content.WriteString(c.statsStyle.Render("Select a test run to view performance"))
	} else {
		content.WriteString(fmt.Sprintf("Total Duration: %s\n", c.formatDuration(selectedRun.Duration)))

		if selectedRun.TotalTests > 0 {
			avgDuration := selectedRun.Duration / time.Duration(selectedRun.TotalTests)
			content.WriteString(fmt.Sprintf("Average Test Duration: %s\n", c.formatDuration(avgDuration)))
		}

		// Add more performance metrics here as needed
	}

	// Set viewport content
	c.viewport.SetContent(content.String())
	return c.viewport.View()
}

// renderTabs renders the tab navigation
func (c *TestViewComponentNew) renderTabs(testViewState *state.TestViewState) string {
	var tabs []string

	for i, tabName := range testTabNames {
		if state.TestTab(i) == testViewState.GetSelectedTab() {
			tabs = append(tabs, c.activeTabStyle.Render(tabName))
		} else {
			tabs = append(tabs, c.tabStyle.Render(tabName))
		}
	}

	return strings.Join(tabs, " ")
}

// renderCoverageBar renders a visual coverage bar
func (c *TestViewComponentNew) renderCoverageBar(coverage float64) string {
	width := 50
	filled := int(coverage * float64(width) / 100.0)

	var bar strings.Builder
	bar.WriteString("[")

	for i := 0; i < width; i++ {
		if i < filled {
			if coverage >= 80 {
				bar.WriteString("█") // High coverage - green would be ideal
			} else if coverage >= 60 {
				bar.WriteString("▓") // Medium coverage
			} else {
				bar.WriteString("░") // Low coverage
			}
		} else {
			bar.WriteString(" ")
		}
	}

	bar.WriteString(fmt.Sprintf("] %.1f%%", coverage))

	// Color based on coverage level
	var style lipgloss.Style
	if coverage >= 80 {
		style = c.successStyle
	} else if coverage >= 60 {
		style = c.runningStyle
	} else {
		style = c.failureStyle
	}

	return style.Render(bar.String())
}

// renderLoadingState renders the loading state
func (c *TestViewComponentNew) renderLoadingState() string {
	content := c.headerStyle.Render("Test View") + "\n"
	content += strings.Repeat("─", 80) + "\n"
	content += c.statsStyle.Render("Loading test data...")

	c.viewport.SetContent(content)
	return c.viewport.View()
}

// renderErrorState renders the error state
func (c *TestViewComponentNew) renderErrorState(message string) string {
	content := c.headerStyle.Render("Test View") + "\n"
	content += strings.Repeat("─", 80) + "\n"
	content += c.failureStyle.Render(fmt.Sprintf("Error: %s", message))

	c.viewport.SetContent(content)
	return c.viewport.View()
}

// getStatusStyle returns the appropriate style for a status
func (c *TestViewComponentNew) getStatusStyle(status string) lipgloss.Style {
	switch status {
	case "passed", "completed":
		return c.successStyle
	case "failed", "error":
		return c.failureStyle
	case "skipped":
		return c.skippedStyle
	case "running":
		return c.runningStyle
	default:
		return c.normalStyle
	}
}

// getStatusIcon returns the appropriate icon for a status
func (c *TestViewComponentNew) getStatusIcon(status string) string {
	switch status {
	case "passed", "completed":
		return "✓"
	case "failed", "error":
		return "✗"
	case "skipped":
		return "⊘"
	case "running":
		return "⟳"
	case "cancelled":
		return "⏹"
	default:
		return "?"
	}
}

// formatDuration formats a duration for display
func (c *TestViewComponentNew) formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.0fμs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1000000)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// updateViewportFromState updates the viewport based on current state
func (c *TestViewComponentNew) updateViewportFromState(testViewState *state.TestViewState) {
	width, height := testViewState.GetSize()
	c.viewport.Width = width
	c.viewport.Height = height - 6 // Account for header and tabs

	// Update viewport offset from state
	offset := testViewState.GetViewportOffset()
	if offset >= 0 && offset < c.viewport.TotalLineCount() {
		c.viewport.YOffset = offset
	}
}

// SetSize updates component dimensions
func (c *TestViewComponentNew) SetSize(width, height int) {
	c.viewport.Width = width
	c.viewport.Height = height - 6 // Account for header and tabs
}

// UpdateSize updates dimensions with pre-calculated content height
func (c *TestViewComponentNew) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	c.viewport.Width = width
	c.viewport.Height = contentHeight
}

// View renders the component (legacy method for compatibility)
func (c *TestViewComponentNew) View() string {
	if !c.IsVisible() {
		return ""
	}
	return "" // Rendering is done via ViewWithState
}

// Update handles messages (minimal - stateless component)
func (c *TestViewComponentNew) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil
	}

	// Update viewport
	c.viewport, cmd = c.viewport.Update(msg)
	return c, cmd
}

// Init initializes the component
func (c *TestViewComponentNew) Init() tea.Cmd {
	return nil
}