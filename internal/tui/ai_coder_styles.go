package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// AI Coder View Styles

var (
	// Base colors for AI coder status
	aiCoderRunningColor   = lipgloss.Color("2")  // Green
	aiCoderCompletedColor = lipgloss.Color("4")  // Blue
	aiCoderFailedColor    = lipgloss.Color("1")  // Red
	aiCoderPausedColor    = lipgloss.Color("3")  // Yellow
	aiCoderCreatingColor  = lipgloss.Color("6")  // Cyan
	aiCoderStoppedColor   = lipgloss.Color("8")  // Gray
	aiCoderDefaultColor   = lipgloss.Color("7")  // White

	// Status styles
	greenStyle  = lipgloss.NewStyle().Foreground(aiCoderRunningColor)
	blueStyle   = lipgloss.NewStyle().Foreground(aiCoderCompletedColor)
	redStyle    = lipgloss.NewStyle().Foreground(aiCoderFailedColor)
	yellowStyle = lipgloss.NewStyle().Foreground(aiCoderPausedColor)
	cyanStyle   = lipgloss.NewStyle().Foreground(aiCoderCreatingColor)
	grayStyle   = lipgloss.NewStyle().Foreground(aiCoderStoppedColor)

	// Text styles
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Faint(true)

	boldStyle = lipgloss.NewStyle().
			Bold(true)

	// Box styles
	normalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	focusedBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69"))

	// List item styles
	normalItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62"))

	// Title and header styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 2).
			Align(lipgloss.Center)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("69")).
			Underline(true).
			MarginBottom(1)

	// Status bar style
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	// Command input styles
	commandInputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(0, 1)

	// Dialog styles
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69")).
			Padding(1, 2).
			Background(lipgloss.Color("236"))

	// Progress bar styles
	progressBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("69"))

	progressBarEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	// AI Coder specific styles
	aiCoderNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	aiCoderTaskStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("247"))

	aiCoderProviderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	aiCoderWorkspaceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Faint(true)

	aiCoderOutputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	aiCoderErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Background(lipgloss.Color("52")).
			Padding(0, 1)

	// Help styles
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	// Badge styles for AI coder counts
	runningBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(aiCoderRunningColor).
			Padding(0, 1).
			Bold(true)

	pausedBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(aiCoderPausedColor).
			Padding(0, 1).
			Bold(true)

	completedBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(aiCoderCompletedColor).
			Padding(0, 1).
			Bold(true)

	failedBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(aiCoderFailedColor).
			Padding(0, 1).
			Bold(true)
)

// GetAICoderTheme returns a theme configuration for AI coder view
type AICoderTheme struct {
	// Status colors
	RunningColor   lipgloss.Color
	CompletedColor lipgloss.Color
	FailedColor    lipgloss.Color
	PausedColor    lipgloss.Color
	CreatingColor  lipgloss.Color
	StoppedColor   lipgloss.Color

	// Component styles
	ListStyle      lipgloss.Style
	DetailStyle    lipgloss.Style
	CommandStyle   lipgloss.Style
	StatusBarStyle lipgloss.Style

	// Text styles
	TitleStyle   lipgloss.Style
	HeaderStyle  lipgloss.Style
	SectionStyle lipgloss.Style
	DimStyle     lipgloss.Style
	BoldStyle    lipgloss.Style
}

// DefaultAICoderTheme returns the default theme for AI coder view
func DefaultAICoderTheme() AICoderTheme {
	return AICoderTheme{
		RunningColor:   aiCoderRunningColor,
		CompletedColor: aiCoderCompletedColor,
		FailedColor:    aiCoderFailedColor,
		PausedColor:    aiCoderPausedColor,
		CreatingColor:  aiCoderCreatingColor,
		StoppedColor:   aiCoderStoppedColor,

		ListStyle:      normalBoxStyle,
		DetailStyle:    normalBoxStyle,
		CommandStyle:   commandInputBoxStyle,
		StatusBarStyle: statusBarStyle,

		TitleStyle:   titleStyle,
		HeaderStyle:  headerStyle,
		SectionStyle: sectionStyle,
		DimStyle:     dimStyle,
		BoldStyle:    boldStyle,
	}
}

// RenderAICoderStatusBadge renders a colored badge for AI coder status
func RenderAICoderStatusBadge(status string, count int) string {
	if count == 0 {
		return ""
	}

	var style lipgloss.Style
	switch status {
	case "running":
		style = runningBadgeStyle
	case "paused":
		style = pausedBadgeStyle
	case "completed":
		style = completedBadgeStyle
	case "failed":
		style = failedBadgeStyle
	default:
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)
	}

	return style.Render(fmt.Sprintf("%d", count))
}

// RenderAICoderProgress renders a styled progress indicator
func RenderAICoderProgress(progress float64, width int) string {
	if width < 10 {
		return fmt.Sprintf("%.0f%%", progress*100)
	}

	// Calculate filled and empty portions
	barWidth := width - 7 // Leave room for percentage
	filled := int(progress * float64(barWidth))
	empty := barWidth - filled

	// Build progress bar
	filledBar := progressBarStyle.Render(strings.Repeat("█", filled))
	emptyBar := progressBarEmptyStyle.Render(strings.Repeat("░", empty))
	percentage := fmt.Sprintf("%3.0f%%", progress*100)

	return fmt.Sprintf("[%s%s] %s", filledBar, emptyBar, percentage)
}

// ApplyAICoderListStyle applies appropriate styling to AI coder list item
func ApplyAICoderListStyle(content string, isSelected bool, status string) string {
	var baseStyle lipgloss.Style
	if isSelected {
		baseStyle = selectedItemStyle
	} else {
		baseStyle = normalItemStyle
	}

	// Apply status-specific accent if needed
	return baseStyle.Render(content)
}

// GetAICoderStatusStyle returns the appropriate style for a given status
func GetAICoderStatusStyle(status string) lipgloss.Style {
	switch status {
	case "running":
		return greenStyle
	case "completed":
		return blueStyle
	case "failed":
		return redStyle
	case "paused":
		return yellowStyle
	case "creating":
		return cyanStyle
	case "stopped":
		return grayStyle
	default:
		return dimStyle
	}
}