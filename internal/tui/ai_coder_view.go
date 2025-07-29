package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/aicoder"
)

// AICoderView handles the AI coder management interface
type AICoderView struct {
	// UI Components
	coderList    list.Model
	detailPanel  viewport.Model
	commandInput textinput.Model
	statusBar    string

	// State
	selectedCoder *aicoder.AICoderProcess
	coders        []*aicoder.AICoderProcess
	manager       *aicoder.AICoderManager

	// Layout
	width       int
	height      int
	listWidth   int
	detailWidth int

	// UI State
	focusMode   FocusMode
	showDetails bool
	commandMode bool
}

type FocusMode int

const (
	FocusList FocusMode = iota
	FocusDetail
	FocusCommand
)

// NewAICoderView creates a new AI coder view
func NewAICoderView(manager *aicoder.AICoderManager) AICoderView {
	// Initialize list component
	coderList := list.New([]list.Item{}, NewAICoderDelegate(), 0, 0)
	coderList.Title = "AI Coders"
	coderList.SetShowStatusBar(true)
	coderList.SetFilteringEnabled(true)
	coderList.Styles.Title = titleStyle

	// Initialize detail panel
	detailPanel := viewport.New(0, 0)

	// Initialize command input
	commandInput := textinput.New()
	commandInput.Placeholder = "Enter command for AI coder..."
	commandInput.CharLimit = 500

	return AICoderView{
		coderList:    coderList,
		detailPanel:  detailPanel,
		commandInput: commandInput,
		manager:      manager,
		focusMode:    FocusList,
		showDetails:  true,
	}
}

// Init initializes the AI coder view
func (v AICoderView) Init() tea.Cmd {
	return tea.Batch(
		v.coderList.StartSpinner(),
		v.refreshCoders(),
	)
}

// Update handles messages for the AI coder view
func (v AICoderView) Update(msg tea.Msg) (AICoderView, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.updateLayout()

	case tea.KeyMsg:

		// Handle command mode
		if v.commandMode {
			switch msg.String() {
			case "esc":
				v.commandMode = false
				v.focusMode = FocusList
				v.commandInput.Blur()
				v.commandInput.SetValue("")
				return v, nil
			case "enter":
				if v.commandInput.Value() != "" && v.selectedCoder != nil {
					command := v.commandInput.Value()
					v.commandInput.SetValue("")
					return v, v.sendCommand(v.selectedCoder.ID, command)
				}
			default:
				v.commandInput, cmd = v.commandInput.Update(msg)
				return v, cmd
			}
		}

		// Normal key handling
		switch msg.String() {
		case "tab":
			if v.showDetails {
				v.focusMode = (v.focusMode + 1) % 2 // Toggle between list and detail
			}
			return v, nil

		// Removed 'n' key - use /ai command instead

		case "d", "delete":
			if v.focusMode == FocusList && v.selectedCoder != nil {
				return v, v.deleteCoder(v.selectedCoder.ID)
			}

		case "s":
			if v.focusMode == FocusList && v.selectedCoder != nil {
				return v, v.startCoder(v.selectedCoder.ID)
			}

		case "p":
			if v.focusMode == FocusList && v.selectedCoder != nil {
				return v, v.pauseCoder(v.selectedCoder.ID)
			}

		case "r":
			if v.focusMode == FocusList && v.selectedCoder != nil {
				return v, v.resumeCoder(v.selectedCoder.ID)
			}

		case "c":
			if v.focusMode == FocusList && v.selectedCoder != nil {
				v.commandMode = true
				v.commandInput.SetValue("")
				v.commandInput.Placeholder = "Enter command for AI coder..."
				v.commandInput.Focus()
				return v, textinput.Blink
			}

		case "enter":
			if v.focusMode == FocusList {
				selected := v.coderList.SelectedItem()
				if item, ok := selected.(AICoderItem); ok {
					v.selectedCoder = item.coder
					v.updateDetailPanel()
				}
			}

		case "?":
			// Show help for AI coder shortcuts
			return v, nil
		}

	case AICoderListUpdatedMsg:
		v.coders = msg.Coders
		v.updateCoderList()

	case AICoderStatusUpdatedMsg:
		v.updateCoderStatus(msg.CoderID, msg.Status)
		if v.selectedCoder != nil && v.selectedCoder.ID == msg.CoderID {
			v.updateDetailPanel()
		}

	case AICoderSelectedMsg:
		if coder := v.findCoder(msg.CoderID); coder != nil {
			v.selectedCoder = coder
			v.updateDetailPanel()
		}

	case AICoderCreatedMsg:
		cmds = append(cmds, v.refreshCoders())

	case AICoderDeletedMsg:
		if v.selectedCoder != nil && v.selectedCoder.ID == msg.CoderID {
			v.selectedCoder = nil
			v.detailPanel.SetContent("")
		}
		cmds = append(cmds, v.refreshCoders())

	case AICoderCommandSentMsg:
		// Handle command result feedback
		if !msg.Success && msg.Error != "" {
			// TODO: Show error message in UI
		}
	}

	// Update components based on focus
	switch v.focusMode {
	case FocusList:
		v.coderList, cmd = v.coderList.Update(msg)
		cmds = append(cmds, cmd)

		// Update selected coder if list selection changed
		if selected := v.coderList.SelectedItem(); selected != nil {
			if item, ok := selected.(AICoderItem); ok {
				v.selectedCoder = item.coder
				v.updateDetailPanel()
			}
		}

	case FocusDetail:
		v.detailPanel, cmd = v.detailPanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// HasActiveDialog returns true if the AI coder view is showing a dialog that should capture input
func (v AICoderView) HasActiveDialog() bool {
	return v.commandMode
}

// View renders the AI coder view
func (v AICoderView) View() string {
	if v.width == 0 {
		return "Loading AI Coder view..."
	}
	
	if v.manager == nil {
		return "AI Coder manager not available - check initialization errors"
	}

	// Build layout
	leftPanel := v.renderLeftPanel()
	rightPanel := v.renderRightPanel()

	// Combine panels
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		rightPanel,
	)

	// Add command input if in command mode
	if v.commandMode {
		commandBox := v.renderCommandInput()
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			content,
			commandBox,
		)
	}

	// Add status bar
	statusBar := v.renderStatusBar()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		statusBar,
	)
}

