# Data Model: Complete Tab Functionality

**Date**: 2025-10-19
**Feature**: Complete all 7 TUI tabs with error grouping and automated testing

## Core Entities

### GroupedError

**Purpose**: Represents a multi-line error entry that has been intelligently grouped from log streams

**Fields**:
- `ID`: string - Unique identifier for the error (timestamp + hash)
- `ProcessName`: string - Name of the process that emitted the error
- `ProcessPID`: int - Process ID of the source process
- `FirstLine`: string - The initial error line (e.g., "Error: Cannot find module")
- `FullContent`: []string - All lines belonging to this error group
- `ErrorType`: ErrorType enum - Classification of error (StackTrace, CompilationError, TestFailure, Generic)
- `Timestamp`: time.Time - When the first line of the error was captured
- `Source`: string - File path if extracted from error content
- `LineNumber`: int - Line number if extracted from error content
- `Version`: uint64 - State version for tracking changes

**Validation Rules**:
- `ID` must be non-empty
- `ProcessName` must be non-empty
- `FullContent` must have at least one line
- `Timestamp` must be valid (not zero time)
- `FirstLine` must match the first element of `FullContent`

**State Transitions**:
```
Created → Displayed → (User Action) → Cleared
```

### ErrorType Enum

**Values**:
- `StackTrace`: JavaScript/TypeScript/Go stack traces with "at " or file:line patterns
- `CompilationError`: TypeScript/Babel compilation errors (TS####, SyntaxError)
- `TestFailure`: Jest/Vitest test failure with expected/received
- `Generic`: Other multi-line errors that don't match specific patterns

### ProcessesState

**Purpose**: State for Processes tab

**Fields**:
- `Processes`: []ProcessInfo - List of all tracked processes
- `SelectedIndex`: int - Currently selected process index
- `GroupByStatus`: bool - Whether to group by running/stopped status
- `FilterQuery`: string - Process name filter text
- `SortBy`: SortField enum - Current sort field
- `LastUpdated`: time.Time - Last state update timestamp
- `Version`: uint64 - State version for change tracking

### ErrorsState

**Purpose**: State for Errors tab with intelligent grouping

**Fields**:
- `GroupedErrors`: []GroupedError - List of grouped error entries
- `FilterProcess`: string - Process name filter (empty = all)
- `SelectedIndex`: int - Currently selected error index
- `MaxRetention`: int - Maximum errors to retain (default: 1000)
- `ParserState`: ErrorParserState - Current parser state for streaming
- `LastUpdated`: time.Time - Last state update timestamp
- `Version`: uint64 - State version for change tracking

**Validation Rules**:
- `SelectedIndex` must be -1 (none) or valid index in GroupedErrors
- `MaxRetention` must be > 0
- `GroupedErrors` length must not exceed MaxRetention

### ErrorParserState

**Purpose**: Maintains state for stream-based error grouping parser

**Fields**:
- `CurrentGroup`: []string - Lines accumulated for current error group
- `InErrorGroup`: bool - Whether currently inside an error group
- `LastErrorType`: ErrorType - Type of error being parsed
- `LinesSinceError`: int - Counter for detecting group end

**Behavior**:
- Reset when error group is finalized
- Tracks indentation and continuation patterns
- Handles nested errors (e.g., error within stack trace)

### LogsState

**Purpose**: State for Logs tab with filtering and search

**Fields**:
- `Entries`: []LogEntry - All log entries across processes
- `SeverityFilter`: []LogLevel - Active severity filters (Info, Warn, Error, Debug)
- `SourceFilter`: string - Process name filter (empty = all)
- `SearchQuery`: string - Text search query
- `AutoScroll`: bool - Whether auto-scroll is enabled
- `ScrollPosition`: int - Current scroll offset
- `MaxRetention`: int - Maximum logs to retain (default: 5000)
- `LastUpdated`: time.Time - Last state update timestamp
- `Version`: uint64 - State version for change tracking

### LogEntry

**Purpose**: Single log entry with metadata

**Fields**:
- `Timestamp`: time.Time - When log was emitted
- `Level`: LogLevel enum - Severity level
- `Source`: string - Process name
- `Message`: string - Log message content
- `Highlighted`: bool - Whether this entry matches search query

### WebState

**Purpose**: State for Web tab (proxy HTTP events)

**Fields**:
- `Events`: []ProxyEvent - Captured HTTP request/response events
- `EndpointFilter`: string - Filter by endpoint URL
- `StatusCodeFilter`: []int - Filter by HTTP status codes
- `MethodFilter`: []string - Filter by HTTP methods (GET, POST, etc.)
- `SelectedIndex`: int - Currently selected event index
- `ShowDetails`: bool - Whether detail view is expanded
- `MaxRetention`: int - Maximum events to retain (default: 1000)
- `LastUpdated`: time.Time - Last state update timestamp
- `Version`: uint64 - State version for change tracking

### ProxyEvent

**Purpose**: Single HTTP request/response captured by proxy

**Fields**:
- `ID`: string - Unique event identifier
- `Timestamp`: time.Time - When request was captured
- `Method`: string - HTTP method (GET, POST, etc.)
- `URL`: string - Request URL
- `StatusCode`: int - Response status code
- `RequestHeaders`: map[string]string - HTTP request headers
- `ResponseHeaders`: map[string]string - HTTP response headers
- `RequestBody`: string - Request body (truncated if large)
- `ResponseBody`: string - Response body (truncated if large)
- `Duration`: time.Duration - Request/response round-trip time
- `Size`: int - Response body size in bytes

### URLsState

**Purpose**: State for URLs tab (discovered from process logs)

**Fields**:
- `URLs`: []DiscoveredURL - List of URLs found in logs
- `ProcessFilter`: string - Filter by process name
- `SelectedIndex`: int - Currently selected URL index
- `DeduplicationMap`: map[string]bool - Process+URL deduplication
- `LastUpdated`: time.Time - Last state update timestamp
- `Version`: uint64 - State version for change tracking

### DiscoveredURL

**Purpose**: URL discovered from process output

**Fields**:
- `URL`: string - The discovered URL
- `Process`: string - Source process name
- `Timestamp`: time.Time - When URL was discovered
- `FirstSeen`: time.Time - First time this URL was seen
- `Count`: int - Number of times URL appeared

### ProxyState

**Purpose**: State for Proxy tab (proxy server status and stats)

**Fields**:
- `Enabled`: bool - Whether proxy server is running
- `Port`: int - Proxy server port
- `ActiveConnections`: int - Current active connections
- `Endpoints`: []ProxyEndpoint - Discovered endpoints
- `Stats`: ProxyStats - Traffic statistics
- `ReverseMode`: bool - Whether reverse proxy mode is enabled
- `ShareableURLs`: map[string]string - Public URLs for reverse mode
- `LastUpdated`: time.Time - Last state update timestamp
- `Version`: uint64 - State version for change tracking

### SettingsState

**Purpose**: State for Settings tab (application configuration)

**Fields**:
- `Settings`: []Setting - All configuration settings
- `SelectedIndex`: int - Currently selected setting index
- `Modified`: map[string]bool - Track which settings changed
- `ValidationErrors`: map[string]string - Validation error messages
- `LastUpdated`: time.Time - Last state update timestamp
- `Version`: uint64 - State version for change tracking

### Setting

**Purpose**: Single configuration setting with metadata

**Fields**:
- `Key`: string - Setting key (e.g., "mcp.port")
- `Value`: string - Current value
- `Default`: string - Default value
- `Source`: ConfigSource enum - Where value came from (Default, File, Environment)
- `Description`: string - Human-readable description
- `Type`: SettingType enum - Value type (String, Int, Bool, Path)
- `ValidationRule`: string - Validation regex or rule

### CommandWindowState

**Purpose**: State for Command Window (slash command palette)

**Fields**: (Already well-defined, documented here for completeness)
- `ShowingCommandWindow`: bool
- `CommandWindowWidth`: int
- `ShowingRunDialog`: bool
- `DetectedCommands`: []ExecutableCommand
- `SelectedCommandIndex`: int
- `ShowingCustomCommand`: bool
- `CustomCommandText`: string
- `LastError`: string
- `Version`: uint64

## Entity Relationships

```
MasterData (single source of truth)
├── ProcessesState
├── ErrorsState
│   └── GroupedErrors: []GroupedError
├── LogsState
│   └── Entries: []LogEntry
├── URLsState
│   └── URLs: []DiscoveredURL
├── WebState
│   └── Events: []ProxyEvent
├── ProxyState
│   └── Endpoints: []ProxyEndpoint
├── SettingsState
│   └── Settings: []Setting
└── CommandWindowState
```

## Immutability Contract

**All state objects MUST be immutable**:
- No mutexes or locks allowed
- All update methods return new state instances
- Copy-on-write pattern for all mutations
- Version field incremented on every change
- Thread safety via atomic pointer swaps in Model

**Example Immutable Update**:
```go
func (s *ErrorsState) AddGroupedError(err GroupedError) *ErrorsState {
    newState := *s  // Shallow copy
    newState.GroupedErrors = make([]GroupedError, len(s.GroupedErrors)+1)
    copy(newState.GroupedErrors, s.GroupedErrors)
    newState.GroupedErrors[len(s.GroupedErrors)] = err
    newState.Version++
    newState.LastUpdated = time.Now()
    return &newState
}
```

## State Size Limits

| State | Retention | Rationale |
|-------|-----------|-----------|
| ErrorsState.GroupedErrors | 1000 | Balance history vs. memory |
| LogsState.Entries | 5000 | Logs are high-volume |
| WebState.Events | 1000 | HTTP events are detailed |
| URLsState.URLs | 500 | URLs are typically low-volume |

When limits exceeded, oldest entries are discarded (FIFO).

## Validation Summary

All entities must pass validation before being added to state:
- Non-empty required fields
- Valid index ranges (SelectedIndex)
- Timestamp sanity checks (not in future)
- Retention limits enforced
- Relationship integrity (e.g., ProcessName exists in ProcessesState)

---
*Phase 1: Data Model Complete*
