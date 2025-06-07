---
sidebar_position: 4
---

# URL Detection

Brummer automatically detects and highlights URLs in your process output, making it easy to access your development servers and services.

## How It Works

Brummer scans process output for URL patterns and provides interactive features:

1. **Automatic detection** of various URL formats
2. **Interactive highlighting** in the TUI
3. **Quick actions** for opening URLs
4. **Status monitoring** for detected endpoints

## Detected URL Patterns

### Local Development

```
✅ http://localhost:3000
✅ http://127.0.0.1:8080
✅ http://0.0.0.0:4000
✅ https://localhost:3443
```

### Network URLs

```
✅ http://192.168.1.100:3000
✅ http://10.0.0.5:8080
✅ http://[::1]:3000 (IPv6)
✅ http://[fe80::1]:8080 (IPv6)
```

### Custom Domains

```
✅ http://myapp.local:3000
✅ https://dev.example.com:8443
✅ http://api.test:5000
```

### Special Protocols

```
✅ ws://localhost:3001 (WebSocket)
✅ wss://localhost:3443 (Secure WebSocket)
✅ mongodb://localhost:27017
✅ redis://localhost:6379
```

## Interactive Features

### URL List View

Press `u` to see all detected URLs:

```
┌─ Detected URLs ─────────────────────────────┐
│ Process    URL                     Status   │
│ ─────────────────────────────────────────── │
│ dev        http://localhost:3000   ✅ 200   │
│ api        http://localhost:4000   ✅ 200   │
│ admin      http://localhost:5000   ✅ 200   │
│ websocket  ws://localhost:3001     ✅ Open  │
│ database   mongodb://localhost:27017 ✅ Connected │
│                                             │
│ [Enter] Open  [C] Copy  [T] Test  [R] Refresh │
└─────────────────────────────────────────────┘
```

### Quick Actions

When a URL is highlighted:

- **Enter** - Open in default browser
- **c** - Copy to clipboard
- **t** - Test connectivity
- **i** - Show URL info
- **o** - Open with specific browser

### URL Status Monitoring

Brummer periodically checks URL availability:

```
┌─ URL Monitor ───────────────────────────────┐
│ http://localhost:3000                       │
│                                             │
│ Status: ✅ Online                           │
│ Response Time: 45ms                         │
│ Last Checked: 30s ago                       │
│                                             │
│ Headers:                                    │
│   Content-Type: text/html                   │
│   Server: webpack-dev-server                │
│                                             │
│ History:                                    │
│   10:23:45 - Started (took 3.2s)           │
│   10:23:48 - First request                  │
│   10:24:12 - Brief outage (500ms)          │
│   10:24:13 - Recovered                      │
└─────────────────────────────────────────────┘
```

## Smart Features

### Environment-Specific URLs

Brummer can detect environment-based URLs:

```javascript
// Detected from console output:
console.log(`Server running at ${process.env.API_URL || 'http://localhost:3000'}`);

// Brummer shows:
┌─ Environment URLs ──────────────────────────┐
│ Development: http://localhost:3000          │
│ Staging: https://staging.example.com        │
│ Production: https://api.example.com         │
└─────────────────────────────────────────────┘
```

### GraphQL Endpoints

Special handling for GraphQL URLs:

```
┌─ GraphQL Endpoint ──────────────────────────┐
│ http://localhost:4000/graphql               │
│                                             │
│ [G] Open GraphQL Playground                 │
│ [S] View Schema                             │
│ [Q] Test Query                              │
└─────────────────────────────────────────────┘
```

### API Documentation

Detect and link to API documentation:

```
┌─ API Documentation ─────────────────────────┐
│ Server: http://localhost:3000               │
│                                             │
│ Detected Docs:                              │
│   Swagger UI: /api-docs                     │
│   ReDoc: /redoc                            │
│   GraphQL: /graphql                        │
│                                             │
│ [D] Open documentation                      │
└─────────────────────────────────────────────┘
```

## Configuration

### URL Detection Settings

```yaml
# .brummer.yaml
url_detection:
  enabled: true
  
  # Check URL availability
  monitoring:
    enabled: true
    interval: 30s
    timeout: 5s
  
  # Custom patterns
  patterns:
    - pattern: "ngrok.io"
      label: "Tunnel"
    - pattern: "\.local:\d+"
      label: "Local Domain"
  
  # Ignore patterns
  ignore:
    - "**/node_modules/**"
    - "*.test.js"
```

### Browser Configuration

