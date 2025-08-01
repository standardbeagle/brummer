# Test Management System

## Overview

The Test Management System provides comprehensive testing capabilities with a focus on minimal context bloat for AI agents while maintaining rich functionality for developers.

## Core Principles

### 1. Minimal Context for Success
- Passing tests generate minimal output: `✅ 23 tests passed (1.2s)`
- Only counts and timing for successful runs
- No detailed output unless requested

### 2. Rich Context for Failures
- Complete error messages and stack traces
- File locations and line numbers
- Related code context for debugging
- Suggested fixes when possible

### 3. Agent-Optimized Events
- Automatic notifications on test state changes
- Context-aware filtering based on recent file changes
- Smart relevance scoring for test results

## Architecture

### Component Structure
```
/internal/testing/
├── manager.go           # Test execution orchestration
├── runner.go           # Framework-specific test runners
├── parser.go           # Test output parsing and analysis
├── watcher.go          # File system watching for auto-testing
└── results.go          # Result storage and querying

/internal/tui/
└── test_view.go        # TUI test management interface

/internal/mcp/
└── test_tools.go       # MCP tools for test operations
```

### Test Runner Architecture
```go
type TestRunner interface {
    // Detect if this runner can handle the project
    CanHandle(projectPath string) bool
    
    // Run tests with optional patterns/files
    RunTests(ctx context.Context, opts RunOptions) (*TestResults, error)
    
    // Start watch mode for continuous testing
    StartWatch(ctx context.Context, opts WatchOptions) (<-chan TestResults, error)
    
    // Get test coverage information
    GetCoverage(ctx context.Context) (*CoverageData, error)
}

type RunOptions struct {
    Pattern     string   // Test pattern/filter
    Files       []string // Specific files to test
    Parallel    bool     // Run tests in parallel
    Verbose     bool     // Include verbose output
    Coverage    bool     // Collect coverage data
    Timeout     time.Duration
}
```

## Framework Support

### Built-in Runners

#### JavaScript/TypeScript
- **Jest**: Full support with watch mode and coverage
- **Vitest**: Native ESM support with hot reload
- **Mocha**: Traditional test runner support
- **Playwright**: E2E testing integration

#### Go
- **go test**: Native Go testing with race detection
- **testify**: Enhanced assertions and suites
- **ginkgo**: BDD-style testing

#### Python
- **pytest**: Full pytest support with plugins
- **unittest**: Standard library testing
- **coverage.py**: Coverage integration

#### Rust
- **cargo test**: Native Rust testing
- **criterion**: Benchmarking integration

### Auto-Detection Logic
```go
func DetectTestFramework(projectPath string) []TestRunner {
    var runners []TestRunner
    
    // Check for package.json and test scripts
    if hasFile(projectPath, "package.json") {
        if hasDevDep(projectPath, "jest") {
            runners = append(runners, &JestRunner{})
        }
        if hasDevDep(projectPath, "vitest") {
            runners = append(runners, &VitestRunner{})
        }
    }
    
    // Check for Go modules
    if hasFile(projectPath, "go.mod") {
        runners = append(runners, &GoTestRunner{})
    }
    
    // Check for Python tests
    if hasAnyFile(projectPath, "pytest.ini", "pyproject.toml", "test_*.py") {
        runners = append(runners, &PytestRunner{})
    }
    
    return runners
}
```

## TUI Integration

### Test View Layout
```
┌────────────────────────────────────────────────────────────────┐
│ Tests                                                    t     │
├────────────────────────────────────────────────────────────────┤
│ Status: ✅ 45 passed, ❌ 2 failed (3.4s)             r=run    │
│                                                                │
│ Recent Results:                                                │
│ ✅ user/auth_test.go                              (234ms)     │
│ ❌ user/profile_test.go                           (1.2s)      │
│   • TestUserProfile failed at line 23                        │
│   • Expected: "john@example.com"                             │
│   • Actual:   ""                                             │
│                                                                │
│ ✅ api/handlers_test.go                           (567ms)     │
│ ❌ api/middleware_test.go                         (89ms)      │
│   • TestAuthMiddleware panicked                              │
│   • runtime error: invalid memory address                    │
│   • handlers.go:45 -> middleware.go:12                       │
│                                                                │
│ Coverage: 78.5% (↑ 2.1%)                                     │
├────────────────────────────────────────────────────────────────┤
│ r: Run Tests | R: Run Pattern | c: Clear | w: Watch | f: Failed│
└────────────────────────────────────────────────────────────────┘
```

