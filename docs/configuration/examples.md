# Configuration Examples

## Multi-Project Development Workflow

### Scenario: Frontend + Backend + Database

**Step 1: Set up individual instances**
```bash
# Terminal 1: Frontend (React/Vite)
cd frontend/
brum dev                 # Starts on port 7777

# Terminal 2: Backend (Node.js API)  
cd backend/
brum -p 7778 dev        # Starts on port 7778

# Terminal 3: Database utilities
cd database/
brum -p 7779 migrate    # Starts on port 7779
```

**Step 2: Configure MCP hub for coordination**
```json
// Claude Desktop config
{
  "servers": {
    "my-project-hub": {
      "command": "brum",
      "args": ["--mcp"]
    }
  }
}
```

**Step 3: Use hub tools to coordinate**
```bash
# List all running instances
instances_list

# Connect to frontend instance
instances_connect frontend-abc123

# Run frontend-specific commands
hub_scripts_list        # Lists frontend package.json scripts
hub_logs_stream         # Streams frontend logs
hub_browser_screenshot  # Takes screenshot of frontend

# Switch to backend
instances_connect backend-def456
hub_logs_search "error" # Search backend logs for errors
```

## Proxy Configuration Examples

### Automatic URL Detection
Brummer automatically detects URLs in process logs:

```bash
# Start development server
brum dev

# Logs show: "Local: http://localhost:3000"
# Brummer automatically:
# 1. Detects the URL
# 2. Creates reverse proxy: http://localhost:20888
# 3. Makes it shareable across network
```

### Manual Proxy Configuration
```bash
# Start with specific URL proxying
brum --proxy-url http://localhost:3000 dev

# Use traditional HTTP proxy mode
brum --proxy-mode full --proxy-port 8888

# Configure browser to use proxy:
# HTTP Proxy: localhost:8888
# PAC URL: http://localhost:8888/proxy.pac
```

### Multiple URL Handling
```bash
# Start multiple services
brum "npm run dev & npm run api & npm run docs"

# Brummer detects and proxies:
# Frontend: http://localhost:3000 → http://localhost:20888
# API:      http://localhost:3001 → http://localhost:20889  
# Docs:     http://localhost:3002 → http://localhost:20890
```

## Browser Automation Examples

### Screenshot Workflow
```javascript
// Single instance
browser_screenshot({
  "format": "png",
  "fullPage": true,
  "quality": 90
})

// Hub mode (routes to connected instance)
hub_browser_screenshot({
  "format": "jpeg", 
  "selector": "#main-content",
  "quality": 85
})
```

### JavaScript Testing
```javascript
// Execute JavaScript in browser
repl_execute({
  "code": "document.title = 'Test'; return document.title;"
})

// Hub mode with session
instances_connect("frontend-instance")
hub_repl_execute({
  "code": "console.log('Testing frontend'); return window.location.href;"
})
```

## Advanced Configuration

### Multi-Instance Hub Setup
```toml
# ~/.brum.toml
[instances]
discovery_dir = "~/.brum/instances"
cleanup_interval = "1m"
stale_timeout = "5m"

[hub]
health_check_interval = "30s" 
max_retry_attempts = 3
retry_backoff = "exponential"

[proxy]
mode = "reverse"
base_port = 20888
enable_telemetry = true
```

### Project-Specific Configuration
```toml
# project/.brum.toml
[process]
preferred_package_manager = "pnpm"

[proxy]
mode = "reverse"
proxy_url = "http://localhost:3000"

[mcp]
port = 7777
enable_browser_tools = true
```

## Integration with External Tools

### VS Code Integration
```json
// .vscode/settings.json
{
  "mcp.servers": {
    "brummer": {
      "command": "brum",
      "args": ["--no-tui", "--port", "7777"]
    }
  }
}
```

### CI/CD Integration
```bash
# Headless operation for CI
brum --no-tui --no-proxy test

# Export test results
logs_search "test.*" > test-results.log

# Health check endpoint
curl http://localhost:7777/mcp -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

### Docker Integration
```dockerfile
# Dockerfile
FROM node:18
COPY . /app
WORKDIR /app
RUN npm install && npm install -g brum
EXPOSE 7777 20888-20899
CMD ["brum", "--no-tui", "--port", "7777"]
```

## Performance Optimization

### Log Management
```toml
# .brum.toml
[logs]
max_entries = 10000          # Limit memory usage
max_line_length = 2048       # Truncate long lines
enable_url_detection = true  # Auto-detect URLs
```

### Resource Limits
```bash
# Limit process resources
ulimit -n 1024              # File descriptor limit
ulimit -u 256               # Process limit

# Monitor resource usage
logs_search "memory|cpu"    # Search for resource logs
```