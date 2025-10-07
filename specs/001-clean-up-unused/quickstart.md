# Quickstart Guide - Code Cleanup and AI Coder Features

## Prerequisites
- Go 1.21+ installed
- Brummer development environment set up
- Git repository with code to analyze

## Setup Code Cleanup Features

### 1. Basic Code Analysis
```bash
# Analyze current directory for unused imports
brum analyze --type=imports

# Full code analysis (imports, functions, dead code)
brum analyze --type=all --output=analysis-report.json

# Analyze specific directories
brum analyze --target=./internal --target=./cmd --type=functions
```

### 2. Safe Code Cleanup
```bash
# Preview cleanup changes before applying
brum cleanup --dry-run --analysis=analysis-report.json

# Apply cleanup with backup creation
brum cleanup --backup --analysis=analysis-report.json

# Selective cleanup (only unused imports)
brum cleanup --analysis=analysis-report.json --imports-only
```

## Setup AI Coder Features

### 1. Start AI Coding Session
```bash
# Start AI chat terminal emulator
brum ai coder --session-type=chat

# Start AI coding session with browser integration
brum ai coder --session-type=coding --connect-browser

# Start AI debugging session
brum ai coder --session-type=debugging --working-dir=./src
```

### 2. Browser Integration Setup
```bash
# Connect browser debug tools automatically
brum ai coder --browser=chrome --auto-connect

# Manual browser connection
brum ai coder --browser-proxy-port=19888
# Then connect browser to: http://localhost:19888/debug

# Stream browser/server errors to TUI
brum ai coder --error-stream --level=all
```

## Testing Key Handler Fixes

### 1. Verify Global Key Handlers
```bash
# Test key handling in isolation
brum test-keys --global-handlers

# Test key handling during AI sessions
brum test-keys --ai-session-active

# Generate key handling report
brum test-keys --report=key-handling-report.json
```

### 2. Manual Key Testing
1. Start Brummer: `brum`
2. Test global keys: `Ctrl+C` (should exit cleanly), `q` (should quit)
3. Test navigation keys: `?` (help), number keys (view switching)
4. Start AI session: Press AI coder hotkey
5. Test keys during AI session: Ensure terminal keys work, global keys still function

## Performance Profiling

### 1. Generate Performance Profile
```bash
# Profile key event handling
brum profile --feature=key-events --duration=30s

# Profile code analysis performance
brum profile --feature=analysis --target=./internal

# Full application profiling
brum profile --all --output=profile-report.pprof
```

### 2. Analyze Profile Results
```bash
# Analyze CPU profile
go tool pprof profile-report.pprof

# View key event latency statistics
brum profile --report=key-latencies

# Identify performance hotspots
brum profile --hotspots --sort=cpu
```

## Integration Testing

### 1. End-to-End Workflow Test
```bash
# Complete workflow test
brum test-e2e --scenario=code-cleanup-ai-integration

# Test with sample codebase
brum test-e2e --sample-repo=./test-data/sample-project
```

### 2. Validate Results
```bash
# Check code cleanup results
brum validate --cleanup-report=cleanup-report.json

# Verify AI session functionality
brum validate --ai-session --test-browser-integration

# Performance validation
brum validate --performance-baseline --compare-with=baseline.json
```

## Troubleshooting

### Common Issues
1. **Key handlers not working**: Run `brum test-keys --diagnose`
2. **AI session not connecting**: Check browser proxy settings with `brum ai --check-connection`
3. **Code analysis errors**: Verify Go version with `brum doctor`
4. **Performance issues**: Generate profile with `brum profile --quick`

### Recovery Commands
```bash
# Restore from backup if cleanup caused issues
brum restore --backup=./backups/pre-cleanup-<timestamp>.tar.gz

# Reset AI session state
brum ai reset --session-id=<session-id>

# Reset key handler configuration
brum config reset --section=key-handlers
```

## Success Criteria
- [ ] Global key handlers (Ctrl+C, q) work consistently
- [ ] Code cleanup removes unused elements without breaking functionality
- [ ] AI coding sessions connect to browser debug tools successfully
- [ ] Error streaming from browser/server displays in TUI
- [ ] Performance profiling identifies optimization opportunities
- [ ] All existing user workflows remain functional