# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Centralized layout region calculation for AI Coder view
  - Single source of truth for terminal dimensions
  - Automatic PTY resizing when terminal window changes
  - Proper viewport dimensions accounting for header/footer/padding
  - Integration with layout_regions.go system
- Implemented previously stubbed command executor features
  - `/show <pattern>` now sets log filter pattern in model state
  - `/hide <pattern>` now sets log hide pattern in model state
  - `/clear <process>` now clears logs for specific processes
  - `/proxy list` now displays registered proxy URL mappings
  - `/toggle-proxy` now switches between full and reverse proxy modes

### Removed
- Commented-out TODO code in command handler
  - Removed message type mismatch workarounds
  - Removed placeholder message sending code
  - Cleaned up 115 lines of dead code from command handler

### Changed
- AI Coder state architecture simplified and modernized
  - Removed deprecated PTYOutputBuffers field (use RawBuffers instead)
  - Removed deprecated TerminalStates field (use ScreenBuffer instead)
  - Removed deprecated polling state management fields
  - Removed legacy AddPTYOutput and UpdateTerminalState methods
  - Cleaner, more maintainable state management

### Added
- AI Coder viewport scrolling with full scrollback buffer support
  - Arrow keys (↑↓) and vim keys (j/k) now scroll through terminal history
  - PageUp/PageDown for page-based scrolling
  - Full 1000-line scrollback buffer from vt10x terminal emulator
  - Visual scrollbar indicator on right side of viewport
  - Scroll percentage indicator in footer (e.g., "[45%]")
  - Comprehensive test suite with 6 test cases and performance benchmarks
- Comprehensive test suite for slash command autocomplete (46 test cases)
  - Tests for AI provider validation with config and fallback
  - Tests for run, clear, proxy, and arbitrary command validation
  - Tests for suggestion filtering and initialization

### Fixed
- AI Coder viewport scrolling not working
  - Screen buffer now provides full vt10x scrollback (1000 lines) instead of fixed 24-line window
  - Viewport properly handles windowing and scroll position
  - Matches behavior of professional terminal emulators (iTerm2, gnome-terminal)
  - Performance: <1ms per scroll operation, ~1.6ms for full buffer read
- Command palette autocomplete dropdown now shows immediately when typing "/"
  - Previously required pressing backspace or another key to display suggestions
  - Autocomplete now initializes with "/" on command palette open
- AI provider validation now accepts all configured providers
  - `/ai bash` and `/ai gemini` commands now execute correctly
  - Dynamic validation reads from config instead of hardcoded list
  - Supports custom providers defined in .brum.toml configuration

### Added
- Defensive update listener pattern for automatic TUI updates
  - Persistent listener ensures all updateChan events trigger UI updates immediately
  - Eliminates need for manual `waitForUpdates()` calls in message handlers
  - Self-healing design with timeout protection prevents UI freezes
  - Real-time event processing without requiring user input to "pump" updates
- Proxy creation logging for better visibility
  - Headless mode displays proxy creation in stdout
  - TUI mode logs proxy creation to system logs
  - Shows mapping between detected URLs and proxy URLs

### Fixed
- Web view not updating in real-time when proxy requests arrive
  - Events were queued but only processed on keypresses
  - Now updates immediately as HTTP requests occur
- EventBus worker deadlock when updateChan is full
  - Changed to non-blocking send with select statement
  - Prevents worker goroutines from blocking indefinitely

### Changed
- Simplified message handler code by removing 13+ manual update scheduling calls
- Data source consistency: all view badges now use masterData states as single source of truth
- Removed redundant `selectedProcess` field from Model struct (now uses LogsState)

### Added
- TUI keyboard navigation and command improvements
  - Context-aware slash command palette (opens on all screens except input fields and PTY)
  - Docker and Compose command autocomplete (docker-up, docker-down, docker-build, compose-up, compose-down)
  - Command palette now displays autocomplete suggestions with proper syncing to state
  - Commands automatically clear from palette after execution
  - Comprehensive command documentation for 18 slash commands
- Logs view scrolling and navigation
  - Arrow key (↑/↓) and vim-style (j/k) navigation for scrolling through logs
  - Space key to toggle auto-scroll on/off
  - Auto-scroll automatically pauses when user scrolls up to view history
  - Visual indicator shows auto-scroll state (🔄 Auto-scroll / ⏸ Paused)
  - Scrolling preserves view offset for manual navigation
- Web view proxy integration
  - Real-time HTTP request display from proxy server
  - Event-driven architecture connecting proxy events to web view state
  - Request tracking with full details (status, duration, size, authentication, telemetry)
  - Request history (keeps last 1000 requests)
  - Statistics tracking by status code and HTTP method
  - Auto-scroll for newest requests with filtering support
