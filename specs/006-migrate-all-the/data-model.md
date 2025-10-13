# Data Model: Controller-to-Component Migration

**Feature**: 006-migrate-all-the
**Created**: 2025-10-11
**Status**: Draft

## Entity Overview

This data model defines the entities involved in migrating controller functionality from the legacy controller-based architecture to the modern service+component framework in Brummer.

## Core Entities

### 1. Component
**Description**: A UI component that follows the BubbleTea Model/Update/View pattern

**Fields**:
- `id` (string, required): Unique component identifier
- `type` (enum, required): Component type - [INPUT, NAVIGATION, SYSTEM, NOTIFICATIONS, FILE_BROWSER]
- `state` (ComponentState, required): Current component state
- `services` (ServiceReference[], required): References to associated services
- `eventSubscriptions` (string[], optional): List of EventBus topics subscribed to

**Validation Rules**:
- Component ID must be unique within the application
- Component type must match one of the 5 migrated controller types
- State must only be modified in BubbleTea Update() methods

**State Transitions**:
```
INITIALIZED → ACTIVE → VISIBLE → FOCUSED → VISIBLE → ACTIVE → SHUTDOWN
```

### 2. Service
**Description**: Business logic service that provides data and operations to components

**Fields**:
- `id` (string, required): Unique service identifier
- `type` (enum, required): Service type - [INPUT, NAVIGATION, SYSTEM, NOTIFICATIONS, FILE_BROWSER]
- `state` (ServiceState, required): Current service state
- `eventBus` (EventBus, required): Reference to system event bus
- `config` (ServiceConfig, optional): Service-specific configuration

**Validation Rules**:
- Service ID must be unique within the application
- Service must register with EventBus during initialization
- Service must handle events asynchronously

**State Transitions**:
```
CREATED → INITIALIZING → ACTIVE → STOPPING → TERMINATED
```

### 3. ComponentMessage
**Description**: Message type for service-to-component communication

**Fields**:
- `id` (string, required): Unique message identifier
- `type` (MessageType, required): Message type
- `sourceService` (string, required): ID of originating service
- `targetComponent` (string, required): ID of target component
- `payload` (interface{}, optional): Message payload data
- `timestamp` (datetime, required): Message creation timestamp

**Validation Rules**:
- Message ID must be unique across the system
- Source service must be registered and active
- Target component must exist and be initialized

### 4. InputState
**Description**: State management for keyboard input and global key bindings

**Fields**:
- `activeHandlers` (KeyHandler[], required): List of active key handlers
- `globalKeyBindings` (KeyBinding[], required): Global keyboard shortcuts
- `hierarchicalContext` (HierarchicalContext, required): Current input context
- `lastQuitPress` (timestamp, optional): Last time quit key was pressed

**Validation Rules**:
- Key handlers must be ordered by priority
- Global key bindings must not conflict
- Hierarchical context must properly route input to correct handler

### 5. NavigationState
**Description**: State management for view switching and navigation

**Fields**:
- `currentView` (ViewType, required): Currently active view
- `viewHistory` (ViewType[], optional): Stack of previously visited views
- `viewOrder` (ViewType[], required): Defined order for view cycling
- `transitionCallbacks` (Callback[], optional): Callbacks for view changes

**Validation Rules**:
- Current view must be one of the defined view types
- View history must maintain proper navigation stack
- Transition callbacks must be called during view changes

### 6. SystemMessageState
**Description**: State management for system messages and panel display

**Fields**:
- `messages` (SystemMessage[], required): Buffer of system messages
- `unreadCount` (integer, required): Number of unread messages
- `isExpanded` (boolean, required): Whether system panel is expanded
- `viewport` (Viewport, required): Viewport dimensions and scroll position

**Validation Rules**:
- Messages must be ordered by timestamp
- Unread count must match actual unread messages
- Viewport must handle scrolling correctly

### 7. NotificationState
**Description**: State management for toast notifications

**Fields**:
- `activeNotifications` (Notification[], required): List of active notifications
- `defaultDuration` (duration, required): Default display duration
- `position` (NotificationPosition, required): Screen position for notifications
- `maxVisible` (integer, required): Maximum simultaneous notifications

