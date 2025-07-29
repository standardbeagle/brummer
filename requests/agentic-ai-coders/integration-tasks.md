# AI Coder PTY Integration - Task Breakdown

## Task 1: Create TUI Data Provider Implementation
**Priority**: Critical - Foundation for PTY/TUI integration
**Estimated Time**: 1-2 hours

### Implementation Steps
1. Create `internal/tui/brummer_data_provider_impl.go`
2. Implement thread-safe access to TUI model data:
   ```go
   type TUIDataProvider struct {
       model *Model
   }
   
   func (p *TUIDataProvider) GetLastError() *logs.ErrorContext {
       // Access model.errorDetector safely
   }
   ```
3. Handle nil checks and synchronization
4. Add factory method in Model: `NewDataProvider() aicoder.BrummerDataProvider`

### Acceptance Criteria
- [ ] All interface methods implemented
- [ ] Thread-safe access to model data
- [ ] No race conditions in concurrent access
- [ ] Unit tests for each method

---

## Task 2: Update TUI Model for PTY Support
**Priority**: Critical - Core integration point
**Estimated Time**: 2-3 hours

### Implementation Steps
1. Add to `Model` struct:
   ```go
   // PTY support
   aiCoderPTYView   *AICoderPTYView
   ptyManager       *aicoder.PTYManager
   ptyEventSub      chan aicoder.PTYEvent
   ```

2. Update `NewModel()`:
   ```go
   // Create PTY manager with data provider
   dataProvider := NewTUIDataProvider(m)
   m.ptyManager = aicoder.NewPTYManager(dataProvider, m.eventBus)
   m.aiCoderPTYView = NewAICoderPTYView(m.ptyManager)
   ```

3. Add PTY event subscription in `Init()`
4. Handle PTY cleanup in shutdown

### Acceptance Criteria
- [ ] Model compiles with PTY fields
- [ ] PTY manager initialized correctly
- [ ] Clean shutdown without leaks
- [ ] Event subscriptions working

---

## Task 3: Replace AI Coder View Rendering
**Priority**: High - User-visible change
**Estimated Time**: 1 hour

### Implementation Steps
1. Update `View()` method in model.go:
   ```go
   case 8: // AI Coders
       if m.aiCoderPTYView != nil {
           return m.renderAICoderPTYView()
       }
       return m.renderAICoderView() // Fallback
   ```

2. Create `renderAICoderPTYView()`:
   ```go
   func (m *Model) renderAICoderPTYView() string {
       content := m.aiCoderPTYView.View()
       return m.renderTabContent("AI Coders", content)
   }
   ```

3. Pass window size updates to PTY view
4. Handle focus state for terminal input

### Acceptance Criteria
- [ ] AI Coder tab shows PTY view
- [ ] Terminal renders correctly
- [ ] Window resize handled
- [ ] Tab styling consistent

---

## Task 4: Create PTY Event Bridge
**Priority**: High - Required for real-time updates
**Estimated Time**: 2-3 hours

### Implementation Steps
1. Create `internal/tui/pty_events.go`
2. Define tea.Msg types:
   ```go
   type ptyOutputMsg struct {
       sessionID string
       data      []byte
   }
   
   type ptySessionClosedMsg struct {
       sessionID string
       reason    string
   }
   ```

3. Create event listener:
   ```go
   func (m *Model) listenPTYEvents() tea.Cmd {
       return func() tea.Msg {
           event := <-m.ptyEventSub
           return m.convertPTYEvent(event)
       }
   }
   ```

4. Add to Update() method to re-subscribe

### Acceptance Criteria
- [ ] PTY events converted to tea.Msg
- [ ] Output updates trigger re-render
- [ ] Session lifecycle events handled
- [ ] No event loss or duplication

---

## Task 5: Integrate /ai Command
**Priority**: High - Primary user interface
**Estimated Time**: 2 hours

### Implementation Steps
1. Update `handleAICommand()` in model.go:
   ```go
   func (m *Model) handleAICommand(args []string) tea.Cmd {
       if len(args) == 0 {
           return m.showMessage("Usage: /ai <provider> [task]")
       }
       
       provider := args[0]
       task := strings.Join(args[1:], " ")
       
       // Create PTY session
       var session *aicoder.PTYSession
       var err error
       
       if task == "" {
           session, err = m.aiCoderManager.CreateInteractiveCLISession(provider)
       } else {
           session, err = m.aiCoderManager.CreateTaskCLISession(provider, task)
       }
       
       if err != nil {
           return m.showError(err)
       }
       
       // Auto-switch to AI Coder tab
       m.activeView = 8
       m.aiCoderPTYView.AttachToSession(session.ID)
       
       return nil
   }
   ```

2. Add validation for supported providers
3. Show help if provider unknown
4. Add success feedback

