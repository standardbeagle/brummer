package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/google/uuid"
)

// MessageList is a lock-free linked list node for messages
type MessageNode struct {
	Message Message
	Next    *MessageNode
}

// ChannelMessages holds messages for a channel using atomic pointer
type ChannelMessages struct {
	head atomic.Pointer[MessageNode]
	tail atomic.Pointer[MessageNode]
}

// SubscriptionMap wraps sync.Map for type safety
type SubscriptionMap struct {
	m sync.Map
}

func (sm *SubscriptionMap) Load(key string) (*Subscription, bool) {
	val, ok := sm.m.Load(key)
	if !ok {
		return nil, false
	}
	return val.(*Subscription), true
}

func (sm *SubscriptionMap) Store(key string, sub *Subscription) {
	sm.m.Store(key, sub)
}

func (sm *SubscriptionMap) Delete(key string) {
	sm.m.Delete(key)
}

func (sm *SubscriptionMap) Range(f func(key string, sub *Subscription) bool) {
	sm.m.Range(func(k, v interface{}) bool {
		return f(k.(string), v.(*Subscription))
	})
}

// MessageQueueLockFree is a lock-free implementation of the message queue
type MessageQueueLockFree struct {
	// Channel -> Messages mapping using sync.Map
	channels sync.Map // string -> *ChannelMessages

	// Channel -> Subscriptions mapping using sync.Map
	subscriptions sync.Map // string -> *SubscriptionMap

	// Cleanup ticker and control
	cleanupTicker *time.Ticker
	cleanupDone   chan bool

	// Stats counters (atomic)
	totalMessages atomic.Int64
	totalChannels atomic.Int64
	totalSubs     atomic.Int64
}

// NewMessageQueueLockFree creates a new lock-free message queue
func NewMessageQueueLockFree() *MessageQueueLockFree {
	mq := &MessageQueueLockFree{
		cleanupDone: make(chan bool),
	}

	// Start cleanup goroutine
	mq.cleanupTicker = time.NewTicker(10 * time.Second)
	go mq.cleanupExpiredMessages()

	return mq
}

// Stop stops the message queue and cleanup goroutine
func (mq *MessageQueueLockFree) Stop() {
	mq.cleanupTicker.Stop()
	close(mq.cleanupDone)
}

// Send adds a message to a channel
func (mq *MessageQueueLockFree) Send(channel, msgType string, payload json.RawMessage, ttl int) (*Message, error) {
	if channel == "" {
		return nil, fmt.Errorf("channel cannot be empty")
	}

	// Default TTL of 1 hour if not specified
	if ttl <= 0 {
		ttl = 3600
	}

	msg := Message{
		ID:        uuid.New().String(),
		Channel:   channel,
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
		TTL:       ttl,
		ExpiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
	}

	// Get or create channel messages
	channelMsgs := mq.getOrCreateChannel(channel)

	// Create new node
	newNode := &MessageNode{Message: msg}

	// Add to the end of the list atomically
	for {
		tail := channelMsgs.tail.Load()
		if tail == nil {
			// Empty list, try to set both head and tail
			if channelMsgs.head.CompareAndSwap(nil, newNode) {
				channelMsgs.tail.Store(newNode)
				break
			}
			// Someone else added first node, retry
			continue
		}

		// Try to update the tail's next pointer
		if atomic.CompareAndSwapPointer(
			(*unsafe.Pointer)(unsafe.Pointer(&tail.Next)),
			unsafe.Pointer(nil),
			unsafe.Pointer(newNode),
		) {
			// Successfully linked, now update tail
			channelMsgs.tail.CompareAndSwap(tail, newNode)
			break
		}
		// Tail was updated by someone else, help them and retry
		channelMsgs.tail.CompareAndSwap(tail, tail.Next)
	}

	// Update stats
	mq.totalMessages.Add(1)

	// Broadcast to subscribers
	mq.broadcastToSubscribers(channel, msg)

	return &msg, nil
}

// Receive retrieves messages from a channel
func (mq *MessageQueueLockFree) Receive(channel string, limit int, blocking bool, timeout time.Duration) ([]Message, error) {
	if channel == "" {
		return nil, fmt.Errorf("channel cannot be empty")
	}

	if limit <= 0 {
		limit = 100 // Default limit
	}

	// Non-blocking mode
	if !blocking {
		return mq.getMessages(channel, limit), nil
	}

	// Blocking mode - wait for at least one message
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create temporary subscription
	sub, err := mq.Subscribe(channel)
	if err != nil {
		return nil, err
	}
	defer mq.Unsubscribe(sub.ID)

	// Check for existing messages first
	messages := mq.getMessages(channel, limit)
	if len(messages) > 0 {
		return messages, nil
	}

	// Wait for new messages
	select {
	case msg := <-sub.MessageCh:
		return []Message{msg}, nil
	case <-ctx.Done():
		return []Message{}, nil // Return empty array on timeout
	}
}

