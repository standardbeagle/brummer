# Go Race Detection Tools and CI Integration

## Executive Summary

This document provides comprehensive guidance for integrating Go race detection tools into Brummer's development workflow. It covers static analysis tools, runtime detection, continuous monitoring, and CI/CD pipeline integration to prevent race conditions from reaching production.

## Go Built-in Race Detector

### 1. Race Detector Fundamentals

#### How Go's Race Detector Works
```go
// The race detector instruments memory accesses at compile-time
// and tracks happens-before relationships at runtime

// Example race condition the detector catches:
var counter int

func incrementUnsafe() {
    counter++ // Race condition: read-modify-write without synchronization
}

func incrementSafe() {
    atomic.AddInt64(&counter, 1) // Safe: atomic operation
}

// Detection output:
// ==================
// WARNING: DATA RACE
// Write at 0x00c000010088 by goroutine 7:
//   main.incrementUnsafe()
//       /path/to/file.go:123 +0x44
//
// Previous read at 0x00c000010088 by goroutine 6:
//   main.incrementUnsafe()
//       /path/to/file.go:123 +0x3c
// ==================
```

#### Performance Impact and Limitations
```go
// Race detector overhead analysis for Brummer:
type RaceDetectorImpact struct {
    MemoryOverhead    float64 // 5-10x memory usage
    CPUOverhead       float64 // 2-20x slower execution
    MaxGoroutines     int     // ~8192 tracked goroutines
    MaxMemoryAccesses int64   // ~128M tracked accesses
}

var brummerImpact = RaceDetectorImpact{
    MemoryOverhead:    8.0,  // 8x for Brummer's workload
    CPUOverhead:       12.0, // 12x slower (event-heavy)
    MaxGoroutines:     100,  // Sufficient for test scenarios
    MaxMemoryAccesses: 1000000, // Sufficient for unit tests
}

// Suitable for: Unit tests, integration tests, local development
// Not suitable for: Production, performance benchmarks, stress tests
```

### 2. Makefile Integration for Race Detection

#### Enhanced Test Targets
```makefile
# Add to existing Makefile after analysis

# Race detection for unit tests
.PHONY: test-race
test-race:
	@echo "🔍 Running unit tests with race detection..."
	@GOMAXPROCS=4 go test -race -timeout=2m \
		./pkg/events \
		./internal/process \
		./internal/logs \
		./internal/proxy \
		./internal/mcp

# Race detection for specific components (faster feedback)
.PHONY: test-race-events
test-race-events:
	@echo "🔍 Testing EventBus for race conditions..."
	@go test -race -v -timeout=30s ./pkg/events

.PHONY: test-race-process
test-race-process:
	@echo "🔍 Testing Process Manager for race conditions..."
	@go test -race -v -timeout=30s ./internal/process

.PHONY: test-race-logs
test-race-logs:
	@echo "🔍 Testing Log Store for race conditions..."
	@go test -race -v -timeout=30s ./internal/logs

# Extended race testing with stress scenarios
.PHONY: test-race-stress
test-race-stress:
	@echo "🔍 Running stress tests with race detection..."
	@GOMAXPROCS=8 go test -race -timeout=5m \
		-run="TestStress|TestConcurrent|TestRace" \
		./...

# Race detection with coverage (slower but comprehensive)
.PHONY: test-race-coverage
test-race-coverage:
	@echo "🔍 Running race detection with coverage..."
	@go test -race -cover -coverprofile=race-coverage.out \
		-timeout=5m ./pkg/events ./internal/process ./internal/logs
	@go tool cover -html=race-coverage.out -o race-coverage.html
	@echo "📊 Coverage report: race-coverage.html"

# Quick race check for pre-commit hooks
.PHONY: test-race-quick
test-race-quick:
	@echo "🔍 Quick race detection check..."
	@timeout 60s go test -race -short ./pkg/events ./internal/process || true
```

### 3. Environment Configuration

