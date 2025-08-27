package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BubblesListAdapter wraps a Bubbles list component
type BubblesListAdapter struct {
	BaseComponent
	list  list.Model
	items []list.Item
}

// NewBubblesListAdapter creates an adapter for a Bubbles list
func NewBubblesListAdapter(title string, items []list.Item) *BubblesListAdapter {
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	return &BubblesListAdapter{
		BaseComponent: BaseComponent{visible: true},
		list:          l,
		items:         items,
	}
}

// Update delegates to the Bubbles list
func (a *BubblesListAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Bubbles components handle their own sizing
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.SetSize(msg.Width, msg.Height)
		a.list.SetSize(msg.Width, msg.Height)
	}

	newList, cmd := a.list.Update(msg)
	a.list = newList
	return a, cmd
}

// View delegates to the Bubbles list
func (a *BubblesListAdapter) View() string {
	if !a.visible {
		return ""
	}
	return a.list.View()
}

// SetSize updates the list dimensions
func (a *BubblesListAdapter) SetSize(width, height int) {
	a.BaseComponent.SetSize(width, height)
	a.list.SetSize(width, height)
}

// Focus focuses the list
func (a *BubblesListAdapter) Focus() tea.Cmd {
	a.focused = true
	return nil
}

// Blur blurs the list
func (a *BubblesListAdapter) Blur() tea.Cmd {
	a.focused = false
	return nil
}

// BubblesTableAdapter wraps a Bubbles table component
type BubblesTableAdapter struct {
	BaseComponent
	table table.Model
}

// NewBubblesTableAdapter creates an adapter for a Bubbles table
func NewBubblesTableAdapter(columns []table.Column, rows []table.Row) *BubblesTableAdapter {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Style the table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return &BubblesTableAdapter{
		BaseComponent: BaseComponent{visible: true},
		table:         t,
	}
}

// Update delegates to the Bubbles table
func (a *BubblesTableAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newTable, cmd := a.table.Update(msg)
	a.table = newTable
	return a, cmd
}

// View delegates to the Bubbles table
func (a *BubblesTableAdapter) View() string {
	if !a.visible {
		return ""
	}
	return a.table.View()
}

// SetSize updates the table dimensions
func (a *BubblesTableAdapter) SetSize(width, height int) {
	a.BaseComponent.SetSize(width, height)
	a.table.SetHeight(height - 2) // Account for borders
}

// BubblesViewportAdapter wraps a Bubbles viewport for scrollable content
type BubblesViewportAdapter struct {
	BaseComponent
	viewport viewport.Model
	content  string
}

// NewBubblesViewportAdapter creates an adapter for a Bubbles viewport
func NewBubblesViewportAdapter() *BubblesViewportAdapter {
	vp := viewport.New(80, 20)
	vp.SetContent("Loading...")

	return &BubblesViewportAdapter{
		BaseComponent: BaseComponent{visible: true},
		viewport:      vp,
	}
}

// SetContent updates the viewport content
func (a *BubblesViewportAdapter) SetContent(content string) {
	a.content = content
	a.viewport.SetContent(content)
}

// Update delegates to the Bubbles viewport
func (a *BubblesViewportAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.SetSize(msg.Width, msg.Height)
	}

	newViewport, cmd := a.viewport.Update(msg)
	a.viewport = newViewport
	return a, cmd
}

// View delegates to the Bubbles viewport
func (a *BubblesViewportAdapter) View() string {
	if !a.visible {
		return ""
	}
	return a.viewport.View()
}

// SetSize updates viewport dimensions
func (a *BubblesViewportAdapter) SetSize(width, height int) {
	a.BaseComponent.SetSize(width, height)
	a.viewport.Width = width
	a.viewport.Height = height
}

// BubblesTextInputAdapter wraps a Bubbles text input
type BubblesTextInputAdapter struct {
	BaseComponent
	input    textinput.Model
	onSubmit func(string) tea.Cmd
}

// NewBubblesTextInputAdapter creates an adapter for a Bubbles text input
func NewBubblesTextInputAdapter(placeholder string, onSubmit func(string) tea.Cmd) *BubblesTextInputAdapter {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 156
	ti.Width = 50

	return &BubblesTextInputAdapter{
		BaseComponent: BaseComponent{visible: true},
		input:         ti,
		onSubmit:      onSubmit,
	}
}

