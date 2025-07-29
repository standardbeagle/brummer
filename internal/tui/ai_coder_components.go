package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/aicoder"
)

// AICoderItem represents an AI coder in the list
type AICoderItem struct {
	coder *aicoder.AICoderProcess
}

// FilterValue implements list.Item
func (i AICoderItem) FilterValue() string {
	name := i.coder.Name
	if name == "" {
		name = i.coder.SessionID
	}
	return name + " " + i.coder.Task
}

// Title implements list.DefaultItem
func (i AICoderItem) Title() string {
	name := i.coder.Name
	if name == "" {
		name = i.coder.SessionID
	}
	
	status := strings.ToUpper(string(i.coder.Status))
	statusColor := getStatusColor(i.coder.Status)
	
	return fmt.Sprintf("%s %s",
		statusColor.Render(status),
		name,
	)
}

// Description implements list.DefaultItem
func (i AICoderItem) Description() string {
	elapsed := time.Since(i.coder.CreatedAt)
	progress := fmt.Sprintf("%.1f%%", i.coder.Progress*100)
	
	desc := fmt.Sprintf("%s | %s | %s",
		truncateString(i.coder.Task, 40),
		progress,
		formatDuration(elapsed),
	)
	
	if i.coder.CurrentMessage != "" {
		desc = fmt.Sprintf("%s\n%s", desc, dimStyle.Render(truncateString(i.coder.CurrentMessage, 50)))
	}
	
	return desc
}

// AICoderDelegate customizes the list item rendering
type AICoderDelegate struct{}

// NewAICoderDelegate creates a new delegate for AI coder list items
func NewAICoderDelegate() AICoderDelegate {
	return AICoderDelegate{}
}

// Height returns the height of each item
func (d AICoderDelegate) Height() int { return 3 }

// Spacing returns the spacing between items
func (d AICoderDelegate) Spacing() int { return 1 }

// Update handles item updates
func (d AICoderDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

// Render renders a list item
func (d AICoderDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(AICoderItem)
	if !ok {
		return
	}
	
	coder := i.coder
	
	// Determine if item is selected
	isSelected := index == m.Index()
	
	// Get base style
	var baseStyle lipgloss.Style
	if isSelected {
		baseStyle = selectedItemStyle
	} else {
		baseStyle = normalItemStyle
	}
	
	// Status indicator
	statusIcon := getStatusIcon(coder.Status)
	statusColor := getStatusColor(coder.Status)
	
	// Progress bar
	progressBar := renderProgressBar(coder.Progress, 20)
	
	// Format name
	name := coder.Name
	if name == "" {
		name = coder.SessionID
	}
	
	// Build content
	var content strings.Builder
	
	// First line: Status, Name, Provider
	firstLine := fmt.Sprintf("%s %s [%s]",
		statusColor.Render(statusIcon),
		name,
		coder.Provider,
	)
	content.WriteString(firstLine)
	content.WriteString("\n")
	
	// Second line: Task (truncated)
	taskLine := truncateString(coder.Task, 50)
	content.WriteString(dimStyle.Render(taskLine))
	content.WriteString("\n")
	
	// Third line: Progress bar and time
	elapsed := formatDuration(time.Since(coder.CreatedAt))
	progressLine := fmt.Sprintf("%s %s", progressBar, elapsed)
	content.WriteString(progressLine)
	
	// Apply base style and render
	fmt.Fprint(w, baseStyle.Render(content.String()))
}

// Progress bar rendering
func renderProgressBar(progress float64, width int) string {
	if width < 10 {
		return fmt.Sprintf("%.1f%%", progress*100)
	}
	
	// Calculate filled and empty portions
	barWidth := width - 7 // Leave room for percentage
	filled := int(progress * float64(barWidth))
	empty := barWidth - filled
	
	// Build bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	percentage := fmt.Sprintf("%3.0f%%", progress*100)
	
	return fmt.Sprintf("[%s] %s", bar, percentage)
}

// Status color and styling
func getStatusColor(status aicoder.AICoderStatus) lipgloss.Style {
	switch status {
	case aicoder.StatusRunning:
		return greenStyle
	case aicoder.StatusCompleted:
		return blueStyle
	case aicoder.StatusFailed:
		return redStyle
	case aicoder.StatusPaused:
		return yellowStyle
	case aicoder.StatusCreating:
		return cyanStyle
	default:
		return dimStyle
	}
}

// Status icon
func getStatusIcon(status aicoder.AICoderStatus) string {
	switch status {
	case aicoder.StatusRunning:
		return "▶"
	case aicoder.StatusCompleted:
		return "✓"
	case aicoder.StatusFailed:
		return "✗"
	case aicoder.StatusPaused:
		return "⏸"
	case aicoder.StatusCreating:
		return "⚙"
	case aicoder.StatusStopped:
		return "■"
	default:
		return "○"
	}
}

// Utility functions
func truncateString(s string, maxLen int) string {
	if lipgloss.Width(s) <= maxLen {
		return s
	}
	
	// Handle multi-byte characters properly
	runes := []rune(s)
	if len(runes) <= maxLen-3 {
		return s
	}
	
	return string(runes[:maxLen-3]) + "..."
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		if hours > 0 {
			return fmt.Sprintf("%dd%dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}
}

func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}
	
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	
	var lines []string
	var currentLine string
	
	for _, word := range words {
		wordWidth := lipgloss.Width(word)
		currentWidth := lipgloss.Width(currentLine)
		
		if currentLine == "" {
			currentLine = word
		} else if currentWidth+1+wordWidth <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
		
		// Handle very long words
		if wordWidth > width {
			// Break the word
			runes := []rune(word)
			for i := 0; i < len(runes); i += width {
				end := i + width
				if end > len(runes) {
					end = len(runes)
				}
				if currentLine != "" {
					lines = append(lines, currentLine)
				}
				currentLine = string(runes[i:end])
			}
		}
	}
	
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	
	return strings.Join(lines, "\n")
}

// min function is defined in model.go - removing duplicate