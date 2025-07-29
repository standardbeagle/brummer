# Task: TUI Integration for AI Coders
**Generated from Master Planning**: 2025-01-28
**Context Package**: `/requests/agentic-ai-coders/context/`
**Next Phase**: [subtasks-execute.md](../subtasks-execute.md)

## Task Sizing Assessment
**File Count**: 6 files - Within target range (3-7 files)
**Estimated Time**: 30 minutes - At target limit (15-30min)
**Token Estimate**: 140k tokens - Within target (<150k)
**Complexity Level**: 3 (Complex) - TUI integration with multiple UI components
**Parallelization Benefit**: MEDIUM - Requires core service completion for testing
**Atomicity Assessment**: ✅ ATOMIC - Complete TUI view implementation for AI coders
**Boundary Analysis**: ✅ CLEAR - Extends TUI system with new view and components

## Persona Assignment
**Persona**: Frontend Engineer (TUI Specialist)
**Expertise Required**: BubbleTea framework, terminal UI design, Go channels, async patterns
**Worktree**: `~/work/worktrees/agentic-ai-coders/04-tui-integration/`

## Context Summary
**Risk Level**: HIGH (TUI complexity, async integration, user experience)
**Integration Points**: Core AI coder service, TUI model, event system
**Architecture Pattern**: TUI View Pattern (from existing TUI views)
**Similar Reference**: `internal/tui/model.go` - View management and component composition

### Codebase Context (from master analysis)
**Files in Scope**:
```yaml
read_files:   [internal/tui/model.go, internal/tui/script_selector.go, pkg/events/events.go]
modify_files: [internal/tui/model.go]
create_files: [
  /internal/tui/ai_coder_view.go,
  /internal/tui/ai_coder_components.go,
  /internal/tui/ai_coder_keys.go,
  /internal/tui/ai_coder_styles.go,
  /internal/tui/ai_coder_messages.go
]
# Total: 6 files (1 modify, 5 create) - comprehensive TUI integration
```

**Existing Patterns to Follow**:
- `internal/tui/model.go` - View constants, model structure, update patterns
- `internal/tui/script_selector.go` - List component usage, keyboard navigation  
- BubbleTea Model-View-Update pattern with message passing

**Dependencies Context**:
- `github.com/charmbracelet/bubbletea v0.25.0` - Core TUI framework
- `github.com/charmbracelet/bubbles v0.18.0` - Pre-built UI components
- `github.com/charmbracelet/lipgloss v0.10.0` - Styling and layout
- Core AI coder service integration (Task 01 dependency)

### Task Scope Boundaries
**MODIFY Zone** (Direct Changes):
```yaml
primary_files:
  - /internal/tui/model.go                  # Add ViewAICoders constant and integration
  - /internal/tui/ai_coder_view.go          # Main AI coder view implementation
  - /internal/tui/ai_coder_components.go    # UI components (list, detail, command)
  - /internal/tui/ai_coder_keys.go          # Keyboard shortcuts and navigation
  - /internal/tui/ai_coder_styles.go        # Visual styling and themes
  - /internal/tui/ai_coder_messages.go      # TUI messages and event handling

direct_dependencies:
  - /internal/aicoder/manager.go            # Interface with AI coder service
```

**REVIEW Zone** (Check for Impact):
```yaml
check_integration:
  - /internal/tui/script_selector.go        # Review for component pattern consistency
  - /internal/tui/command_autocomplete.go   # Review for keyboard handling patterns
  - /cmd/main.go                           # Review for TUI initialization

check_documentation:
  - /docs/tui-usage.md                     # TUI user guide updates needed
```

**IGNORE Zone** (Do Not Touch):
```yaml
ignore_completely:
  - /internal/mcp/                         # MCP system separate from TUI
  - /internal/process/                     # Process manager separate integration
  - /internal/proxy/                       # Proxy system unrelated
  - /internal/discovery/                   # Discovery system unrelated
  - /internal/logs/                        # Log system separate integration

ignore_search_patterns:
  - "**/testdata/**"                       # Test data files
  - "**/vendor/**"                         # Third-party dependencies
  - "**/node_modules/**"                   # JavaScript dependencies (docs-site)
```

