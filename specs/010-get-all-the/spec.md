# Feature Specification: Complete Tab Functionality with Automated Testing

**Feature Branch**: `010-get-all-the`
**Created**: 2025-10-19
**Status**: Draft
**Input**: User description: "get all the process, error, log, proxy, settings, slash command tabs fully functional with automated service testing 100% respecting the architecture and design of the application"

## Execution Flow (main)
```
1. Parse user description from Input
   → Feature: Complete all TUI tabs with full functionality and automated tests
2. Extract key concepts from description
   → Actors: Developers, users monitoring processes
   → Actions: View processes, errors, logs, proxy status, settings, execute slash commands
   → Data: Process state, error logs, system logs, proxy configuration, app settings
   → Constraints: Must respect Redux/Elm architecture, lock-free patterns, automated testing
3. For each unclear aspect:
   → All tabs already exist in codebase
   → Architecture documented in CLAUDE.md
   → Clear requirements for testing and functionality
4. Fill User Scenarios & Testing section
   → User flows for each tab type
5. Generate Functional Requirements
   → Each tab must be fully functional
   → All tabs must have automated service tests
   → Must follow lock-free immutable state pattern
6. Identify Key Entities
   → Process data, error contexts, log entries, proxy config, settings
7. Run Review Checklist
   → Spec has no ambiguities - all tabs documented
   → No implementation details (follows spec template)
8. Return: SUCCESS (spec ready for planning)
```

---

## ⚡ Quick Guidelines
- ✅ Focus on WHAT users need and WHY
- ❌ Avoid HOW to implement (no tech stack, APIs, code structure)
- 👥 Written for business stakeholders, not developers

### Section Requirements
- **Mandatory sections**: Must be completed for every feature
- **Optional sections**: Include only when relevant to the feature
- When a section doesn't apply, remove it entirely (don't leave as "N/A")

---

## User Scenarios & Testing *(mandatory)*

### Primary User Story
As a developer using Brummer, I need all TUI tabs to be fully functional with reliable automated tests so that I can effectively monitor and control processes, view errors and logs, manage proxy settings, configure the application, and execute slash commands with confidence that the features work correctly and won't break in future updates.

### Acceptance Scenarios

#### Processes Tab
1. **Given** the application is running with active processes, **When** I switch to the Processes tab, **Then** I see a list of all running processes with their status, uptime, and resource usage
2. **Given** I'm viewing the Processes tab, **When** I select a process and press 'k', **Then** the process terminates and its status updates to "stopped"
3. **Given** I'm viewing the Processes tab, **When** I select a stopped process and press 'r', **Then** the process restarts and its status updates to "running"
4. **Given** processes are grouped by status, **When** I toggle the grouping option, **Then** processes reorganize by their running/stopped state

#### Errors Tab
1. **Given** processes have generated error output, **When** I switch to the Errors tab, **Then** I see all error messages with timestamps, process names, and error context
2. **Given** I'm viewing the Errors tab, **When** I filter by process name, **Then** only errors from that process are displayed
3. **Given** errors are cached, **When** new errors occur, **Then** they appear in real-time without requiring manual refresh
4. **Given** I'm viewing an error entry, **When** I press 'c' to clear, **Then** the error is removed from the display

#### Logs Tab
1. **Given** processes are generating log output, **When** I switch to the Logs tab, **Then** I see all log entries with severity levels, timestamps, and sources
2. **Given** I'm viewing the Logs tab, **When** I filter by severity level (info/warn/error), **Then** only matching log entries are displayed
3. **Given** I'm viewing the Logs tab, **When** I search for text, **Then** matching log entries are highlighted
4. **Given** I'm viewing the Logs tab, **When** I view the top row **Then** tabs with sources are displayed and can be selected to filter
4. **Given** I'm viewing the Logs tab, **When** I fiter by source process, **Then** matching log entries are filtered
5. **Given** logs are scrolling, **When** I pause auto-scroll, **Then** new logs accumulate without disrupting my view

