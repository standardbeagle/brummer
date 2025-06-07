---
sidebar_position: 4
---

# MCP Events

Brummer emits various events through the MCP server that clients can subscribe to for real-time updates.

## Event System Overview

The MCP event system provides:

- **Real-time notifications** of process and system changes
- **Structured event data** with consistent schemas
- **Selective subscription** to specific event types
- **Event history** and replay capabilities

## Subscribing to Events

### Basic Subscription

```javascript
// Subscribe to specific events
const subscription = await mcp.call('brummer.subscribe', {
  events: ['process.start', 'process.stop', 'log.error']
});

// Handle incoming events
mcp.on('notification', ({ method, params }) => {
  if (method === 'brummer.event') {
    console.log('Event:', params.type, params.data);
  }
});
```

### Advanced Subscription

```javascript
// Subscribe with filters
const subscription = await mcp.call('brummer.subscribe', {
  events: ['log.error'],
  filters: {
    processName: 'dev',
    severity: ['error', 'critical']
  },
  includeHistory: true,  // Get recent events on subscribe
  historyLimit: 10
});
```

## Event Types

### Process Events

#### process.start

Emitted when a process starts successfully.

```typescript
{
  type: 'process.start',
  timestamp: '2024-01-15T10:30:00Z',
  data: {
    processId: 'dev-server-123',
    name: 'dev',
    pid: 12345,
    command: 'npm run dev',
    env: {
      NODE_ENV: 'development',
      PORT: '3000'
    }
  }
}
```

#### process.stop

Emitted when a process stops.

```typescript
{
  type: 'process.stop',
  timestamp: '2024-01-15T10:35:00Z',
  data: {
    processId: 'dev-server-123',
    name: 'dev',
    exitCode: 0,
    signal: null,
    reason: 'manual',  // manual, crash, error
    runtime: 300000  // milliseconds
  }
}
```

#### process.error

Emitted when a process encounters an error.

```typescript
{
  type: 'process.error',
  timestamp: '2024-01-15T10:32:00Z',
  data: {
    processId: 'dev-server-123',
    name: 'dev',
    error: {
      message: 'Cannot find module',
      code: 'MODULE_NOT_FOUND',
      stack: '...'
    },
    fatal: true
  }
}
```

#### process.restart

Emitted when a process is restarted.

```typescript
{
  type: 'process.restart',
  timestamp: '2024-01-15T10:33:00Z',
  data: {
    processId: 'dev-server-123',
    name: 'dev',
    reason: 'file_change',  // manual, crash, file_change
    restartCount: 3
  }
}
```

### Log Events

#### log.error

Emitted when an error is detected in logs.

```typescript
{
  type: 'log.error',
  timestamp: '2024-01-15T10:31:00Z',
  data: {
    processName: 'dev',
    level: 'error',
    message: 'TypeError: Cannot read property of undefined',
    file: '/src/App.js',
    line: 42,
    column: 15,
    stack: '...',
    context: {
      previousLines: ['...'],
      nextLines: ['...']
    }
  }
}
```

#### log.warning

Emitted for warning-level log entries.

```typescript
{
  type: 'log.warning',
  timestamp: '2024-01-15T10:30:30Z',
  data: {
    processName: 'build',
    message: 'Deprecation warning: componentWillMount',
    file: '/src/components/Legacy.js',
    line: 10,
    severity: 'medium'
  }
}
```

#### log.pattern

Emitted when a configured pattern is matched.

```typescript
{
  type: 'log.pattern',
  timestamp: '2024-01-15T10:31:30Z',
  data: {
    processName: 'api',
    pattern: 'database_connection',
    matches: {
      status: 'connected',
      database: 'myapp_dev',
      latency: '23ms'
    }
  }
}
```

### Build Events

#### build.start

Emitted when a build process begins.

```typescript
{
  type: 'build.start',
  timestamp: '2024-01-15T10:40:00Z',
  data: {
    processName: 'build',
    builder: 'webpack',
    mode: 'production',
    entry: './src/index.js'
  }
}
```

#### build.complete

Emitted when a build completes.

```typescript
{
  type: 'build.complete',
  timestamp: '2024-01-15T10:40:45Z',
  data: {
    processName: 'build',
    success: true,
    duration: 45000,
    stats: {
      assets: 15,
      chunks: 8,
      modules: 234,
      errors: 0,
      warnings: 2,
      size: {
        total: 1234567,
        javascript: 987654,
        css: 123456,
        images: 123457
      }
    }
  }
}
```

#### build.error

Emitted when a build fails.

