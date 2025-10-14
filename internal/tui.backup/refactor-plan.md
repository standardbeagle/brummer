# Model.go Refactoring Plan

## Current Structure Analysis
The model.go file contains ~6000 lines with 138 methods. Here's the logical grouping:

## Challenges Discovered
1. **Method Interdependencies**: Many methods call each other across logical boundaries
2. **Shared State**: The Model struct is accessed by all methods
3. **Compilation Conflicts**: Can't have duplicate method declarations during migration
4. **Import Cycles**: Need to be careful about creating circular dependencies

## Revised Approach: Safe, Incremental Refactoring

### Step 1: Create a Branch
```bash
git checkout -b refactor/split-model-file
```

### Step 2: Extract Types and Constants First
Move these to `types.go` (no compilation conflicts):
- View type and constants
- ViewConfig struct and viewConfigs map
- SystemMessage, UnreadIndicator types
- configAdapter, eventBusWrapper types
- All other non-Model types

### Step 3: Create Helper Files Without Moving Methods
Instead of moving methods immediately, create new files that will eventually hold them:
1. Create empty files with just package declaration and imports
2. Add comments showing which methods will go there
3. This helps visualize the structure without breaking compilation

### Step 4: Use Build Tags for Transition
```go
// +build !refactored
```
This allows us to have two versions during migration.

### Step 5: Move Methods in Dependency Order
1. Start with leaf methods (those that don't call other Model methods)
2. Move related methods together
3. Test compilation after each move

## Alternative Approach: Embedded Structs

Instead of one giant Model struct, we could use composition:

```go
type Model struct {
    // Core fields
    processMgr  *process.Manager
    logStore    *logs.Store
    
    // Embedded view handlers
    URLViews
    ProcessViews
    LogViews
    // etc...
}

type URLViews struct {
    urlsViewport viewport.Model
    // URL-specific fields
}

func (u *URLViews) renderURLsView(m *Model) string {
    // Can access Model fields through m parameter
}
```

## Recommended Immediate Action

Given that you highlighted `renderURLsView`, let's start small:

1. **Create `view_renderer.go`** with an interface:
```go
type ViewRenderer interface {
    RenderURLsView() string
    RenderProcessesView() string
    // etc...
}
```

2. **Create `url_view_renderer.go`** that implements URL rendering:
```go
type URLViewRenderer struct {
    model *Model
}

func (r *URLViewRenderer) RenderURLsView() string {
    // Move implementation here
}
```

3. **Update model.go** to delegate:
```go
func (m *Model) renderURLsView() string {
    renderer := &URLViewRenderer{model: m}
    return renderer.RenderURLsView()
}
```

This approach:
- Doesn't break existing code
- Allows gradual migration
- Makes testing easier
- Reduces model.go size immediately

## Proposed File Structure (Final Goal)

### 1. **model.go** (Core Model struct and initialization)
- Model struct definition
- NewModel functions
- Core initialization

### 2. **view_renders.go** (All view rendering methods)
- All render* methods
- View-specific helpers

### 3. **handlers.go** (Event and input handlers)
- All handle* methods
- Key processing
- Mouse handling

### 4. **updates.go** (State update methods)
- All update* methods
- State management

### 5. **lifecycle.go** (Bubble Tea lifecycle)
- Init()
- Update() 
- View()

### 6. **helpers.go** (Utility methods)
- Format methods
- Validation methods
- Small utilities

## Benefits of This Structure
1. **Logical Separation**: Each file has a clear purpose
2. **Easier Navigation**: 800-1000 lines per file instead of 6000
3. **Better Testability**: Can mock/test rendering separately from logic
4. **Parallel Development**: Multiple developers can work on different aspects
5. **Clear Dependencies**: Makes it obvious what depends on what

## Next Steps
1. Create the refactoring branch
2. Start with types.go extraction
3. Create view_renderer.go with interface
4. Gradually move URL view methods as a pilot
5. If successful, continue with other views