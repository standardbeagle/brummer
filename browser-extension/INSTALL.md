# Installing Brummer Firefox Extension

## Quick Start

1. **Build the extension:**
   ```bash
   cd browser-extension
   ./create-icons.sh
   ./build.sh
   ```

2. **Install in Firefox:**
   - Open Firefox
   - Navigate to `about:debugging`
   - Click "This Firefox"
   - Click "Load Temporary Add-on..."
   - Select `build/manifest.json`

3. **Start using:**
   - Start Brummer: `brummer --port 7777`
   - Open Firefox DevTools (F12)
   - Look for the "🐝 Brummer" tab
   - Click "Connect" in the extension panel

## Detailed Installation

### Prerequisites

- Firefox 69+ (supports Manifest V2 extensions)
- Brummer running with MCP server (default port 7777)

### Step 1: Prepare Extension Files

```bash
# Navigate to the browser extension directory
cd browser-extension

# Create placeholder icons (optional - improves visual appearance)
./create-icons.sh

# Build the extension package
./build.sh
```

This creates:
- `build/` directory with all extension files
- `brummer-firefox-extension.zip` for distribution

### Step 2: Load Extension in Firefox

#### For Development/Testing:

1. **Open Firefox Developer Mode:**
   - Type `about:debugging` in the address bar
   - Press Enter

2. **Access This Firefox:**
   - Click "This Firefox" in the left sidebar

3. **Load Temporary Add-on:**
   - Click "Load Temporary Add-on..." button
   - Navigate to the `build/` directory
   - Select `manifest.json`
   - Click "Open"

4. **Verify Installation:**
   - The extension should appear in the list
   - You should see "Brummer DevTools" with a bee icon

#### For Permanent Installation:

Currently, you need to install as a temporary add-on. For permanent installation:

1. Submit `brummer-firefox-extension.zip` to Firefox Add-ons store
2. Or use Firefox Developer Edition for extended temporary installs

### Step 3: Using the Extension

1. **Start Brummer:**
   ```bash
   brummer --port 7777
   ```

2. **Open Developer Tools:**
   - Press F12 or right-click → "Inspect Element"
   - Look for the "🐝 Brummer" tab

3. **Connect to Brummer:**
   - Click the "🐝 Brummer" tab
   - Verify server URL (default: `http://localhost:7777`)
   - Click "Connect"
   - Status should change to "Connected"

4. **View Detected URLs:**
   - Run scripts in Brummer that output URLs
   - URLs will appear automatically in the extension
   - Click "Open" next to any URL to open it in a new tab

## Troubleshooting

### Extension Not Appearing

- **Check Firefox version**: Requires Firefox 69+
- **Reload extension**: Go to `about:debugging` → "This Firefox" → "Reload"
- **Check console**: Look for errors in Browser Console (Ctrl+Shift+J)

### Connection Issues

- **Verify Brummer is running**: `brummer --port 7777`
- **Check server URL**: Ensure it matches your Brummer instance
- **Test MCP server**: Visit `http://localhost:7777/mcp/scripts` in browser
- **Check CORS**: Brummer allows all origins by default

### No URLs Appearing

- **Run scripts**: Make sure you have scripts running in Brummer
- **Check logs**: Verify scripts are outputting URLs in their logs
- **Manual test**: Check if URLs appear in Brummer's TUI "URLs" tab

### Icons Missing

- **Run icon script**: `./create-icons.sh`
- **Install ImageMagick**: For better icon generation
- **Manual creation**: Create PNG files in `icons/` directory

## Uninstalling

### Temporary Extension:
- Go to `about:debugging` → "This Firefox"
- Find "Brummer DevTools"
- Click "Remove"

### Or restart Firefox (temporary extensions are removed automatically)

## Development

### Making Changes:

1. Edit source files in `browser-extension/`
2. Run `./build.sh` to rebuild
3. Go to `about:debugging` → "This Firefox"
4. Click "Reload" next to "Brummer DevTools"

### File Structure:

- `manifest.json` - Extension metadata and permissions
- `devtools.js` - Creates the DevTools panel
- `panel.html/js` - Main extension UI and logic
- `background.js` - Background processes
- `content.js` - Page content interaction (optional)

## Security

The extension only:
- Connects to localhost by default
- Requires explicit connection to Brummer
- Has no access to browsing data
- Only communicates with Brummer's MCP server

## Support

For issues:
1. Check this troubleshooting guide
2. Verify Brummer is working independently
3. Check browser console for JavaScript errors
4. Open GitHub issue with detailed description