- Script library management system with MCP integration
  - New `repl_library` MCP tool for managing JavaScript/TypeScript debugging functions
  - Scripts stored as TypeScript files with JSON front matter in `.brum/scripts` directory
  - Thread-safe library manager with 60-second caching for performance
  - Auto-injection of library into browser context via `repl_execute` tool
  - CRUD operations: list, add, remove, update, search, categories, get, inject
  - Five built-in debugging scripts:
    - `getDetails` - Get detailed information about DOM elements
    - `componentTree` - Build hierarchical view of React/Vue components
    - `traceEvents` - Trace and log DOM events in real-time
    - `getBoundingBoxTree` - Visualize element boundaries and layout
    - `findLayoutIssues` - Detect common CSS/layout problems
  - Rich metadata support: description, category, tags, examples, parameters, version
  - Global access via `window.brummerLibrary` and `window.lib` aliases

### Fixed
- TUI responsiveness and usability improvements
  - Removed double-press requirement for Ctrl+C and ESC (immediate quit)
  - Removed 5-second startup signal handling test that caused delays
  - Fixed 'c' key clear logs command to clear both state and log store
  - Removed keyboard debug logging that cluttered logs screen
  - Fixed URL display to show full URL with protocol (http:// or https://)
  - Context-aware keyboard handlers with proper fallthrough pattern
- Lock-free state management enhancements
  - Added immutable scroll state methods (ScrollUp, ScrollDown, EnableAutoScroll, DisableAutoScroll)
  - Enhanced LogsState with proper scroll offset management
  - All state updates follow lock-free immutable pattern for 20-30x performance
- PTY terminal responsiveness optimization with cross-platform support
  - Fixed 1-second input delay in AI coder PTY sessions using non-canonical terminal mode
  - Added platform-specific terminal configuration (Unix/Windows) with proper build constraints
  - Implemented terminal state preservation and restoration for clean session lifecycle
  - Enhanced PTY file descriptor validation to prevent unsafe operations
  - Added graceful fallback when terminal optimization fails
- MCP error handling now returns proper Go errors instead of success responses with error fields
  - All MCP tools updated to follow JSON-RPC 2.0 error response standards
  - `about` tool file write failures now return detailed error context
  - Browser tools return "proxy server not available" errors immediately instead of timing out
- Security vulnerabilities in script management:
  - Path traversal prevention with comprehensive validation and Windows case-insensitive comparison
  - JavaScript code sanitization to prevent injection attacks
  - Secure file path handling with multiple validation layers
- Race conditions in library manager initialization using `sync.Once` pattern
- Resource management issues in REPL operations:
  - Fixed race condition in library check cleanup
  - Removed orphaned goroutines in library injection cleanup
  - Proper cleanup order for WebSocket response channels
- Resource leaks with proper cleanup for goroutines and channels
- Race condition in MCP server library manager initialization:
  - Added atomic flags to prevent concurrent builtin script installation
  - Implemented thread-safe installation status tracking with proper error handling
  - Used lock-free atomic operations following project's design philosophy

### Changed
- Updated error messages across MCP tools to provide better debugging context with session IDs
- Browser tool tests updated to expect new error messages
- Standardized MCP error message formats for consistent user experience:
  - Introduced centralized error constants and formatter functions
  - Unified proxy server error messages with helpful configuration guidance
  - Consistent timeout error messages with contextual information
  - Improved parameter validation error messages with operation context
- Refactored `repl_library` handler into separate methods to reduce complexity
- Made timeout values configurable via environment variables:
  - `BRUMMER_LIBRARY_CHECK_TIMEOUT` (default: 1 second)
  - `BRUMMER_LIBRARY_INJECT_TIMEOUT` (default: 2 seconds)  
  - `BRUMMER_REPL_RESPONSE_TIMEOUT` (default: 5 seconds)

### Refactored
- TUI script selector and command window to use proper Bubble Tea patterns:
  - Replaced manual padding calculations with Lipgloss layout features
  - Created constants.go for all UI magic numbers
  - Improved code organization with smaller, focused methods
  - Added comprehensive documentation for critical initialization order
  - Enhanced error handling with nil checks and standardized error messages
  - Fixed terminal size detection for overlay windows
  - Created ScriptSelectorModel as proper tea.Model implementation

### Security
- Implemented comprehensive path traversal protection in script file operations
- Added JavaScript sanitization to prevent code injection attacks
- Validated all user inputs for script names and metadata