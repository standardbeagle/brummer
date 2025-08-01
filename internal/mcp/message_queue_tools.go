package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/standardbeagle/brummer/pkg/events"
)

// Message queue tool handlers

// handleQueueSend handles the queue_send tool
func (s *MCPServer) handleQueueSend(args json.RawMessage) (interface{}, error) {
	var params struct {
		Channel string          `json:"channel"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
		TTL     int             `json:"ttl"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}

	if params.Type == "" {
		params.Type = "message"
	}

	// Send message to queue
	msg, err := s.messageQueue.Send(params.Channel, params.Type, params.Payload, params.TTL)
	if err != nil {
		return nil, err
	}

	// Broadcast via WebSocket to proxy server
	s.broadcastMessageToWebSockets(msg)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Message sent to channel '%s' with ID: %s", params.Channel, msg.ID),
			},
		},
		"message": msg,
	}, nil
}

// handleQueueReceive handles the queue_receive tool
func (s *MCPServer) handleQueueReceive(args json.RawMessage) (interface{}, error) {
	var params struct {
		Channel  string `json:"channel"`
		Limit    int    `json:"limit"`
		Blocking bool   `json:"blocking"`
		Timeout  int    `json:"timeout"` // in seconds
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}

	if params.Limit <= 0 {
		params.Limit = 100
	}

	timeout := 30 * time.Second // Default timeout
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Second
	}

	messages, err := s.messageQueue.Receive(params.Channel, params.Limit, params.Blocking, timeout)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Retrieved %d messages from channel '%s'", len(messages), params.Channel),
			},
		},
		"messages": messages,
	}, nil
}

// handleQueueSubscribe handles the queue_subscribe tool with streaming
func (s *MCPServer) handleQueueSubscribe(args json.RawMessage, session *ClientSession) error {
	var params struct {
		Channel string `json:"channel"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	// Create subscription
	sub, err := s.messageQueue.Subscribe(params.Channel)
	if err != nil {
		return err
	}

	// Store subscription ID in session context for cleanup
	session.Context = context.WithValue(session.Context, "subscriptionID", sub.ID)

	// Send initial response
	initialResponse := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Subscribed to channel '%s'. Subscription ID: %s", params.Channel, sub.ID),
			},
		},
		"subscriptionId": sub.ID,
	}

	// Send initial content
	if err := s.sendStreamingContent(session, initialResponse); err != nil {
		s.messageQueue.Unsubscribe(sub.ID)
		return err
	}

	// Stream messages
	go func() {
		defer s.messageQueue.Unsubscribe(sub.ID)

		for {
			select {
			case msg, ok := <-sub.MessageCh:
				if !ok {
					return
				}

				// Stream the message
				content := map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": fmt.Sprintf("New message on channel '%s'", params.Channel),
						},
					},
					"message": msg,
				}

				if err := s.sendStreamingContent(session, content); err != nil {
					return
				}

			case <-session.Context.Done():
				return
			}
		}
	}()

	return nil
}

// handleQueueUnsubscribe handles the queue_unsubscribe tool
func (s *MCPServer) handleQueueUnsubscribe(args json.RawMessage) (interface{}, error) {
	var params struct {
		SubscriptionID string `json:"subscription_id"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.SubscriptionID == "" {
		return nil, fmt.Errorf("subscription_id is required")
	}

	err := s.messageQueue.Unsubscribe(params.SubscriptionID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Unsubscribed from subscription ID: %s", params.SubscriptionID),
			},
		},
	}, nil
}

// handleQueueListChannels handles the queue_list_channels tool
func (s *MCPServer) handleQueueListChannels(args json.RawMessage) (interface{}, error) {
	channels := s.messageQueue.ListChannels()

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Found %d active channels", len(channels)),
			},
		},
		"channels": channels,
	}, nil
}

// handleQueueStats handles the queue_stats tool
func (s *MCPServer) handleQueueStats(args json.RawMessage) (interface{}, error) {
	stats := s.messageQueue.Stats()

	// Format stats for display
	formattedStats := fmt.Sprintf(
		"Queue Statistics:\n"+
			"- Total Messages: %v\n"+
			"- Total Channels: %v\n"+
			"- Total Subscriptions: %v",
		stats["total_messages"],
		stats["total_channels"],
		stats["total_subscriptions"],
	)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": formattedStats,
			},
		},
		"stats": stats,
	}, nil
}

// broadcastMessageToWebSockets sends a message queue message through WebSocket
func (s *MCPServer) broadcastMessageToWebSockets(msg *Message) {
	// Send to proxy server WebSocket clients
	if s.proxyServer != nil {
		s.proxyServer.BroadcastToWebSockets("queue_message", map[string]interface{}{
			"message": msg,
		})
	}
}

