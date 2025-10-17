package state

import (
	"fmt"
	"sync"
	"time"
)

// UIComponentsManager manages the state of UI components
type UIComponentsManager struct {
	components map[string]*UIComponent
	mu         sync.RWMutex
	maxSize    int
	registry   *ComponentRegistry
}

// UIComponent represents the state of a UI component
type UIComponent struct {
	// Basic identification
	ID     string      `json:"id"`
	Type   string      `json:"type"`
	Name   string      `json:"name"`

	// Component state
	Active   bool          `json:"active"`
	Visible  bool          `json:"visible"`
	Loading  bool          `json:"loading"`
	Error    error         `json:"error,omitempty"`
	Data     interface{}   `json:"data,omitempty"`

	// UI-specific state
	State    map[string]interface{} `json:"state,omitempty"`

	// Navigation and focus
	Focusable bool          `json:"focusable"`
	Focused   bool          `json:"focused"`
	TabOrder  int           `json:"tabOrder,omitempty"`

	// Dimensions and positioning
	Width     int           `json:"width"`
	Height    int           `json:"height"`
	X         int           `json:"x"`
	Y         int           `json:"y"`

	// Performance tracking
	RenderTime     time.Duration `json:"renderTime"`
	LastRender     time.Time     `json:"lastRender"`
	UpdateCount    int64         `json:"updateCount"`
	LastUpdate     time.Time     `json:"lastUpdate"`

	// Event handling
	EventHandlers  map[string][]string `json:"eventHandlers,omitempty"`
	CanInteract   bool               `json:"canInteract"`

	// Timestamps
	CreatedAt      time.Time `json:"createdAt"`
	LastAccess     time.Time `json:"lastAccess"`
	LastActivation time.Time `json:"lastActivation,omitempty"`

	// Component hierarchy
	ParentID    string                 `json:"parentId,omitempty"`
	ChildrenIDs []string               `json:"childrenIds,omitempty"`
}

// ComponentRegistry maintains a registry of component types and their capabilities
type ComponentRegistry struct {
	componentTypes map[string]*ComponentType
	mu             sync.RWMutex
}

// ComponentType defines the capabilities and properties of a component type
type ComponentType struct {
	Name        string                   `json:"name"`
	Category    string                   `json:"category"`
	Focusable   bool                     `json:"focusable"`
	Interactive bool                     `json:"interactive"`
	RequiredProps []string                `json:"requiredProps"`
	OptionalProps []string                `json:"optionalProps"`
	DefaultState map[string]interface{}   `json:"defaultState"`
	Styles      []string                 `json:"styles"`
}


// ComponentEvent represents an event that occurred in a component
type ComponentEvent struct {
	ComponentID string                 `json:"componentId"`
	EventType   string                 `json:"eventType"`
	Data        interface{}            `json:"data,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
}

// ComponentFilter provides filtering capabilities for components
type ComponentFilter struct {
	Types       []string `json:"types,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	ActiveOnly  bool     `json:"activeOnly,omitempty"`
	VisibleOnly bool     `json:"visibleOnly,omitempty"`
	HasError    *bool    `json:"hasError,omitempty"`
	LoadingOnly bool     `json:"loadingOnly,omitempty"`
	Focusable   bool     `json:"focusable,omitempty"`
}

// NewUIComponentsManager creates a new UI components manager
func NewUIComponentsManager(maxSize int) *UIComponentsManager {
	manager := &UIComponentsManager{
		components: make(map[string]*UIComponent),
		maxSize:    maxSize,
		registry:   NewComponentRegistry(),
	}

	// Register default component types
	manager.registerDefaultComponentTypes()

	return manager
}

// NewComponentRegistry creates a new component registry
func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		componentTypes: make(map[string]*ComponentType),
	}
}

