# Centralized Layout System

## Overview

The centralized layout system ensures all views calculate their dimensions consistently without duplicating layout logic. This eliminates bugs where different views calculate space differently and ensures responsive behavior works uniformly across all tabs.

## Architecture

```
tea.WindowSizeMsg
  ↓
ViewMessageHandler
  ↓
Model stores termWidth, termHeight
  ↓
LayoutRegions calculated ONCE
  ↓
All views receive LayoutRegions
  ↓
Views render using provided dimensions
```

## Key Files

1. **`layout_regions.go`** - NEW centralized layout calculator (single source of truth)
2. **`layout_manager.go`** - EXISTING pure calculation utilities (can be deprecated)
3. **View components** - CONSUME layout regions without calculating

---

## Usage Pattern for All Views

### Step 1: Store LayoutRegions in Model

```go
// internal/tui/model.go
type Model struct {
    // ... existing fields ...

    // Layout (calculated once per resize)
    layoutRegions components.LayoutRegions

    // Legacy fields (can be deprecated)
    width  int
    height int
    headerHeight int
    footerHeight int
}
```

### Step 2: Calculate Layout on WindowSizeMsg

```go
// internal/tui/view_message_handler.go
case tea.WindowSizeMsg:
    // Update model dimensions (legacy compatibility)
    model.width = m.Width
    model.height = m.Height

    // ===== NEW: Calculate layout regions ONCE =====
    opts := components.DefaultLayoutOptions(m.Width, m.Height)
    opts.SystemPanelOpen = model.systemPanelOpen  // Check if system panel visible

    model.layoutRegions = components.CalculateLayoutRegions(opts)

    // ===== Pass regions to ALL views =====
    if model.processesView != nil {
        model.processesView.SetLayoutRegions(model.layoutRegions)
    }
    if model.logsView != nil {
        model.logsView.SetLayoutRegions(model.layoutRegions)
    }
    if model.errorsView != nil {
        model.errorsView.SetLayoutRegions(model.layoutRegions)
    }
    if model.webView != nil {
        model.webView.SetLayoutRegions(model.layoutRegions)
    }
    if model.urlsView != nil {
        model.urlsView.SetLayoutRegions(model.layoutRegions)
    }
    if model.aiCodersView != nil {
        model.aiCodersView.SetLayoutRegions(model.layoutRegions)
    }
    if model.mcpView != nil {
        model.mcpView.SetLayoutRegions(model.layoutRegions)
    }
    if model.settingsView != nil {
        model.settingsView.SetLayoutRegions(model.layoutRegions)
    }

    // Update header/footer (they also use regions)
    if model.headerComponent != nil {
        model.headerComponent.SetLayoutRegions(model.layoutRegions)
    }
    if model.footerComponent != nil {
        model.footerComponent.SetLayoutRegions(model.layoutRegions)
    }
```

### Step 3: Update View Components

#### Example: AI Coders View

```go
// internal/tui/components/aicoders_view.go
type AICodersViewComponent struct {
    BaseComponent

    // === LAYOUT (provided by system) ===
    layoutRegions LayoutRegions  // NEW: Single source of truth

    // === REMOVE THESE (calculated from layoutRegions) ===
    // width  int  // DEPRECATED: use layoutRegions.ContentWidth
    // height int  // DEPRECATED: use layoutRegions.ContentHeight

    // Viewport for scrollable content
    viewport viewport.Model

    // Styles
    headerStyle  lipgloss.Style
    contentStyle lipgloss.Style
    footerStyle  lipgloss.Style
}

// SetLayoutRegions updates the view's layout from centralized calculation
func (c *AICodersViewComponent) SetLayoutRegions(regions LayoutRegions) {
    c.layoutRegions = regions

    // Update viewport with calculated dimensions
    viewportWidth, viewportHeight := regions.GetViewportArea()

    c.viewport.Width = viewportWidth
    c.viewport.Height = viewportHeight

    // Update styles with region dimensions
    c.contentStyle = c.contentStyle.
        Width(regions.ContentWidth).
        Height(regions.ContentHeight)
}

// ViewWithState renders using layout regions
func (c *AICodersViewComponent) ViewWithState(masterData *state.MasterData) string {
    if !c.IsVisible() || masterData == nil {
        return ""
    }

    // === USE LAYOUT REGIONS (not manual calculations) ===
    contentWidth := c.layoutRegions.ContentWidth
    contentHeight := c.layoutRegions.ContentHeight

    // Render header (fixed size, derived from regions)
    header := c.renderHeader(contentWidth)

    // Render content (uses viewport which was sized by SetLayoutRegions)
    content := c.viewport.View()

    // Render footer (fixed size)
    footer := c.renderFooter(contentWidth)

    return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

// renderHeader uses region width (no manual calculation)
func (c *AICodersViewComponent) renderHeader(width int) string {
    sessionInfo := "Session: AI Coder"
    separator := strings.Repeat("─", width)

    return lipgloss.JoinVertical(lipgloss.Left,
        c.headerStyle.Width(width).Render(sessionInfo),
        separator,
    )
}

// renderFooter uses region width (no manual calculation)
func (c *AICodersViewComponent) renderFooter(width int) string {
    helpText := "↑↓ scroll • Ctrl+T focus • ESC exit"
    return c.footerStyle.Width(width).Render(helpText)
}
```