#### Proxy Tab
1. **Given** the proxy server is running, **When** I switch to the Proxy tab, **Then** I see proxy status, active connections, and discovered endpoints
2. **Given** I'm viewing the Proxy tab, **When** I view endpoint details, **Then** I see URLs, request counts, and response times
3. **Given** proxy is in reverse mode, **When** I view shareable URLs, **Then** I see the public URLs for each discovered endpoint
4. **Given** I'm viewing proxy stats, **When** traffic flows through the proxy, **Then** statistics update in real-time

#### Settings Tab
1. **Given** I'm on the Settings tab, **When** I view configuration, **Then** I see all current settings with their sources (default/config file/environment)
2. **Given** I'm viewing settings, **When** I modify a setting, **Then** the change is validated and applied immediately
3. **Given** I've changed settings, **When** I save configuration, **Then** settings persist to the config file
4. **Given** I'm viewing settings, **When** I reset to defaults, **Then** all settings revert to their default values

#### Slash Command Tab (Command Window)
1. **Given** I'm viewing any tab, **When** I press '/', **Then** the command window opens with available slash commands
2. **Given** the command window is open, **When** I type a command name, **Then** matching commands are filtered and displayed
3. **Given** I've selected a command, **When** I execute it, **Then** the command runs and results are displayed
4. **Given** a command requires arguments, **When** I execute it without arguments, **Then** I receive clear feedback about required parameters

### Edge Cases
- What happens when a tab loads with no data (e.g., no processes running)?
- How does each tab handle extremely large datasets (1000+ log entries, 100+ processes)?
- What happens when switching tabs rapidly during data updates?
- How do tabs handle service failures (e.g., process manager unavailable)?
- What happens when cached data becomes stale?
- How do tabs recover from View() rendering errors?
- What happens when multiple updates occur simultaneously?

## Requirements *(mandatory)*

### Functional Requirements

#### General Tab Requirements
- **FR-001**: System MUST render all tabs (Processes, Errors, Logs, URLs, Proxy, Settings, Command Window) without UI freezes or deadlocks
- **FR-002**: System MUST support keyboard navigation between tabs with number keys (1-7) and Tab/Shift+Tab
- **FR-003**: System MUST update tab content in real-time as underlying data changes
- **FR-004**: System MUST cache tab state when switching between tabs to preserve scroll position and selections
- **FR-005**: System MUST follow the lock-free immutable state pattern for all tab state management

#### Processes Tab Requirements
- **FR-006**: System MUST display all running processes with name, status, PID, uptime, and resource usage
- **FR-007**: System MUST support process actions: start, stop, restart via keyboard commands
- **FR-008**: System MUST group processes by status (running/stopped) when grouping is enabled
- **FR-009**: System MUST show process selection with visual highlighting
- **FR-010**: System MUST update process list when processes start, stop, or change state

#### Errors Tab Requirements
- **FR-011**: System MUST display all error messages with timestamp, source process, and error context
- **FR-012**: System MUST support filtering errors by process name
- **FR-013**: System MUST support clearing individual or all errors
- **FR-014**: System MUST highlight new errors that arrive while viewing the tab
- **FR-015**: System MUST cache errors with configurable retention (e.g., last 1000 errors)

#### Logs Tab Requirements
- **FR-016**: System MUST display all log entries with timestamp, severity level, source, and message
- **FR-017**: System MUST support filtering by severity level (debug, info, warn, error)
- **FR-018**: System MUST support text search across log messages
- **FR-019**: System MUST support auto-scroll mode that pauses when user scrolls up
- **FR-020**: System MUST cache logs with configurable retention (e.g., last 5000 entries)

#### Proxy Tab Requirements
- **FR-021**: System MUST display proxy server status (enabled/disabled, port)
- **FR-022**: System MUST show discovered endpoints with URLs and metadata
- **FR-023**: System MUST display active proxy connections with source, destination, and status
- **FR-024**: System MUST show traffic statistics (requests, bytes transferred, response times)
- **FR-025**: System MUST display shareable reverse proxy URLs when reverse mode is enabled

