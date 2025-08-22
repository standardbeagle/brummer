package components

import (
	"fmt"
	"io"
	"strings"
	"time"
	
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrorsViewComponent adapts the errors view to the component system
type ErrorsViewComponent struct {
	BaseComponent
	
	// Component state
	errors       []ErrorInfo
	selectedID   string
	list         list.Model
	showDetails  bool
	selectedError *ErrorInfo
	
	// Display options
	groupByProcess bool
	showContext    bool
	
	// Styles
	titleStyle    lipgloss.Style
	errorStyle    lipgloss.Style
	contextStyle  lipgloss.Style
	helpStyle     lipgloss.Style
	processStyle  lipgloss.Style
	timestampStyle lipgloss.Style
	countStyle    lipgloss.Style
}

// NewErrorsViewComponent creates a new errors view component
func NewErrorsViewComponent() *ErrorsViewComponent {
	e := &ErrorsViewComponent{
		errors:         []ErrorInfo{},
		groupByProcess: true,
		showContext:    true,
	}
	
	// Initialize styles
	e.initStyles()
	
	// Initialize list
	e.initializeList()
	
	// Set visible by default
	e.visible = true
	
	return e
}

// initStyles initializes the component styles
func (e *ErrorsViewComponent) initStyles() {
	e.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196"))
	
	e.errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))
	
	e.contextStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		PaddingLeft(2)
	
	e.helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	e.processStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	
	e.timestampStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	e.countStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)
}

// initializeList sets up the bubbles list component
func (e *ErrorsViewComponent) initializeList() {
	items := e.errorsToListItems()
	
	e.list = list.New(items, errorItemDelegate{}, 80, 20)
	e.list.Title = "Errors"
	e.list.SetShowStatusBar(true)
	e.list.SetFilteringEnabled(true)
	e.list.SetShowHelp(true)
	
	// Customize key bindings
	e.list.KeyMap.CursorUp = key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	)
	e.list.KeyMap.CursorDown = key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	)
}

