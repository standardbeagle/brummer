# Phases 5-7 Remaining Work

## Current Status (After ErrorsViewComponent)

**Completed**: 4 of 8 views (50% coverage)
- ✅ Processes (ProcessViewComponent)
- ✅ Logs (LogsViewComponent)
- ✅ URLs (URLsViewComponent)
- ✅ Errors (ErrorsViewComponent) - **JUST ADDED**

**Test Coverage**: 52 tests passing (9 new with ErrorsView)

**Remaining**: 4 of 8 views (50% remaining)
- ❌ Web (proxy request monitoring)
- ❌ AI Coders (AI assistant management)
- ❌ Settings (configuration editing)
- ❌ MCP (MCP connection status)

## Phase 5: Remaining View Components

### 1. WebViewComponent (NEXT)

**Purpose**: Display HTTP proxy requests captured by the proxy server

**Data Source**: `internal/proxy/server.go`

**Message Types Needed**:
```go
// Add to internal/tui/messages/service_events.go
type ProxyRequestMsg struct {
    RequestID   string
    ProcessID   string
    Method      string
    URL         string
    StatusCode  int
    Duration    time.Duration
    Timestamp   time.Time
}
```

**ServiceBridge Integration**:
```go
// Add to internal/tui/bridge/service_bridge.go
func (b *ServiceBridge) subscribeToProxyServer() {
    if b.proxyServer == nil {
        return
    }

    // Subscribe to proxy request events
    b.proxyServer.OnRequest(func(req *proxy.Request) {
        b.program.Send(messages.ProxyRequestMsg{
            RequestID:  req.ID,
            Method:     req.Method,
            URL:        req.URL,
            // ... map fields
        })
    })
}
```

**Component Features**:
- Table view: Method | URL | Status | Duration
- Color-code by status (2xx green, 4xx yellow, 5xx red)
- Filter by process
- Keyboard scrolling
- Selected request shows headers/body

**Estimated**: 8-10 tests, 250 lines of code

### 2. AICodersViewComponent

**Purpose**: Manage and monitor AI coding assistant sessions

**Data Source**: `internal/aicoder/manager.go`

**Message Types Needed**:
```go
type AICoderSessionMsg struct {
    SessionID   string
    Status      string // "idle", "working", "waiting"
    Command     string
    OutputLines []string
    Timestamp   time.Time
}

type AICoderOutputMsg struct {
    SessionID string
    Line      string
    Timestamp time.Time
}
```

**ServiceBridge Integration**:
```go
func (b *ServiceBridge) subscribeToAICoder() {
    if b.aiCoderMgr == nil {
        return
    }

    b.aiCoderMgr.OnStatusChange(func(session *aicoder.Session) {
        b.program.Send(messages.AICoderSessionMsg{
            SessionID: session.ID,
            Status:    session.Status,
            // ...
        })
    })
}
```

**Component Features**:
- List active AI coder sessions
- Show session status (idle/working/waiting)
- Display last command/output
- Navigate between sessions
- Keyboard to select/view details

**Estimated**: 8-10 tests, 300 lines of code

### 3. SettingsViewComponent

**Purpose**: Display and edit configuration settings

**Data Source**: `internal/config/config.go`

**Message Types Needed**:
```go
type SettingsChangedMsg struct {
    Key      string
    OldValue interface{}
    NewValue interface{}
}
```

**Component Features**:
- Display current settings (read-only for now)
- Group by category (Process, Proxy, MCP, UI)
- Show setting descriptions
- Keyboard navigation
- *Future*: Edit mode with form inputs

**Estimated**: 6-8 tests, 200 lines of code

### 4. MCPViewComponent

**Purpose**: Display MCP server status and connected clients

**Data Source**: MCP server interface

**Message Types Needed**:
```go
type MCPConnectionMsg struct {
    ClientID    string
    Connected   bool
    IPAddress   string
    ConnectedAt time.Time
}

type MCPRequestMsg struct {
    ClientID  string
    Method    string
    Timestamp time.Time
}
```

**ServiceBridge Integration**:
```go
func (b *ServiceBridge) subscribeToMCPServer() {
    if b.mcpServer == nil {
        return
    }

    b.mcpServer.OnConnection(func(client *mcp.Client) {
        b.program.Send(messages.MCPConnectionMsg{
            ClientID:    client.ID,
            Connected:   true,
            IPAddress:   client.IP,
            ConnectedAt: time.Now(),
        })
    })
}
```

