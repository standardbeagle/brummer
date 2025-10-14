package components

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
)

// SystemComponent manages system messages and panel state
// This component migrates functionality from the old SystemController
// to use the new service-based architecture
type SystemComponent struct {
	BaseComponent

	// Service integration
	systemService *services.SystemService
	eventBus      *events.EventBus

	// Component state
	currentMessages []services.SystemMessage
	panelExpanded   bool
	unreadCount     int
	bufferUsage     int

	// Event subscription
	subscriptionCh chan events.Event

	// Performance tracking
	lastMessageTime time.Time
	messageCount    int
}

// NewSystemComponent creates a new system component
func NewSystemComponent(systemService *services.SystemService) *SystemComponent {
	if systemService == nil {
		systemService = services.NewSystemService(events.NewEventBus())
	}

	component := &SystemComponent{
		BaseComponent:   NewBaseComponent(),
		systemService:   systemService,
		eventBus:        systemService.EventBus(),
		subscriptionCh:  make(chan events.Event, 100),
		currentMessages: make([]services.SystemMessage, 0),
		panelExpanded:   false,
		unreadCount:     0,
		bufferUsage:     0,
		lastMessageTime: time.Now(),
		messageCount:    0,
	}

	// Initialize component state from service
	component.syncFromService()

	// Subscribe to system service events
	component.subscribeToEvents()

	return component
}

// Init initializes the component
func (sc *SystemComponent) Init() tea.Cmd {
	// Get initial state from service
	return func() tea.Msg {
		ctx := context.Background()

		// Get initial panel state
		panelState, err := sc.systemService.GetPanelState(ctx)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "SystemComponent",
				Message: "Failed to get panel state: " + err.Error(),
				Time:    time.Now(),
			}
		}

		sc.panelExpanded = panelState.Expanded

		// Get initial unread count
		unreadCount, err := sc.systemService.GetUnreadMessageCount(ctx)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "SystemComponent",
				Message: "Failed to get unread count: " + err.Error(),
				Time:    time.Now(),
			}
		}

		sc.unreadCount = unreadCount

		// Get initial buffer usage
		bufferUsage, err := sc.systemService.GetBufferUsage(ctx)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "SystemComponent",
				Message: "Failed to get buffer usage: " + err.Error(),
				Time:    time.Now(),
			}
		}

		sc.bufferUsage = bufferUsage

		return messages.ComponentReadyMsg{
			ComponentID: "system",
			Time:        time.Now(),
		}
	}
}

// Update handles incoming messages and system events
func (sc *SystemComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.SystemMessageAddedMsg:
		return sc.handleSystemMessageAdded(msg)

	case messages.SystemPanelToggledMsg:
		return sc.handleSystemPanelToggled(msg)

	case messages.SystemMessagesReadMsg:
		return sc.handleSystemMessagesRead(msg)

	case messages.SystemMessageBufferClearedMsg:
		return sc.handleSystemMessageBufferCleared(msg)

	case tea.KeyMsg:
		return sc.handleKeyMsg(msg)

	case events.Event:
		return sc.handleServiceEvent(msg)

	default:
		return sc, nil
	}
}

// View renders the system component
// Note: This component is primarily for system message management and doesn't have visual output
// The visual feedback is handled by other components based on the system state
func (sc *SystemComponent) View() string {
	// This component doesn't have a visual representation
	// It handles system message management behind the scenes
	return ""
}

// handleSystemMessageAdded processes new system message additions
func (sc *SystemComponent) handleSystemMessageAdded(msg messages.SystemMessageAddedMsg) (tea.Model, tea.Cmd) {
	sc.unreadCount = msg.UnreadCount
	sc.lastMessageTime = msg.Time
	sc.messageCount++

	// Refresh current messages from service
	sc.refreshMessages()

	return sc, func() tea.Msg {
		return messages.SystemMessageAddedMsg{
			MessageID:   msg.MessageID,
			Level:       msg.Level,
			Content:     msg.Content,
			Source:      msg.Source,
			UnreadCount: msg.UnreadCount,
			Time:        msg.Time,
		}
	}
}

// handleSystemPanelToggled processes panel state changes
func (sc *SystemComponent) handleSystemPanelToggled(msg messages.SystemPanelToggledMsg) (tea.Model, tea.Cmd) {
	sc.panelExpanded = msg.Expanded

	return sc, func() tea.Msg {
		return messages.SystemPanelToggledMsg{
			Expanded: msg.Expanded,
			Trigger:  msg.Trigger,
			Time:     msg.Time,
		}
	}
}

// handleSystemMessagesRead processes messages marked as read
func (sc *SystemComponent) handleSystemMessagesRead(msg messages.SystemMessagesReadMsg) (tea.Model, tea.Cmd) {
	sc.unreadCount = msg.UnreadCount

	// Refresh current messages from service
	sc.refreshMessages()

	return sc, func() tea.Msg {
		return messages.SystemMessagesReadMsg{
			UpToTime:    msg.UpToTime,
			ReadCount:   msg.ReadCount,
			UnreadCount: msg.UnreadCount,
			Time:        msg.Time,
		}
	}
}

