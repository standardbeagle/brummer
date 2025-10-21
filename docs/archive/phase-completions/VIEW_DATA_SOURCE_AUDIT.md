# View Data Source Audit

**Date:** 2025-10-20
**Branch:** 010-get-all-the
**Purpose:** Ensure all views use consistent data sources between rendering and badges

## Executive Summary

### Issues Found
1. ✅ **ViewWeb** - FIXED: Badge was using `proxyServer.GetRequests()`, view uses `WebViewState`
2. ⚠️ **ViewProcesses** - Badge uses `processMgr`, view uses both `ProcessesState` and `processMgr`
3. ⚠️ **ViewErrors** - Badge uses `logStore.GetErrors()`, view uses both `ErrorsState` and `logStore`
4. ⚠️ **ViewLogs** - Inconsistent data flow
5. ⚠️ **ViewURLs** - Badge uses `logStore.GetURLs()`, view uses `URLsState`
6. ⚠️ **ViewAICoders** - Badge uses `aiCoderService`, view uses `AICoderState`

### Compliance Status
- **Good:** ViewWeb, ViewMCPStatus, ViewMCPConnections (use masterData consistently)
- **Mixed:** ViewProcesses, ViewErrors, ViewLogs, ViewURLs, ViewAICoders
- **Architecture Principle:** Should use masterData as single source of truth

## Detailed Analysis

### ViewProcesses

**View Rendering** (line 1593-1596):
```go
case ViewProcesses:
    m.processesView.SetSize(m.width, contentHeight)
    return m.processesView.ViewWithState(m.masterData, m.processMgr)
```
- Uses: `masterData.ProcessesState` + `processMgr` (dual source)
- Component: `ProcessesViewComponent`

**Badge/Status** (line 1680-1688):
```go
case ViewProcesses:
    allProcesses := m.processMgr.GetAllProcesses()
    running := 0
    for _, p := range allProcesses {
        if p.GetStatus() == processes.StatusRunning {
            running++
        }
    }
    return fmt.Sprintf("%d processes, %d running", len(allProcesses), running)
```
- Uses: `processMgr.GetAllProcesses()` directly
- **ISSUE:** Bypasses ProcessesState, reads directly from service

**Tick Handler** (line 1125-1126):
```go
case ViewProcesses:
    m.updateProcessList() // Safe: updates list model without blocking
```
- Calls `updateProcessList()` which likely syncs ProcessesState

**Analysis:**
- ProcessesState exists but badge bypasses it
- View rendering passes both masterData AND processMgr (redundant)
- Should use ProcessesState as single source of truth

---

### ViewLogs

**View Rendering** (line 1597-1606):
```go
case ViewLogs:
    if m.masterData != nil && m.masterData.LogsState != nil {
        if m.masterData.LogsState.GetSelectedProcess() != m.selectedProcess {
            m.masterData.LogsState = m.masterData.LogsState.SetSelectedProcess(m.selectedProcess)
        }
    }
    m.logsView.SetSize(m.width, contentHeight)
    return m.logsView.ViewWithState(m.masterData, m.logStore)
```
- Uses: `masterData.LogsState` + `logStore` (dual source)
- Component: `LogsViewComponent`

**Badge/Status** (line 1690-1694):
```go
case ViewLogs:
    if m.selectedProcess != "" {
        return fmt.Sprintf("Process: %s", m.selectedProcess)
    }
    return "All processes"
```
- Uses: `m.selectedProcess` (Model field)
- **ISSUE:** Uses Model field instead of LogsState.GetSelectedProcess()

**Tick Handler** (line 1127-1128):
```go
case ViewLogs:
    m.updateLogsView() // Safe: updates viewport without blocking
```

**Analysis:**
- LogsState exists but view component still needs logStore
- Badge uses Model.selectedProcess instead of LogsState
- Should consolidate to LogsState as single source

---

### ViewErrors

**View Rendering** (line 1607-1630):
```go
case ViewErrors:
    contentHeight := m.height - m.headerHeight - m.footerHeight

    if m.contentLayout != nil && m.width >= 100 {
        // Split view
        leftWidth := int(float64(m.width) * DefaultSplitRatio)
        rightWidth := m.width - leftWidth

        m.errorsView.UpdateSize(leftWidth-4, m.height, m.headerHeight, m.footerHeight, contentHeight-2)
        leftContent := m.errorsView.ViewWithState(m.masterData, m.logStore)

        rightContent := m.renderErrorDetailsView(rightWidth-4, contentHeight-2)

        return m.contentLayout.RenderSplitView(leftContent, rightContent, DefaultSplitRatio)
    }

    m.errorsView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
    return m.errorsView.ViewWithState(m.masterData, m.logStore)
```
- Uses: `masterData.ErrorsState` + `logStore` (dual source)
- Component: `ErrorsViewComponent`