// registerDefaultComponentTypes registers default component types
func (m *UIComponentsManager) registerDefaultComponentTypes() {
	// Register common component types
	componentTypes := []*ComponentType{
		{
			Name:        "tab",
			Category:    "navigation",
			Focusable:   true,
			Interactive: true,
			RequiredProps: []string{"id", "label"},
			OptionalProps: []string{"icon", "badge", "disabled"},
			DefaultState: map[string]interface{}{
				"active": false,
				"badge":  nil,
			},
			Styles: []string{"default", "active", "disabled"},
		},
		{
			Name:        "view",
			Category:    "content",
			Focusable:   false,
			Interactive: false,
			RequiredProps: []string{"id", "type"},
			OptionalProps: []string{"title", "subtitle", "loading"},
			DefaultState: map[string]interface{}{
				"loading": false,
			},
			Styles: []string{"default", "loading"},
		},
		{
			Name:        "table",
			Category:    "content",
			Focusable:   true,
			Interactive: true,
			RequiredProps: []string{"id", "columns"},
			OptionalProps: []string{"rows", "sortable", "filterable"},
			DefaultState: map[string]interface{}{
				"sortColumn": nil,
				"sortDirection": "asc",
				"filterText": "",
			},
			Styles: []string{"default", "compact", "detailed"},
		},
		{
			Name:        "list",
			Category:    "content",
			Focusable:   true,
			Interactive: true,
			RequiredProps: []string{"id", "items"},
			OptionalProps: []string{"selectedIndex", "multiSelect", "searchable"},
			DefaultState: map[string]interface{}{
				"selectedIndex": 0,
				"multiSelect": false,
				"searchable": false,
				"searchText": "",
			},
			Styles: []string{"default", "compact", "detailed"},
		},
		{
			Name:        "input",
			Category:    "form",
			Focusable:   true,
			Interactive: true,
			RequiredProps: []string{"id", "type"},
			OptionalProps: []string{"placeholder", "value", "disabled", "readonly"},
			DefaultState: map[string]interface{}{
				"value": "",
				"disabled": false,
				"readonly": false,
				"placeholder": "",
			},
			Styles: []string{"default", "error", "disabled", "readonly"},
		},
		{
			Name:        "button",
			Category:    "form",
			Focusable:   true,
			Interactive: true,
			RequiredProps: []string{"id", "label"},
			OptionalProps: []string{"variant", "disabled", "loading"},
			DefaultState: map[string]interface{}{
				"variant": "primary",
				"disabled": false,
				"loading": false,
			},
			Styles: []string{"primary", "secondary", "danger", "disabled", "loading"},
		},
		{
			Name:        "modal",
			Category:    "overlay",
			Focusable:   true,
			Interactive: true,
			RequiredProps: []string{"id", "title"},
			OptionalProps: []string{"content", "size", "closable"},
			DefaultState: map[string]interface{}{
				"size": "medium",
				"closable": true,
			},
			Styles: []string{"default", "small", "medium", "large"},
		},
	}

	for _, compType := range componentTypes {
		m.registry.RegisterComponentType(compType)
	}
}

// RegisterComponentType registers a new component type
func (r *ComponentRegistry) RegisterComponentType(compType *ComponentType) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.componentTypes[compType.Name] = compType
}

// GetComponentType returns a component type by name
func (r *ComponentRegistry) GetComponentType(name string) (*ComponentType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	compType, exists := r.componentTypes[name]
	return compType, exists
}

// GetComponentTypes returns all registered component types
func (r *ComponentRegistry) GetComponentTypes() map[string]*ComponentType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*ComponentType)
	for name, compType := range r.componentTypes {
		result[name] = compType
	}

	return result
}

