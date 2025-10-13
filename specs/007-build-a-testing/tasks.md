# Tasks: TUI Integration Testing Harness

**Input**: Design documents from `/specs/007-build-a-testing/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/, quickstart.md
**Total Tasks**: 29

## Execution Flow (main)
```
1. Load plan.md from feature directory → Extract: Go 1.21+, BubbleTea, testing framework
2. Load design documents:
   → data-model.md: Extract TestHarness, InputSimulator, StateValidator, TestReporter entities
   → contracts/testing-harness-api.md: Extract API interfaces and endpoints
   → research.md: Extract custom solution using BubbleTea testing patterns
   → quickstart.md: Extract test scenarios and integration examples
3. Generate tasks by category:
   → Setup: PTY layer, project structure, dependencies
   → Tests: contract tests, integration tests with PTY
   → Core: PTY layer, models, services, harness components
   → Integration: TUI components, pre-commit hooks, scripts
   → Polish: performance, documentation, CI/CD
4. Apply task rules:
   → Different files = mark [P] for parallel
   → Same file = sequential (no [P])
   → Tests before implementation (TDD)
5. Number tasks sequentially (T001, T002...)
6. Generate dependency graph
7. Create parallel execution examples
8. Validate task completeness
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Phase 3.1: Setup
- [x] T001 Create TUI testing project structure per implementation plan
- [x] T002 Initialize Go project with BubbleTea and testing dependencies
- [x] T003 [P] Configure linting and formatting tools for TUI testing

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**
- [x] T004 [P] Contract test TestHarness interface in test/tui/harness/test_harness_contract_test.go
- [x] T005 [P] Contract test InputSimulator interface in test/tui/harness/test_input_simulator_contract_test.go
- [x] T006 [P] Contract test StateValidator interface in test/tui/harness/test_state_validator_contract_test.go
- [x] T007 [P] Contract test TestReporter interface in test/tui/harness/test_test_reporter_contract_test.go
- [x] T008 [P] Integration test PTY layer functionality in test/tui/pty/test_pty_layer_integration_test.go
- [x] T009 [P] Integration test keyboard input through PTY in test/tui/integration/test_keyboard_input_pty_test.go
- [x] T010 [P] Integration test view switching via PTY in test/tui/integration/test_view_switching_pty_test.go
- [x] T011 [P] Integration test command palette through PTY in test/tui/integration/test_command_palette_pty_test.go

## Phase 3.3: Core Implementation (ONLY after tests are failing)
- [x] T012 [P] PTY layer implementation in test/tui/pty/pty_layer.go
- [x] T013 [P] TestHarness model in test/tui/harness/test_harness.go
- [x] T014 [P] InputSimulator model in test/tui/harness/input_simulator.go
- [x] T015 [P] StateValidator model in test/tui/harness/state_validator.go
- [x] T016 [P] TestReporter model in test/tui/harness/test_reporter.go
- [x] T017 [P] PTY integration components in test/tui/pty/pty_integration.go
- [x] T018 [P] TestHarnessConfiguration model in test/tui/harness/configuration.go
- [x] T019 [P] KeyMapping model in test/tui/harness/key_mapping.go
- [x] T020 [P] ExpectedState model in test/tui/harness/expected_state.go
- [x] T021 [P] TestResult model in test/tui/harness/test_result.go

## Phase 3.4: Integration
- [x] T022 Connect PTY layer to BubbleTea program
- [x] T023 Integrate with existing Brummer InputService
- [x] T024 Integrate with existing Brummer NavigationService
- [x] T025 Create pre-commit hook in githooks/pre-commit-tui-tests.sh
- [x] T026 Create test execution script in test/scripts/run_tui_tests.sh
- [x] T027 Create test validation script in test/scripts/validate_tui.sh
- [x] T028 Create test environment setup in test/scripts/setup_test_env.sh

## Phase 3.5: Polish
- [x] T029 [P] Create comprehensive documentation in docs/tui-testing-guide.md

## Dependencies
- Tests (T004-T011) before implementation (T012-T021)
- T012 blocks T013-T017
- T022 blocks T023-T024
- T025-T028 before T029