// Subscribe creates a subscription to a channel
func (mq *MessageQueueLockFree) Subscribe(channel string) (*Subscription, error) {
	if channel == "" {
		return nil, fmt.Errorf("channel cannot be empty")
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := &Subscription{
		ID:        uuid.New().String(),
		Channel:   channel,
		MessageCh: make(chan Message, 100), // Buffered channel
		ctx:       ctx,
		cancel:    cancel,
	}

	// Get or create subscription map for channel
	subsMap := mq.getOrCreateSubscriptions(channel)
	subsMap.Store(sub.ID, sub)

	// Update stats
	mq.totalSubs.Add(1)

	return sub, nil
}

// Unsubscribe removes a subscription
func (mq *MessageQueueLockFree) Unsubscribe(subscriptionID string) error {
	var found bool

	// Search all channels for the subscription
	mq.subscriptions.Range(func(channel, value interface{}) bool {
		subsMap := value.(*SubscriptionMap)
		if sub, ok := subsMap.Load(subscriptionID); ok {
			sub.cancel()
			close(sub.MessageCh)
			subsMap.Delete(subscriptionID)
			found = true

			// Update stats
			mq.totalSubs.Add(-1)

			// Check if channel has no more subscriptions
			empty := true
			subsMap.Range(func(_ string, _ *Subscription) bool {
				empty = false
				return false
			})

			if empty {
				mq.subscriptions.Delete(channel)
			}

			return false // Stop iteration
		}
		return true // Continue iteration
	})

	if !found {
		return fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	return nil
}

// ListChannels returns all active channels
func (mq *MessageQueueLockFree) ListChannels() []string {
	channelMap := make(map[string]bool)

	// Include channels with messages
	mq.channels.Range(func(channel, _ interface{}) bool {
		channelMap[channel.(string)] = true
		return true
	})

	// Include channels with subscriptions
	mq.subscriptions.Range(func(channel, _ interface{}) bool {
		channelMap[channel.(string)] = true
		return true
	})

	channels := make([]string, 0, len(channelMap))
	for channel := range channelMap {
		channels = append(channels, channel)
	}

	return channels
}

// Stats returns queue statistics
func (mq *MessageQueueLockFree) Stats() map[string]interface{} {
	channelStats := make(map[string]map[string]int)

	// Count messages per channel
	mq.channels.Range(func(channel, value interface{}) bool {
		channelName := channel.(string)
		channelMsgs := value.(*ChannelMessages)

		count := 0
		now := time.Now()
		node := channelMsgs.head.Load()

		for node != nil {
			if node.Message.ExpiresAt.After(now) {
				count++
			}
			node = node.Next
		}

		channelStats[channelName] = map[string]int{
			"messages":      count,
			"subscriptions": 0,
		}
		return true
	})

	// Count subscriptions per channel
	mq.subscriptions.Range(func(channel, value interface{}) bool {
		channelName := channel.(string)
		subsMap := value.(*SubscriptionMap)

		count := 0
		subsMap.Range(func(_ string, _ *Subscription) bool {
			count++
			return true
		})

		if stats, exists := channelStats[channelName]; exists {
			stats["subscriptions"] = count
		} else {
			channelStats[channelName] = map[string]int{
				"messages":      0,
				"subscriptions": count,
			}
		}
		return true
	})

	return map[string]interface{}{
		"total_messages":      mq.totalMessages.Load(),
		"total_channels":      int64(len(channelStats)),
		"total_subscriptions": mq.totalSubs.Load(),
		"channels":            channelStats,
	}
}

// Helper methods

func (mq *MessageQueueLockFree) getOrCreateChannel(channel string) *ChannelMessages {
	val, _ := mq.channels.LoadOrStore(channel, &ChannelMessages{})
	return val.(*ChannelMessages)
}

func (mq *MessageQueueLockFree) getOrCreateSubscriptions(channel string) *SubscriptionMap {
	val, _ := mq.subscriptions.LoadOrStore(channel, &SubscriptionMap{})
	return val.(*SubscriptionMap)
}

func (mq *MessageQueueLockFree) getMessages(channel string, limit int) []Message {
	val, ok := mq.channels.Load(channel)
	if !ok {
		return []Message{}
	}

	channelMsgs := val.(*ChannelMessages)
	messages := make([]Message, 0)
	now := time.Now()

	// Traverse the linked list
	node := channelMsgs.head.Load()
	for node != nil && len(messages) < limit {
		if node.Message.ExpiresAt.After(now) {
			messages = append(messages, node.Message)
		}
		node = node.Next
	}

	// Return last 'limit' messages
	if len(messages) > limit {
		return messages[len(messages)-limit:]
	}

	return messages
}

func (mq *MessageQueueLockFree) broadcastToSubscribers(channel string, msg Message) {
	val, ok := mq.subscriptions.Load(channel)
	if !ok {
		return
	}

	subsMap := val.(*SubscriptionMap)
	subsMap.Range(func(_ string, sub *Subscription) bool {
		// Use defer to catch potential panic from sending on closed channel
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Channel was closed, skip this subscriber
				}
			}()

			select {
			case sub.MessageCh <- msg:
				// Message sent successfully
			case <-sub.ctx.Done():
				// Subscription cancelled, skip
			default:
				// Channel full, skip this subscriber
			}
		}()
		return true
	})
}

func (mq *MessageQueueLockFree) cleanupExpiredMessages() {
	for {
		select {
		case <-mq.cleanupTicker.C:
			mq.removeExpiredMessages()
		case <-mq.cleanupDone:
			return
		}
	}
}

func (mq *MessageQueueLockFree) removeExpiredMessages() {
	now := time.Now()

	mq.channels.Range(func(channel, value interface{}) bool {
		channelMsgs := value.(*ChannelMessages)

		// Remove expired messages from the head
		for {
			head := channelMsgs.head.Load()
			if head == nil || head.Message.ExpiresAt.After(now) {
				break
			}

			// Try to remove the head
			if channelMsgs.head.CompareAndSwap(head, head.Next) {
				mq.totalMessages.Add(-1)

				// If we removed the last node, update tail
				if head.Next == nil {
					channelMsgs.tail.CompareAndSwap(head, nil)
				}
			}
		}

		// Check if channel is empty
		if channelMsgs.head.Load() == nil {
			mq.channels.Delete(channel)
		}

		return true
	})
}

