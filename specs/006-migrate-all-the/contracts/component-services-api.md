# Component Services API Contract

**Version**: 1.0.0
**Feature**: 006-migrate-all-the
**Created**: 2025-10-11

## Overview

This API contract defines the interfaces between services and components in the migrated Brummer architecture. The contract ensures loose coupling between service layer (business logic) and component layer (UI implementation) while maintaining type safety and clear communication patterns.

## API Endpoints

### Input Service API

#### Subscribe to Global Key Events
```go
// Subscribe to global keyboard input events
SubscribeToGlobalKeys(ctx context.Context, handler KeyEventHandler) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `handler` (KeyEventHandler): Event handler function

**Response**: Empty on success

**Errors**:
- `ErrServiceNotActive`: Input service not initialized
- `ErrHandlerExists`: Handler already registered

#### Register Key Binding
```go
// Register a new global key binding
RegisterKeyBinding(ctx context.Context, binding KeyBinding) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `binding` (KeyBinding): Key binding configuration

**Response**: Empty on success

**Errors**:
- `ErrInvalidBinding`: Key binding validation failed
- `ErrBindingConflict`: Key already bound

#### Unregister Key Binding
```go
// Unregister an existing key binding
UnregisterKeyBinding(ctx context.Context, key tea.Key) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `key` (tea.Key): Key to unbind

**Response**: Empty on success

**Errors**:
- `ErrKeyNotFound`: Key not found in bindings
- `ErrServiceNotActive`: Input service not initialized

### Navigation Service API

#### Switch to View
```go
// Switch to specified view
SwitchToView(ctx context.Context, viewType ViewType) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `viewType` (ViewType): Target view type

**Response**: NavigationState on success

**Errors**:
- `ErrInvalidView`: View type not recognized
- `ErrTransitionFailed`: View transition failed

#### Get Current View
```go
// Get currently active view
GetCurrentView(ctx context.Context) (ViewType, error)
```

**Parameters**:
- `ctx` (context.Context): Request context

**Response**: Current ViewType on success

**Errors**:
- `ErrNoActiveView`: No view currently active

#### Cycle to Next View
```go
// Cycle to next view in defined order
CycleToNextView(ctx context.Context) (ViewType, error)
```

**Parameters**:
- `ctx` (context.Context): Request context

**Response**: New ViewType on success

**Errors**:
- `ErrCycleFailed`: View cycling failed

#### Get View for Number Key
```go
// Get view type corresponding to number key
GetViewForNumberKey(ctx context.Context, number int) (ViewType, error)
```

**Parameters**:
- `ctx` (context.Context): Request context
- `number` (int): Number key (1-9)

**Response**: ViewType on success

**Errors**:
- `ErrInvalidNumber`: Number key out of range (1-9)

### System Service API

#### Add System Message
```go
// Add new system message to buffer
AddSystemMessage(ctx context.Context, message SystemMessage) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `message` (SystemMessage): Message to add

**Response**: Message ID on success

**Errors**:
- `ErrMessageTooLarge`: Message exceeds size limit
- `ErrBufferFull`: Message buffer at capacity

#### Get Unread Message Count
```go
// Get count of unread system messages
GetUnreadMessageCount(ctx context.Context) (int, error)
```

**Parameters**:
- `ctx` (context.Context): Request context

**Response**: Unread count on success

**Errors**:
- `ErrServiceNotActive`: System service not initialized

#### Mark Messages as Read
```go
// Mark messages up to timestamp as read
MarkMessagesAsRead(ctx context.Context, upTo time.Time) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `upTo` (time.Time): Timestamp cutoff for read messages

**Response**: Empty on success

**Errors**:
- `ErrInvalidTimestamp`: Timestamp validation failed

#### Toggle System Panel
```go
// Toggle expanded state of system panel
ToggleSystemPanel(ctx context.Context) (bool, error)
```

**Parameters**:
- `ctx` (context.Context): Request context

**Response**: New expanded state on success

**Errors**:
- `ErrServiceNotActive`: System service not initialized

### Notifications Service API