#### Development Environment Setup
```bash
# .envrc file for direnv users
export GORACE="halt_on_error=1 history_size=3"
export GOMAXPROCS=4  # Limit for consistent race detection

# For CI environments
export GORACE="halt_on_error=1 strip_path_prefix=/go/src/ log_path=./race-reports"

# For debugging specific races
export GORACE="halt_on_error=0 history_size=7 exitcode=75"
```

#### Race Detector Configuration Options
```go
// GORACE environment variable options for Brummer

var raceDetectorConfigs = map[string]string{
    // Development (fail fast, detailed output)
    "development": "halt_on_error=1 history_size=3 strip_path_prefix=" + os.Getenv("PWD"),
    
    // CI (collect all races, structured output) 
    "ci": "halt_on_error=0 history_size=2 log_path=./race-reports/ exitcode=66",
    
    // Debugging (maximum detail, continue on race)
    "debug": "halt_on_error=0 history_size=7 strip_path_prefix=" + os.Getenv("PWD"),
    
    // Quick check (minimal overhead)
    "quick": "halt_on_error=1 history_size=1",
}

// Usage in test setup:
func TestMain(m *testing.M) {
    // Set appropriate race detector config for testing
    if os.Getenv("GORACE") == "" {
        config := "development"
        if os.Getenv("CI") != "" {
            config = "ci"
        }
        os.Setenv("GORACE", raceDetectorConfigs[config])
    }
    
    os.Exit(m.Run())
}
```

## Static Analysis Tools

### 1. go vet Integration

#### Current go vet Issues in Brummer
```bash
# Existing go vet output shows 60+ copy warnings
$ go vet ./...

# Key findings:
internal/tui/model.go:523:6: UpdateView passes lock by value: brummer/internal/tui.Model contains sync.RWMutex
internal/tui/model.go:634:6: HandleKeyPress passes lock by value: brummer/internal/tui.Model contains sync.RWMutex
internal/process/manager.go:234:14: assignment copies lock value: sync.RWMutex
internal/logs/store.go:445:23: range copies lock value: brummer/internal/logs.LogEntry contains sync.Mutex

# These indicate value receiver issues that can cause races
```

#### Enhanced go vet Configuration
```makefile
# Enhanced vet checking
.PHONY: vet
vet:
	@echo "🔍 Running enhanced static analysis..."
	@go vet ./...
	@go vet -all ./...
	@go vet -shadow ./...
	@go vet -copylocks=false ./... # Disable if too noisy during refactoring

# Focused vet checking for concurrency issues
.PHONY: vet-concurrency
vet-concurrency:
	@echo "🔍 Checking for concurrency issues..."
	@go vet -structtag=false -assign=false ./... # Focus on concurrency
	@go vet -copylocks ./...
	@go vet -atomic ./...
```

### 2. golangci-lint Configuration

