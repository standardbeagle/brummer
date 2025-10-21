# ROOT CAUSE FOUND: BubbleTea Event Loop Hangs After First Keyboard Input

## The Evidence

From `/tmp/brum-debug.log`:

```
UPDATE: Keyboard message received! 1
HANDLER: Processing key '1' with 9 handlers
HANDLER: ✅ GlobalNavigation CAN HANDLE this key!
HANDLER: ✅ GlobalNavigation returned a COMMAND
UPDATE: Returning 1 commands, batching now...
UPDATE: tea.Batch returned non-nil command
[END OF LOG - NO MORE MESSAGES]
```

## What This Tells Us

1. ✅ Keyboard message arrives at Update()
2. ✅ Handler processes it correctly
3. ✅ Command is created and returned
4. ❌ **Command NEVER executes** (no "COMMAND EXECUTED" log)
5. ❌ **No more Update() calls** (no TickMsg, no ViewSwitchedMsg)
6. ❌ **BubbleTea event loop is FROZEN**

## The Problem

After returning the command from Update(), BubbleTea should:
1. Execute the command (call the function)
2. Get the ViewSwitchedMsg result
3. Call Update() again with that message
4. Call View() to render

But **NONE of this happens**! The event loop completely stops.

## Why This Happens

BubbleTea's event loop is blocked trying to:
- Execute the command, OR
- Call View() to render, OR
- Read from stdin for next input

The most likely cause: **Something in the execution path is blocking on a channel read or other blocking operation.**

## Why ProgressiveModel Works

ProgressiveModel has a simple Update() that:
- Doesn't create complex commands
- Doesn't call View() rendering methods
- Returns immediately

The real Model:
- Creates commands that may block
- Has complex View() rendering
- Calls methods that might block (like `waitForUpdates()`)

## The Solution

We need to find what's blocking. Candidates:

1. **The command execution** - Does `SwitchToView()` command block?
   - No - it's simple, just returns a message

2. **View() method** - Does rendering block?
   - Possibly - if View() calls something that reads from a channel

3. **Component rendering** - Do components block in their View() methods?
   - Very likely - components may be reading from channels

4. **updateChan or other channels** - Is something blocked waiting for a message?
   - Possibly - if View() or components try to read from updateChan

## Next Steps

1. Add logging to View() method to see if it's called
2. Check if any component View() methods block
3. Look for channel reads in rendering code
4. Find violations of "NEVER block in View()" rule

## The Key Insight

**The problem is NOT in component initialization!**
**The problem is NOT in Update() logic!**
**The problem is in the RENDERING path (View() or component rendering)!**

Something in the View() call chain is blocking, which prevents BubbleTea from processing more events.
