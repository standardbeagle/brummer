# Todo: Refactor model.go into Controller Pattern

**Generated from**: Full Planning on 2025-01-31
**Next Phase**: Execute in phases to avoid massive merge conflicts

## Context Summary
- **Risk Level**: HIGH (4,641 lines → controller pattern) 
- **Project Phase**: Production refactoring
- **Estimated Effort**: 8-12 hours across 4 phases
- **Files**: ~20 files affected (extractions + test updates)
- **Feature Flag Required**: No (internal refactoring)

## Refactoring Strategy
- **Zero Tolerance**: Complete extraction, no half-measures or legacy code
- **Pattern Consistency**: Make code look like it was designed this way from start
- **Tight Coupling**: Keep related components together in same controller files
- **Helper Strategy**: Only shared utilities go to helpers.go

## Phase-Based Execution Plan

### Phase 1: View Controllers & Renderers (Risk: MEDIUM)
**Objective**: Extract all view rendering logic into dedicated controllers
**Files**: 6 controller files + model.go updates + tests

#### Task 1.1: Web View Controller Enhancement
- [ ] **Action**: Extract all web-related methods into enhanced WebViewController
- [ ] **Files**:
  - `internal/tui/web_view_controller.go` (enhance existing)
  - `internal/tui/model.go` (remove methods)
  - `internal/tui/model_test.go` (update)
- [ ] **Methods to Extract**:
  - `renderWebView()`, `renderRequestsList()`, `renderRequestDetail()`
  - `renderTelemetryDetails()`, `renderTelemetrySummary()`
  - `updateWebView()`, `updateWebRequestsList()`, `updateSelectedRequest*()`
  - `getFilteredRequests()`, `isPageRequest()`, `isAPIRequest()`, `isImageRequest()`
  - `formatStatus()`, `formatTelemetryEvent()`
- [ ] **Message Types**: Move to controller: `webUpdateMsg`
- [ ] **Success Criteria**:
  - [ ] All web view functionality in WebViewController
  - [ ] Web view renders identically: manual test
  - [ ] Tests pass: `go test -run "TestModel|TestWeb" ./internal/tui/`

#### Task 1.2: Settings Controller Creation
- [ ] **Action**: Create new SettingsController for all settings functionality
- [ ] **Files**:
  - `internal/tui/settings_controller.go` (new)
  - `internal/tui/model.go` (remove methods)
  - `internal/tui/model_test.go` (update)
- [ ] **Methods to Extract**:
  - `renderSettings()`, `updateSettingsList()`, `updateFileBrowserList()`
  - `installMCPForTool()`, `installMCPToFile()`
  - `getCLICommandFromConfig()`, `getCLICommand()`
- [ ] **Success Criteria**:
  - [ ] Settings view works identically
  - [ ] MCP installation still functional
  - [ ] Tests pass: `go test -run "TestSettings" ./internal/tui/`

#### Task 1.3: Layout & Core Renderers
- [ ] **Action**: Create LayoutController for core rendering logic
- [ ] **Files**:
  - `internal/tui/layout_controller.go` (new)
  - `internal/tui/model.go` (remove methods)
- [ ] **Methods to Extract**:
  - `renderLayout()`, `renderContent()`, `renderHeader()`
  - `renderProcessesView()`, `renderLogsView()`, `renderFiltersView()`
  - `updateSizes()`, `getViewStatus()`
- [ ] **Success Criteria**:
  - [ ] All views render identically
  - [ ] Header status updates work
  - [ ] Layout responds to resizing

### Phase 2: Input & Event Controllers (Risk: HIGH)
**Objective**: Extract all input handling and event management
**Files**: 3 controller files + model.go updates

#### Task 2.1: Input Handler Controller
- [ ] **Action**: Create InputController for all key/input handling
- [ ] **Files**:
  - `internal/tui/input_controller.go` (new)
  - `internal/tui/model.go` (remove methods, keep delegating Update())
- [ ] **Methods to Extract**:
  - `handleGlobalKeys()`, `handleEnter()`
  - `handleCommandWindow()`, `handleScriptSelector()`
  - All view-specific key handling from Update() method
- [ ] **Message Types**: Keep message handling in model.go, delegate to controllers
- [ ] **Success Criteria**:
  - [ ] All keyboard shortcuts work identically
  - [ ] View switching functions correctly
  - [ ] Command palette still opens with "/"

