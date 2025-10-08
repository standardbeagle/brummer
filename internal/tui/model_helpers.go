package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/process"
)

// HeaderRenderer handles header rendering logic
type HeaderRenderer struct {
	model *Model
}

// NewHeaderRenderer creates a new header renderer
func NewHeaderRenderer(model *Model) *HeaderRenderer {
	return &HeaderRenderer{model: model}
}

// RenderProcessInfo returns formatted process information for the header
func (h *HeaderRenderer) RenderProcessInfo() string {
	processes := h.model.processMgr.GetAllProcesses()
	runningCount := 0
	for _, proc := range processes {
		if proc.GetStatus() == process.StatusRunning {
			runningCount++
		}
	}

	if len(processes) > 0 {
		if runningCount > 0 {
			return fmt.Sprintf(" (%d processes, %d running)", len(processes), runningCount)
		}
		return fmt.Sprintf(" (%d processes)", len(processes))
	}
	return ""
}

// RenderTitle returns the formatted title for the header
func (h *HeaderRenderer) RenderTitle() string {
	baseTitle := "🐝 Brummer - Development Buddy"
	processInfo := h.RenderProcessInfo()

	// Add copy notification if recent
	notification := ""
	if h.model.notificationsController.IsActive() {
		notificationStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)
		notification = " " + notificationStyle.Render(h.model.notificationsController.GetMessage())
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Render(baseTitle + processInfo + notification)

	return title
}