**Badge/Status** (line 1696-1699):
```go
case ViewErrors:
    errors := m.logStore.GetErrors()
    errorCount := len(errors)
    return fmt.Sprintf("%d errors", errorCount)
```
- Uses: `logStore.GetErrors()` directly
- **ISSUE:** Bypasses ErrorsState, reads directly from logStore

**Tick Handler:** Not included (no periodic update)

**Analysis:**
- ErrorsState exists but badge bypasses it
- View component needs both ErrorsState and logStore
- Should sync ErrorsState.GetErrorCount() with logStore in tick handler

---

### ViewURLs

**View Rendering** (line 1631-1634):
```go
case ViewURLs:
    m.urlsView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
    return m.urlsView.ViewWithState(m.masterData)
```
- Uses: `masterData.URLsState` only ✅
- Component: `URLsViewComponentNew`

**Badge/Status** (line 1701-1703):
```go
case ViewURLs:
    urls := m.logStore.GetURLs()
    return fmt.Sprintf("%d URLs detected", len(urls))
```
- Uses: `logStore.GetURLs()` directly
- **ISSUE:** Bypasses URLsState, reads directly from logStore

**Tick Handler** (line 1106-1120):
```go
// Sync URLs from log store to URLsState (non-blocking)
if m.masterData != nil && m.masterData.URLsState != nil {
    urls := m.logStore.GetURLs()
    urlInfos := make([]state.URLInfo, len(urls))
    for i, url := range urls {
        urlInfos[i] = state.URLInfo{
            URL:         url.URL,
            Label:       url.URL,
            ProcessName: url.ProcessID,
            Timestamp:   url.Timestamp,
            VisitCount:  0,
        }
    }
    m.masterData.URLsState = m.masterData.URLsState.UpdateURLs(urlInfos)
}
// ⚠️ ViewURLs gets its updates through URLsState sync above
```
- **GOOD:** Syncs URLsState from logStore in tick handler
- **ISSUE:** Badge still reads from logStore instead of URLsState

**Analysis:**
- Good pattern: URLsState synced in tick handler
- View uses URLsState correctly
- Badge should use `masterData.URLsState.GetURLCount()` instead of logStore

---

### ViewWeb

**View Rendering** (line 1635-1638):
```go
case ViewWeb:
    m.webView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
    return m.webView.ViewWithState(m.masterData)
```
- Uses: `masterData.WebViewState` only ✅
- Component: `WebViewComponentNew`

**Badge/Status** (line 1705-1710):
```go
case ViewWeb:
    // Use WebViewState as single source of truth (matches view rendering)
    if m.masterData != nil && m.masterData.WebViewState != nil {
        return fmt.Sprintf("%d requests", m.masterData.WebViewState.GetTotalRequests())
    }
    return "0 requests"
```
- Uses: `masterData.WebViewState.GetTotalRequests()` ✅
- **FIXED:** Now uses WebViewState consistently

**Tick Handler** (line 1130-1132):
```go
// ⚠️ ViewWeb gets its updates through ProxyRequestMsg handler (event-driven)
//    WebViewState is updated via events, no periodic sync needed
```
- Event-driven updates via ProxyRequestMsg ✅

**Analysis:**
- **EXCELLENT:** Complete consistency - view, badge, and updates all use WebViewState
- Badge was fixed to use WebViewState instead of proxyServer

---

### ViewAICoders

**View Rendering** (line 1665-1669):
```go
case ViewAICoders:
    if m.aicodersView != nil && m.masterData != nil {
        return m.aicodersView.ViewWithState(m.masterData)
    }
    return "AI Coder feature is not initialized"
```
- Uses: `masterData.AICoderState` only ✅
- Component: `AICodersViewComponent`

**Badge/Status** (line 1712-1719):
```go
case ViewAICoders:
    // TODO(Feature 005): Get session count from service
    if m.aiCoderService != nil {
        svc := m.aiCoderService.(*services.AICoderService)
        sessions := svc.ListSessions()
        return fmt.Sprintf("%d AI coder sessions", len(sessions))
    }
    return "0 AI coder sessions"
```
- Uses: `aiCoderService.ListSessions()` directly
- **ISSUE:** Bypasses AICoderState, reads directly from service
- **TODO:** Comment indicates this should be fixed

