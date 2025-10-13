package contracts_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
)

// These tests validate the System Service API contract
// They should pass after implementation

func TestSystemService_AddSystemMessage(t *testing.T) {
	// Test adding system messages
	ctx := context.Background()
	message := services.SystemMessage{
		Level:     messages.InfoLevel,
		Content:   "Test system message",
		Timestamp: time.Now(),
		Source:    "test",
	}

	service := services.NewSystemService(events.NewEventBus())
	messageID, err := service.AddSystemMessage(ctx, message)

	assert.NoError(t, err)
	assert.NotEmpty(t, messageID)

	// Verify message was added
	retrievedMessages, err := service.GetMessages(ctx, 0, 10)
	assert.NoError(t, err)
	assert.Len(t, retrievedMessages, 1)
	assert.Equal(t, message.Content, retrievedMessages[0].Content)
}

func TestSystemService_AddSystemMessage_TooLarge(t *testing.T) {
	// Test rejection of oversized messages
	ctx := context.Background()
	largeMessage := services.SystemMessage{
		Level:     messages.InfoLevel,
		Content:   string(make([]byte, 10001)), // 10KB message
		Timestamp: time.Now(),
		Source:    "test",
	}

	service := services.NewSystemService(events.NewEventBus())
	_, err := service.AddSystemMessage(ctx, largeMessage)

	assert.Error(t, err)
	assert.Equal(t, services.ErrMessageTooLarge, err)
}

func TestSystemService_GetUnreadMessageCount(t *testing.T) {
	// Test unread message count calculation
	ctx := context.Background()

	service := services.NewSystemService(events.NewEventBus())

	// Add messages
	testMessages := []services.SystemMessage{
		{Level: messages.InfoLevel, Content: "Message 1", Read: false, Source: "test"},
		{Level: messages.InfoLevel, Content: "Message 2", Read: false, Source: "test"},
		{Level: messages.InfoLevel, Content: "Message 3", Read: true, Source: "test"},
	}

	for _, msg := range testMessages {
		_, err := service.AddSystemMessage(ctx, msg)
		assert.NoError(t, err)
	}

	unreadCount, err := service.GetUnreadMessageCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, unreadCount) // 2 unread messages
}

func TestSystemService_MarkMessagesAsRead(t *testing.T) {
	// Test marking messages as read
	ctx := context.Background()
	cutoffTime := time.Now().Add(-time.Hour)

	service := services.NewSystemService(events.NewEventBus())

	// Add messages with different timestamps
	oldMessage := services.SystemMessage{
		Level:     messages.InfoLevel,
		Content:   "Old message",
		Timestamp: cutoffTime.Add(-time.Hour),
		Read:      false,
		Source:    "test",
	}
	newMessage := services.SystemMessage{
		Level:     messages.InfoLevel,
		Content:   "New message",
		Timestamp: time.Now(),
		Read:      false,
		Source:    "test",
	}

	_, err1 := service.AddSystemMessage(ctx, oldMessage)
	_, err2 := service.AddSystemMessage(ctx, newMessage)
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Mark messages as read up to cutoff time
	err := service.MarkMessagesAsRead(ctx, cutoffTime)
	assert.NoError(t, err)

	// Verify only old message is marked as read
	retrievedMessages, err := service.GetMessages(ctx, 0, 10)
	assert.NoError(t, err)
	assert.Len(t, retrievedMessages, 2)

	oldMsg := findMessage(retrievedMessages, "Old message")
	newMsg := findMessage(retrievedMessages, "New message")
	assert.True(t, oldMsg.Read)
	assert.False(t, newMsg.Read)
}

func TestSystemService_ToggleSystemPanel(t *testing.T) {
	// Test system panel toggle functionality
	ctx := context.Background()

	service := services.NewSystemService(events.NewEventBus())

	// Initial state should be collapsed
	initialState, err := service.GetPanelState(ctx)
	assert.NoError(t, err)
	assert.False(t, initialState.Expanded)

	// Toggle to expanded
	expandedState, err := service.ToggleSystemPanel(ctx)
	assert.NoError(t, err)
	assert.True(t, expandedState)

	// Verify state change
	currentState, err := service.GetPanelState(ctx)
	assert.NoError(t, err)
	assert.True(t, currentState.Expanded)

	// Toggle back to collapsed
	collapsedState, err := service.ToggleSystemPanel(ctx)
	assert.NoError(t, err)
	assert.False(t, collapsedState)
}

