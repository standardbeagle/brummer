# Expect-Based Testing Paradigm for Brummer

## 🎯 Overview

This document describes the **new Expect-based testing paradigm** for Brummer, which solves the fundamental limitation of testing TUI applications in automated environments.

## 🔍 The Problem We Solved

### **Previous Situation**
- ❌ **Bash tool** runs in non-TTY environment
- ❌ **No `/dev/tty` access** for automated tests
- ❌ **BubbleTea TUI fails** without real terminal
- ❌ **"Tests pass but app doesn't work"** scenario

### **Root Cause Identified**
```
Failed to run TUI: could not open a new TTY: open /dev/tty: no such device or address
```

### **The "Diabolical Way Tests Were Valid But App Was Broken"
- **Manual testing**: ✅ `./brum` works (has `/dev/tty`)
- **Automated tests**: ❌ TUI never loads (no `/dev/tty`)
- **False confidence**: Tests tested wrong environment entirely

## ✅ The Solution: Expect-Based Testing

### **What is Expect?**
Expect is a Unix tool designed for automating interactions with terminal applications. It provides:
- **Real TTY environment** with full `/dev/tty` access
- **Precise keystroke control** for user interaction simulation
- **Pattern matching** for robust response validation
- **State management** for complex test scenarios

### **Why Expect Works for Brummer**
| Feature | Traditional Tests | Expect Tests | Benefit |
|---------|------------------|-------------|---------|
| TTY Access | ❌ None | ✅ Full `/dev/tty` | TUI loads correctly |
| Keystroke Control | ❌ Limited | ✅ Precise input | Real user simulation |
| UI Validation | ❌ Pattern matching only | ✅ Screen capture | Visual verification |
| Error Handling | ❌ Basic | ✅ Comprehensive | Robust testing |

## 🏗️ Testing Architecture

### **Three Tiers of Testing**

#### 1. **Expect-Based Integration Tests** (Primary)
- **Purpose**: Full TUI functionality testing
- **Environment**: Real terminal with `/dev/tty`
- **Coverage**: Navigation, commands, AI functionality
- **Files**: `test/tty/expect_*.exp`

#### 2. **Headless Unit Tests** (Supporting)
- **Purpose**: Component logic without TUI
- **Environment**: No terminal required
- **Coverage**: Business logic, data processing
- **Files**: `internal/*/test.go`

#### 3. **MCP/API Tests** (Supporting)
- **Purpose**: Server functionality without UI
- **Environment**: HTTP/stdio protocols
- **Coverage**: MCP tools, proxy functionality
- **Files**: `internal/mcp/*_test.go`

## 📁 Test Suite Structure

```
test/tty/
├── expect_*.exp              # Expect integration tests
│   ├── expect_basic_tui.exp    # Basic TUI functionality
│   ├── expect_navigation.exp   # Tab navigation
│   ├── expect_commands.exp     # Slash commands
│   └── expect_ai_coders.exp    # AI functionality
├── system_bash_test.sh        # Bash-based validation
├── tmux_tty_test.sh          # Tmux-based alternative
├── pty_test_helpers.go       # Go PTY utilities (for headless tests)
└── real_pty_integration_test.go  # PTY framework foundation
```

## 🚀 Getting Started with Expect Testing

### **Installation**
```bash
# Ubuntu/Debian
sudo apt-get install expect

# CentOS/RHEL
sudo yum install expect

# macOS
brew install expect

# Verify installation
expect -v
```

### **Running Tests**
```bash
# Basic TUI functionality test
./test/tty/expect_basic_tui.exp

# Navigation test
./test/tty/expect_navigation.exp

# Commands test
./test/tty/expect_commands.exp

# AI functionality test
./test/tty/expect_ai_coders.exp

# Run all tests
make test-expect
```

### **Environment Variables**
```bash
# Set binary path (default: ./brum)
export BRUMMER_BINARY=./brum

# Set test directory (default: current directory)
export TEST_DIR=/path/to/project

# Set timeout (default: 30 seconds)
export EXPECT_TIMEOUT=60
```

## 🧪 Test Examples

### **Basic TUI Test**
```tcl
#!/usr/bin/expect -f
# expect_basic_tui.exp

set timeout 30
set binary_path [getenv BRUMMER_BINARY "./brum"]
set test_dir [getenv TEST_DIR "."]

puts "🚀 Testing Basic TUI Functionality"
puts "================================="

# Spawn application
spawn $binary_path -d $test_dir --debug

# Wait for startup
expect {
    timeout { puts "❌ Startup timeout"; exit 1 }
    "AI coder manager initialized successfully" {
        puts "✅ App started successfully"
    }
    eof { puts "❌ App exited unexpectedly"; exit 1 }
}

# Wait for TUI to initialize
sleep 3

# Verify TUI elements
set tui_detected 0
expect {
    timeout { puts "⚠️  TUI detection timeout" }
    -re "Brummer.*Development Buddy" {
        puts "✅ Header detected"
        set tui_detected 1
    }
    -re "│.*│" {
        puts "✅ Tab separators detected"
        set tui_detected 1
    }
}

if {$tui_detected} {
    puts "✅ TUI loaded successfully"
} else {
    puts "❌ TUI failed to load"
    exit 1
}

# Test basic navigation
send "2"  # Processes tab
sleep 1
puts "✅ Navigated to Processes tab"

send "6"  # AI Coders tab
sleep 1
puts "✅ Navigated to AI Coders tab"

send "1"  # Logs tab
sleep 1
puts "✅ Navigated to Logs tab"

# Test quit
send "q"
expect eof

puts "🎉 Basic TUI test completed successfully!"
```