#### Settings Tab Requirements
- **FR-026**: System MUST display all configuration settings organized by category
- **FR-027**: System MUST show configuration sources for each setting (default/file/environment)
- **FR-028**: System MUST support modifying settings through the UI
- **FR-029**: System MUST validate setting changes before applying them
- **FR-030**: System MUST persist setting changes to the configuration file
- **FR-031**: System MUST support resetting settings to default values

#### Slash Command Tab Requirements
- **FR-032**: System MUST display command palette when '/' key is pressed
- **FR-033**: System MUST show all available slash commands with intelligent context-aware autocomplete
- **FR-034**: System MUST filter commands as user types with prefix matching (case-insensitive)
- **FR-035**: System MUST execute selected commands with proper argument handling and validation
- **FR-036**: System MUST display command results or errors clearly with appropriate styling
- **FR-037**: System MUST support tab completion for multi-level commands (command + arguments)
- **FR-038-CMD**: System MUST provide autocomplete suggestions based on command context:
  - `/run` - Show available scripts excluding already running ones
  - `/restart`, `/stop` - Show "all" plus currently running processes
  - `/clear` - Show "all", "logs", "errors", "web", plus all script names
  - `/show`, `/hide` - Show common log filter patterns
  - `/proxy` - Show subcommands and URL suggestions
  - `/ai` - Show configured AI providers
  - `/help`, `/exit`, `/quit` - No additional arguments
- **FR-039-CMD**: System MUST support keyboard navigation in autocomplete:
  - Tab: Accept selected suggestion and add trailing space
  - Up/Down arrows: Navigate through suggestions
  - Enter: Execute complete command
  - Esc: Cancel and close command palette
- **FR-040-CMD**: System MUST validate commands before execution and show error messages for invalid input
- **FR-041-CMD**: System MUST display suggestions in a dropdown with:
  - Visual selection indicator (▶)
  - Highlighted selected item
  - Scrollable list for many suggestions (max 10 visible)
  - Dynamic filtering as user types

#### Automated Testing Requirements
- **FR-038**: System MUST have automated service tests for all tab state management
- **FR-039**: System MUST have automated tests verifying View() functions are non-blocking
- **FR-040**: System MUST have automated tests for all tab update operations (state mutations)
- **FR-041**: System MUST have automated tests for tab switching without state loss
- **FR-042**: System MUST have automated race condition tests for concurrent tab updates
- **FR-043**: System MUST have automated tests verifying lock-free immutable state patterns
- **FR-044**: System MUST have automated integration tests for tab data refresh cycles
- **FR-045**: System MUST achieve >80% code coverage for all tab-related code

#### Architecture Compliance Requirements
- **FR-046**: System MUST follow Redux/Elm architecture pattern (Model → Update → View unidirectional flow)
- **FR-047**: System MUST use stateless services that return tea.Cmd for async operations
- **FR-048**: System MUST implement all state updates in Update() method only, never in View()
- **FR-049**: System MUST use immutable state objects with zero mutexes
- **FR-050**: System MUST implement ViewWithState() methods that are pure and non-blocking

### Key Entities

- **ProcessState**: Represents the state of the Processes tab, including process list, selected process, grouping settings, and cache timestamp
- **ErrorsState**: Represents the state of the Errors tab, including error contexts, filters, selected error, and cache validity
- **LogsState**: Represents the state of the Logs tab, including log entries, severity filters, search query, scroll position, and auto-scroll mode
- **URLsState**: Represents discovered URLs from process logs with deduplication per process
- **ProxyState**: Represents proxy server status, discovered endpoints, active connections, and traffic statistics
- **SettingsState**: Represents application configuration with setting values, sources, and modification tracking
- **CommandWindowState**: Represents slash command palette state, including available commands, filter query, selected command, and execution results
- **MasterData**: Central state object containing all tab states, following single source of truth pattern

