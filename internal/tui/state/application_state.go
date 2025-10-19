package state

import (
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// StateManager manages versioned state for the TUI application
// It works alongside the Model struct, not as a replacement
type StateManager struct {
	// Core state fields
	version    uint64    `json:"version"`
	lastUpdate time.Time `json:"lastUpdate"`

	// UI state
	uiState *UIState `json:"uiState"`

	// Component states
	components map[string]*ComponentState `json:"components"`

	// Operation tracking
	operations *OperationsTracker `json:"operations"`

	// Performance metrics
	metrics *PerformanceMetrics `json:"metrics"`

	// Configuration and settings
	config *Configuration `json:"config"`

	// Synchronization
}

// UIState represents the current UI state
type UIState struct {
	// Active tab and view
	ActiveTab    string `json:"activeTab"`
	ActiveView   string `json:"activeView"`

	// Command input
	CommandInput string `json:"commandInput"`
	CommandMode  bool   `json:"commandMode"`

	// Navigation state
	CurrentFocus string            `json:"currentFocus"`
	FocusHistory  []string          `json:"focusHistory"`

	// Selection and scrolling
	Selections     map[string]int `json:"selections"`
	ScrollPositions map[string]int `json:"scrollPositions"`

	// Modal and overlay state
	ModalVisible   bool   `json:"modalVisible"`
	ModalType      string `json:"modalType"`
	ModalContent   interface{} `json:"modalContent"`

	// Loading and error states
	GlobalLoading  bool          `json:"globalLoading"`
	GlobalError    error         `json:"globalError,omitempty"`

	// Dimensions
	Width  int `json:"width"`
	Height int `json:"height"`

	// Timestamps
	LastTabChange    time.Time `json:"lastTabChange"`
	LastViewChange   time.Time `json:"lastViewChange"`
	LastFocusChange  time.Time `json:"lastFocusChange"`
}

// ComponentState represents the state of an individual UI component
type ComponentState struct {
	ID         string      `json:"id"`
	Type       string      `json:"type"`
	Active     bool        `json:"active"`
	Visible    bool        `json:"visible"`
	Loading    bool        `json:"loading"`
	Error      error       `json:"error,omitempty"`
	Data       interface{} `json:"data,omitempty"`

	// Component-specific state
	State      map[string]interface{} `json:"state,omitempty"`

	// Timestamps
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	LastAccess time.Time `json:"lastAccess"`

	// Performance metrics
	RenderTime time.Duration `json:"renderTime"`
	UpdateCount int64         `json:"updateCount"`
}

// PerformanceMetrics tracks application performance
type PerformanceMetrics struct {
	// Response time tracking
	AverageResponseTime time.Duration `json:"averageResponseTime"`
	LastResponseTime    time.Duration `json:"lastResponseTime"`
	PeakResponseTime    time.Duration `json:"peakResponseTime"`

	// Update frequency
	UpdateCount     int64     `json:"updateCount"`
	UpdatesPerSecond float64   `json:"updatesPerSecond"`
	LastUpdateTime  time.Time `json:"lastUpdateTime"`

	// Memory usage
	MemoryUsage     int64     `json:"memoryUsage"`
	PeakMemoryUsage int64     `json:"peakMemoryUsage"`

	// Command metrics
	CommandsExecuted int64 `json:"commandsExecuted"`
	CommandsFailed    int64 `json:"commandsFailed"`
	CommandSuccessRate float64 `json:"commandSuccessRate"`

	// UI performance
	RenderCount      int64         `json:"renderCount"`
	AverageRenderTime time.Duration `json:"averageRenderTime"`
}

// UpdateResponseTime updates the response time metrics
func (pm *PerformanceMetrics) UpdateResponseTime(duration time.Duration) {
	pm.LastResponseTime = duration
	if pm.LastResponseTime > pm.PeakResponseTime {
		pm.PeakResponseTime = pm.LastResponseTime
	}
	// Update average response time using exponential moving average
	if pm.AverageResponseTime == 0 {
		pm.AverageResponseTime = duration
	} else {
		pm.AverageResponseTime = time.Duration(
			(float64(pm.AverageResponseTime)*0.9 + float64(duration)*0.1),
		)
	}
}

// Configuration represents application configuration
type Configuration struct {
	// UI preferences
	Theme           string            `json:"theme"`
	AnimationSpeed  string            `json:"animationSpeed"`
	ShowHelp        bool              `json:"showHelp"`
	ShowStatusBar   bool              `json:"showStatusBar"`

	// Performance settings
	MaxConcurrentCommands int              `json:"maxConcurrentCommands"`
	CommandTimeout        time.Duration    `json:"commandTimeout"`
	// Feature flags
	EnableDebugMode      bool              `json:"enableDebugMode"`
	EnableMetrics        bool              `json:"enableMetrics"`
	EnableAutoSave       bool              `json:"enableAutoSave"`

	// Custom settings
	CustomSettings       map[string]interface{} `json:"customSettings"`

	// Last modified
	LastModified time.Time `json:"lastModified"`
}

// StateSnapshot represents a snapshot of the application state at a point in time
type StateSnapshot struct {
	Version    uint64                 `json:"version"`
	Timestamp  time.Time              `json:"timestamp"`
	State      map[string]interface{} `json:"state"`
	Operations map[string]*Operation  `json:"operations"`
	Metrics    *PerformanceMetrics    `json:"metrics"`
}

// StateTransition represents a change from one state to another
type StateTransition struct {
	FromVersion uint64                 `json:"fromVersion"`
	ToVersion   uint64                 `json:"toVersion"`
	Timestamp   time.Time              `json:"timestamp"`
	Changes     []StateChange          `json:"changes"`
	Duration    time.Duration          `json:"duration"`
	Trigger     string                 `json:"trigger"`
}

// StateChange represents a single change in state
type StateChange struct {
	Path     string      `json:"path"`
	OldValue interface{} `json:"oldValue"`
	NewValue interface{} `json:"newValue"`
	Type     ChangeType  `json:"type"`
}

// ChangeType represents the type of state change
type ChangeType int

const (
	ChangeTypeCreate ChangeType = iota
	ChangeTypeUpdate
	ChangeTypeDelete
	ChangeTypeMove
)

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	state := &StateManager{
		version:    0,
		lastUpdate: time.Now(),
		uiState:    NewUIState(),
		components: make(map[string]*ComponentState),
		operations: NewOperationsTracker(),
		metrics:    NewPerformanceMetrics(),
		config:     NewConfiguration(),
	}

	return state
}