// Update delegates to the Bubbles text input
func (a *BubblesTextInputAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle enter key for submission
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
		if a.onSubmit != nil && a.input.Value() != "" {
			cmd := a.onSubmit(a.input.Value())
			a.input.SetValue("")
			return a, cmd
		}
	}

	newInput, cmd := a.input.Update(msg)
	a.input = newInput
	return a, cmd
}

// View delegates to the Bubbles text input
func (a *BubblesTextInputAdapter) View() string {
	if !a.visible {
		return ""
	}
	return a.input.View()
}

// Focus focuses the input
func (a *BubblesTextInputAdapter) Focus() tea.Cmd {
	a.focused = true
	return a.input.Focus()
}

// Blur blurs the input
func (a *BubblesTextInputAdapter) Blur() tea.Cmd {
	a.focused = false
	a.input.Blur()
	return nil
}

// BubblesSpinnerAdapter wraps a Bubbles spinner for loading states
type BubblesSpinnerAdapter struct {
	BaseComponent
	spinner spinner.Model
	message string
}

// NewBubblesSpinnerAdapter creates an adapter for a Bubbles spinner
func NewBubblesSpinnerAdapter(message string) *BubblesSpinnerAdapter {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &BubblesSpinnerAdapter{
		BaseComponent: BaseComponent{visible: true},
		spinner:       s,
		message:       message,
	}
}

// Update delegates to the Bubbles spinner
func (a *BubblesSpinnerAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newSpinner, cmd := a.spinner.Update(msg)
	a.spinner = newSpinner
	return a, cmd
}

// View renders the spinner with message
func (a *BubblesSpinnerAdapter) View() string {
	if !a.visible {
		return ""
	}
	return fmt.Sprintf("%s %s", a.spinner.View(), a.message)
}

// Init starts the spinner animation
func (a *BubblesSpinnerAdapter) Init() tea.Cmd {
	return a.spinner.Tick
}

// BubblesProgressAdapter wraps a Bubbles progress bar
type BubblesProgressAdapter struct {
	BaseComponent
	progress progress.Model
	percent  float64
	label    string
}

// NewBubblesProgressAdapter creates an adapter for a Bubbles progress bar
func NewBubblesProgressAdapter(label string) *BubblesProgressAdapter {
	p := progress.New(progress.WithDefaultGradient())

	return &BubblesProgressAdapter{
		BaseComponent: BaseComponent{visible: true},
		progress:      p,
		label:         label,
	}
}

// SetPercent updates the progress percentage
func (a *BubblesProgressAdapter) SetPercent(percent float64) {
	a.percent = percent
}

// Update handles progress updates
func (a *BubblesProgressAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Progress bars typically update via external commands
	// This is a simplified version
	return a, nil
}

// View renders the progress bar with label
func (a *BubblesProgressAdapter) View() string {
	if !a.visible {
		return ""
	}

	progressView := a.progress.ViewAs(a.percent)
	return fmt.Sprintf("%s\n%s %.0f%%", a.label, progressView, a.percent*100)
}

// SetSize updates progress bar width
func (a *BubblesProgressAdapter) SetSize(width, height int) {
	a.BaseComponent.SetSize(width, height)
	a.progress.Width = width - 10 // Leave room for percentage
}

// Init implements tea.Model interface for BubblesProgressAdapter
func (*BubblesProgressAdapter) Init() tea.Cmd { return nil }

// CompositeExample shows how to compose multiple Bubbles components
type CompositeExample struct {
	BaseComponent
	layout   *BoxLayout
	list     *BubblesListAdapter
	input    *BubblesTextInputAdapter
	viewport *BubblesViewportAdapter
}

// NewCompositeExample creates a composite using multiple Bubbles components
func NewCompositeExample() *CompositeExample {
	// Create Bubbles components
	items := []list.Item{
		simpleItem("Item 1"),
		simpleItem("Item 2"),
		simpleItem("Item 3"),
	}
	listAdapter := NewBubblesListAdapter("Select an item", items)

	inputAdapter := NewBubblesTextInputAdapter("Type a command...", nil)

	viewportAdapter := NewBubblesViewportAdapter()
	viewportAdapter.SetContent("This is scrollable content\nLine 2\nLine 3\nLine 4\nLine 5")

	// Compose them in a layout
	layout := NewBoxLayout(Vertical)
	layout.AddChild(listAdapter, 0.5)     // 50% height for list
	layout.AddChild(viewportAdapter, 0.3) // 30% for viewport
	layout.AddChild(inputAdapter, 0.2)    // 20% for input

	return &CompositeExample{
		BaseComponent: BaseComponent{visible: true},
		layout:        layout,
		list:          listAdapter,
		input:         inputAdapter,
		viewport:      viewportAdapter,
	}
}

