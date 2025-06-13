package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/process"
)

// NewScriptSelectorAutocomplete creates an autocomplete for initial script selection
func NewScriptSelectorAutocomplete(scripts map[string]string) CommandAutocomplete {
	ti := textinput.New()
	ti.Placeholder = "Type to search scripts..."
	ti.Prompt = ""
	ti.Focus()
	ti.CharLimit = 100

	c := CommandAutocomplete{
		input:            ti,
		availableScripts: scripts,
		showDropdown:     true,
	}

	// Initialize with all scripts as suggestions
	c.updateScriptSelectorSuggestions()

	return c
}

// NewScriptSelectorAutocompleteWithProcessManager creates a script selector with process manager
func NewScriptSelectorAutocompleteWithProcessManager(scripts map[string]string, processMgr *process.Manager) CommandAutocomplete {
	c := NewScriptSelectorAutocomplete(scripts)
	c.processMgr = processMgr
	// Re-update suggestions with process manager filter
	c.updateScriptSelectorSuggestions()
	return c
}

// updateScriptSelectorSuggestions updates suggestions for script selector mode
func (c *CommandAutocomplete) updateScriptSelectorSuggestions() {
	value := strings.ToLower(c.input.Value())

	// Get running scripts if process manager is available
	runningScripts := make(map[string]bool)
	if c.processMgr != nil {
		for _, proc := range c.processMgr.GetAllProcesses() {
			if proc.Status == process.StatusRunning {
				runningScripts[proc.Name] = true
			}
		}
	}

	// Get all scripts that aren't already running
	scripts := make([]string, 0, len(c.availableScripts))
	for name := range c.availableScripts {
		if !runningScripts[name] {
			scripts = append(scripts, name)
		}
	}
	sort.Strings(scripts)

	// Filter based on input
	if value == "" {
		c.suggestions = scripts
	} else {
		c.suggestions = []string{}
		for _, script := range scripts {
			if strings.Contains(strings.ToLower(script), value) {
				c.suggestions = append(c.suggestions, script)
			}
		}
	}

	c.selected = 0
	c.showDropdown = len(c.suggestions) > 0
}

// RenderScriptSelector renders the script selector view
func (m Model) renderScriptSelector() string {
	// Adaptive sizing based on terminal size
	var containerWidth, containerHeight int
	var showSkipSection bool
	var padding int

	// Determine container size based on terminal dimensions
	if m.width < 40 {
		containerWidth = m.width - 4
		padding = 1
	} else if m.width < 80 {
		containerWidth = min(70, m.width-8)
		padding = 2
	} else {
		containerWidth = 80
		padding = 2
	}

	if m.height < 12 {
		containerHeight = m.height - 2
		showSkipSection = false
	} else if m.height < 20 {
		containerHeight = min(15, m.height-2)
		showSkipSection = true
	} else {
		containerHeight = 18 // Much more compact max height
		showSkipSection = true
	}

	// Create a centered container with adaptive sizing
	containerStyle := lipgloss.NewStyle().
		Width(containerWidth).
		Height(containerHeight).
		Padding(padding, padding).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("226"))

	// Calculate content width
	contentWidth := containerWidth - (2*padding + 2) // Account for padding and border

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		MarginBottom(1). // Reduced from 2
		Width(contentWidth).
		Align(lipgloss.Center)

	var title string
	if m.height < 12 {
		title = titleStyle.Render("🐝 Select Script")
	} else {
		title = titleStyle.Render("🐝 Brummer - Select a Script to Run")
	}

	// Skip scripts section (conditional)
	var skipSection string
	if showSkipSection {
		skipStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			MarginBottom(0). // Reduced from 1
			Width(contentWidth).
			Align(lipgloss.Center)

		if m.scriptSelector.arbitraryMode {
			if containerWidth < 60 {
				skipSection = skipStyle.Render("🚀 Arbitrary Command Mode")
			} else {
				skipSection = skipStyle.Render("🚀 Arbitrary Command Mode - Type any command to run")
			}
		} else {
			if containerWidth < 60 {
				skipSection = skipStyle.Render("💡 Ctrl+S: skip • Ctrl+N: arbitrary • /: commands")
			} else {
				skipSection = skipStyle.Render("💡 Skip Scripts: Ctrl+S to skip, Ctrl+N for arbitrary commands, or / for command palette")
			}
		}
	}

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginBottom(1). // Reduced from 2
		Width(contentWidth).
		Align(lipgloss.Center)

	var instructions string
	if m.scriptSelector.arbitraryMode {
		if containerWidth < 50 {
			instructions = instructionStyle.Render("Enter: run command • Esc: exit")
		} else {
			instructions = instructionStyle.Render("Type any command (e.g., 'ls', 'node server.js') • Enter to run • Esc to exit")
		}
	} else {
		if containerWidth < 50 {
			instructions = instructionStyle.Render("↑↓ Navigate • Enter: run • Esc: exit")
		} else {
			instructions = instructionStyle.Render("Type script name or ↑↓ Navigate • Enter to run script • Esc/Ctrl+C to exit")
		}
	}

	// Input field
	inputStyle := lipgloss.NewStyle().
		Width(contentWidth).
		MarginBottom(1)

	inputView := inputStyle.Render(m.scriptSelector.View())

	// Dropdown suggestions with proper width (hide in arbitrary mode)
	var dropdownView string
	if !m.scriptSelector.arbitraryMode {
		dropdownView = m.scriptSelector.RenderScriptSelectorDropdownWithWidth(6, contentWidth) // Reduced from 10 to 6
	}

	// Error message if any
	errorMsg := m.scriptSelector.GetErrorMessage()
	errorView := ""
	if errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			MarginTop(1)
		errorView = errorStyle.Render("⚠ " + errorMsg)
	}

	// Available scripts info removed to save space

	// Combine all elements
	var contentParts []string
	contentParts = append(contentParts, title)
	if showSkipSection && skipSection != "" {
		contentParts = append(contentParts, skipSection)
	}
	contentParts = append(contentParts, instructions)
	contentParts = append(contentParts, inputView)
	if dropdownView != "" {
		contentParts = append(contentParts, dropdownView)
	}
	if errorView != "" {
		contentParts = append(contentParts, errorView)
	}
	// Script count section removed to save vertical space

	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)
	container := containerStyle.Render(content)

	// Center the container on screen
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		container,
	)
}