---

## Review & Acceptance Checklist
*GATE: Automated checks run during main() execution*

### Content Quality
- [X] No implementation details (languages, frameworks, APIs)
- [X] Focused on user value and business needs
- [X] Written for non-technical stakeholders
- [X] All mandatory sections completed

### Requirement Completeness
- [X] No [NEEDS CLARIFICATION] markers remain
- [X] Requirements are testable and unambiguous
- [X] Success criteria are measurable (>80% coverage, all tabs functional, 0 deadlocks)
- [X] Scope is clearly bounded (7 tabs, automated tests, architecture compliance)
- [X] Dependencies and assumptions identified (existing lock-free architecture)

---

## Execution Status
*Updated by main() during processing*

- [X] User description parsed
- [X] Key concepts extracted
- [X] Ambiguities marked (none - all tabs exist, architecture documented)
- [X] User scenarios defined
- [X] Requirements generated
- [X] Entities identified
- [X] Review checklist passed

---

## Dependencies and Assumptions

### Dependencies
- Existing lock-free Redux/Elm architecture (completed in 009-make-the-application)
- Lock-free developer guidelines documentation
- State management infrastructure (MasterData, StateManager)
- Existing tab components (processes_view.go, errors_view.go, logs_view.go, etc.)
- Service layer infrastructure (stateless services with tea.Cmd pattern)
- Testing infrastructure (go test framework, race detector)

### Assumptions
- All tab component files already exist in `internal/tui/components/`
- State files already exist in `internal/tui/state/`
- BubbleTea framework is already integrated
- EventBus for inter-component communication is available
- Process manager, log store, and proxy server are functional
- Configuration system (TOML-based) is operational
- Slash command system infrastructure exists

### Success Metrics
- **Coverage**: >80% code coverage for all tab-related code
- **Performance**: <5ms View() rendering time for all tabs
- **Reliability**: Zero race conditions detected in `make test-race`
- **Architecture**: 100% compliance with lock-free immutable state pattern
- **Testing**: 100% of tabs have automated service tests
- **Deadlocks**: Zero deadlocks possible (no mutexes in state system)

---

## Appendix: Slash Command System Details

### Available Slash Commands

#### Process Management Commands

1. **`/run <script-name>`**
   - **Purpose**: Start a new process from package.json scripts
   - **Autocomplete**: Shows available scripts excluding already running ones
   - **Behavior**: Detects scripts from package.json, filters out running processes
   - **Example**: `/run dev`, `/run test`, `/run build`

2. **`/restart <process-name|all>`**
   - **Purpose**: Restart one or all running processes
   - **Autocomplete**: Shows "all" + list of currently running process names
   - **Behavior**: Stops and starts the specified process(es)
   - **Example**: `/restart dev`, `/restart all`

3. **`/stop <process-name|all>`**
   - **Purpose**: Stop one or all running processes
   - **Autocomplete**: Shows "all" + list of currently running process names
   - **Behavior**: Terminates the specified process(es)
   - **Example**: `/stop dev`, `/stop all`

#### Display Management Commands

4. **`/clear <target>`**
   - **Purpose**: Clear logs, errors, or web view data
   - **Autocomplete**: Shows "all", "logs", "errors", "web", plus all script names
   - **Behavior**: Clears specified data type
   - **Example**: `/clear logs`, `/clear all`, `/clear dev`

5. **`/show <pattern>`**
   - **Purpose**: Show log lines matching a pattern (regex)
   - **Autocomplete**: Common patterns: "error", "warn", "info", "debug", "^\\[", "\\]$", "|"
   - **Behavior**: Filters log display to show matching lines
   - **Example**: `/show error`, `/show "^\\[ERROR\\]"`

6. **`/hide <pattern>`**
   - **Purpose**: Hide log lines matching a pattern (regex)
   - **Autocomplete**: Common patterns: "error", "warn", "info", "debug", "^\\[", "\\]$", "|"
   - **Behavior**: Filters log display to hide matching lines
   - **Example**: `/hide debug`, `/hide "\\[INFO\\]"`

