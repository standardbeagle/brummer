---
sidebar_position: 1
---

# Your First Project with Brummer

A complete walkthrough to get you productive with Brummer in 10 minutes.

## What We'll Build

In this tutorial, we'll set up Brummer for a typical full-stack JavaScript project with:
- Frontend development server
- Backend API server  
- Database
- Tests
- Linting and formatting

By the end, you'll understand how Brummer transforms chaotic terminal management into an organized, efficient workflow.

## Prerequisites

- Node.js 16+ installed
- npm, yarn, or pnpm available
- Basic familiarity with terminal/command line

## Step 1: Install Brummer

First, let's install Brummer using the quick install method:

```bash
curl -sSL https://raw.githubusercontent.com/standardbeagle/brummer/main/quick-install.sh | bash
```

Verify installation:

```bash
brum --version
```

You should see the version number displayed.

## Step 2: Create a Sample Project

Let's create a simple full-stack project to demonstrate Brummer's capabilities:

```bash
mkdir my-fullstack-app
cd my-fullstack-app
npm init -y
```

### Add Dependencies

```bash
npm install express cors dotenv
npm install -D nodemon jest @types/node typescript tsx prettier eslint
```

### Create Project Structure

```bash
mkdir src tests
touch src/server.js src/app.js .env
```

## Step 3: Set Up npm Scripts

Edit your `package.json` to include these scripts:

```json title="package.json"
{
  "name": "my-fullstack-app",
  "scripts": {
    "dev:server": "nodemon src/server.js",
    "dev:frontend": "echo 'Frontend server running on http://localhost:3001' && sleep infinity",
    "test": "jest --watchAll",
    "test:once": "jest",
    "lint": "eslint src/**/*.js",
    "format": "prettier --write 'src/**/*.js'",
    "build": "echo 'Building project...' && sleep 3 && echo 'Build complete!'",
    "db:start": "echo 'Starting database on port 5432...' && sleep infinity",
    "clean": "rm -rf dist coverage"
  }
}
```

### Create a Simple Server

```javascript title="src/server.js"
const express = require('express');
const cors = require('cors');

const app = express();
const PORT = process.env.PORT || 3000;

app.use(cors());
app.use(express.json());

app.get('/api/health', (req, res) => {
  console.log('Health check requested');
  res.json({ status: 'ok', timestamp: new Date() });
});

app.get('/api/users', (req, res) => {
  console.log('Fetching users...');
  res.json([
    { id: 1, name: 'Alice' },
    { id: 2, name: 'Bob' }
  ]);
});

app.listen(PORT, () => {
  console.log(`Server running on http://localhost:${PORT}`);
  console.log('Press Ctrl+C to stop');
});
```

## Step 4: Start Brummer

Now for the magic moment. In your project directory, run:

```bash
brum
```

![Brummer First Launch](../img/screenshots/tutorial-first-launch.png)

## Step 5: Navigate the Interface

### Understanding the Layout

When Brummer starts, you'll see:

1. **Header**: Shows current view and navigation hints
2. **Main Area**: Lists available scripts from package.json
3. **Footer**: Shows keyboard shortcuts

### Key Navigation

Try these keys:
- `↑/↓` or `j/k`: Move between scripts
- `Tab`: Switch between views
- `?`: Show help
- `Enter`: Run selected script

## Step 6: Start Your First Process

1. Use arrow keys to highlight `dev:server`
2. Press `Enter` to start it

![Server Started](../img/screenshots/tutorial-server-started.png)

Notice:
- ✅ The script starts immediately
- ✅ You remain in the Scripts view
- ✅ Status shows in the header

## Step 7: Monitor Running Processes

Press `Tab` to switch to the **Processes** view:

![Processes View](../img/screenshots/tutorial-processes-view.png)

Here you can see:
- 🟢 Running status (green dot)
- Process ID and name
- CPU and memory usage
- Runtime duration

### Try Process Control

1. Select the running process with arrow keys
2. Press `s` to stop it
3. Press `r` to restart it

## Step 8: Start Multiple Processes

Go back to Scripts view (`Tab`) and start more services:

1. Start `db:start` (database simulation)
2. Start `dev:frontend` (frontend simulation)
3. Start `test` (test watcher)

Now switch to Processes view:

![Multiple Processes](../img/screenshots/tutorial-multiple-processes.png)

All your services running in one organized view!

## Step 9: View Logs

Press `Tab` to reach the **Logs** view:

![Logs View](../img/screenshots/tutorial-logs-view.png)

Features to try:
- Logs from all processes are consolidated
- Different processes have different colors
- Newest logs appear at the bottom
- Use `/` to search logs

### Filtering Logs

Type `/` and enter a search term:

```
/health
```

This filters logs to show only lines containing "health".

## Step 10: Error Detection

Let's trigger an error. Modify your server.js to include an error:

```javascript title="src/server.js (add this route)"
app.get('/api/error', (req, res) => {
  console.error('ERROR: Something went wrong!');
  throw new Error('Deliberate error for testing');
});
```

1. Save the file (nodemon will restart)
2. In another terminal: `curl http://localhost:3000/api/error`
3. Switch to **Errors** tab in Brummer