// CreateComponent creates a new UI component
func (m *UIComponentsManager) CreateComponent(id, compType, name string, properties map[string]interface{}) (*UIComponent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if component type exists
	componentType, exists := m.registry.GetComponentType(compType)
	if !exists {
		return nil, fmt.Errorf("unknown component type: %s", compType)
	}

	// Check if we need to clean up old components
	if len(m.components) >= m.maxSize {
		m.cleanupOldComponents()
	}

	// Validate required properties
	for _, requiredProp := range componentType.RequiredProps {
		if _, exists := properties[requiredProp]; !exists {
			return nil, fmt.Errorf("missing required property: %s", requiredProp)
		}
	}

	// Create component with default state
	now := time.Now()
	component := &UIComponent{
		ID:             id,
		Type:           compType,
		Name:           name,
		Active:         false,
		Visible:        true,
		Loading:        false,
		Data:           properties,
		State:          make(map[string]interface{}),
		Focusable:      componentType.Focusable,
		Focused:        false,
		CanInteract:    componentType.Interactive,
		RenderTime:     0,
		LastRender:     now,
		UpdateCount:    0,
		LastUpdate:     now,
		EventHandlers:  make(map[string][]string),
		CreatedAt:      now,
		LastAccess:     now,
		ChildrenIDs:    make([]string, 0),
	}

	// Apply default state
	for key, value := range componentType.DefaultState {
		component.State[key] = value
	}

	// Apply properties
	for key, value := range properties {
		component.State[key] = value
	}

	m.components[id] = component
	return component, nil
}

// GetComponent returns a component by ID
func (m *UIComponentsManager) GetComponent(id string) (*UIComponent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	component, exists := m.components[id]
	if !exists {
		return nil, false
	}

	// Update last access
	component.LastAccess = time.Now()

	// Return a copy
	return m.copyComponent(component), true
}

// GetAllComponents returns all components
func (m *UIComponentsManager) GetAllComponents() map[string]*UIComponent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*UIComponent)
	for id, component := range m.components {
		result[id] = m.copyComponent(component)
	}

	return result
}

// GetComponentsByType returns components filtered by type
func (m *UIComponentsManager) GetComponentsByType(compTypes ...string) map[string]*UIComponent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*UIComponent)
	for id, component := range m.components {
		for _, compType := range compTypes {
			if component.Type == compType {
				result[id] = m.copyComponent(component)
				break
			}
		}
	}

	return result
}

// GetActiveComponents returns all active components
func (m *UIComponentsManager) GetActiveComponents() map[string]*UIComponent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*UIComponent)
	for id, component := range m.components {
		if component.Active {
			result[id] = m.copyComponent(component)
		}
	}

	return result
}

// GetVisibleComponents returns all visible components
func (m *UIComponentsManager) GetVisibleComponents() map[string]*UIComponent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*UIComponent)
	for id, component := range m.components {
		if component.Visible {
			result[id] = m.copyComponent(component)
		}
	}

	return result
}

// GetComponentsWithError returns components that have errors
func (m *UIComponentsManager) GetComponentsWithError() map[string]*UIComponent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*UIComponent)
	for id, component := range m.components {
		if component.Error != nil {
			result[id] = m.copyComponent(component)
		}
	}

	return result
}

// GetLoadingComponents returns components that are currently loading
func (m *UIComponentsManager) GetLoadingComponents() map[string]*UIComponent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*UIComponent)
	for id, component := range m.components {
		if component.Loading {
			result[id] = m.copyComponent(component)
		}
	}

	return result
}

// GetFocusableComponents returns all focusable components
func (m *UIComponentsManager) GetFocusableComponents() map[string]*UIComponent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*UIComponent)
	for id, component := range m.components {
		if component.Focusable && component.Visible && component.CanInteract {
			result[id] = m.copyComponent(component)
		}
	}

	return result
}

// GetFilteredComponents returns components filtered by the provided filter
func (m *UIComponentsManager) GetFilteredComponents(filter *ComponentFilter) map[string]*UIComponent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*UIComponent)
	for id, component := range m.components {
		if m.matchesFilter(component, filter) {
			result[id] = m.copyComponent(component)
		}
	}

	return result
}

