---
sidebar_position: 4
---

# Troubleshooting

Common issues and solutions for the Brummer browser extension.

## Installation Issues

### Extension Not Appearing

If the extension icon doesn't appear after installation:

1. **Check browser compatibility**:
   - Chrome/Chromium: Version 88+
   - Firefox: Version 78+
   - Edge: Version 88+

2. **Verify installation**:
   - Chrome: Navigate to `chrome://extensions/`
   - Firefox: Navigate to `about:addons`
   - Ensure Brummer extension is enabled

3. **Restart browser**:
   - Complete browser restart (not just closing tabs)
   - Clear browser cache if needed

### Permission Errors

If you see permission-related errors:

```
Error: Missing host permission for the tab
```

**Solution**:
1. Click the extension icon
2. Select "Manage Extension"
3. Enable "Allow access to file URLs" if working with local files
4. Grant necessary permissions when prompted

## Connection Issues

### Cannot Connect to Brummer TUI

If the extension can't connect to the Brummer process:

1. **Verify Brummer is running**:
   ```bash
   ps aux | grep brum
   ```

2. **Check MCP server status**:
   ```bash
   brum --status
   ```

3. **Verify port availability**:
   ```bash
   lsof -i :3280  # Default MCP port
   ```

4. **Firewall settings**:
   - Ensure localhost connections are allowed
   - Check if antivirus is blocking connections

### Connection Drops Frequently

If the connection is unstable:

1. **Increase timeout settings**:
   - Open extension options
   - Increase "Connection Timeout" to 30 seconds

2. **Check system resources**:
   - High CPU/Memory usage can cause drops
   - Close unnecessary applications

3. **Disable conflicting extensions**:
   - Ad blockers
   - Privacy extensions
   - Other developer tools

## Display Issues

### Icons Not Showing

If status icons appear as boxes or missing:

1. **Clear extension cache**:
   - Right-click extension icon
   - Select "Manage Extension"
   - Click "Clear Data"

2. **Reinstall extension**:
   - Remove and reinstall from store
   - Or reload unpacked extension in developer mode

### Incorrect Status Display

If the status doesn't match actual process state:

1. **Force refresh**:
   - Click extension icon
   - Select "Refresh Status"

2. **Check time sync**:
   - Ensure system time is correct
   - Extension uses timestamps for synchronization

## Performance Issues

### High Memory Usage

If the extension uses excessive memory:

1. **Limit log retention**:
   - Open extension settings
   - Reduce "Max Log Entries" (default: 1000)
   - Enable "Auto-clear old logs"

2. **Disable unused features**:
   - Turn off network monitoring if not needed
   - Disable auto-refresh for static sites

### Browser Slowdown

If the browser becomes sluggish:

1. **Reduce update frequency**:
   - Settings → Performance
   - Increase "Update Interval" to 1000ms

2. **Limit monitored processes**:
   - Configure process filters
   - Monitor only essential processes

## Feature-Specific Issues

### Auto-refresh Not Working

If pages don't refresh automatically:

1. **Check refresh settings**:
   - Ensure auto-refresh is enabled
   - Verify URL patterns match your site

2. **Content Security Policy**:
   - Some sites block auto-refresh
   - Add site to whitelist in extension settings

### DevTools Panel Missing

If the Brummer tab doesn't appear in DevTools:

1. **Enable DevTools experiments**:
   - Chrome: `chrome://flags/#enable-devtools-experiments`
   - Enable experimental features

2. **Reload DevTools**:
   - Close and reopen DevTools
   - Try different DevTools docking positions

### Error Overlay Not Appearing

If error notifications don't show:

1. **Check notification permissions**:
   - Browser settings → Notifications
   - Allow notifications from extension

2. **Verify error detection**:
   - Check if errors appear in Brummer TUI
   - Ensure error patterns are configured

## Debug Mode

Enable debug mode for detailed diagnostics:

1. **Open extension options**
2. **Enable "Debug Mode"**
3. **Open browser console** (F12)
4. **Look for "[Brummer]" prefixed messages**

### Debug Information to Collect

When reporting issues, include:

```
1. Browser version
2. Extension version
3. Brummer version
4. Console errors (F12)
5. Network tab HAR file
6. Steps to reproduce
```

## Common Error Messages

### "WebSocket connection failed"

**Cause**: MCP server not accessible
**Solution**: 
- Ensure Brummer is running with MCP enabled
- Check firewall settings
- Verify port 3280 is available

### "Invalid response format"

**Cause**: Version mismatch between extension and Brummer
**Solution**:
- Update both to latest versions
- Clear extension cache
- Restart Brummer

### "Process not found"

**Cause**: Process ended or PID changed
**Solution**:
- Refresh process list
- Restart the process from Brummer TUI

## Getting Help

If you're still experiencing issues:

1. **Check GitHub Issues**: [github.com/yourusername/brummer/issues](https://github.com/yourusername/brummer/issues)
2. **Join Discord**: [discord.gg/brummer](https://discord.gg/brummer)
3. **File a bug report** with:
   - System information
   - Steps to reproduce
   - Debug logs
   - Screenshots if applicable

## Reset to Default

If all else fails, reset the extension:

1. **Backup settings** (if needed)
2. **Remove extension** completely
3. **Clear browser storage**:
   ```
   localStorage.clear()
   sessionStorage.clear()
   ```
4. **Reinstall extension**
5. **Reconfigure settings**