// Update handles messages
func (e *ErrorsViewComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Handle component-specific messages
	switch msg := msg.(type) {
	case ErrorOccurredMsg:
		// Add new error
		newError := ErrorInfo{
			ID:          msg.ProcessID + "-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			ProcessID:   msg.ProcessID,
			ProcessName: msg.ProcessName,
			Message:     msg.Error,
			Context:     msg.Context,
			Timestamp:   msg.Timestamp,
			Type:        msg.Type,
			Count:       1,
		}
		
		// Check if similar error exists
		found := false
		for i, err := range e.errors {
			if err.ProcessID == newError.ProcessID && err.Message == newError.Message {
				// Increment count
				e.errors[i].Count++
				e.errors[i].Timestamp = newError.Timestamp
				found = true
				break
			}
		}
		
		if !found {
			e.errors = append(e.errors, newError)
		}
		
		e.updateList()
		return e, SendSystemNotification(NotificationError, 
			fmt.Sprintf("Error in %s", msg.ProcessName), msg.Error)
		
	case ErrorsUpdateMsg:
		// Update entire error list
		e.errors = msg.Errors
		e.updateList()
		return e, nil
		
	case ErrorClearMsg:
		// Clear errors
		if msg.ErrorID != "" {
			// Clear specific error
			newErrors := []ErrorInfo{}
			for _, err := range e.errors {
				if err.ID != msg.ErrorID {
					newErrors = append(newErrors, err)
				}
			}
			e.errors = newErrors
		} else if msg.ProcessID != "" {
			// Clear errors for specific process
			newErrors := []ErrorInfo{}
			for _, err := range e.errors {
				if err.ProcessID != msg.ProcessID {
					newErrors = append(newErrors, err)
				}
			}
			e.errors = newErrors
		} else {
			// Clear all errors
			e.errors = []ErrorInfo{}
		}
		e.updateList()
		e.showDetails = false
		e.selectedError = nil
		return e, nil
		
	case ErrorSelectMsg:
		// Handle error selection
		e.selectedID = msg.ErrorID
		for _, err := range e.errors {
			if err.ID == msg.ErrorID {
				e.selectedError = &err
				e.showDetails = true
				break
			}
		}
		return e, nil
		
	case tea.WindowSizeMsg:
		e.width = msg.Width
		e.height = msg.Height
		e.list.SetSize(msg.Width, msg.Height-4)
		
	case tea.KeyMsg:
		// Handle key presses
		if e.showDetails {
			switch msg.String() {
			case "esc", "q", "enter":
				// Close details view
				e.showDetails = false
				e.selectedError = nil
				return e, nil
				
			case "d":
				// Delete this error
				if e.selectedError != nil {
					return e, func() tea.Msg {
						return ErrorClearMsg{ErrorID: e.selectedError.ID}
					}
				}
				
			case "l":
				// View logs for this process
				if e.selectedError != nil {
					cmds = append(cmds, SendViewChange("logs", e.selectedError.ProcessID))
					cmds = append(cmds, SendLogsFilter("", e.selectedError.ProcessID, "error"))
				}
			}
			return e, tea.Batch(cmds...)
		}
		
		// Handle list navigation
		switch msg.String() {
		case "enter", " ":
			// View error details
			if item := e.list.SelectedItem(); item != nil {
				if ei, ok := item.(errorItem); ok {
					e.selectedError = &ei.error
					e.showDetails = true
				}
			}
			return e, nil
			
		case "c":
			// Clear selected error
			if item := e.list.SelectedItem(); item != nil {
				if ei, ok := item.(errorItem); ok {
					return e, func() tea.Msg {
						return ErrorClearMsg{ErrorID: ei.error.ID}
					}
				}
			}
			
		case "C":
			// Clear all errors
			return e, func() tea.Msg {
				return ErrorClearMsg{}
			}
			
		case "p":
			// Clear errors for process
			if item := e.list.SelectedItem(); item != nil {
				if ei, ok := item.(errorItem); ok {
					return e, func() tea.Msg {
						return ErrorClearMsg{ProcessID: ei.error.ProcessID}
					}
				}
			}
			
		case "g":
			// Toggle grouping
			e.groupByProcess = !e.groupByProcess
			e.updateList()
			return e, nil
			
		case "t":
			// Toggle context
			e.showContext = !e.showContext
			return e, nil
			
		case "l":
			// View logs for selected error's process
			if item := e.list.SelectedItem(); item != nil {
				if ei, ok := item.(errorItem); ok {
					cmds = append(cmds, SendViewChange("logs", ei.error.ProcessID))
					cmds = append(cmds, SendLogsFilter("error", ei.error.ProcessID, ""))
				}
			}
			return e, tea.Batch(cmds...)
		}
	}
	
	// Update the list component
	if e.visible && !e.showDetails {
		newList, cmd := e.list.Update(msg)
		e.list = newList
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	return e, tea.Batch(cmds...)
}

// View renders the component
func (e *ErrorsViewComponent) View() string {
	if !e.visible {
		return ""
	}
	
	// Show details view if selected
	if e.showDetails && e.selectedError != nil {
		return e.renderDetailsView()
	}
	
	// Show list view
	return e.list.View()
}

// renderDetailsView renders the error details view
func (e *ErrorsViewComponent) renderDetailsView() string {
	var s strings.Builder
	
	err := e.selectedError
	
	// Header
	s.WriteString(e.titleStyle.Render("❌ Error Details") + "\n\n")
	
	// Process info
	s.WriteString(e.processStyle.Render("Process: ") + err.ProcessName)
	if err.ProcessID != "" {
		s.WriteString(e.helpStyle.Render(" (" + err.ProcessID + ")"))
	}
	s.WriteString("\n")
	
	// Timestamp
	s.WriteString(e.timestampStyle.Render("Time: ") + 
		err.Timestamp.Format("2006-01-02 15:04:05") + "\n")
	
	// Type
	if err.Type != "" {
		s.WriteString(e.helpStyle.Render("Type: ") + err.Type + "\n")
	}
	
	// Count
	if err.Count > 1 {
		s.WriteString(e.countStyle.Render(fmt.Sprintf("Occurrences: %d", err.Count)) + "\n")
	}
	
	s.WriteString("\n")
	
	// Error message
	s.WriteString(e.errorStyle.Render("Error Message:") + "\n")
	s.WriteString(e.errorStyle.Render(err.Message) + "\n\n")
	
	// Context
	if len(err.Context) > 0 {
		s.WriteString(e.helpStyle.Render("Context:") + "\n")
		for _, line := range err.Context {
			s.WriteString(e.contextStyle.Render(line) + "\n")
		}
		s.WriteString("\n")
	}
	
	// Help
	help := []string{
		"esc: back",
		"d: delete",
		"l: view logs",
	}
	s.WriteString(e.helpStyle.Render(strings.Join(help, " | ")))
	
	return s.String()
}

// updateList updates the list items
func (e *ErrorsViewComponent) updateList() {
	items := e.errorsToListItems()
	e.list.SetItems(items)
	
	// Update title with count
	if len(e.errors) > 0 {
		e.list.Title = fmt.Sprintf("Errors (%d)", len(e.errors))
	} else {
		e.list.Title = "Errors"
	}
}

// errorsToListItems converts errors to list items
func (e *ErrorsViewComponent) errorsToListItems() []list.Item {
	items := make([]list.Item, len(e.errors))
	for i, err := range e.errors {
		items[i] = errorItem{
			error: err,
			index: i,
			showContext: e.showContext,
		}
	}
	return items
}

// Focus sets focus to this component
func (e *ErrorsViewComponent) Focus() tea.Cmd {
	e.focused = true
	return nil
}

// Blur removes focus from this component
func (e *ErrorsViewComponent) Blur() tea.Cmd {
	e.focused = false
	return nil
}

// errorItem implements list.Item
type errorItem struct {
	error       ErrorInfo
	index       int
	showContext bool
}

func (i errorItem) Title() string {
	title := fmt.Sprintf("🔴 %s", i.error.ProcessName)
	if i.error.Count > 1 {
		title += fmt.Sprintf(" (%d)", i.error.Count)
	}
	return title
}

func (i errorItem) Description() string {
	desc := i.error.Message
	if len(desc) > 80 {
		desc = desc[:77] + "..."
	}
	
	// Add timestamp
	timeAgo := time.Since(i.error.Timestamp)
	var timeStr string
	if timeAgo < time.Minute {
		timeStr = fmt.Sprintf("%ds ago", int(timeAgo.Seconds()))
	} else if timeAgo < time.Hour {
		timeStr = fmt.Sprintf("%dm ago", int(timeAgo.Minutes()))
	} else {
		timeStr = fmt.Sprintf("%dh ago", int(timeAgo.Hours()))
	}
	
	return fmt.Sprintf("%s | %s", desc, timeStr)
}

func (i errorItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s", i.error.ProcessName, i.error.Message, i.error.Type)
}

// errorItemDelegate implements list.ItemDelegate
type errorItemDelegate struct{}

func (d errorItemDelegate) Height() int { return 2 }

func (d errorItemDelegate) Spacing() int { return 1 }

func (d errorItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d errorItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(errorItem)
	if !ok {
		return
	}
	
	str := fmt.Sprintf("%s\n  %s", i.Title(), i.Description())
	
	// Create styles inline
	itemStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("196")).
		Bold(true)
	
	var fn func(...string) string
	if index == m.Index() {
		fn = func(strs ...string) string {
			str := strings.Join(strs, "")
			return selectedStyle.Render("> " + str)
		}
	} else {
		fn = itemStyle.Render
	}
	
	fmt.Fprintf(w, "%s", fn(str))
}