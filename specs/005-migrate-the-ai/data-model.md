# Data Model: AI Coder Service and Component Architecture

**Feature**: 005-migrate-the-ai
**Date**: 2025-10-07
**Status**: Complete

## Overview

This document defines the data entities, state structures, and relationships for the AI coder service and component architecture. The model supports session management, PTY terminal state, and UI view state.

## Core Entities

### 1. AICoderSession

Represents an active AI coding session with complete lifecycle tracking.

**Attributes**:
```go
type AICoderSession struct {
    // Identity
    ID           string    // Unique session identifier (UUID)
    Name         string    // User-friendly session name

    // Lifecycle
    Status       SessionStatus    // Current session state
    CreatedAt    time.Time       // Session creation timestamp
    LastActivity time.Time       // Last user interaction or output

    // PTY Integration
    PTYManager   *LockFreePTYManager  // PTY process manager
    Command      string               // Command being executed
    WorkingDir   string               // Session working directory

    // Metrics
    BytesRead    atomic.Uint64   // Total bytes read from PTY
    BytesWritten atomic.Uint64   // Total bytes written to PTY
    ErrorCount   atomic.Uint32   // Number of errors encountered
}

type SessionStatus string

const (
    SessionPending  SessionStatus = "pending"   // Created, not started
    SessionActive   SessionStatus = "active"    // Running with PTY
    SessionIdle     SessionStatus = "idle"      // Running but no recent activity
    SessionError    SessionStatus = "error"     // PTY failure or crash
    SessionStopped  SessionStatus = "stopped"   // Gracefully stopped
)
```

**Validation Rules**:
- ID must be non-empty UUID format
- Name must be 1-100 characters
- Status transitions: Pending → Active → (Idle ↔ Active)* → (Stopped | Error)
- CreatedAt must be ≤ current time
- LastActivity must be ≥ CreatedAt
- Command must be non-empty
- PTYManager must be non-nil when Status is Active or Idle

**State Transitions**:
```
Pending → Active     (PTY started successfully)
Active → Idle        (No activity for 5 seconds)
Idle → Active        (New input or output)
Active → Error       (PTY crash or communication failure)
Idle → Error         (PTY crash or communication failure)
Active → Stopped     (User terminates session)
Idle → Stopped       (User terminates session)
Error → Stopped      (Cleanup after error notification)
```

**Relationships**:
- Each session has exactly one PTYManager (1:1, composition)
- Each session has exactly one TerminalState (1:1, derived)
- Each session may have multiple PTYOutputEvents (1:N, temporal)

---

### 2. TerminalState

Read-only view of terminal state for rendering. Derived from PTYManager's lock-free buffer.

**Attributes**:
```go
type TerminalState struct {
    // Display Content
    Lines    []string      // Terminal screen lines (height lines)
    Cursor   CursorPos     // Current cursor position

    // Dimensions
    Width    int           // Terminal width (columns)
    Height   int           // Terminal height (rows)

    // Viewport
    TotalLines   int       // Total lines in scrollback buffer
    ViewportTop  int       // First visible line (for scrolling)

    // Metadata
    UpdatedAt    time.Time // Last update timestamp
}

type CursorPos struct {
    Row    int   // Cursor row (0-based)
    Col    int   // Cursor column (0-based)
    Visible bool // Cursor visibility flag
}
```

**Validation Rules**:
- Lines length must equal Height
- Each line length must be ≤ Width (padded/truncated as needed)
- Cursor.Row must be in range [0, Height)
- Cursor.Col must be in range [0, Width)
- Width must be ≥ 20 (minimum usable terminal)
- Height must be ≥ 5 (minimum usable terminal)
- TotalLines must be ≥ Height
- ViewportTop must be in range [0, TotalLines - Height]

**Derivation**:
- Computed on-demand from PTY manager's ring buffer
- No persistent storage (ephemeral)
- Lock-free reads from shared PTY state

---

### 3. SessionViewState

UI-specific state for displaying sessions. Managed by component, not service.

**Attributes**:
```go
type SessionViewState struct {
    // View Mode
    Mode             ViewMode     // List view or PTY view
    FocusedSessionID string       // Currently focused session (empty if none)

    // List View State
    SelectedIndex    int          // Selected session index in list
    ListScrollOffset int          // Scroll position in session list

    // PTY View State
    PTYScrollOffset  int          // Scroll position in terminal view
    ShowHelp         bool         // Help overlay visibility

    // Viewport Dimensions
    ViewportWidth    int          // Available width for rendering
    ViewportHeight   int          // Available height for rendering
}

type ViewMode string

const (
    ViewModeList ViewMode = "list"  // Show session list
    ViewModePTY  ViewMode = "pty"   // Show active session PTY
)
```

**Validation Rules**:
- Mode must be either ViewModeList or ViewModePTY
- FocusedSessionID must be empty in ViewModeList
- FocusedSessionID must reference existing session in ViewModePTY
- SelectedIndex must be in range [0, session count)
- Scroll offsets must be ≥ 0
- Viewport dimensions must be > 0

**Lifecycle**:
- Created and managed entirely by component
- Not persisted (ephemeral UI state)
- Reset when switching between sessions

---

### 4. PTYOutputEvent

Event representing PTY output data. Flows from service to component via messages.

**Attributes**:
```go
type PTYOutputEvent struct {
    SessionID  string       // Session that produced output
    Data       []byte       // Raw PTY output bytes
    Timestamp  time.Time    // When output was received
    Sequence   uint64       // Monotonic sequence number
}
```