// Update delegates to the layout
func (c *CompositeExample) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newLayout, cmd := c.layout.Update(msg)
	c.layout = newLayout.(*BoxLayout)
	return c, cmd
}

// View delegates to the layout
func (c *CompositeExample) View() string {
	return c.layout.View()
}

// SetSize propagates to layout
func (c *CompositeExample) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.layout.SetSize(width, height)
}

// Init implements tea.Model interface for CompositeExample
func (*CompositeExample) Init() tea.Cmd { return nil }

// simpleItem is a simple list item for examples
type simpleItem string

func (i simpleItem) FilterValue() string { return string(i) }
func (i simpleItem) Title() string       { return string(i) }
func (i simpleItem) Description() string { return "" }

// ValidateBubblesIntegration demonstrates that our component system
// works seamlessly with Bubbles components
func ValidateBubblesIntegration() {
	// 1. Individual Bubbles components work as Components
	var _ Component = &BubblesListAdapter{}
	var _ Component = &BubblesTableAdapter{}
	var _ Component = &BubblesViewportAdapter{}
	var _ Component = &BubblesTextInputAdapter{}
	var _ Component = &BubblesSpinnerAdapter{}
	var _ Component = &BubblesProgressAdapter{}

	// 2. Bubbles components work with our layouts
	var _ Component = &CompositeExample{}

	// 3. Bubbles components support our traits
	var _ Sizable = &BubblesListAdapter{}
	var _ Visible = &BubblesViewportAdapter{}

	// This proves our architecture is compatible with Bubbles
}

// CreateSearchableList creates a filterable list using Bubbles
func CreateSearchableList(title string, items []string) Component {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = simpleItem(item)
	}
	return NewBubblesListAdapter(title, listItems)
}

// CreateScrollableLog creates a scrollable log viewer using Bubbles viewport
func CreateScrollableLog(content string) Component {
	vp := NewBubblesViewportAdapter()
	vp.SetContent(content)
	return vp
}

// CreateCommandInput creates a command input using Bubbles text input
func CreateCommandInput(onSubmit func(string) tea.Cmd) Component {
	return NewBubblesTextInputAdapter("Enter command...", onSubmit)
}

// CreateLoadingSpinner creates a loading indicator using Bubbles spinner
func CreateLoadingSpinner(message string) Component {
	return NewBubblesSpinnerAdapter(message)
}

// CreateProgressBar creates a progress indicator using Bubbles progress
func CreateProgressBar(label string, percent float64) Component {
	pb := NewBubblesProgressAdapter(label)
	pb.SetPercent(percent)
	return pb
}

// Example: Using Bubbles components in our layouts
func ExampleBubblesInLayout() Component {
	// Create a vertical layout
	layout := NewBoxLayout(Vertical)

	// Add Bubbles components
	layout.AddChild(CreateSearchableList("Processes", []string{
		"npm run dev",
		"npm test",
		"npm run build",
	}), 0.4)

	layout.AddChild(CreateScrollableLog(strings.Repeat("Log line\n", 100)), 0.4)

	layout.AddChild(CreateCommandInput(func(cmd string) tea.Cmd {
		return nil // Handle command
	}), 0.2)

	return layout
}

// Example: Bubbles components in overlays
func ExampleBubblesInOverlay() Component {
	// Base layer with content
	base := CreateScrollableLog("Main content...")

	// Overlay with Bubbles list
	overlay := NewWindow("Select Action", CreateSearchableList("Actions", []string{
		"Run Script",
		"Stop Process",
		"View Logs",
		"Clear Cache",
	}))
	overlay.SetModal(true)
	overlay.SetSize(60, 20)

	// Stack them
	stack := NewStackLayout()
	stack.AddLayer(base)
	stack.AddLayer(overlay)

	return stack
}

// Init implements tea.Model interface for BubblesListAdapter
func (*BubblesListAdapter) Init() tea.Cmd { return nil }

// Init implements tea.Model interface for BubblesTableAdapter
func (*BubblesTableAdapter) Init() tea.Cmd { return nil }

// Init implements tea.Model interface for BubblesTextInputAdapter
func (*BubblesTextInputAdapter) Init() tea.Cmd { return nil }

// Init implements tea.Model interface for BubblesViewportAdapter
func (*BubblesViewportAdapter) Init() tea.Cmd { return nil }