```yaml
url_detection:
  browser:
    default: "system"  # system, chrome, firefox, safari, edge
    
  # Custom browser commands
  browsers:
    chrome: "google-chrome"
    firefox: "firefox"
    safari: "open -a Safari"
```

### Auto-Open Settings

```yaml
url_detection:
  auto_open:
    enabled: false
    delay: 3s
    rules:
      - pattern: "localhost:3000"
        process: "dev"
        browser: "chrome"
```

## Advanced Features

### URL Grouping

Related URLs are grouped together:

```
┌─ URL Groups ────────────────────────────────┐
│ Frontend                                    │
│   Main: http://localhost:3000               │
│   Assets: http://localhost:3000/static      │
│   HMR: ws://localhost:3000/ws               │
│                                             │
│ Backend                                     │
│   API: http://localhost:4000/api            │
│   GraphQL: http://localhost:4000/graphql   │
│   Health: http://localhost:4000/health     │
│                                             │
│ Services                                    │
│   Database: postgresql://localhost:5432     │
│   Redis: redis://localhost:6379            │
│   Queue: http://localhost:4567             │
└─────────────────────────────────────────────┘
```

### URL History

Track URL availability over time:

```
┌─ URL History ───────────────────────────────┐
│ http://localhost:3000                       │
│                                             │
│ Uptime: 98.5% (last 24h)                   │
│                                             │
│ Timeline:                                   │
│ ████████████░░████████████████████████████ │
│ 00:00      06:00      12:00      18:00     │
│                                             │
│ Outages:                                    │
│   06:34-06:45 (11m) - Process crashed       │
│   14:22-14:23 (1m) - Restart               │
└─────────────────────────────────────────────┘
```

### Multi-URL Testing

Test multiple URLs simultaneously:

```
┌─ Batch URL Test ────────────────────────────┐
│ Testing 5 URLs...                           │
│                                             │
│ ✅ http://localhost:3000     (45ms)        │
│ ✅ http://localhost:4000     (23ms)        │
│ ❌ http://localhost:5000     (timeout)     │
│ ✅ ws://localhost:3001       (12ms)        │
│ ⏳ http://localhost:6000     (testing...)  │
│                                             │
│ Summary: 3/4 online (1 pending)             │
└─────────────────────────────────────────────┘
```

## Integration with Browser Extension

The browser extension enhances URL detection:

1. **Automatic navigation** to detected URLs
2. **Status badges** in browser toolbar
3. **Quick switching** between environments
4. **Request inspection** for detected endpoints

## MCP Integration

External tools can access URL information:

```javascript
// Get all detected URLs
const urls = await mcp.call('brummer.getUrls');

// Monitor specific URL
const status = await mcp.call('brummer.checkUrl', {
  url: 'http://localhost:3000'
});

// Subscribe to URL events
mcp.subscribe('url.detected', (event) => {
  console.log('New URL:', event.url);
});
```

## Troubleshooting

### URLs Not Detected

1. **Check output format** - URLs must be complete
2. **Verify patterns** - Custom patterns may be needed
3. **Check ignore rules** - URL might be filtered
4. **Enable debug mode** - See detection details

### False Positives

Exclude patterns that shouldn't be detected:

```yaml
url_detection:
  ignore_patterns:
    - "example.com"  # Documentation examples
    - "test.*"       # Test domains
    - ".*\\.md"      # Markdown files
```

### Connection Issues

If URL monitoring shows errors:

1. Check if service is actually running
2. Verify firewall settings
3. Test manually with curl
4. Check for port conflicts

## Best Practices

1. **Use standard ports** for better detection
2. **Include scheme** (http://) in output
3. **Configure monitoring** for critical URLs
4. **Group related URLs** for better organization
5. **Set up auto-open** for frequently used URLs
6. **Monitor URL health** to catch issues early
7. **Document special URLs** in your README

## Security Considerations

### HTTPS Detection

Brummer identifies secure connections:

```
┌─ Security Info ─────────────────────────────┐
│ https://localhost:3443                      │
│                                             │
│ 🔒 Secure Connection (TLS 1.3)              │
│ Certificate: Self-signed                    │
│ Valid Until: 2025-12-31                     │
│                                             │
│ ⚠️ Warning: Self-signed certificate         │
└─────────────────────────────────────────────┘
```

### Internal URLs

Be cautious with internal URLs:

```yaml
url_detection:
  security:
    warn_internal: true
    block_external_access: true
    allowed_networks:
      - "192.168.1.0/24"
      - "10.0.0.0/8"
```