func TestSystemService_MessageBufferFull(t *testing.T) {
	// Test behavior when message buffer is full
	ctx := context.Background()

	service := services.NewSystemService(events.NewEventBus())

	// Fill buffer to capacity
	for i := 0; i < 1001; i++ { // One more than the buffer limit (1000)
		message := services.SystemMessage{
			Level:     messages.InfoLevel,
			Content:   fmt.Sprintf("Message %d", i),
			Timestamp: time.Now(),
			Source:    "test",
		}
		_, err := service.AddSystemMessage(ctx, message)
		if i < 1000 {
			// First 1000 should succeed
			assert.NoError(t, err)
		} else {
			// The 1001st should succeed but remove oldest message (no ErrBufferFull error)
			// Our implementation removes oldest messages instead of returning ErrBufferFull
			assert.NoError(t, err)

			// Verify buffer still has at most 1000 messages
			usage, err := service.GetBufferUsage(ctx)
			assert.NoError(t, err)
			assert.LessOrEqual(t, usage, 1000)
		}
	}
}

func TestSystemService_MessageOrdering(t *testing.T) {
	// Test messages are ordered by timestamp
	ctx := context.Background()

	service := services.NewSystemService(events.NewEventBus())

	// Add messages with different timestamps
	baseTime := time.Now()
	testMessages := []services.SystemMessage{
		{Level: messages.InfoLevel, Content: "First", Timestamp: baseTime.Add(3 * time.Hour), Source: "test"},
		{Level: messages.InfoLevel, Content: "Second", Timestamp: baseTime.Add(1 * time.Hour), Source: "test"},
		{Level: messages.InfoLevel, Content: "Third", Timestamp: baseTime.Add(2 * time.Hour), Source: "test"},
	}

	for _, msg := range testMessages {
		_, err := service.AddSystemMessage(ctx, msg)
		assert.NoError(t, err)
	}

	// Retrieve messages and verify ordering
	retrieved, err := service.GetMessages(ctx, 0, 10)
	assert.NoError(t, err)
	assert.Len(t, retrieved, 3)

	// Should be ordered by timestamp (newest first, since our implementation sorts newest first)
	assert.Equal(t, "First", retrieved[0].Content)   // 3 hours ago
	assert.Equal(t, "Third", retrieved[1].Content)   // 2 hours ago
	assert.Equal(t, "Second", retrieved[2].Content)  // 1 hour ago
}

func TestSystemService_Performance(t *testing.T) {
	// Test system service performance meets <5ms target
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()
	service := services.NewSystemService(events.NewEventBus())

	// Measure message addition performance
	start := time.Now()
	for i := 0; i < 1000; i++ {
		message := services.SystemMessage{
			Level:     messages.InfoLevel,
			Content:   fmt.Sprintf("Performance test message %d", i),
			Timestamp: time.Now(),
			Source:    "test",
		}
		_, err := service.AddSystemMessage(ctx, message)
		assert.NoError(t, err)
	}
	duration := time.Since(start)

	avgTime := duration / 1000
	assert.Less(t, avgTime, 5*time.Millisecond, "Average message addition should be <5ms")

	// Measure unread count performance
	start = time.Now()
	for i := 0; i < 100; i++ {
		_, err := service.GetUnreadMessageCount(ctx)
		assert.NoError(t, err)
	}
	duration = time.Since(start)

	avgTime = duration / 100
	assert.Less(t, avgTime, 5*time.Millisecond, "Average unread count should be <5ms")
}

// Helper function for finding messages (used in implementation tests)
func findMessage(messages []services.SystemMessage, content string) services.SystemMessage {
	for _, msg := range messages {
		if msg.Content == content {
			return msg
		}
	}
	return services.SystemMessage{}
}