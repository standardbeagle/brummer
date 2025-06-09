---
sidebar_position: 1
---

# Navigation

Master the Brummer TUI with these navigation tips and keyboard shortcuts.

## Interface Overview

The Brummer interface consists of several views accessible via tabs:

```
🐝 Brummer - Development Buddy (2 processes, 1 running) 🌐
▶ 1.scripts | 2.processes | 3.logs | 4.errors | 5.urls | 6.settings
─────────────────────────────────────────────────────────────────
[Main Content Area]
[Help Bar]
```

## View Navigation

### Tab Switching
- **Tab** - Cycle forward through views
- **Shift+Tab** - Cycle backward through views (if supported)
- **Number Keys (1-6)** - Jump directly to a view

### Views Available
1. **Scripts** - Available npm/yarn/pnpm/bun scripts
2. **Processes** - Running and completed processes
3. **Logs** - Combined log output from all processes
4. **Errors** - Filtered view of errors only
5. **URLs** - Detected URLs from logs
6. **Settings** - Configuration and MCP setup

## Common Navigation Keys

### Movement
| Key | Action |
|-----|--------|
| **↑/k** | Move up |
| **↓/j** | Move down |
| **PgUp** | Page up |
| **PgDn** | Page down |
| **Home** | Go to top |
| **End** | Go to bottom |

### Selection and Actions
| Key | Action |
|-----|--------|
| **Enter** | Select/Execute |
| **Esc** | Go back/Cancel |
| **q** | Quit (with confirmation) |
| **?** | Show help |

## View-Specific Controls

### Scripts View
- **Enter** - Run selected script
- **/** - Search scripts
- **Esc** - Clear search

### Processes View
- **Enter** - View logs for process
- **s** - Stop running process
- **r** - Restart process
- **Ctrl+R** - Restart all running processes

### Logs View
- **/** - Search logs
- **p** - Toggle high-priority logs
- **c** - Copy latest error
- **Ctrl+L** - Clear logs

### Settings View
- **Enter** - Select option
- **Space** - Toggle checkbox
- **← →** - Navigate options

## Advanced Navigation

### Search Mode
When in search mode (**/**):
- Type to search
- **Enter** - Confirm search
- **Esc** - Cancel search
- **↑/↓** - Navigate results

### Multi-Select (Future Feature)
- **Space** - Toggle selection
- **Ctrl+A** - Select all
- **Ctrl+D** - Deselect all

## Tips and Tricks

### Quick Actions
1. **Quick Script Run**: Press number key for view, then Enter on script
2. **Emergency Stop**: `Tab` → `2` → `s` (go to processes, stop)
3. **Quick Error Check**: Press `4` to jump to errors view

### Efficient Workflow
1. Start scripts from Scripts view
2. Monitor in Processes view
3. Debug in Logs/Errors view
4. Open detected URLs from URLs view

### Vim-Style Navigation
For Vim users, these bindings work:
- **j/k** - Up/down movement
- **/** - Search (similar to Vim)
- **q** - Quit
- **?** - Help

## Customization

Currently, keyboard shortcuts are not customizable. This feature is planned for a future release.

## Accessibility

Brummer is designed to be keyboard-only accessible:
- All features accessible via keyboard
- Clear focus indicators
- Consistent navigation patterns
- Screen reader compatible (terminal permitting)

## Common Issues

### Keys Not Working
- Ensure terminal has focus
- Check if search mode is active (Esc to exit)
- Some terminals may intercept certain keys

### Navigation Feels Slow
- Use PgUp/PgDn for faster scrolling
- Jump directly to views with number keys
- Use search to find specific items quickly