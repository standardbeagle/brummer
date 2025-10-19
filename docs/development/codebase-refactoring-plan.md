# Brummer Codebase Refactoring Plan

## Executive Summary

This plan outlines a systematic refactoring of the Brummer codebase to improve maintainability, reduce complexity, and eliminate technical debt. The application has grown significantly (15,385 lines in TUI alone) and accumulated legacy code during the Redux/Elm migration.

## Current State Analysis

### 📊 Codebase Metrics
- **Total Go Files**: 359 files in `internal/`
- **TUI Complexity**: 15,385 lines across 80+ files
- **Test Coverage**: Extensive but fragmented test structure
- **Architecture**: Mixed (legacy + Redux/Elm patterns)

### 🚨 Issues Identified

#### 1. **Duplicate Architecture Patterns**
- `internal/tui/` - Main TUI implementation (Redux/Elm completed)
- `internal/tui_redux/` - Duplicate Redux implementation (15 files, 34 references)
- `architecture_switch.go` - Switching mechanism between architectures

#### 2. **Legacy Code Accumulation**
- `internal/mcp/legacy_handlers.go` - 179 lines of legacy HTTP handlers
- Multiple `.disabled` and `.bak` files (9 files found)
- Technical debt markers (TODO/FIXME) across 30+ files

#### 3. **Fragmented Directory Structure**
```
internal/
├── tui/                    # Main TUI (80+ files)
├── tui_redux/             # Duplicate Redux code ❌
├── tui/components/        # UI components
├── tui/state/             # State management
├── tui/services/          # Business logic
├── tui/handlers/          # Event handlers
├── tui/commands/          # Tea commands
├── tui/bridge/            # Integration layer
└── mcp/                   # MCP server + legacy handlers ❌
```

#### 4. **Bloated TUI Package**
- Single package with too many responsibilities
- Mixed concerns (UI, state, business logic, networking)
- Deep coupling between components

## 🎯 Refactoring Goals

1. **Eliminate Code Duplication** - Remove `tui_redux` duplicate
2. **Consolidate Legacy Code** - Remove or properly deprecate legacy handlers
3. **Improve Package Organization** - Split large packages into focused modules
4. **Reduce Complexity** - Clear separation of concerns
5. **Maintain Functionality** - Zero breaking changes for users

## 📋 Proposed Directory Structure

### New Package Organization
```
internal/
├── app/                    # Application entry point & coordination
│   ├── app.go             # Main application coordinator
│   ├── config.go          # Application configuration
│   └── lifecycle.go       # Startup/shutdown logic
├── tui/                   # Terminal UI (simplified)
│   ├── view/              # Pure view components
│   │   ├── processes/     # Process management UI
│   │   ├── logs/          # Log display UI
│   │   ├── settings/      # Settings UI
│   │   ├── errors/        # Error display UI
│   │   ├── urls/          # URL management UI
│   │   ├── commands/      # Command palette UI
│   │   └── layout/        # Layout management
│   ├── state/             # State management (keep current)
│   └── input/             # Input handling & routing
├── domain/                # Business logic domains
│   ├── processes/         # Process management domain
│   ├── logs/              # Log management domain
│   ├── mcp/               # MCP server domain
│   ├── proxy/             # Proxy server domain
│   └── scripts/           # Script management domain
├── infrastructure/        # External integrations
│   ├── http/              # HTTP clients & servers
│   ├── filesystem/        # File operations
│   ├── terminal/          # PTY & terminal management
│   └── events/            # Event system (keep current pkg/events)
├── shared/                # Shared utilities
│   ├── types/             # Common types
│   ├── errors/            # Error handling
│   └── utils/             # Utility functions
└── ports/                 # Application ports/interfaces
    ├── cli/               # CLI interface
    └── mcp/               # MCP interface
```

## 🚀 Migration Strategy

### Phase 1: Cleanup (Low Risk)
**Timeline**: 1-2 days

#### 1.1 Remove Legacy Code
- [ ] Delete `internal/mcp/legacy_handlers.go`
- [ ] Remove all `.disabled` and `.bak` files
- [ ] Clean up TODO/FIXME comments that are resolved
- [ ] Remove unused imports and dead code

#### 1.2 Eliminate Duplication
- [ ] Delete `internal/tui_redux/` directory
- [ ] Update imports in affected files (17 files)
- [ ] Remove `architecture_switch.go`
- [ ] Consolidate any unique logic from tui_redux

#### 1.3 Test & Validate
- [ ] Run full test suite
- [ ] Build verification
- [ ] Functional testing

### Phase 2: Package Reorganization (Medium Risk)
**Timeline**: 3-4 days

#### 2.1 Create New Package Structure
```bash
# Create new directories
mkdir -p internal/app
mkdir -p internal/tui/view/{processes,logs,settings,errors,urls,commands,layout}
mkdir -p internal/tui/input
mkdir -p internal/domain/{processes,logs,mcp,proxy,scripts}
mkdir -p internal/infrastructure/{http,filesystem,terminal}
mkdir -p internal/shared/{types,errors,utils}
mkdir -p internal/ports/{cli,mcp}
```

#### 2.2 Migrate Components by Domain

**Processes Domain**
```go
// Move from: internal/tui/components/process_view_component.go
// To: internal/tui/view/processes/view.go
// Also move: internal/process/* → internal/domain/processes/
```

**Logs Domain**
```go
// Move from: internal/tui/components/logs_view.go
// To: internal/tui/view/logs/view.go
// Also move: internal/logs/* → internal/domain/logs/
```

**Settings Domain**
```go
// Move from: internal/tui/components/settings_view.go
// To: internal/tui/view/settings/view.go
```

