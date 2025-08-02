# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

### Security
- Implemented comprehensive path traversal protection in script file operations
- Added JavaScript sanitization to prevent code injection attacks
- Validated all user inputs for script names and metadata