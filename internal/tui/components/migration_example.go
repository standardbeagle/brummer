package components

import (
	"fmt"
	"strings"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// MigratedProcessView shows how ProcessViewController can be replaced
// with Bubbles components in our component system
type MigratedProcessView struct {
	BaseComponent
	
	// Use Bubbles table for process list
	processTable *BubblesTableAdapter
	
	// Use Bubbles viewport for process output
	outputView *BubblesViewportAdapter
	
	// Layout to arrange them
	layout *BoxLayout
	
	// Data source (would come from process manager)
	processes []ProcessInfo
}

// ProcessInfo represents process data
type ProcessInfo struct {
	ID     string
	Name   string
	Status string
	CPU    string
	Memory string
}

// NewMigratedProcessView creates a process view using Bubbles components
func NewMigratedProcessView() *MigratedProcessView {
	// Create Bubbles table for process list
	columns := []table.Column{
		{Title: "ID", Width: 20},
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 10},
		{Title: "CPU", Width: 10},
		{Title: "Memory", Width: 10},
	}
	
	// Sample data (would come from process manager)
	rows := []table.Row{
		{"dev-1234", "npm run dev", "running", "2.3%", "45MB"},
		{"test-5678", "npm test", "stopped", "0%", "0MB"},
	}
	
	processTable := NewBubblesTableAdapter(columns, rows)
	
	// Create viewport for output
	outputView := NewBubblesViewportAdapter()
	outputView.SetContent("Process output will appear here...")
	
	// Arrange in layout
	layout := NewBoxLayout(Vertical)
	layout.AddChild(processTable, 0.4)  // 40% for process list
	layout.AddChild(outputView, 0.6)    // 60% for output
	
	return &MigratedProcessView{
		BaseComponent: BaseComponent{visible: true},
		processTable: processTable,
		outputView: outputView,
		layout: layout,
	}
}

// Update handles process view updates
func (v *MigratedProcessView) Update(msg tea.Msg) (Component, tea.Cmd) {
	// Could handle process-specific messages here
	// For now, delegate to layout
	newLayout, cmd := v.layout.Update(msg)
	v.layout = newLayout.(*BoxLayout)
	return v, cmd
}

// View renders the process view
func (v *MigratedProcessView) View() string {
	return v.layout.View()
}

// SetSize propagates sizing
func (v *MigratedProcessView) SetSize(width, height int) {
	v.BaseComponent.SetSize(width, height)
	v.layout.SetSize(width, height)
}

// UpdateProcesses updates the process list
func (v *MigratedProcessView) UpdateProcesses(processes []ProcessInfo) {
	rows := make([]table.Row, len(processes))
	for i, p := range processes {
		rows[i] = table.Row{p.ID, p.Name, p.Status, p.CPU, p.Memory}
	}
	// Would update table here
	v.processes = processes
}

// MigratedLogsView shows how LogsViewController can use Bubbles
type MigratedLogsView struct {
	BaseComponent
	
	// Use Bubbles viewport for logs
	viewport *viewport.Model
	
	// Filters using Bubbles list
	filterList *BubblesListAdapter
	
	// Search using Bubbles text input
	searchInput *BubblesTextInputAdapter
	
	// Layout
	layout *BoxLayout
	
	// Log content
	logs []string
	filtered []string
}

// NewMigratedLogsView creates a logs view using Bubbles components
func NewMigratedLogsView() *MigratedLogsView {
	// Create viewport for logs
	vp := viewport.New(80, 20)
	vp.SetContent("Logs loading...")
	
	// Create filter list
	filterItems := []list.Item{
		simpleItem("All"),
		simpleItem("Error"),
		simpleItem("Warning"),
		simpleItem("Info"),
		simpleItem("Debug"),
	}
	filterList := NewBubblesListAdapter("Log Level", filterItems)
	
	// Create search input
	searchInput := NewBubblesTextInputAdapter("Search logs...", nil)
	
	// Create layout
	layout := NewBoxLayout(Horizontal)
	
	// Left side: filters and search
	leftPanel := NewBoxLayout(Vertical)
	leftPanel.AddChild(filterList, 0.7)
	leftPanel.AddChild(searchInput, 0.3)
	
	// Add to main layout
	layout.AddChild(leftPanel, 0.3)    // 30% for controls
	layout.AddChild(&viewportAdapter{viewport: &vp}, 0.7)  // 70% for logs
	
	return &MigratedLogsView{
		BaseComponent: BaseComponent{visible: true},
		viewport: &vp,
		filterList: filterList,
		searchInput: searchInput,
		layout: layout,
		logs: []string{},
		filtered: []string{},
	}
}

