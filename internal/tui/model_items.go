package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/standardbeagle/brummer/internal/parser"
	
	processes "github.com/standardbeagle/brummer/internal/domain/processes"
	"github.com/standardbeagle/brummer/internal/proxy"
)

// ScriptItem represents a script in the script selector
type ScriptItem struct {
	name   string
	script string
}

func (i ScriptItem) FilterValue() string { return i.name }
func (i ScriptItem) Title() string       { return i.name }
func (i ScriptItem) Description() string { return i.script }

// RunCustomItem represents the "run custom command" option
type RunCustomItem struct{}

func (i RunCustomItem) FilterValue() string { return "custom command" }
func (i RunCustomItem) Title() string       { return "➕ Run Custom Command..." }
func (i RunCustomItem) Description() string { return "Run a custom command not listed above" }

// ProcessItem represents a process in the processes view
type ProcessItem struct {
	process    *processes.Process
	isHeader   bool
	headerText string
}

func (i ProcessItem) FilterValue() string {
	if i.isHeader || i.process == nil {
		return ""
	}
	return i.process.Name
}

func (i ProcessItem) Title() string {
	if i.isHeader {
		// Return section header with styling
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
		if i.headerText == "Closed Processes" {
			headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
		}
		return headerStyle.Render(i.headerText)
	}

	// Safety check for nil process
	if i.process == nil {
		return ""
	}

	// Use ProcessState for atomic access to status and name
	state := i.process.GetStateAtomic()
	status := string(state.Status)
	var statusEmoji string
	switch state.Status {
	case processes.StatusRunning:
		statusEmoji = "🟢"
	case processes.StatusStopped:
		statusEmoji = "✓" // Thin checkmark for gracefully stopped
	case processes.StatusFailed:
		statusEmoji = "❌"
	case processes.StatusSuccess:
		statusEmoji = "✓" // Thin checkmark for success
	default:
		statusEmoji = "⏸️"
	}
	title := fmt.Sprintf("%s [%s] %s", statusEmoji, status, state.Name)

	return title
}

func (i ProcessItem) Description() string {
	if i.isHeader {
		// Return separator line for headers using lipgloss
		separatorStyle := lipgloss.NewStyle().
			Width(40).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderBottom(false).
			BorderLeft(false).
			BorderRight(false).
			BorderForeground(lipgloss.Color("240"))
		return separatorStyle.Render("")
	}

	// Safety check for nil process
	if i.process == nil {
		return ""
	}

	var parts []string

	// Use ProcessState for atomic access to multiple fields
	state := i.process.GetStateAtomic()

	// Add PID and start time
	parts = append(parts, fmt.Sprintf("PID: %s", state.ID))
	parts = append(parts, fmt.Sprintf("Started: %s", state.StartTime.Format("15:04:05")))

	// Add exit code for failed/stopped processes
	if state.IsFinished() && state.ExitCode != nil {
		parts = append(parts, fmt.Sprintf("Exit: %d", *state.ExitCode))
	}

	// Add runtime if process has ended
	if state.EndTime != nil {
		runtime := state.EndTime.Sub(state.StartTime)
		parts = append(parts, fmt.Sprintf("Runtime: %s", runtime.Round(time.Millisecond).String()))
	}

	// Add actions
	var actions string
	if state.IsRunning() {
		actions = "Press 's' to stop, 'r' to restart"
	} else {
		actions = "Press 'Enter' to view logs"
	}
	parts = append(parts, actions)

	return strings.Join(parts, " | ")
}

// PackageManagerItem represents a package manager in settings
type PackageManagerItem struct {
	manager  parser.InstalledPackageManager
	current  bool
	fromJSON bool
}

func (i PackageManagerItem) FilterValue() string { return string(i.manager.Manager) }
func (i PackageManagerItem) Title() string {
	title := string(i.manager.Manager)
	if i.current {
		title = "▶ " + title
	} else {
		title = "  " + title
	}
	if i.fromJSON {
		title += " (from package.json)"
	}
	return title
}
func (i PackageManagerItem) Description() string {
	return fmt.Sprintf("v%s | %s", i.manager.Version, i.manager.Path)
}

