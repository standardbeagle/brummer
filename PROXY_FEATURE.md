# Brummer Automatic Proxy Feature

## Overview

Brummer now includes an automatic proxy detection and injection feature that enhances browser-based development workflows. When Brummer detects a development server URL in process logs, it automatically creates a proxy server that injects logging capabilities into your web pages.

## How It Works

1. **URL Detection**: Brummer monitors process logs for patterns indicating a development server has started (e.g., "Server running at http://localhost:3000")

2. **Automatic Proxy Creation**: When a local development URL is detected, Brummer automatically:
   - Allocates a free port (8080-8099)
   - Creates a reverse proxy to your development server
   - Starts the proxy server

3. **JavaScript Injection**: The proxy injects a logging script into HTML pages that:
   - Captures console.log, console.error, console.warn, etc.
   - Tracks page navigation
   - Monitors network requests (fetch)
   - Reports JavaScript errors and unhandled promise rejections

4. **Log Integration**: All browser logs are sent back to Brummer and appear in the main log view, tagged with the process name

## Supported Development Servers

The proxy detector recognizes URLs from:
- Next.js (`ready - started server on...`)
- Vite (`Local: http://...`)
- Webpack Dev Server (`Project is running at...`)
- Create React App (`You can now view...`)
- Generic patterns (`server running at...`, `listening on...`)

## UI Indicators

- **Process List**: Processes with active proxies show a 🔗 icon
- **Process Description**: Shows the proxy URL (e.g., "Proxy: http://localhost:8081")
- **URLs Tab**: Displays all active proxies with their target URLs

## Example Usage

1. Start a development server script:
   ```bash
   npm run dev  # or yarn dev, pnpm dev, etc.
   ```

2. When Brummer detects the server URL in logs, you'll see:
   ```
   🚀 Starting proxy on http://localhost:8081 → http://localhost:3000 for dev
   ```

3. Open the proxy URL in your browser instead of the original URL

4. All browser console logs, errors, and network activity will appear in Brummer's log view

## Test Script

A test script is included at `test-project/proxy-test.js` that demonstrates the feature:

```javascript
// Creates a simple HTTP server on port 3456
// Brummer will detect and proxy it automatically
npm run proxy-test
```

## Technical Details

- **Port Range**: Proxies use ports 8080-8099
- **Cleanup**: Proxies are automatically stopped when the process exits
- **Security**: Each proxy session uses a unique token for authentication
- **Performance**: Minimal overhead, only HTML responses are modified

## Limitations

- Only works with local development servers (localhost, 127.0.0.1)
- HTML content must have a proper `</body>` or `</html>` tag for injection
- Content Security Policy headers are removed to allow inline scripts

## Integration with Browser Extension

The proxy feature complements the Brummer browser extension by providing an alternative way to capture browser logs without requiring extension installation.