// NewUIState creates a new UI state
func NewUIState() *UIState {
	return &UIState{
		ActiveTab:       "processes",
		ActiveView:      "table",
		CommandInput:    "",
		CommandMode:     false,
		CurrentFocus:    "main",
		FocusHistory:    make([]string, 0),
		Selections:      make(map[string]int),
		ScrollPositions: make(map[string]int),
		ModalVisible:    false,
		GlobalLoading:   false,
		Width:           80,
		Height:          24,
		LastTabChange:   time.Now(),
		LastViewChange:  time.Now(),
		LastFocusChange: time.Now(),
	}
}

// NewPerformanceMetrics creates new performance metrics
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		AverageResponseTime: 0,
		LastResponseTime:    0,
		PeakResponseTime:    0,
		UpdateCount:        0,
		UpdatesPerSecond:    0,
		LastUpdateTime:     time.Now(),
		MemoryUsage:         0,
		PeakMemoryUsage:     0,
		CommandsExecuted:   0,
		CommandsFailed:      0,
		CommandSuccessRate: 0,
		RenderCount:         0,
		AverageRenderTime:   0,
	}
}

// NewConfiguration creates new configuration
func NewConfiguration() *Configuration {
	return &Configuration{
		Theme:                 "default",
		AnimationSpeed:        "normal",
		ShowHelp:              true,
		ShowStatusBar:         true,
		MaxConcurrentCommands: 10,
		CommandTimeout:        30 * time.Second,
		EnableDebugMode:       false,
		EnableMetrics:         true,
		EnableAutoSave:        true,
		CustomSettings:        make(map[string]interface{}),
		LastModified:          time.Now(),
	}
}

// Version management methods

// GetVersion returns the current state version
func (s *StateManager) GetVersion() uint64 {
	return atomic.LoadUint64(&s.version)
}

// IncrementVersion increments the state version atomically
func (s *StateManager) IncrementVersion() uint64 {
	newVersion := atomic.AddUint64(&s.version, 1)
	s.lastUpdate = time.Now()
	return newVersion
}

// SetVersion sets the state version (for initialization)
func (s *StateManager) SetVersion(version uint64) {
	atomic.StoreUint64(&s.version, version)
	s.lastUpdate = time.Now()
}

// State modification methods