#### Show Notification
```go
// Display notification with default duration
ShowNotification(ctx context.Context, notification Notification) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `notification` (Notification): Notification to display

**Response**: Notification ID on success

**Errors**:
- `ErrTooManyNotifications`: Maximum active notifications exceeded
- `ErrInvalidNotification`: Notification validation failed

#### Show Notification with Duration
```go
// Display notification with custom duration
ShowNotificationWithDuration(ctx context.Context, notification Notification, duration time.Duration) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `notification` (Notification): Notification to display
- `duration` (time.Duration): Custom display duration

**Response**: Notification ID on success

**Errors**:
- `ErrTooManyNotifications`: Maximum active notifications exceeded
- `ErrInvalidDuration`: Duration validation failed

#### Clear Notification
```go
// Clear specific notification by ID
ClearNotification(ctx context.Context, notificationID string) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `notificationID` (string): ID of notification to clear

**Response**: Empty on success

**Errors**:
- `ErrNotificationNotFound`: Notification ID not found

#### Clear All Notifications
```go
// Clear all active notifications
ClearAllNotifications(ctx context.Context) error
```

**Parameters**:
- `ctx` (context.Context): Request context

**Response**: Empty on success

**Errors**:
- `ErrServiceNotActive`: Notifications service not initialized

### File Browser Service API

#### Show File Browser
```go
// Display file browser dialog at specified path
ShowFileBrowser(ctx context.Context, initialPath string, callback FileSelectionCallback) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `initialPath` (string): Starting directory path
- `callback` (FileSelectionCallback): Callback for file selection

**Response**: Dialog ID on success

**Errors**:
- `ErrPathNotAccessible`: Initial path cannot be accessed
- `ErrInvalidCallback**: Callback function validation failed

#### Hide File Browser
```go
// Hide file browser dialog
HideFileBrowser(ctx context.Context, dialogID string) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `dialogID` (string): ID of dialog to hide

**Response**: Empty on success

**Errors**:
- `ErrDialogNotFound`: Dialog ID not found

#### Navigate to Directory
```go
// Navigate file browser to specified directory
NavigateToDirectory(ctx context.Context, dialogID string, path string) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `dialogID` (string): ID of file browser dialog
- `path` (string): Target directory path

**Response**: Updated file list on success

**Errors**:
- `ErrPathNotAccessible`: Target path cannot be accessed
- `ErrDialogNotFound`: Dialog ID not found

#### Select File
```go
// Select file by index in file browser
SelectFile(ctx context.Context, dialogID string, index int) error
```

**Parameters**:
- `ctx` (context.Context): Request context
- `dialogID` (string): ID of file browser dialog
- `index` (int): File index to select

**Response**: Selected file info on success

**Errors**:
- `ErrIndexOutOfRange`: File index out of range
- `ErrDialogNotFound`: Dialog ID not found

## Data Types

### ViewType
```go
type ViewType string

const (
    ProcessesView ViewType = "processes"
    LogsView      ViewType = "logs"
    ErrorsView    ViewType = "errors"
    AICodersView  ViewType = "aicoders"
    SettingsView  ViewType = "settings"
)
```

### LogLevel
```go
type LogLevel string

const (
    DebugLevel LogLevel = "debug"
    InfoLevel  LogLevel = "info"
    WarnLevel  LogLevel = "warn"
    ErrorLevel LogLevel = "error"
)
```

### NotificationLevel
```go
type NotificationLevel string

