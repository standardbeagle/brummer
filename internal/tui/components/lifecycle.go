// Package components provides lifecycle management for TUI components
package components

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// ComponentLifecycle defines the lifecycle stages for components
type ComponentLifecycle interface {
	// PreInit validates dependencies exist and are properly configured
	PreInit() error

	// Init initializes component state and returns initial commands
	Init() tea.Cmd

	// PostInit verifies successful initialization
	PostInit() error

	// Update handles messages during runtime
	Update(tea.Msg) (tea.Model, tea.Cmd)

	// View renders the component
	View() string

	// Cleanup releases resources
	Cleanup() error
}

// LifecycleState represents the current lifecycle stage
type LifecycleState int

const (
	StateUninitialized LifecycleState = iota
	StatePreInitialized
	StateInitialized
	StateReady
	StateRunning
	StateCleaning
	StateCleaned
	StateError
)

func (s LifecycleState) String() string {
	switch s {
	case StateUninitialized:
		return "Uninitialized"
	case StatePreInitialized:
		return "PreInitialized"
	case StateInitialized:
		return "Initialized"
	case StateReady:
		return "Ready"
	case StateRunning:
		return "Running"
	case StateCleaning:
		return "Cleaning"
	case StateCleaned:
		return "Cleaned"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// LifecycleManager manages component lifecycle transitions
type LifecycleManager struct {
	components map[string]ManagedComponent
	order      []string // Initialization order
	state      LifecycleState
	errors     []error
}

// ManagedComponent wraps a component with lifecycle tracking
type ManagedComponent struct {
	Name      string
	Component ComponentLifecycle
	State     LifecycleState
	Error     error

	// Dependencies
	DependsOn []string

	// Timing
	InitTime   int64 // microseconds
	UpdateTime int64 // average microseconds
	RenderTime int64 // average microseconds
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		components: make(map[string]ManagedComponent),
		order:      []string{},
		state:      StateUninitialized,
		errors:     []error{},
	}
}

// Register adds a component to lifecycle management
func (m *LifecycleManager) Register(name string, component ComponentLifecycle, deps ...string) error {
	if m.state != StateUninitialized {
		return fmt.Errorf("cannot register component after initialization")
	}

	if _, exists := m.components[name]; exists {
		return fmt.Errorf("component %s already registered", name)
	}

	m.components[name] = ManagedComponent{
		Name:      name,
		Component: component,
		State:     StateUninitialized,
		DependsOn: deps,
	}

	// Add to initialization order
	m.order = append(m.order, name)

	return nil
}

// Initialize runs the initialization lifecycle for all components
func (m *LifecycleManager) Initialize() ([]tea.Cmd, error) {
	if m.state != StateUninitialized {
		return nil, fmt.Errorf("already initialized")
	}

	// Sort components by dependencies
	ordered, err := m.topologicalSort()
	if err != nil {
		m.state = StateError
		return nil, err
	}

	var allCmds []tea.Cmd

	// Phase 1: PreInit
	for _, name := range ordered {
		mc := m.components[name]

		if err := mc.Component.PreInit(); err != nil {
			mc.State = StateError
			mc.Error = fmt.Errorf("PreInit failed: %w", err)
			m.components[name] = mc
			m.errors = append(m.errors, mc.Error)
			m.state = StateError
			return nil, fmt.Errorf("component %s PreInit failed: %w", name, err)
		}

		mc.State = StatePreInitialized
		m.components[name] = mc
	}

	// Phase 2: Init
	for _, name := range ordered {
		mc := m.components[name]

		cmd := mc.Component.Init()
		if cmd != nil {
			allCmds = append(allCmds, cmd)
		}

		mc.State = StateInitialized
		m.components[name] = mc
	}

	// Phase 3: PostInit
	for _, name := range ordered {
		mc := m.components[name]

		if err := mc.Component.PostInit(); err != nil {
			mc.State = StateError
			mc.Error = fmt.Errorf("PostInit failed: %w", err)
			m.components[name] = mc
			m.errors = append(m.errors, mc.Error)
			// PostInit failures are warnings, not fatal
			Debug().CheckState("LifecycleManager", func() error {
				return fmt.Errorf("component %s PostInit failed: %w", name, err)
			})
		} else {
			mc.State = StateReady
			m.components[name] = mc
		}
	}

	m.state = StateReady
	return allCmds, nil
}

// Cleanup runs cleanup for all components in reverse order
func (m *LifecycleManager) Cleanup() error {
	if m.state == StateCleaned {
		return nil
	}

	m.state = StateCleaning

	// Cleanup in reverse order
	for i := len(m.order) - 1; i >= 0; i-- {
		name := m.order[i]
		mc := m.components[name]

		if mc.State == StateCleaned {
			continue
		}

		if err := mc.Component.Cleanup(); err != nil {
			mc.Error = fmt.Errorf("Cleanup failed: %w", err)
			m.errors = append(m.errors, mc.Error)
			// Continue cleaning other components
		}

		mc.State = StateCleaned
		m.components[name] = mc
	}

	m.state = StateCleaned

	if len(m.errors) > 0 {
		return fmt.Errorf("cleanup completed with %d errors", len(m.errors))
	}

	return nil
}

// topologicalSort orders components by dependencies
func (m *LifecycleManager) topologicalSort() ([]string, error) {
	// Simple implementation - can be improved with proper graph algorithms
	sorted := []string{}
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(string) error
	visit = func(name string) error {
		if visited[name] {
			return nil
		}

		if visiting[name] {
			return fmt.Errorf("circular dependency detected at %s", name)
		}

		visiting[name] = true

		mc, exists := m.components[name]
		if !exists {
			return fmt.Errorf("unknown component: %s", name)
		}

		// Visit dependencies first
		for _, dep := range mc.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}

		visiting[name] = false
		visited[name] = true
		sorted = append(sorted, name)

		return nil
	}

	// Visit all components
	for name := range m.components {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return sorted, nil
}

// GetComponentState returns the state of a specific component
func (m *LifecycleManager) GetComponentState(name string) (LifecycleState, error) {
	mc, exists := m.components[name]
	if !exists {
		return StateUninitialized, fmt.Errorf("component %s not found", name)
	}
	return mc.State, mc.Error
}

// GetErrors returns all lifecycle errors
func (m *LifecycleManager) GetErrors() []error {
	return m.errors
}

// BaseLifecycleComponent provides default lifecycle implementations
type BaseLifecycleComponent struct {
	name  string
	state LifecycleState
}

// NewBaseLifecycleComponent creates a new base component
func NewBaseLifecycleComponent(name string) *BaseLifecycleComponent {
	return &BaseLifecycleComponent{
		name:  name,
		state: StateUninitialized,
	}
}

// PreInit provides default PreInit (no-op)
func (b *BaseLifecycleComponent) PreInit() error {
	b.state = StatePreInitialized
	return nil
}

// Init provides default Init (no-op)
func (b *BaseLifecycleComponent) Init() tea.Cmd {
	b.state = StateInitialized
	return nil
}

// PostInit provides default PostInit (no-op)
func (b *BaseLifecycleComponent) PostInit() error {
	b.state = StateReady
	return nil
}

// Cleanup provides default Cleanup (no-op)
func (b *BaseLifecycleComponent) Cleanup() error {
	b.state = StateCleaned
	return nil
}

// Example lifecycle-aware component wrapper
type LifecycleWrapper struct {
	BaseLifecycleComponent
	inner tea.Model

	// Dependencies
	processMgr interface{}
	logStore   interface{}
}

// WrapWithLifecycle wraps an existing component with lifecycle management
func WrapWithLifecycle(name string, component tea.Model) *LifecycleWrapper {
	return &LifecycleWrapper{
		BaseLifecycleComponent: *NewBaseLifecycleComponent(name),
		inner:                  component,
	}
}

// PreInit validates dependencies
func (w *LifecycleWrapper) PreInit() error {
	// For now, skip dependency validation as it's complex to track
	// In a real implementation, dependencies would be injected during construction
	// This is just a demonstration of the pattern

	return w.BaseLifecycleComponent.PreInit()
}

// Init initializes the wrapped component
func (w *LifecycleWrapper) Init() tea.Cmd {
	if initer, ok := w.inner.(interface{ Init() tea.Cmd }); ok {
		return initer.Init()
	}
	return w.BaseLifecycleComponent.Init()
}

// PostInit validates initialization succeeded
func (w *LifecycleWrapper) PostInit() error {
	// Component-specific validation
	switch c := w.inner.(type) {
	case *ProcessViewNew:
		if err := ValidateProcessView(c); err != nil {
			return fmt.Errorf("ProcessViewNew validation failed: %w", err)
		}
	case *LogsViewComponent:
		if err := ValidateLogsView(c); err != nil {
			return fmt.Errorf("LogsViewComponent validation failed: %w", err)
		}
	}

	return w.BaseLifecycleComponent.PostInit()
}

// Update delegates to wrapped component
func (w *LifecycleWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if updater, ok := w.inner.(interface {
		Update(tea.Msg) (tea.Model, tea.Cmd)
	}); ok {
		newModel, cmd := updater.Update(msg)
		w.inner = newModel
		return w, cmd
	}
	return w, nil
}

// View delegates to wrapped component
func (w *LifecycleWrapper) View() string {
	if viewer, ok := w.inner.(interface{ View() string }); ok {
		return viewer.View()
	}
	return ""
}

// Cleanup cleans up the wrapped component
func (w *LifecycleWrapper) Cleanup() error {
	if cleaner, ok := w.inner.(interface{ Cleanup() error }); ok {
		if err := cleaner.Cleanup(); err != nil {
			return err
		}
	}
	return w.BaseLifecycleComponent.Cleanup()
}