#### Task 2.2: Event System Controller
- [ ] **Action**: Create EventController for event subscriptions and handling
- [ ] **Files**:
  - `internal/tui/event_controller.go` (new)
  - `internal/tui/model.go` (remove setupEventSubscriptions)
- [ ] **Methods to Extract**:
  - `setupEventSubscriptions()`, `waitForUpdates()`, `tickCmd()`
  - Event message handling: `processUpdateMsg`, `logUpdateMsg`, etc.
- [ ] **Success Criteria**:
  - [ ] Process events still trigger updates
  - [ ] Log updates still appear in real-time
  - [ ] Error notifications still work

#### Task 2.3: Command & Dialog Controller Enhancement
- [ ] **Action**: Enhance existing CommandWindowController with extracted methods
- [ ] **Files**:
  - `internal/tui/command_window_controller.go` (enhance)
  - `internal/tui/model.go` (remove methods)
- [ ] **Methods to Extract**:
  - `handleSlashCommand()`, `handleClearCommand()`, `handleProxyCommand()`
  - `handleAICommand()`, `showRunDialog()`, `handleRunCommand()`
  - `showCommandWindow()`, `showTerminal()`, `toggleProxyMode()`
- [ ] **Success Criteria**:
  - [ ] All slash commands work: /show, /hide, /clear, /ai, /proxy
  - [ ] Command palette opens and functions
  - [ ] Run dialog creates processes correctly

### Phase 3: Process & System Controllers (Risk: MEDIUM)
**Objective**: Extract process management and system functionality
**Files**: 3 controller files + model.go updates

#### Task 3.1: Process Management Controller Enhancement
- [ ] **Action**: Enhance ProcessViewController with process operations
- [ ] **Files**:
  - `internal/tui/process_view_controller.go` (enhance)
  - `internal/tui/model.go` (remove methods)
- [ ] **Methods to Extract**:
  - `updateProcessList()`, `handleRestartProcess()`, `handleRestartAll()`
  - Process-specific key handling logic from Update()
- [ ] **Message Types**: Move to controller: `processUpdateMsg`, `restartProcessMsg`, `restartAllMsg`
- [ ] **Success Criteria**:
  - [ ] Process list updates correctly
  - [ ] Stop/restart/restart-all functions work
  - [ ] Process status changes reflect immediately

#### Task 3.2: System Message Controller Enhancement
- [ ] **Action**: Enhance existing system.Controller with extracted methods
- [ ] **Files**:
  - `internal/tui/system/controller.go` (enhance)
  - `internal/tui/model.go` (remove methods)
- [ ] **Methods to Extract**:
  - `addSystemMessage()`, `hasSystemMessages()`, `isSystemPanelExpanded()`
  - `toggleSystemPanel()`, `clearSystemMessages()`, `overlaySystemPanel()`
  - `updateUnreadIndicator()`, `clearUnreadIndicator()`
- [ ] **Message Types**: Move to system controller: `systemMessageMsg`
- [ ] **Success Criteria**:
  - [ ] System messages still appear
  - [ ] Panel toggle works with "e" key
  - [ ] Unread indicators function correctly

#### Task 3.3: Navigation Controller Enhancement
- [ ] **Action**: Enhance navigation.Controller with view management
- [ ] **Files**:
  - `internal/tui/navigation/controller.go` (enhance)
  - `internal/tui/model.go` (remove methods)
- [ ] **Methods to Extract**:
  - `cycleView()`, `cyclePrevView()`, `switchToView()`
  - `currentView()` (keep wrapper in model.go for external access)
- [ ] **Success Criteria**:
  - [ ] Tab/Shift+Tab view cycling works
  - [ ] Number key view switching works
  - [ ] Left/right arrow view navigation works

### Phase 4: Specialized Controllers & Cleanup (Risk: LOW)
**Objective**: Handle remaining specialized functionality and final cleanup
**Files**: 4 controller files + helpers + final cleanup

#### Task 4.1: Error Management Controller Enhancement
- [ ] **Action**: Enhance ErrorsViewController with all error functionality
- [ ] **Files**:
  - `internal/tui/errors_view_controller.go` (enhance)
  - `internal/tui/model.go` (remove methods)
- [ ] **Methods to Extract**:
  - `updateErrorsList()`, `updateErrorDetailView()`, `renderErrorsViewSplit()`
  - `handleClearErrors()`, `handleCopyError()`, `findLowestCodeReference()`