## Parallel Example
```
# Launch T004-T011 together (all contract/integration tests):
Task: "Contract test TestHarness interface in test/tui/harness/test_harness_contract_test.go"
Task: "Contract test InputSimulator interface in test/tui/harness/test_input_simulator_contract_test.go"
Task: "Contract test StateValidator interface in test/tui/harness/test_state_validator_contract_test.go"
Task: "Contract test TestReporter interface in test/tui/harness/test_test_reporter_contract_test.go"
Task: "Integration test PTY layer functionality in test/tui/pty/test_pty_layer_integration_test.go"
Task: "Integration test keyboard input through PTY in test/tui/integration/test_keyboard_input_pty_test.go"
Task: "Integration test view switching via PTY in test/tui/integration/test_view_switching_pty_test.go"
Task: "Integration test command palette through PTY in test/tui/integration/test_command_palette_pty_test.go"

# Launch T012-T021 together (all model implementations in different files):
Task: "PTY layer implementation in test/tui/pty/pty_layer.go"
Task: "TestHarness model in test/tui/harness/test_harness.go"
Task: "InputSimulator model in test/tui/harness/input_simulator.go"
Task: "StateValidator model in test/tui/harness/state_validator.go"
Task: "TestReporter model in test/tui/harness/test_reporter.go"
Task: "PTY integration components in test/tui/pty/pty_integration.go"
Task: "TestHarnessConfiguration model in test/tui/harness/configuration.go"
Task: "KeyMapping model in test/tui/harness/key_mapping.go"
Task: "ExpectedState model in test/tui/harness/expected_state.go"
Task: "TestResult model in test/tui/harness/test_result.go"
```

## Critical Path
**T001 → T002 → T003 → T004 → T005 → T006 → T007 → T008 → T009 → T010 → T011 → T012 → T013 → T014 → T015 → T016 → T017 → T018 → T019 → T020 → T021 → T022 → T023 → T024 → T025 → T026 → T027 → T028 → T029**

## Task Details

### Phase 3.1: Setup Tasks

**T001**: Create TUI testing project structure per implementation plan
- Create test/tui/ directory structure
- Create test/tui/harness/, test/tui/integration/, test/tui/pty/, test/tui/fixtures/
- Create test/scripts/ directory
- Ensure following Brummer single project structure

**T002**: Initialize Go project with BubbleTea and testing dependencies
- Add testing dependencies to go.mod
- Ensure BubbleTea framework compatibility
- Add PTY-related dependencies (github.com/creack/pty)
- Configure build tags for integration tests

**T003**: Configure linting and formatting tools for TUI testing
- Set up golangci-lint configuration for TUI testing packages
- Add formatting rules for test files
- Configure static analysis for PTY-related code

### Phase 3.2: Tests First (TDD)

**T004**: Contract test TestHarness interface in test/tui/harness/test_harness_contract_test.go
- Test all TestHarness interface methods
- Verify lifecycle management (Initialize, Start, Stop)
- Test component access methods
- Ensure tests fail initially (no implementation)

**T005**: Contract test InputSimulator interface in test/tui/harness/test_input_simulator_contract_test.go
- Test keyboard input generation methods
- Verify timing configuration
- Test key mapping functionality
- Ensure PTY compatibility

**T006**: Contract test StateValidator interface in test/tui/harness/test_state_validator_contract_test.go
- Test state capture and validation
- Verify expected state management
- Test validation rules functionality
- Test state comparison methods

**T007**: Contract test TestReporter interface in test/tui/harness/test_test_reporter_contract_test.go
- Test result management methods
- Verify report generation
- Test metrics calculation
- Test export functionality

**T008**: Integration test PTY layer functionality in test/tui/pty/test_pty_layer_integration_test.go
- Test PTY allocation and management
- Verify PTY-to-TTY bridge functionality
- Test PTY cleanup
- Test error handling for PTY failures

**T009**: Integration test keyboard input through PTY in test/tui/integration/test_keyboard_input_pty_test.go
- Test keyboard input simulation through PTY
- Verify tea.KeyMsg generation via PTY
- Test input timing and delays
- Test complex keyboard sequences

**T010**: Integration test view switching via PTY in test/tui/integration/test_view_switching_pty_test.go
- Test view switching through PTY layer
- Verify number key functionality (1-5)
- Test Tab key cycling
- Validate view state changes

**T011**: Integration test command palette through PTY in test/tui/integration/test_command_palette_pty_test.go
- Test command palette opening via PTY
- Verify command input and execution
- Test command window state management
- Test keyboard navigation in command interface