### Key Bindings
- `t`: Switch to test view
- `r`: Run all tests
- `R`: Run with pattern (opens input dialog)
- `c`: Clear test results
- `w`: Toggle watch mode
- `f`: Show only failed tests
- `v`: Toggle verbose output
- `g`: Show coverage report

### Progressive Disclosure
- **Summary View**: Default showing only counts and recent results
- **Detail View**: Full error messages and stack traces on selection
- **Coverage View**: Code coverage overlay when requested

## MCP Tools

### Core Test Tools
```yaml
test_run:
  description: "Execute tests with optional filtering and configuration"
  parameters:
    pattern: string (optional) # Test pattern or file filter
    files: array (optional)    # Specific files to test
    coverage: boolean          # Include coverage data
    verbose: boolean           # Include verbose output
    watch: boolean            # Start watch mode
  returns:
    results: TestResults      # Detailed test execution results
    summary: TestSummary      # Condensed results for AI agents

test_status:
  description: "Get current test execution status and recent results"
  parameters:
    since: string (optional)  # ISO timestamp for recent results
    failures_only: boolean    # Only return failed tests
  returns:
    status: TestStatus        # Current execution state
    results: TestSummary      # Condensed results

test_failures:
  description: "Get detailed information about failed tests"
  parameters:
    file: string (optional)   # Filter failures by file
    since: string (optional)  # Recent failures only
  returns:
    failures: array[TestFailure] # Detailed failure information

test_coverage:
  description: "Get code coverage data and analysis"
  parameters:
    format: string            # "summary" | "detailed" | "diff"
    files: array (optional)   # Specific files to analyze
  returns:
    coverage: CoverageData    # Coverage statistics and file details

test_watch:
  description: "Control test watch mode for continuous testing"
  parameters:
    action: string           # "start" | "stop" | "status"
    pattern: string (optional) # Watch pattern
  returns:
    status: WatchStatus      # Current watch mode status

test_clear:
  description: "Clear test result history and reset state"
  parameters:
    scope: string            # "all" | "failures" | "results"
  returns:
    cleared: boolean         # Success status
```

### Advanced Tools
```yaml
test_analyze:
  description: "Analyze test patterns and suggest improvements"
  parameters:
    scope: string            # "performance" | "coverage" | "flaky"
  returns:
    analysis: TestAnalysis   # Insights and recommendations

test_generate:
  description: "Generate test scaffolding for files or functions"
  parameters:
    file: string             # Target file for test generation
    functions: array (optional) # Specific functions to test
    framework: string (optional) # Preferred test framework
  returns:
    generated: GeneratedTests # Test file contents and suggestions

test_debug:
  description: "Get debugging context for failed tests"
  parameters:
    test_id: string          # Specific test to debug
    include_logs: boolean    # Include application logs
  returns:
    debug_info: DebugContext # Comprehensive debugging information
```

## Data Structures

### Test Results
```go
type TestResults struct {
    ID          string                 `json:"id"`
    Timestamp   time.Time             `json:"timestamp"`
    Duration    time.Duration         `json:"duration"`
    Framework   string                `json:"framework"`
    Command     string                `json:"command"`
    Summary     TestSummary           `json:"summary"`
    Tests       []TestCase            `json:"tests"`
    Coverage    *CoverageData         `json:"coverage,omitempty"`
    Metadata    map[string]interface{} `json:"metadata"`
}

type TestSummary struct {
    Total    int `json:"total"`
    Passed   int `json:"passed"`
    Failed   int `json:"failed"`
    Skipped  int `json:"skipped"`
    Duration time.Duration `json:"duration"`
}

type TestCase struct {
    ID          string        `json:"id"`
    Name        string        `json:"name"`
    File        string        `json:"file"`
    Line        int          `json:"line"`
    Status      TestStatus   `json:"status"`
    Duration    time.Duration `json:"duration"`
    Error       *TestError   `json:"error,omitempty"`
    Output      string       `json:"output,omitempty"`
}

type TestError struct {
    Message    string   `json:"message"`
    Stack      string   `json:"stack"`
    File       string   `json:"file"`
    Line       int      `json:"line"`
    Expected   string   `json:"expected,omitempty"`
    Actual     string   `json:"actual,omitempty"`
    Diff       string   `json:"diff,omitempty"`
}
```

## AI Agent Integration

