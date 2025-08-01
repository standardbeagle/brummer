package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Message represents a message in the queue
type Message struct {
	ID        string          `json:"id"`
	Channel   string          `json:"channel"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
	TTL       int             `json:"ttl"` // Time to live in seconds
	ExpiresAt time.Time       `json:"-"`   // Internal expiration time
}

// Subscription represents an active subscription to a channel
type Subscription struct {
	ID        string
	Channel   string
	MessageCh chan Message
	ctx       context.Context
	cancel    context.CancelFunc
}

// MessageQueue manages in-memory message queuing with TTL support
type MessageQueue struct {
	mu            sync.RWMutex
	messages      map[string][]Message                // channel -> messages
	subscriptions map[string]map[string]*Subscription // channel -> subscriptionID -> subscription
	cleanupTicker *time.Ticker
	cleanupDone   chan bool
}

// NewMessageQueue creates a new message queue
func NewMessageQueue() *MessageQueue {
	mq := &MessageQueue{
		messages:      make(map[string][]Message),
		subscriptions: make(map[string]map[string]*Subscription),
		cleanupDone:   make(chan bool),
	}

	// Start cleanup goroutine
	mq.cleanupTicker = time.NewTicker(10 * time.Second)
	go mq.cleanupExpiredMessages()

	return mq
}

// Stop stops the message queue and cleanup goroutine
func (mq *MessageQueue) Stop() {
	mq.cleanupTicker.Stop()
	close(mq.cleanupDone)
}

// Send adds a message to a channel
func (mq *MessageQueue) Send(channel, msgType string, payload json.RawMessage, ttl int) (*Message, error) {
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

	mq.mu.Lock()
	mq.messages[channel] = append(mq.messages[channel], msg)
	mq.mu.Unlock()

	// Broadcast to subscribers
	mq.broadcastToSubscribers(channel, msg)

	return &msg, nil
}

// Receive retrieves messages from a channel
func (mq *MessageQueue) Receive(channel string, limit int, blocking bool, timeout time.Duration) ([]Message, error) {
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
func (mq *MessageQueue) Subscribe(channel string) (*Subscription, error) {
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

	mq.mu.Lock()
	if _, exists := mq.subscriptions[channel]; !exists {
		mq.subscriptions[channel] = make(map[string]*Subscription)
	}
	mq.subscriptions[channel][sub.ID] = sub
	mq.mu.Unlock()

	return sub, nil
}

// Unsubscribe removes a subscription
func (mq *MessageQueue) Unsubscribe(subscriptionID string) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	// Find and remove subscription
	for channel, subs := range mq.subscriptions {
		if sub, exists := subs[subscriptionID]; exists {
			sub.cancel()
			close(sub.MessageCh)
			delete(subs, subscriptionID)

			// Clean up empty channel map
			if len(subs) == 0 {
				delete(mq.subscriptions, channel)
			}

			return nil
		}
	}

	return fmt.Errorf("subscription not found: %s", subscriptionID)
}

// ListChannels returns all active channels
func (mq *MessageQueue) ListChannels() []string {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	channelMap := make(map[string]bool)

	// Include channels with messages
	for channel := range mq.messages {
		channelMap[channel] = true
	}

	// Include channels with subscriptions
	for channel := range mq.subscriptions {
		channelMap[channel] = true
	}

	channels := make([]string, 0, len(channelMap))
	for channel := range channelMap {
		channels = append(channels, channel)
	}

	return channels
}

// Stats returns queue statistics
func (mq *MessageQueue) Stats() map[string]interface{} {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	totalMessages := 0
	channelStats := make(map[string]map[string]int)

	for channel, messages := range mq.messages {
		totalMessages += len(messages)

		stats := map[string]int{
			"messages":      len(messages),
			"subscriptions": 0,
		}

		if subs, exists := mq.subscriptions[channel]; exists {
			stats["subscriptions"] = len(subs)
		}

		channelStats[channel] = stats
	}

	// Add channels that only have subscriptions
	for channel, subs := range mq.subscriptions {
		if _, exists := channelStats[channel]; !exists {
			channelStats[channel] = map[string]int{
				"messages":      0,
				"subscriptions": len(subs),
			}
		}
	}

	return map[string]interface{}{
		"total_messages":      totalMessages,
		"total_channels":      len(channelStats),
		"total_subscriptions": mq.countSubscriptions(),
		"channels":            channelStats,
	}
}

// Helper methods

func (mq *MessageQueue) getMessages(channel string, limit int) []Message {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	messages, exists := mq.messages[channel]
	if !exists {
		return []Message{}
	}

	// Filter out expired messages
	validMessages := make([]Message, 0)
	now := time.Now()
	for _, msg := range messages {
		if msg.ExpiresAt.After(now) {
			validMessages = append(validMessages, msg)
		}
	}

	// Apply limit
	if len(validMessages) > limit {
		return validMessages[len(validMessages)-limit:]
	}

	return validMessages
}

func (mq *MessageQueue) broadcastToSubscribers(channel string, msg Message) {
	mq.mu.RLock()
	subs, exists := mq.subscriptions[channel]
	if !exists {
		mq.mu.RUnlock()
		return
	}

	// Create a copy of the subscription map to avoid holding the lock during send
	activeSubs := make([]*Subscription, 0, len(subs))
	for _, sub := range subs {
		activeSubs = append(activeSubs, sub)
	}
	mq.mu.RUnlock()

	// Send to subscribers without holding the lock
	for _, sub := range activeSubs {
		select {
		case sub.MessageCh <- msg:
			// Message sent successfully
		case <-sub.ctx.Done():
			// Subscription cancelled, skip
		default:
			// Channel full, skip this subscriber
		}
	}
}

func (mq *MessageQueue) cleanupExpiredMessages() {
	for {
		select {
		case <-mq.cleanupTicker.C:
			mq.removeExpiredMessages()
		case <-mq.cleanupDone:
			return
		}
	}
}

func (mq *MessageQueue) removeExpiredMessages() {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	now := time.Now()

	for channel, messages := range mq.messages {
		validMessages := make([]Message, 0)

		for _, msg := range messages {
			if msg.ExpiresAt.After(now) {
				validMessages = append(validMessages, msg)
			}
		}

		if len(validMessages) == 0 {
			delete(mq.messages, channel)
		} else {
			mq.messages[channel] = validMessages
		}
	}
}

func (mq *MessageQueue) countSubscriptions() int {
	count := 0
	for _, subs := range mq.subscriptions {
		count += len(subs)
	}
	return count
}