### Phase 3.3: Core Implementation

**T012**: PTY layer implementation in test/tui/pty/pty_layer.go
- Implement PTY allocation and management
- Create PTY-to-TTY bridge for terminal emulation
- Build PTY cleanup and resource management
- Handle PTY allocation failures gracefully

**T013**: TestHarness model in test/tui/harness/test_harness.go
- Implement TestHarness interface
- Create lifecycle management methods
- Build component coordination
- Add PTY integration support

**T014**: InputSimulator model in test/tui/harness/input_simulator.go
- Implement InputSimulator interface
- Create keyboard input generation
- Build timing configuration
- Add PTY-based input simulation

**T015**: StateValidator model in test/tui/harness/state_validator.go
- Implement StateValidator interface
- Create state capture and validation
- Build expected state management
- Add validation rules functionality

**T016**: TestReporter model in test/tui/harness/test_reporter.go
- Implement TestReporter interface
- Create result management system
- Build report generation
- Add metrics calculation

**T017**: PTY integration components in test/tui/pty/pty_integration.go
- Build PTYProgram wrapper for tea.Program
- Create PTYInputController for keyboard simulation
- Implement PTYRenderer for output capture
- Build PTYBridge for event conversion

**T018**: TestHarnessConfiguration model in test/tui/harness/configuration.go
- Implement configuration structure
- Add PTY-specific settings
- Create validation methods
- Build default configurations

**T019**: KeyMapping model in test/tui/harness/key_mapping.go
- Implement key mapping structure
- Add PTY-specific key mappings
- Create validation methods
- Build mapping utilities

**T020**: ExpectedState model in test/tui/harness/expected_state.go
- Implement expected state structure
- Add PTY-specific state validation
- Create timeout handling
- Build state comparison methods

**T021**: TestResult model in test/tui/harness/test_result.go
- Implement test result structure
- Add PTY-specific result data
- Create status management
- Build error handling

### Phase 3.4: Integration

**T022**: Connect PTY layer to BubbleTea program
- Integrate PTY layer with tea.Program execution
- Create PTY-based program startup
- Add PTY size control and terminal emulation
- Handle PTY program termination

**T023**: Integrate with existing Brummer InputService
- Connect PTY input simulation to InputService
- Test keyboard routing through PTY
- Validate InputService integration
- Ensure backward compatibility

**T024**: Integrate with existing Brummer NavigationService
- Connect PTY view switching to NavigationService
- Test navigation through PTY input
- Validate NavigationService integration
- Ensure consistent state management

**T025**: Create pre-commit hook in githooks/pre-commit-tui-tests.sh
- Implement PTY-based test execution
- Add test result validation
- Create commit blocking on failures
- Add PTY environment detection

**T026**: Create test execution script in test/scripts/run_tui_tests.sh
- Implement comprehensive test suite execution
- Add PTY environment setup
- Create test result collection
- Add performance metrics

**T027**: Create test validation script in test/scripts/validate_tui.sh
- Implement TUI functionality validation
- Add PTY layer validation
- Create keyboard input validation
- Add component integration validation

**T028**: Create test environment setup in test/scripts/setup_test_env.sh
- Implement PTY test environment configuration
- Add test data preparation
- Create test isolation procedures
- Add environment validation

### Phase 3.5: Polish

**T029**: Create comprehensive documentation in docs/tui-testing-guide.md
- Document PTY-based testing approach
- Create troubleshooting guides
- Add best practices documentation
- Include example test scenarios

## Validation Checklist
- [x] All contracts have corresponding tests (T004-T007)
- [x] All entities have model tasks (T013-T021)
- [x] All tests come before implementation
- [x] Parallel tasks truly independent (different files)
- [x] Each task specifies exact file path
- [x] No task modifies same file as another [P] task
- [x] PTY layer requirements fully addressed
- [x] Integration with existing Brummer architecture included

## Notes
- [P] tasks = different files, no dependencies
- Verify tests fail before implementing (TDD approach)
- Commit after each task
- PTY layer is critical for agent-based testing environments
- Focus on preventing TTY allocation false positives
- Ensure real TUI code path testing through PTY simulation

---
*Tasks generated according to Brummer Constitution v1.0.0 - Test-First Development required*