### Acceptance Criteria
- [ ] /ai creates PTY sessions
- [ ] Interactive and task modes work
- [ ] Auto-switches to AI Coder tab
- [ ] Helpful error messages

---

## Task 6: Implement Output Streaming
**Priority**: Critical - Core functionality
**Estimated Time**: 3-4 hours

### Implementation Steps
1. Create output subscription in Init():
   ```go
   cmds = append(cmds, m.subscribeToActivePTY())
   ```

2. Implement subscription logic:
   ```go
   func (m *Model) subscribeToActivePTY() tea.Cmd {
       if m.aiCoderPTYView.currentSession == nil {
           return nil
       }
       
       return func() tea.Msg {
           data := <-m.aiCoderPTYView.currentSession.OutputChan
           return PTYOutputMsg{
               SessionID: m.aiCoderPTYView.currentSession.ID,
               Data: data,
           }
       }
   }
   ```

3. Handle in Update() to re-subscribe
4. Add backpressure handling

### Acceptance Criteria
- [ ] Terminal output appears in real-time
- [ ] No UI freezing on heavy output
- [ ] Graceful handling of closed sessions
- [ ] Memory-efficient buffering

---

## Task 7: Add Debug Mode Integration
**Priority**: Medium - Enhanced functionality
**Estimated Time**: 2 hours

### Implementation Steps
1. Add debug mode toggle in PTY view
2. Create automatic event forwarder:
   ```go
   func (m *Model) handleBrummerEvent(event interface{}) {
       if !m.aiCoderPTYView.currentSession.IsDebugModeEnabled() {
           return
       }
       
       switch e := event.(type) {
       case ErrorEvent:
           m.ptyManager.InjectDataToCurrent(aicoder.DataInjectError)
       case TestFailureEvent:
           m.ptyManager.InjectDataToCurrent(aicoder.DataInjectTestFailure)
       }
   }
   ```

3. Add visual indicators for auto-injection
4. Configure injection frequency limits

### Acceptance Criteria
- [ ] Debug mode toggleable per session
- [ ] Automatic error injection works
- [ ] Visual feedback for injections
- [ ] No overwhelming with events

---

## Task 8: Polish and Cleanup
**Priority**: Medium - Code quality
**Estimated Time**: 2-3 hours

### Implementation Steps
1. Remove unused code from old AI coder view
2. Update help text with PTY commands
3. Add session indicators to status bar
4. Improve error messages
5. Add inline documentation

### Code Cleanup
- [ ] Remove `renderAICoderList()`
- [ ] Clean up unused event types
- [ ] Consolidate duplicate logic
- [ ] Update comments

### UI Polish
- [ ] Session count in tab title
- [ ] Activity indicators
- [ ] Improved key binding help
- [ ] Better error formatting

### Acceptance Criteria
- [ ] No dead code remains
- [ ] Help text accurate
- [ ] Consistent UI styling
- [ ] Clear error messages

---

## Task 9: Testing and Documentation
**Priority**: High - Quality assurance
**Estimated Time**: 3-4 hours

### Test Implementation
1. Unit tests for data provider
2. Integration tests for PTY/TUI bridge
3. Manual test scenarios
4. Performance benchmarks

### Documentation Updates
1. Update README with PTY features
2. Add examples to CLAUDE.md
3. Create user guide for AI coders
4. Document key bindings

### Acceptance Criteria
- [ ] 80%+ test coverage for new code
- [ ] No race conditions detected
- [ ] Documentation complete
- [ ] Examples working

---

## Implementation Order

### Phase 1: Foundation (Tasks 1-3)
- Create data provider
- Update model structure  
- Basic view integration

### Phase 2: Core Features (Tasks 4-6)
- Event bridge
- /ai command
- Output streaming

### Phase 3: Enhancement (Tasks 7-8)
- Debug mode
- Polish and cleanup

### Phase 4: Quality (Task 9)
- Testing
- Documentation

## Risk Mitigation

### Technical Risks
1. **Thread Safety**: Use channels for communication
2. **Memory Leaks**: Proper cleanup in all paths
3. **Performance**: Buffer management for high output

### User Experience Risks
1. **Confusion**: Clear help and feedback
2. **Data Loss**: Session persistence
3. **Complexity**: Progressive disclosure

## Success Metrics

1. **Functionality**
   - All /ai commands create PTY sessions
   - Terminal interaction smooth
   - Data injection working

2. **Performance**
   - <10ms input latency
   - <50ms output rendering
   - <100MB memory per session

3. **Reliability**
   - No crashes in 24h usage
   - Clean shutdown always
   - No orphaned processes

This task breakdown provides a clear path to integrating the PTY system naturally into Brummer's TUI, creating a seamless tmux-style AI coder experience.