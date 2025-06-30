# Brummer Telemetry Enhancements

## Overview

The JavaScript telemetry injector (`internal/proxy/monitor.js`) has been significantly enhanced to provide comprehensive browser-side metrics collection. These enhancements enable deeper insights into web application behavior during development.

## New Features

### 1. Advanced Network Interception

- **Fetch API Interception**: Captures all fetch requests with detailed request/response data
- **XMLHttpRequest Monitoring**: Full XHR lifecycle tracking including headers and response bodies
- **Request Correlation**: Each request gets a unique ID for tracking request/response pairs
- **Error Tracking**: Network failures are captured with detailed error information
- **JSON Response Bodies**: Automatically captures JSON response bodies (up to 5KB)

### 2. DOM Mutation Monitoring

- **Real-time DOM Change Tracking**: Monitors additions, removals, and attribute changes
- **Resource Loading Detection**: Tracks when scripts, stylesheets, and links are added
- **Style Change Monitoring**: Captures changes to body and HTML element styles
- **Batched Reporting**: Mutations are batched for 1 second to reduce telemetry volume

### 3. Storage Event Monitoring

- **localStorage Tracking**: Monitors all localStorage operations (set, remove, clear)
- **sessionStorage Tracking**: Monitors all sessionStorage operations
- **Cross-Tab Events**: Captures storage changes from other tabs/windows
- **Size Tracking**: Reports the size of stored values

### 4. Enhanced Performance Monitoring

- **Web Vitals**: Tracks FCP, LCP, FID, CLS, and other core metrics
- **Paint Timing**: Captures first-paint and first-contentful-paint
- **Layout Shift Detection**: Monitors cumulative layout shift with source tracking
- **First Input Delay**: Measures responsiveness to user interactions
- **Navigation Timing**: Detailed page load performance metrics
- **Long Task Detection**: Identifies JavaScript tasks blocking the main thread

### 5. Custom Metrics API

The page now exposes `window.brummerTelemetry` (and `window.mcpTelemetry` alias) with:

- `track(eventName, data)`: Track custom events
- `mark(name)`: Create performance marks
- `measure(name, startMark, endMark)`: Measure between marks
- `error(message, details)`: Log custom errors
- `action(action, target, metadata)`: Track user actions
- `feature(featureName, metadata)`: Track feature usage

### 6. Enhanced Interaction Tracking

- **Comprehensive Click Tracking**: Captures coordinates (client, page, screen, element-relative)
- **Viewport Information**: Includes viewport size and scroll position with each interaction
- **Selector Path Building**: Creates precise CSS selectors for clicked elements
- **Form Field Metadata**: Tracks form structure without capturing sensitive values
- **Input Debouncing**: Groups rapid input changes to reduce noise
- **Focus Duration**: Measures time spent in form fields
- **Keyboard Shortcuts**: Tracks modifier key combinations
- **Mouse Movement**: Throttled tracking of cursor position
- **Double-click and Right-click**: Separate tracking for different click types

### 7. Scroll Tracking

- **Debounced Reporting**: Only sends telemetry after scrolling stops (150ms delay)
- **Scroll Distance**: Tracks total distance scrolled in both X and Y directions
- **Scroll Duration**: Measures how long a scroll session lasts
- **Scroll Percentage**: Reports how far through the document the user has scrolled
- **Performance Optimized**: Uses passive event listeners

## Testing

To test the enhanced telemetry:

1. Build Brummer: `make build`
2. Run the test script: `./test-telemetry.sh`
3. In another terminal, start Brummer: `./brum`
4. Open the proxy URL in your browser
5. Interact with the test page and observe telemetry

## Debug Commands

The enhanced telemetry includes powerful debug commands accessible via browser console:

- `__brummer.debug.status()` - Show telemetry dashboard
- `__brummer.debug.timeline()` - Show event timeline
- `__brummer.debug.flush()` - Force send buffered events
- `__brummer.debug.ping()` - Test endpoint connectivity
- `__brummer.debug.clear()` - Clear event buffer
- `__brummer.debug.ws()` - Get WebSocket connection
- `__brummer.debug.reconnect()` - Reconnect WebSocket

## Performance Considerations

- **Throttling**: Interaction events are throttled to prevent overwhelming the system
- **Debouncing**: Input and scroll events are debounced to reduce noise
- **Batching**: Telemetry is batched and sent every 2 seconds
- **Size Limits**: Response bodies and other large data are truncated
- **Selective Monitoring**: Each monitoring feature can be individually disabled

## Privacy Considerations

- **No Sensitive Data**: Form values, passwords, and other sensitive data are not captured
- **Metadata Only**: Only form structure and interaction metadata is collected
- **Truncated Content**: Text content is limited to 100 characters
- **No Personal Information**: The system is designed for development use only