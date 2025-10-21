# Definitive Keyboard Test

## Goal
Determine if keyboard input works at all, and if so, why it works in some cases but not others.

## Test 1: Known Working - Minimal BubbleTea

```bash
cd /home/beagle/work/brummer
go run ./test_minimal_bubbletea.go
```

**Expected**: Should show keys pressed in real-time. Press 'q' to quit.

**If this works**: BubbleTea itself is fine, problem is in brum initialization.

## Test 2: Real Model Test

```bash
cd /home/beagle/work/brummer
go run ./test_with_real_model.go
```

**Expected**: Should receive keyboard input (won't show visually but will log).
Kill with Ctrl+C or from another terminal.

**If this works**: The Model is fine, problem is in main.go initialization.

## Test 3: Actual Brum Binary

```bash
cd /tmp
rm -rf test_kb && mkdir test_kb && cd test_kb
echo '{"name":"test","scripts":{"dev":"sleep 30"}}' > package.json
rm -f /tmp/brum-debug.log
/home/beagle/work/brummer/brum
```

Press keys: 1, 2, 3, Tab
Kill from another terminal: `pkill -9 brum`

Check log:
```bash
cat /tmp/brum-debug.log | grep -E "Keyboard|HANDLER|VIEWSWITCH"
```

**If this DOESN'T work**: Something in main.go is breaking keyboard input.

## Analysis

Compare what's different between the working and non-working cases:
- How tea.NewProgram() is called
- What options are passed
- Any initialization before tea.NewProgram()
- Any goroutines or signal handlers

The key question: **Which tests actually receive keyboard input?**
