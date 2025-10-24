# Lipgloss Responsive Layout Guide

## Current System Analysis

### Window Size Flow

```
tea.WindowSizeMsg
  ↓
ViewMessageHandler.HandleMessage()
  ↓
model.width = m.Width
model.height = m.Height
  ↓
contentHeight = calculateContentHeight()
  ↓
Components receive: UpdateSize(width, height, headerHeight, footerHeight, contentHeight)
```

### Content Height Calculation

```go
func (m *Model) calculateContentHeight() int {
    contentHeight := m.height - m.headerHeight - m.footerHeight - 1
    if contentHeight < 1 {
        contentHeight = 1
    }
    return contentHeight
}
```

**Current Constants**:
- `headerHeight = 3` (logo/tabs + underline)
- `footerHeight = 1` (status bar)
- Spacing: `-1` (gap between sections)

---

## Option 1: Fixed Padding with Responsive Content (RECOMMENDED)

### Description
Use fixed header/footer heights with responsive content area. Content scales while maintaining consistent UI chrome.

### Implementation Pattern

```go
// Component setup
type AICodersViewComponent struct {
    width         int
    height        int
    contentHeight int  // NEW: Store calculated content height

    // Styles with responsive sizing
    containerStyle lipgloss.Style
    contentStyle   lipgloss.Style
}

// Size update from WindowSizeMsg
func (c *AICodersViewComponent) SetSize(width, height int) {
    c.width = width
    c.height = height
    c.contentHeight = height - 4  // Subtract header/footer/spacing

    // Update responsive styles
    c.containerStyle = lipgloss.NewStyle().
        Width(width).
        Height(height)

    c.contentStyle = lipgloss.NewStyle().
        Width(width - 4).  // Padding on sides
        Height(c.contentHeight)
}

// Render with calculated dimensions
func (c *AICodersViewComponent) ViewWithState(masterData *state.MasterData) string {
    // Header (fixed height)
    header := c.renderHeader()  // 1-2 lines

    // Content (responsive height)
    content := c.contentStyle.Render(c.renderContent())

    // Footer (fixed height)
    footer := c.renderFooter()  // 1 line

    return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}
```

### Pros
- ✅ Consistent UI chrome (header/footer always same size)
- ✅ Maximum content area utilization
- ✅ Simple to implement
- ✅ Works well with existing system

### Cons
- ⚠️ Small terminals may have very small content area
- ⚠️ Need minimum size checks

---

## Option 2: Percentage-Based Layout

### Description
Allocate screen space as percentages. Example: Header 10%, Content 80%, Footer 10%.

### Implementation Pattern

```go
func (c *AICodersViewComponent) SetSize(width, height int) {
    c.width = width
    c.height = height

    // Calculate heights as percentages
    headerHeight := int(float64(height) * 0.10)  // 10%
    footerHeight := int(float64(height) * 0.10)  // 10%
    contentHeight := height - headerHeight - footerHeight

    // Enforce minimums
    if headerHeight < 2 {
        headerHeight = 2
    }
    if footerHeight < 1 {
        footerHeight = 1
    }

    c.headerStyle = lipgloss.NewStyle().Height(headerHeight)
    c.contentStyle = lipgloss.NewStyle().Height(contentHeight)
    c.footerStyle = lipgloss.NewStyle().Height(footerHeight)
}
```

### Pros
- ✅ Scales proportionally with terminal size
- ✅ Predictable layout ratios
- ✅ Good for various screen sizes

### Cons
- ⚠️ Fixed percentages may not work for all content
- ⚠️ Complex to adjust per-view

---

## Option 3: Adaptive Layout with Breakpoints

### Description
Change layout based on terminal size thresholds (like CSS media queries).

### Implementation Pattern