// handleSystemMessageBufferCleared processes buffer clearing
func (sc *SystemComponent) handleSystemMessageBufferCleared(msg messages.SystemMessageBufferClearedMsg) (tea.Model, tea.Cmd) {
	sc.currentMessages = make([]services.SystemMessage, 0)
	sc.unreadCount = 0
	sc.bufferUsage = 0
	sc.messageCount = 0

	return sc, func() tea.Msg {
		return messages.SystemMessageBufferClearedMsg{
			ClearedCount: msg.ClearedCount,
			Trigger:      msg.Trigger,
			Time:         msg.Time,
		}
	}
}

// handleKeyMsg processes keyboard input for system operations
func (sc *SystemComponent) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyF1:
		// Toggle system panel
		return sc.toggleSystemPanel()

	default:
		// Check for specific key strings
		keyStr := msg.String()
		if keyStr == "f1" {
			return sc.toggleSystemPanel()
		}

		return sc, nil
	}
}

// handleServiceEvent handles events from the system service
func (sc *SystemComponent) handleServiceEvent(event events.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case events.SystemMessage:
		return sc.handleSystemServiceEvent(event)
	default:
		return sc, nil
	}
}

// handleSystemServiceEvent handles system service events
func (sc *SystemComponent) handleSystemServiceEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	// Check event type
	if eventType, ok := data["event_type"].(string); ok {
		switch eventType {
		case "messages_read":
			return sc.handleMessagesReadEvent(event)
		case "panel_toggled":
			return sc.handlePanelToggledEvent(event)
		case "buffer_cleared":
			return sc.handleBufferClearedEvent(event)
		default:
			// Handle new message event
			return sc.handleNewMessageEvent(event)
		}
	}

	return sc, nil
}

// handleNewMessageEvent handles new message events from service
func (sc *SystemComponent) handleNewMessageEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	messageID, _ := data["message_id"].(string)
	level, _ := data["level"].(string)
	content, _ := data["content"].(string)
	source, _ := data["source"].(string)
	unreadCount, _ := data["unread_count"].(int)

	sc.unreadCount = unreadCount
	sc.lastMessageTime = event.Timestamp
	sc.messageCount++

	// Refresh current messages from service
	sc.refreshMessages()

	return sc, func() tea.Msg {
		return messages.SystemMessageAddedMsg{
			MessageID:   messageID,
			Level:       messages.LogLevel(level),
			Content:     content,
			Source:      source,
			UnreadCount: unreadCount,
			Time:        event.Timestamp,
		}
	}
}

// handleMessagesReadEvent handles messages read events from service
func (sc *SystemComponent) handleMessagesReadEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	readCount, _ := data["read_count"].(int)
	unreadCount, _ := data["unread_count"].(int)
	upToTime, _ := data["up_to_time"].(time.Time)

	sc.unreadCount = unreadCount

	// Refresh current messages from service
	sc.refreshMessages()

	return sc, func() tea.Msg {
		return messages.SystemMessagesReadMsg{
			UpToTime:    upToTime,
			ReadCount:   readCount,
			UnreadCount: unreadCount,
			Time:        event.Timestamp,
		}
	}
}

// handlePanelToggledEvent handles panel toggled events from service
func (sc *SystemComponent) handlePanelToggledEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	expanded, _ := data["expanded"].(bool)
	trigger, _ := data["trigger"].(string)

	sc.panelExpanded = expanded

	return sc, func() tea.Msg {
		return messages.SystemPanelToggledMsg{
			Expanded: expanded,
			Trigger:  trigger,
			Time:     event.Timestamp,
		}
	}
}

// handleBufferClearedEvent handles buffer cleared events from service
func (sc *SystemComponent) handleBufferClearedEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	clearedCount, _ := data["cleared_count"].(int)
	trigger, _ := data["trigger"].(string)

	sc.currentMessages = make([]services.SystemMessage, 0)
	sc.unreadCount = 0
	sc.bufferUsage = 0
	sc.messageCount = 0

	return sc, func() tea.Msg {
		return messages.SystemMessageBufferClearedMsg{
			ClearedCount: clearedCount,
			Trigger:      trigger,
			Time:         event.Timestamp,
		}
	}
}

// syncFromService synchronizes component state from the service
func (sc *SystemComponent) syncFromService() {
	ctx := context.Background()

	// Get panel state
	if panelState, err := sc.systemService.GetPanelState(ctx); err == nil {
		sc.panelExpanded = panelState.Expanded
	}

	// Get unread count
	if unreadCount, err := sc.systemService.GetUnreadMessageCount(ctx); err == nil {
		sc.unreadCount = unreadCount
	}

	// Get buffer usage
	if bufferUsage, err := sc.systemService.GetBufferUsage(ctx); err == nil {
		sc.bufferUsage = bufferUsage
	}

	// Refresh messages
	sc.refreshMessages()
}