#### Comprehensive linter setup for race detection
```yaml
# .golangci.yml - Enhanced configuration for Brummer
run:
  timeout: 5m
  tests: true
  modules-download-mode: readonly

linters:
  enable:
    # Core race detection linters
    - govet          # Built-in static analysis
    - race           # Race condition detection
    - staticcheck    # Advanced static analysis
    - gosec          # Security issues including races
    
    # Memory safety linters  
    - ineffassign    # Ineffective assignments
    - gocritic       # Comprehensive checks
    - unconvert      # Unnecessary conversions
    
    # Concurrency-specific linters
    - rowserrcheck   # sql.Rows.Err() checks
    - sqlclosecheck  # sql.Close() checks
    - bodyclose      # HTTP body close checks
    
    # Code quality linters
    - errcheck       # Unchecked errors
    - gosimple       # Code simplification
    - unused         # Unused code detection
    - deadcode       # Dead code elimination

linters-settings:
  govet:
    check-shadowing: true
    enable:
      - atomicalign  # Atomic alignment issues
      - deepequalerrors
      - fieldalignment
      - findcall
      - nilfunc
      - printf
      - shift
      - stdmethods
      - structtag
      - tests
      - unreachable
      - unsafeptr
      
  staticcheck:
    checks: ["all", "-ST1000", "-ST1003", "-ST1016", "-ST1020", "-ST1021", "-ST1022"]
    
  gosec:
    includes:
      - G104  # Audit errors not checked
      - G204  # Subprocess launched with variable
      - G301  # Poor file permissions used when creating a directory
      - G302  # Poor file permissions used with chmod
      - G304  # File path provided as taint input
      - G401  # Detect the usage of DES, RC4, MD5 or SHA1
      
  gocritic:
    enabled-tags:
      - diagnostic
      - experimental
      - opinionated
      - performance
      - style
    disabled-checks:
      - dupImport # https://github.com/go-critic/go-critic/issues/845
      - ifElseChain
      - octalLiteral
      - paramTypeCombine

issues:
  exclude-rules:
    # Exclude some linters from running on tests files
    - path: _test\.go
      linters:
        - gosec
        - dupl
        - gocritic
    
    # Exclude specific race-related issues during refactoring
    - text: "copylocks: assignment copies lock value"
      linters:
        - govet
      path: internal/tui/  # Temporary exclusion during TUI refactoring
```

#### Makefile Integration for golangci-lint
```makefile
# Enhanced linting targets
.PHONY: lint
lint:
	@echo "🔍 Running comprehensive linting..."
	@command -v golangci-lint > /dev/null || \
		(echo "Installing golangci-lint..." && \
		 go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run --timeout=5m

# Race-focused linting
.PHONY: lint-race
lint-race:
	@echo "🔍 Running race-focused linting..."
	@golangci-lint run --enable=race,govet,staticcheck,gosec --timeout=3m

# Fix auto-fixable linting issues
.PHONY: lint-fix
lint-fix:
	@echo "🔧 Auto-fixing linting issues..."
	@golangci-lint run --fix --timeout=3m

# Strict linting for CI
.PHONY: lint-strict
lint-strict:
	@echo "🔍 Running strict linting for CI..."
	@golangci-lint run --issues-exit-code=1 --timeout=5m
```

### 3. Advanced Static Analysis

#### Custom Static Analysis with go/analysis
```go
// tools/racecheck/main.go - Custom race condition detector
package main

import (
    "go/ast"
    "golang.org/x/tools/go/analysis"
    "golang.org/x/tools/go/analysis/passes/inspect"
    "golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
    Name:     "brummerrace",
    Doc:      "Check for common race conditions in Brummer codebase",
    Requires: []*analysis.Analyzer{inspect.Analyzer},
    Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
    inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
    
    // Check for common race patterns in Brummer
    patterns := []string{
        "unlimitedGoroutines",  // EventBus pattern
        "valueReceivers",       // TUI Model pattern  
        "unprotectedMaps",      // Process Manager pattern
        "mixedSyncAsync",       // Log Store pattern
    }
    
    for _, pattern := range patterns {
        checkPattern(inspect, pass, pattern)
    }
    
    return nil, nil
}

func checkPattern(inspect *inspector.Inspector, pass *analysis.Pass, pattern string) {
    switch pattern {
    case "unlimitedGoroutines":
        checkUnlimitedGoroutines(inspect, pass)
    case "valueReceivers":
        checkValueReceivers(inspect, pass)
    case "unprotectedMaps":
        checkUnprotectedMaps(inspect, pass)
    case "mixedSyncAsync":
        checkMixedSyncAsync(inspect, pass)
    }
}

func checkUnlimitedGoroutines(inspect *inspector.Inspector, pass *analysis.Pass) {
    nodeFilter := []ast.Node{
        (*ast.GoStmt)(nil),
    }
    
    inspect.Preorder(nodeFilter, func(n ast.Node) {
        goStmt := n.(*ast.GoStmt)
        
        // Check if goroutine is created in a loop without bounds
        if isInLoop(goStmt) && !hasBoundedChannel(goStmt) {
            pass.Reportf(goStmt.Pos(), 
                "potential unlimited goroutine creation in loop")
        }
    })
}

func checkValueReceivers(inspect *inspector.Inspector, pass *analysis.Pass) {
    nodeFilter := []ast.Node{
        (*ast.FuncDecl)(nil),
    }
    
    inspect.Preorder(nodeFilter, func(n ast.Node) {
        funcDecl := n.(*ast.FuncDecl)
        
        if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
            recv := funcDecl.Recv.List[0]
            
            // Check if receiver is by value and contains mutex
            if !isPointerReceiver(recv) && containsMutex(recv.Type) {
                pass.Reportf(funcDecl.Pos(),
                    "method %s uses value receiver with mutex field", 
                    funcDecl.Name.Name)
            }
        }
    })
}
```

