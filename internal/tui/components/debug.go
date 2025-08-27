// Package components provides debug utilities for runtime validation
package components

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// DebugMode provides runtime validation and debugging utilities
type DebugMode struct {
	enabled bool
	verbose bool
	mu      sync.RWMutex

	// State tracking
	stateChecks int
	violations  []string
	lastCheck   time.Time

	// Performance tracking
	updateTimes []time.Duration
	renderTimes []time.Duration
}

var (
	debugInstance *DebugMode
	debugOnce     sync.Once
)

// Debug returns the singleton debug instance
func Debug() *DebugMode {
	debugOnce.Do(func() {
		debugInstance = &DebugMode{
			enabled:    os.Getenv("BRUMMER_DEBUG") == "true",
			verbose:    os.Getenv("BRUMMER_DEBUG_VERBOSE") == "true",
			violations: []string{},
		}
	})
	return debugInstance
}

// Enabled returns whether debug mode is active
func (d *DebugMode) Enabled() bool {
	return d.enabled
}

// Verbose returns whether verbose debug output is enabled
func (d *DebugMode) Verbose() bool {
	return d.verbose
}

// CheckState performs a state validation and panics on violation
func (d *DebugMode) CheckState(component string, validator func() error) {
	if !d.enabled {
		return
	}

	d.mu.Lock()
	d.stateChecks++
	d.lastCheck = time.Now()
	d.mu.Unlock()

	if err := validator(); err != nil {
		// Get caller info for better error reporting
		_, file, line, _ := runtime.Caller(1)
		parts := strings.Split(file, "/")
		shortFile := parts[len(parts)-1]

		violation := fmt.Sprintf("[%s:%d] %s: %v", shortFile, line, component, err)

		d.mu.Lock()
		d.violations = append(d.violations, violation)
		d.mu.Unlock()

		if d.verbose {
			panic(fmt.Sprintf("State invariant violation in %s:\n%v\n\nStack trace:", component, err))
		}
	}
}

// TrackUpdate records update timing for performance analysis
func (d *DebugMode) TrackUpdate(component string, fn func()) {
	if !d.enabled {
		fn()
		return
	}

	start := time.Now()
	fn()
	duration := time.Since(start)

	d.mu.Lock()
	d.updateTimes = append(d.updateTimes, duration)
	// Keep only last 100 measurements
	if len(d.updateTimes) > 100 {
		d.updateTimes = d.updateTimes[1:]
	}
	d.mu.Unlock()

	if d.verbose && duration > 50*time.Millisecond {
		fmt.Fprintf(os.Stderr, "WARN: Slow update in %s: %v\n", component, duration)
	}
}

// TrackRender records render timing for performance analysis
func (d *DebugMode) TrackRender(component string, fn func() string) string {
	if !d.enabled {
		return fn()
	}

	start := time.Now()
	result := fn()
	duration := time.Since(start)

	d.mu.Lock()
	d.renderTimes = append(d.renderTimes, duration)
	// Keep only last 100 measurements
	if len(d.renderTimes) > 100 {
		d.renderTimes = d.renderTimes[1:]
	}
	d.mu.Unlock()

	if d.verbose && duration > 20*time.Millisecond {
		fmt.Fprintf(os.Stderr, "WARN: Slow render in %s: %v\n", component, duration)
	}

	return result
}

// GetStats returns debug statistics
func (d *DebugMode) GetStats() DebugStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := DebugStats{
		Enabled:     d.enabled,
		StateChecks: d.stateChecks,
		Violations:  len(d.violations),
		LastCheck:   d.lastCheck,
	}

	// Calculate average times
	if len(d.updateTimes) > 0 {
		var total time.Duration
		for _, t := range d.updateTimes {
			total += t
		}
		stats.AvgUpdateTime = total / time.Duration(len(d.updateTimes))
	}

	if len(d.renderTimes) > 0 {
		var total time.Duration
		for _, t := range d.renderTimes {
			total += t
		}
		stats.AvgRenderTime = total / time.Duration(len(d.renderTimes))
	}

	// Get recent violations
	if len(d.violations) > 0 {
		start := len(d.violations) - 5
		if start < 0 {
			start = 0
		}
		stats.RecentViolations = d.violations[start:]
	}

	return stats
}

// DebugStats contains debug mode statistics
type DebugStats struct {
	Enabled          bool
	StateChecks      int
	Violations       int
	LastCheck        time.Time
	AvgUpdateTime    time.Duration
	AvgRenderTime    time.Duration
	RecentViolations []string
}

