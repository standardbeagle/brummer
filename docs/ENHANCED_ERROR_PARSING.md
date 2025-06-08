# Enhanced Error Parsing in Brummer

## Overview

Brummer now features ultra-robust error parsing that intelligently groups multi-line errors, extracts stack traces, and provides comprehensive error context. This feature transforms sloppy build and development tool log output into useful, structured error information.

## Key Features

### 1. Multi-Line Error Detection
- Automatically detects and groups related error lines
- Handles complex nested error structures (like MongoDB errors)
- Preserves full error context across multiple log entries

### 2. Language-Specific Parsing
Brummer recognizes error patterns from multiple languages:
- **JavaScript/Node.js**: UnhandledRejection, TypeError, ReferenceError, etc.
- **Go**: panic messages and error traces
- **Python**: Tracebacks and exceptions
- **Java**: Stack traces and exceptions
- **Rust**: Compiler errors with error codes
- **TypeScript**: Build and compilation errors

### 3. Smart Log Prefix Removal
Automatically strips common log prefixes:
- Timestamps: `[12:52:32]`, `(12:52:32)`, `12:52:32`
- Process names: `[dev]`, `(dev)`, `dev:`
- Preserves the actual error content

### 4. Error Context Extraction
- **Error Type**: Identifies specific error types (MongoError, TypeError, etc.)
- **Main Message**: Extracts the primary error message
- **Stack Traces**: Captures and formats stack trace information
- **Additional Context**: Preserves related information like error codes, hostnames

### 5. Enhanced Error Display
The Errors tab now shows:
- Structured error information with clear formatting
- Stack traces with proper indentation
- Additional context lines
- Visual separators between different errors

## Example: MongoDB Connection Error

Input (fragmented log lines):
```
[12:52:32] dev:  ⨯ unhandledRejection: [MongoServerSelectionError: getaddrinfo ENOTFOUND mongodb.localhost] {
[12:52:32] dev:   errorLabelSet: Set(0) {},
[12:52:32] dev:   reason: [TopologyDescription],
[12:52:32] dev:   code: undefined,
[12:52:32] dev:   [cause]: [MongoNetworkError: getaddrinfo ENOTFOUND mongodb.localhost] {
[12:52:32] dev:     errorLabelSet: Set(1) { 'ResetPool' },
[12:52:32] dev:     beforeHandshake: false,
[12:52:32] dev:     [cause]: [Error: getaddrinfo ENOTFOUND mongodb.localhost] {
[12:52:32] dev:       errno: -3008,
[12:52:32] dev:       code: 'ENOTFOUND',
[12:52:32] dev:       syscall: 'getaddrinfo',
[12:52:32] dev:       hostname: 'mongodb.localhost'
[12:52:32] dev:     }
[12:52:32] dev:   }
[12:52:32] dev: }
```

Output (structured error display):
```
12:52:32 [dev] MongoServerSelectionError
DNS lookup failed - ENOTFOUND mongodb.localhost (hostname: mongodb.localhost)
  errorLabelSet: Set(0) {},
  reason: [TopologyDescription],
  code: undefined,
  errno: -3008,
  syscall: 'getaddrinfo'
─────────────────────────────────────────
```

## Testing

To test the enhanced error parsing:

```bash
cd test-project
brum
# Run the mongo-error script from the Scripts tab
```

## Benefits

1. **Better Error Understanding**: Multi-line errors are no longer fragmented
2. **Faster Debugging**: Stack traces and context are properly grouped
3. **Cleaner Display**: Log prefixes don't clutter the error message
4. **Copy-Friendly**: The 'c' key copies the complete error context

## Implementation Details

The error parser uses:
- State machine for tracking error context
- Pattern matching for different error formats
- Intelligent line continuation detection
- Language-specific error handling

This makes Brummer exceptionally good at turning messy development tool output into actionable error information.