// TrackUpdate tracks a state update for metrics and versioning
func (s *StateManager) TrackUpdate(msg tea.Msg) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		s.metrics.UpdateResponseTime(duration)
	}()

	// Increment version for this update
	newVersion := s.IncrementVersion()

	// Process the message and update state
	switch m := msg.(type) {
	case *UIStateUpdateMessage:
		s.handleUIStateUpdate(m, newVersion)
	case *ComponentStateUpdateMessage:
		s.handleComponentStateUpdate(m, newVersion)
	case *ConfigurationUpdateMessage:
		s.handleConfigurationUpdate(m, newVersion)
	case *MetricsUpdateMessage:
		s.handleMetricsUpdate(m, newVersion)
	case *PerformanceMetricsMessage:
		s.handlePerformanceMetrics(m, newVersion)
	default:
		// Handle other message types as needed
	}

	// Update performance metrics
	s.metrics.UpdateCount++
	s.metrics.LastUpdateTime = time.Now()
}

// handleUIStateUpdate processes UI state updates
func (s *StateManager) handleUIStateUpdate(msg *UIStateUpdateMessage, version uint64) {

	switch msg.Field {
	case "activeTab":
		s.uiState.ActiveTab = msg.Value.(string)
		s.uiState.LastTabChange = time.Now()
	case "activeView":
		s.uiState.ActiveView = msg.Value.(string)
		s.uiState.LastViewChange = time.Now()
	case "commandInput":
		s.uiState.CommandInput = msg.Value.(string)
	case "commandMode":
		s.uiState.CommandMode = msg.Value.(bool)
	case "currentFocus":
		oldFocus := s.uiState.CurrentFocus
		s.uiState.CurrentFocus = msg.Value.(string)
		s.uiState.LastFocusChange = time.Now()

		// Update focus history
		if oldFocus != s.uiState.CurrentFocus {
			s.uiState.FocusHistory = append(s.uiState.FocusHistory, oldFocus)
			if len(s.uiState.FocusHistory) > 10 {
				s.uiState.FocusHistory = s.uiState.FocusHistory[1:]
			}
		}
	case "selection":
		selections := msg.Value.(map[string]int)
		for key, value := range selections {
			s.uiState.Selections[key] = value
		}
	case "scrollPosition":
		positions := msg.Value.(map[string]int)
		for key, value := range positions {
			s.uiState.ScrollPositions[key] = value
		}
	case "modalVisible":
		s.uiState.ModalVisible = msg.Value.(bool)
	case "modalType":
		s.uiState.ModalType = msg.Value.(string)
	case "modalContent":
		s.uiState.ModalContent = msg.Value
	case "globalLoading":
		s.uiState.GlobalLoading = msg.Value.(bool)
	case "globalError":
		s.uiState.GlobalError = msg.Value.(error)
	case "dimensions":
		dimensions := msg.Value.(map[string]int)
		s.uiState.Width = dimensions["width"]
		s.uiState.Height = dimensions["height"]
	}
}

// handleComponentStateUpdate processes component state updates
func (s *StateManager) handleComponentStateUpdate(msg *ComponentStateUpdateMessage, version uint64) {

	component, exists := s.components[msg.ComponentID]
	if !exists {
		// Create new component
		component = &ComponentState{
			ID:         msg.ComponentID,
			Type:       msg.ComponentType,
			State:      make(map[string]interface{}),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			LastAccess: time.Now(),
		}
		s.components[msg.ComponentID] = component
	}

	// Update component state
	for key, value := range msg.Updates {
		component.State[key] = value
	}

	if msg.Active != nil {
		component.Active = *msg.Active
	}
	if msg.Visible != nil {
		component.Visible = *msg.Visible
	}
	if msg.Loading != nil {
		component.Loading = *msg.Loading
	}
	if msg.Error != nil {
		component.Error = *msg.Error
	}
	if msg.Data != nil {
		component.Data = *msg.Data
	}

	component.UpdatedAt = time.Now()
	component.LastAccess = time.Now()
	atomic.AddInt64(&component.UpdateCount, 1)
}

// handleConfigurationUpdate processes configuration updates
func (s *StateManager) handleConfigurationUpdate(msg *ConfigurationUpdateMessage, version uint64) {

	switch msg.Field {
	case "theme":
		s.config.Theme = msg.Value.(string)
	case "animationSpeed":
		s.config.AnimationSpeed = msg.Value.(string)
	case "showHelp":
		s.config.ShowHelp = msg.Value.(bool)
	case "showStatusBar":
		s.config.ShowStatusBar = msg.Value.(bool)
	case "maxConcurrentCommands":
		s.config.MaxConcurrentCommands = msg.Value.(int)
	case "commandTimeout":
		s.config.CommandTimeout = msg.Value.(time.Duration)
	case "enableDebugMode":
		s.config.EnableDebugMode = msg.Value.(bool)
	case "enableMetrics":
		s.config.EnableMetrics = msg.Value.(bool)
	case "enableAutoSave":
		s.config.EnableAutoSave = msg.Value.(bool)
	case "customSettings":
		settings := msg.Value.(map[string]interface{})
		for key, value := range settings {
			s.config.CustomSettings[key] = value
		}
	}

	s.config.LastModified = time.Now()
}

