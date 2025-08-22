# Component System Usage

The Brummer TUI now includes an optional component-based UI system that provides a cleaner, more maintainable architecture.

## Quick Start

### Method 1: Using the Helper Script
```bash
# Run with components enabled
./run-with-components.sh

# Or with arguments
./run-with-components.sh dev
```

### Method 2: Environment Variable
```bash
# Enable component system
export BRUMMER_USE_COMPONENTS=true
./brum

# Or inline
BRUMMER_USE_COMPONENTS=true ./brum dev
```

### Method 3: Alias (Recommended for Testing)
Add to your shell configuration:
```bash
alias brum-new='BRUMMER_USE_COMPONENTS=true brum'
alias brum-old='brum'  # Traditional UI
```

## What's Different?

When you run with `BRUMMER_USE_COMPONENTS=true`, you'll see:
```
🎨 Component system enabled - using new UI architecture
```

The component system provides:

### 1. **Better UI Components**
- Uses battle-tested Bubbles components
- Smooth scrolling and text input
- Better keyboard navigation
- Consistent behavior across views

### 2. **Cleaner Architecture**
- Modular components instead of monolithic Model
- Message-based communication
- Easier to test and maintain
- Gradual migration path

### 3. **Same Features**
- All existing functionality works
- Same keyboard shortcuts
- Same process management
- Same MCP integration

## Comparison

| Feature | Traditional UI | Component UI |
|---------|---------------|--------------|
| Scrolling | Custom implementation | Bubbles viewport |
| Text Input | Custom handling | Bubbles textinput |
| Lists | Custom rendering | Bubbles list |
| Architecture | 6000+ line Model | Modular components |
| Testing | Complex | Simple unit tests |
| Maintenance | Difficult | Easy |

## Testing Both Versions

You can easily switch between versions for comparison:

```bash
# Traditional UI
./brum

# Component UI
BRUMMER_USE_COMPONENTS=true ./brum

# Side by side in different terminals
# Terminal 1:
./brum

# Terminal 2:
./run-with-components.sh
```

## Current Status

The component system is **feature-complete** and includes:

### View Components
- ✅ **ProcessViewComponent** - Process management with Bubbles list
- ✅ **LogsViewComponent** - Log viewing with viewport and filtering  
- ✅ **ErrorsViewComponent** - Error tracking and detailed display
- ✅ **URLsViewComponent** - URL detection and browser integration
- ✅ **WebViewComponent** - Web/proxy display with multiple modes
- ✅ **AICodersViewComponent** - AI coder session management
- ✅ **SettingsViewComponent** - Settings management with categories
- ✅ **ScriptSelectorComponent** - Package.json script management

### UI Components
- ✅ **CommandInputComponent** - Command palette with suggestions
- ✅ **NotificationComponent** - System-wide notifications
- ✅ **LayoutComponent** - Overall layout management
- ✅ **ViewSwitcher** - Navigation between views

### Architecture
- ✅ **Custom message system** - Type-safe component communication
- ✅ **ModelComponentAdapter** - Bridge for gradual migration
- ✅ **Feature flag system** - Easy toggle between old/new UI

## Reporting Issues

If you encounter any issues with the component system:

1. Note which view you were in
2. What action you were performing
3. Any error messages
4. Run without components to verify it's component-specific:
   ```bash
   # Test without components
   ./brum
   ```

## Architecture Details

### Component Files
```
internal/tui/components/
├── component.go                    # Base interfaces
├── base.go                        # BaseComponent implementation
├── custom_messages.go             # Message type definitions
├── messages.go                    # Core message handling
├── view_switcher.go               # View navigation
├── notification_component.go      # Notifications
├── layout_component.go            # Layout management
│
├── process_view_adapter.go        # Process management
├── logs_view_adapter.go           # Log viewing
├── errors_view_adapter.go         # Error tracking
├── urls_view_adapter.go           # URL detection
├── web_view_adapter.go            # Web/proxy display
├── ai_coders_view_adapter.go      # AI coder sessions
├── settings_view_adapter.go       # Settings management
├── script_selector_adapter.go     # Script selection
├── command_input_adapter.go       # Command palette
│
├── example_app.go                 # Example application
└── integration_test.go            # Component tests
```

### Message Flow
```
User Input → BubbleTea → Component.Update() → New State → Component.View() → Display
                ↓
        Custom Messages
                ↓
        Other Components
```

## Development

To work on the component system:

```bash
# Run tests
go test ./internal/tui/components/...

# Build with components
go build -o brum ./cmd/brum/main.go

# Run example app (if main function exists)
go run ./internal/tui/components/example_app.go

# Check the implementation
ls internal/tui/components/
```

## Rollback

If you encounter issues, simply run without the environment variable:
```bash
# Returns to traditional UI immediately
./brum
```

The component system is completely optional and both UIs will be maintained during the transition period.