const (
    InfoNotification  NotificationLevel = "info"
    SuccessNotification NotificationLevel = "success"
    WarningNotification NotificationLevel = "warning"
    ErrorNotification   NotificationLevel = "error"
)
```

### KeyBinding
```go
type KeyBinding struct {
    Key       tea.Key           `json:"key"`
    Action    string            `json:"action"`
    Context   string            `json:"context"`
    Priority  int               `json:"priority"`
    HandlerID string            `json:"handler_id"`
}
```

### SystemMessage
```go
type SystemMessage struct {
    ID        string    `json:"id"`
    Level     LogLevel  `json:"level"`
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
    Read      bool      `json:"read"`
    Source    string    `json:"source"`
}
```

### Notification
```go
type Notification struct {
    ID        string           `json:"id"`
    Message   string           `json:"message"`
    Level     NotificationLevel `json:"level"`
    Timestamp time.Time        `json:"timestamp"`
    Duration  time.Duration    `json:"duration"`
}
```

### FileInfo
```go
type FileInfo struct {
    Name        string    `json:"name"`
    Path        string    `json:"path"`
    IsDirectory bool      `json:"is_directory"`
    Size        int64     `json:"size"`
    Modified    time.Time `json:"modified"`
    Permissions string    `json:"permissions"`
}
```

## Error Codes

```go
const (
    // Service errors
    ErrServiceNotActive = "SERVICE_NOT_ACTIVE"
    ErrServiceTimeout   = "SERVICE_TIMEOUT"

    // Input service errors
    ErrHandlerExists    = "HANDLER_EXISTS"
    ErrInvalidBinding   = "INVALID_BINDING"
    ErrBindingConflict  = "BINDING_CONFLICT"
    ErrKeyNotFound      = "KEY_NOT_FOUND"

    // Navigation service errors
    ErrInvalidView      = "INVALID_VIEW"
    ErrTransitionFailed = "TRANSITION_FAILED"
    ErrNoActiveView     = "NO_ACTIVE_VIEW"
    ErrCycleFailed      = "CYCLE_FAILED"
    ErrInvalidNumber    = "INVALID_NUMBER"

    // System service errors
    ErrMessageTooLarge  = "MESSAGE_TOO_LARGE"
    ErrBufferFull       = "BUFFER_FULL"
    ErrInvalidTimestamp = "INVALID_TIMESTAMP"

    // Notifications service errors
    ErrTooManyNotifications = "TOO_MANY_NOTIFICATIONS"
    ErrInvalidNotification   = "INVALID_NOTIFICATION"
    ErrInvalidDuration       = "INVALID_DURATION"
    ErrNotificationNotFound  = "NOTIFICATION_NOT_FOUND"

    // File browser service errors
    ErrPathNotAccessible = "PATH_NOT_ACCESSIBLE"
    ErrInvalidCallback   = "INVALID_CALLBACK"
    ErrDialogNotFound    = "DIALOG_NOT_FOUND"
    ErrIndexOutOfRange   = "INDEX_OUT_OF_RANGE"
)
```

## Event Schema

### Service-to-Component Events

#### Input Events
```json
{
  "type": "input.key_pressed",
  "data": {
    "key": "string",
    "context": "string",
    "handled": "boolean"
  },
  "timestamp": "2025-10-11T12:00:00Z"
}
```

#### Navigation Events
```json
{
  "type": "navigation.view_changed",
  "data": {
    "from_view": "string",
    "to_view": "string",
    "trigger": "string"
  },
  "timestamp": "2025-10-11T12:00:00Z"
}
```

#### System Events
```json
{
  "type": "system.message_added",
  "data": {
    "message_id": "string",
    "level": "string",
    "content": "string",
    "unread_count": "number"
  },
  "timestamp": "2025-10-11T12:00:00Z"
}
```

#### Notification Events
```json
{
  "type": "notification.shown",
  "data": {
    "notification_id": "string",
    "message": "string",
    "level": "string",
    "duration": "number"
  },
  "timestamp": "2025-10-11T12:00:00Z"
}
```

#### File Browser Events
```json
{
  "type": "file_browser.selection_changed",
  "data": {
    "dialog_id": "string",
    "selected_path": "string",
    "file_info": "object"
  },
  "timestamp": "2025-10-11T12:00:00Z"
}
```

## Performance Requirements

### Response Time Targets
- API calls: <5ms p95
- Event processing: <1ms p95
- State synchronization: <2ms p95

### Concurrency Limits
- Maximum 100 concurrent API calls per service
- Maximum 1000 events per second processing rate
- Maximum 10MB event payload per request

### Resource Limits
- Maximum 100MB memory usage per service
- Maximum 1GB total memory for all services
- Maximum 10% CPU usage during normal operation

## Security Considerations

### Input Validation
- All API parameters must be validated
- File paths must be normalized and validated against directory traversal
- Message content must be sanitized for display

### Access Control
- Components can only access their associated service
- Services validate caller identity and permissions
- Event subscriptions are scoped by service type

### Error Handling
- Sensitive information must not be exposed in error messages
- Error logs must not contain user data
- Rate limiting applies to error-prone operations

This API contract ensures consistent, type-safe communication between services and components while maintaining performance and security standards.