---
sidebar_position: 2
---

# Getting Started

Get up and running with Brummer in less than 5 minutes.

## Prerequisites

Before installing Brummer, ensure you have:

- **Go 1.21 or later** installed ([Download Go](https://golang.org/dl/))
- A project with `package.json` file
- One of the supported package managers: npm, yarn, pnpm, or bun

## Quick Install

The fastest way to install Brummer:

```bash
curl -sSL https://raw.githubusercontent.com/beagle/brummer/main/quick-install.sh | bash
```

Or using wget:

```bash
wget -qO- https://raw.githubusercontent.com/beagle/brummer/main/quick-install.sh | bash
```

## First Run

1. Navigate to your project directory:
   ```bash
   cd my-project
   ```

2. Start Brummer:
   ```bash
   brum
   ```

3. You'll see the interactive TUI with your available scripts:
   
   ![Brummer TUI Screenshot](./img/brummer-tui.png)

## Basic Navigation

- **Tab**: Switch between views (Scripts, Processes, Logs, etc.)
- **↑/↓** or **j/k**: Navigate items
- **Enter**: Select/execute
- **?**: Show help
- **q** or **Ctrl+C**: Quit

## Running Your First Script

1. Use arrow keys to select a script
2. Press **Enter** to run it
3. Switch to the **Processes** tab to see it running
4. View logs in the **Logs** tab

## What's Next?

- Learn about [Advanced Installation Options](./installation)
- Explore the [User Guide](./user-guide/navigation)
- Set up [MCP Integration](./mcp-integration/overview) with your IDE
