# Manual Keyboard Test (No Script)

## The Issue
When running brummer via background script, keyboard input isn't captured.
We need to test it directly in the foreground.

## Test Steps

### 1. Open TWO terminal windows/tabs

**Terminal 1 (for running brummer):**
```bash
cd /tmp
rm -rf test_kb && mkdir test_kb && cd test_kb
echo '{"name":"test","scripts":{"dev":"sleep 30"}}' > package.json
rm -f /tmp/brum-debug.log
/home/beagle/work/brummer/brum
```

**While brummer is running, press these keys:**
- `1` (three times)
- `2` (three times)
- `Tab` (three times)

**Terminal 2 (while brummer is still running):**
```bash
# Monitor the debug log in real-time
tail -f /tmp/brum-debug.log
```

### 2. Kill brummer from Terminal 2

Once you've pressed the keys and seen the log output, kill it:

```bash
# Find the PID
ps aux | grep brum | grep -v grep

# Kill it (replace XXXX with actual PID)
kill -9 XXXX
```

### 3. Check the final log

```bash
cat /tmp/brum-debug.log
```

## What We're Looking For

If keyboard messages ARE received, you should see:
```
UPDATE: Keyboard message received! 1
HANDLER: Processing key '1' with 9 handlers
GLOBALNAV: Handle called for key: 1
GLOBALNAV: Number key 1 → view processes
HANDLER: ✅ GlobalNavigation returned a COMMAND
UPDATE: Returning 1 commands
VIEWSWITCH: ViewSwitchedMsg received! logs → processes
```

The critical question: **Do you see "VIEWSWITCH: ViewSwitchedMsg received!"?**

- ✅ YES → The command IS executing, problem is in view rendering
- ❌ NO → The command is NOT executing, BubbleTea isn't running the command

## Alternative: Use pkill

If you can't get the PID easily:
```bash
# From Terminal 2, after pressing keys:
pkill -9 brum
```

Then check the log:
```bash
cat /tmp/brum-debug.log
```