#### Example: Logs View

```go
// internal/tui/components/logs_view.go
type LogsViewComponent struct {
    BaseComponent

    // === LAYOUT (provided by system) ===
    layoutRegions LayoutRegions

    // Viewport for scrollable logs
    viewport viewport.Model

    // Styles
    headerStyle lipgloss.Style
    // ...
}

func (c *LogsViewComponent) SetLayoutRegions(regions LayoutRegions) {
    c.layoutRegions = regions

    // Configure viewport from regions
    viewportWidth, viewportHeight := regions.GetViewportArea()
    c.viewport.Width = viewportWidth
    c.viewport.Height = viewportHeight - 3  // Minus header lines
}

func (c *LogsViewComponent) ViewWithState(masterData *state.MasterData, logStore *logs.Store) string {
    if !c.IsVisible() {
        return ""
    }

    // Use layout regions (no calculation needed)
    width := c.layoutRegions.ContentWidth

    // Build view
    header := c.renderHeader(width)
    content := c.viewport.View()
    footer := c.renderFooter(width)

    return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}
```

#### Example: Processes View

```go
// internal/tui/components/processes_view.go
type ProcessesViewComponent struct {
    BaseComponent

    // === LAYOUT (provided by system) ===
    layoutRegions LayoutRegions

    // List component
    list list.Model

    // Styles
    // ...
}

func (c *ProcessesViewComponent) SetLayoutRegions(regions LayoutRegions) {
    c.layoutRegions = regions

    // Configure list component from regions
    contentWidth, contentHeight := regions.ContentDimensions()
    c.list.SetWidth(contentWidth)
    c.list.SetHeight(contentHeight - 2)  // Minus header
}

func (c *ProcessesViewComponent) ViewWithState(masterData *state.MasterData) string {
    // Use regions directly
    width := c.layoutRegions.ContentWidth

    return c.list.View()
}
```

---

## Migration Strategy

### Phase 1: Add LayoutRegions Support (Non-Breaking)

```go
// Each view component adds:
type ViewComponent struct {
    // ... existing fields ...

    // NEW: Layout regions (optional during migration)
    layoutRegions *LayoutRegions  // Pointer to allow nil check

    // OLD: Legacy sizing (keep for backward compatibility)
    width  int
    height int
}

// NEW method (add this)
func (c *ViewComponent) SetLayoutRegions(regions LayoutRegions) {
    c.layoutRegions = &regions

    // Also update legacy fields for backward compatibility
    c.width = regions.ContentWidth
    c.height = regions.ContentHeight
}

// OLD method (keep for now)
func (c *ViewComponent) SetSize(width, height int) {
    c.width = width
    c.height = height

    // If layoutRegions not set, create basic one
    if c.layoutRegions == nil {
        opts := DefaultLayoutOptions(width, height)
        regions := CalculateLayoutRegions(opts)
        c.layoutRegions = &regions
    }
}

// View method (support both old and new)
func (c *ViewComponent) ViewWithState(masterData *state.MasterData) string {
    var width, height int

    if c.layoutRegions != nil {
        // NEW: Use layout regions
        width = c.layoutRegions.ContentWidth
        height = c.layoutRegions.ContentHeight
    } else {
        // OLD: Fallback to legacy fields
        width = c.width
        height = c.height
    }

    // ... render using width/height ...
}
```

### Phase 2: Update ViewMessageHandler

```go
// internal/tui/view_message_handler.go
case tea.WindowSizeMsg:
    // Update model dimensions
    model.width = m.Width
    model.height = m.Height

    // NEW: Calculate layout regions
    opts := components.DefaultLayoutOptions(m.Width, m.Height)
    opts.SystemPanelOpen = model.systemPanelOpen
    model.layoutRegions = components.CalculateLayoutRegions(opts)

    // NEW: Pass to views that support it
    views := []interface{
        SetLayoutRegions(components.LayoutRegions)
    }{
        model.processesView,
        model.logsView,
        model.errorsView,
        model.webView,
        model.urlsView,
        model.aiCodersView,
        model.mcpView,
        model.settingsView,
    }

    for _, view := range views {
        if view != nil {
            view.SetLayoutRegions(model.layoutRegions)
        }
    }

    // OLD: Keep legacy SetSize calls for views not migrated yet
    // (Remove these one by one as views are migrated)
    // if model.someOldView != nil {
    //     model.someOldView.SetSize(m.Width, m.Height)
    // }
```

