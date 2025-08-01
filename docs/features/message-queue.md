# Message Queue Feature

The Brummer Message Queue provides a bidirectional, in-memory message queuing system accessible through MCP tools. It enables real-time communication between AI assistants and browser-based applications via WebSocket integration.

## Features

- **In-memory message storage** with TTL (Time To Live) support
- **Channel-based messaging** for topic separation
- **Real-time subscriptions** with buffered channels
- **WebSocket integration** for browser communication
- **Lock-free operations** using atomic primitives and sync.Map
- **Automatic cleanup** of expired messages

## MCP Tools

### queue_send
Send a message to a specific channel.

```json
{
  "channel": "notifications",
  "type": "alert",
  "payload": {
    "message": "Build completed successfully",
    "level": "info"
  },
  "ttl": 3600
}
```

### queue_receive
Retrieve messages from a channel.

```json
{
  "channel": "notifications",
  "limit": 10,
  "blocking": false,
  "timeout": 30
}
```

### queue_subscribe
Subscribe to real-time updates from a channel.

```json
{
  "channel": "notifications"
}
```

Returns a subscription ID that can be used to unsubscribe later.

### queue_unsubscribe
Unsubscribe from a channel.

```json
{
  "subscription_id": "uuid-here"
}
```

### queue_list_channels
List all active channels.

```json
{}
```

### queue_stats
Get queue statistics.

```json
{}
```

## Message Structure

Messages follow this structure:

```json
{
  "id": "unique-message-id",
  "channel": "channel-name",
  "type": "message-type",
  "payload": {
    // Custom payload data
  },
  "timestamp": "2025-01-31T12:00:00Z",
  "ttl": 3600
}
```

## WebSocket Integration

The message queue integrates with Brummer's proxy server WebSocket infrastructure. Messages can be sent from browser applications through WebSocket connections:

```javascript
// Browser-side example
const ws = new WebSocket('ws://localhost:8080/__brummer_ws__');

ws.send(JSON.stringify({
  type: 'queue_message',
  data: {
    channel: 'browser-events',
    type: 'user-action',
    payload: {
      action: 'button-clicked',
      target: 'submit-button'
    },
    ttl: 300
  }
}));
```

## Use Cases

1. **Browser-to-AI Communication**: Send events from web applications to AI assistants
2. **Real-time Notifications**: Push updates from backend processes to frontend
3. **Event Streaming**: Subscribe to process events and react in real-time
4. **Coordination**: Coordinate actions between multiple AI agents
5. **Debugging**: Monitor application events during development

## Example Workflow

1. AI assistant subscribes to a channel:
   ```
   Tool: queue_subscribe
   Args: {"channel": "user-events"}
   ```

2. Browser sends a message via WebSocket:
   ```javascript
   ws.send(JSON.stringify({
     type: 'queue_message',
     data: {
       channel: 'user-events',
       type: 'form-submitted',
       payload: { formId: 'contact' }
     }
   }));
   ```

3. AI assistant receives the message and takes action:
   ```
   Tool: queue_receive
   Args: {"channel": "user-events", "limit": 1}
   ```

4. AI sends response back:
   ```
   Tool: queue_send
   Args: {
     "channel": "ai-responses",
     "type": "form-processed",
     "payload": {"status": "success", "formId": "contact"}
   }
   ```

## Performance Considerations

- Messages are stored in memory only (no persistence)
- TTL cleanup runs every 10 seconds
- Subscription channels are buffered (100 messages)
- Lock-free architecture provides high performance under contention
- Uses atomic operations and sync.Map for thread-safe access
- No message ordering guarantees across subscribers

## Performance Benchmarks

The lock-free implementation shows significant performance improvements over mutex-based approaches:

- **Receive operations**: 53x faster (183μs → 3.4μs)
- **Concurrent sending**: 2.1x faster under load
- **High contention scenarios**: 3.4x faster with 99% fewer allocations
- **Subscribe operations**: 1.7x faster