```go
type LayoutMode int

const (
    LayoutCompact  LayoutMode = iota  // < 80 cols or < 24 rows
    LayoutStandard                     // 80-120 cols, 24-40 rows
    LayoutExpanded                     // > 120 cols or > 40 rows
)

func (c *AICodersViewComponent) determineLayoutMode(width, height int) LayoutMode {
    if width < 80 || height < 24 {
        return LayoutCompact
    } else if width > 120 || height > 40 {
        return LayoutExpanded
    }
    return LayoutStandard
}

func (c *AICodersViewComponent) SetSize(width, height int) {
    c.width = width
    c.height = height
    c.layoutMode = c.determineLayoutMode(width, height)

    switch c.layoutMode {
    case LayoutCompact:
        // Minimal UI, more content
        c.headerHeight = 2
        c.showFooter = false
        c.sidePadding = 1

    case LayoutStandard:
        // Standard layout
        c.headerHeight = 3
        c.showFooter = true
        c.sidePadding = 2

    case LayoutExpanded:
        // Extra spacing, larger fonts
        c.headerHeight = 4
        c.showFooter = true
        c.sidePadding = 4
    }

    c.contentHeight = height - c.headerHeight - (if c.showFooter { 1 } else { 0 })
}
```

### Pros
- ✅ Optimized for different screen sizes
- ✅ Better UX on small/large terminals
- ✅ Can hide/show UI elements

### Cons
- ⚠️ More complex logic
- ⚠️ Need to maintain multiple layouts
- ⚠️ Testing across breakpoints

---

## Option 4: Flexible Box Layout (CSS Flexbox-like)

### Description
Use lipgloss JoinVertical/JoinHorizontal with Height() to create flexible layouts.

### Implementation Pattern

```go
func (c *AICodersViewComponent) ViewWithState(masterData *state.MasterData) string {
    // Calculate available space
    totalHeight := c.height

    // Fixed-size header
    header := lipgloss.NewStyle().
        Height(3).
        Width(c.width).
        Render(c.renderHeader())

    // Flexible content (takes remaining space)
    contentHeight := totalHeight - 3 - 1  // minus header and footer
    content := lipgloss.NewStyle().
        Height(contentHeight).
        Width(c.width).
        Render(c.renderContent())

    // Fixed-size footer
    footer := lipgloss.NewStyle().
        Height(1).
        Width(c.width).
        Render(c.renderFooter())

    // Join with automatic spacing
    return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}
```

### Pros
- ✅ Explicit control over each section
- ✅ Easy to reason about
- ✅ Works well with lipgloss primitives

### Cons
- ⚠️ Manual height calculations needed
- ⚠️ Need to handle overflow

---

## Option 5: Viewport-Based Scrollable Layout

### Description
Use Bubble Tea viewports for content that exceeds available space.

### Implementation Pattern

```go
import "github.com/charmbracelet/bubbles/viewport"

type AICodersViewComponent struct {
    viewport viewport.Model  // Scrollable content area
    width    int
    height   int
}

func (c *AICodersViewComponent) SetSize(width, height int) {
    c.width = width
    c.height = height

    // Calculate viewport dimensions
    headerHeight := 3
    footerHeight := 1
    viewportHeight := height - headerHeight - footerHeight

    // Update viewport
    c.viewport.Width = width - 4  // Leave padding
    c.viewport.Height = viewportHeight
}

func (c *AICodersViewComponent) ViewWithState(masterData *state.MasterData) string {
    // Set viewport content
    content := c.renderContent()
    c.viewport.SetContent(content)

    // Render with fixed header/footer
    header := c.renderHeader()
    viewportView := c.viewport.View()
    footer := c.renderFooter()

    return lipgloss.JoinVertical(lipgloss.Left, header, viewportView, footer)
}
```

### Pros
- ✅ Handles overflow gracefully
- ✅ Built-in scrolling support
- ✅ User can navigate large content

### Cons
- ⚠️ Requires viewport state management
- ⚠️ More complex Update() handling
- ⚠️ May need mouse support

---

## Recommended Approach for AI Coder VT10x

### Hybrid: Fixed Header + Viewport Content + Fixed Footer

This combines **Option 1** (fixed padding) with **Option 5** (viewport) for the best of both worlds.