// SettingsItem interface for all settings items
type SettingsItem interface {
	list.Item
	isSettingsItem()
}

// PackageManagerSettingsItem wraps PackageManagerItem for settings
type PackageManagerSettingsItem struct {
	PackageManagerItem
}

func (i PackageManagerSettingsItem) isSettingsItem() {}

// MCPInstallItem represents an MCP tool that can be installed
type MCPInstallItem struct {
	tool      mcp.Tool
	installed bool
}

func (i MCPInstallItem) FilterValue() string { return i.tool.Name }
func (i MCPInstallItem) Title() string {
	title := i.tool.Name
	if i.installed {
		title = "✓ " + title
	} else {
		title = "  " + title
	}
	if !i.tool.Supported {
		title += " (experimental)"
	}
	return title
}
func (i MCPInstallItem) Description() string {
	if i.installed {
		return "MCP server installed"
	}
	return "Click to install MCP server"
}
func (i MCPInstallItem) isSettingsItem() {}

// SettingsSectionItem represents a section header in settings
type SettingsSectionItem struct {
	title string
}

func (i SettingsSectionItem) FilterValue() string { return "" }
func (i SettingsSectionItem) Title() string {
	// Add visual styling for section headers
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("238")).
		Padding(0, 1).
		Width(60)
	return style.Render(i.title)
}
func (i SettingsSectionItem) Description() string { return "" }
func (i SettingsSectionItem) isSettingsItem()     {}

// MCPFileBrowserItem represents the file browser option for MCP configs
type MCPFileBrowserItem struct{}

func (i MCPFileBrowserItem) FilterValue() string { return "custom file" }
func (i MCPFileBrowserItem) Title() string       { return "Browse for Custom Config..." }
func (i MCPFileBrowserItem) Description() string {
	return "Browse for a JSON config file to add Brummer"
}
func (i MCPFileBrowserItem) isSettingsItem() {}

// ProxyInfoItem represents proxy information in settings
type ProxyInfoItem struct {
	pacURL string
	mode   proxy.ProxyMode
	port   int
}

func (i ProxyInfoItem) FilterValue() string { return "proxy" }
func (i ProxyInfoItem) Title() string {
	modeStr := "Full Proxy"
	if i.mode == proxy.ProxyModeReverse {
		modeStr = "Reverse Proxy"
	}
	return fmt.Sprintf("🌐 %s (Port %d)", modeStr, i.port)
}
func (i ProxyInfoItem) Description() string {
	return fmt.Sprintf("📄 PAC URL: %s • Press Enter to copy", i.pacURL)
}
func (i ProxyInfoItem) isSettingsItem() {}

// MCPServerInfoItem represents MCP server information in settings
type MCPServerInfoItem struct {
	port   int
	status string
}

func (i MCPServerInfoItem) FilterValue() string { return "mcp server" }
func (i MCPServerInfoItem) Title() string {
	return fmt.Sprintf("🔗 MCP Server (Port %d)", i.port)
}
func (i MCPServerInfoItem) Description() string {
	return fmt.Sprintf("Model Context Protocol server • %s • Multiple tools via single endpoint", i.status)
}
func (i MCPServerInfoItem) isSettingsItem() {}

// InfoDisplayItem represents general information display items
type InfoDisplayItem struct {
	title       string
	description string
	value       string
	copyable    bool
}

func (i InfoDisplayItem) FilterValue() string { return i.title }
func (i InfoDisplayItem) Title() string       { return i.title }
func (i InfoDisplayItem) Description() string {
	if i.copyable {
		return i.description + " (Press Enter to copy)"
	}
	return i.description
}
func (i InfoDisplayItem) isSettingsItem() {}
