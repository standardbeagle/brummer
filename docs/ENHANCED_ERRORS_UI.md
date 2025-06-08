# Enhanced Error View - Split Layout with Detail Navigation

## Overview

The Errors tab now features a sophisticated split-layout interface that transforms the error viewing experience from a simple list to an interactive debugging tool.

## New Features

### 1. Split Layout Design
- **Left Panel (1/3 width)**: Compact error list with key information
- **Right Panel (2/3 width)**: Detailed error analysis and context
- **Responsive**: Falls back to single-column on narrow screens (< 100 chars)

### 2. Interactive Error List
Each error shows:
- **Icon**: Visual severity indicator (❌ Error, ⚠️ Warning, 🔥 Critical)
- **Type**: Error type (MongoServerSelectionError, TypeError, etc.)
- **Message**: Truncated error message (50 chars max)
- **Meta**: Process name and timestamp

### 3. Detailed Error Panel
When an error is selected, the detail panel shows:

#### Core Information
- **Error Type**: Prominently displayed with color coding
- **Metadata**: Time, process, detected language
- **Main Message**: Full error message with proper formatting

#### Code Navigation
- **📍 Code Location**: Automatically extracts the lowest-level file:line reference
- **Smart Filtering**: Prioritizes project files over node_modules/system files
- **Multi-Language Support**: Handles JS, TS, Go, Python, Java, Rust file patterns

#### Stack Trace Analysis
- **Formatted Stack**: Clean, indented stack trace display
- **Nested Errors**: Properly displays error chains and causes
- **Context Preservation**: Shows additional error context

#### Raw Log Access
- **Complete Log**: Full raw log output for debugging
- **Preserved Formatting**: Original log structure maintained

### 4. Copy Enhancement
- **Header Notification**: Shows "📋 Error copied to clipboard" for 3 seconds
- **Complete Context**: Copies full error details including stack traces
- **Structured Format**: Well-formatted for sharing/debugging

### 5. Navigation
- **Arrow Keys**: Browse through error list
- **Enter**: Select error for detailed view
- **'c' Key**: Copy current error with full context
- **Auto-Selection**: Automatically selects first error when available

## Usage Examples

### MongoDB Connection Error
```
Left Panel:
❌ MongoServerSelectionError: DNS lookup failed - ENOTFOUND...
dev | 12:52:32

Right Panel:
MongoServerSelectionError Error

Time: 12:52:32 | Process: dev | Language: javascript

Error Message:
DNS lookup failed - ENOTFOUND mongodb.localhost (hostname: mongodb.localhost)

📍 Code Location:
database/connection.js:23:5

Additional Context:
  errorLabelSet: Set(0) {},
  reason: [TopologyDescription],
  code: undefined,
  errno: -3008
```

### JavaScript Stack Trace
```
Left Panel:
❌ TypeError: Cannot read property 'someMethod' of null
test | 14:32:10

Right Panel:
TypeError Error

📍 Code Location:
test-project/multi-error-test.js:18:9

Stack Trace:
  at Object.<anonymous> (test-project/multi-error-test.js:18:9)
  at Module._compile (node:internal/modules/cjs/loader.js:1126:14)
  at Object.Module._extensions..js (node:internal/modules/cjs/loader.js:1180:10)
```

## Technical Implementation

### State Management
- `selectedError`: Currently selected error context
- `errorsList`: Interactive list component
- `errorDetailView`: Scrollable detail viewport
- `copyNotification`: Temporary header message

### Event Handling
- Real-time error updates via event bus
- Selection tracking with navigation keys
- Automatic error list refresh
- Copy notification timing

### Layout Logic
- Dynamic sizing based on terminal width
- Border styling for visual separation
- Responsive fallback for narrow terminals
- Proper viewport management

## Benefits

1. **Faster Debugging**: Quickly scan error list and dive into details
2. **Better Context**: See full error structure and stack traces
3. **Code Navigation**: Immediately identify problematic file/line
4. **Enhanced Copy**: Share complete error context easily
5. **Visual Clarity**: Clear separation between errors and detailed analysis

This enhanced error view transforms Brummer from a simple log viewer into a powerful debugging tool.