// handleMetricsUpdate processes metrics updates
func (s *StateManager) handleMetricsUpdate(msg *MetricsUpdateMessage, version uint64) {

	switch msg.Type {
	case "responseTime":
		s.metrics.LastResponseTime = msg.Value.(time.Duration)
		if s.metrics.LastResponseTime > s.metrics.PeakResponseTime {
			s.metrics.PeakResponseTime = s.metrics.LastResponseTime
		}
	case "memoryUsage":
		s.metrics.MemoryUsage = msg.Value.(int64)
		if s.metrics.MemoryUsage > s.metrics.PeakMemoryUsage {
			s.metrics.PeakMemoryUsage = s.metrics.MemoryUsage
		}
	case "commandExecuted":
		atomic.AddInt64(&s.metrics.CommandsExecuted, 1)
	case "commandFailed":
		atomic.AddInt64(&s.metrics.CommandsFailed, 1)
		s.updateCommandSuccessRate()
	case "renderTime":
		s.metrics.RenderCount++
		// Update average render time
		if s.metrics.AverageRenderTime == 0 {
			s.metrics.AverageRenderTime = msg.Value.(time.Duration)
		} else {
			s.metrics.AverageRenderTime = time.Duration(
				(float64(s.metrics.AverageRenderTime) + float64(msg.Value.(time.Duration))) / 2,
			)
		}
	}
}

// handlePerformanceMetrics processes performance metrics
func (s *StateManager) handlePerformanceMetrics(msg *PerformanceMetricsMessage, version uint64) {

	s.metrics = msg.Metrics
	s.metrics.LastUpdateTime = time.Now()
}

// updateCommandSuccessRate updates the command success rate
func (s *StateManager) updateCommandSuccessRate() {
	total := atomic.LoadInt64(&s.metrics.CommandsExecuted)
	failed := atomic.LoadInt64(&s.metrics.CommandsFailed)

	if total > 0 {
		s.metrics.CommandSuccessRate = float64(total-failed) / float64(total)
	}
}

// Query methods

// GetUIState returns a copy of the current UI state
func (s *StateManager) GetUIState() UIState {

	// Return a deep copy
	state := *s.uiState
	state.Selections = make(map[string]int)
	state.ScrollPositions = make(map[string]int)
	state.FocusHistory = make([]string, len(s.uiState.FocusHistory))

	for k, v := range s.uiState.Selections {
		state.Selections[k] = v
	}
	for k, v := range s.uiState.ScrollPositions {
		state.ScrollPositions[k] = v
	}
	copy(state.FocusHistory, s.uiState.FocusHistory)

	return state
}

// GetComponentState returns the state of a specific component
func (s *StateManager) GetComponentState(componentID string) (*ComponentState, bool) {

	component, exists := s.components[componentID]
	if !exists {
		return nil, false
	}

	// Return a copy
	copy := &ComponentState{
		ID:          component.ID,
		Type:        component.Type,
		Active:      component.Active,
		Visible:     component.Visible,
		Loading:     component.Loading,
		Error:       component.Error,
		Data:        component.Data,
		State:       make(map[string]interface{}),
		CreatedAt:   component.CreatedAt,
		UpdatedAt:   component.UpdatedAt,
		LastAccess:  component.LastAccess,
		RenderTime:  component.RenderTime,
		UpdateCount: atomic.LoadInt64(&component.UpdateCount),
	}

	for k, v := range component.State {
		copy.State[k] = v
	}

	return copy, true
}

// GetConfiguration returns a copy of the current configuration
func (s *StateManager) GetConfiguration() Configuration {

	config := *s.config
	config.CustomSettings = make(map[string]interface{})

	for k, v := range s.config.CustomSettings {
		config.CustomSettings[k] = v
	}

	return config
}

// GetMetrics returns a copy of the current performance metrics
func (s *StateManager) GetMetrics() PerformanceMetrics {

	return *s.metrics
}

