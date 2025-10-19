# Tasks: Code Cleanup and Refactoring for Key Handling & AI Coder Support

**Input**: Design documents from `/specs/001-clean-up-unused/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/, quickstart.md

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- **Single project**: `src/`, `tests/` at repository root
- **Web app**: `backend/src/`, `frontend/src/`
- **Mobile**: `api/src/`, `ios/src/` or `android/src/`
- Paths shown below assume single project - adjust based on plan.md structure

## Phase 3.1: Setup
- [X] T001 Create project structure for code analysis and AI coder components
- [X] T002 Initialize Go modules with required dependencies (BubbleTea, EventBus, etc.)
- [X] T003 [P] Configure linting and formatting tools (golangci-lint, gofmt)

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**
- [X] T004 [P] Contract test POST /api/v1/analysis in tests/contract/test_code_analysis_api.go
- [X] T005 [P] Contract test GET /api/v1/analysis/{id} in tests/contract/test_code_analysis_api.go
- [X] T006 [P] Contract test POST /api/v1/cleanup in tests/contract/test_code_analysis_api.go
- [X] T007 [P] Contract test POST /api/v1/ai/sessions in tests/contract/test_ai_coder_api.go
- [X] T008 [P] Contract test POST /api/v1/ai/sessions/{id}/terminal in tests/contract/test_ai_coder_api.go
- [X] T009 [P] Contract test POST /api/v1/ai/sessions/{id}/browser in tests/contract/test_ai_coder_api.go
- [X] T010 [P] Integration test code analysis workflow in tests/integration/test_code_analysis_workflow.go
- [X] T011 [P] Integration test AI coder session lifecycle in tests/integration/test_ai_coder_session.go
- [X] T012 [P] Integration test key handler consistency in tests/integration/test_key_handling.go
- [X] T013 [P] TTY allocation test for key handling validation in tests/integration/test_key_handling_tty.go

## Phase 3.3: Core Implementation (ONLY after tests are failing)
- [X] T014 [P] CodeAnalysisResult model in internal/models/analysis_result.go
- [X] T015 [P] KeyEventState model in internal/models/key_event_state.go
- [X] T016 [P] AICoderSession model in internal/models/ai_coder_session.go
- [X] T017 [P] PerformanceProfile model in internal/models/performance_profile.go
- [X] T018 Static analysis engine for unused imports detection in internal/analysis/import_analyzer.go
- [X] T019 Unreachable function identification algorithm in internal/analysis/function_analyzer.go
- [X] T020 Dead code block detection and reporting in internal/analysis/dead_code_detector.go
- [X] T021 Global key handler priority system in internal/tui/key_handler_manager.go
- [X] T022 Event queue management for key events in internal/tui/key_event_queue.go
- [X] T023 Terminal emulator component for AI coder in internal/ai/terminal_emulator.go
- [X] T024 Browser debug tool reverse proxy connections in internal/ai/browser_proxy.go
- [X] T025 Error streaming from browser/server to TUI in internal/ai/error_stream.go
- [X] T026 AI session lifecycle management in internal/ai/session_manager.go

## Phase 3.4: API Implementation
- [X] T027 POST /api/v1/analysis endpoint in internal/api/analysis_handler.go
- [X] T028 GET /api/v1/analysis/{id} endpoint in internal/api/analysis_handler.go
- [X] T029 POST /api/v1/cleanup endpoint in internal/api/cleanup_handler.go
- [X] T030 POST /api/v1/ai/sessions endpoint in internal/api/ai_session_handler.go
- [X] T031 POST /api/v1/ai/sessions/{id}/terminal endpoint in internal/api/ai_session_handler.go
- [X] T032 POST /api/v1/ai/sessions/{id}/browser endpoint in internal/api/ai_session_handler.go
- [X] T033 GET /api/v1/ai/sessions/{id}/errors endpoint in internal/api/ai_session_handler.go

## Phase 3.5: Integration
- [X] T034 Connect code analysis models to API handlers
- [X] T035 Integrate key handler manager with TUI BubbleTea model
- [X] T036 Connect AI terminal emulator to existing TUI framework
- [X] T037 Integrate browser proxy with existing MCP server
- [X] T038 Request/response logging for all new endpoints
- [X] T039 CORS and security headers for API endpoints
- [X] T040 Error handling and logging for new components

## Phase 3.6: CLI Commands
- [X] T041 brum analyze command implementation in cmd/brum/commands/analyze.go
- [X] T042 brum cleanup command implementation in cmd/brum/commands/cleanup.go
- [X] T043 brum ai coder command implementation in cmd/brum/commands/ai_coder.go
- [X] T044 brum test-keys command implementation in cmd/brum/commands/test_keys.go
- [X] T045 brum profile command implementation in cmd/brum/commands/profile.go

## Phase 3.7: Performance Optimization
- [X] T046 Profiling instrumentation setup using Go pprof in internal/profiling/profiler.go
- [X] T047 Key event latency measurement in internal/profiling/key_latency.go
- [X] T048 Memory usage monitoring for analysis operations in internal/profiling/memory_monitor.go
- [X] T049 Hotspot detection and optimization utilities in internal/profiling/hotspot_detector.go

## Phase 3.8: Polish
- [X] T050 [P] Unit tests for CodeAnalysisResult in tests/unit/test_analysis_result.go
- [X] T051 [P] Unit tests for KeyEventState in tests/unit/test_key_event_state.go
- [X] T052 [P] Unit tests for AICoderSession in tests/unit/test_ai_coder_session.go
- [X] T053 [P] Unit tests for static analysis components in tests/unit/test_import_analyzer.go
- [X] T054 [P] Unit tests for key handler components in tests/unit/test_key_handler_manager.go
- [X] T055 [P] Unit tests for AI terminal emulator in tests/unit/test_terminal_emulator.go
- [X] T056 Performance tests for key event handling (<50ms response) in tests/performance/test_key_performance.go
- [X] T057 Performance tests for code analysis operations in tests/performance/test_analysis_performance.go
- [X] T058 [P] Update API documentation in docs/api/code-analysis.md
- [X] T059 [P] Update API documentation in docs/api/ai-coder.md
- [X] T060 [P] Update CLI help documentation and man pages
- [X] T061 Remove duplicate code and consolidate shared utilities
- [X] T062 Run integration tests from quickstart.md scenarios

## Dependencies
- Tests (T004-T013) before implementation (T014-T049)
- Models (T014-T017) block analysis components (T018-T020)
- Analysis components (T018-T020) block API handlers (T027-T029)
- Key handler components (T021-T022) block TUI integration (T035)
- AI components (T023-T026) block API handlers (T030-T033)
- API handlers block CLI commands (T041-T045)
- Core implementation before integration (T034-T040)
- Integration before performance optimization (T046-T049)
- Performance optimization before polish (T050-T062)

## Parallel Example
```
# Launch T004-T013 together (contract and integration tests):
Task: "Contract test POST /api/v1/analysis in tests/contract/test_code_analysis_api.go"
Task: "Contract test GET /api/v1/analysis/{id} in tests/contract/test_code_analysis_api.go"
Task: "Contract test POST /api/v1/cleanup in tests/contract/test_code_analysis_api.go"
Task: "Contract test POST /api/v1/ai/sessions in tests/contract/test_ai_coder_api.go"
Task: "Contract test POST /api/v1/ai/sessions/{id}/terminal in tests/contract/test_ai_coder_api.go"
Task: "Contract test POST /api/v1/ai/sessions/{id}/browser in tests/contract/test_ai_coder_api.go"
Task: "Integration test code analysis workflow in tests/integration/test_code_analysis_workflow.go"
Task: "Integration test AI coder session lifecycle in tests/integration/test_ai_coder_session.go"
Task: "Integration test key handler consistency in tests/integration/test_key_handling.go"
Task: "TTY allocation test for key handling validation in tests/integration/test_key_handling_tty.go"

