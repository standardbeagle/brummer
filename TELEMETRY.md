# Brummer Web Telemetry

Brummer's proxy server now includes advanced web monitoring capabilities that automatically inject telemetry collection into proxied web pages.

## Features

### 1. **Automatic JavaScript Injection**
- Monitoring script is automatically injected into all HTML responses
- No client-side configuration required
- Works with gzip-compressed responses

### 2. **Comprehensive Telemetry Collection**

#### Performance Metrics
- Page load timing (DNS, connect, request, response, DOM processing)
- Paint timing (first paint, first contentful paint)
- Long task detection
- Resource timing for all network requests

#### Memory Monitoring
- JavaScript heap size usage
- Memory allocation trends
- Periodic snapshots for memory leak detection

#### Console Output Tracking
- All console methods (log, info, warn, error, debug)
- Stack traces for debugging
- Aggregated counts by log level

#### Error Detection
- JavaScript errors with stack traces
- Unhandled promise rejections
- Network errors

#### User Interaction Tracking
- Click events with element selectors
- Form submissions
- Input field focus duration
- Page visibility changes

### 3. **Session Management**
- Unique session IDs for each page load
- Process association for multi-app environments
- Automatic cleanup of old sessions

### 4. **Real-time Data Collection**
- Batched telemetry sending (every 2 seconds)
- Uses sendBeacon API for reliability
- Fallback to fetch API when needed

## Usage

### 1. Enable Proxy with Telemetry

```bash
# Start Brummer with proxy enabled (telemetry is on by default)
brum --proxy

# Or run a script with proxy
brum proxy-test
```

### 2. Configure Browser

Set your browser's proxy to `localhost:8888` or use the PAC file at `http://localhost:8888/proxy.pac`

### 3. Access Telemetry Data

Telemetry data is available through:
- The Brummer TUI (Telemetry view - coming soon)
- MCP API endpoints
- Direct access via proxy server methods

## API Access

### Get All Sessions
```javascript
// Via MCP client
const sessions = await mcpClient.getTelemetrySessions();
```

### Get Sessions for Process
```javascript
// Get telemetry for specific process
const sessions = await mcpClient.getTelemetryForProcess('proxy-test');
```

### Session Data Structure

```javascript
{
  sessionId: "brummer_1234567890_abc123",
  url: "http://localhost:3457/",
  processName: "proxy-test",
  startTime: "2024-01-20T10:30:00Z",
  lastActivity: "2024-01-20T10:35:00Z",
  events: [...],
  performanceMetrics: {
    navigationStart: 1234567890,
    loadCompleteTime: 1500,
    domContentLoadedTime: 800,
    // ... more metrics
  },
  memorySnapshots: [...],
  errorCount: 2,
  interactionCount: 15,
  consoleLogCount: {
    log: 10,
    warn: 2,
    error: 3
  }
}
```

## Testing

Use the included telemetry test server:

```bash
# In test-project directory
npm run telemetry-test
```

Then visit `http://localhost:3457/` through the proxy to see all telemetry features in action.

## Configuration

### Disable Telemetry
```go
// In code
server.EnableTelemetry(false)
```

### Custom Telemetry Settings
Edit `internal/proxy/monitor.js` to customize:
- Collection intervals
- Event types to monitor
- Batch sizes
- Endpoint URLs

## Security Considerations

- Telemetry data is only collected from proxied requests
- No data is sent to external servers
- Content Security Policy headers are removed to allow script injection
- All data stays within your local Brummer instance

## Performance Impact

The monitoring script is designed to be lightweight:
- Minimal CPU overhead
- Batched data transmission
- Automatic cleanup of old data
- Configurable collection intervals