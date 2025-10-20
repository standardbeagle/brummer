# Keyboard Handler Testing Instructions

## The Problem
Keyboard navigation is broken. We know:
- ✅ Keyboard messages ARE received by Update()
- ❓ Unknown if HandleKey() is being called
- ❓ Unknown if GlobalNavigationHandler is executing

## Testing Approach

Since 'q' doesn't work to quit, you'll need to use **Ctrl+C** to exit brummer.

### Steps:

1. **Clear the debug log:**
   ```bash
   rm -f /tmp/brum-debug.log
   ```

2. **Start brummer:**
   ```bash
   cd /tmp
   mkdir -p test_keyboard
   cd test_keyboard
   echo '{"name":"test","scripts":{"dev":"sleep 30"}}' > package.json
   /home/beagle/work/brummer/brum
   ```

3. **Test keyboard (while brummer is running):**
   - Press `1` (three times)
   - Press `2` (three times)
   - Press `Tab` (three times)
   - Press `Ctrl+C` to quit

4. **Check the debug log:**
   ```bash
   cat /tmp/brum-debug.log
   ```

## What to Look For

### Scenario 1: HandleKey() is NOT being called
```
UPDATE: Keyboard message received! 1
UPDATE: Keyboard message received! 1
UPDATE: Keyboard message received! 1
```
**No "HANDLER:" messages** → Problem is in model.go Update() routing

### Scenario 2: HandleKey() called but no handler matches
```
UPDATE: Keyboard message received! 1
HANDLER: Processing key '1' with 5 handlers
HANDLER: [0] Checking GlobalNavigationHandler (priority 100)
HANDLER: [1] Checking AICoderKeyHandler (priority 250)
...
HANDLER: ❌ NO HANDLER MATCHED this key
```
**Has "HANDLER:" but NO "CAN HANDLE"** → Problem is in CanHandle() logic

### Scenario 3: Handler matches but Handle() NOT called
```
UPDATE: Keyboard message received! 1
HANDLER: Processing key '1' with 5 handlers
HANDLER: [0] Checking GlobalNavigationHandler (priority 100)
HANDLER: ✅ GlobalNavigationHandler CAN HANDLE this key!
```
**No "GLOBALNAV: Handle called"** → Problem between CanHandle() and Handle()

### Scenario 4: Handle() called but no view switch
```
UPDATE: Keyboard message received! 1
HANDLER: ✅ GlobalNavigationHandler CAN HANDLE this key!
GLOBALNAV: Handle called for key: 1 (Type: 6)
GLOBALNAV: Rune key 1 received
```
**No "Number key 1 → view"** → Problem in GetViewForNumberKey() or Handle() logic

### Scenario 5: View switch attempted (GOOD!)
```
UPDATE: Keyboard message received! 1
HANDLER: ✅ GlobalNavigationHandler CAN HANDLE this key!
GLOBALNAV: Handle called for key: 1 (Type: 6)
GLOBALNAV: Rune key 1 received
GLOBALNAV: Number key 1 → view ViewProcesses
```
**✅ Everything works!** → Problem is in viewManager.SwitchToView() or visual update

## Quick Analysis Commands

After quitting brummer, run these to quickly analyze:

```bash
# Count keyboard events received
grep -c "UPDATE: Keyboard message" /tmp/brum-debug.log

# Check if handler was called
grep "HANDLER: Processing key" /tmp/brum-debug.log | head -5

# Check if any handler matched
grep "CAN HANDLE" /tmp/brum-debug.log

# Check if GlobalNav executed
grep "GLOBALNAV:" /tmp/brum-debug.log

# Check view switch attempts
grep "GLOBALNAV: Number key" /tmp/brum-debug.log
```

## Expected Output Format

The log entries will look like this:

```
MODEL: NewModel completed successfully
DEBUG: About to call tea.NewProgram...
DEBUG: tea.NewProgram returned, about to call Run()...
DEBUG: Calling p.Run()...
UPDATE: WindowSizeMsg received
UPDATE: Message type cursor.initialBlinkMsg
UPDATE: Message type tui.TickMsg
UPDATE: Keyboard message received! 1
HANDLER: Processing key '1' with 5 handlers
HANDLER: Context - View=ViewProcesses, Context=
HANDLER: [0] Checking GlobalNavigationHandler (priority 100)
HANDLER: ✅ GlobalNavigationHandler CAN HANDLE this key!
HANDLER: ✅ GlobalNavigationHandler returned a COMMAND
GLOBALNAV: Handle called for key: 1 (Type: 6)
GLOBALNAV: Rune key 1 received
GLOBALNAV: Number key 1 → view ViewProcesses
```

The presence or absence of specific lines will tell us EXACTLY where the problem is.