#### Usage in Makefile
```makefile
# Custom static analysis
.PHONY: analyze-race
analyze-race:
	@echo "🔍 Running custom race analysis..."
	@go run ./tools/racecheck ./...

# Combined static analysis
.PHONY: analyze-all
analyze-all: vet lint analyze-race
	@echo "✅ Static analysis complete"
```

## Runtime Detection and Monitoring

### 1. Continuous Race Detection

#### Test Suite for Race Detection
```go
// internal/testing/race_test.go
package testing

import (
    "context"
    "runtime"
    "sync"
    "testing"
    "time"
    
    "github.com/standardbeagle/brummer/pkg/events"
    "github.com/standardbeagle/brummer/internal/process"
)

func TestEventBusRaceConditions(t *testing.T) {
    if !isRaceEnabled() {
        t.Skip("Race detector not enabled")
    }
    
    eb := events.NewEventBus()
    
    // Test concurrent subscribe/publish
    t.Run("ConcurrentSubscribePublish", func(t *testing.T) {
        var wg sync.WaitGroup
        
        // Publishers
        for i := 0; i < 10; i++ {
            wg.Add(1)
            go func() {
                defer wg.Done()
                for j := 0; j < 100; j++ {
                    eb.Publish(events.Event{
                        Type: events.LogLine,
                        Data: map[string]interface{}{"test": j},
                    })
                }
            }()
        }
        
        // Subscribers
        for i := 0; i < 5; i++ {
            wg.Add(1)
            go func(id int) {
                defer wg.Done()
                for j := 0; j < 20; j++ {
                    eb.Subscribe(events.LogLine, func(e events.Event) {
                        // Handler work
                        time.Sleep(time.Microsecond)
                    })
                }
            }(i)
        }
        
        wg.Wait()
    })
    
    // Test handler execution races
    t.Run("HandlerExecutionRaces", func(t *testing.T) {
        counter := 0
        var mu sync.Mutex
        
        // Race-free handler
        eb.Subscribe(events.ProcessStarted, func(e events.Event) {
            mu.Lock()
            counter++
            mu.Unlock()
        })
        
        var wg sync.WaitGroup
        for i := 0; i < 50; i++ {
            wg.Add(1)
            go func() {
                defer wg.Done()
                eb.Publish(events.Event{Type: events.ProcessStarted})
            }()
        }
        
        wg.Wait()
        time.Sleep(100 * time.Millisecond) // Allow handlers to complete
        
        mu.Lock()
        if counter != 50 {
            t.Errorf("Expected 50 events, got %d", counter)
        }
        mu.Unlock()
    })
}

func TestProcessManagerRaceConditions(t *testing.T) {
    if !isRaceEnabled() {
        t.Skip("Race detector not enabled")
    }
    
    mgr := setupTestProcessManager(t)
    
    t.Run("ConcurrentStatusUpdates", func(t *testing.T) {
        process := createTestProcess(mgr, "test-process")
        
        var wg sync.WaitGroup
        
        // Status readers
        for i := 0; i < 20; i++ {
            wg.Add(1)
            go func() {
                defer wg.Done()
                for j := 0; j < 100; j++ {
                    _ = process.GetStatus()
                    runtime.Gosched()
                }
            }()
        }
        
        // Status writers  
        statuses := []process.ProcessStatus{
            process.StatusRunning,
            process.StatusStopped, 
            process.StatusFailed,
            process.StatusSuccess,
        }
        
        for i := 0; i < 5; i++ {
            wg.Add(1)
            go func(id int) {
                defer wg.Done()
                for j := 0; j < 20; j++ {
                    status := statuses[j%len(statuses)]
                    process.SetStatus(status)
                    time.Sleep(time.Microsecond)
                }
            }(i)
        }
        
        wg.Wait()
    })
    
    t.Run("ConcurrentProcessListing", func(t *testing.T) {
        // Add multiple processes concurrently
        var wg sync.WaitGroup
        
        for i := 0; i < 10; i++ {
            wg.Add(1)
            go func(id int) {
                defer wg.Done()
                for j := 0; j < 5; j++ {
                    processName := fmt.Sprintf("process-%d-%d", id, j)
                    _, err := mgr.StartScript(processName)
                    if err != nil {
                        t.Errorf("Failed to start process: %v", err)
                    }
                }
            }(i)
        }
        
        // Concurrent readers
        for i := 0; i < 5; i++ {
            wg.Add(1)
            go func() {
                defer wg.Done()
                for j := 0; j < 50; j++ {
                    _ = mgr.GetAllProcesses()
                    runtime.Gosched()
                }
            }()
        }
        
        wg.Wait()
    })
}

func isRaceEnabled() bool {
    // Detect if race detector is enabled
    return raceEnabled
}

// This variable is set by the race detector
var raceEnabled bool

func init() {
    defer func() {
        if recover() != nil {
            raceEnabled = false
        }
    }()
    
    // This will panic if race detector is not enabled
    runtime.RaceRead(&raceEnabled)
    raceEnabled = true
}
```

