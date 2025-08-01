package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageQueue_SendReceive(t *testing.T) {
	mq := NewMessageQueue()
	defer mq.Stop()

	// Test sending a message
	channel := "test-channel"
	msgType := "test-message"
	payload := json.RawMessage(`{"data": "test"}`)
	ttl := 3600

	msg, err := mq.Send(channel, msgType, payload, ttl)
	require.NoError(t, err)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, channel, msg.Channel)
	assert.Equal(t, msgType, msg.Type)
	assert.Equal(t, payload, msg.Payload)
	assert.Equal(t, ttl, msg.TTL)

	// Test receiving messages
	messages, err := mq.Receive(channel, 10, false, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, msg.ID, messages[0].ID)
}

func TestMessageQueue_EmptyChannel(t *testing.T) {
	mq := NewMessageQueue()
	defer mq.Stop()

	// Test receiving from empty channel
	messages, err := mq.Receive("empty-channel", 10, false, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 0)
}

func TestMessageQueue_MultipleMessages(t *testing.T) {
	mq := NewMessageQueue()
	defer mq.Stop()

	channel := "multi-channel"

	// Send multiple messages
	for i := 0; i < 5; i++ {
		payload := json.RawMessage(`{"index": ` + string(rune('0'+i)) + `}`)
		_, err := mq.Send(channel, "test", payload, 3600)
		require.NoError(t, err)
	}

	// Receive with limit
	messages, err := mq.Receive(channel, 3, false, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 3)

	// Receive all
	messages, err = mq.Receive(channel, 100, false, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 5)
}

func TestMessageQueue_TTLExpiration(t *testing.T) {
	mq := NewMessageQueue()
	defer mq.Stop()

	channel := "ttl-channel"

	// Send message with 1 second TTL
	_, err := mq.Send(channel, "expiring", json.RawMessage(`{}`), 1)
	require.NoError(t, err)

	// Message should be available immediately
	messages, err := mq.Receive(channel, 10, false, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 1)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Message should be expired
	messages, err = mq.Receive(channel, 10, false, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 0)
}

func TestMessageQueue_Subscription(t *testing.T) {
	mq := NewMessageQueue()
	defer mq.Stop()

	channel := "sub-channel"

	// Create subscription
	sub, err := mq.Subscribe(channel)
	require.NoError(t, err)
	assert.NotEmpty(t, sub.ID)
	assert.Equal(t, channel, sub.Channel)

	// Send message
	payload := json.RawMessage(`{"test": "data"}`)
	msg, err := mq.Send(channel, "test", payload, 3600)
	require.NoError(t, err)

	// Receive from subscription
	select {
	case receivedMsg := <-sub.MessageCh:
		assert.Equal(t, msg.ID, receivedMsg.ID)
		assert.Equal(t, msg.Channel, receivedMsg.Channel)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for message")
	}

	// Unsubscribe
	err = mq.Unsubscribe(sub.ID)
	require.NoError(t, err)
}

func TestMessageQueue_BlockingReceive(t *testing.T) {
	mq := NewMessageQueue()
	defer mq.Stop()

	channel := "blocking-channel"

	// Start blocking receive in goroutine
	done := make(chan bool)
	var messages []Message
	var err error

	go func() {
		messages, err = mq.Receive(channel, 10, true, 1*time.Second)
		done <- true
	}()

	// Send message after delay
	time.Sleep(100 * time.Millisecond)
	_, sendErr := mq.Send(channel, "test", json.RawMessage(`{}`), 3600)
	require.NoError(t, sendErr)

	// Wait for receive to complete
	<-done
	require.NoError(t, err)
	assert.Len(t, messages, 1)
}

func TestMessageQueue_BlockingReceiveTimeout(t *testing.T) {
	mq := NewMessageQueue()
	defer mq.Stop()

	channel := "timeout-channel"

	// Blocking receive with short timeout
	start := time.Now()
	messages, err := mq.Receive(channel, 10, true, 500*time.Millisecond)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, messages, 0)
	assert.True(t, duration >= 500*time.Millisecond)
	assert.True(t, duration < 600*time.Millisecond)
}