**Validation Rules**:
- Active notifications must be ordered by priority and timestamp
- Default duration must be positive
- Max visible must be reasonable (≤5)

### 8. FileBrowserState
**Description**: State management for file selection dialogs

**Fields**:
- `isShowing` (boolean, required): Whether file browser is visible
- `currentPath` (string, required): Current directory path
- `fileList` (FileInfo[], required): List of files in current directory
- `selectedIndex` (integer, required): Index of selected file
- `filter` (FileFilter, optional): File filter settings

**Validation Rules**:
- Current path must be accessible directory
- Selected index must be within file list bounds
- File list must be sorted and include metadata

## Relationships

### Component-Service Relationships
- **Component** 1↔1 **Service**: Each component has exactly one associated service
- **Service** 1→* **ComponentMessage**: Services send messages to components
- **Component** *→1 **EventBus**: Components subscribe to events via EventBus

### State Management Relationships
- **InputState** 1→* **KeyHandler**: Input state manages multiple key handlers
- **NavigationState** 1→* **ViewType**: Navigation state manages view transitions
- **SystemMessageState** 1→* **SystemMessage**: System state manages message buffer
- **NotificationState** 1→* **Notification**: Notification state manages active notifications
- **FileBrowserState** 1→* **FileInfo**: File browser state manages file list

## Enums

### ComponentType
```
INPUT_COMPONENT
NAVIGATION_COMPONENT
SYSTEM_COMPONENT
NOTIFICATIONS_COMPONENT
FILE_BROWSER_COMPONENT
```

### MessageType
```
INPUT_KEY_PRESSED
NAVIGATION_VIEW_CHANGED
SYSTEM_MESSAGE_ADDED
NOTIFICATION_SHOWN
FILE_BROWSER_SELECTION_CHANGED
```

### ViewType
```
PROCESSES_VIEW
LOGS_VIEW
ERRORS_VIEW
AI_CODERS_VIEW
SETTINGS_VIEW
```

## Data Structures

### KeyBinding
```go
type KeyBinding struct {
    Key        tea.Key
    Action     string
    Context    string
    Priority   int
    Handler    func(tea.KeyMsg) tea.Cmd
}
```

### SystemMessage
```go
type SystemMessage struct {
    ID        string
    Level     LogLevel
    Content   string
    Timestamp time.Time
    Read      bool
}
```

### Notification
```go
type Notification struct {
    ID        string
    Message   string
    Duration  time.Duration
    Timestamp time.Time
    Level     NotificationLevel
}
```

### FileInfo
```go
type FileInfo struct {
    Name        string
    Path        string
    IsDirectory bool
    Size        int64
    Modified    time.Time
}
```

## Constraints

### Uniqueness Constraints
- Component ID must be unique across the application
- Service ID must be unique across the application
- Message ID must be unique across message history

### Integrity Constraints
- Component must have associated service of same type
- Service must be initialized before component can use it
- State changes must only occur in BubbleTea Update() methods

### Performance Constraints
- Component rendering must complete in <16ms (60fps)
- Message processing must complete in <5ms
- State synchronization must not block UI thread

## Migration Mapping

### Controller → Component Mapping
| Controller | Target Component | Target Service |
|------------|------------------|----------------|
| InputController | InputComponent | InputService |
| NavigationController | NavigationComponent | NavigationService |
| SystemController | SystemComponent | SystemService |
| NotificationsController | NotificationsComponent | NotificationsService |
| FileBrowserController | FileBrowserComponent | FileBrowserService |

### State Migration Mapping
| Controller State | Component State | Migration Strategy |
|------------------|----------------|-------------------|
| InputController.quitTracking | InputState.lastQuitPress | Direct field mapping |
| NavigationController.currentView | NavigationState.currentView | Direct field mapping |
| SystemController.messageBuffer | SystemMessageState.messages | Buffer transfer |
| NotificationsController.active | NotificationState.activeNotifications | List transfer |
| FileBrowserController.selected | FileBrowserState.selectedIndex | Index mapping |

This data model provides the foundation for implementing the service+component architecture that will replace the legacy controller system while preserving all existing functionality.