```typescript
{
  type: 'build.error',
  timestamp: '2024-01-15T10:41:00Z',
  data: {
    processName: 'build',
    error: {
      file: '/src/components/Button.js',
      line: 23,
      column: 5,
      message: 'Unexpected token',
      code: 'BABEL_PARSE_ERROR'
    },
    partial: false  // true if build partially succeeded
  }
}
```

### Test Events

#### test.start

Emitted when tests begin running.

```typescript
{
  type: 'test.start',
  timestamp: '2024-01-15T10:45:00Z',
  data: {
    processName: 'test',
    runner: 'jest',
    totalSuites: 12,
    totalTests: 145,
    pattern: '**/*.test.js'
  }
}
```

#### test.complete

Emitted when all tests finish.

```typescript
{
  type: 'test.complete',
  timestamp: '2024-01-15T10:46:30Z',
  data: {
    processName: 'test',
    passed: 143,
    failed: 2,
    skipped: 0,
    duration: 90000,
    coverage: {
      lines: 87.5,
      functions: 92.3,
      branches: 78.9,
      statements: 88.1
    }
  }
}
```

#### test.failure

Emitted for each test failure.

```typescript
{
  type: 'test.failure',
  timestamp: '2024-01-15T10:45:45Z',
  data: {
    processName: 'test',
    suite: 'UserAPI',
    test: 'should create new user',
    error: {
      message: 'Expected 201 but received 400',
      expected: 201,
      actual: 400,
      stack: '...'
    },
    file: '/tests/api/user.test.js',
    line: 45
  }
}
```

### URL Events

#### url.detected

Emitted when a new URL is detected.

```typescript
{
  type: 'url.detected',
  timestamp: '2024-01-15T10:30:15Z',
  data: {
    processName: 'dev',
    url: 'http://localhost:3000',
    type: 'http',  // http, https, ws, wss
    status: 'pending'  // pending, online, offline
  }
}
```

#### url.status

Emitted when URL status changes.

```typescript
{
  type: 'url.status',
  timestamp: '2024-01-15T10:30:20Z',
  data: {
    url: 'http://localhost:3000',
    previousStatus: 'pending',
    currentStatus: 'online',
    responseTime: 234,
    statusCode: 200
  }
}
```

### System Events

#### memory.warning

Emitted when memory usage is high.

```typescript
{
  type: 'memory.warning',
  timestamp: '2024-01-15T10:50:00Z',
  data: {
    processName: 'dev',
    usage: {
      current: 512000000,  // bytes
      peak: 612000000,
      limit: 1073741824,
      percentage: 47.6
    },
    trend: 'increasing',
    rate: 1048576  // bytes per second
  }
}
```

#### cpu.warning

Emitted when CPU usage is high.

```typescript
{
  type: 'cpu.warning',
  timestamp: '2024-01-15T10:51:00Z',
  data: {
    processName: 'build',
    usage: {
      current: 85.5,  // percentage
      average: 72.3,
      cores: 2
    },
    duration: 30000  // high usage duration in ms
  }
}
```

### File Events

#### file.change

Emitted when watched files change.

```typescript
{
  type: 'file.change',
  timestamp: '2024-01-15T10:35:30Z',
  data: {
    path: '/src/App.js',
    type: 'modify',  // create, modify, delete, rename
    size: 4567,
    affectedProcesses: ['dev', 'test']
  }
}
```

#### config.change

Emitted when configuration files change.

```typescript
{
  type: 'config.change',
  timestamp: '2024-01-15T10:36:00Z',
  data: {
    file: 'package.json',
    changes: {
      dependencies: {
        added: ['axios@1.6.0'],
        removed: [],
        updated: ['react@18.2.0->18.3.0']
      }
    },
    requiresRestart: ['dev', 'build']
  }
}
```

## Event Filtering

### Process-based Filtering

```javascript
// Only receive events from specific processes
await mcp.call('brummer.subscribe', {
  events: ['log.error', 'process.error'],
  filters: {
    processNames: ['dev', 'api']
  }
});
```

### Severity Filtering

```javascript
// Only high-severity events
await mcp.call('brummer.subscribe', {
  events: ['log.error', 'memory.warning'],
  filters: {
    severity: ['critical', 'high']
  }
});
```

### Pattern-based Filtering

```javascript
// Filter by content patterns
await mcp.call('brummer.subscribe', {
  events: ['log.error'],
  filters: {
    patterns: ['database', 'connection', 'timeout']
  }
});
```

## Event Aggregation

### Batch Events