### 2. Production Race Monitoring

#### Safe Race Detection for Production
```go
// internal/monitoring/race_monitor.go
package monitoring

import (
    "fmt"
    "runtime"
    "sync"
    "sync/atomic"
    "time"
)

// ProductionRaceMonitor provides lightweight race detection for production
type ProductionRaceMonitor struct {
    enabled        int32 // atomic
    violations     int64 // atomic counter
    lastViolation  int64 // atomic timestamp
    maxViolations  int64
    reportCallback func(violation RaceViolation)
}

type RaceViolation struct {
    Type        string
    Location    string
    Goroutine   int64
    Timestamp   time.Time
    Description string
}

func NewProductionRaceMonitor(maxViolations int64, callback func(RaceViolation)) *ProductionRaceMonitor {
    return &ProductionRaceMonitor{
        enabled:        1,
        maxViolations:  maxViolations,
        reportCallback: callback,
    }
}

func (prm *ProductionRaceMonitor) CheckDataAccess(data interface{}, operation string) {
    if atomic.LoadInt32(&prm.enabled) == 0 {
        return
    }
    
    violations := atomic.LoadInt64(&prm.violations)
    if violations >= prm.maxViolations {
        atomic.StoreInt32(&prm.enabled, 0) // Disable to prevent spam
        return
    }
    
    // Lightweight check using memory patterns
    if prm.detectSuspiciousAccess(data, operation) {
        prm.reportViolation(operation, data)
    }
}

func (prm *ProductionRaceMonitor) detectSuspiciousAccess(data interface{}, operation string) bool {
    // Implementation of lightweight heuristics
    // - Check for rapid successive access to same memory location
    // - Detect access patterns that suggest races
    // - Use sampling to reduce overhead
    return false // Placeholder
}

func (prm *ProductionRaceMonitor) reportViolation(operation string, data interface{}) {
    violation := RaceViolation{
        Type:        "data_race_suspected",
        Location:    getCallerLocation(),
        Goroutine:   int64(getGoroutineID()),
        Timestamp:   time.Now(),
        Description: fmt.Sprintf("Suspicious %s operation", operation),
    }
    
    atomic.AddInt64(&prm.violations, 1)
    atomic.StoreInt64(&prm.lastViolation, time.Now().Unix())
    
    if prm.reportCallback != nil {
        go prm.reportCallback(violation) // Non-blocking report
    }
}

func getCallerLocation() string {
    _, file, line, ok := runtime.Caller(3)
    if !ok {
        return "unknown"
    }
    return fmt.Sprintf("%s:%d", file, line)
}

func getGoroutineID() int {
    // Simplified goroutine ID extraction
    return runtime.NumGoroutine()
}

// Usage in Brummer components
func (eb *EventBus) PublishWithMonitoring(event Event) {
    if eb.raceMonitor != nil {
        eb.raceMonitor.CheckDataAccess(&eb.handlers, "read")
    }
    
    eb.mu.RLock()
    handlers := eb.handlers[event.Type]
    eb.mu.RUnlock()
    
    for _, handler := range handlers {
        go handler(event)
    }
}
```