**Component Features**:
- Show MCP server status (running/stopped, port)
- List connected clients
- Show recent requests
- Connection duration
- Keyboard navigation

**Estimated**: 6-8 tests, 200 lines of code

### Phase 5 Summary

**Total Estimated**:
- 4 new components
- 28-36 new tests
- ~950-1000 lines of code
- Total tests: 80-88 tests

## Phase 6: UI Chrome & Navigation

### TabsComponent

**Purpose**: Visual tab bar with icons and notification badges

**Original UI Reference**: `internal/tui/tabs_component.go` (if exists)

**Features**:
- Tab icons: 🏃 📄 ❌ 🔗 🌐 🤖 ⚙️ 🔌
- Active tab highlighting
- Notification badges (colored dots/counts)
- Keyboard shortcuts displayed
- Responsive width handling

**Implementation**:
```go
type TabsComponent struct {
    BaseComponent
    tabs          []TabInfo
    activeTab     string
    notifications map[string]int // tab -> count
}

type TabInfo struct {
    Key         string
    Icon        string
    Title       string
    KeyBinding  string
    Description string
}
```

**Estimated**: 6-8 tests, 200 lines of code

### HeaderComponent

**Purpose**: System status header

**Features**:
- Application title/version
- MCP server status indicator
- Proxy server status
- Active processes count
- Error count indicator
- Resource usage (optional)

**Layout**:
```
┌─────────────────────────────────────────────────────────────────────┐
│ Brummer v1.0    MCP:✓ 7777    Proxy:✓ 8888    Processes: 3    ❌ 2 │
└─────────────────────────────────────────────────────────────────────┘
```

**Estimated**: 4-6 tests, 150 lines of code

### FooterComponent

**Purpose**: Keyboard shortcut hints and help

**Features**:
- Context-sensitive shortcuts
- Current view indicator
- Help text
- Notification area

**Layout**:
```
┌─────────────────────────────────────────────────────────────────────┐
│ [1-8] Switch View │ [j/k] Navigate │ [Enter] Select │ [?] Help     │
└─────────────────────────────────────────────────────────────────────┘
```

**Estimated**: 4-6 tests, 150 lines of code

### NotificationSystem

**Purpose**: Toast notifications and status messages

**Features**:
- Color-coded notifications (success/warning/error/info)
- Auto-dismiss after timeout
- Notification queue
- Priority handling

**Implementation**:
```go
type NotificationManager struct {
    notifications []Notification
    maxVisible    int
}

type Notification struct {
    ID       string
    Type     string // "success", "error", "warning", "info"
    Message  string
    Duration time.Duration
    Created  time.Time
}
```

**Estimated**: 6-8 tests, 200 lines of code

### EnhancedIntegratedShell

**Updates to integrate UI chrome**:
- Add HeaderComponent, TabsComponent, FooterComponent
- Layout manager for chrome + content
- Height calculations (header + content + footer)
- Tab notifications triggered by component events

**Estimated**: 8-10 tests, 300 lines of code

### Phase 6 Summary

**Total Estimated**:
- 4 UI components + shell enhancements
- 28-38 new tests
- ~1000-1100 lines of code
- Total tests: 108-126 tests

## Phase 7: Interactive Features

### Command System

**Purpose**: Execute commands on selected items

**Features**:

1. **Process Commands**:
   - Start/Stop/Restart process
   - Clear process logs
   - Open process in new terminal

2. **Error Commands**:
   - Copy error message
   - Clear specific error
   - Clear all errors

3. **URL Commands**:
   - Open URL in browser
   - Copy URL to clipboard
   - Copy proxy URL

4. **Web Request Commands**:
   - View request details
   - Copy request/response
   - Clear requests

**Implementation**:
```go
type CommandHandler struct {
    processMgr  *process.Manager
    logStore    *logs.Store
    proxyServer *proxy.Server
}

func (h *CommandHandler) Execute(cmd Command) tea.Cmd {
    switch cmd.Type {
    case "process.stop":
        return h.stopProcess(cmd.TargetID)
    case "error.copy":
        return h.copyError(cmd.TargetID)
    // ...
    }
}
```

**Message Flow**:
```
User presses 'r' → Component sends RestartProcessCmd →
CommandHandler.Execute() → ProcessManager.Restart() →
EventBus.Publish(ProcessStarted) → ServiceBridge → ProcessStartedMsg →
ProcessViewComponent updates
```

**Estimated**: 15-20 tests, 400 lines of code

### Overlays & Modals

