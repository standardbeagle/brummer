# Tasks: Controller to Service/Component Migration

**Input**: Design documents from `/specs/002-the-controllers-need/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/, quickstart.md

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- Paths shown below are relative to repository root
- Components: `internal/tui/components/`
- Tests: `test/integration/`
- Controllers to remove: `internal/tui/*_controller.go`

## Phase 3.1: Setup and Documentation
- [ ] T001 [P] Create component interface documentation in internal/tui/components/README.md
- [ ] T002 [P] Create migration guide in docs/architecture/controller-migration.md
- [ ] T003 [P] Set up integration test framework with TTY allocation in test/integration/test_helpers.go
- [ ] T004 [P] Create component test utilities and mocks in test/integration/component_test_utils.go
- [ ] T005 [P] Set up performance testing framework in test/integration/performance_helpers.go

## Phase 3.2: Pre-Migration Integration Tests (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST PASS (verifying current functionality) before ANY controller migration**

### Functional Tests (Write to verify current behavior)
- [ ] T006 [P] Integration test for process management workflow in test/integration/process_management_test.go
- [ ] T007 [P] Integration test for log viewing and filtering in test/integration/log_viewing_test.go
- [ ] T008 [P] Integration test for error display and management in test/integration/error_management_test.go
- [ ] T009 [P] Integration test for URL discovery and display in test/integration/url_discovery_test.go
- [ ] T010 [P] Integration test for proxy request viewing in test/integration/proxy_viewing_test.go
- [ ] T011 [P] Integration test for AI coder session lifecycle in test/integration/ai_coder_test.go
- [ ] T012 [P] Integration test for command palette functionality in test/integration/command_palette_test.go
- [ ] T013 [P] Integration test for settings management in test/integration/settings_test.go
- [ ] T014 [P] Integration test for keyboard navigation (all global keys) in test/integration/keyboard_navigation_test.go
- [ ] T015 [P] Integration test for script selector functionality in test/integration/script_selector_test.go

### Performance Baseline Tests
- [ ] T016 [P] Key event latency baseline test (<50ms requirement) in test/integration/key_latency_test.go
- [ ] T017 [P] View rendering performance baseline test (<16ms for 60fps) in test/integration/render_performance_test.go

## Phase 3.3: Foundation Migration (ONLY after tests pass)

### Foundation Layer (No Dependencies)
- [X] T018 Migrate event_controller.go: Move EventBus subscription to model.go Init()
- [X] T019 Create layout_manager.go as pure function utility in internal/tui/components/layout_manager.go
- [X] T020 Remove event_controller.go file and all references from model.go
- [X] T021 Remove layout_controller.go file and update model.go to use layout_manager
- [X] T022 Run integration tests T006-T017 to verify foundation migration (tests compile, run with `go test -tags=integration`)

## Phase 3.4: Simple Views Migration

### URLs View
- [X] T023 Create or enhance urls_view.go component in internal/tui/components/urls_view.go
- [X] T024 Migrate URL display logic from urls_view_controller.go to urls_view component
- [X] T025 Implement URL service integration (process manager) in urls_view component
- [X] T026 Remove urls_view_controller.go and update model.go references
- [ ] T027 Run test/integration/url_discovery_test.go to verify URLs view migration

### MCP Debug View
- [X] T028 Create mcp_debug_view.go component in internal/tui/components/mcp_debug_view.go
- [X] T029 Migrate MCP status display from mcp_debug_controller.go to mcp_debug_view component
- [X] T030 Implement MCP service integration in mcp_debug_view component
- [X] T031 Remove mcp_debug_controller.go and update model.go references
- [ ] T032 Run manual test of MCP view (press 9 or appropriate number key)

### Command Palette
- [X] T033 Create command_palette.go component in internal/tui/components/command_palette.go
- [X] T034 Migrate command window logic from command_window_controller.go to command_palette
- [X] T035 Integrate command autocomplete component into command_palette
- [X] T036 Remove command_window_controller.go and update model.go references
- [ ] T037 Run test/integration/command_palette_test.go to verify command palette migration

## Phase 3.5: Core Views Migration

### Processes View
- [X] T038 Create processes_view.go component in internal/tui/components/processes_view.go
- [X] T039 Migrate process list display from process_view_controller.go to processes_view
- [X] T040 Implement process manager service integration in processes_view
- [X] T041 Add process selection and navigation logic to processes_view
- [X] T042 Remove process_view_controller.go and update model.go references
- [ ] T043 Run test/integration/process_management_test.go to verify processes view migration

### Logs View
- [X] T044 Create or enhance logs_view.go component in internal/tui/components/logs_view.go
- [X] T045 Migrate log display and filtering from logs_view_controller.go to logs_view
- [X] T046 Implement log store service integration in logs_view
- [X] T047 Add auto-scroll and follow mode to logs_view
- [X] T048 Remove logs_view_controller.go and update model.go references
- [ ] T049 Run test/integration/log_viewing_test.go to verify logs view migration

### Errors View
- [X] T050 Enhance errors_view.go component (already exists) in internal/tui/components/errors_view.go
- [X] T051 Migrate error list and filtering from errors_view_controller.go to errors_view
- [X] T052 Implement error store service integration in errors_view
- [X] T053 Add error dismissal and clearing logic to errors_view
- [X] T054 Remove errors_view_controller.go and update model.go references
- [ ] T055 Run test/integration/error_management_test.go to verify errors view migration

## Phase 3.6: Complex Views Migration

### Web/Proxy View
- [X] T056 Enhance web_view.go component (already exists) in internal/tui/components/web_view.go
- [X] T057 Migrate proxy request display from web_view_controller.go to web_view
- [X] T058 Implement proxy server service integration in web_view
- [X] T059 Add request filtering and detail view to web_view
- [X] T060 Remove web_view_controller.go and update model.go references
- [ ] T061 Run test/integration/proxy_viewing_test.go to verify web view migration

### Settings View
- [X] T062 Create settings_view.go component in internal/tui/components/settings_view.go
- [X] T063 Migrate settings display from settings_controller.go to settings_view
- [X] T064 Implement config service integration in settings_view
- [X] T065 Add settings editing and save logic to settings_view
- [X] T066 Remove settings_controller.go and update model.go references
- [ ] T067 Run test/integration/settings_test.go to verify settings view migration

### Script Selector
- [X] T068 Create script_selector.go component in internal/tui/components/script_selector.go
- [X] T069 Migrate script list display from script_selector_controller.go to script_selector
- [X] T070 Implement script parsing and process manager integration in script_selector
- [X] T071 Add script selection and launch logic to script_selector
- [X] T072 Remove script_selector_controller.go and update model.go references
- [ ] T073 Run test/integration/script_selector_test.go to verify script selector migration

## Phase 3.7: Advanced Features Migration

### AI Coder / Integrated Shell
- [ ] T074 Enhance integrated_shell.go component (already exists) in internal/tui/components/integrated_shell.go
- [ ] T075 Migrate AI coder session management from ai_coder_controller.go to integrated_shell
- [ ] T076 Migrate PTY terminal integration and input routing to integrated_shell
- [ ] T077 Migrate slash command interception logic to integrated_shell
- [ ] T078 Remove ai_coder_controller.go and update model.go references
- [ ] T079 Run test/integration/ai_coder_test.go to verify AI coder migration

## Phase 3.8: Input System Migration (MUST BE LAST)

### Keyboard Handling
- [ ] T080 Enhance hierarchical_key_handler.go (already exists) in internal/tui/components/hierarchical_key_handler.go
- [ ] T081 Migrate global key handling from input_controller.go to hierarchical_key_handler
- [ ] T082 Update all components to implement KeyHandler interface
- [ ] T083 Migrate view-specific key routing from input_controller.go to components
- [ ] T084 Migrate command window input handling from input_controller.go to command_palette
- [ ] T085 Migrate quit confirmation logic from input_controller.go to hierarchical_key_handler
- [ ] T086 Remove input_controller.go and update model.go to use hierarchical_key_handler directly
- [ ] T087 Run test/integration/keyboard_navigation_test.go to verify keyboard handling migration

## Phase 3.9: Final Integration and Cleanup

### Model Cleanup
- [ ] T088 Remove all controller field declarations from internal/tui/model.go
- [ ] T089 Remove all controller initialization code from model.go NewModel()
- [ ] T090 Update model.go Update() method to route messages to components directly
- [ ] T091 Update model.go View() method to call component View() methods directly
- [ ] T092 Remove all controller method calls from model.go (should be 0 references)

### Verification
- [ ] T093 Run full integration test suite (T006-T017) to verify all functionality
- [ ] T094 Run performance tests (T016-T017) to verify no regressions
- [ ] T095 Run `make test-race` to verify concurrency safety
- [ ] T096 Manual testing: Verify all keyboard shortcuts work
- [ ] T097 Manual testing: Verify all views display correctly
- [ ] T098 Manual testing: Verify real-time updates work (process start/stop, logs, etc.)

## Phase 3.10: Performance Optimization and Polish

### Performance
- [ ] T099 [P] Profile key event handling latency with pprof in test/integration/profiling_test.go
- [ ] T100 [P] Profile view rendering performance with pprof
- [ ] T101 [P] Identify and fix any performance hotspots (if found)

### Unit Tests
- [ ] T102 [P] Unit tests for ProcessesViewComponent in internal/tui/components/processes_view_test.go
- [ ] T103 [P] Unit tests for LogsViewComponent in internal/tui/components/logs_view_test.go
- [ ] T104 [P] Unit tests for ErrorsViewComponent in internal/tui/components/errors_view_test.go
- [ ] T105 [P] Unit tests for WebViewComponent in internal/tui/components/web_view_test.go
- [ ] T106 [P] Unit tests for SettingsViewComponent in internal/tui/components/settings_view_test.go
- [ ] T107 [P] Unit tests for CommandPaletteComponent in internal/tui/components/command_palette_test.go
- [ ] T108 [P] Unit tests for ScriptSelectorComponent in internal/tui/components/script_selector_test.go
- [ ] T109 [P] Unit tests for IntegratedShellComponent in internal/tui/components/integrated_shell_test.go
- [ ] T110 [P] Unit tests for HierarchicalKeyHandler in internal/tui/components/hierarchical_key_handler_test.go
- [ ] T111 [P] Unit tests for LayoutManager utility in internal/tui/components/layout_manager_test.go

### Documentation
- [ ] T112 [P] Update architecture documentation in docs/architecture/overview.md
- [ ] T113 [P] Update component documentation in docs/architecture/components.md
- [ ] T114 [P] Update CLAUDE.md with new architecture patterns
- [ ] T115 [P] Create component development guide in docs/development/component-guide.md

### Code Quality
- [ ] T116 Run golangci-lint on all modified files
- [ ] T117 Update code coverage report (target >80%)
- [ ] T118 Remove any dead code or unused imports
- [ ] T119 Format all code with gofmt
- [ ] T120 Final code review and cleanup

## Dependencies

**Critical Path** (must be sequential):
- Setup (T001-T005) before tests (T006-T017)
- Tests (T006-T017) MUST PASS before any migration (T018+)
- Foundation (T018-T022) before views (T023+)
- Simple views (T023-T037) before core views (T038-T055)
- Core views (T038-T055) before complex views (T056-T073)
- Complex views (T056-T073) before advanced features (T074-T079)
- Advanced features (T074-T079) before input system (T080-T087)
- Input system (T080-T087) before cleanup (T088-T092)
- Cleanup (T088-T092) before verification (T093-T098)
- Verification (T093-T098) before polish (T099-T120)

**Within-Phase Dependencies**:
- Each controller migration must be followed by its verification test
- Model cleanup (T088-T092) must happen AFTER all controllers removed
- Unit tests (T102-T111) can run after corresponding component created

**Parallel Opportunities**:
- Setup tasks (T001-T005) can all run in parallel
- Integration tests (T006-T015) can all be written in parallel
- Performance tests (T016-T017) can be written in parallel with integration tests
- Unit tests (T102-T111) can run in parallel after all components exist
- Documentation tasks (T112-T115) can run in parallel

## Parallel Example

```bash
# Launch T001-T005 together (setup tasks):
Task: "Create component interface documentation in internal/tui/components/README.md"
Task: "Create migration guide in docs/architecture/controller-migration.md"
Task: "Set up integration test framework with TTY allocation in test/integration/test_helpers.go"
Task: "Create component test utilities and mocks in test/integration/component_test_utils.go"
Task: "Set up performance testing framework in test/integration/performance_helpers.go"

# Launch T006-T015 together (integration tests):
Task: "Integration test for process management workflow in test/integration/process_management_test.go"
Task: "Integration test for log viewing and filtering in test/integration/log_viewing_test.go"
Task: "Integration test for error display and management in test/integration/error_management_test.go"
Task: "Integration test for URL discovery and display in test/integration/url_discovery_test.go"
Task: "Integration test for proxy request viewing in test/integration/proxy_viewing_test.go"
Task: "Integration test for AI coder session lifecycle in test/integration/ai_coder_test.go"
Task: "Integration test for command palette functionality in test/integration/command_palette_test.go"
Task: "Integration test for settings management in test/integration/settings_test.go"
Task: "Integration test for keyboard navigation (all global keys) in test/integration/keyboard_navigation_test.go"
Task: "Integration test for script selector functionality in test/integration/script_selector_test.go"

# Launch T102-T111 together (unit tests - after all components created):
Task: "Unit tests for ProcessesViewComponent in internal/tui/components/processes_view_test.go"
Task: "Unit tests for LogsViewComponent in internal/tui/components/logs_view_test.go"
Task: "Unit tests for ErrorsViewComponent in internal/tui/components/errors_view_test.go"
Task: "Unit tests for WebViewComponent in internal/tui/components/web_view_test.go"
Task: "Unit tests for SettingsViewComponent in internal/tui/components/settings_view_test.go"
Task: "Unit tests for CommandPaletteComponent in internal/tui/components/command_palette_test.go"
Task: "Unit tests for ScriptSelectorComponent in internal/tui/components/script_selector_test.go"
Task: "Unit tests for IntegratedShellComponent in internal/tui/components/integrated_shell_test.go"
Task: "Unit tests for HierarchicalKeyHandler in internal/tui/components/hierarchical_key_handler_test.go"
Task: "Unit tests for LayoutManager utility in internal/tui/components/layout_manager_test.go"

# Launch T112-T115 together (documentation):
Task: "Update architecture documentation in docs/architecture/overview.md"
Task: "Update component documentation in docs/architecture/components.md"
Task: "Update CLAUDE.md with new architecture patterns"
Task: "Create component development guide in docs/development/component-guide.md"
```

## Notes
- [P] tasks = different files, no dependencies
- Verify tests pass before migrating each controller (TDD requirement from Constitution)
- Commit after each controller migration
- Run integration tests after each migration to catch regressions early
- Avoid: vague tasks, same file conflicts
- Ensure all tasks follow lock-free architecture and event-driven design principles
- Performance target: <50ms key event response, no regressions
- Race condition testing required before final commit
- Each controller migration is ONE file at a time to minimize risk

## Migration Checklist (Per Controller)

For each controller removal (T018-T087):
1. Create or enhance target component
2. Migrate controller logic to component
3. Update service calls to be direct (no controller wrapper)
4. Implement BubbleTea Update() and View() methods
5. Add component to model (if new)
6. Remove controller file
7. Remove all controller references from model.go
8. Run corresponding integration test
9. Run manual verification
10. Commit with clear message

## Success Criteria

Migration is successful when:
- [ ] All 13 controller files deleted (event, layout, urls_view, mcp_debug, command_window, process_view, logs_view, errors_view, web_view, settings, script_selector, ai_coder, input)
- [ ] Zero references to deleted controllers in model.go (verified with grep)
- [ ] All integration tests pass (T006-T017)
- [ ] All unit tests pass (T102-T111)
- [ ] Performance tests pass (T016-T017, T099-T101)
- [ ] Race condition tests pass (`make test-race`)
- [ ] Manual testing passes all scenarios
- [ ] Code coverage >80%
- [ ] Documentation updated
- [ ] Code review approved

---
*Tasks generated according to Brummer Constitution v1.0.0 - Test-First Development required*
*Total: 120 tasks across 10 phases*
*Estimated effort: 3-4 weeks for systematic migration*