Some events are automatically batched:

```typescript
{
  type: 'log.errors.batch',
  timestamp: '2024-01-15T10:40:00Z',
  data: {
    processName: 'dev',
    count: 5,
    timespan: 1000,  // milliseconds
    errors: [
      { message: 'Error 1', count: 3 },
      { message: 'Error 2', count: 2 }
    ]
  }
}
```

### Summary Events

Periodic summary events:

```typescript
{
  type: 'system.summary',
  timestamp: '2024-01-15T11:00:00Z',
  data: {
    period: 3600000,  // 1 hour
    processes: {
      total: 5,
      running: 3,
      stopped: 2
    },
    errors: {
      total: 45,
      critical: 2,
      resolved: 40
    },
    performance: {
      avgMemory: 234567890,
      avgCpu: 23.4,
      builds: 12,
      avgBuildTime: 45000
    }
  }
}
```

## Event History

### Querying Historical Events

```javascript
// Get events from the last hour
const history = await mcp.call('brummer.getEventHistory', {
  since: '1 hour ago',
  types: ['process.error', 'build.error'],
  limit: 100
});
```

### Event Replay

```javascript
// Replay events from a specific time
await mcp.call('brummer.replayEvents', {
  from: '2024-01-15T10:00:00Z',
  to: '2024-01-15T11:00:00Z',
  speed: 2.0,  // 2x speed
  types: ['process.start', 'process.stop']
});
```

## Custom Events

### Emitting Custom Events

```javascript
// Emit a custom event
await mcp.call('brummer.emitEvent', {
  type: 'custom.deployment',
  data: {
    version: '1.2.3',
    environment: 'staging',
    status: 'success'
  }
});
```

### Subscribing to Custom Events

```javascript
await mcp.call('brummer.subscribe', {
  events: ['custom.*'],
  includeMetadata: true
});
```

## Event Configuration

### Configure Event Behavior

```yaml
# .brummer.yaml
mcp:
  events:
    buffer_size: 1000
    history_retention: 24h
    batch_interval: 500ms
    
    filters:
      - type: log.error
        min_severity: warning
        
      - type: memory.warning
        threshold: 80
        
    aggregation:
      - types: [log.error]
        window: 1m
        min_count: 5
```

## Best Practices

1. **Subscribe selectively** - Only subscribe to events you need
2. **Handle reconnection** - Events may be missed during disconnection
3. **Process events asynchronously** - Don't block the event handler
4. **Use filters** - Reduce network traffic and processing
5. **Monitor event volume** - High event rates may indicate issues
6. **Clean up subscriptions** - Unsubscribe when no longer needed
7. **Log event errors** - Track issues with event processing

## Example: Complete Event Handler

```javascript
class BrummerEventHandler {
  constructor(mcpClient) {
    this.mcp = mcpClient;
    this.handlers = new Map();
  }

  async initialize() {
    // Subscribe to all relevant events
    await this.mcp.call('brummer.subscribe', {
      events: [
        'process.*',
        'log.error',
        'build.complete',
        'test.failure',
        'url.detected'
      ]
    });

    // Set up notification handler
    this.mcp.on('notification', ({ method, params }) => {
      if (method === 'brummer.event') {
        this.handleEvent(params);
      }
    });
  }

  on(eventType, handler) {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, []);
    }
    this.handlers.get(eventType).push(handler);
  }

  handleEvent(event) {
    const { type, timestamp, data } = event;
    
    // Call specific handlers
    const handlers = this.handlers.get(type) || [];
    handlers.forEach(handler => {
      try {
        handler({ type, timestamp, data });
      } catch (error) {
        console.error(`Error in event handler for ${type}:`, error);
      }
    });

    // Call wildcard handlers
    const wildcardHandlers = this.handlers.get('*') || [];
    wildcardHandlers.forEach(handler => {
      try {
        handler({ type, timestamp, data });
      } catch (error) {
        console.error('Error in wildcard handler:', error);
      }
    });
  }
}

// Usage
const eventHandler = new BrummerEventHandler(mcpClient);
await eventHandler.initialize();

// Handle specific events
eventHandler.on('process.error', ({ data }) => {
  console.error(`Process ${data.name} error:`, data.error.message);
  // Send alert, restart process, etc.
});

eventHandler.on('build.complete', ({ data }) => {
  if (data.success) {
    console.log('Build successful!');
    // Deploy, notify team, etc.
  }
});

// Handle all events
eventHandler.on('*', (event) => {
  console.log(`[${event.timestamp}] ${event.type}`, event.data);
});
```