- [ ] **Message Types**: Move to controller: `errorUpdateMsg`
- [ ] **Success Criteria**:
  - [ ] Error list populates correctly
  - [ ] Error detail view shows context
  - [ ] Copy error functionality works

#### Task 4.2: MCP Debug Controller
- [ ] **Action**: Create MCPDebugController for debug-mode MCP functionality
- [ ] **Files**:
  - `internal/tui/mcp_debug_controller.go` (new)
  - `internal/tui/model.go` (remove methods)
- [ ] **Methods to Extract**:
  - `handleMCPConnection()`, `handleMCPActivity()`
  - `updateMCPConnectionsList()`, `updateMCPActivityView()`
  - MCP view rendering and handling logic
- [ ] **Message Types**: Move to controller: `mcpConnectionMsg`, `mcpActivityMsg`
- [ ] **Success Criteria**:
  - [ ] MCP connections view works in debug mode
  - [ ] MCP activity logging functions
  - [ ] Debug mode toggle shows/hides MCP view

#### Task 4.3: AI Coder Controller Enhancement
- [ ] **Action**: Enhance existing AI Coder components with remaining methods
- [ ] **Files**:
  - `internal/tui/ai_coder_pty_view.go` (enhance)
  - `internal/tui/model.go` (remove methods)
- [ ] **Methods to Extract**:
  - `renderAICoderPTYView()`, PTY event handling
  - AI Coder specific update logic from Update()
- [ ] **Success Criteria**:
  - [ ] AI Coder PTY view renders correctly
  - [ ] Slash command routing works
  - [ ] PTY focus/unfocus functions

#### Task 4.4: Shared Helpers & Final Cleanup
- [ ] **Action**: Create helpers.go and finalize model.go
- [ ] **Files**:
  - `internal/tui/helpers.go` (new)
  - `internal/tui/model.go` (final cleanup)
- [ ] **Methods to Extract to helpers.go**:
  - `formatBytes()`, `cleanLogContent()`, `getLogStyle()`
  - `convertToCollapsedEntries()`, `areLogsIdentical()`
  - Other utility functions used by multiple controllers
- [ ] **Final model.go should contain**:
  - Model struct definition
  - NewModel() constructor
  - Init(), Update(), View() methods (delegating to controllers)
  - Minimal orchestration logic
- [ ] **Success Criteria**:
  - [ ] model.go under 1000 lines
  - [ ] All functionality preserved
  - [ ] No code duplication
  - [ ] All tests pass: `go test ./internal/tui/...`

## Final Validation Commands
```bash
# Verify file structure
find internal/tui/ -name "*.go" | grep -E "(controller|helpers)" | sort

# Test all functionality
go test ./internal/tui/...

# Check code coverage
go test -cover ./internal/tui/...

# Verify line count reduction
wc -l internal/tui/model.go  # Should be ~500-1000 lines

# Manual testing checklist
echo "Manual tests required:"
echo "- All views render correctly"
echo "- All keyboard shortcuts work"
echo "- Process start/stop/restart works"
echo "- Command palette and slash commands work"
echo "- System messages and notifications work"
echo "- Settings and MCP installation works"
echo "- AI Coder integration works"
```

## Definition of Done
- [ ] model.go reduced from 4,641 to ~500-1000 lines
- [ ] All functionality extracted to appropriate controllers
- [ ] Zero old/dead code remaining
- [ ] All tests updated and passing
- [ ] Code looks like it was designed with this pattern from start
- [ ] No circular dependencies
- [ ] Manual testing confirms all features work identically
- [ ] Performance unchanged (no measurable regressions)

## Gotchas & Considerations
- **Event Subscription Timing**: Event subscriptions must remain in constructor or early Init()
- **Message Type Coupling**: Some message types are tightly coupled to Update() method
- **Controller Dependencies**: Some controllers may need references to others
- **Test Coverage**: Ensure controller tests cover extracted functionality
- **Import Cycles**: Watch for circular imports between controllers
- **Performance**: Large method extractions may affect compile times temporarily

## Risk Mitigation
- **Phase-by-phase execution**: Each phase is independently testable
- **Comprehensive testing**: Manual + automated testing after each phase
- **Rollback capability**: Each phase can be independently reverted
- **Conservative extraction**: Keep core orchestration in model.go
- **Pattern consistency**: Follow established controller patterns in codebase