**Validation Rules**:
- SessionID must reference existing session
- Data length must be > 0 and ≤ 64KB (chunk size)
- Timestamp must be ≥ session creation time
- Sequence must be monotonically increasing per session

**Lifecycle**:
- Created when PTY emits output
- Sent immediately to component via BubbleTea message
- Ephemeral (not stored after processing)

---

## Service State

### AICoderService

Internal state managed by service layer.

```go
type AICoderService struct {
    // Session Registry (sync.Map for lock-free reads)
    sessions sync.Map  // map[string]*AICoderSession

    // Lifecycle
    eventBus   *events.EventBus
    wg         sync.WaitGroup  // For graceful shutdown
    shutdownCh chan struct{}   // Signal shutdown to goroutines

    // Configuration
    maxConcurrentSessions int  // 0 = unlimited
    ptyCleanupTimeout     time.Duration
}
```

**Concurrency Model**:
- `sync.Map` for lock-free session lookup
- Per-session goroutine for PTY monitoring
- EventBus for service → component communication
- Bounded goroutine pool (one per session, not unbounded)

---

## Component State

### AICodersViewComponent

Component state structure (extends BaseComponent).

```go
type AICodersViewComponent struct {
    BaseComponent  // From components/base.go

    // Service Reference
    service *services.AICoderService

    // View State
    viewState SessionViewState

    // Cached Data (updated via messages)
    sessions []SessionSummary  // Cached session list
    terminal *TerminalState    // Cached terminal state (if viewing PTY)

    // Viewport (BubbleTea viewport component)
    viewport viewport.Model

    // Styles
    styles AICoderStyles
}

type SessionSummary struct {
    ID           string
    Name         string
    Status       SessionStatus
    LastActivity time.Time
}

type AICoderStyles struct {
    HeaderStyle   lipgloss.Style
    ActiveStyle   lipgloss.Style
    IdleStyle     lipgloss.Style
    ErrorStyle    lipgloss.Style
    SelectedStyle lipgloss.Style
    NormalStyle   lipgloss.Style
}
```

---

## Message Types

### Service → Component Messages

```go
// Session lifecycle events
type AICoderSessionCreatedMsg struct {
    SessionID string
    Name      string
}

type AICoderSessionUpdatedMsg struct {
    SessionID    string
    Status       SessionStatus
    LastActivity time.Time
}

type AICoderSessionEndedMsg struct {
    SessionID string
    Reason    string  // "stopped", "crashed", "error"
}

// PTY output events
type AICoderPTYOutputMsg struct {
    SessionID string
    Data      []byte
    Sequence  uint64
}

// Error notifications
type AICoderErrorMsg struct {
    SessionID string
    Error     error
    Severity  ErrorSeverity  // "warning", "error", "critical"
}
```

### Component → Service Commands

```go
// Session control commands
type StartAICoderSessionCmd struct {
    Name       string
    Command    string
    WorkingDir string
}

type StopAICoderSessionCmd struct {
    SessionID string
}

type SendAICoderInputCmd struct {
    SessionID string
    Input     []byte
}

type ResizePTYCmd struct {
    SessionID string
    Width     int
    Height    int
}
```

---

## Data Flow Diagrams

### Session Creation Flow
```
User Input → Component
             ↓
           StartAICoderSessionCmd (tea.Cmd)
             ↓
           Service.CreateSession()
             ↓
           PTYManager created
             ↓
           EventBus.Publish(SessionCreated)
             ↓
           AICoderSessionCreatedMsg → Component
             ↓
           Component updates session list
```

### PTY Output Flow
```
PTY Process → PTY Manager (lock-free buffer)
               ↓
             Service goroutine reads
               ↓
             PTYOutputEvent created
               ↓
             AICoderPTYOutputMsg → Component
               ↓
             Component requests TerminalState
               ↓
             Service returns read-only snapshot
               ↓
             Component renders terminal
```

### Session Cleanup Flow
```
User Input → StopAICoderSessionCmd
             ↓
           Service.StopSession()
             ↓
           PTYManager.Stop() (goroutine cleanup)
             ↓
           Verify no goroutine leaks
             ↓
           Remove from session registry
             ↓
           EventBus.Publish(SessionEnded)
             ↓
           AICoderSessionEndedMsg → Component
```

---

## Invariants

1. **Session Uniqueness**: No two sessions can have the same ID
2. **PTY Ownership**: Each PTYManager is owned by exactly one session
3. **Status Consistency**: Session status matches PTY state (active ↔ running)
4. **Cleanup Guarantee**: Stopped sessions have no running goroutines
5. **Message Ordering**: PTY output messages for a session are delivered in sequence order
6. **View Consistency**: Component terminal state is eventually consistent with PTY state
7. **Resource Bounds**: Service enforces session limits (if configured)
8. **Error Recovery**: Component handles missing sessions gracefully (session ended)

---

## Testing Considerations

### Service Tests
- Session creation with valid/invalid parameters
- Concurrent session creation and lookup
- PTY lifecycle (start, monitor, stop)
- Cleanup verification (goroutine count)
- Event emission verification

### Component Tests
- Message handling (session created, updated, ended)
- PTY output rendering
- Keyboard input forwarding
- View mode switching (list ↔ PTY)
- Error state display

### Integration Tests
- Full session lifecycle with real PTY
- Terminal resize handling
- Race condition detection (make test-race)
- Resource leak detection
- Performance verification (<50ms typing latency)

---

## References

- Feature Spec: `spec.md` (Key Entities section)
- Research: `research.md` (Decisions 1-4)
- Constitution: `.specify/memory/constitution.md` v1.0.0 (Lock-Free Architecture)
- BubbleTea patterns: Feature 002 components
