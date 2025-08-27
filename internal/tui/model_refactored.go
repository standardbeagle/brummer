package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/internal/tui/navigation"
)

// RefactoredModel is a thin orchestrator using the component system
type RefactoredModel struct {
	// Component tree
	root         components.Component
	viewSwitcher *components.ViewSwitcher

	// Current state
	width  int
	height int

	// Keep existing data communication unchanged
	model *Model // Reference to existing model for data
}

// NewRefactoredModel creates a model using the component system
func NewRefactoredModel(existingModel *Model) *RefactoredModel {
	// Build the UI tree
	builder := components.NewUIBuilder()

	// Set header and footer using existing layout controller
	// Note: layoutController would need to implement HeaderRenderer/FooterRenderer
	// For now, we'll skip this
	// if existingModel.layoutController != nil {
	//	builder.SetHeader(existingModel.layoutController)
	//	builder.SetFooter(existingModel.layoutController)
	// }

	// Add all views as adapters
	builder.AddView(navigation.ViewProcesses, existingModel.processViewController)
	builder.AddView(navigation.ViewLogs, existingModel.logsViewController)
	builder.AddView(navigation.ViewErrors, existingModel.errorsViewController)
	builder.AddView(navigation.ViewURLs, existingModel.urlsViewController)
	builder.AddView(navigation.ViewWeb, existingModel.webViewController)
	builder.AddView(navigation.ViewSettings, existingModel.settingsController)

	// Add overlays (will implement these as components later)
	// builder.AddOverlay(systemPanelComponent)
	// builder.AddOverlay(commandPaletteComponent)

	// Build the tree
	root := builder.Build()

	return &RefactoredModel{
		root:         root,
		viewSwitcher: components.ExtractViewSwitcher(root),
		model:        existingModel,
	}
}

// Init initializes the model
func (m *RefactoredModel) Init() tea.Cmd {
	// Delegate to existing model
	return m.model.Init()
}

// Update handles all messages through the component tree
func (m *RefactoredModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle data updates through existing model
	// This keeps all data communication unchanged
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	// Handle view change messages if available
	// case navigation.ChangeViewMsg:
	//	// Switch view using component system
	//	if m.viewSwitcher != nil {
	//		m.viewSwitcher.SwitchTo(string(msg.View))
	//	}

	default:
		// Let existing model handle data events
		newModel, cmd := m.model.Update(msg)
		m.model = newModel.(*Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Let component tree handle UI updates
	newRoot, cmd := m.root.Update(msg)
	if c, ok := newRoot.(components.Component); ok {
		m.root = c
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders through the component tree
func (m *RefactoredModel) View() string {
	// Simple delegation to root component
	return m.root.View()
}

// Helper methods to interact with the component tree

// SwitchView switches to a different view
func (m *RefactoredModel) SwitchView(view navigation.View) {
	if m.viewSwitcher != nil {
		m.viewSwitcher.SwitchTo(string(view))
	}
}

// GetCurrentView returns the current view
func (m *RefactoredModel) GetCurrentView() navigation.View {
	if m.viewSwitcher != nil {
		return navigation.View(m.viewSwitcher.CurrentView())
	}
	return navigation.ViewProcesses
}