func TestMessageQueue_ListChannels(t *testing.T) {
	mq := NewMessageQueue()
	defer mq.Stop()

	// Initially empty
	channels := mq.ListChannels()
	assert.Len(t, channels, 0)

	// Add messages to channels
	_, err := mq.Send("channel1", "test", json.RawMessage(`{}`), 3600)
	require.NoError(t, err)
	_, err = mq.Send("channel2", "test", json.RawMessage(`{}`), 3600)
	require.NoError(t, err)

	// Create subscription without messages
	sub, err := mq.Subscribe("channel3")
	require.NoError(t, err)
	defer mq.Unsubscribe(sub.ID)

	channels = mq.ListChannels()
	assert.Len(t, channels, 3)
	assert.Contains(t, channels, "channel1")
	assert.Contains(t, channels, "channel2")
	assert.Contains(t, channels, "channel3")
}

func TestMessageQueue_Stats(t *testing.T) {
	mq := NewMessageQueue()
	defer mq.Stop()

	// Initial stats
	stats := mq.Stats()
	assert.Equal(t, 0, stats["total_messages"])
	assert.Equal(t, 0, stats["total_channels"])
	assert.Equal(t, 0, stats["total_subscriptions"])

	// Add data
	_, err := mq.Send("channel1", "test", json.RawMessage(`{}`), 3600)
	require.NoError(t, err)
	_, err = mq.Send("channel1", "test", json.RawMessage(`{}`), 3600)
	require.NoError(t, err)
	_, err = mq.Send("channel2", "test", json.RawMessage(`{}`), 3600)
	require.NoError(t, err)

	sub1, err := mq.Subscribe("channel1")
	require.NoError(t, err)
	defer mq.Unsubscribe(sub1.ID)

	sub2, err := mq.Subscribe("channel2")
	require.NoError(t, err)
	defer mq.Unsubscribe(sub2.ID)

	// Check stats
	stats = mq.Stats()
	assert.Equal(t, 3, stats["total_messages"])
	assert.Equal(t, 2, stats["total_channels"])
	assert.Equal(t, 2, stats["total_subscriptions"])

	channelStats := stats["channels"].(map[string]map[string]int)
	assert.Equal(t, 2, channelStats["channel1"]["messages"])
	assert.Equal(t, 1, channelStats["channel1"]["subscriptions"])
	assert.Equal(t, 1, channelStats["channel2"]["messages"])
	assert.Equal(t, 1, channelStats["channel2"]["subscriptions"])
}

func TestMessageQueue_InvalidParameters(t *testing.T) {
	mq := NewMessageQueue()
	defer mq.Stop()

	// Test empty channel
	_, err := mq.Send("", "test", json.RawMessage(`{}`), 3600)
	assert.Error(t, err)

	_, err = mq.Receive("", 10, false, 0)
	assert.Error(t, err)

	_, err = mq.Subscribe("")
	assert.Error(t, err)

	// Test invalid subscription ID
	err = mq.Unsubscribe("invalid-id")
	assert.Error(t, err)
}

func TestMessageQueue_CleanupOnStop(t *testing.T) {
	mq := NewMessageQueue()

	// Send message with short TTL
	_, err := mq.Send("test", "test", json.RawMessage(`{}`), 1)
	require.NoError(t, err)

	// Stop the queue
	mq.Stop()

	// Cleanup ticker should be stopped
	// This test mainly ensures Stop() doesn't panic
}

func TestMessageQueue_ConcurrentAccess(t *testing.T) {
	mq := NewMessageQueue()
	defer mq.Stop()

	channel := "concurrent-channel"
	done := make(chan bool, 10)

	// Multiple senders
	for i := 0; i < 5; i++ {
		go func(index int) {
			payload := json.RawMessage(`{"sender": ` + string(rune('0'+index)) + `}`)
			_, err := mq.Send(channel, "test", payload, 3600)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Multiple receivers
	for i := 0; i < 5; i++ {
		go func() {
			_, err := mq.Receive(channel, 10, false, 0)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all operations
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	messages, err := mq.Receive(channel, 100, false, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 5)
}

