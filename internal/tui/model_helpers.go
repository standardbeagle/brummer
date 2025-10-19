package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/domain/processes"
	"github.com/standardbeagle/brummer/internal/tui/services"
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
	allProcesses := h.model.processMgr.GetAllProcesses()
	runningCount := 0
	for _, proc := range allProcesses {
		if proc.GetStatus() == processes.StatusRunning {
			runningCount++
		}
	}

	if len(allProcesses) > 0 {
		if runningCount > 0 {
			return fmt.Sprintf(" (%d processes, %d running)", len(allProcesses), runningCount)
		}
		return fmt.Sprintf(" (%d processes)", len(allProcesses))
	}
	return ""
}

// RenderTitle returns the formatted title for the header
func (h *HeaderRenderer) RenderTitle() string {
	baseTitle := "🐝 Brummer - Development Buddy"
	processInfo := h.RenderProcessInfo()

	// Add copy notification if recent
	notification := ""
	if h.model.notificationsComponent != nil {
		// Check if notifications component has active notifications
		if activeNotifications := h.model.notificationsComponent.GetActiveNotifications(); len(activeNotifications) > 0 {
			notificationStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("82")).
				Bold(true)
			// Get the most recent notification message
			notification = " " + notificationStyle.Render(activeNotifications[len(activeNotifications)-1].Message)
		}
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
			if h.model.systemComponent != nil {
				unreadCount := h.model.systemComponent.GetUnreadCount()
				if unreadCount > 0 {
					indicatorIcon = "🔴" // Simple indicator for any unread messages
				}
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
		allProcesses := v.model.processMgr.GetAllProcesses()
		running := 0
		for _, p := range allProcesses {
			if p.GetStatus() == processes.StatusRunning {
				running++
			}
		}
		return fmt.Sprintf("%d processes, %d running", len(allProcesses), running)

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
		// TODO(Feature 005): Get session count from service
		if v.model.aiCoderService != nil {
			svc := v.model.aiCoderService.(*services.AICoderService)
			sessions := svc.ListSessions()
			return fmt.Sprintf("%d AI coder sessions", len(sessions))
		}
		return "0 AI coder sessions"

	default:
		return ""
	}
}

// CLICommandHelper provides CLI command lookup functionality
type CLICommandHelper struct {
	cfg *config.Config
}

// NewCLICommandHelper creates a new CLI command helper
func NewCLICommandHelper(cfg *config.Config) *CLICommandHelper {
	return &CLICommandHelper{cfg: cfg}
}

// GetCLICommandFromConfig retrieves CLI command and args from configuration with template substitution
func (c *CLICommandHelper) GetCLICommandFromConfig(configKey string, task string) (string, []string, error) {
	// First try to get configuration from the loaded config
	if c.cfg != nil && c.cfg.AICoders != nil {
		provider, exists := c.cfg.AICoders.Providers[configKey]
		if exists && provider.CLITool != nil {
			return c.buildCommandFromConfig(configKey, provider.CLITool, task)
		}
	}

	// Fallback to hardcoded mappings for backward compatibility
	return c.getFallbackCommand(configKey, task)
}

// buildCommandFromConfig builds CLI command from configuration with template substitution
func (c *CLICommandHelper) buildCommandFromConfig(providerKey string, cliTool *config.CLIToolConfig, task string) (string, []string, error) {
	if cliTool.Command == nil {
		return "", nil, fmt.Errorf("no command specified for provider '%s'", providerKey)
	}

	command := *cliTool.Command

	// Process base_args with template substitution
	var baseArgs []string
	if len(cliTool.BaseArgs) > 0 {
		processedArgs, err := c.substituteTemplateVars(cliTool.BaseArgs)
		if err != nil {
			return "", nil, fmt.Errorf("template substitution failed for provider '%s': %w", providerKey, err)
		}
		baseArgs = processedArgs
	}

	// Start with processed base args
	args := make([]string, len(baseArgs))
	copy(args, baseArgs)

	// Add task if provided, using configured task flag if available
	if task != "" {
		if cliTool.FlagMapping != nil {
			if taskFlag, hasTaskFlag := cliTool.FlagMapping["task"]; hasTaskFlag {
				args = append(args, taskFlag, task)
			} else if messageFlag, hasMessageFlag := cliTool.FlagMapping["message"]; hasMessageFlag {
				args = append(args, messageFlag, task)
			} else {
				// No task flag configured, add as direct argument
				args = append(args, task)
			}
		} else {
			// No flag mapping, use heuristics based on command
			args = c.addTaskWithHeuristics(providerKey, command, args, task)
		}
	}

	return command, args, nil
}

// substituteTemplateVars performs template variable substitution in string slices
func (c *CLICommandHelper) substituteTemplateVars(args []string) ([]string, error) {
	// Prepare template data
	templateData := c.prepareTemplateData()

	result := make([]string, len(args))
	for i, arg := range args {
		// Check if arg contains template variables
		if strings.Contains(arg, "${") {
			processed := c.substituteVars(arg, templateData)
			result[i] = processed
		} else {
			result[i] = arg
		}
	}

	return result, nil
}

// prepareTemplateData prepares the data map for template substitution
func (c *CLICommandHelper) prepareTemplateData() map[string]interface{} {
	data := map[string]interface{}{}

	// Add MCP configuration
	if c.cfg != nil {
		// Get the actual MCP port from configuration
		mcpPort := c.cfg.GetMCPPort()
		data["BRUMMER_MCP_PORT"] = mcpPort

		// Build MCP URL - use localhost for now, could be configurable in future
		data["BRUMMER_MCP_URL"] = fmt.Sprintf("http://localhost:%d/mcp", mcpPort)

		// Add other potentially useful variables
		data["BRUMMER_PROXY_PORT"] = c.cfg.GetProxyPort()
		data["BRUMMER_PROXY_MODE"] = c.cfg.GetProxyMode()
	}

	// Add environment variables (they can override config values)
	data["PWD"], _ = os.Getwd()
	data["HOME"] = os.Getenv("HOME")
	data["USER"] = os.Getenv("USER")

	// Add any environment variables that might be referenced
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "BRUMMER_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				data[parts[0]] = parts[1]
			}
		}
	}

	return data
}

// substituteVars performs ${VARIABLE} substitution in a string
func (c *CLICommandHelper) substituteVars(input string, data map[string]interface{}) string {
	result := input

	// Find all ${VAR} patterns
	for key, value := range data {
		pattern := fmt.Sprintf("${%s}", key)
		replacement := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, pattern, replacement)
	}

	return result
}

// addTaskWithHeuristics adds task argument using command-specific heuristics
func (c *CLICommandHelper) addTaskWithHeuristics(providerKey, command string, args []string, task string) []string {
	switch providerKey {
	case "claude", "sonnet", "opus":
		// Claude CLI variants - add streaming flags for non-interactive mode
		args = append(args, "--print", "--verbose", "--output-format", "stream-json")
		args = append(args, task)
	case "aider":
		// Aider uses --message flag
		args = append(args, "--message", task)
	case "opencode":
		// OpenCode uses --prompt flag
		args = append(args, "--prompt", task)
	case "gemini":
		// Gemini uses --prompt flag
		args = append(args, "--prompt", task)
	default:
		// Default: add as direct argument
		args = append(args, task)
	}

	return args
}

// getFallbackCommand provides backward compatibility with hardcoded mappings
func (c *CLICommandHelper) getFallbackCommand(configKey string, task string) (string, []string, error) {
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