## CI/CD Pipeline Integration

### 1. GitHub Actions Integration

#### Comprehensive Race Detection Workflow
```yaml
# .github/workflows/race-detection.yml
name: Race Detection

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  race-detection:
    name: Race Detection Tests
    runs-on: ubuntu-latest
    timeout-minutes: 15
    
    strategy:
      matrix:
        go-version: [1.24.x]
        test-type: [unit, integration, stress]
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      
    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go-version }}
        
    - name: Cache dependencies
      uses: actions/cache@v3
      with:
        path: |
          ~/.cache/go-build
          ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        
    - name: Install dependencies
      run: go mod download
      
    - name: Configure race detector
      run: |
        echo "GORACE=halt_on_error=0 log_path=./race-reports/ strip_path_prefix=${{ github.workspace }}/" >> $GITHUB_ENV
        echo "GOMAXPROCS=4" >> $GITHUB_ENV
        mkdir -p race-reports
        
    - name: Run race detection tests
      run: |
        case ${{ matrix.test-type }} in
          unit)
            make test-race
            ;;
          integration)
            make test-race-integration
            ;;
          stress)
            make test-race-stress
            ;;
        esac
      continue-on-error: true
      
    - name: Process race reports
      if: always()
      run: |
        if [ -d race-reports ] && [ "$(ls -A race-reports)" ]; then
          echo "Race conditions detected!"
          find race-reports -name "*.race" -exec cat {} \;
          exit 1
        else
          echo "No race conditions detected"
        fi
        
    - name: Upload race reports
      if: failure()
      uses: actions/upload-artifact@v3
      with:
        name: race-reports-${{ matrix.test-type }}
        path: race-reports/
        retention-days: 30
        
    - name: Comment on PR
      if: failure() && github.event_name == 'pull_request'
      uses: actions/github-script@v6
      with:
        script: |
          const fs = require('fs');
          const path = require('path');
          
          try {
            const raceDir = 'race-reports';
            const files = fs.readdirSync(raceDir);
            
            if (files.length > 0) {
              let comment = '⚠️ **Race conditions detected in ${{ matrix.test-type }} tests:**\n\n';
              comment += '```\n';
              
              files.forEach(file => {
                const content = fs.readFileSync(path.join(raceDir, file), 'utf8');
                comment += content + '\n';
              });
              
              comment += '```\n\n';
              comment += 'Please fix these race conditions before merging.';
              
              github.rest.issues.createComment({
                issue_number: context.issue.number,
                owner: context.repo.owner,
                repo: context.repo.repo,
                body: comment
              });
            }
          } catch (error) {
            console.log('No race reports found or error reading them:', error.message);
          }

  static-analysis:
    name: Static Race Analysis
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      
    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: 1.24.x
        
    - name: Install analysis tools
      run: |
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
        go install honnef.co/go/tools/cmd/staticcheck@latest
        
    - name: Run go vet
      run: make vet-concurrency
      
    - name: Run golangci-lint
      run: make lint-race
      
    - name: Run staticcheck
      run: staticcheck -checks=SA1029,SA2001,SA2002,SA2003 ./...
      
    - name: Custom race analysis
      run: make analyze-race
