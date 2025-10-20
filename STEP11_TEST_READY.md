# Step 11 Test: AICoderService - The Suspected Culprit

## What We're Testing

**Step 11** adds the AICoderService exactly as it appears in the real Model (lines 727-804):

1. **AICoderService creation** - Creates its own EventBus
2. **5 event subscriptions** that push to `updateChan`:
   - `aicoder.session.created`
   - `aicoder.session.updated`
   - `aicoder.session.ended`
   - `aicoder.pty.output`
   - `aicoder.error`
3. **AICodersViewComponent** - The view component

## Why This is Likely the Culprit

1. **Violates Redux/Elm Architecture**:
   - Creates background goroutines (EventBus workers)
   - State changes happen outside Update() method
   - Event subscriptions push to `updateChan` from worker threads

2. **Goroutines Interfere with stdin**:
   - BubbleTea needs exclusive access to stdin for keyboard input
   - Background goroutines may be blocking or competing for stdin
   - EventBus worker pool spawns goroutines during initialization

3. **All Previous Steps Worked**:
   - Steps 1-9 all had working keyboard
   - No other component creates background workers
   - This is the first component that violates the command pattern

## How to Test

```bash
./test_step11_aicoder.sh
```

The script will:
1. Run brum with ProgressiveModel Step 11
2. Ask you to press keys: 1 1 1 2 2 2 Tab Tab q
3. Check the debug log for keyboard messages
4. Report whether keyboard worked or broke

## Expected Result

**If keyboard BREAKS** (no messages in log):
- ✅ We've found the culprit!
- Next step: Refactor AICoderService to use `tea.Cmd` pattern
- Remove EventBus and background goroutines
- Make all operations return commands instead

**If keyboard WORKS** (messages appear):
- ❓ AICoderService is NOT the culprit
- Need to test remaining components:
  - SlashCommandHandler
  - AICoderKeyHandler
  - updateProcessList() call
  - Or something in how real Model's Update() differs

## The Fix (if AICoderService is the culprit)

Refactor AICoderService to:
1. **Remove `events.NewEventBus()`** - don't create worker pool
2. **Remove event subscriptions** in constructor
3. **Make all methods return `tea.Cmd`** instead of publishing events
4. **Use BubbleTea's message pattern** for async operations

This aligns with:
- Your existing Redux/Elm architecture
- The Goroutine to Command Pattern conversion you completed
- The TUI Deadlock Prevention Guide

## Ready to Test!

Run the test script now to confirm our hypothesis.
