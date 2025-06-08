package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/beagle/brummer/internal/process"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
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
	// Create a centered container
	containerStyle := lipgloss.NewStyle().
		Width(80).
		Height(30).
		Padding(2, 4).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("226"))
	
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		MarginBottom(2).
		Width(72).
		Align(lipgloss.Center)
	
	title := titleStyle.Render("🐝 Brummer - Select a Script to Run")
	
	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginBottom(2).
		Width(72).
		Align(lipgloss.Center)
	
	instructions := instructionStyle.Render("Type to search • ↑↓ Navigate • Enter to run • / for command palette")
	
	// Input field
	inputStyle := lipgloss.NewStyle().
		Width(72).
		MarginBottom(1)
	
	inputView := inputStyle.Render(m.scriptSelector.View())
	
	// Dropdown suggestions
	dropdownView := m.scriptSelector.RenderScriptSelectorDropdown(10)
	
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
	
	// Available scripts info
	scriptCountStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("242")).
		MarginTop(2).
		Width(72).
		Align(lipgloss.Center)
	
	scriptCount := fmt.Sprintf("%d scripts available", len(m.scriptSelector.availableScripts))
	scriptCountView := scriptCountStyle.Render(scriptCount)
	
	// Combine all elements
	var contentParts []string
	contentParts = append(contentParts, title)
	contentParts = append(contentParts, instructions)
	contentParts = append(contentParts, inputView)
	if dropdownView != "" {
		contentParts = append(contentParts, dropdownView)
	}
	if errorView != "" {
		contentParts = append(contentParts, errorView)
	}
	contentParts = append(contentParts, scriptCountView)
	
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
	if !c.showDropdown || len(c.suggestions) == 0 {
		return ""
	}
	
	var s strings.Builder
	
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("237")).
		Width(72).
		Padding(0, 2)
	
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Width(72).
		Padding(0, 2)
	
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
		
		// Truncate command if too long
		if len(scriptCmd) > 50 {
			scriptCmd = scriptCmd[:47] + "..."
		}
		
		// Format the display
		display := fmt.Sprintf("%-20s %s", scriptName, scriptDescStyle.Render(scriptCmd))
		
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
			Width(72).
			Align(lipgloss.Center)
		s.WriteString("\n")
		moreCount := len(c.suggestions) - maxSuggestions
		s.WriteString(moreStyle.Render(fmt.Sprintf("... and %d more", moreCount)))
	}
	
	return s.String()
}