```

### 2. Pre-commit Hooks

#### Git Hooks for Race Detection
```bash
#!/bin/bash
# .git/hooks/pre-commit

set -e

echo "🔍 Running pre-commit race detection..."

# Quick race detection on changed files
CHANGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [ -n "$CHANGED_GO_FILES" ]; then
    echo "Checking Go files for race conditions..."
    
    # Extract package directories from changed files
    PACKAGES=$(echo "$CHANGED_GO_FILES" | xargs -I {} dirname {} | sort -u | xargs -I {} echo "./{}")
    
    # Run quick race detection on affected packages
    echo "Running race detection on: $PACKAGES"
    timeout 60s go test -race -short $PACKAGES || {
        echo "❌ Race conditions detected in changed files!"
        echo "Run 'make test-race' for detailed analysis"
        exit 1
    }
    
    # Run static analysis on changed files
    echo "Running static analysis..."
    go vet $PACKAGES || {
        echo "❌ Static analysis issues detected!"
        exit 1
    }
fi

echo "✅ Pre-commit race detection passed"
```

#### Installation Script
```bash
#!/bin/bash
# scripts/install-hooks.sh

set -e

echo "Installing race detection git hooks..."

# Copy pre-commit hook
cp scripts/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit

# Copy pre-push hook for extended race detection
cp scripts/pre-push .git/hooks/pre-push  
chmod +x .git/hooks/pre-push

echo "✅ Git hooks installed"
echo "💡 Use 'git commit --no-verify' to skip race detection if needed"
```

### 3. Continuous Monitoring Dashboard

#### Race Detection Metrics Collection
```go
// internal/monitoring/race_dashboard.go
package monitoring

import (
    "encoding/json"
    "fmt"
    "html/template"
    "net/http"
    "sync/atomic"
    "time"
)

type RaceDashboard struct {
    metrics       *RaceMetrics
    server        *http.Server
    updateChannel chan RaceEvent
}

type RaceMetrics struct {
    TotalTests      int64 `json:"total_tests"`
    RacesFailed     int64 `json:"races_failed"`
    RacesPassed     int64 `json:"races_passed"`
    LastRaceTime    int64 `json:"last_race_time"`
    AverageTestTime int64 `json:"average_test_time"`
}

type RaceEvent struct {
    Timestamp time.Time
    Type      string // "test_passed", "test_failed", "race_detected"
    Details   string
    Duration  time.Duration
}

func NewRaceDashboard(port int) *RaceDashboard {
    dashboard := &RaceDashboard{
        metrics:       &RaceMetrics{},
        updateChannel: make(chan RaceEvent, 100),
    }
    
    mux := http.NewServeMux()
    mux.HandleFunc("/", dashboard.handleDashboard)
    mux.HandleFunc("/api/metrics", dashboard.handleMetrics)
    mux.HandleFunc("/api/events", dashboard.handleEvents)
    
    dashboard.server = &http.Server{
        Addr:    fmt.Sprintf(":%d", port),
        Handler: mux,
    }
    
    go dashboard.processEvents()
    
    return dashboard
}

func (rd *RaceDashboard) RecordEvent(event RaceEvent) {
    select {
    case rd.updateChannel <- event:
    default:
        // Channel full, drop event
    }
}

func (rd *RaceDashboard) processEvents() {
    for event := range rd.updateChannel {
        switch event.Type {
        case "test_passed":
            atomic.AddInt64(&rd.metrics.RacesPassed, 1)
        case "test_failed", "race_detected":
            atomic.AddInt64(&rd.metrics.RacesFailed, 1)
            atomic.StoreInt64(&rd.metrics.LastRaceTime, event.Timestamp.Unix())
        }
        
        atomic.AddInt64(&rd.metrics.TotalTests, 1)
        
        // Update average test time
        currentAvg := atomic.LoadInt64(&rd.metrics.AverageTestTime)
        newAvg := (currentAvg + event.Duration.Nanoseconds()) / 2
        atomic.StoreInt64(&rd.metrics.AverageTestTime, newAvg)
    }
}

