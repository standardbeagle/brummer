---
sidebar_position: 3
---

# Contributing to Brummer

Thank you for your interest in contributing to Brummer! This guide will help you get started.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:

- **Be respectful** and inclusive
- **Be patient** with new contributors
- **Be constructive** in feedback
- **Be collaborative** and helpful

## Getting Started

### 1. Fork and Clone

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/brummer.git
cd brummer
git remote add upstream https://github.com/original/brummer.git
```

### 2. Development Setup

```bash
# Install Go dependencies
go mod download

# Install development tools
make install-tools

# Run tests to verify setup
make test
```

### 3. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

## Development Guidelines

### Code Style

We use standard Go conventions with some additions:

1. **Format code** with `gofmt`
2. **Lint code** with `golangci-lint`
3. **Use meaningful** variable names
4. **Add comments** for exported functions
5. **Keep functions small** and focused

Example:

```go
// ProcessManager handles the lifecycle of external processes.
type ProcessManager struct {
    processes map[string]*Process
    mu        sync.RWMutex
}

// Start begins execution of the named script.
// It returns an error if the script doesn't exist or fails to start.
func (pm *ProcessManager) Start(name string) error {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    process, exists := pm.processes[name]
    if !exists {
        return fmt.Errorf("process %q not found", name)
    }
    
    return process.Start()
}
```

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
type(scope): subject

body

footer
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test additions/changes
- `chore`: Build process or auxiliary tool changes

Examples:

```bash
# Feature
git commit -m "feat(tui): add keyboard shortcut for process restart"

# Bug fix
git commit -m "fix(process): handle SIGTERM gracefully"

# Documentation
git commit -m "docs(readme): update installation instructions"
```

### Testing

#### Writing Tests

1. **Unit tests** for individual functions
2. **Integration tests** for component interactions
3. **Table-driven tests** for multiple scenarios

Example test:

```go
func TestProcessManager_Start(t *testing.T) {
    tests := []struct {
        name    string
        process string
        wantErr bool
    }{
        {"valid process", "build", false},
        {"invalid process", "nonexistent", true},
        {"already running", "running", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            pm := NewProcessManager()
            err := pm.Start(tt.process)
            if (err != nil) != tt.wantErr {
                t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

#### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package
go test ./internal/process/...

# Run with verbose output
go test -v ./...
```

### Documentation

#### Code Documentation

- Document all exported types and functions
- Use examples in documentation
- Keep comments up-to-date with code

```go
// LogEntry represents a single log line from a process.
// It includes metadata for filtering and searching.
//
// Example:
//
//     entry := LogEntry{
//         ProcessID: "build-123",
//         Level:     LogLevelError,
//         Content:   "Build failed: missing dependency",
//     }
type LogEntry struct {
    ProcessID string
    Level     LogLevel
    Content   string
    Timestamp time.Time
}
```

#### User Documentation

When adding features:

1. Update relevant documentation in `docs-site/`
2. Add examples if applicable
3. Update CLI help text
4. Add to changelog

## Submitting Changes

### 1. Pre-submission Checklist

- [ ] Code follows style guidelines
- [ ] Tests pass locally
- [ ] New tests added for new features
- [ ] Documentation updated
- [ ] Commit messages follow convention
- [ ] Branch is up-to-date with main

### 2. Pull Request Process

1. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create Pull Request** on GitHub

3. **Fill out PR template**:
   ```markdown
   ## Description
   Brief description of changes
   
   ## Type of Change
   - [ ] Bug fix
   - [ ] New feature
   - [ ] Breaking change
   - [ ] Documentation update
   
   ## Testing
   - [ ] Unit tests pass
   - [ ] Integration tests pass
   - [ ] Manual testing completed
   
   ## Checklist
   - [ ] My code follows project style
   - [ ] I've updated documentation
   - [ ] I've added tests
   ```

### 3. Code Review

Expect reviewers to:

- Check code quality and style
- Verify test coverage
- Suggest improvements
- Ask questions for clarity

Respond to feedback:

- Be open to suggestions
- Explain your decisions
- Make requested changes
- Ask for clarification if needed

## Types of Contributions

### Bug Reports

File issues with:

1. **Clear title** describing the bug
2. **Steps to reproduce**
3. **Expected behavior**
4. **Actual behavior**
5. **System information**
6. **Error messages/logs**

Example:

```markdown
**Title**: Process fails to restart after crash

**Steps to reproduce**:
1. Start a process with `brummer`
2. Kill the process externally
3. Try to restart from TUI

**Expected**: Process restarts successfully
**Actual**: Error "process already running"

**System**: macOS 14.0, Brummer v1.2.3
**Logs**: [attached]
```

### Feature Requests

Propose features with:

1. **Use case** description
2. **Proposed solution**
3. **Alternative approaches**
4. **Additional context**

### Documentation

Help improve docs by:

1. **Fixing typos** and grammar
2. **Adding examples**
3. **Clarifying confusing sections**
4. **Translating** to other languages

### Code Contributions

Areas needing help:

1. **Bug fixes** from issue tracker
2. **Feature implementations** from roadmap
3. **Performance improvements**
4. **Test coverage** increase
5. **Refactoring** for maintainability

## Development Environment

### Recommended Tools

1. **Editor**: VSCode with Go extension
2. **Debugger**: Delve
3. **Profiler**: pprof
4. **Terminal**: iTerm2/Windows Terminal

### VSCode Settings

`.vscode/settings.json`:

```json
{
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.formatTool": "goimports",
  "go.testOnSave": true,
  "editor.formatOnSave": true
}
```

### Debugging

```bash
# Debug with Delve
dlv debug cmd/brummer/main.go

# Attach to running process
dlv attach $(pgrep brummer)

# Debug tests
dlv test ./internal/process
```

## Release Process

### Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes

### Release Checklist

1. [ ] Update version in `version.go`
2. [ ] Update CHANGELOG.md
3. [ ] Run full test suite
4. [ ] Build for all platforms
5. [ ] Create GitHub release
6. [ ] Update documentation
7. [ ] Announce in Discord

## Getting Help

### Resources

- **Discord**: [discord.gg/brummer](https://discord.gg/brummer)
- **Discussions**: GitHub Discussions
- **Issues**: GitHub Issues
- **Wiki**: Project Wiki

### Maintainers

- @maintainer1 - Core architecture
- @maintainer2 - TUI components

## Recognition

Contributors are recognized in:

- CONTRIBUTORS.md file
- Release notes
- Project README
- Annual contributor spotlight

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (MIT).