// RenderScriptSelectorDropdown renders the dropdown for script selector
func (c CommandAutocomplete) RenderScriptSelectorDropdown(maxSuggestions int) string {
	return c.RenderScriptSelectorDropdownWithWidth(maxSuggestions, c.width)
}

// RenderScriptSelectorDropdownWithWidth renders the dropdown with specific width
func (c CommandAutocomplete) RenderScriptSelectorDropdownWithWidth(maxSuggestions int, containerWidth int) string {
	if !c.showDropdown || len(c.suggestions) == 0 {
		return ""
	}

	var s strings.Builder

	// Use dynamic width based on container, with sensible limits
	dropdownWidth := containerWidth
	if dropdownWidth < 20 {
		dropdownWidth = 20
	}
	if dropdownWidth > 80 {
		dropdownWidth = 80
	}

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("237")).
		Width(dropdownWidth).
		Padding(0, 1)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Width(dropdownWidth).
		Padding(0, 1)

	scriptDescStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("242")).
		Italic(true)

	count := len(c.suggestions)
	if count > maxSuggestions {
		count = maxSuggestions
	}

	for i := 0; i < count; i++ {
		scriptName := c.suggestions[i]
		scriptCmd := c.availableScripts[scriptName]

		// Calculate available space for script command
		prefixLength := 3 // "▶ " or "  "
		nameLength := len(scriptName)
		paddingLength := 4 // padding (2 chars each side)
		spacerLength := 2  // space between name and command

		availableForCmd := dropdownWidth - prefixLength - nameLength - paddingLength - spacerLength
		if availableForCmd < 0 {
			availableForCmd = 0
		}

		// Truncate command if necessary
		if len(scriptCmd) > availableForCmd && availableForCmd > 3 {
			scriptCmd = scriptCmd[:availableForCmd-3] + "..."
		} else if len(scriptCmd) > availableForCmd {
			scriptCmd = "" // Hide command if no space
		}

		// Format the display without fixed width formatting
		var display string
		if scriptCmd != "" {
			display = fmt.Sprintf("%s  %s", scriptName, scriptDescStyle.Render(scriptCmd))
		} else {
			display = scriptName
		}

		if i == c.selected {
			s.WriteString(selectedStyle.Render("▶ " + display))
		} else {
			s.WriteString(normalStyle.Render("  " + display))
		}

		if i < count-1 {
			s.WriteString("\n")
		}
	}

	// Show more indicator if there are more suggestions
	if len(c.suggestions) > maxSuggestions {
		moreStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true).
			Width(dropdownWidth).
			Align(lipgloss.Center)
		s.WriteString("\n")
		moreCount := len(c.suggestions) - maxSuggestions
		s.WriteString(moreStyle.Render(fmt.Sprintf("... and %d more", moreCount)))
	}

	return s.String()
}
