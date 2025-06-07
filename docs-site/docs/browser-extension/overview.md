---
sidebar_position: 1
---

# Browser Extension Overview

:::warning Alpha Feature
The Brummer browser extension is currently in **Alpha**. Features may change, and you may encounter bugs. We welcome feedback and bug reports!
:::

## Introduction

The Brummer browser extension integrates with Firefox/Chrome DevTools to enhance your debugging experience by:

- Displaying URLs detected in your application logs
- Forwarding browser console logs to Brummer
- Providing real-time connection status
- Tracking active browser tabs with Brummer integration

## Key Features

### 🔗 URL Detection and Management
- Automatically detects URLs from your running scripts
- One-click opening of detected URLs
- Shows context and timestamp for each URL
- Real-time updates via Server-Sent Events

### 📊 Browser Log Integration
When enabled, captures and forwards to Brummer:
- Console logs (log, warn, error, info, debug)
- JavaScript errors with stack traces
- Promise rejections
- Network requests and responses
- Resource loading errors
- Page navigation events

### 🔌 Connection Monitoring
- Visual connection status indicator
- Real-time ping/pong health checks
- Latency display
- Automatic reconnection handling

### 📑 Active Tab Tracking
- Shows all browser tabs with Brummer logging enabled
- Visual indicators for active/inactive tabs
- Quick tab switching functionality
- Process information for each tab

## How It Works

1. **DevTools Integration**: Adds a "🐝 Brummer" panel to your browser's Developer Tools

2. **MCP Server Connection**: Connects to Brummer's MCP server (default port 7777)

3. **Bidirectional Communication**:
   - Receives URL detections from Brummer
   - Sends browser logs back to Brummer

4. **Content Script Enhancement**: When opening URLs with Brummer parameters, provides:
   - Styled console notifications
   - Floating connection status widget
   - Automatic log forwarding

## Browser Support

### Current Support
- **Firefox**: Full support with Manifest V2
- **Chrome/Edge**: Full support with Manifest V3

### Coming Soon
- Safari support
- Opera support

## Use Cases

### Development Debugging
- Track API endpoints your application calls
- Monitor browser-side errors alongside server logs
- Correlate client and server events

### Testing and QA
- Capture browser errors during test runs
- Monitor network failures
- Track navigation flows

### URL Management
- Quickly access development URLs
- Open multiple services from a single interface
- Track which process generated which URL

## Security Considerations

The extension:
- Only connects to localhost by default
- Uses bearer token authentication for browser tabs
- Automatically cleans up inactive connections
- Doesn't persist sensitive data

## Getting Started

Ready to enhance your browser debugging experience? Check out the [Installation Guide](./installation) to get started.

:::tip
The browser extension works best when used alongside the Brummer TUI, providing a complete view of your application's behavior from both server and client perspectives.
:::