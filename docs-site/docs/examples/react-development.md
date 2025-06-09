---
sidebar_position: 1
---

# React Development with Brummer

Learn how Brummer transforms your React development workflow by providing intelligent process management, error detection, and real-time monitoring.

## Overview

Developing React applications often involves juggling multiple processes:
- Development server with hot reload
- Test runner in watch mode
- Linting and type checking
- Backend API server
- Build processes

Brummer consolidates all these into a single, organized interface.

## Setting Up Your React Project

### 1. Project Structure

```json title="package.json"
{
  "name": "my-react-app",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview",
    "test": "vitest",
    "test:ui": "vitest --ui",
    "lint": "eslint src --ext ts,tsx --report-unused-disable-directives",
    "type-check": "tsc --noEmit",
    "format": "prettier --write 'src/**/*.{ts,tsx,css}'",
    "api": "json-server --watch db.json --port 3001"
  }
}
```

### 2. Starting Brummer

```bash
cd my-react-app
brum
```

![Brummer React Scripts](../img/screenshots/react-scripts.png)

## Common Development Workflows

### Running Development Server

1. Navigate to the `dev` script using arrow keys
2. Press `Enter` to start the Vite dev server
3. Brummer automatically detects and displays the local URL

![React Dev Server Running](../img/screenshots/react-dev-server.png)

**Key Benefits:**
- ✅ Instant feedback on compilation errors
- ✅ Hot Module Replacement (HMR) status visible
- ✅ Build time monitoring
- ✅ Memory usage tracking

### Running Tests in Watch Mode

Start your test runner alongside development:

1. Press `Tab` to go back to Scripts view
2. Select `test` script
3. Press `Enter` to run tests in watch mode

![React Tests Running](../img/screenshots/react-tests.png)

**What Brummer Shows You:**
- Test suite progress
- Failed test details with stack traces
- Test execution time
- Coverage summary (if configured)

### Monitoring Multiple Processes

The real power comes from running multiple processes:

```bash
# Traditional approach (multiple terminals)
# Terminal 1: npm run dev
# Terminal 2: npm run test
# Terminal 3: npm run api
# Terminal 4: npm run type-check --watch

# With Brummer (single interface)
# Just select and run each script!
```

![Multiple Processes](../img/screenshots/react-multiple-processes.png)

## Error Detection and Debugging

### TypeScript Errors

Brummer intelligently parses and highlights TypeScript errors:

![TypeScript Error Detection](../img/screenshots/react-typescript-error.png)

**Features:**
- 🔴 Errors appear in dedicated Errors tab
- 📍 File paths and line numbers are clickable
- 🔍 Full error context preserved
- 📋 Copy error with `c` key

### Build Errors

When builds fail, Brummer helps you quickly identify issues:

```typescript
// Example error in component
import { useState } from 'react'

function App() {
  const [count, setCount] = useState(0)
  
  // Error: 'countt' is not defined
  return <div>{countt}</div>
}
```

Brummer shows:
- Compilation error with exact location
- Suggested fixes (if available)
- Build timing information

### Test Failures

Test failures are prominently displayed:

![Test Failure](../img/screenshots/react-test-failure.png)

## Advanced Features

### URL Detection

Brummer automatically detects and collects URLs from your processes:

- Development server URLs
- API endpoints
- Documentation links
- Preview URLs

Access them quickly in the URLs tab (`Tab` to navigate).

### Log Filtering

Use slash commands to filter logs:

```bash
# Show only error logs
/show error

# Hide verbose webpack output
/hide webpack

# Show only test-related logs
/show test
```

### Process Management

Control your development environment efficiently:

| Key | Action | Use Case |
|-----|--------|----------|
| `s` | Stop process | Stop dev server to free port |
| `r` | Restart process | Restart after config change |
| `Ctrl+R` | Restart all | Fresh start after major changes |

## Performance Monitoring

### Build Performance

Monitor build times to identify bottlenecks:

![Build Performance](../img/screenshots/react-build-perf.png)

Look for:
- Initial build time
- Rebuild time after changes
- Memory usage trends
- Bundle size warnings

### Development Tips

1. **Start Core Services First**
   ```
   dev → api → test (in that order)
   ```

2. **Use the Errors Tab**
   - Check Errors tab (`Tab` navigation) for quick error summary
   - Compilation errors appear immediately
   - Test failures are grouped by suite

3. **Monitor Memory Usage**
   - Watch for memory leaks in dev server
   - Restart processes if memory grows too high

## Integration with React DevTools

While Brummer handles process management, it complements React DevTools:

1. Run your React app through Brummer
2. Open browser DevTools as usual
3. Use Brummer for logs and error tracking
4. Use React DevTools for component inspection

## Example: Full-Stack React Development

Here's a complete workflow for full-stack React development:

```json title="package.json"
{
  "scripts": {
    "dev": "concurrently \"npm:dev:*\"",
    "dev:frontend": "vite",
    "dev:backend": "nodemon server.js",
    "dev:db": "json-server --watch db.json",
    "test": "vitest",
    "test:e2e": "playwright test",
    "lint": "eslint .",
    "build": "vite build",
    "preview": "vite preview"
  }
}
```

With Brummer:
1. Run `brum`
2. Select `dev` to start all services
3. Monitor all processes in one view
4. Quickly identify which service has errors
5. Restart individual services as needed

## Troubleshooting Common Issues

### Port Already in Use

If you see "Port 3000 is already in use":
1. Go to Processes tab
2. Find the process using the port
3. Press `s` to stop it
4. Restart the script

### Hot Reload Not Working

1. Check the Logs tab for WebSocket errors
2. Restart the dev server with `r`
3. Clear browser cache if needed

### Memory Issues

If dev server becomes slow:
1. Check process memory in Processes tab
2. Restart the process with `r`
3. Consider increasing Node.js memory limit

## Best Practices

1. **Organize Scripts Logically**
   - Group related scripts with prefixes
   - Use clear, descriptive names
   - Document complex scripts

2. **Use Brummer's MCP Server**
   - Integrate with VS Code for seamless workflow
   - Access logs from your IDE
   - Execute scripts without leaving your editor

3. **Leverage Keyboard Shortcuts**
   - Learn the navigation keys
   - Use filtering for large logs
   - Master process control keys

## Next Steps

- Explore [Next.js Full-Stack Development](./nextjs-fullstack)
- Learn about [Monorepo Workflows](./monorepo-workflows)
- Set up [IDE Integration](../mcp-integration/client-setup)