// RenderTabs returns the formatted tabs for the header
func (h *HeaderRenderer) RenderTabs() string {
	viewConfigs := GetViewConfigs()
	orderedViews := GetOrderedViews(h.model.debugMode)

	tabs := []string{}
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226"))
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	for i, viewType := range orderedViews {
		if cfg, ok := viewConfigs[viewType]; ok {
			// Build the base label with icon and space before number
			label := fmt.Sprintf("%s.%s", cfg.KeyBinding, cfg.Title)
			if cfg.Icon != "" {
				label = cfg.Icon + " " + label
			}

			// Get unread indicator for this view
			var indicatorIcon string
			indicators := h.model.systemController.GetUnreadIndicators()
			if indicator, exists := indicators[viewType]; exists && indicator.Count > 0 {
				indicatorIcon = indicator.Icon
			}

			// Format the tab
			var tab string
			if viewType == h.model.currentView() {
				// Active tab: ▶icon1.Titleindicator
				tab = activeStyle.Render("▶" + label + indicatorIcon)
			} else {
				// Inactive tab:  icon1.Titleindicator
				tab = inactiveStyle.Render(" " + label + indicatorIcon)
			}

			// Add separator except for the last tab
			if i < len(orderedViews)-1 {
				tab += separatorStyle.Render(" | ")
			}

			tabs = append(tabs, tab)
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, tabs...)
}

// ViewStatusProvider provides status information for different views
type ViewStatusProvider struct {
	model *Model
}

// NewViewStatusProvider creates a new view status provider
func NewViewStatusProvider(model *Model) *ViewStatusProvider {
	return &ViewStatusProvider{model: model}
}

// GetStatus returns status information for the current view
func (v *ViewStatusProvider) GetStatus() string {
	switch v.model.currentView() {
	case ViewProcesses:
		processes := v.model.processMgr.GetAllProcesses()
		running := 0
		for _, p := range processes {
			if p.GetStatus() == process.StatusRunning {
				running++
			}
		}
		return fmt.Sprintf("%d processes, %d running", len(processes), running)

	case ViewLogs:
		if v.model.selectedProcess != "" {
			return fmt.Sprintf("Process: %s", v.model.selectedProcess)
		}
		return "All processes"

	case ViewErrors:
		errors := v.model.logStore.GetErrors()
		errorCount := len(errors)
		return fmt.Sprintf("%d errors", errorCount)

	case ViewURLs:
		urls := v.model.logStore.GetURLs()
		return fmt.Sprintf("%d URLs detected", len(urls))

	case ViewWeb:
		requests := v.model.proxyServer.GetRequests()
		return fmt.Sprintf("%d requests", len(requests))

	case ViewAICoders:
		if v.model.aiCoderController != nil {
			running, total := v.model.aiCoderController.GetStatusInfo()
			return fmt.Sprintf("%d AI coders, %d running", total, running)
		}
		return "AI Coder feature not initialized"

	default:
		return ""
	}
}

// CLICommandHelper provides CLI command lookup functionality
type CLICommandHelper struct{}

// NewCLICommandHelper creates a new CLI command helper
func NewCLICommandHelper() *CLICommandHelper {
	return &CLICommandHelper{}
}

// GetCLICommandFromConfig retrieves CLI command and args from configuration
func (c *CLICommandHelper) GetCLICommandFromConfig(configKey string, task string) (string, []string, error) {
	cliMappings := map[string]struct {
		command  string
		baseArgs []string
		taskFlag string
	}{
		"claude": {
			command:  "claude",
			baseArgs: []string{},
			taskFlag: "", // Claude CLI accepts task as direct input
		},
		"sonnet": {
			command:  "claude",
			baseArgs: []string{"--model", "sonnet"},
			taskFlag: "",
		},
		"opus": {
			command:  "claude",
			baseArgs: []string{"--model", "opus"},
			taskFlag: "",
		},
		"aider": {
			command:  "aider",
			baseArgs: []string{"--yes"},
			taskFlag: "--message",
		},
		"opencode": {
			command:  "opencode",
			baseArgs: []string{"run"},
			taskFlag: "--prompt",
		},
		"gemini": {
			command:  "gemini",
			baseArgs: []string{},
			taskFlag: "--prompt",
		},
	}

	mapping, exists := cliMappings[configKey]
	if !exists {
		return "", nil, fmt.Errorf("CLI configuration not found for '%s'", configKey)
	}

	args := make([]string, len(mapping.baseArgs))
	copy(args, mapping.baseArgs)

	// If task is provided, add it based on the tool's requirements
	if task != "" {
		if mapping.taskFlag != "" {
			// Tools that use flags for tasks (aider, opencode, gemini)
			args = append(args, mapping.taskFlag, task)
		} else {
			// Tools that accept task as direct input (claude)
			// For non-interactive mode, use structured streaming output
			if configKey == "claude" || configKey == "sonnet" || configKey == "opus" {
				args = append(args, "--print", "--verbose", "--output-format", "stream-json")
			}
			args = append(args, task)
		}
	}

	return mapping.command, args, nil
}

// ContentHeightCalculator calculates available content height
type ContentHeightCalculator struct{}

// NewContentHeightCalculator creates a new content height calculator
func NewContentHeightCalculator() *ContentHeightCalculator {
	return &ContentHeightCalculator{}
}

// Calculate returns the available content height with consistent calculation
func (c *ContentHeightCalculator) Calculate(totalHeight, headerHeight, footerHeight int) int {
	// Subtract 1 to account for proper spacing between header and content
	contentHeight := totalHeight - headerHeight - footerHeight - 1
	if contentHeight < 1 {
		contentHeight = 1
	}
	return contentHeight
}

// StringTrimmer provides string truncation utilities
type StringTrimmer struct{}

// NewStringTrimmer creates a new string trimmer
func NewStringTrimmer() *StringTrimmer {
	return &StringTrimmer{}
}

// TruncateWithEllipsis truncates a string and adds ellipsis if needed
func (s *StringTrimmer) TruncateWithEllipsis(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	if maxLength <= 3 {
		return strings.Repeat(".", maxLength)
	}
	return text[:maxLength-3] + "..."
}

// SanitizeForDisplay removes or replaces problematic characters
func (s *StringTrimmer) SanitizeForDisplay(text string) string {
	// Replace common problematic characters
	replacements := map[string]string{
		"\r\n": " ",
		"\r":   " ",
		"\n":   " ",
		"\t":   " ",
	}

	result := text
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	// Trim whitespace
	return strings.TrimSpace(result)
}