**Boundary Analysis Results**:
- **Usage Count**: TUI system is self-contained with clear interfaces
- **Scope Assessment**: MODERATE scope - extends established TUI patterns
- **Impact Radius**: 1 core file to modify, 5 new files for complete view

### External Context Sources (from master research)
**Primary Documentation**:
- [BubbleTea Tutorial](https://github.com/charmbracelet/bubbletea/tree/master/tutorials) - TUI architecture patterns
- [Bubbles Components](https://github.com/charmbracelet/bubbles) - UI component usage
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss) - Terminal styling patterns

**Standards Applied**:
- BubbleTea Model-View-Update architecture
- Elm-style message passing for state updates
- Component composition patterns
- Responsive terminal layout design

**Reference Implementation**:
- Existing TUI views for architectural consistency
- List component usage from script selector
- Async operation handling patterns

## Task Requirements
**Objective**: Implement complete TUI view for AI coder management with intuitive user experience

**Success Criteria**:
- [ ] New `ViewAICoders` integrated with existing view system
- [ ] AI coder list component with status, progress, and controls
- [ ] Detail panel showing AI coder workspace and output
- [ ] Command input for AI coder interaction
- [ ] Real-time status updates via event system integration
- [ ] Keyboard navigation and shortcuts consistent with existing TUI
- [ ] Responsive layout adapting to terminal size
- [ ] Error handling and user feedback for all operations

**UI Components to Implement**:
1. **AI Coder List** - Active coders with status indicators
2. **Detail Panel** - Workspace files, output, progress
3. **Command Input** - Send commands to selected AI coder
4. **Status Bar** - Global AI coder statistics and health
5. **Context Menu** - Actions (start, pause, stop, delete)

**Validation Commands**:
```bash
# TUI Integration Verification
grep -q "ViewAICoders" internal/tui/model.go          # View constant added
go build ./internal/tui                               # TUI package compiles
./brum --no-mcp | grep -i "ai.*coder"                # TUI shows AI coder tab
go test ./internal/tui -v                             # TUI tests pass
```

## Implementation Specifications

### View Integration
```go
// Addition to internal/tui/model.go
const (
    // Existing views...
    ViewScripts        View = "scripts"
    ViewProcesses      View = "processes"
    ViewLogs           View = "logs"
    ViewErrors         View = "errors"
    ViewURLs           View = "urls"
    ViewWeb            View = "web"
    ViewSettings       View = "settings"
    ViewMCPConnections View = "mcp-connections"
    ViewSearch         View = "search"
    ViewFilters        View = "filters"
    ViewScriptSelector View = "script-selector"
    
    // Add AI coder view
    ViewAICoders       View = "ai-coders"
)

// Add to viewConfigs map
var viewConfigs = map[View]ViewConfig{
    // Existing view configs...
    
    ViewAICoders: {
        Title:       "AI Coders",
        Description: "Manage and monitor agentic AI coding assistants",
        KeyMap:      aiCoderKeyMap,
    },
}

// Update Model struct to include AI coder view
type Model struct {
    // Existing fields...
    
    // Add AI coder view
    aiCoderView AICoderView
}
```

### AI Coder View Implementation
```go
// internal/tui/ai_coder_view.go
import (
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/standardbeagle/brummer/internal/aicoder"
)

type AICoderView struct {
    // UI Components
    coderList    list.Model
    detailPanel  viewport.Model
    commandInput textinput.Model
    statusBar    string
    
    // State
    selectedCoder *aicoder.AICoderProcess
    coders        []*aicoder.AICoderProcess
    manager       *aicoder.AICoderManager
    
    // Layout
    width         int
    height        int
    listWidth     int
    detailWidth   int
    
    // UI State
    focusMode     FocusMode
    showDetails   bool
    commandMode   bool
}

type FocusMode int

const (
    FocusList FocusMode = iota
    FocusDetail
    FocusCommand
)

func NewAICoderView(manager *aicoder.AICoderManager) AICoderView {
    // Initialize list component
    coderList := list.New([]list.Item{}, NewAICoderDelegate(), 0, 0)
    coderList.Title = "AI Coders"
    coderList.SetShowStatusBar(true)
    coderList.SetFilteringEnabled(true)
    
    // Initialize detail panel
    detailPanel := viewport.New(0, 0)
    detailPanel.Style = detailPanelStyle
    
    // Initialize command input
    commandInput := textinput.New()
    commandInput.Placeholder = "Enter command for AI coder..."
    commandInput.CharLimit = 500
    
    return AICoderView{
        coderList:    coderList,
        detailPanel:  detailPanel,
        commandInput: commandInput,
        manager:      manager,
        focusMode:    FocusList,
        showDetails:  true,
    }
}

func (v AICoderView) Init() tea.Cmd {
    return tea.Batch(
        v.coderList.StartSpinner(),
        v.refreshCoders(),
    )
}

func (v AICoderView) Update(msg tea.Msg) (AICoderView, tea.Cmd) {
    var cmd tea.Cmd
    var cmds []tea.Cmd
    
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        v.width = msg.Width
        v.height = msg.Height
        v.updateLayout()
        
    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            v.focusMode = (v.focusMode + 1) % 3
            return v, nil
            
        case "n", "ctrl+n":
            if v.focusMode == FocusList {
                return v, v.createNewCoder()
            }
            
        case "d", "delete":
            if v.focusMode == FocusList && v.selectedCoder != nil {
                return v, v.deleteCoder(v.selectedCoder.ID)
            }
            
        case "s":
            if v.focusMode == FocusList && v.selectedCoder != nil {
                return v, v.startCoder(v.selectedCoder.ID)
            }
            
        case "p":
            if v.focusMode == FocusList && v.selectedCoder != nil {
                return v, v.pauseCoder(v.selectedCoder.ID)
            }
            
        case "enter":
            if v.focusMode == FocusCommand && v.commandInput.Value() != "" {
                command := v.commandInput.Value()
                v.commandInput.SetValue("")
                if v.selectedCoder != nil {
                    return v, v.sendCommand(v.selectedCoder.ID, command)
                }
            }
            
        case "esc":
            if v.commandMode {
                v.commandMode = false
                v.focusMode = FocusList
            }
        }
        
    case AICoderListUpdatedMsg:
        v.coders = msg.Coders
        v.updateCoderList()
        
    case AICoderStatusUpdatedMsg:
        v.updateCoderStatus(msg.CoderID, msg.Status)
        
    case AICoderSelectedMsg:
        if coder, exists := v.findCoder(msg.CoderID); exists {
            v.selectedCoder = coder
            v.updateDetailPanel()
        }
    }
    
    // Update components based on focus
    switch v.focusMode {
    case FocusList:
        v.coderList, cmd = v.coderList.Update(msg)
        cmds = append(cmds, cmd)
        
    case FocusDetail:
        v.detailPanel, cmd = v.detailPanel.Update(msg)
        cmds = append(cmds, cmd)
        
    case FocusCommand:
        v.commandInput, cmd = v.commandInput.Update(msg)
        cmds = append(cmds, cmd)
    }
    
    return v, tea.Batch(cmds...)
}

func (v AICoderView) View() string {
    if v.width == 0 {
        return "Loading AI Coder view..."
    }
    
    // Build layout
    leftPanel := v.renderLeftPanel()
    rightPanel := v.renderRightPanel()
    
    // Combine panels
    content := lipgloss.JoinHorizontal(
        lipgloss.Top,
        leftPanel,
        rightPanel,
    )
    
    // Add status bar
    statusBar := v.renderStatusBar()
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        content,
        statusBar,
    )
}

func (v *AICoderView) updateLayout() {
    listHeight := v.height - 3 // Reserve space for status bar
    
    if v.showDetails {
        v.listWidth = v.width / 3
        v.detailWidth = v.width - v.listWidth - 1
    } else {
        v.listWidth = v.width
        v.detailWidth = 0
    }
    
    v.coderList.SetSize(v.listWidth, listHeight)
    v.detailPanel.Width = v.detailWidth
    v.detailPanel.Height = listHeight - 4 // Reserve space for command input
}
```

### AI Coder Components
```go
// internal/tui/ai_coder_components.go
import (
    "fmt"
    "strings"
    "time"
    
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/lipgloss"
    "github.com/standardbeagle/brummer/internal/aicoder"
)

// AI Coder List Item
type AICoderItem struct {
    coder *aicoder.AICoderProcess
}

func (i AICoderItem) FilterValue() string {
    return i.coder.Name + " " + i.coder.Task
}

func (i AICoderItem) Title() string {
    status := strings.ToUpper(string(i.coder.Status))
    statusColor := getStatusColor(i.coder.Status)
    
    return fmt.Sprintf("%s %s",
        statusColor.Render(status),
        i.coder.Name,
    )
}

func (i AICoderItem) Description() string {
    elapsed := time.Since(i.coder.CreatedAt)
    progress := fmt.Sprintf("%.1f%%", i.coder.Progress*100)
    
    return fmt.Sprintf("%s | %s | %s ago",
        truncateString(i.coder.Task, 40),
        progress,
        formatDuration(elapsed),
    )
}

// AI Coder List Delegate
type AICoderDelegate struct{}

func NewAICoderDelegate() AICoderDelegate {
    return AICoderDelegate{}
}

func (d AICoderDelegate) Height() int                               { return 2 }
func (d AICoderDelegate) Spacing() int                              { return 1 }
func (d AICoderDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d AICoderDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
    i, ok := listItem.(AICoderItem)
    if !ok {
        return
    }
    
    coder := i.coder
    
    // Style based on selection and status
    var style lipgloss.Style
    if index == m.Index() {
        style = selectedItemStyle
    } else {
        style = normalItemStyle
    }
    
    // Status indicator
    statusIcon := getStatusIcon(coder.Status)
    statusColor := getStatusColor(coder.Status)
    
    // Progress bar
    progressBar := renderProgressBar(coder.Progress, 20)
    
    // Format content
    title := fmt.Sprintf("%s %s [%s]",
        statusColor.Render(statusIcon),
        coder.Name,
        coder.Provider,
    )
    
    description := fmt.Sprintf("%s %s",
        truncateString(coder.Task, 50),
        progressBar,
    )
    
    content := fmt.Sprintf("%s\n%s", title, description)
    
    fmt.Fprint(w, style.Render(content))
}

// Detail Panel Rendering
func (v *AICoderView) renderDetailPanel() string {
    if v.selectedCoder == nil {
        return detailPanelStyle.Render("No AI coder selected")
    }
    
    coder := v.selectedCoder
    
    var content strings.Builder
    
    // Header
    content.WriteString(detailHeaderStyle.Render(fmt.Sprintf("AI Coder: %s", coder.Name)))
    content.WriteString("\n\n")
    
    // Status section
    content.WriteString(detailSectionStyle.Render("Status"))
    content.WriteString("\n")
    content.WriteString(fmt.Sprintf("State: %s\n", getStatusColor(coder.Status).Render(string(coder.Status))))
    content.WriteString(fmt.Sprintf("Provider: %s\n", coder.Provider))
    content.WriteString(fmt.Sprintf("Progress: %.1f%%\n", coder.Progress*100))
    content.WriteString(fmt.Sprintf("Created: %s\n", coder.CreatedAt.Format("2006-01-02 15:04:05")))
    content.WriteString("\n")
    
    // Task section
    content.WriteString(detailSectionStyle.Render("Task"))
    content.WriteString("\n")
    content.WriteString(wordWrap(coder.Task, v.detailWidth-4))
    content.WriteString("\n\n")
    
    // Workspace section
    content.WriteString(detailSectionStyle.Render("Workspace"))
    content.WriteString("\n")
    content.WriteString(fmt.Sprintf("Directory: %s\n", coder.WorkspaceDir))
    
    // List workspace files (if available)
    if files, err := coder.ListWorkspaceFiles(); err == nil {
        content.WriteString("Files:\n")
        for _, file := range files[:min(len(files), 10)] { // Show first 10 files
            content.WriteString(fmt.Sprintf("  - %s\n", file))
        }
        if len(files) > 10 {
            content.WriteString(fmt.Sprintf("  ... and %d more files\n", len(files)-10))
        }
    }
    
    return content.String()
}

// Progress Bar Rendering
func renderProgressBar(progress float64, width int) string {
    filled := int(progress * float64(width))
    empty := width - filled
    
    bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
    percentage := fmt.Sprintf("%.1f%%", progress*100)
    
    return fmt.Sprintf("[%s] %s", bar, percentage)
}

// Status Styling
func getStatusColor(status aicoder.AICoderStatus) lipgloss.Style {
    switch status {
    case aicoder.StatusRunning:
        return lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // Green
    case aicoder.StatusCompleted:
        return lipgloss.NewStyle().Foreground(lipgloss.Color("4")) // Blue
    case aicoder.StatusFailed:
        return lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // Red
    case aicoder.StatusPaused:
        return lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // Yellow
    default:
        return lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
    }
}

func getStatusIcon(status aicoder.AICoderStatus) string {
    switch status {
    case aicoder.StatusRunning:
        return "▶"
    case aicoder.StatusCompleted:
        return "✓"
    case aicoder.StatusFailed:
        return "✗"
    case aicoder.StatusPaused:
        return "⏸"
    case aicoder.StatusCreating:
        return "⚙"
    default:
        return "○"
    }
}

// Utility functions
func truncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen-3] + "..."
}

func formatDuration(d time.Duration) string {
    if d < time.Minute {
        return fmt.Sprintf("%ds", int(d.Seconds()))
    } else if d < time.Hour {
        return fmt.Sprintf("%dm", int(d.Minutes()))
    } else {
        return fmt.Sprintf("%dh", int(d.Hours()))
    }
}

func wordWrap(text string, width int) string {
    words := strings.Fields(text)
    if len(words) == 0 {
        return text
    }
    
    var lines []string
    var currentLine string
    
    for _, word := range words {
        if len(currentLine)+len(word)+1 <= width {
            if currentLine == "" {
                currentLine = word
            } else {
                currentLine += " " + word
            }
        } else {
            if currentLine != "" {
                lines = append(lines, currentLine)
            }
            currentLine = word
        }
    }
    
    if currentLine != "" {
        lines = append(lines, currentLine)
    }
    
    return strings.Join(lines, "\n")
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

### Event Integration and Messages
```go
// internal/tui/ai_coder_messages.go
import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/standardbeagle/brummer/internal/aicoder"
)

