## 2025-09-21
- AIPTYComponent: Stop treating EventChan PTYEventOutput as UI output; rely solely on OutputChan for rendering. Keeps EventChan for lifecycle only and avoids duplicate output signaling.
- Deprecated old AIViewComponent (placeholder) with file-level comment to prevent accidental use; AIPTYComponent is the active AI UI.
- Build and component tests passed (`TestAIPTY_StartTerminalProducesOutput`).

- Added debug AI provider: "/ai test" starts a PTY that prints BRUMMER_TEST_COUNTER N every 0.5s for visual verification of streaming and tab persistence.


# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- Race conditions in BubbleTea PTY component:
  - Fixed concurrent read/write access to component state in `startProviderSession()`
  - Fixed goroutine access to session/provider fields in `pokePTYCmd()`
  - Captured component state before entering goroutines
  - Pass data via messages instead of direct state modification
  - All component state updates now happen only in Update() method
  - Race detector validation: CLEAN (zero warnings)

### Added
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