### Phase 3: Remove Legacy Fields (After All Views Migrated)

Once all views use `SetLayoutRegions`:

1. Remove `width`, `height` fields from components
2. Remove `SetSize()` methods
3. Remove legacy calculation logic
4. Update tests to use `LayoutRegions`

---

## Benefits

### ✅ Consistency
All views use the same layout calculations - no divergence or bugs from different math

### ✅ Maintainability
Layout logic in ONE place - change once, applies everywhere

### ✅ Performance
Calculate layout ONCE per resize, not per view

### ✅ Flexibility
Easy to add responsive features (breakpoints, adaptive padding) uniformly

### ✅ Testing
Test layout logic independently without view coupling

### ✅ Debugging
When layout is wrong, there's only ONE place to look

---

## Advanced Features

### Responsive Padding

```go
// In ViewMessageHandler
opts := components.DefaultLayoutOptions(m.Width, m.Height)

// Adjust padding based on terminal size
if m.Width < 80 {
    opts.MinHorizontalPadding = 0  // No padding on small screens
} else if m.Width < 120 {
    opts.MinHorizontalPadding = 2  // Standard padding
} else {
    opts.MinHorizontalPadding = 4  // Extra padding on large screens
}

model.layoutRegions = components.CalculateLayoutRegions(opts)
```

### Per-View Adjustments

```go
// If a view needs custom padding
func (c *AICodersViewComponent) SetLayoutRegions(regions LayoutRegions) {
    // Add extra horizontal padding for this view
    c.layoutRegions = regions.WithHorizontalPadding(4)

    // Update viewport with adjusted dimensions
    c.viewport.Width = c.layoutRegions.ViewportWidth
    c.viewport.Height = c.layoutRegions.ViewportHeight
}
```

### Breakpoint-Aware Rendering

```go
func (c *ViewComponent) ViewWithState(masterData *state.MasterData) string {
    // Get breakpoint from regions
    breakpoint := c.layoutRegions.GetBreakpoint()

    switch breakpoint {
    case BreakpointCompact:
        return c.renderCompactView()
    case BreakpointExpanded:
        return c.renderExpandedView()
    default:
        return c.renderStandardView()
    }
}
```

---

## Testing

### Unit Test Example

```go
func TestLayoutRegions(t *testing.T) {
    // Test standard terminal size
    opts := DefaultLayoutOptions(80, 24)
    regions := CalculateLayoutRegions(opts)

    assert.Equal(t, 80, regions.TerminalWidth)
    assert.Equal(t, 24, regions.TerminalHeight)
    assert.Equal(t, 3, regions.HeaderHeight)
    assert.Equal(t, 1, regions.FooterHeight)
    assert.Equal(t, 20, regions.ContentHeight)  // 24 - 3 - 1

    // Test with system panel
    opts.SystemPanelOpen = true
    regions = CalculateLayoutRegions(opts)
    assert.Equal(t, 10, regions.SystemPanelHeight)
    assert.Equal(t, 10, regions.ContentHeight)  // 24 - 3 - 1 - 10
}
```

---

## Migration Checklist

- [ ] Create `layout_regions.go` with centralized calculations
- [ ] Add `layoutRegions` field to Model
- [ ] Update `ViewMessageHandler` to calculate regions on WindowSizeMsg
- [ ] Add `SetLayoutRegions()` to each view component:
  - [ ] ProcessesViewComponent
  - [ ] LogsViewComponent
  - [ ] ErrorsViewComponent
  - [ ] WebViewComponent
  - [ ] URLsViewComponent
  - [ ] AICodersViewComponent
  - [ ] MCPViewComponent
  - [ ] SettingsViewComponent
- [ ] Update each view's render logic to use `layoutRegions`
- [ ] Add tests for `LayoutRegions`
- [ ] Remove legacy `SetSize()` methods
- [ ] Remove legacy `width`, `height` fields
- [ ] Update documentation

---

## Example: Complete AI Coders View Migration

See the AI Coders View section above for a complete before/after example showing:
1. How to add `SetLayoutRegions()` method
2. How to update `ViewWithState()` to use regions
3. How to configure viewport from regions
4. How to remove manual width/height calculations

This pattern should be applied to ALL view components for consistency.
