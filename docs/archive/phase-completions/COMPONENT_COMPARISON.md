# Component Comparison: ProgressiveModel vs Real Model

## Components in ProgressiveModel (Step 8) - ALL WORKING ✅

1. **ProcessManager, LogStore, EventBus** - Core dependencies
2. **SimpleViewManager** - View switching
3. **UpdateChan, MasterData, StateManager** - State management
4. **MessageRouter** - Message routing
5. **KeyHandlerRegistry** - Keyboard handling with all handlers registered
6. **ScriptSelector** - Script selection component
7. **All View Components**:
   - ProcessesViewComponent
   - LogsViewComponent
   - ErrorsViewComponent
   - URLsViewComponentNew
   - WebViewComponentNew
   - CommandWindowViewComponentNew
8. **InputComponent** - With InputService
9. **NavigationComponent** - With NavigationService
10. **Init() returns**: `textinput.Blink` + `tickCmd()`

## Components in Real Model but NOT in ProgressiveModel

### Additional Components:
1. **AICoderService** (line 727) - Creates its own EventBus
2. **aicodersView** (line 804) - AICodersViewComponent
3. **mcpView** (line 709) - MCPViewComponent
4. **mcpDebugView** (line 710) - MCPDebugViewComponentNew
5. **tabsComponent** (line 822) - TabsComponent
6. **contentLayout** (line 827) - ContentLayout
7. **settingsView** (line 626) - SettingsViewComponentNew
8. **fileBrowserComponent** (line 605) - FileBrowserComponent (nil in real Model)
9. **systemComponent** (line 606) - SystemComponent (nil in real Model)
10. **notificationsComponent** (line 607) - NotificationsComponent (nil in real Model)

### Additional Event Subscriptions (lines 737-799):
The real Model subscribes to AI Coder events and pushes to `updateChan`:
- `aicoder.session.created`
- `aicoder.session.updated`
- `aicoder.session.ended`
- `aicoder.pty.output`
- `aicoder.error`

### Additional Keyboard Handlers:
1. **AICoderKeyHandler** (line 849) - Only if aicodersView exists
2. **SlashCommandHandler** (line 853) - Context-aware slash command

### Additional Method Calls:
1. **updateProcessList()** (line 632) - Updates process list
2. **setupEventSubscriptions()** (line 887) - Sets up event subscriptions (currently empty)

## Next Testing Steps

Test these components one by one in ProgressiveModel:

**Step 9**: Add tabsComponent + contentLayout
**Step 10**: Add settingsView + mcpView + mcpDebugView
**Step 11**: Add AICoderService + aicodersView + event subscriptions
**Step 12**: Add SlashCommandHandler + AICoderKeyHandler
**Step 13**: Add updateProcessList() call

One of these will break keyboard input!
