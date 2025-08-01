package mcp

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageQueueLockFree_SendReceive(t *testing.T) {
	mq := NewMessageQueueLockFree()
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

func TestMessageQueueLockFree_ConcurrentSend(t *testing.T) {
	mq := NewMessageQueueLockFree()
	defer mq.Stop()

	channel := "concurrent-channel"
	numGoroutines := 100
	messagesPerGoroutine := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent senders
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				payload := json.RawMessage(
					`{"goroutine": ` + string(rune('0'+goroutineID%10)) +
						`, "message": ` + string(rune('0'+j)) + `}`)
				_, err := mq.Send(channel, "test", payload, 3600)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all messages were stored
	messages, err := mq.Receive(channel, 10000, false, 0)
	require.NoError(t, err)
	assert.Len(t, messages, numGoroutines*messagesPerGoroutine)
}

func TestMessageQueueLockFree_ConcurrentSubscribe(t *testing.T) {
	mq := NewMessageQueueLockFree()
	defer mq.Stop()

	channel := "sub-channel"
	numSubscribers := 50

	var wg sync.WaitGroup
	wg.Add(numSubscribers)

	subscriptions := make([]*Subscription, numSubscribers)
	receivedCounts := make([]int, numSubscribers)
	var mu sync.Mutex

	// Create concurrent subscribers
	for i := 0; i < numSubscribers; i++ {
		go func(subID int) {
			defer wg.Done()

			sub, err := mq.Subscribe(channel)
			require.NoError(t, err)
			subscriptions[subID] = sub

			// Count received messages
			go func() {
				for range sub.MessageCh {
					mu.Lock()
					receivedCounts[subID]++
					mu.Unlock()
				}
			}()
		}(i)
	}

	wg.Wait()

	// Send messages
	numMessages := 10
	for i := 0; i < numMessages; i++ {
		_, err := mq.Send(channel, "test", json.RawMessage(`{}`), 3600)
		require.NoError(t, err)
	}

	// Give time for messages to propagate
	time.Sleep(100 * time.Millisecond)

	// Verify each subscriber received messages
	mu.Lock()
	for i, count := range receivedCounts {
		assert.Equal(t, numMessages, count, "Subscriber %d should receive all messages", i)
	}
	mu.Unlock()

	// Cleanup subscriptions
	for _, sub := range subscriptions {
		err := mq.Unsubscribe(sub.ID)
		assert.NoError(t, err)
	}
}

func TestMessageQueueLockFree_ConcurrentReceive(t *testing.T) {
	mq := NewMessageQueueLockFree()
	defer mq.Stop()

	channel := "receive-channel"

	// Pre-populate messages
	numMessages := 1000
	for i := 0; i < numMessages; i++ {
		_, err := mq.Send(channel, "test", json.RawMessage(`{}`), 3600)
		require.NoError(t, err)
	}

	// Concurrent receivers
	numReceivers := 10
	var wg sync.WaitGroup
	wg.Add(numReceivers)

	totalReceived := 0
	var mu sync.Mutex

	for i := 0; i < numReceivers; i++ {
		go func() {
			defer wg.Done()

			messages, err := mq.Receive(channel, 200, false, 0)
			assert.NoError(t, err)

			mu.Lock()
			totalReceived += len(messages)
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Each receiver should see up to the limit they requested (200 messages each)
	// Since the queue has 1000 messages total, each receiver can get up to 200
	assert.LessOrEqual(t, totalReceived, 200*numReceivers)
	assert.Greater(t, totalReceived, 0)
}

func TestMessageQueueLockFree_StatsAccuracy(t *testing.T) {
	mq := NewMessageQueueLockFree()
	defer mq.Stop()

	// Send messages to multiple channels
	channels := []string{"ch1", "ch2", "ch3"}
	messagesPerChannel := 5

	for _, ch := range channels {
		for i := 0; i < messagesPerChannel; i++ {
			_, err := mq.Send(ch, "test", json.RawMessage(`{}`), 3600)
			require.NoError(t, err)
		}
	}

	// Create subscriptions
	subs := make([]*Subscription, 0)
	for _, ch := range channels {
		sub, err := mq.Subscribe(ch)
		require.NoError(t, err)
		subs = append(subs, sub)
	}

	// Check stats
	stats := mq.Stats()
	assert.Equal(t, int64(len(channels)*messagesPerChannel), stats["total_messages"])
	assert.Equal(t, int64(len(channels)), stats["total_channels"])
	assert.Equal(t, int64(len(subs)), stats["total_subscriptions"])

	// Cleanup
	for _, sub := range subs {
		err := mq.Unsubscribe(sub.ID)
		assert.NoError(t, err)
	}
}

func TestMessageQueueLockFree_MessageOrdering(t *testing.T) {
	mq := NewMessageQueueLockFree()
	defer mq.Stop()

	channel := "order-channel"

	// Send messages in order
	for i := 0; i < 10; i++ {
		payload := json.RawMessage(`{"order": ` + string(rune('0'+i)) + `}`)
		_, err := mq.Send(channel, "test", payload, 3600)
		require.NoError(t, err)
	}

	// Receive and verify order
	messages, err := mq.Receive(channel, 100, false, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 10)

	// Messages should be in FIFO order
	for i, msg := range messages {
		var data map[string]int
		err := json.Unmarshal(msg.Payload, &data)
		require.NoError(t, err)
		assert.Equal(t, i, data["order"])
	}
}

func TestMessageQueueLockFree_TTLCleanup(t *testing.T) {
	mq := NewMessageQueueLockFree()
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

	// Force cleanup
	mq.removeExpiredMessages()

	// Message should be expired
	messages, err = mq.Receive(channel, 10, false, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 0)

	// Stats should reflect removal
	stats := mq.Stats()
	assert.Equal(t, int64(0), stats["total_messages"])
}

func TestMessageQueueLockFree_HighContention(t *testing.T) {
	mq := NewMessageQueueLockFree()
	defer mq.Stop()

	channel := "contention-channel"
	numOperations := 100

	var wg sync.WaitGroup

	// Multiple senders
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_, err := mq.Send(channel, "test", json.RawMessage(`{}`), 60)
				assert.NoError(t, err)
			}
		}(i)
	}

	// Multiple receivers
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_, err := mq.Receive(channel, 10, false, 0)
				assert.NoError(t, err)
			}
		}()
	}

	// Multiple subscribers
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				sub, err := mq.Subscribe(channel)
				if err == nil {
					time.Sleep(time.Microsecond)
					mq.Unsubscribe(sub.ID)
				}
			}
		}()
	}

	wg.Wait()

	// Verify final state
	stats := mq.Stats()
	assert.Greater(t, stats["total_messages"], int64(0))
}

