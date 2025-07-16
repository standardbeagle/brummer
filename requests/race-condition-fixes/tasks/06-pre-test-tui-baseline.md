# Task: Pre-Development TUI Baseline Testing

## Persona: Test Engineer
Role: Baseline Tester
Expertise: System validation, regression testing, Go testing frameworks

## Current State Assessment
**Before starting this task:**
```yaml
existing_state:
  - ✅ Research analysis completed with specific recommendations
  - ✅ Synchronized architecture design completed
  - ✅ TUI Model identified with 60+ value receiver methods
  - ✅ Static analysis shows Model copy warnings from go vet
  - ❌ Baseline test results not captured before critical changes
  - ❌ TUI functionality test coverage not measured
  - ❌ Performance baseline not established
  - ❓ Unknown: Current memory usage and goroutine count during TUI operations

current_files:
  - internal/tui/model.go: Has value receiver race conditions (60+ methods)
  - internal/tui/model_test.go: Basic tests, may have copy issues
  - cmd/brum/main.go: TUI initialization code
  - internal/tui/: All TUI component files that use Model
```

## File Scope Definition
**Explicit file list for this task:**
```yaml
# Focus on TUI area that will be modified
test_scope:
  - internal/tui/**/*.go         # All TUI files
  - cmd/brum/main.go             # TUI initialization
  
baseline_files:
  - internal/tui/model.go        # Primary change target
  - internal/tui/mcp_connections.go  # Model usage
  - internal/tui/script_selector.go  # Model usage
  - internal/tui/model_test.go   # Existing tests
  
run_tests:
  - go test -v ./internal/tui/...
  - go test -race -v ./internal/tui/...
  - go vet ./internal/tui/...
  - golangci-lint run ./internal/tui/...

create_files:
  - requests/race-condition-fixes/baselines/tui-baseline-report.md
  - requests/race-condition-fixes/baselines/tui-performance-baseline.txt
  - requests/race-condition-fixes/baselines/tui-race-detector-baseline.txt

# Total: ~20 files (TUI package + baselines)
```

## Purpose
Establish comprehensive baseline before TUI Model critical changes to ensure:
- **Existing functionality** works correctly
- **No pre-existing failures** that could be blamed on new code
- **Clear before/after comparison** possible
- **Performance impact** can be accurately measured
- **Race condition status** is documented

## Task Requirements
- **Objective**: Create comprehensive TUI baseline before critical pointer receiver changes
- **Risk Level**: LOW (Testing only, no code changes)
- **Dependencies**: 02-design-sync-architecture (completed)
- **Deliverables**: 
  - Complete baseline test report
  - Performance metrics baseline
  - Race detector baseline results
  - Current TUI functionality status

## Success Criteria Checklist
- [ ] Run all TUI unit tests and document results
- [ ] Execute race detector tests on TUI package
- [ ] Capture performance baseline for TUI operations
- [ ] Document current Model value receiver warnings (go vet)
- [ ] Test TUI functionality manually if needed
- [ ] Record current memory usage during TUI operations
- [ ] Document any existing failures or issues
- [ ] Establish test coverage baseline for TUI package

## Baseline Validation
```bash
# Run all unit tests for TUI area
go test -v ./internal/tui/
# Expected: Document pass/fail status and any existing issues

# Run race detector tests  
go test -race -v ./internal/tui/
# Expected: Document race conditions found

# Static analysis for TUI
go vet ./internal/tui/
# Expected: Document 60+ Model copy warnings

# Linting check
golangci-lint run ./internal/tui/
# Expected: Document existing lint issues

# Test coverage baseline
go test -cover ./internal/tui/
# Record: Current coverage percentage

# Performance baseline
go test -bench=. -benchmem ./internal/tui/
# Record: Current benchmark results if any exist

# Memory usage check during tests
go test -memprofile=tui-mem.prof ./internal/tui/
# Record: Memory allocation patterns
```

## Baseline Report Structure

### Test Results Baseline
- Unit test pass/fail status
- Existing test failures (if any)
- Test coverage percentage
- Test execution time

### Race Detector Baseline  
- Race conditions detected in current code
- Specific files and line numbers
- Types of races (data races, map races, etc.)
- Severity assessment

### Performance Baseline
- Benchmark results (if benchmarks exist)
- Memory allocation patterns
- Goroutine count during operations
- CPU usage patterns

### Static Analysis Baseline
- go vet warnings (especially Model copy warnings)
- golangci-lint issues
- Cyclomatic complexity metrics
- Code quality indicators

## Risk Mitigation
- **No Code Changes**: This task only observes and documents current state
- **Comprehensive Documentation**: All findings documented for comparison
- **Issue Categorization**: Separate pre-existing issues from future changes

## Success Validation
```bash
# Verify baseline documents created
ls -la requests/race-condition-fixes/baselines/
# Expected: 3 baseline documents

# Check baseline completeness
wc -l requests/race-condition-fixes/baselines/*.{md,txt}
# Expected: Substantial documentation of current state

# Verify no code changes made
git diff --name-only
# Expected: Only baseline documents added, no source code changes
```

## Go/No-Go Decision Criteria

### PROCEED Conditions:
- [ ] All baseline tests documented (pass or fail)
- [ ] Race detector results captured  
- [ ] Performance baseline established
- [ ] Current issues clearly identified and categorized
- [ ] No unexpected critical failures that would block development

### STOP Conditions (Fix existing issues first):
- [ ] Critical TUI functionality completely broken
- [ ] Build failures in TUI package
- [ ] Unresolvable dependency conflicts

## Expected Baseline Findings

Based on previous analysis, expecting to document:

### Race Detector Results
- **Model copy warnings**: 60+ instances from go vet
- **Concurrent access**: Potential races in Model field access
- **Channel operations**: Race conditions in updateChan usage
- **Goroutine spawning**: Races from Init() and Update() methods

### Test Coverage Areas
- **Existing tests**: Basic Model functionality
- **Missing coverage**: Concurrent operations testing
- **Performance tests**: Likely minimal or none
- **Integration tests**: TUI component interactions

### Performance Characteristics
- **Memory allocation**: High allocation due to Model copying
- **Goroutine usage**: Potential goroutine leaks from value receivers
- **Lock contention**: Minimal due to lack of proper synchronization
- **Event processing**: Inefficient due to copying overhead

## Context from PRD
This baseline establishes the "before" state for the most critical fix in the entire project. The TUI Model pointer receiver conversion affects 60+ methods and is the foundation for all other race condition fixes.

## Constraints
- **Time**: 1 hour maximum (testing and documentation only)
- **No Changes**: Absolutely no code modifications
- **Comprehensive**: Must capture complete current state
- **Accurate**: Baseline must be precise for later comparison

## Execution Checklist
- [ ] Set up clean test environment
- [ ] Run comprehensive TUI test suite
- [ ] Execute race detector on TUI package
- [ ] Capture performance baseline measurements
- [ ] Document all static analysis warnings
- [ ] Test basic TUI functionality manually
- [ ] Record current resource usage patterns
- [ ] Create comprehensive baseline report
- [ ] Categorize existing issues vs. future fixes
- [ ] Make Go/No-Go decision for proceeding with TUI Model fixes

## Success Indicators
- **Comprehensive Baseline**: All aspects of current TUI system documented
- **Clear Categorization**: Existing issues vs. race condition fixes clearly separated  
- **Measurable Metrics**: Quantitative baseline for later comparison
- **Go Decision**: Clear recommendation to proceed with TUI Model pointer conversion
- **Foundation Set**: Solid baseline for measuring success of critical fixes