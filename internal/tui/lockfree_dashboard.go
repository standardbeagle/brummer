package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// LockFreeDashboard provides a performance monitoring view for lock-free operations
type LockFreeDashboard struct {
	width  int
	height int
}

// NewLockFreeDashboard creates a new dashboard
func NewLockFreeDashboard() *LockFreeDashboard {
	return &LockFreeDashboard{}
}

// UpdateSize updates the dashboard dimensions
func (d *LockFreeDashboard) UpdateSize(width, height int) {
	d.width = width
	d.height = height
}

// RenderMetrics renders the lock-free performance metrics
func (d *LockFreeDashboard) RenderMetrics(managerMetrics PTYManagerMetrics, bufferMetrics BufferMetrics) string {
	if d.width < 60 || d.height < 20 {
		return "Terminal too small for lock-free dashboard"
	}

	var sections []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")). // Bright blue
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(0, 2).
		Width(d.width - 4)

	header := headerStyle.Render("🚀 Lock-Free PTY Performance Dashboard")
	sections = append(sections, header)

	// Create two-column layout
	leftColumn := d.renderBufferMetrics(bufferMetrics)
	rightColumn := d.renderManagerMetrics(managerMetrics)

	columnsStyle := lipgloss.NewStyle().
		Width(d.width - 2).
		Height(d.height - 8) // Leave room for header and footer

	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		rightColumn,
	)

	sections = append(sections, columnsStyle.Render(columns))

	// Performance summary
	summary := d.renderPerformanceSummary(managerMetrics, bufferMetrics)
	sections = append(sections, summary)

	return strings.Join(sections, "\n")
}

// renderBufferMetrics renders buffer-specific metrics
func (d *LockFreeDashboard) renderBufferMetrics(metrics BufferMetrics) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("36")). // Cyan
		Padding(1, 2).
		Width((d.width / 2) - 2)

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("🔄 Ring Buffer"))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("Buffer Size:     %s", formatNumber(metrics.BufferSize)))
	lines = append(lines, fmt.Sprintf("Current Used:    %s", formatNumber(metrics.CurrentUsed)))
	lines = append(lines, fmt.Sprintf("Max Used:        %s", formatNumber(metrics.MaxUsedSlots)))

	// Usage bar
	usagePercent := float64(metrics.CurrentUsed) / float64(metrics.BufferSize) * 100
	usageBar := renderProgressBar(int(usagePercent), 30)
	lines = append(lines, fmt.Sprintf("Utilization:     %s %.1f%%", usageBar, usagePercent))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("Messages Written: %s", formatNumber(metrics.MessagesWritten)))
	lines = append(lines, fmt.Sprintf("Messages Dropped: %s", formatNumber(metrics.MessagesDropped)))
	lines = append(lines, fmt.Sprintf("Buffer Full Count: %s", formatNumber(metrics.BufferFullCount)))

	if metrics.MessagesWritten > 0 {
		dropRate := float64(metrics.MessagesDropped) / float64(metrics.MessagesWritten) * 100
		lines = append(lines, fmt.Sprintf("Drop Rate:       %.2f%%", dropRate))
	}

	return style.Render(strings.Join(lines, "\n"))
}

// renderManagerMetrics renders manager-specific metrics
func (d *LockFreeDashboard) renderManagerMetrics(metrics PTYManagerMetrics) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("35")). // Magenta
		Padding(1, 2).
		Width((d.width / 2) - 2)

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("⚡ Manager"))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("Active Sessions:  %d", metrics.activeSessions.Load()))
	lines = append(lines, fmt.Sprintf("Total Messages:   %s", formatNumber(metrics.totalMessages.Load())))
	lines = append(lines, fmt.Sprintf("Dropped Messages: %s", formatNumber(metrics.droppedMessages.Load())))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("Batches Processed: %s", formatNumber(metrics.batchesProcessed.Load())))
	lines = append(lines, fmt.Sprintf("Avg Batch Size:    %d", metrics.averageBatchSize.Load()))

	// Poll duration
	lastPollNs := metrics.lastPollDuration.Load()
	if lastPollNs > 0 {
		pollDuration := time.Duration(lastPollNs)
		lines = append(lines, fmt.Sprintf("Last Poll:        %v", pollDuration))
	}

	// Buffer utilization from manager perspective
	bufferUtil := metrics.bufferUtilization.Load()
	if bufferUtil > 0 {
		utilPercent := float64(bufferUtil) / 100.0
		utilBar := renderProgressBar(int(utilPercent), 20)
		lines = append(lines, fmt.Sprintf("Buffer Util:      %s %.1f%%", utilBar, utilPercent))
	}

	return style.Render(strings.Join(lines, "\n"))
}

// renderPerformanceSummary renders overall performance summary
func (d *LockFreeDashboard) renderPerformanceSummary(managerMetrics PTYManagerMetrics, bufferMetrics BufferMetrics) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("33")). // Yellow
		Padding(0, 2).
		Width(d.width - 4)

	var summary []string

	// Calculate throughput if we have batch data
	batchesProcessed := managerMetrics.batchesProcessed.Load()
	totalMessages := managerMetrics.totalMessages.Load()

	if batchesProcessed > 0 && totalMessages > 0 {
		avgBatchSize := float64(totalMessages) / float64(batchesProcessed)
		summary = append(summary, fmt.Sprintf("📊 Avg Batch Size: %.1f", avgBatchSize))
	}

	// Memory efficiency
	if bufferMetrics.BufferSize > 0 {
		efficiency := float64(bufferMetrics.MaxUsedSlots) / float64(bufferMetrics.BufferSize) * 100
		summary = append(summary, fmt.Sprintf("🎯 Peak Efficiency: %.1f%%", efficiency))
	}

	// Performance status
	dropRate := bufferMetrics.SaturationRatio * 100
	var status string
	if dropRate < 1 {
		status = "🟢 Excellent"
	} else if dropRate < 5 {
		status = "🟡 Good"
	} else {
		status = "🔴 Degraded"
	}
	summary = append(summary, fmt.Sprintf("Status: %s (%.2f%% drops)", status, dropRate))

	summaryText := "Performance: " + strings.Join(summary, " | ")
	return style.Render(summaryText)
}

// renderProgressBar renders a simple text progress bar
func renderProgressBar(percent int, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := (percent * width) / 100
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	// Color based on usage
	if percent < 50 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("32")).Render(bar) // Green
	} else if percent < 80 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render(bar) // Yellow
	} else {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("31")).Render(bar) // Red
	}
}

// formatNumber formats large numbers with K/M/G suffixes
func formatNumber(n uint64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	} else if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000.0)
	} else if n < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000.0)
	} else {
		return fmt.Sprintf("%.1fG", float64(n)/1000000000.0)
	}
}

// RenderQuickStats renders a compact one-line performance summary
func (d *LockFreeDashboard) RenderQuickStats(managerMetrics PTYManagerMetrics, bufferMetrics BufferMetrics) string {
	if bufferMetrics.BufferSize == 0 {
		return "Lock-free PTY: Initializing..."
	}

	utilization := float64(bufferMetrics.CurrentUsed) / float64(bufferMetrics.BufferSize) * 100
	dropRate := bufferMetrics.SaturationRatio * 100
	sessions := managerMetrics.activeSessions.Load()

	var indicator string
	if dropRate < 1 {
		indicator = "🟢"
	} else if dropRate < 5 {
		indicator = "🟡"
	} else {
		indicator = "🔴"
	}

	return fmt.Sprintf("%s LockFree: %d sessions, %.0f%% util, %.1f%% drops",
		indicator, sessions, utilization, dropRate)
}
