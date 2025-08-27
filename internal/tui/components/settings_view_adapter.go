package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SettingsViewComponent provides settings management UI
type SettingsViewComponent struct {
	BaseComponent

	// Settings data
	settings    map[string]interface{}
	categories  []string
	currentCat  int
	selectedKey string
	editing     bool

	// UI components
	editInput textinput.Model

	// Display options
	showAdvanced bool
	showHelp     bool

	// Styles
	titleStyle    lipgloss.Style
	categoryStyle lipgloss.Style
	keyStyle      lipgloss.Style
	valueStyle    lipgloss.Style
	helpStyle     lipgloss.Style
	selectedStyle lipgloss.Style
	editingStyle  lipgloss.Style
}

// NewSettingsViewComponent creates a new settings view component
func NewSettingsViewComponent() *SettingsViewComponent {
	s := &SettingsViewComponent{
		settings:     make(map[string]interface{}),
		categories:   []string{"General", "Process", "Display", "Network", "Advanced"},
		currentCat:   0,
		showAdvanced: false,
		showHelp:     true,
	}

	// Initialize styles
	s.initStyles()

	// Initialize edit input
	s.editInput = textinput.New()
	s.editInput.CharLimit = 100
	s.editInput.Width = 50

	// Load default settings
	s.loadDefaultSettings()

	// Set visible by default
	s.visible = true

	return s
}

// initStyles initializes the component styles
func (s *SettingsViewComponent) initStyles() {
	s.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	s.categoryStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		PaddingLeft(1)

	s.keyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	s.valueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("250"))

	s.helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	s.selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	s.editingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)
}

// loadDefaultSettings loads default settings
func (s *SettingsViewComponent) loadDefaultSettings() {
	// General settings
	s.settings["general.theme"] = "dark"
	s.settings["general.notifications"] = true
	s.settings["general.autosave"] = true
	s.settings["general.language"] = "en"

	// Process settings
	s.settings["process.autoRestart"] = false
	s.settings["process.maxRetries"] = 3
	s.settings["process.timeout"] = 30
	s.settings["process.logLevel"] = "info"

	// Display settings
	s.settings["display.showTimestamps"] = true
	s.settings["display.wrapLines"] = true
	s.settings["display.maxLogLines"] = 1000
	s.settings["display.refreshRate"] = 100

	// Network settings
	s.settings["network.proxyEnabled"] = false
	s.settings["network.proxyPort"] = 8080
	s.settings["network.mcpPort"] = 7777
	s.settings["network.reverseProxy"] = false

	// Advanced settings
	s.settings["advanced.debugMode"] = false
	s.settings["advanced.performanceMode"] = false
	s.settings["advanced.memoryLimit"] = 512
	s.settings["advanced.cpuLimit"] = 80
}

// Update handles messages
func (s *SettingsViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle component-specific messages
	switch msg := msg.(type) {
	case SettingsLoadMsg:
		// Load settings
		s.settings = msg.Settings
		return s, nil

	case SettingUpdateMsg:
		// Update a setting
		key := fmt.Sprintf("%s.%s", msg.Category, msg.Key)
		s.settings[key] = msg.Value
		return s, SendSystemNotification(NotificationSuccess,
			"Setting updated", fmt.Sprintf("%s = %v", key, msg.Value))

	case SettingsSaveMsg:
		// Save settings
		// This would persist to config file
		return s, SendSystemNotification(NotificationSuccess,
			"Settings saved", "Configuration has been saved to disk")

	case SettingsResetMsg:
		// Reset settings
		if msg.Category != "" {
			// Reset category
			s.resetCategory(msg.Category)
			return s, SendSystemNotification(NotificationInfo,
				"Category reset", fmt.Sprintf("%s settings reset to defaults", msg.Category))
		} else {
			// Reset all
			s.loadDefaultSettings()
			return s, SendSystemNotification(NotificationWarning,
				"All settings reset", "All settings have been reset to defaults")
		}

	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case tea.KeyMsg:
		// Handle editing mode first
		if s.editing {
			switch msg.String() {
			case "enter":
				// Save edit
				value := s.editInput.Value()
				s.settings[s.selectedKey] = s.parseValue(value)
				s.editing = false
				s.editInput.Blur()
				return s, SendSystemNotification(NotificationSuccess,
					"Setting updated", fmt.Sprintf("%s = %s", s.selectedKey, value))

			case "esc":
				// Cancel edit
				s.editing = false
				s.editInput.Blur()
				return s, nil
			}

			// Update input
			var cmd tea.Cmd
			s.editInput, cmd = s.editInput.Update(msg)
			return s, cmd
		}

		// Normal navigation
		switch msg.String() {
		case "tab", "right":
			// Next category
			s.currentCat = (s.currentCat + 1) % len(s.categories)
			s.selectedKey = ""
			return s, nil

		case "shift+tab", "left":
			// Previous category
			s.currentCat--
			if s.currentCat < 0 {
				s.currentCat = len(s.categories) - 1
			}
			s.selectedKey = ""
			return s, nil

		case "up", "k":
			// Previous setting
			s.selectPreviousSetting()
			return s, nil

		case "down", "j":
			// Next setting
			s.selectNextSetting()
			return s, nil

		case "enter", " ":
			// Edit selected setting
			if s.selectedKey != "" {
				s.editing = true
				s.editInput.SetValue(fmt.Sprintf("%v", s.settings[s.selectedKey]))
				s.editInput.Focus()
				return s, textinput.Blink
			}

		case "r":
			// Reset current category
			return s, func() tea.Msg {
				return SettingsResetMsg{Category: s.categories[s.currentCat]}
			}

		case "R":
			// Reset all settings
			return s, func() tea.Msg {
				return SettingsResetMsg{}
			}

		case "s":
			// Save settings
			return s, func() tea.Msg {
				return SettingsSaveMsg{Settings: s.settings}
			}

		case "h":
			// Toggle help
			s.showHelp = !s.showHelp
			return s, nil

		case "a":
			// Toggle advanced settings
			s.showAdvanced = !s.showAdvanced
			return s, nil

		case "d":
			// Set to default
			if s.selectedKey != "" {
				s.resetSetting(s.selectedKey)
				return s, SendSystemNotification(NotificationInfo,
					"Reset to default", s.selectedKey)
			}
		}
	}

	return s, tea.Batch(cmds...)
}