// Layout helpers

func (v *AICoderView) updateLayout() {
	statusHeight := 1
	commandHeight := 0
	if v.commandMode {
		commandHeight = 3
	}

	availableHeight := v.height - statusHeight - commandHeight

	if v.showDetails {
		v.listWidth = v.width / 3
		v.detailWidth = v.width - v.listWidth - 1
	} else {
		v.listWidth = v.width
		v.detailWidth = 0
	}

	v.coderList.SetSize(v.listWidth, availableHeight)
	v.detailPanel.Width = v.detailWidth
	v.detailPanel.Height = availableHeight
}

func (v *AICoderView) renderLeftPanel() string {
	// Apply focus styling
	listStyle := normalBoxStyle
	if v.focusMode == FocusList {
		listStyle = focusedBoxStyle
	}

	return listStyle.
		Width(v.listWidth).
		Height(v.height - 1). // Leave room for status bar
		Render(v.coderList.View())
}

func (v *AICoderView) renderRightPanel() string {
	if !v.showDetails || v.detailWidth == 0 {
		return ""
	}

	// Apply focus styling
	detailStyle := normalBoxStyle
	if v.focusMode == FocusDetail {
		detailStyle = focusedBoxStyle
	}

	content := v.renderDetailPanel()
	v.detailPanel.SetContent(content)

	return detailStyle.
		Width(v.detailWidth).
		Height(v.height - 1). // Leave room for status bar
		Render(v.detailPanel.View())
}

func (v *AICoderView) renderDetailPanel() string {
	if v.selectedCoder == nil {
		return centerText("No AI coder selected\n\nPress 'n' to create a new AI coder", v.detailWidth, v.height-3)
	}

	coder := v.selectedCoder

	var content strings.Builder

	// Header
	header := fmt.Sprintf("AI Coder: %s", coder.Name)
	if coder.Name == "" {
		header = fmt.Sprintf("AI Coder: %s", coder.SessionID)
	}
	content.WriteString(headerStyle.Width(v.detailWidth - 4).Render(header))
	content.WriteString("\n\n")

	// Status section
	content.WriteString(sectionStyle.Render("Status"))
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("State: %s\n", getStatusColor(coder.Status).Render(string(coder.Status))))
	content.WriteString(fmt.Sprintf("Provider: %s\n", coder.Provider))
	content.WriteString(fmt.Sprintf("Progress: %.1f%%\n", coder.Progress*100))
	if coder.CurrentMessage != "" {
		content.WriteString(fmt.Sprintf("Current: %s\n", coder.CurrentMessage))
	}
	content.WriteString(fmt.Sprintf("Created: %s\n", coder.CreatedAt.Format("2006-01-02 15:04:05")))
	content.WriteString("\n")

	// Task section
	content.WriteString(sectionStyle.Render("Task"))
	content.WriteString("\n")
	content.WriteString(wordWrap(coder.Task, v.detailWidth-4))
	content.WriteString("\n\n")

	// Workspace section
	content.WriteString(sectionStyle.Render("Workspace"))
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("Directory: %s\n", coder.WorkspaceDir))

	// List workspace files (if available)
	if files, err := coder.ListWorkspaceFiles(); err == nil && len(files) > 0 {
		content.WriteString("\nFiles:\n")
		maxFiles := 10
		for i, file := range files {
			if i >= maxFiles {
				content.WriteString(fmt.Sprintf("  ... and %d more files\n", len(files)-maxFiles))
				break
			}
			content.WriteString(fmt.Sprintf("  • %s\n", file))
		}
	}

	// Controls help
	content.WriteString("\n")
	content.WriteString(sectionStyle.Render("Controls"))
	content.WriteString("\n")
	content.WriteString("s: Start  p: Pause  r: Resume  d: Delete  c: Send Command\n")

	return content.String()
}