**Tick Handler:** Not included

**Analysis:**
- AICoderState exists and view uses it correctly
- Badge bypasses AICoderState and reads from service
- Should use `masterData.AICoderState.GetSessionCount()` or similar

---

### ViewMCPStatus

**View Rendering** (line 1650-1654):
```go
case ViewMCPStatus:
    contentHeight := m.height - m.headerHeight - m.footerHeight
    m.mcpView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
    return m.mcpView.ViewWithState(m.masterData)
```
- Uses: `masterData.MCPState` only ✅
- Component: `MCPViewComponentNew`

**Badge/Status** (line 1721-1731):
```go
case ViewMCPStatus:
    if m.masterData != nil && m.masterData.MCPState != nil {
        mcpState := m.masterData.MCPState
        if mcpState.GetActive() {
            return fmt.Sprintf("MCP Active - %d tools, %d requests", mcpState.GetToolsCount(), mcpState.GetRequestCount())
        } else {
            return "MCP Inactive"
        }
    }
    return "MCP Status Unknown"
```
- Uses: `masterData.MCPState` only ✅

**Analysis:**
- **EXCELLENT:** Complete consistency - both view and badge use MCPState

---

### ViewMCPConnections

**View Rendering** (line 1655-1664):
```go
case ViewMCPConnections:
    if m.debugMode {
        contentHeight := m.height - m.headerHeight - m.footerHeight
        m.mcpDebugView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
        return m.mcpDebugView.ViewWithState(m.masterData)
    }
    // Fallback to settings if not in debug mode
    m.settingsView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
    return m.settingsView.ViewWithState(m.masterData)
```
- Uses: `masterData.MCPDebugState` only ✅
- Component: `MCPDebugViewComponentNew`

**Badge/Status** (line 1733-1741):
```go
case ViewMCPConnections:
    if m.masterData != nil && m.masterData.MCPDebugState != nil {
        debugState := m.masterData.MCPDebugState
        totalConns := debugState.GetConnectionsCount()
        activeConns := debugState.GetActiveConnectionsCount()
        return fmt.Sprintf("MCP Debug - %d total, %d active connections", totalConns, activeConns)
    }
    return "MCP Debug Unavailable"
```
- Uses: `masterData.MCPDebugState` only ✅

**Analysis:**
- **EXCELLENT:** Complete consistency - both view and badge use MCPDebugState

---

### ViewSettings

**View Rendering** (line 1639-1649):
```go
case ViewSettings:
    if m.fileBrowserComponent != nil {
        if dialogView := m.fileBrowserComponent.View(); dialogView != "" {
            return dialogView
        }
    }
    m.settingsView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
    return m.settingsView.ViewWithState(m.masterData)
```
- Uses: `masterData.SettingsState` + `fileBrowserComponent` ✅
- Component: `SettingsViewComponentNew`

**Badge/Status:** Not in getViewStatus() - settings has no badge

**Analysis:**
- Settings view uses masterData correctly
- No badge to check for consistency

---

## Redundant Data Storage Analysis

### Model Fields That May Be Redundant

1. **`selectedProcess string`** (line in Model struct)
   - Used by: ViewLogs badge
   - Should be in: `LogsState.SelectedProcess`
   - **REDUNDANT:** State exists in LogsState

2. **View components getting dual sources:**
   - `ProcessesViewComponent` gets both `masterData` AND `processMgr`
   - `LogsViewComponent` gets both `masterData` AND `logStore`
   - `ErrorsViewComponent` gets both `masterData` AND `logStore`

3. **Services accessed directly in badges:**
   - `processMgr.GetAllProcesses()` (should use ProcessesState)
   - `logStore.GetErrors()` (should use ErrorsState)
   - `logStore.GetURLs()` (should use URLsState)
   - `aiCoderService.ListSessions()` (should use AICoderState)

## Unused Model Properties Check

### Legacy/Deprecated Fields
- `stateManager *state.StateManager` - Marked as "to be deprecated"
- Should verify if still used

### Component Redundancy
Components pass multiple data sources when they should only use masterData:
- ProcessesViewComponent(masterData, processMgr) → should be (masterData)
- LogsViewComponent(masterData, logStore) → should be (masterData)
- ErrorsViewComponent(masterData, logStore) → should be (masterData)

## Recommendations

### Priority 1: Fix Data Source Inconsistencies

1. **ViewProcesses Badge**
   ```go
   // Current
   allProcesses := m.processMgr.GetAllProcesses()

   // Should be
   if m.masterData != nil && m.masterData.ProcessesState != nil {
       return m.masterData.ProcessesState.GetStatusSummary()
   }
   ```

