# Browser Compatibility

## Firefox
- Use `manifest-v2.json` (rename to `manifest.json`)
- Firefox doesn't fully support Manifest V3 yet
- Tested with Firefox Developer Edition 120+

## Chrome/Edge
- Use the default `manifest.json` (Manifest V3)
- Chrome requires Manifest V3 for new extensions
- Tested with Chrome 120+ and Edge 120+

## Installation Instructions

### For Firefox:
```bash
cp manifest-v2.json manifest.json
# Then load in about:debugging
```

### For Chrome/Edge:
```bash
# Use the default manifest.json (already v3)
# Load in chrome://extensions or edge://extensions
```

## Key Differences

### Manifest V2 (Firefox):
- Uses `background.scripts` with `persistent: false`
- Has `devtools` permission
- Simple `web_accessible_resources` array

### Manifest V3 (Chrome/Edge):
- Uses `background.service_worker`
- Separate `host_permissions`
- Structured `web_accessible_resources` with matches

## Testing
The extension includes enhanced debugging features that work in both manifest versions:
- Styled console logs with emojis
- Connection status indicators
- Real-time tab tracking
- Comprehensive error logging

Use the `test.html` file to verify functionality in your browser.