// Helper function to send streaming content
func (s *MCPServer) sendStreamingContent(session *ClientSession, content interface{}) error {
	// Create a streaming response
	response := &JSONRPCMessage{
		Jsonrpc: "2.0",
		Result:  content,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	// Write response
	session.mu.Lock()
	defer session.mu.Unlock()

	if _, err := session.ResponseWriter.Write(data); err != nil {
		return err
	}

	if _, err := session.ResponseWriter.Write([]byte("\n")); err != nil {
		return err
	}

	if session.Flusher != nil {
		session.Flusher.Flush()
	}

	return nil
}

// Register message queue tools
func (s *MCPServer) registerMessageQueueTools() {
	s.tools["queue_send"] = MCPTool{
		Name:        "queue_send",
		Description: "Send a message to a channel",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"channel": {
					"type": "string",
					"description": "The channel to send the message to"
				},
				"type": {
					"type": "string",
					"description": "The message type (optional, defaults to 'message')"
				},
				"payload": {
					"type": "object",
					"description": "The message payload"
				},
				"ttl": {
					"type": "integer",
					"description": "Time to live in seconds (optional, defaults to 3600)"
				}
			},
			"required": ["channel", "payload"]
		}`),
		Handler: s.handleQueueSend,
	}

	s.tools["queue_receive"] = MCPTool{
		Name:        "queue_receive",
		Description: "Receive messages from a channel",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"channel": {
					"type": "string",
					"description": "The channel to receive messages from"
				},
				"limit": {
					"type": "integer",
					"description": "Maximum number of messages to retrieve (optional, defaults to 100)"
				},
				"blocking": {
					"type": "boolean",
					"description": "Whether to wait for messages if none are available (optional, defaults to false)"
				},
				"timeout": {
					"type": "integer",
					"description": "Timeout in seconds for blocking mode (optional, defaults to 30)"
				}
			},
			"required": ["channel"]
		}`),
		Handler: s.handleQueueReceive,
	}

	s.tools["queue_subscribe"] = MCPTool{
		Name:        "queue_subscribe",
		Description: "Subscribe to real-time updates from a channel",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"channel": {
					"type": "string",
					"description": "The channel to subscribe to"
				}
			},
			"required": ["channel"]
		}`),
		Streaming: false,
		Handler: func(args json.RawMessage) (interface{}, error) {
			// For now, return subscription info without streaming
			var params struct {
				Channel string `json:"channel"`
			}

			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid parameters: %w", err)
			}

			if params.Channel == "" {
				return nil, fmt.Errorf("channel is required")
			}

			// Create subscription
			sub, err := s.messageQueue.Subscribe(params.Channel)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": fmt.Sprintf("Subscribed to channel '%s'. Subscription ID: %s", params.Channel, sub.ID),
					},
				},
				"subscriptionId": sub.ID,
			}, nil
		},
	}

	s.tools["queue_unsubscribe"] = MCPTool{
		Name:        "queue_unsubscribe",
		Description: "Unsubscribe from a channel",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"subscription_id": {
					"type": "string",
					"description": "The subscription ID to unsubscribe"
				}
			},
			"required": ["subscription_id"]
		}`),
		Handler: s.handleQueueUnsubscribe,
	}

	s.tools["queue_list_channels"] = MCPTool{
		Name:        "queue_list_channels",
		Description: "List all active channels",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
		Handler: s.handleQueueListChannels,
	}

	s.tools["queue_stats"] = MCPTool{
		Name:        "queue_stats",
		Description: "Get queue statistics",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
		Handler: s.handleQueueStats,
	}
}

// WebSocket command handler for incoming queue messages
func (s *MCPServer) handleWebSocketQueueMessage(data map[string]interface{}) error {
	// Extract message fields
	channel, _ := data["channel"].(string)
	msgType, _ := data["type"].(string)
	payload, _ := json.Marshal(data["payload"])
	ttl, _ := data["ttl"].(float64)

	if channel == "" {
		return fmt.Errorf("channel is required")
	}

	// Send to message queue
	_, err := s.messageQueue.Send(channel, msgType, payload, int(ttl))
	return err
}

// setupMessageQueueEventHandlers sets up event listeners for WebSocket integration
func (s *MCPServer) setupMessageQueueEventHandlers() {
	// Listen for queue messages from WebSocket
	s.eventBus.Subscribe(events.EventType("queue.message"), func(e events.Event) {
		if err := s.handleWebSocketQueueMessage(e.Data); err != nil {
			// Log error but don't crash
			fmt.Printf("Error handling WebSocket queue message: %v\n", err)
		}
	})
}