// refreshMessages refreshes the current messages from the service
func (sc *SystemComponent) refreshMessages() {
	ctx := context.Background()

	// Get latest messages (limit to 50 for performance)
	messages, err := sc.systemService.GetMessages(ctx, 0, 50)
	if err == nil {
		sc.currentMessages = messages
	}
}

// subscribeToEvents subscribes to system service events
func (sc *SystemComponent) subscribeToEvents() {
	// Subscribe to system service events through the event bus
	// This would typically use the event bus subscription mechanism
	// For now, we'll simulate this with direct channel subscription
}

// Public API methods

// AddSystemMessage adds a new system message
func (sc *SystemComponent) AddSystemMessage(level messages.LogLevel, content, source string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		message := services.SystemMessage{
			Level:   level,
			Content: content,
			Source:  source,
		}

		messageID, err := sc.systemService.AddSystemMessage(ctx, message)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "SystemComponent",
				Message: "Failed to add system message: " + err.Error(),
				Time:    time.Now(),
			}
		}

		// Return a message indicating success
		return messages.SystemMessageAddedMsg{
			MessageID:   messageID,
			Level:       level,
			Content:     content,
			Source:      source,
			UnreadCount: sc.unreadCount + 1, // Approximate
			Time:        time.Now(),
		}
	}
}

// GetMessages returns the current system messages
func (sc *SystemComponent) GetMessages() []services.SystemMessage {
	return sc.currentMessages
}

// GetUnreadCount returns the current unread message count
func (sc *SystemComponent) GetUnreadCount() int {
	return sc.unreadCount
}

// GetPanelState returns the current panel state
func (sc *SystemComponent) GetPanelState() bool {
	return sc.panelExpanded
}

// ToggleSystemPanel toggles the system panel expanded state
func (sc *SystemComponent) ToggleSystemPanel() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		expanded, err := sc.systemService.ToggleSystemPanel(ctx)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "SystemComponent",
				Message: "Failed to toggle system panel: " + err.Error(),
				Time:    time.Now(),
			}
		}

		return messages.SystemPanelToggledMsg{
			Expanded: expanded,
			Trigger:  "keyboard",
			Time:     time.Now(),
		}
	}
}

// toggleSystemPanel is an internal helper
func (sc *SystemComponent) toggleSystemPanel() (tea.Model, tea.Cmd) {
	return sc, sc.ToggleSystemPanel()
}

// MarkMessagesAsRead marks messages as read up to the specified time
func (sc *SystemComponent) MarkMessagesAsRead(upToTime time.Time) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := sc.systemService.MarkMessagesAsRead(ctx, upToTime)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "SystemComponent",
				Message: "Failed to mark messages as read: " + err.Error(),
				Time:    time.Now(),
			}
		}

		return messages.SystemMessagesReadMsg{
			UpToTime:  upToTime,
			ReadCount: 0, // Service will provide actual count
			UnreadCount: 0, // Service will provide actual count
			Time:      time.Now(),
		}
	}
}

// ClearMessages clears all system messages
func (sc *SystemComponent) ClearMessages() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := sc.systemService.ClearMessages(ctx)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "SystemComponent",
				Message: "Failed to clear messages: " + err.Error(),
				Time:    time.Now(),
			}
		}

		return messages.SystemMessageBufferClearedMsg{
			ClearedCount: len(sc.currentMessages),
			Trigger:      "user_action",
			Time:         time.Now(),
		}
	}
}

// GetBufferUsage returns current buffer usage statistics
func (sc *SystemComponent) GetBufferUsage() (current, limit int) {
	ctx := context.Background()
	currentUsage, err := sc.systemService.GetBufferUsage(ctx)
	if err != nil {
		return sc.bufferUsage, sc.systemService.GetBufferLimit()
	}
	return currentUsage, sc.systemService.GetBufferLimit()
}

// GetMessageStats returns message processing statistics
func (sc *SystemComponent) GetMessageStats() (int, time.Time) {
	return sc.messageCount, sc.lastMessageTime
}

// Focus gives keyboard focus to the system component
func (sc *SystemComponent) Focus() tea.Cmd {
	sc.BaseComponent.Focus()

	// Publish focus change event
	sc.eventBus.Publish(events.Event{
		Type: "system.focus.changed",
		Data: map[string]interface{}{
			"component": "SystemComponent",
			"focused":   true,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// Blur removes keyboard focus from the system component
func (sc *SystemComponent) Blur() tea.Cmd {
	sc.BaseComponent.Blur()

	// Publish focus change event
	sc.eventBus.Publish(events.Event{
		Type: "system.focus.changed",
		Data: map[string]interface{}{
			"component": "SystemComponent",
			"focused":   false,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// Cleanup performs cleanup when component is destroyed
func (sc *SystemComponent) Cleanup() error {
	ctx := context.Background()
	return sc.systemService.Shutdown(ctx)
}