#### Proxy Commands

7. **`/proxy <subcommand|url>`**
   - **Purpose**: Manage proxy server
   - **Autocomplete**: "list", "http://localhost:3000", "http://localhost:8080", "https://example.com"
   - **Behavior**: Proxy configuration and URL forwarding
   - **Example**: `/proxy list`, `/proxy http://localhost:3000`

8. **`/toggle-proxy`**
   - **Purpose**: Toggle proxy server on/off
   - **Autocomplete**: None (no arguments)
   - **Behavior**: Enables/disables proxy server
   - **Example**: `/toggle-proxy`

#### AI Coder Commands

9. **`/ai <provider>`**
   - **Purpose**: Open AI coder session with specified provider
   - **Autocomplete**: "claude", "claude-3-7", "claude-o3" (from configuration)
   - **Behavior**: Starts interactive AI coding session
   - **Example**: `/ai claude`, `/ai claude-3-7`

10. **`/term`**
    - **Purpose**: Open terminal/PTY session
    - **Autocomplete**: None (no arguments)
    - **Behavior**: Opens interactive terminal
    - **Example**: `/term`

#### Utility Commands

11. **`/help`**
    - **Purpose**: Display help information
    - **Autocomplete**: None (no arguments)
    - **Behavior**: Shows command reference
    - **Example**: `/help`

12. **`/exit` or `/quit`**
    - **Purpose**: Exit the application
    - **Autocomplete**: None (no arguments)
    - **Behavior**: Gracefully shuts down all processes and exits
    - **Example**: `/exit`, `/quit`

### Autocomplete Behavior Details

#### Multi-Level Completion
The autocomplete system supports multi-level completion where:
1. **First segment (command)**: Matches against root commands
2. **Subsequent segments (arguments)**: Context-aware based on the command

#### Filtering Algorithm
- **Prefix matching**: Case-insensitive prefix search
- **Dynamic**: Updates as user types each character
- **Smart**: Shows only relevant suggestions based on command context

#### Visual Indicators
- **Selected item**: Shows "▶" prefix and highlighted background
- **Unselected items**: Shows "  " prefix and normal foreground
- **Scroll indicator**: Shows limited suggestions (max 10) with implicit scrolling

#### Keyboard Interaction
- **Tab**: Autocompletes selected suggestion, adds trailing space for multi-arg commands
- **Up/Down**: Navigate through suggestion list (wraps at boundaries)
- **Enter**: Validates and executes the complete command
- **Esc**: Cancels and closes the command palette
- **Typing**: Filters suggestions in real-time

#### Context-Aware Filtering

**Example: `/run` command**
```
Input: "/run d"
Autocomplete shows: ["dev", "debug", "deploy"] (only non-running scripts starting with 'd')
```

**Example: `/restart` command**
```
Input: "/restart "
Autocomplete shows: ["all", "dev", "api", "worker"] (all + running processes)
```

**Example: `/clear` command**
```
Input: "/clear "
Autocomplete shows: ["all", "logs", "errors", "web", "dev", "api"] (types + all scripts)
```

#### Process-Aware Suggestions
The autocomplete integrates with the process manager to:
- **Exclude running processes** from `/run` suggestions
- **Include only running processes** in `/stop`, `/restart` suggestions
- **Show all processes** regardless of state in `/clear` suggestions

#### Error Handling
- **Invalid command**: Shows error message "Unknown command: <name>"
- **Missing arguments**: Shows error message "Command requires arguments"
- **Validation failure**: Shows specific validation error message
- **Empty input**: Shows all root commands as suggestions

### Command Validation Rules

1. **Command must start with '/'** - Enforced by prompt
2. **Command name must be recognized** - Validated against root commands list
3. **Arguments must match expected count** - Validated per command
4. **Process names must exist** - Validated for process-specific commands
5. **Regex patterns must be valid** - Validated for show/hide commands

---
