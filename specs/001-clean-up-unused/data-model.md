# Data Model - Code Cleanup and Refactoring

## Key Entities

### CodeAnalysisResult
**Purpose**: Represents results from static code analysis
**Attributes**:
- UnusedImports: []string - List of unused import statements
- UnreachableFunctions: []string - Functions that can never be called
- DeadCodeBlocks: []CodeBlock - Blocks of code with no execution paths
- AnalysisTimestamp: time.Time - When analysis was performed
- AffectedFiles: []string - Files that require cleanup

### KeyEventState
**Purpose**: Tracks keyboard event handling state
**Attributes**:
- ActiveHandlers: map[string]KeyHandler - Current active key handlers
- GlobalHandlers: []KeyHandler - Global key handlers (Ctrl+C, q, etc.)
- EventQueue: chan KeyEvent - Buffered channel for key events
- HandlerPriority: map[string]int - Priority levels for handler resolution

### AICoderSession
**Purpose**: Manages AI coding terminal session
**Attributes**:
- SessionID: string - Unique session identifier
- TerminalState: TerminalState - Current terminal emulator state
- BrowserConnections: []BrowserProxy - Connected browser debug tools
- ErrorStream: chan ErrorEvent - Stream of errors from browser/server
- ProcessBindings: map[string]Process - Associated managed processes

### PerformanceProfile
**Purpose**: Stores performance profiling data
**Attributes**:
- CPUSamples: []CPUSample - CPU usage samples
- MemorySnapshots: []MemorySnapshot - Memory usage over time
- KeyEventLatencies: []LatencyMeasurement - Key event response times
- HotspotFunctions: []FunctionStats - Functions consuming most resources
- ProfileTimestamp: time.Time - When profiling was performed

## Entity Relationships

```
CodeAnalysisResult (1) --affects--> (many) SourceFiles
KeyEventState (1) --manages--> (many) KeyHandlers
AICoderSession (1) --connects--> (many) BrowserProxies
PerformanceProfile (1) --measures--> (many) FunctionStats
```

## State Transitions

### KeyEventState Lifecycle
1. **Initialization** → Setup global handlers
2. **Active** → Process key events with priority resolution
3. **AISession** → Route specific keys to AI terminal, others to TUI
4. **Cleanup** → Remove handlers and flush queues

### AICoderSession Lifecycle
1. **Created** → Initialize terminal emulator
2. **Connecting** → Establish browser debug tool connections
3. **Active** → Full AI coding session with error streaming
4. **Closing** → Graceful shutdown of connections and terminal

## Validation Rules

### CodeAnalysisResult Validation
- All file paths must exist in project
- Unused imports must not be used in any reachable code
- Unreachable functions must have no callers

### KeyEventState Validation
- Global handlers must always be registered
- Handler priorities must be unique within scope
- Event queue cannot exceed buffer capacity

### AICoderSession Validation
- SessionID must be unique across active sessions
- Browser connections must validate connectivity
- Error stream must not block on slow consumers