# Launch T014-T017 together (model creation):
Task: "CodeAnalysisResult model in internal/models/analysis_result.go"
Task: "KeyEventState model in internal/models/key_event_state.go"
Task: "AICoderSession model in internal/models/ai_coder_session.go"
Task: "PerformanceProfile model in internal/models/performance_profile.go"

# Launch T050-T056 together (unit tests for different components):
Task: "Unit tests for CodeAnalysisResult in tests/unit/test_analysis_result.go"
Task: "Unit tests for KeyEventState in tests/unit/test_key_event_state.go"
Task: "Unit tests for AICoderSession in tests/unit/test_ai_coder_session.go"
Task: "Unit tests for static analysis components in tests/unit/test_import_analyzer.go"
Task: "Unit tests for key handler components in tests/unit/test_key_handler_manager.go"
Task: "Unit tests for AI terminal emulator in tests/unit/test_terminal_emulator.go"

# Launch T058-T060 together (documentation updates):
Task: "Update API documentation in docs/api/code-analysis.md"
Task: "Update API documentation in docs/api/ai-coder.md"
Task: "Update CLI help documentation and man pages"
```

## Notes
- [P] tasks = different files, no dependencies
- Verify tests fail before implementing (TDD requirement from Constitution)
- Commit after each task
- Avoid: vague tasks, same file conflicts
- Ensure all tasks follow lock-free architecture and event-driven design principles
- Performance targets: <50ms key event response, profiling-driven optimization

---
*Tasks generated according to Brummer Constitution v1.0.0 - Test-First Development required*
---

## ✅ COMPLETED - MERGED TO MAIN

**Date**: 2025-10-19
**Status**: All tasks completed and integrated into main branch
**Architecture**: Lock-free Redux/Elm pattern with immutable states
**Branch**: Merged via 009-make-the-application → main

All work from this specification has been successfully integrated into the production-ready lock-free architecture now on the main branch.