![Errors View](../img/screenshots/tutorial-errors-view.png)

Brummer automatically:
- ✅ Detects errors in logs
- ✅ Highlights them in red
- ✅ Groups them in Errors tab
- ✅ Shows full stack traces

## Step 11: URL Management

Switch to the **URLs** tab:

![URLs View](../img/screenshots/tutorial-urls-view.png)

Brummer automatically detected:
- Your server URL (http://localhost:3000)
- Frontend URL (http://localhost:3001)
- Any other URLs mentioned in logs

## Step 12: Advanced Features

### Running One-Time Scripts

From Scripts view, try running `build`:
1. Select `build` script
2. Press `Enter`
3. Watch it complete in Processes view

### Using Slash Commands

In Logs view, try these filters:

```bash
# Show only server logs
/show server

# Hide frontend logs
/hide frontend

# Show only errors
/show ERROR
```

### Quick Actions

- `Ctrl+R`: Restart all running processes
- `c`: Copy last error to clipboard (in Errors view)
- `p`: Toggle high-priority logs only

## Step 13: MCP Integration (Optional)

Brummer includes an MCP server for IDE integration:

1. Go to **Settings** tab
2. See the MCP server status
3. Note the port (default: 7777)

This allows tools like VS Code to:
- Execute scripts remotely
- Access logs
- Monitor process status

## Common Workflows

### Development Workflow

1. Start Brummer: `brum`
2. Start core services: database → backend → frontend
3. Start test watcher
4. Monitor all in Processes view
5. Check Errors tab periodically

### Debugging Workflow

1. Error appears in development
2. Switch to Errors tab (`Tab` navigation)
3. Read full error context
4. Press `c` to copy error
5. Fix issue
6. Press `r` to restart affected process

### Performance Monitoring

1. Run all services
2. Switch to Processes view
3. Monitor memory usage
4. Restart services if memory grows too high

## Tips for Success

### 1. Organize Your Scripts

Group related scripts with prefixes:
```json
{
  "scripts": {
    "dev:frontend": "...",
    "dev:backend": "...",
    "dev:db": "...",
    "test:unit": "...",
    "test:e2e": "..."
  }
}
```

### 2. Use Descriptive Names

Instead of:
```json
"start": "node server.js"
```

Use:
```json
"api:start": "node server.js"
```

### 3. Add Helpful Echo Statements

```json
"db:migrate": "echo 'Running database migrations...' && npm run migrate"
```

### 4. Create Composite Scripts

```json
"dev": "concurrently \"npm:dev:*\""
```

## Troubleshooting

### "Command not found: brum"

Add Brummer to your PATH:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

### Processes Not Starting

Check for:
- Port conflicts
- Missing dependencies
- Syntax errors in scripts

### High Memory Usage

- Restart individual processes with `r`
- Check for memory leaks in your code
- Use production builds when possible

## What's Next?

Congratulations! You've learned the fundamentals of Brummer. Here's what to explore next:

1. **[Migrate from Terminal Tabs](./migrate-from-terminal)** - Transition your existing workflow
2. **[React Development](../examples/react-development)** - React-specific workflows
3. **[MCP Integration](../mcp-integration/client-setup)** - Connect your IDE
4. **[Team Collaboration](./team-collaboration)** - Share configurations

## Summary

You've learned how to:
- ✅ Install and start Brummer
- ✅ Navigate between views
- ✅ Start and manage processes
- ✅ Monitor logs and errors
- ✅ Use filtering and search
- ✅ Control multiple services

Brummer is now ready to streamline your development workflow!