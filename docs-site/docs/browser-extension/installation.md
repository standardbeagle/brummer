---
sidebar_position: 2
---

# Installation Guide

:::warning Alpha Notice
The browser extension is in Alpha. Installation requires manual steps and developer mode.
:::

## Prerequisites

- Firefox Developer Edition or Chrome/Edge
- Brummer installed and running
- Access to browser extension developer tools

## Firefox Installation

### 1. Prepare the Extension

```bash
cd browser-extension

# For Firefox, use Manifest V2
cp manifest-v2.json manifest.json

# Create required icon files
# Add bee-16.png, bee-32.png, bee-48.png, bee-128.png to icons/
```

### 2. Load in Firefox

1. Open Firefox
2. Navigate to `about:debugging`
3. Click "This Firefox"
4. Click "Load Temporary Add-on..."
5. Select the `manifest.json` file

:::note
Temporary add-ons are removed when Firefox restarts. For persistent installation, the extension needs to be signed by Mozilla.
:::

## Chrome/Edge Installation

### 1. Prepare the Extension

```bash
cd browser-extension

# Chrome uses Manifest V3 (default)
# Ensure manifest.json is the V3 version
```

### 2. Load in Chrome/Edge

1. Open Chrome/Edge
2. Navigate to `chrome://extensions` or `edge://extensions`
3. Enable "Developer mode" toggle
4. Click "Load unpacked"
5. Select the `browser-extension` directory

## Verifying Installation

1. Open Developer Tools (F12)
2. Look for the "🐝 Brummer" tab
3. The panel should appear with connection settings

## Connecting to Brummer

1. Ensure Brummer is running:
   ```bash
   brum --port 7777
   ```

2. In the Brummer DevTools panel:
   - Server URL should be `http://localhost:7777`
   - Click "Connect"
   - Status should change to "Connected"

## Enabling Browser Log Forwarding

To capture browser logs:

1. In the Brummer panel, toggle "Forward Browser Logs to Brummer"
2. Browser logs will now appear in Brummer's log view
3. This captures:
   - Console logs
   - JavaScript errors
   - Network requests
   - Resource loading errors

## Testing the Extension

Open the test page to verify functionality:

```bash
# In your browser
file:///path/to/brummer/browser-extension/test.html
```

Use the test buttons to verify:
- Console logging
- Error capture
- Network monitoring
- URL parameter handling

## Troubleshooting

### Extension Not Loading

- Ensure all icon files are present
- Check manifest.json syntax
- Look for errors in browser console

### Connection Failed

- Verify Brummer is running
- Check the port number (default 7777)
- Ensure no firewall is blocking localhost connections

### No Logs Appearing

- Enable browser logging in the panel
- Check browser console for errors
- Verify bearer token in network requests

## Building for Production

:::info Coming Soon
Production packaging and distribution through browser stores will be available in future releases.
:::

For now, use the development installation method described above.