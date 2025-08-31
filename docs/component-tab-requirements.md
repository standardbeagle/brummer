# Brummer TUI Component Tab Requirements

## Overview
Brummer TUI has 7 main tabs, each with specific functionality and strict requirements for responsive display using lipgloss.

## Critical Lipgloss Requirements

### 1. **ALWAYS Force Color Output**
```go
// Required in EVERY component constructor
lipgloss.SetColorProfile(termenv.ANSI256)
```

### 2. **Border-Box Sizing**
Lipgloss adds borders OUTSIDE the specified width/height:
- A 100x20 box with border becomes 102x22 total
- Use `BorderWindow` component for automatic adjustment
- Account for 2 extra width, 2 extra height when using borders

### 3. **Responsive Sizing**
Every component MUST handle `tea.WindowSizeMsg`:
```go
func (c *Component) Update(msg tea.Msg) (Component, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        c.width = msg.Width
        c.height = msg.Height
        c.redistributeSpace() // Recalculate internal layout
    }
    return c, nil
}
```

### 4. **Width Calculation**
- Account for emoji width (all tab emojis are 2 characters wide)
- Account for ANSI escape sequences (use `lipgloss.Width()`)
- Never use `len()` for display width calculation

## The 7 Tabs

### Tab 1: Processes (🏃)
**Key**: `1`  
**Icon**: 🏃 (2 display width)  
**Purpose**: Process management and monitoring

**Requirements**:
- Display running processes with status indicators
- Show process uptime and resource usage
- Support start/stop/restart operations
- Color-coded status (🟢 running, 🔴 stopped, ❌ failed)
- List layout with scrolling support
- Keyboard shortcuts: `s` stop, `r` restart, `n` new process

**Component**: `ProcessViewNew`
- Must handle process data from ProcessManager
- Subscribe to process status updates via EventBus
- Responsive list rendering with proper truncation

### Tab 2: Logs (📄)
**Key**: `2`  
**Icon**: 📄 (2 display width)  
**Purpose**: Real-time log streaming and viewing

**Requirements**:
- Stream logs from selected process
- Auto-scroll to bottom (toggle with `end` key)
- Color-coded log levels (ERROR=red, WARN=yellow, INFO=default)
- Filter by process ID
- Search within logs
- Clear logs with `x` key
- Handle high-frequency log updates efficiently

**Component**: `LogsViewComponent`
- Must buffer logs for performance
- Implement virtual scrolling for large logs
- Route stdout to normal logs, stderr to error logs

### Tab 3: Errors (❌)
**Key**: `3`  
**Icon**: ❌ (2 display width)  
**Purpose**: Error tracking and debugging

**Requirements**:
- Capture ALL stderr output from processes
- Display errors with timestamps
- Group errors by process
- Show stack traces when available
- Copy error to clipboard with `c` key
- Clear errors with `z` key
- Count badge for unread errors

**Component**: `ErrorsViewComponent`
- MUST subscribe to stderr streams from ProcessManager
- Parse error formats (JavaScript, Python, Go, etc.)
- Maintain error history per session

### Tab 4: URLs (🔗)
**Key**: `4`  
**Icon**: 🔗 (2 display width)  
**Purpose**: Detected URL management

**Requirements**:
- Auto-detect URLs from process output
- Categorize URLs (local dev, external, API endpoints)
- Click/Enter to open in browser
- Copy URL to clipboard
- Show which process generated each URL
- Deduplicate URLs per process

**Component**: `URLsViewComponent`
- Parse logs for URL patterns
- Maintain URL registry per process
- Handle localhost port detection

### Tab 5: Web/Proxy (🌐)
**Key**: `5`  
**Icon**: 🌐 (2 display width)  
**Purpose**: HTTP proxy and request monitoring

**Requirements**:
- Display intercepted HTTP requests
- Show request/response details
- Status codes with color coding
- Request timing information
- Filter by method, status, URL pattern
- Indicators: ❌ error, 🔐 auth, 📊 telemetry

**Component**: `WebViewComponent`
- Connect to proxy server for request data
- Real-time request updates
- Request detail view on selection

### Tab 6: AI Coders (🤖)
**Key**: `6`  
**Icon**: 🤖 (2 display width)  
**Purpose**: AI coding assistant management

**Requirements**:
- List active AI coder sessions
- Show session status and activity
- Terminal/PTY integration for each session
- Session creation/termination controls
- Resource usage per session
- Command history

**Component**: `AICodersViewComponent`
- Manage multiple PTY sessions
- Route I/O to/from AI coders
- Session state persistence

### Tab 7: Settings (⚙️)
**Key**: `7`  
**Icon**: ⚙️ (2 display width)  
**Purpose**: Configuration and server info

**Requirements**:
- Display MCP server status and endpoints
- Show proxy configuration
- Environment variables management
- Theme/color preferences
- Keyboard shortcut customization
- Server URLs (MCP endpoint, proxy PAC)

**Component**: `SettingsViewAdapter`
- Read/write configuration files
- Apply settings without restart
- Validate configuration changes

## Tab Navigation System

### Header Requirements
- Show all 7 tabs with icons and numbers
- Highlight active tab
- Show notification badges (🔴🟡🔵🟢 with counts)
- Responsive width modes:
  - **Full**: Icon + Full Name + Badge
  - **Normal**: Icon + Short Name + Badge
  - **Compact**: Icon + Number + Badge
  - **Minimal**: Icon only with color for notifications

### Tab Switching
- Number keys (1-7) for direct access
- Tab/Shift+Tab for cycling
- Update notification badges on switch
- Clear notifications when tab becomes active

## Event-Driven Updates

### EventBus Integration
All components MUST subscribe to relevant events:

```go
// Process events
events.ProcessStarted
events.ProcessStopped
events.ProcessOutput

// Log events  
events.LogEntry
events.ErrorOccurred

// System events
events.URLDetected
events.ProxyRequest
events.ConfigChanged
```

### Message Passing
Never directly modify state. Use tea.Cmd:

```go
// Wrong
c.logs = append(c.logs, newLog)

// Correct
return c, func() tea.Msg {
    return LogAddedMsg{Log: newLog}
}
```

## Performance Requirements

### Rendering
- < 16ms render time for 60fps
- Use virtual scrolling for lists > 100 items
- Cache static content between renders
- Batch updates within same frame

### Memory
- Buffer logs with max size (10,000 lines default)
- Implement log rotation for long-running processes
- Clean up terminated process data

### Concurrency
- Use atomic operations for shared state
- No mutexes in render path
- Worker pools for event processing

## Testing Requirements

Each tab component must have:
1. Unit tests for core functionality
2. Integration tests with EventBus
3. Render performance benchmarks
4. Resize handling tests
5. Keyboard navigation tests

## Common Pitfalls to Avoid

1. **Not forcing color profile** - Colors won't work
2. **Using len() for display width** - Incorrect with emojis/ANSI
3. **Not handling WindowSizeMsg** - Component won't resize
4. **Direct state mutation** - Causes race conditions
5. **Forgetting border width** - Layout calculations wrong
6. **Blocking in Update()** - Freezes entire TUI
7. **Not buffering high-frequency updates** - Performance issues

## Implementation Checklist

For each tab component:
- [ ] Force ANSI256 color profile in constructor
- [ ] Implement responsive View() with proper sizing
- [ ] Handle WindowSizeMsg in Update()
- [ ] Subscribe to relevant EventBus events  
- [ ] Use message passing for all state changes
- [ ] Add notification support with badges
- [ ] Implement keyboard shortcuts
- [ ] Add proper error handling
- [ ] Write comprehensive tests
- [ ] Document component-specific requirements