#### 2.3 Update Imports & Dependencies
- [ ] Update import paths across all files
- [ ] Fix package dependencies
- [ ] Resolve circular dependencies

#### 2.4 Preserve State Management
- [ ] Keep `internal/tui/state/` as-is (working well)
- [ ] Update state imports from new locations
- [ ] Verify Redux/Elm pattern still works

### Phase 3: Service Layer Refactoring (Medium Risk)
**Timeline**: 2-3 days

#### 3.1 Consolidate Services
```go
// Current: internal/tui/services/ (scattered)
// New: internal/domain/{name}/service.go

// Examples:
internal/domain/processes/service.go      // ProcessService
internal/domain/logs/service.go          // LogService
internal/domain/mcp/service.go           // MCPService
```

#### 3.2 Interface Extraction
- [ ] Extract interfaces from services
- [ ] Place in appropriate domain packages
- [ ] Update dependency injection

### Phase 4: Integration & Testing (Low Risk)
**Timeline**: 1-2 days

#### 4.1 Update Integration Points
- [ ] Update `cmd/brum/main.go` imports
- [ ] Fix test imports
- [ ] Update documentation

#### 4.2 Comprehensive Testing
- [ ] Unit tests for each package
- [ ] Integration tests across domains
- [ ] End-to-end functional tests
- [ ] Performance validation

## 📋 Detailed Migration Checklist

### Phase 1: Cleanup Tasks

#### Legacy Code Removal
```bash
# Files to remove:
rm internal/mcp/legacy_handlers.go
rm internal/tui/architecture_switch.go
rm internal/tui/architecture_example.go.disabled
rm internal/tui_redux/app.go.disabled
find . -name "*.bak" -delete
find . -name "*.disabled" -delete
```

#### tui_redux Cleanup
```bash
# Directory to remove:
rm -rf internal/tui_redux/

# Files that import tui_redux (need updates):
- internal/tui/architecture_switch.go (deleted)
- test/tui/integration/quickstart_test.go
- test/tui/contracts/update_contract_test.go
```

### Phase 2: Package Migration Tasks

#### Component Migration Mapping
```
Current → New Location

internal/tui/components/ → internal/tui/view/{domain}/
├── command_executor.go → internal/tui/view/commands/
├── integrated_shell.go → internal/tui/view/layout/
├── script_selector.go → internal/tui/view/processes/
├── test_view.go → internal/tui/view/layout/
├── urls_view.go → internal/tui/view/urls/
├── web_view.go → internal/tui/view/urls/
├── settings_view.go → internal/tui/view/settings/
├── errors_view.go → internal/tui/view/errors/
├── aicoders_view.go → internal/tui/view/layout/
└── command_window_view.go → internal/tui/view/commands/

internal/tui/services/ → internal/domain/{name}/
├── aicoder_service.go → internal/domain/mcp/
├── file_browser_service.go → internal/domain/filesystem/
├── notifications_service.go → internal/shared/
├── pty_input_adapter.go → internal/infrastructure/terminal/
└── pty_navigation_adapter.go → internal/infrastructure/terminal/

Domain Logic Migration:
internal/process/ → internal/domain/processes/
internal/logs/ → internal/domain/logs/
internal/mcp/ → internal/domain/mcp/
internal/proxy/ → internal/domain/proxy/
```

### Import Updates

#### Before/After Examples
```go
// Before
import "github.com/standardbeagle/brummer/internal/tui/components"
import "github.com/standardbeagle/brummer/internal/tui/services"

// After
import "github.com/standardbeagle/brummer/internal/tui/view/processes"
import "github.com/standardbeagle/brummer/internal/domain/processes"
```

## 🎯 Success Criteria

1. **Build Success**: Application builds without errors
2. **Test Coverage**: All existing tests pass
3. **Functionality**: No regressions in user-facing features
4. **Performance**: No performance degradation
5. **Code Quality**: Improved maintainability metrics
6. **Documentation**: Updated for new structure

## 🚨 Risk Mitigation

### High-Risk Operations
1. **Package deletion** - Mitigate with git version control
2. **Import mass updates** - Use automated refactoring tools
3. **Dependency resolution** - Incremental validation

### Rollback Strategy
- **Git checkpoints** after each phase
- **Feature flags** for critical components
- **Automated testing** at each step
- **Canary testing** for major changes

## 📊 Expected Benefits

### Code Quality Improvements
- **Reduced Complexity**: Smaller, focused packages
- **Better Separation**: Clear domain boundaries
- **Improved Testability**: Isolated components
- **Enhanced Readability**: Logical organization

### Development Experience
- **Faster Onboarding**: Clearer structure
- **Easier Debugging**: Isolated concerns
- **Better IDE Support**: Improved autocomplete
- **Simpler Testing**: Focused unit tests

### Maintenance Benefits
- **Reduced Technical Debt**: Cleaner codebase
- **Easier Refactoring**: Modular structure
- **Better Performance**: Optimized imports
- **Future-Proof**: Scalable architecture

## 📈 Implementation Timeline

**Total Estimated Time**: 8-11 days

### Week 1
- **Day 1-2**: Phase 1 - Cleanup (Legacy code removal)
- **Day 3-5**: Phase 2 - Package reorganization

### Week 2
- **Day 6-7**: Phase 3 - Service layer refactoring
- **Day 8-9**: Phase 4 - Integration & testing
- **Day 10-11**: Buffer for unforeseen issues

## Next Steps

1. **Stakeholder Review**: Review and approve this plan
2. **Branch Creation**: Create `refactor/package-reorganization` branch
3. **Automated Tools**: Set up automated refactoring scripts
4. **Begin Phase 1**: Start with low-risk cleanup operations

This refactoring will significantly improve the codebase while maintaining all existing functionality and Redux/Elm architectural benefits.