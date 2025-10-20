# Keyboard Input Testing Status

## Current State

We've been systematically adding components to ProgressiveModel to identify which one breaks keyboard input.

### ✅ Components Tested and WORKING:

**Step 1-7**: All basic components
- ProcessManager, LogStore, EventBus
- ViewManager
- UpdateChan, MasterData, StateManager
- MessageRouter, KeyHandlerRegistry
- ScriptSelector
- All view components (ProcessesView, LogsView, ErrorsView, URLsView, WebView, CommandWindowView)
- Init commands (textinput.Blink, tickCmd)

**Step 8**: InputComponent + NavigationComponent
- ✅ **TESTED**: Keyboard still works!

**Step 9**: Layout components (READY TO TEST)
- tabsComponent
- contentLayout
- **Status**: Built successfully, ready for manual testing

## Test Script

Run this to test Step 9:
```bash
./test_step8.sh  # Will show "Step 9" in the display
```

Or manually:
```bash
cd /tmp/test_step8
rm -f /tmp/brum-debug.log
./brum
# Press: 1 1 1 2 2 2 Tab Tab q
# Check: grep "PROGRESSIVE: Keyboard" /tmp/brum-debug.log
```

## Next Steps (if Step 9 works)

**Step 10**: MCP and Settings views
- settingsView
- mcpView
- mcpDebugView

**Step 11**: AI Coder Service (MOST LIKELY CULPRIT)
- AICoderService (creates its own EventBus)
- aicodersView
- Event subscriptions that push to updateChan:
  - `aicoder.session.created`
  - `aicoder.session.updated`
  - `aicoder.session.ended`
  - `aicoder.pty.output`
  - `aicoder.error`

**Step 12**: Additional keyboard handlers
- AICoderKeyHandler
- SlashCommandHandler

**Step 13**: Additional initialization
- updateProcessList() call

## Why AI Coder Service is Suspicious

1. Creates its own EventBus (`events.NewEventBus()`) which spawns worker goroutines
2. Sets up 5 event subscriptions that push messages to `updateChan`
3. These subscriptions are closures that capture `m.updateChan`
4. Event handlers run in separate goroutines (EventBus worker pool)
5. Could be interfering with stdin/BubbleTea's input handling

## To Confirm Real Model is Broken

Switch to real Model and test:
```bash
# Edit cmd/brum/main.go - uncomment real Model line
make build
./test_real_model_confirm.sh
```

Expected: NO keyboard messages in debug log.