// RenderOverlay creates a debug overlay for the TUI
func (d *DebugMode) RenderOverlay(width, height int) string {
	if !d.enabled {
		return ""
	}

	stats := d.GetStats()

	// Create overlay style
	overlayStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("250")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(40).
		Align(lipgloss.Right)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	warnStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))

	// Build content
	var content strings.Builder

	content.WriteString(titleStyle.Render("🔍 DEBUG MODE"))
	content.WriteString("\n\n")

	// Performance stats
	content.WriteString(fmt.Sprintf("State Checks: %d\n", stats.StateChecks))

	if stats.Violations > 0 {
		content.WriteString(errorStyle.Render(fmt.Sprintf("⚠️ Violations: %d\n", stats.Violations)))
	} else {
		content.WriteString(fmt.Sprintf("✅ Violations: 0\n"))
	}

	content.WriteString(fmt.Sprintf("Avg Update: %.1fms\n", float64(stats.AvgUpdateTime.Microseconds())/1000))
	content.WriteString(fmt.Sprintf("Avg Render: %.1fms\n", float64(stats.AvgRenderTime.Microseconds())/1000))

	// Show performance warnings
	if stats.AvgUpdateTime > 50*time.Millisecond {
		content.WriteString(warnStyle.Render("\n⚠️ Slow updates detected\n"))
	}
	if stats.AvgRenderTime > 20*time.Millisecond {
		content.WriteString(warnStyle.Render("⚠️ Slow renders detected\n"))
	}

	// Show recent violations
	if len(stats.RecentViolations) > 0 {
		content.WriteString("\n" + titleStyle.Render("Recent Issues:") + "\n")
		for _, v := range stats.RecentViolations {
			// Truncate long violations
			if len(v) > 35 {
				v = v[:32] + "..."
			}
			content.WriteString(errorStyle.Render("• " + v + "\n"))
		}
	}

	// Time since last check
	if !stats.LastCheck.IsZero() {
		since := time.Since(stats.LastCheck)
		content.WriteString(fmt.Sprintf("\nLast check: %v ago", since.Round(time.Second)))
	}

	return overlayStyle.Render(content.String())
}

// Component validation helpers

// ValidateProcessView validates ProcessViewNew state invariants
func ValidateProcessView(pv *ProcessViewNew) error {
	// Check selection bounds - allow selectedIndex == 0 when we have processes
	if len(pv.processes) == 0 {
		// No processes: selection should be cleared
		if pv.selectedIndex != -1 {
			return fmt.Errorf("selectedIndex should be -1 when no processes, got %d", pv.selectedIndex)
		}
		if pv.selectedProcess != "" {
			return fmt.Errorf("selectedProcess should be empty when no processes, got %s", pv.selectedProcess)
		}
	} else {
		// Have processes: selection should be valid
		if pv.selectedIndex < -1 || pv.selectedIndex >= len(pv.processes) {
			return fmt.Errorf("selectedIndex out of bounds: %d (valid range: -1 to %d)",
				pv.selectedIndex, len(pv.processes)-1)
		}
	}

	// Check selected process exists
	if pv.selectedProcess != "" {
		found := false
		for _, proc := range pv.processes {
			if proc.ID == pv.selectedProcess {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("selectedProcess '%s' not in process list", pv.selectedProcess)
		}
	}

	// Check scroll offset
	if pv.scrollOffset < 0 {
		return fmt.Errorf("negative scrollOffset: %d", pv.scrollOffset)
	}

	maxScroll := len(pv.processes) - pv.visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if pv.scrollOffset > maxScroll {
		return fmt.Errorf("scrollOffset %d exceeds max %d", pv.scrollOffset, maxScroll)
	}

	// Check that selected item is visible if we have items
	if len(pv.processes) > 0 && pv.selectedIndex >= 0 {
		if pv.selectedIndex < pv.scrollOffset ||
			pv.selectedIndex >= pv.scrollOffset+pv.visibleLines {
			return fmt.Errorf("selected item %d not visible (scroll: %d, visible: %d)",
				pv.selectedIndex, pv.scrollOffset, pv.visibleLines)
		}
	}

	return nil
}

// ValidateLogsView validates LogsViewComponent state invariants
func ValidateLogsView(lv *LogsViewComponent) error {
	// No longer checking against logEntries since we fetch from store

	// Check viewport dimensions
	if lv.viewport.Width <= 0 {
		return fmt.Errorf("invalid viewport width: %d", lv.viewport.Width)
	}
	if lv.viewport.Height <= 0 {
		return fmt.Errorf("invalid viewport height: %d", lv.viewport.Height)
	}

	// Check dimensions match
	if lv.width > 0 && lv.viewport.Width > lv.width {
		return fmt.Errorf("viewport width %d > component width %d",
			lv.viewport.Width, lv.width)
	}

	return nil
}

// ValidateShell validates ProcessLogsShell state invariants
func ValidateShell(s *ProcessLogsShell) error {
	// Check tab index
	if s.activeTab < 0 || s.activeTab >= len(s.tabs) {
		return fmt.Errorf("activeTab out of bounds: %d (tabs: %d)",
			s.activeTab, len(s.tabs))
	}

	// Check components exist
	if s.processView == nil {
		return fmt.Errorf("processView is nil")
	}
	if s.logsView == nil {
		return fmt.Errorf("logsView is nil")
	}

	// Check dependencies
	if s.processMgr == nil {
		return fmt.Errorf("processMgr is nil")
	}
	if s.logStore == nil {
		return fmt.Errorf("logStore is nil")
	}

	// Check dimensions
	if s.width <= 0 {
		return fmt.Errorf("invalid width: %d", s.width)
	}
	if s.height <= 0 {
		return fmt.Errorf("invalid height: %d", s.height)
	}

	return nil
}