// matchesFilter checks if a component matches the provided filter
func (m *UIComponentsManager) matchesFilter(component *UIComponent, filter *ComponentFilter) bool {
	// Check types filter
	if len(filter.Types) > 0 {
		found := false
		for _, compType := range filter.Types {
			if component.Type == compType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check categories filter
	if len(filter.Categories) > 0 {
		compType, exists := m.registry.GetComponentType(component.Type)
		if !exists {
			return false
		}

		found := false
		for _, category := range filter.Categories {
			if compType.Category == category {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check active filter
	if filter.ActiveOnly && !component.Active {
		return false
	}

	// Check visible filter
	if filter.VisibleOnly && !component.Visible {
		return false
	}

	// Check error filter
	if filter.HasError != nil {
		hasError := component.Error != nil
		if hasError != *filter.HasError {
			return false
		}
	}

	// Check loading filter
	if filter.LoadingOnly && !component.Loading {
		return false
	}

	// Check focusable filter
	if filter.Focusable && !component.Focusable {
		return false
	}

	return true
}

// UpdateComponent updates a component's state
func (m *UIComponentsManager) UpdateComponent(id string, updates map[string]interface{}) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	component, exists := m.components[id]
	if !exists {
		return false
	}

	// Apply updates
	for key, value := range updates {
		component.State[key] = value
	}

	component.LastUpdate = time.Now()
	component.LastAccess = time.Now()
	component.UpdateCount++

	return true
}

// UpdateComponentProperty updates a specific property of a component
func (m *UIComponentsManager) UpdateComponentProperty(id string, property string, value interface{}) bool {
	return m.UpdateComponent(id, map[string]interface{}{
		property: value,
	})
}

// SetComponentActive sets the active state of a component
func (m *UIComponentsManager) SetComponentActive(id string, active bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	component, exists := m.components[id]
	if !exists {
		return false
	}

	component.Active = active
	if active {
		component.LastActivation = time.Now()
	}
	component.LastUpdate = time.Now()
	component.LastAccess = time.Now()

	return true
}

// SetComponentVisible sets the visible state of a component
func (m *UIComponentsManager) SetComponentVisible(id string, visible bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	component, exists := m.components[id]
	if !exists {
		return false
	}

	component.Visible = visible
	component.LastUpdate = time.Now()
	component.LastAccess = time.Now()

	return true
}

// SetComponentLoading sets the loading state of a component
func (m *UIComponentsManager) SetComponentLoading(id string, loading bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	component, exists := m.components[id]
	if !exists {
		return false
	}

	component.Loading = loading
	component.LastUpdate = time.Now()
	component.LastAccess = time.Now()

	return true
}

// SetComponentError sets the error state of a component
func (m *UIComponentsManager) SetComponentError(id string, err error) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	component, exists := m.components[id]
	if !exists {
		return false
	}

	component.Error = err
	component.LastUpdate = time.Now()
	component.LastAccess = time.Now()

	return true
}

// SetComponentData sets the data of a component
func (m *UIComponentsManager) SetComponentData(id string, data interface{}) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	component, exists := m.components[id]
	if !exists {
		return false
	}

	component.Data = data
	component.LastUpdate = time.Now()
	component.LastAccess = time.Now()

	return true
}

// SetComponentFocus sets the focus state of a component
func (m *UIComponentsManager) SetComponentFocus(id string, focused bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	component, exists := m.components[id]
	if !exists || !component.Focusable {
		return false
	}

	component.Focused = focused
	component.LastUpdate = time.Now()
	component.LastAccess = time.Now()

	return true
}

// SetComponentDimensions sets the dimensions of a component
func (m *UIComponentsManager) SetComponentDimensions(id string, width, height int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	component, exists := m.components[id]
	if !exists {
		return false
	}

	component.Width = width
	component.Height = height
	component.LastUpdate = time.Now()
	component.LastAccess = time.Now()

	return true
}

// SetComponentPosition sets the position of a component
func (m *UIComponentsManager) SetComponentPosition(id string, x, y int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	component, exists := m.components[id]
	if !exists {
		return false
	}

	component.X = x
	component.Y = y
	component.LastUpdate = time.Now()
	component.LastAccess = time.Now()

	return true
}

// AddChildComponent adds a child component to a parent
func (m *UIComponentsManager) AddChildComponent(parentID, childID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	parent, exists := m.components[parentID]
	child, childExists := m.components[childID]
	if !exists || !childExists {
		return false
	}

	parent.ChildrenIDs = append(parent.ChildrenIDs, childID)
	child.ParentID = parentID

	parent.LastUpdate = time.Now()
	parent.LastAccess = time.Now()
	child.LastUpdate = time.Now()
	child.LastAccess = time.Now()

	return true
}

// RemoveChildComponent removes a child component from a parent
func (m *UIComponentsManager) RemoveChildComponent(parentID, childID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	parent, exists := m.components[parentID]
	child, childExists := m.components[childID]
	if !exists || !childExists {
		return false
	}

	// Remove from parent's children
	for i, id := range parent.ChildrenIDs {
		if id == childID {
			parent.ChildrenIDs = append(parent.ChildrenIDs[:i], parent.ChildrenIDs[i+1:]...)
			break
		}
	}

	// Remove parent reference from child
	child.ParentID = ""

	parent.LastUpdate = time.Now()
	parent.LastAccess = time.Now()
	child.LastUpdate = time.Now()
	child.LastAccess = time.Now()

	return true
}

// GetChildren returns all child components of a parent
func (m *UIComponentsManager) GetChildren(parentID string) map[string]*UIComponent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	parent, exists := m.components[parentID]
	if !exists {
		return nil
	}

	result := make(map[string]*UIComponent)
	for _, childID := range parent.ChildrenIDs {
		if child, exists := m.components[childID]; exists {
			result[childID] = m.copyComponent(child)
		}
	}

	return result
}

// GetParent returns the parent component of a child
func (m *UIComponentsManager) GetParent(childID string) (*UIComponent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	child, exists := m.components[childID]
	if !exists || child.ParentID == "" {
		return nil, false
	}

	parent, exists := m.components[child.ParentID]
	if !exists {
		return nil, false
	}

	return m.copyComponent(parent), true
}

// RemoveComponent removes a component from tracking
func (m *UIComponentsManager) RemoveComponent(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	component, exists := m.components[id]
	if !exists {
		return false
	}

	// Remove from parent if it has one
	if component.ParentID != "" {
		m.RemoveChildComponent(component.ParentID, id)
	}

	// Remove all children
	for _, childID := range component.ChildrenIDs {
		if child, exists := m.components[childID]; exists {
			child.ParentID = ""
		}
	}

	delete(m.components, id)
	return true
}

// cleanupOldComponents removes the oldest inactive components
func (m *UIComponentsManager) cleanupOldComponents() {
	if len(m.components) <= m.maxSize {
		return
	}

	// Find oldest inactive components
	var oldestID string
	var oldestTime time.Time

	for id, component := range m.components {
		if !component.Active && (oldestTime.IsZero() || component.LastAccess.Before(oldestTime)) {
			oldestID = id
			oldestTime = component.LastAccess
		}
	}

	if oldestID != "" {
		m.RemoveComponent(oldestID)
	}
}

// copyComponent creates a deep copy of a component
func (m *UIComponentsManager) copyComponent(original *UIComponent) *UIComponent {
	copy := &UIComponent{
		ID:             original.ID,
		Type:           original.Type,
		Name:           original.Name,
		Active:         original.Active,
		Visible:        original.Visible,
		Loading:        original.Loading,
		Error:          original.Error,
		Data:           original.Data,
		State:          make(map[string]interface{}),
		Focusable:      original.Focusable,
		Focused:        original.Focused,
		TabOrder:       original.TabOrder,
		Width:          original.Width,
		Height:         original.Height,
		X:              original.X,
		Y:              original.Y,
		RenderTime:     original.RenderTime,
		LastRender:     original.LastRender,
		UpdateCount:    original.UpdateCount,
		LastUpdate:     original.LastUpdate,
		CanInteract:    original.CanInteract,
		CreatedAt:      original.CreatedAt,
		LastAccess:     original.LastAccess,
		ParentID:       original.ParentID,
		ChildrenIDs:    make([]string, len(original.ChildrenIDs)),
	}

	// Copy state
	for k, v := range original.State {
		copy.State[k] = v
	}

	// Copy children IDs
	copy.ChildrenIDs = append([]string{}, original.ChildrenIDs...)

	return copy
}

// CreateComponentState creates a state snapshot for a component
func (m *UIComponentsManager) CreateComponentState(id string) (*ComponentState, error) {
	component, exists := m.GetComponent(id)
	if !exists {
		return nil, fmt.Errorf("component not found: %s", id)
	}

	now := time.Now()
	return &ComponentState{
		ID:         component.ID,
		Type:       component.Type,
		Active:     component.Active,
		Visible:    component.Visible,
		Loading:    component.Loading,
		Error:      component.Error,
		Data:       component.Data,
		State:      component.State,
		CreatedAt:  component.CreatedAt,
		UpdatedAt:  now,
		LastAccess: now,
		RenderTime: component.RenderTime,
		UpdateCount: component.UpdateCount,
	}, nil
}

// HandleComponentEvent processes an event for a component
func (m *UIComponentsManager) HandleComponentEvent(event *ComponentEvent) bool {
	component, exists := m.GetComponent(event.ComponentID)
	if !exists {
		return false
	}

	// Update last access
	component.LastAccess = time.Now()

	// Handle specific event types
	switch event.EventType {
	case "focus":
		if component.Focusable {
			return m.SetComponentFocus(event.ComponentID, true)
		}
	case "blur":
		if component.Focusable {
			return m.SetComponentFocus(event.ComponentID, false)
		}
	case "activate":
		return m.SetComponentActive(event.ComponentID, true)
	case "deactivate":
		return m.SetComponentActive(event.ComponentID, false)
	case "show":
		return m.SetComponentVisible(event.ComponentID, true)
	case "hide":
		return m.SetComponentVisible(event.ComponentID, false)
	case "load":
		return m.SetComponentLoading(event.ComponentID, true)
	case "loaded":
		return m.SetComponentLoading(event.ComponentID, false)
	}

	return false
}

// GetStats returns component manager statistics
func (m *UIComponentsManager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"totalComponents":   len(m.components),
		"activeComponents":  0,
		"visibleComponents": 0,
		"loadingComponents": 0,
		"componentsWithError": 0,
		"focusableComponents": 0,
		"componentsByType":  make(map[string]int),
	}

	for _, component := range m.components {
		if component.Active {
			stats["activeComponents"] = stats["activeComponents"].(int) + 1
		}
		if component.Visible {
			stats["visibleComponents"] = stats["visibleComponents"].(int) + 1
		}
		if component.Loading {
			stats["loadingComponents"] = stats["loadingComponents"].(int) + 1
		}
		if component.Error != nil {
			stats["componentsWithError"] = stats["componentsWithError"].(int) + 1
		}
		if component.Focusable {
			stats["focusableComponents"] = stats["focusableComponents"].(int) + 1
		}

		// Count by type
		byType := stats["componentsByType"].(map[string]int)
		byType[component.Type]++
	}

	return stats
}

// Cleanup removes old inactive components
func (m *UIComponentsManager) Cleanup(maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, component := range m.components {
		if !component.Active && component.LastAccess.Before(cutoff) {
			m.RemoveComponent(id)
		}
	}
}

// Reset clears all components
func (m *UIComponentsManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.components = make(map[string]*UIComponent)
}