// GetActiveComponents returns all active components
func (s *StateManager) GetActiveComponents() map[string]*ComponentState {

	result := make(map[string]*ComponentState)
	for id, component := range s.components {
		if component.Active {
			// Return a copy
			copy := &ComponentState{
				ID:          component.ID,
				Type:        component.Type,
				Active:      component.Active,
				Visible:     component.Visible,
				Loading:     component.Loading,
				Error:       component.Error,
				Data:        component.Data,
				State:       make(map[string]interface{}),
				CreatedAt:   component.CreatedAt,
				UpdatedAt:   component.UpdatedAt,
				LastAccess:  component.LastAccess,
				RenderTime:  component.RenderTime,
				UpdateCount: atomic.LoadInt64(&component.UpdateCount),
			}

			for k, v := range component.State {
				copy.State[k] = v
			}

			result[id] = copy
		}
	}

	return result
}

// GetLoadingComponents returns all components that are currently loading
func (s *StateManager) GetLoadingComponents() map[string]*ComponentState {

	result := make(map[string]*ComponentState)
	for id, component := range s.components {
		if component.Loading {
			result[id] = component
		}
	}

	return result
}

// GetComponentsWithError returns all components that have errors
func (s *StateManager) GetComponentsWithError() map[string]*ComponentState {

	result := make(map[string]*ComponentState)
	for id, component := range s.components {
		if component.Error != nil {
			result[id] = component
		}
	}

	return result
}

// CreateSnapshot creates a snapshot of the current state
func (s *StateManager) CreateSnapshot() *StateSnapshot {

	return &StateSnapshot{
		Version:    s.GetVersion(),
		Timestamp:  time.Now(),
		State:      s.createStateMap(),
		Operations: s.operations.GetAllOperations(),
		Metrics:    s.metrics,
	}
}

// createStateMap creates a map representation of the current state
func (s *StateManager) createStateMap() map[string]interface{} {
	state := make(map[string]interface{})

	// Add UI state
	state["ui"] = s.uiState

	// Add components
	components := make(map[string]interface{})
	for id, component := range s.components {
		components[id] = component
	}
	state["components"] = components

	// Add configuration
	state["config"] = s.config

	// Add metrics
	state["metrics"] = s.metrics

	return state
}

// RestoreFromSnapshot restores state from a snapshot
func (s *StateManager) RestoreFromSnapshot(snapshot *StateSnapshot) error {

	// Restore version
	s.SetVersion(snapshot.Version)

	// Restore state from snapshot map
	stateMap := snapshot.State
	if uiState, ok := stateMap["ui"].(*UIState); ok {
		s.uiState = uiState
	}

		if components, ok := stateMap["components"].(map[string]interface{}); ok {
			s.components = make(map[string]*ComponentState)
			for id, comp := range components {
				if component, ok := comp.(*ComponentState); ok {
					s.components[id] = component
				}
			}
		}

		if config, ok := stateMap["config"].(*Configuration); ok {
			s.config = config
		}

		if metrics, ok := stateMap["metrics"].(*PerformanceMetrics); ok {
			s.metrics = metrics
		}

	return nil
}

// Cleanup removes old and unused state data
func (s *StateManager) Cleanup(maxAge time.Duration) {

	cutoff := time.Now().Add(-maxAge)

	// Clean up inactive components
	for id, component := range s.components {
		if !component.Active && component.LastAccess.Before(cutoff) {
			delete(s.components, id)
		}
	}

	// Clean up operations tracker
	s.operations.Cleanup(maxAge)

	// Cache has its own automatic cleanup
}

// Message types for state updates

// UIStateUpdateMessage represents a UI state update
type UIStateUpdateMessage struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

// ComponentStateUpdateMessage represents a component state update
type ComponentStateUpdateMessage struct {
	ComponentID string                 `json:"componentId"`
	ComponentType string                 `json:"componentType"`
	Updates     map[string]interface{} `json:"updates"`
	Active      *bool                  `json:"active,omitempty"`
	Visible     *bool                  `json:"visible,omitempty"`
	Loading     *bool                  `json:"loading,omitempty"`
	Error       *error                 `json:"error,omitempty"`
	Data        *interface{}            `json:"data,omitempty"`
}

// ConfigurationUpdateMessage represents a configuration update
type ConfigurationUpdateMessage struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

// MetricsUpdateMessage represents a metrics update
type MetricsUpdateMessage struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// PerformanceMetricsMessage represents performance metrics
type PerformanceMetricsMessage struct {
	Metrics *PerformanceMetrics `json:"metrics"`
}