**Command Palette** (Ctrl+K):
- Fuzzy search all commands
- Recent commands
- Keyboard shortcuts
- Context-aware suggestions

**File Browser**:
- Navigate project files
- Select script to run
- Integration with process manager

**Script Selector**:
- List available package.json scripts
- Quick start scripts
- Script descriptions

**Confirmation Dialogs**:
- Stop all processes
- Clear all logs
- Destructive actions

**Implementation**:
```go
type OverlayManager struct {
    active     Overlay
    overlays   map[string]Overlay
    background tea.Model
}

type Overlay interface {
    View() string
    Update(tea.Msg) (Overlay, tea.Cmd)
    IsVisible() bool
    Close()
}
```

**Estimated**: 20-25 tests, 600 lines of code

### Keyboard Shortcuts

**Global Shortcuts**:
- Ctrl+K: Command palette
- Ctrl+C: Quit
- ?: Help overlay
- /: Search/filter

**View-Specific**:
- Processes: r (restart), s (stop), Enter (view logs)
- Logs: c (clear), f (filter), / (search)
- Errors: c (copy), x (clear), Enter (details)
- URLs: o (open), c (copy), Enter (details)

**Implementation**:
```go
type KeyBindingManager struct {
    global      map[string]KeyBinding
    viewSpecific map[string]map[string]KeyBinding
}

type KeyBinding struct {
    Key         string
    Description string
    Handler     func() tea.Cmd
}
```

**Estimated**: 10-15 tests, 300 lines of code

### Help System

**Features**:
- Keyboard shortcut reference
- View-specific help
- Component documentation
- Tutorial mode (optional)

**Implementation**:
```go
type HelpOverlay struct {
    sections []HelpSection
    viewport viewport.Model
}

type HelpSection struct {
    Title    string
    Bindings []KeyBinding
}
```

**Estimated**: 5-8 tests, 200 lines of code

### Phase 7 Summary

**Total Estimated**:
- Command system, overlays, shortcuts, help
- 50-68 new tests
- ~1500-1700 lines of code
- Total tests: 158-194 tests

## Complete Phases 5-7 Summary

**Total New Implementation**:
- 4 view components (Web, AICoders, Settings, MCP)
- 4 UI chrome components (Tabs, Header, Footer, Notifications)
- Interactive command system
- Overlays and modals
- Keyboard shortcuts
- Help system

**Total Estimated Work**:
- ~106-142 new tests
- ~3450-3800 lines of code
- Total tests: ~158-194 tests
- Current: 2132 lines → Final: ~5600-6000 lines

**Time Estimate** (conservative):
- Phase 5 (4 components): 8-12 hours
- Phase 6 (UI chrome): 6-10 hours
- Phase 7 (interactivity): 10-15 hours
- **Total**: 24-37 hours of focused development

## Incremental Approach

### Minimum Viable (Quick Win)

1. **Web + MCP views** (simplest, display-only)
   - ~400 lines, 12-16 tests
   - 2-3 hours

2. **Basic TabsComponent** (no notifications)
   - ~150 lines, 4-6 tests
   - 1-2 hours

3. **Simple command system** (start/stop only)
   - ~200 lines, 8-10 tests
   - 2-3 hours

**Quick Win Total**: ~6-8 hours, achieves 75% feature parity

### Full Implementation

Follow phases 5-7 as outlined above for 100% feature parity.

## Next Steps

1. **Immediate** (this session if possible):
   - Create WebViewComponent (simplest remaining)
   - Create MCPViewComponent (also simple)
   - Update IntegratedShell routing
   - Commit Phase 5 progress

2. **Next Session**:
   - AI Coders + Settings components
   - Basic TabsComponent
   - Update documentation

3. **Final Push**:
   - Full UI chrome
   - Interactive commands
   - Overlays
   - Polish and testing

## Testing Strategy

Each new component should follow the established pattern:

1. **Unit tests** (component behavior)
   - New component creation
   - Message handling
   - Keyboard navigation
   - Size/visibility
   - Formatting

2. **Integration tests** (message flow)
   - ServiceBridge → Component
   - Component → Command → Service
   - End-to-end user actions

3. **Regression tests** (existing features)
   - All previous tests still pass
   - No performance degradation
   - No memory leaks

## Documentation Updates

After each phase:

1. Update `SERVICE_BRIDGE_ARCHITECTURE.md`
2. Update `COMPONENT_VS_ORIGINAL_UI.md`
3. Add component-specific docs if needed
4. Update test count summaries
5. Commit with detailed messages