// Update handles logs view updates
func (v *MigratedLogsView) Update(msg tea.Msg) (Component, tea.Cmd) {
	// Delegate to layout
	newLayout, cmd := v.layout.Update(msg)
	v.layout = newLayout.(*BoxLayout)
	return v, cmd
}

// View renders the logs view
func (v *MigratedLogsView) View() string {
	return v.layout.View()
}

// SetSize propagates sizing
func (v *MigratedLogsView) SetSize(width, height int) {
	v.BaseComponent.SetSize(width, height)
	v.layout.SetSize(width, height)
}

// Focus focuses the logs view
func (v *MigratedLogsView) Focus() tea.Cmd {
	v.focused = true
	return nil
}

// Blur blurs the logs view
func (v *MigratedLogsView) Blur() tea.Cmd {
	v.focused = false
	return nil
}

// viewportAdapter wraps a raw viewport.Model
type viewportAdapter struct {
	BaseComponent
	viewport *viewport.Model
}

func (a *viewportAdapter) Update(msg tea.Msg) (Component, tea.Cmd) {
	newVP, cmd := a.viewport.Update(msg)
	a.viewport = &newVP
	return a, cmd
}

func (a *viewportAdapter) View() string {
	return a.viewport.View()
}

func (a *viewportAdapter) SetSize(width, height int) {
	a.BaseComponent.SetSize(width, height)
	a.viewport.Width = width
	a.viewport.Height = height
}

// MigratedErrorsView demonstrates error view with Bubbles
type MigratedErrorsView struct {
	BaseComponent
	
	// Error list using Bubbles list with custom rendering
	errorList *list.Model
	
	// Details viewport
	detailsView *viewport.Model
	
	// Layout
	layout *BoxLayout
}

// ErrorItem represents an error for the list
type ErrorItem struct {
	Level     string
	Message   string
	File      string
	Line      int
	Timestamp string
}

func (e ErrorItem) Title() string       { return e.Message }
func (e ErrorItem) Description() string { return fmt.Sprintf("%s:%d - %s", e.File, e.Line, e.Timestamp) }
func (e ErrorItem) FilterValue() string { return e.Message }

// NewMigratedErrorsView creates an errors view using Bubbles
func NewMigratedErrorsView() *MigratedErrorsView {
	// Create custom delegate for error rendering
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("196")).
		BorderForeground(lipgloss.Color("196"))
	
	// Create error list
	errorItems := []list.Item{
		ErrorItem{
			Level: "error",
			Message: "Connection timeout",
			File: "server.js",
			Line: 42,
			Timestamp: "10:23:45",
		},
		ErrorItem{
			Level: "warning",
			Message: "Deprecated API usage",
			File: "api.js",
			Line: 17,
			Timestamp: "10:24:12",
		},
	}
	
	errorList := list.New(errorItems, delegate, 0, 0)
	errorList.Title = "Errors & Warnings"
	
	// Create details viewport
	detailsView := viewport.New(80, 10)
	detailsView.SetContent("Select an error to see details...")
	
	// Layout
	layout := NewBoxLayout(Vertical)
	layout.AddChild(&listAdapter{list: &errorList}, 0.6)
	layout.AddChild(&viewportAdapter{viewport: &detailsView}, 0.4)
	
	return &MigratedErrorsView{
		BaseComponent: BaseComponent{visible: true},
		errorList: &errorList,
		detailsView: &detailsView,
		layout: layout,
	}
}

// Update handles errors view updates
func (v *MigratedErrorsView) Update(msg tea.Msg) (Component, tea.Cmd) {
	// Delegate to layout
	newLayout, cmd := v.layout.Update(msg)
	v.layout = newLayout.(*BoxLayout)
	return v, cmd
}

// View renders the errors view
func (v *MigratedErrorsView) View() string {
	return v.layout.View()
}

// SetSize propagates sizing
func (v *MigratedErrorsView) SetSize(width, height int) {
	v.BaseComponent.SetSize(width, height)
	v.layout.SetSize(width, height)
}

// Focus focuses the errors view
func (v *MigratedErrorsView) Focus() tea.Cmd {
	v.focused = true
	return nil
}

// Blur blurs the errors view
func (v *MigratedErrorsView) Blur() tea.Cmd {
	v.focused = false
	return nil
}

// listAdapter wraps a raw list.Model
type listAdapter struct {
	BaseComponent
	list *list.Model
}

func (a *listAdapter) Update(msg tea.Msg) (Component, tea.Cmd) {
	newList, cmd := a.list.Update(msg)
	a.list = &newList
	return a, cmd
}

func (a *listAdapter) View() string {
	return a.list.View()
}