// TUI Messages for AI Coder events
type AICoderListUpdatedMsg struct {
    Coders []*aicoder.AICoderProcess
}

type AICoderStatusUpdatedMsg struct {
    CoderID string
    Status  aicoder.AICoderStatus
    Message string
}

type AICoderSelectedMsg struct {
    CoderID string
}

type AICoderCreatedMsg struct {
    Coder *aicoder.AICoderProcess
}

type AICoderDeletedMsg struct {
    CoderID string
}

type AICoderCommandSentMsg struct {
    CoderID string
    Command string
    Success bool
    Error   string
}

// Command functions that return tea.Cmd
func (v AICoderView) refreshCoders() tea.Cmd {
    return func() tea.Msg {
        coders := v.manager.ListCoders()
        return AICoderListUpdatedMsg{Coders: coders}
    }
}

func (v AICoderView) createNewCoder() tea.Cmd {
    return func() tea.Msg {
        // This would typically open a form or dialog
        // For now, create with default parameters
        req := aicoder.CreateCoderRequest{
            Task:     "General coding assistance",
            Provider: "claude",
        }
        
        coder, err := v.manager.CreateCoder(context.Background(), req)
        if err != nil {
            return AICoderCommandSentMsg{
                Success: false,
                Error:   fmt.Sprintf("Failed to create AI coder: %v", err),
            }
        }
        
        return AICoderCreatedMsg{Coder: coder}
    }
}

