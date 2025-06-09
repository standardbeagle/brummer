---
sidebar_position: 1
---

# Common Issues

Quick solutions to frequently encountered problems with Brummer.

## Installation Issues

### "Command not found: brum"

**Problem**: After installation, `brum` command is not recognized.

**Solutions**:

1. **Add to PATH**:
   ```bash
   # For bash
   echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
   source ~/.bashrc
   
   # For zsh
   echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
   source ~/.zshrc
   ```

2. **Verify installation location**:
   ```bash
   # Check if binary exists
   ls -la ~/.local/bin/brum
   
   # If not there, check system location
   which brum
   ```

3. **Reinstall**:
   ```bash
   curl -sSL https://raw.githubusercontent.com/standardbeagle/brummer/main/quick-install.sh | bash
   ```

### "Permission denied" during installation

**Problem**: Installation fails with permission errors.

**Solution**:
```bash
# Install to user directory instead
make install-user

# Or manually
go build -o brum ./cmd/brum
mkdir -p ~/.local/bin
mv brum ~/.local/bin/
chmod +x ~/.local/bin/brum
```

### Go version errors

**Problem**: "Go version 1.21 or higher required"

**Solution**:
```bash
# Check current version
go version

# Update Go
# macOS
brew upgrade go

# Linux
sudo snap refresh go

# Or download from https://golang.org/dl/
```

## Runtime Issues

### Brummer won't start

**Problem**: Brummer crashes immediately or shows errors.

**Debugging steps**:

1. **Check for package.json**:
   ```bash
   # Brummer needs package.json
   ls package.json
   ```

2. **Try verbose mode**:
   ```bash
   brum --debug
   ```

3. **Check terminal compatibility**:
   ```bash
   # Ensure you're using a compatible terminal
   echo $TERM
   # Should show: xterm-256color or similar
   ```

### "No scripts found"

**Problem**: Brummer starts but shows no scripts.

**Solutions**:

1. **Verify package.json has scripts**:
   ```json
   {
     "scripts": {
       "dev": "node server.js",
       "test": "jest"
     }
   }
   ```

2. **Check file permissions**:
   ```bash
   ls -la package.json
   # Should be readable
   ```

### UI rendering issues

**Problem**: Interface looks broken or characters are wrong.

**Solutions**:

1. **Check terminal encoding**:
   ```bash
   locale
   # Should show UTF-8
   ```

2. **Try different terminal**:
   - Use iTerm2 (macOS)
   - Use Windows Terminal (Windows)
   - Use modern terminal emulator

3. **Disable Unicode**:
   ```bash
   BRUMMER_ASCII=1 brum
   ```

## Process Management Issues

### Process won't start

**Problem**: Selecting a script doesn't start the process.

**Debugging**:

1. **Check script validity**:
   ```bash
   # Test script directly
   npm run <script-name>
   ```

2. **Look for port conflicts**:
   ```bash
   # Check if port is in use
   lsof -i :3000
   ```

3. **Check environment variables**:
   ```bash
   # Ensure required env vars are set
   env | grep NODE_ENV
   ```

### Process starts but immediately exits

**Problem**: Process shows as failed immediately.

**Common causes**:

1. **Missing dependencies**:
   ```bash
   npm install
   ```

2. **Script errors**:
   - Check the Logs view for error details
   - Run script manually to debug

3. **Missing files**:
   - Verify all required files exist
   - Check working directory

### Can't stop processes

**Problem**: Pressing 's' doesn't stop the process.

**Solutions**:

1. **Force quit Brummer**:
   - Press `Ctrl+C` twice
   - All child processes will be terminated

2. **Manual cleanup**:
   ```bash
   # Find stuck processes
   ps aux | grep node
   
   # Kill specific process
   kill -9 <PID>
   ```

3. **Use cleanup script**:
   ```bash
   ./cleanup-processes.sh
   ```

## Log Management Issues

### Logs not showing

**Problem**: Processes are running but no logs appear.

**Solutions**:

1. **Check log buffering**:
   - Some processes buffer output
   - Add `--force-color` to scripts
   - Use `unbuffer` command

2. **Verify process is actually logging**:
   ```bash
   # Run directly to see output
   npm run <script> 
   ```

### Log filtering not working

**Problem**: Slash commands don't filter logs.

**Solutions**:

1. **Correct syntax**:
   ```bash
   /show error     # Correct
   /show "error"   # Also correct
   /show error.*   # Regex pattern
   ```

2. **Clear filters**:
   - Press `/` then Enter (empty filter)
   - Restart Brummer

### Too many logs

**Problem**: Log view is overwhelming.

**Solutions**:

1. **Use high-priority mode**:
   - Press `p` to show only important logs

2. **Filter by process**:
   ```bash
   /show process-name
   ```

3. **Hide verbose output**:
   ```bash
   /hide webpack
   /hide "GET /static"
   ```

## MCP Server Issues

### MCP server not starting

**Problem**: MCP server fails to start or connect.

**Solutions**:

1. **Check port availability**:
   ```bash
   lsof -i :7777
   ```

2. **Use different port**:
   ```bash
   brum -p 8888
   ```

3. **Disable MCP**:
   ```bash
   brum --no-mcp
   ```

### Can't connect from IDE

**Problem**: VS Code or other tools can't connect to MCP.

**Debugging**:

1. **Verify MCP is running**:
   ```bash
   curl http://localhost:7777/mcp/health
   ```

2. **Check firewall**:
   - Allow localhost connections
   - Disable firewall temporarily to test

3. **Use correct token**:
   - Check Settings tab for connection details
   - Ensure token matches in IDE config

## Performance Issues

### High CPU usage

**Problem**: Brummer uses excessive CPU.

**Solutions**:

1. **Limit concurrent processes**:
   - Run fewer processes simultaneously
   - Use process groups

2. **Check for runaway processes**:
   - Look for infinite loops in scripts
   - Monitor in Processes tab

3. **Reduce log volume**:
   - Disable verbose logging in scripts
   - Use log filtering

### High memory usage

**Problem**: Brummer or processes use too much memory.

**Solutions**:

1. **Monitor and restart**:
   - Watch memory in Processes tab
   - Restart high-memory processes with `r`

2. **Limit log retention**:
   ```bash
   BRUMMER_MAX_LOGS=1000 brum
   ```

3. **Use production builds**:
   - Development builds often use more memory
   - Run production builds when possible

## Platform-Specific Issues

### macOS Issues

**Gatekeeper blocking execution**:
```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine ~/.local/bin/brum
```

**Terminal app permissions**:
- Grant Terminal full disk access in System Preferences

### Windows Issues

**PowerShell execution policy**:
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

**Path separator issues**:
- Use forward slashes in scripts
- Or use Node.js path.join()

### Linux Issues

**TTY required error**:
```bash
# Run with TTY allocated
docker run -it ...

# Or use headless mode
brum --no-tui
```

## Getting Help

### Debug Information

Collect this information when reporting issues:

```bash
# Brummer version
brum --version

# System info
uname -a
node --version
npm --version

# Terminal info
echo $TERM
echo $SHELL

# Package.json scripts
cat package.json | grep -A 10 scripts
```

### Community Support

- GitHub Issues: [github.com/standardbeagle/brummer/issues](https://github.com/standardbeagle/brummer/issues)
- Documentation: [standardbeagle.github.io/brummer](https://standardbeagle.github.io/brummer)

### Logs Location

Brummer doesn't persist logs by default, but you can:

```bash
# Redirect output for debugging
brum 2>&1 | tee brummer-debug.log
```

Remember: Most issues have simple solutions. Check the basics first!