func (a *listAdapter) SetSize(width, height int) {
	a.BaseComponent.SetSize(width, height)
	a.list.SetSize(width, height)
}

// MigrationBenefits documents the advantages of using Bubbles components
type MigrationBenefits struct{}

func (m MigrationBenefits) Document() string {
	return strings.Join([]string{
		"Benefits of using Bubbles components in our architecture:",
		"",
		"1. REDUCED COMPLEXITY",
		"   - No need to handle scrolling logic (viewport does it)",
		"   - No need to handle text input edge cases (textinput does it)",
		"   - No need to handle list filtering (list does it)",
		"   - No need to handle table navigation (table does it)",
		"",
		"2. BATTLE-TESTED FUNCTIONALITY",
		"   - Viewport: Smooth scrolling, proper line wrapping",
		"   - List: Built-in filtering, keyboard navigation, pagination",
		"   - Table: Row selection, column sizing, scrolling",
		"   - TextInput: Cursor movement, text selection, IME support",
		"   - Spinner: Multiple styles, smooth animation",
		"   - Progress: Gradient support, percentage display",
		"",
		"3. CONSISTENT BEHAVIOR",
		"   - All Bubbles components follow BubbleTea patterns",
		"   - Predictable Update/View cycle",
		"   - Standard keyboard shortcuts users expect",
		"   - Proper focus management",
		"",
		"4. EASY MIGRATION PATH",
		"   - ViewAdapter can wrap existing controllers",
		"   - BubblesAdapter patterns wrap Bubbles components",
		"   - Gradual migration possible (mix old and new)",
		"   - No need to rewrite everything at once",
		"",
		"5. COMPOSITION FRIENDLY",
		"   - Bubbles components work in our layouts",
		"   - Can be used in overlays and windows",
		"   - Support our Component interface",
		"   - Work with FocusChain for tab navigation",
		"",
		"6. MAINTAINED BY COMMUNITY",
		"   - Active development and bug fixes",
		"   - Growing ecosystem of components",
		"   - Well-documented APIs",
		"   - Examples and tutorials available",
	}, "\n")
}

// ExampleMigrationStrategy shows how to migrate incrementally
func ExampleMigrationStrategy() Component {
	// Phase 1: Wrap existing controllers with ViewAdapter
	// (Already done)
	
	// Phase 2: Replace internals with Bubbles components
	// Start with simple views like URLs, Settings
	
	// Phase 3: Migrate complex views
	// ProcessView, LogsView using composition
	
	// Phase 4: Add new features using Bubbles
	// Command palette with list
	// Search with textinput
	// Progress bars for long operations
	
	// Example of mixed old/new
	layout := NewBoxLayout(Vertical)
	
	// Old controller wrapped
	// layout.AddChild(NewViewAdapter(existingController), 0.5)
	
	// New Bubbles-based view
	layout.AddChild(NewMigratedProcessView(), 0.5)
	
	return layout
}

// RealWorldExample shows a complete view using Bubbles
func RealWorldExample() Component {
	// Main layout
	main := NewBoxLayout(Horizontal)
	
	// Sidebar with navigation
	sidebar := NewBoxLayout(Vertical)
	
	// Navigation list
	navItems := []list.Item{
		simpleItem("Processes"),
		simpleItem("Logs"),
		simpleItem("Errors"),
		simpleItem("URLs"),
		simpleItem("Settings"),
	}
	nav := NewBubblesListAdapter("Views", navItems)
	sidebar.AddChild(nav, 0.3)
	
	// Quick actions
	actionItems := []list.Item{
		simpleItem("Start All"),
		simpleItem("Stop All"),
		simpleItem("Clear Logs"),
		simpleItem("Refresh"),
	}
	actions := NewBubblesListAdapter("Actions", actionItems)
	sidebar.AddChild(actions, 0.3)
	
	// Status
	status := NewBubblesProgressAdapter("CPU Usage")
	status.SetPercent(0.45)
	sidebar.AddChild(status, 0.4)
	
	// Content area with view switcher
	content := NewViewSwitcher()
	content.AddView("processes", NewMigratedProcessView())
	content.AddView("logs", NewMigratedLogsView())
	content.AddView("errors", NewMigratedErrorsView())
	
	// Compose
	main.AddChild(sidebar, 0.2)   // 20% for sidebar
	main.AddChild(content, 0.8)   // 80% for content
	
	// Add overlay support
	root := NewStackLayout()
	root.AddLayer(main)
	
	// Command palette overlay (hidden by default)
	cmdPalette := NewCommandPalette(nil)
	root.AddLayer(cmdPalette)
	
	return root
}