func (v *AICoderView) renderStatusBar() string {
	// Count coders by status
	statusCounts := make(map[aicoder.AICoderStatus]int)
	for _, coder := range v.coders {
		statusCounts[coder.Status]++
	}

	parts := []string{
		fmt.Sprintf("Total: %d", len(v.coders)),
	}

	if count := statusCounts[aicoder.StatusRunning]; count > 0 {
		parts = append(parts, fmt.Sprintf("Running: %d", count))
	}
	if count := statusCounts[aicoder.StatusPaused]; count > 0 {
		parts = append(parts, fmt.Sprintf("Paused: %d", count))
	}
	if count := statusCounts[aicoder.StatusCompleted]; count > 0 {
		parts = append(parts, fmt.Sprintf("Completed: %d", count))
	}
	if count := statusCounts[aicoder.StatusFailed]; count > 0 {
		parts = append(parts, fmt.Sprintf("Failed: %d", count))
	}

	// Add current focus indicator
	focusText := "List"
	if v.focusMode == FocusDetail {
		focusText = "Detail"
	} else if v.focusMode == FocusCommand || v.commandMode {
		focusText = "Command"
	}
	parts = append(parts, fmt.Sprintf("Focus: %s", focusText))

	status := strings.Join(parts, " | ")
	return statusBarStyle.Width(v.width).Render(status)
}

func (v *AICoderView) renderCommandInput() string {
	return commandInputBoxStyle.
		Width(v.width).
		Render(v.commandInput.View())
}

func (v *AICoderView) renderCreateDialog() string {
	dialog := lipgloss.JoinVertical(
		lipgloss.Center,
		headerStyle.Render("Create New AI Coder"),
		"",
		"Enter the task description:",
		"",
		v.commandInput.View(),
		"",
		dimStyle.Render("Press Enter to create, Esc to cancel"),
	)

	return placeDialog(v.width, v.height, dialogBoxStyle.Render(dialog))
}

// Helper methods

func (v *AICoderView) updateCoderList() {
	items := make([]list.Item, len(v.coders))
	for i, coder := range v.coders {
		items[i] = AICoderItem{coder: coder}
	}
	v.coderList.SetItems(items)
}

func (v *AICoderView) updateCoderStatus(coderID string, status aicoder.AICoderStatus) {
	for _, coder := range v.coders {
		if coder.ID == coderID {
			coder.SetStatus(status)
			break
		}
	}
	v.updateCoderList()
}

func (v *AICoderView) findCoder(coderID string) *aicoder.AICoderProcess {
	for _, coder := range v.coders {
		if coder.ID == coderID {
			return coder
		}
	}
	return nil
}

func (v *AICoderView) updateDetailPanel() {
	if v.selectedCoder != nil && v.showDetails {
		content := v.renderDetailPanel()
		v.detailPanel.SetContent(content)
		v.detailPanel.GotoTop()
	}
}

// Helper functions

func centerText(text string, width, height int) string {
	lines := strings.Split(text, "\n")
	centeredLines := make([]string, len(lines))
	
	for i, line := range lines {
		padding := (width - lipgloss.Width(line)) / 2
		if padding > 0 {
			centeredLines[i] = strings.Repeat(" ", padding) + line
		} else {
			centeredLines[i] = line
		}
	}
	
	content := strings.Join(centeredLines, "\n")
	
	// Vertical centering
	contentHeight := len(lines)
	topPadding := (height - contentHeight) / 2
	if topPadding > 0 {
		content = strings.Repeat("\n", topPadding) + content
	}
	
	return content
}

func placeDialog(width, height int, dialog string) string {
	dialogWidth := lipgloss.Width(dialog)
	dialogHeight := lipgloss.Height(dialog)
	
	// Calculate padding
	horizontalPadding := (width - dialogWidth) / 2
	verticalPadding := (height - dialogHeight) / 2
	
	// Build the centered view
	var result strings.Builder
	
	// Top padding
	for i := 0; i < verticalPadding; i++ {
		result.WriteString("\n")
	}
	
	// Dialog with horizontal padding
	lines := strings.Split(dialog, "\n")
	for _, line := range lines {
		if horizontalPadding > 0 {
			result.WriteString(strings.Repeat(" ", horizontalPadding))
		}
		result.WriteString(line)
		result.WriteString("\n")
	}
	
	return result.String()
}