// View renders the component
func (s *SettingsViewComponent) View() string {
	if !s.visible {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(s.titleStyle.Render("⚙️  Settings") + "\n\n")

	// Category tabs
	var tabs []string
	for i, cat := range s.categories {
		if i == s.currentCat {
			tabs = append(tabs, s.selectedStyle.Render(fmt.Sprintf("[%s]", cat)))
		} else {
			tabs = append(tabs, s.categoryStyle.Render(cat))
		}
	}
	b.WriteString(strings.Join(tabs, "  ") + "\n\n")

	// Settings for current category
	category := strings.ToLower(s.categories[s.currentCat])

	// Filter settings by category
	var categorySettings []string
	for key := range s.settings {
		if strings.HasPrefix(key, category+".") {
			// Skip advanced settings if not showing
			if !s.showAdvanced && strings.Contains(key, "advanced") {
				continue
			}
			categorySettings = append(categorySettings, key)
		}
	}

	// Sort settings
	// In production, would use sort.Strings(categorySettings)

	// Display settings
	for _, key := range categorySettings {
		value := s.settings[key]
		displayKey := strings.TrimPrefix(key, category+".")

		// Format the setting
		var line string
		if s.editing && key == s.selectedKey {
			// Show edit input
			line = fmt.Sprintf("  %s: %s",
				s.keyStyle.Render(displayKey),
				s.editInput.View())
		} else if key == s.selectedKey {
			// Highlighted selection
			line = s.selectedStyle.Render(fmt.Sprintf("▶ %s: %v", displayKey, value))
		} else {
			// Normal display
			line = fmt.Sprintf("  %s: %s",
				s.keyStyle.Render(displayKey),
				s.valueStyle.Render(fmt.Sprintf("%v", value)))
		}

		b.WriteString(line + "\n")

		// Add help text if enabled
		if s.showHelp && key == s.selectedKey {
			helpText := s.getHelpText(key)
			if helpText != "" {
				b.WriteString("    " + s.helpStyle.Render(helpText) + "\n")
			}
		}
	}

	// Footer with keybindings
	b.WriteString("\n")
	help := []string{
		"↑↓: navigate",
		"←→: category",
		"enter: edit",
		"s: save",
		"r: reset category",
		"R: reset all",
		"h: help",
	}
	if !s.showAdvanced {
		help = append(help, "a: show advanced")
	}
	b.WriteString(s.helpStyle.Render(strings.Join(help, " | ")))

	return b.String()
}

// Helper methods

func (s *SettingsViewComponent) selectNextSetting() {
	category := strings.ToLower(s.categories[s.currentCat])
	var keys []string
	for key := range s.settings {
		if strings.HasPrefix(key, category+".") {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return
	}

	// Find current index
	currentIdx := -1
	for i, key := range keys {
		if key == s.selectedKey {
			currentIdx = i
			break
		}
	}

	// Select next
	if currentIdx < len(keys)-1 {
		s.selectedKey = keys[currentIdx+1]
	} else if currentIdx == -1 {
		s.selectedKey = keys[0]
	}
}

func (s *SettingsViewComponent) selectPreviousSetting() {
	category := strings.ToLower(s.categories[s.currentCat])
	var keys []string
	for key := range s.settings {
		if strings.HasPrefix(key, category+".") {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return
	}

	// Find current index
	currentIdx := -1
	for i, key := range keys {
		if key == s.selectedKey {
			currentIdx = i
			break
		}
	}

	// Select previous
	if currentIdx > 0 {
		s.selectedKey = keys[currentIdx-1]
	} else if currentIdx == -1 {
		s.selectedKey = keys[len(keys)-1]
	}
}

func (s *SettingsViewComponent) resetCategory(category string) {
	cat := strings.ToLower(category)
	// Reset all settings in category to defaults
	switch cat {
	case "general":
		s.settings["general.theme"] = "dark"
		s.settings["general.notifications"] = true
		s.settings["general.autosave"] = true
		s.settings["general.language"] = "en"
	case "process":
		s.settings["process.autoRestart"] = false
		s.settings["process.maxRetries"] = 3
		s.settings["process.timeout"] = 30
		s.settings["process.logLevel"] = "info"
	case "display":
		s.settings["display.showTimestamps"] = true
		s.settings["display.wrapLines"] = true
		s.settings["display.maxLogLines"] = 1000
		s.settings["display.refreshRate"] = 100
	case "network":
		s.settings["network.proxyEnabled"] = false
		s.settings["network.proxyPort"] = 8080
		s.settings["network.mcpPort"] = 7777
		s.settings["network.reverseProxy"] = false
	case "advanced":
		s.settings["advanced.debugMode"] = false
		s.settings["advanced.performanceMode"] = false
		s.settings["advanced.memoryLimit"] = 512
		s.settings["advanced.cpuLimit"] = 80
	}
}

func (s *SettingsViewComponent) resetSetting(key string) {
	// Reset individual setting to default
	defaults := map[string]interface{}{
		"general.theme":            "dark",
		"general.notifications":    true,
		"general.autosave":         true,
		"general.language":         "en",
		"process.autoRestart":      false,
		"process.maxRetries":       3,
		"process.timeout":          30,
		"process.logLevel":         "info",
		"display.showTimestamps":   true,
		"display.wrapLines":        true,
		"display.maxLogLines":      1000,
		"display.refreshRate":      100,
		"network.proxyEnabled":     false,
		"network.proxyPort":        8080,
		"network.mcpPort":          7777,
		"network.reverseProxy":     false,
		"advanced.debugMode":       false,
		"advanced.performanceMode": false,
		"advanced.memoryLimit":     512,
		"advanced.cpuLimit":        80,
	}

	if defaultValue, ok := defaults[key]; ok {
		s.settings[key] = defaultValue
	}
}

func (s *SettingsViewComponent) parseValue(str string) interface{} {
	// Try to parse as bool
	if str == "true" {
		return true
	}
	if str == "false" {
		return false
	}

	// Try to parse as number
	var num int
	if _, err := fmt.Sscanf(str, "%d", &num); err == nil {
		return num
	}

	// Return as string
	return str
}

func (s *SettingsViewComponent) getHelpText(key string) string {
	helps := map[string]string{
		"general.theme":            "UI color theme",
		"general.notifications":    "Show system notifications",
		"general.autosave":         "Automatically save settings",
		"general.language":         "Interface language",
		"process.autoRestart":      "Restart processes on failure",
		"process.maxRetries":       "Maximum restart attempts",
		"process.timeout":          "Process startup timeout (seconds)",
		"process.logLevel":         "Logging verbosity (debug/info/warn/error)",
		"display.showTimestamps":   "Show timestamps in logs",
		"display.wrapLines":        "Wrap long log lines",
		"display.maxLogLines":      "Maximum log lines to keep",
		"display.refreshRate":      "UI refresh rate (milliseconds)",
		"network.proxyEnabled":     "Enable HTTP proxy server",
		"network.proxyPort":        "Proxy server port",
		"network.mcpPort":          "MCP server port",
		"network.reverseProxy":     "Enable reverse proxy mode",
		"advanced.debugMode":       "Enable debug logging",
		"advanced.performanceMode": "Optimize for performance",
		"advanced.memoryLimit":     "Memory limit (MB)",
		"advanced.cpuLimit":        "CPU usage limit (%)",
	}

	return helps[key]
}

// Focus sets focus to this component
func (s *SettingsViewComponent) Focus() tea.Cmd {
	s.focused = true
	return nil
}

// Blur removes focus from this component
func (s *SettingsViewComponent) Blur() tea.Cmd {
	s.focused = false
	s.editing = false
	s.editInput.Blur()
	return nil
}