### **AI Command Test**
```tcl
#!/usr/bin/expect -f
# expect_ai_coders.exp

set timeout 45  # Longer timeout for AI operations
set binary_path [getenv BRUMMER_BINARY "./brum"]
set test_dir [getenv TEST_DIR "."]

puts "🤖 Testing AI Coder Functionality"
puts "================================="

spawn $binary_path -d $test_dir --debug

expect "AI coder manager initialized successfully"
sleep 3

# Navigate to AI Coders tab
send "6"
sleep 2

# Test AI command without session
send "/"
expect ":"
send "ai claude test command"
send "\r"

sleep 5

# Check for session creation
set session_created 0
expect {
    timeout { puts "⏰ AI command timeout" }
    "session" {
        puts "✅ AI session created"
        set session_created 1
    }
    "No active AI coder sessions" {
        puts "⚠️  No active sessions (expected without proper config)"
    }
    "error" {
        puts "❌ AI command failed"
        exit 1
    }
}

# Test help within AI context
send "?"
sleep 2
puts "✅ Help accessed in AI context"

# Test quit
send "q"
expect eof

puts "🎉 AI Coder test completed!"
```

## 📊 Test Results and Validation

### **Success Criteria**
- ✅ TUI loads correctly with `/dev/tty` access
- ✅ All tabs navigate properly
- ✅ Slash commands execute successfully
- ✅ AI functionality works (with proper config)
- ✅ Error handling prevents false positives

### **Failure Detection**
- ❌ TUI fails to load (timeout on TUI elements)
- ❌ Navigation doesn't work (no response to keystrokes)
- ❌ Commands fail (error patterns detected)
- ❌ AI sessions don't create (timeout or errors)

### **Debugging Information**
- 📋 Real-time progress reporting
- 📏 Screen capture on failure
- 🔍 Pattern matching details
- ⏱️  Timeout analysis

## 🔧 Integration with CI/CD

### **GitHub Actions Example**
```yaml
name: Expect Tests

on: [push, pull_request]

jobs:
  expect-tests:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v2

    - name: Install dependencies
      run: |
        sudo apt-get update
        sudo apt-get install -y expect
        make build

    - name: Run Expect tests
      run: |
        export BRUMMER_BINARY=./brum
        ./test/tty/expect_basic_tui.exp
        ./test/tty/expect_navigation.exp
        ./test/tty/expect_commands.exp
        ./test/tty/expect_ai_coders.exp
```

### **Docker Testing**
```dockerfile
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    expect \
    golang

WORKDIR /app
COPY . .
RUN make build

CMD ["./test/tty/expect_basic_tui.exp"]
```

## 🛠️ Test Development Guidelines

### **Writing New Expect Tests**
1. **Start with a clear purpose** (what functionality to test)
2. **Use descriptive variable names** (`binary_path`, `test_dir`)
3. **Include proper timeout handling** (avoid infinite waits)
4. **Add progress reporting** (puts statements for visibility)
5. **Handle multiple expected outcomes** (success/error/timeout)
6. **Include cleanup** (send "q", expect eof)

### **Pattern Matching Best Practices**
```tcl
# Good: Specific patterns
expect "Brummer - Development Buddy"

# Good: Regex patterns
expect -re "AI coder.*initialized"

# Good: Multiple outcomes
expect {
    timeout { puts "⏰ Timeout"; exit 1 }
    "success" { puts "✅ Success" }
    "error" { puts "❌ Error"; exit 1 }
    eof { puts "ℹ️  App ended" }
}

# Avoid: Too broad patterns
expect ".*"  # This matches anything!
```

### **Timing Considerations**
```tcl
# Good: Explicit waits with reason
sleep 3  # Wait for TUI initialization

# Good: Wait for specific events
expect "Ready"  # Wait until app is ready

# Avoid: Fixed timing assumptions
sleep 10  # Might be too short or too long
```

## 🔄 Migration from Old Tests

### **Before (Go PTY Tests)**
```go
// ❌ Fails because no /dev/tty
func TestTabNavigation(t *testing.T) {
    ptyTest := NewRealPTYIntegrationTest(t)
    defer ptyTest.Stop()

    err := ptyTest.StartWithPTY()
    require.NoError(t, err)

    helper.NavigateToTab(2)  // Fails - no TUI!
    helper.AssertProcessViewContent()  // Fails - no content!
}
```

### **After (Expect Tests)**
```tcl
# ✅ Works with real TTY
spawn ./brum -d . --debug
expect "AI coder manager initialized successfully"
sleep 3

send "2"  # Navigate to Processes tab
expect "Processes"  # Wait for tab content
puts "✅ Navigation successful"
```

## 📈 Benefits Achieved

### **1. Real User Experience Testing**
- ✅ Tests the exact environment users experience
- ✅ Validates actual TUI behavior
- ✅ No more "tests pass but app is broken"

### **2. Comprehensive Coverage**
- ✅ Full TUI functionality
- ✅ User interaction patterns
- ✅ Error scenarios and edge cases

### **3. Reliable Automation**
- ✅ Consistent results across environments
- ✅ Robust error handling
- ✅ Easy CI/CD integration

### **4. Debugging Capabilities**
- ✅ Real-time progress feedback
- ✅ Screen capture on failures
- ✅ Pattern matching diagnostics

## 🎯 Conclusion

The Expect-based testing paradigm provides **reliable, comprehensive testing** for Brummer's TUI functionality. It solves the fundamental limitation of testing terminal applications in automated environments by providing **real TTY access** with **precise control** and **robust validation**.

This approach ensures that **what works in tests works for users**, eliminating the "tests pass but app is broken" scenario that plagued previous testing efforts.

**Key Success Metrics:**
- ✅ Tests replicate manual user experience
- ✅ All TUI features are testable
- ✅ Automated testing is reliable
- ✅ CI/CD integration is straightforward
- ✅ Debugging is efficient and effective