```go
type AICodersViewComponent struct {
    BaseComponent

    // Sizing
    width         int
    height        int
    contentHeight int

    // Viewport for VT10x terminal output
    viewport viewport.Model

    // Styles
    headerStyle  lipgloss.Style
    contentStyle lipgloss.Style
    footerStyle  lipgloss.Style
}

func NewAICodersViewComponent(service interface{}) *AICodersViewComponent {
    vp := viewport.New(80, 20)  // Initial size
    vp.MouseWheelEnabled = true

    return &AICodersViewComponent{
        BaseComponent: NewBaseComponent(),
        viewport:      vp,
        width:         80,
        height:        24,

        headerStyle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
        contentStyle: lipgloss.NewStyle().Padding(0, 1),
        footerStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
    }
}

func (c *AICodersViewComponent) SetSize(width, height int) {
    c.BaseComponent.SetSize(width, height)
    c.width = width
    c.height = height

    // Calculate content area
    headerHeight := 3  // Session name + divider
    footerHeight := 2  // Status + help text
    spacingLines := 1  // Gap between sections

    c.contentHeight = height - headerHeight - footerHeight - spacingLines

    // Ensure minimum size
    if c.contentHeight < 10 {
        c.contentHeight = 10
    }

    // Update viewport dimensions
    // Subtract 2 for padding on sides
    c.viewport.Width = width - 2
    c.viewport.Height = c.contentHeight

    // Update styles with new dimensions
    c.contentStyle = c.contentStyle.
        Width(width).
        Height(c.contentHeight)
}

func (c *AICodersViewComponent) ViewWithState(masterData *state.MasterData) string {
    if !c.IsVisible() || masterData == nil || masterData.AICoderState == nil {
        return ""
    }

    aiState := masterData.AICoderState

    // === HEADER (Fixed) ===
    sessionID := aiState.FocusedSession
    sessionInfo, _ := aiState.GetSessionInfo(sessionID)

    headerLines := []string{
        c.headerStyle.Render(fmt.Sprintf("Session: %s (%s)", sessionInfo.Name, sessionInfo.Status)),
        strings.Repeat("─", c.width),
    }
    header := strings.Join(headerLines, "\n")

    // === CONTENT (Scrollable Viewport) ===
    // Get VT10x screen buffer
    var contentLines []string
    if aiState.ScreenBuffer != nil && len(aiState.ScreenBuffer) > 0 {
        for _, runeSlice := range aiState.ScreenBuffer {
            contentLines = append(contentLines, string(runeSlice))
        }
    } else {
        contentLines = []string{
            "Waiting for terminal output...",
            "",
            "Press Ctrl+T to toggle terminal focus",
            "Press ESC to return to session list",
        }
    }

    // Set viewport content
    c.viewport.SetContent(strings.Join(contentLines, "\n"))

    // Get viewport view (handles scrolling)
    content := c.contentStyle.Render(c.viewport.View())

    // === FOOTER (Fixed) ===
    focusIndicator := ""
    if aiState.TerminalFocused {
        focusIndicator = "🔒 Terminal Focused"
    }

    scrollInfo := fmt.Sprintf("Line %d/%d",
        c.viewport.YOffset+1,
        len(contentLines))

    footerLines := []string{
        lipgloss.JoinHorizontal(lipgloss.Left,
            c.footerStyle.Render(focusIndicator),
            c.footerStyle.Render(strings.Repeat(" ", c.width-len(focusIndicator)-len(scrollInfo))),
            c.footerStyle.Render(scrollInfo),
        ),
        c.footerStyle.Render("↑↓/j/k scroll • Ctrl+T focus • ESC exit"),
    }
    footer := strings.Join(footerLines, "\n")

    // === COMBINE ===
    return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

// Update handles viewport scrolling
func (c *AICodersViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd

    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        c.SetSize(msg.Width, msg.Height)

    case tea.KeyMsg:
        // Let viewport handle scrolling keys
        c.viewport, cmd = c.viewport.Update(msg)
    }

    return c, cmd
}
```

---

## Spacing and Padding Best Practices

### 1. Consistent Spacing Constants

```go
const (
    // Header spacing
    HeaderHeight       = 3
    HeaderPaddingTop   = 1
    HeaderPaddingBottom = 1

    // Content spacing
    ContentPaddingSide = 2
    ContentLineSpacing = 0  // Lines are already separated

    // Footer spacing
    FooterHeight       = 2
    FooterPaddingTop   = 1

    // Section gaps
    SectionGap         = 1
)
```

### 2. Use Lipgloss Padding Helpers

```go
// All sides
style.Padding(1)

// Vertical, Horizontal
style.Padding(1, 2)  // top/bottom=1, left/right=2

// Top, Right, Bottom, Left (like CSS)
style.Padding(1, 2, 1, 2)

// Or explicit
style.PaddingTop(1).PaddingBottom(1).PaddingLeft(2).PaddingRight(2)
```

### 3. Width Calculation Pattern