2. **ViewErrors Badge**
   ```go
   // Current
   errors := m.logStore.GetErrors()

   // Should be
   if m.masterData != nil && m.masterData.ErrorsState != nil {
       return fmt.Sprintf("%d errors", m.masterData.ErrorsState.GetErrorCount())
   }
   ```

3. **ViewURLs Badge**
   ```go
   // Current
   urls := m.logStore.GetURLs()

   // Should be
   if m.masterData != nil && m.masterData.URLsState != nil {
       return fmt.Sprintf("%d URLs detected", m.masterData.URLsState.GetURLCount())
   }
   ```

4. **ViewLogs Badge**
   ```go
   // Current
   if m.selectedProcess != "" {
       return fmt.Sprintf("Process: %s", m.selectedProcess)
   }

   // Should be
   if m.masterData != nil && m.masterData.LogsState != nil {
       selectedProcess := m.masterData.LogsState.GetSelectedProcess()
       if selectedProcess != "" {
           return fmt.Sprintf("Process: %s", selectedProcess)
       }
   }
   ```

5. **ViewAICoders Badge**
   ```go
   // Current
   svc := m.aiCoderService.(*services.AICoderService)
   sessions := svc.ListSessions()

   // Should be
   if m.masterData != nil && m.masterData.AICoderState != nil {
       return fmt.Sprintf("%d AI coder sessions", m.masterData.AICoderState.GetSessionCount())
   }
   ```

### Priority 2: Add Missing Getters to State Objects

State objects need these getters for badge consistency:
- `ProcessesState.GetStatusSummary()` → "(n) processes, (m) running"
- `ErrorsState.GetErrorCount()` → int
- `URLsState.GetURLCount()` → int
- `AICoderState.GetSessionCount()` → int

### Priority 3: Remove Dual Data Sources from Components

Update component signatures to only accept masterData:
```go
// Current
ViewWithState(masterData *state.MasterData, logStore *logs.Store)

// Should be
ViewWithState(masterData *state.MasterData)
```

Components affected:
- ProcessesViewComponent
- LogsViewComponent
- ErrorsViewComponent

### Priority 4: Add Sync Logic to Tick Handler

For views that need data from services, add sync in tick handler:
```go
case ViewProcesses:
    m.updateProcessList() // Already exists
    // Ensure ProcessesState has current process count for badge

case ViewErrors:
    // Add: Sync error count from logStore to ErrorsState
    if m.masterData != nil && m.masterData.ErrorsState != nil {
        errors := m.logStore.GetErrors()
        m.masterData.ErrorsState = m.masterData.ErrorsState.UpdateErrorCount(len(errors))
    }
```

### Priority 5: Eliminate Redundant Model Fields

- Remove `selectedProcess` from Model struct
- Always use `LogsState.GetSelectedProcess()` instead

## Architecture Compliance

### Redux/Elm Pattern Compliance

✅ **Good Examples:**
- ViewWeb: Pure masterData usage in both view and badge
- ViewMCPStatus: Pure masterData usage
- ViewMCPConnections: Pure masterData usage

❌ **Non-Compliant:**
- ViewProcesses: Badge bypasses state
- ViewErrors: Badge bypasses state
- ViewURLs: Badge bypasses state (despite sync in tick)
- ViewLogs: Badge uses Model field instead of state
- ViewAICoders: Badge bypasses state

### Single Source of Truth Violations

All view badges should read from masterData states, not from:
- ❌ Direct service calls (`processMgr.GetAllProcesses()`)
- ❌ Direct store calls (`logStore.GetErrors()`, `logStore.GetURLs()`)
- ❌ Model fields (`m.selectedProcess`)
- ❌ Service casts (`aiCoderService.ListSessions()`)

✅ Should always use:
- `masterData.[ViewName]State.Get...()`

## Testing Strategy

1. **Unit Tests:** Verify each state object has required getters
2. **Integration Tests:** Verify badge count matches view content
3. **Consistency Tests:** Verify sync logic keeps states updated
4. **Regression Tests:** Ensure fixing one view doesn't break others

## Completion Criteria

- [ ] All badges use masterData states (no direct service/store access)
- [ ] All state objects have required getter methods
- [ ] All view components only receive masterData parameter
- [ ] Tick handler syncs all necessary states from services
- [ ] Remove redundant Model.selectedProcess field
- [ ] Documentation updated to reflect single source of truth pattern
- [ ] Tests verify data consistency between views and badges