### Event System
```go
type TestEvent struct {
    Type      TestEventType `json:"type"`
    Timestamp time.Time     `json:"timestamp"`
    Data      interface{}   `json:"data"`
    Context   EventContext  `json:"context"`
}

type TestEventType string
const (
    TestStarted    TestEventType = "test_started"
    TestCompleted  TestEventType = "test_completed"
    TestFailed     TestEventType = "test_failed"
    WatchTriggered TestEventType = "watch_triggered"
    CoverageChanged TestEventType = "coverage_changed"
)

type EventContext struct {
    RecentFiles   []string `json:"recent_files"`   // Files changed recently
    RelevantTests []string `json:"relevant_tests"` // Tests related to changes
    Severity      string   `json:"severity"`       // "low" | "medium" | "high"
}
```

### Context Optimization
```go
// For passing tests - minimal context
type MinimalTestResult struct {
    Summary TestSummary `json:"summary"`
    Duration time.Duration `json:"duration"`
    Timestamp time.Time `json:"timestamp"`
}

// For failing tests - rich context
type DetailedTestFailure struct {
    TestCase TestCase `json:"test"`
    RelatedFiles []string `json:"related_files"`
    RecentChanges []FileChange `json:"recent_changes"`
    SuggestedFixes []string `json:"suggested_fixes"`
    SimilarFailures []TestCase `json:"similar_failures"`
}
```

## Watch Mode Implementation

### File System Watching
```go
type TestWatcher struct {
    runner    TestRunner
    patterns  []string
    debounce  time.Duration
    ctx       context.Context
    cancel    context.CancelFunc
    events    chan TestEvent
}

func (w *TestWatcher) Start() error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    
    go w.watchLoop(watcher)
    return w.addWatchPaths(watcher)
}

func (w *TestWatcher) watchLoop(watcher *fsnotify.Watcher) {
    debounced := debounce.New(w.debounce)
    
    for {
        select {
        case event := <-watcher.Events:
            if w.shouldTrigger(event) {
                debounced(func() {
                    w.runTests(event)
                })
            }
        case <-w.ctx.Done():
            return
        }
    }
}
```

### Smart Test Selection
```go
func (w *TestWatcher) selectRelevantTests(changedFiles []string) []string {
    var relevantTests []string
    
    for _, file := range changedFiles {
        // Direct test files
        if strings.Contains(file, "_test.") {
            relevantTests = append(relevantTests, file)
            continue
        }
        
        // Find tests that import this file
        tests := w.findTestsThatImport(file)
        relevantTests = append(relevantTests, tests...)
        
        // Pattern-based matching (e.g., user.go -> user_test.go)
        testFile := w.inferTestFile(file)
        if w.fileExists(testFile) {
            relevantTests = append(relevantTests, testFile)
        }
    }
    
    return uniqueStrings(relevantTests)
}
```

## Performance Optimizations

### Parallel Test Execution
- Run independent test suites in parallel
- Respect framework-specific parallelism settings
- Queue management for resource-constrained environments

### Result Caching
- Cache test results based on file checksums
- Skip unchanged tests in watch mode
- Incremental coverage calculation

### Memory Management
- Limit stored test history (configurable, default 100 runs)
- Compress old test results
- Stream large test outputs instead of storing in memory

## Configuration

### Test Configuration
```toml
[testing]
# Default test runner timeout
timeout = "5m"

# Maximum number of stored test results
max_history = 100

# Enable coverage by default
default_coverage = false

# Watch mode debounce interval
watch_debounce = "500ms"

# Framework-specific settings
[testing.jest]
coverage_threshold = 80
parallel = true

[testing.go]
race_detection = true
coverage_profile = "coverage.out"

[testing.pytest]
markers = ["unit", "integration"]
capture = "no"
```

## Error Handling & Recovery

### Graceful Degradation
- Continue operation if specific framework fails
- Fallback to basic test detection
- Maintain partial results on interruption

### Error Context
- Capture framework-specific error details
- Provide actionable error messages
- Suggest fixes for common test setup issues

## Future Enhancements

### AI-Powered Features
- **Intelligent Test Generation**: Generate tests based on code analysis
- **Flaky Test Detection**: Identify and suggest fixes for unreliable tests
- **Performance Regression Detection**: Alert on significant performance changes
- **Test Quality Scoring**: Rate test effectiveness and coverage quality

### Advanced Integrations
- **CI/CD Pipeline Integration**: Import test results from remote builds
- **Code Review Integration**: Show test status in pull requests
- **Monitoring Integration**: Connect test failures to production issues
- **Team Dashboards**: Aggregate test metrics across team members