```go
// Calculate usable width accounting for padding and borders
func calculateContentWidth(terminalWidth int) int {
    borderWidth := 2  // Left + right borders
    paddingWidth := 4  // 2 on each side

    contentWidth := terminalWidth - borderWidth - paddingWidth

    // Ensure minimum
    if contentWidth < 40 {
        contentWidth = 40
    }

    return contentWidth
}
```

### 4. Responsive Padding

```go
func (c *AICodersViewComponent) calculatePadding(width int) (vertical, horizontal int) {
    switch {
    case width < 80:
        return 0, 1  // Minimal padding on small screens
    case width < 120:
        return 1, 2  // Standard padding
    default:
        return 1, 4  // Extra padding on large screens
    }
}
```

---

## Terminal Size Handling

### Minimum Size Checks

```go
const (
    MinTerminalWidth  = 60
    MinTerminalHeight = 20
)

func (c *AICodersViewComponent) SetSize(width, height int) {
    // Enforce minimums
    if width < MinTerminalWidth {
        width = MinTerminalWidth
    }
    if height < MinTerminalHeight {
        height = MinTerminalHeight
    }

    c.width = width
    c.height = height

    // Continue with normal sizing...
}
```

### Size-Dependent Features

```go
func (c *AICodersViewComponent) shouldShowFooter() bool {
    // Hide footer on very small terminals
    return c.height >= 24
}

func (c *AICodersViewComponent) shouldShowSessionList() bool {
    // Show abbreviated list on small terminals
    return c.width >= 80
}

func (c *AICodersViewComponent) getMaxVisibleSessions() int {
    // Adjust list size based on height
    switch {
    case c.height < 30:
        return 5
    case c.height < 50:
        return 10
    default:
        return 20
    }
}
```

---

## Integration with Current System

### 1. Update ViewMessageHandler

The current handler already calls `SetSize`:

```go
case tea.WindowSizeMsg:
    model.width = m.Width
    model.height = m.Height

    contentHeight := model.calculateContentHeight()

    // Add AICodersView size update
    if model.aiCodersView != nil {
        model.aiCodersView.SetSize(m.Width, contentHeight)
    }
```

### 2. Update AICodersView SetSize Implementation

```go
func (c *AICodersViewComponent) SetSize(width, height int) {
    c.BaseComponent.SetSize(width, height)
    c.width = width
    c.height = height

    // Calculate content area (VT10x viewport)
    headerHeight := 3
    footerHeight := 2
    c.contentHeight = height - headerHeight - footerHeight - 1

    // Update viewport if it exists
    if c.viewport != nil {
        c.viewport.Width = width - 4  // Account for padding
        c.viewport.Height = c.contentHeight
    }
}
```

### 3. Update State to Store VT10x Size

```go
// In internal/tui/state/ai_coder_state.go
type AICoderState struct {
    // ... existing fields ...

    // VT10x terminal dimensions
    TerminalWidth  int
    TerminalHeight int
}

// Method to resize VT10x terminal
func (s *AICoderState) ResizeTerminal(width, height int) *AICoderState {
    newState := s.copy()
    newState.TerminalWidth = width
    newState.TerminalHeight = height
    return newState
}
```

---

## Summary Comparison

| Option | Best For | Complexity | Flexibility | Recommended? |
|--------|----------|------------|-------------|--------------|
| **Option 1: Fixed Padding** | Simple views, consistent chrome | Low | Medium | ✅ Yes |
| **Option 2: Percentage** | Proportional layouts | Medium | Medium | ⚠️ Maybe |
| **Option 3: Breakpoints** | Multi-device support | High | High | ⚠️ Advanced |
| **Option 4: Flex Layout** | Explicit control | Medium | High | ✅ Yes |
| **Option 5: Viewport** | Scrollable content | Medium | High | ✅ **BEST** |
| **Hybrid (Recommended)** | AI Coder VT10x | Medium | High | ✅ **YES!** |

---

## Next Steps

1. **Implement Hybrid Approach** for AI Coder VT10x view
2. **Add viewport** to handle terminal output scrolling
3. **Hook WindowSizeMsg** to update viewport dimensions
4. **Test resizing** with various terminal sizes
5. **Add minimum size checks** for graceful degradation
6. **Document** the sizing patterns for other views

---

## Example: Complete Implementation

See `/home/beagle/work/brummer/internal/tui/components/aicoders_view.go` for the full implementation following this guide.