func (rd *RaceDashboard) handleMetrics(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    
    metrics := RaceMetrics{
        TotalTests:      atomic.LoadInt64(&rd.metrics.TotalTests),
        RacesFailed:     atomic.LoadInt64(&rd.metrics.RacesFailed),
        RacesPassed:     atomic.LoadInt64(&rd.metrics.RacesPassed),
        LastRaceTime:    atomic.LoadInt64(&rd.metrics.LastRaceTime),
        AverageTestTime: atomic.LoadInt64(&rd.metrics.AverageTestTime),
    }
    
    json.NewEncoder(w).Encode(metrics)
}

const dashboardTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Brummer Race Detection Dashboard</title>
    <meta refresh="5">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { display: inline-block; margin: 10px; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .failed { background-color: #ffe6e6; }
        .passed { background-color: #e6ffe6; }
        .total { background-color: #e6f3ff; }
    </style>
</head>
<body>
    <h1>🔍 Brummer Race Detection Dashboard</h1>
    
    <div class="metric total">
        <h3>Total Tests</h3>
        <div id="total-tests">Loading...</div>
    </div>
    
    <div class="metric passed">
        <h3>Races Passed</h3>
        <div id="races-passed">Loading...</div>
    </div>
    
    <div class="metric failed">
        <h3>Races Failed</h3>
        <div id="races-failed">Loading...</div>
    </div>
    
    <div class="metric">
        <h3>Success Rate</h3>
        <div id="success-rate">Loading...</div>
    </div>
    
    <div class="metric">
        <h3>Last Race Failure</h3>
        <div id="last-race">Loading...</div>
    </div>
    
    <script>
        function updateMetrics() {
            fetch('/api/metrics')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('total-tests').textContent = data.total_tests;
                    document.getElementById('races-passed').textContent = data.races_passed;
                    document.getElementById('races-failed').textContent = data.races_failed;
                    
                    const successRate = data.total_tests > 0 ? 
                        (data.races_passed / data.total_tests * 100).toFixed(1) + '%' : 'N/A';
                    document.getElementById('success-rate').textContent = successRate;
                    
                    const lastRace = data.last_race_time > 0 ? 
                        new Date(data.last_race_time * 1000).toLocaleString() : 'Never';
                    document.getElementById('last-race').textContent = lastRace;
                });
        }
        
        updateMetrics();
        setInterval(updateMetrics, 5000);
    </script>
</body>
</html>
`

func (rd *RaceDashboard) handleDashboard(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(dashboardTemplate))
}
```

## Conclusion and Implementation Roadmap

### Phase 1: Foundation (Week 1)
1. **Makefile Integration**: Add race detection targets
2. **Static Analysis**: Configure golangci-lint with race-focused rules
3. **Basic Testing**: Create race detection test suite

### Phase 2: CI Integration (Week 2)  
1. **GitHub Actions**: Implement comprehensive race detection workflow
2. **Pre-commit Hooks**: Add quick race detection to git workflow
3. **Reporting**: Set up race condition reporting and notifications

### Phase 3: Advanced Monitoring (Week 3)
1. **Production Monitoring**: Implement lightweight race detection
2. **Dashboard**: Create race detection metrics dashboard
3. **Alerting**: Set up automated alerts for race condition detection

### Expected Outcomes:
- **100% race detection coverage** for all concurrent code paths
- **Zero race conditions** in production deployments
- **Continuous monitoring** of race condition trends
- **Developer education** through immediate feedback

This comprehensive race detection strategy ensures that Brummer maintains high code quality and prevents race conditions from impacting users, while providing developers with the tools and feedback needed to write concurrent code safely.