func (v AICoderView) deleteCoder(coderID string) tea.Cmd {
    return func() tea.Msg {
        err := v.manager.DeleteCoder(coderID)
        if err != nil {
            return AICoderCommandSentMsg{
                CoderID: coderID,
                Success: false,
                Error:   fmt.Sprintf("Failed to delete AI coder: %v", err),
            }
        }
        
        return AICoderDeletedMsg{CoderID: coderID}
    }
}

func (v AICoderView) startCoder(coderID string) tea.Cmd {
    return func() tea.Msg {
        err := v.manager.StartCoder(coderID)
        success := err == nil
        errorMsg := ""
        if err != nil {
            errorMsg = err.Error()
        }
        
        return AICoderCommandSentMsg{
            CoderID: coderID,
            Command: "start",
            Success: success,
            Error:   errorMsg,
        }
    }
}

func (v AICoderView) pauseCoder(coderID string) tea.Cmd {
    return func() tea.Msg {
        err := v.manager.PauseCoder(coderID)
        success := err == nil
        errorMsg := ""
        if err != nil {
            errorMsg = err.Error()
        }
        
        return AICoderCommandSentMsg{
            CoderID: coderID,
            Command: "pause",
            Success: success,
            Error:   errorMsg,
        }
    }
}

func (v AICoderView) sendCommand(coderID, command string) tea.Cmd {
    return func() tea.Msg {
        coder, exists := v.manager.GetCoder(coderID)
        if !exists {
            return AICoderCommandSentMsg{
                CoderID: coderID,
                Command: command,
                Success: false,
                Error:   "AI coder not found",
            }
        }
        
        err := coder.SendCommand(command)
        success := err == nil
        errorMsg := ""
        if err != nil {
            errorMsg = err.Error()
        }
        
        return AICoderCommandSentMsg{
            CoderID: coderID,
            Command: command,
            Success: success,
            Error:   errorMsg,
        }
    }
}
```

## Risk Mitigation (from master analysis)
**High-Risk Mitigations**:
- TUI complexity - Follow established BubbleTea patterns from existing views - Testing: Manual TUI testing and component unit tests
- Async integration - Use proper message passing for all AI coder operations - Recovery: Error handling with user feedback
- User experience - Responsive layout and intuitive keyboard navigation - Validation: User testing and keyboard navigation testing

**Context Validation**:
- [ ] BubbleTea patterns from `internal/tui/model.go` successfully applied
- [ ] Component composition from existing TUI components properly implemented
- [ ] Event integration maintains TUI responsiveness

## Integration with Other Tasks
**Dependencies**: Task 01 (Core Service) - Requires AICoderManager interface
**Integration Points**: 
- Task 02 (MCP Tools) integration for external control
- Task 05 (Process Integration) for process status display
- Task 06 (Event System) for real-time updates

**Shared Context**: TUI becomes primary user interface for AI coder management

## Execution Notes
- **Start Pattern**: Use existing TUI view patterns from `internal/tui/model.go` as foundation
- **Key Context**: Focus on responsive async operations and clear user feedback
- **Integration Test**: Verify TUI updates in real-time with AI coder operations
- **Review Focus**: User experience, keyboard navigation, and component composition

This task creates a comprehensive, user-friendly TUI interface that integrates seamlessly with Brummer's existing terminal interface while providing full control over AI coder instances.