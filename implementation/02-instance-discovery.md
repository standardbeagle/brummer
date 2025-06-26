# Implementation Step 2: Instance Discovery

## Overview

This step implements the file-based instance discovery system. Instances write JSON files to signal their availability, and the hub watches these files to initiate connections. The files are signals only - the actual connection state determines availability.

## Goals

1. Implement file watcher for instance discovery
2. Ensure instances only register AFTER MCP server is listening
3. Use OS-specific configuration directories
4. Make file operations non-blocking with channels
5. Connect discovery to hub's instances/list tool

## Technical Design

### Discovery Flow

```
Instance Startup:
1. Start MCP server (HTTP)
2. Call net.Listen() 
3. Get actual port from listener
4. Write instance file with port
5. File appears in discovery directory

Hub Discovery:
1. Watch discovery directory
2. Detect new instance file
3. Parse instance metadata
4. Initiate connection (step 3)
5. Add to available instances
```

### File Structure

```json
// ~/.local/share/brummer/instances/{instance-id}.json
{
  "id": "uuid-v4",
  "pid": 12345,
  "path": "/home/user/project",
  "port": 7778,
  "started": "2024-01-01T00:00:00Z",
  "name": "my-project",
  "has_package_json": true,
  "last_seen": "2024-01-01T00:00:00Z"
}
```

### Directory Structure

```
~/.local/share/brummer/
├── instances/
│   ├── {instance-id-1}.json
│   ├── {instance-id-2}.json
│   └── ...
└── temp/
    └── {instance-id}.json.tmp  # For atomic writes
```

## Implementation

### 1. Update Instance Registration

```go
// internal/mcp/server.go
func StartInstanceServer(ctx context.Context, port int, projectPath string) error {
    // Create MCP server first
    server := NewServer(/* ... */)
    
    // CRITICAL: Listen first, register after
    listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return fmt.Errorf("failed to listen: %w", err)
    }
    defer listener.Close()
    
    // Get actual port (might be different if port was 0)
    actualPort := listener.Addr().(*net.TCPAddr).Port
    
    // Now register with discovery
    registry := discovery.NewRegistry()
    projectName := getProjectName(projectPath)
    hasPackageJSON := fileExists(filepath.Join(projectPath, "package.json"))
    
    if err := registry.Register(projectPath, actualPort, projectName, hasPackageJSON); err != nil {
        // Log but don't fail - instance can run without discovery
        log.Printf("Failed to register with discovery: %v", err)
    }
    
    // Ensure cleanup on shutdown
    defer func() {
        if err := registry.Unregister(); err != nil {
            log.Printf("Failed to unregister: %v", err)
        }
    }()
    
    // Start serving
    return server.Serve(listener)
}
```

### 2. Instance Watcher Implementation

```go
// internal/mcp/instance_watcher.go
package mcp

import (
    "context"
    "encoding/json"
    "io/ioutil"
    "path/filepath"
    "time"
    
    "github.com/fsnotify/fsnotify"
    "github.com/standardbeagle/brummer/internal/discovery"
)

type InstanceWatcher struct {
    instancesChan chan<- *discovery.Instance
    errorsChan    chan<- error
    watcher       *fsnotify.Watcher
}

func NewInstanceWatcher(instancesChan chan<- *discovery.Instance, errorsChan chan<- error) (*InstanceWatcher, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }
    
    return &InstanceWatcher{
        instancesChan: instancesChan,
        errorsChan:    errorsChan,
        watcher:       watcher,
    }, nil
}

func (iw *InstanceWatcher) Start(ctx context.Context) error {
    instancesDir, err := discovery.GetInstancesDir()
    if err != nil {
        return err
    }
    
    // Watch the instances directory
    if err := iw.watcher.Add(instancesDir); err != nil {
        return err
    }
    
    // Initial scan of existing files
    if err := iw.scanDirectory(instancesDir); err != nil {
        return err
    }
    
    // Watch for changes
    go iw.watchLoop(ctx)
    
    return nil
}

func (iw *InstanceWatcher) watchLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            iw.watcher.Close()
            return
            
        case event, ok := <-iw.watcher.Events:
            if !ok {
                return
            }
            
            // Only care about JSON files
            if !strings.HasSuffix(event.Name, ".json") {
                continue
            }
            
            switch {
            case event.Op&fsnotify.Create == fsnotify.Create:
                iw.handleNewFile(event.Name)
            case event.Op&fsnotify.Write == fsnotify.Write:
                iw.handleUpdatedFile(event.Name)
            case event.Op&fsnotify.Remove == fsnotify.Remove:
                iw.handleRemovedFile(event.Name)
            }
            
        case err, ok := <-iw.watcher.Errors:
            if !ok {
                return
            }
            select {
            case iw.errorsChan <- err:
            case <-ctx.Done():
                return
            }
        }
    }
}

func (iw *InstanceWatcher) handleNewFile(path string) {
    // Small delay to ensure file is fully written
    time.Sleep(50 * time.Millisecond)
    
    instance, err := iw.readInstanceFile(path)
    if err != nil {
        select {
        case iw.errorsChan <- err:
        default:
        }
        return
    }
    
    // Send to discovery channel
    select {
    case iw.instancesChan <- instance:
    default:
        // Channel full, skip
    }
}

func (iw *InstanceWatcher) readInstanceFile(path string) (*discovery.Instance, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var instance discovery.Instance
    if err := json.Unmarshal(data, &instance); err != nil {
        return nil, err
    }
    
    return &instance, nil
}

func (iw *InstanceWatcher) scanDirectory(dir string) error {
    entries, err := ioutil.ReadDir(dir)
    if err != nil {
        return err
    }
    
    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
            continue
        }
        
        path := filepath.Join(dir, entry.Name())
        instance, err := iw.readInstanceFile(path)
        if err != nil {
            continue // Skip bad files
        }
        
        select {
        case iw.instancesChan <- instance:
        default:
            // Channel full, skip
        }
    }
    
    return nil
}
```

### 3. Update Hub Server

```go
// internal/mcp/hub_server.go
type HubServer struct {
    connMgr       *ConnectionManager
    watcher       *InstanceWatcher
    instancesChan chan *discovery.Instance
    errorsChan    chan error
}

func NewHubServer() (*HubServer, error) {
    instancesChan := make(chan *discovery.Instance, 100)
    errorsChan := make(chan error, 100)
    
    watcher, err := NewInstanceWatcher(instancesChan, errorsChan)
    if err != nil {
        return nil, err
    }
    
    return &HubServer{
        connMgr:       NewConnectionManager(),
        watcher:       watcher,
        instancesChan: instancesChan,
        errorsChan:    errorsChan,
    }, nil
}

func (h *HubServer) Start(ctx context.Context) error {
    // Start instance watcher
    if err := h.watcher.Start(ctx); err != nil {
        return err
    }
    
    // Process discovered instances
    go h.processDiscoveredInstances(ctx)
    
    return nil
}

func (h *HubServer) processDiscoveredInstances(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
            
        case instance := <-h.instancesChan:
            // For now, just track it
            // In step 3, we'll establish connections
            info := &ConnectionInfo{
                InstanceID:     instance.ID,
                Name:           instance.Name,
                Path:           instance.Path,
                Port:           instance.Port,
                PID:            instance.PID,
                HasPackageJSON: instance.HasPackageJSON,
                State:          StateListening,
                ConnectedAt:    time.Now(),
                LastActivity:   time.Now(),
            }
            
            if err := h.connMgr.Register(info); err != nil {
                log.Printf("Failed to register instance %s: %v", instance.ID, err)
            }
            
        case err := <-h.errorsChan:
            log.Printf("Discovery error: %v", err)
        }
    }
}

func (h *HubServer) CallTool(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    switch request.Name {
    case "instances/list":
        // Get instances from connection manager
        instances := h.connMgr.List()
        
        // Convert to JSON
        data, err := json.Marshal(instances)
        if err != nil {
            return nil, err
        }
        
        return &mcp.CallToolResult{
            Content: []mcp.Content{{
                Type: "text",
                Text: string(data),
            }},
        }, nil
        
    // ... other tools ...
    }
}
```

### 4. Atomic File Writing

```go
// internal/discovery/registry.go
func (r *Registry) writeInstance(instance *Instance) error {
    data, err := json.MarshalIndent(instance, "", "  ")
    if err != nil {
        return err
    }
    
    instancesDir, err := GetInstancesDir()
    if err != nil {
        return err
    }
    
    tempDir, err := GetTempDir()
    if err != nil {
        return err
    }
    
    // Write to temp file first
    tempFile := filepath.Join(tempDir, instance.ID+".json.tmp")
    if err := ioutil.WriteFile(tempFile, data, 0644); err != nil {
        return err
    }
    
    // Atomic rename to final location
    finalFile := filepath.Join(instancesDir, instance.ID+".json")
    if err := os.Rename(tempFile, finalFile); err != nil {
        os.Remove(tempFile) // Clean up on failure
        return err
    }
    
    return nil
}
```

## Testing Plan

### 1. Unit Tests

```go
// internal/mcp/instance_watcher_test.go
func TestInstanceWatcher(t *testing.T) {
    // Create temp directory
    tempDir := t.TempDir()
    
    // Create watcher
    instancesChan := make(chan *discovery.Instance, 10)
    errorsChan := make(chan error, 10)
    watcher, err := NewInstanceWatcher(instancesChan, errorsChan)
    require.NoError(t, err)
    
    // Start watching
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    err = watcher.Start(ctx)
    require.NoError(t, err)
    
    // Create instance file
    instance := &discovery.Instance{
        ID:   "test-123",
        Port: 7778,
        Name: "test-project",
    }
    
    data, _ := json.Marshal(instance)
    err = ioutil.WriteFile(filepath.Join(tempDir, "test-123.json"), data, 0644)
    require.NoError(t, err)
    
    // Should receive instance
    select {
    case received := <-instancesChan:
        assert.Equal(t, "test-123", received.ID)
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for instance")
    }
}
```

### 2. Integration Test

```bash
#!/bin/bash
# test_discovery.sh

# Start hub
brum --mcp &
HUB_PID=$!
sleep 0.5

# Start an instance
cd test-project
brum --no-tui &
INSTANCE_PID=$!
sleep 1

# Query hub for instances
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"instances/list"}}' | \
    brum --mcp | grep -q "test-project"

# Cleanup
kill $INSTANCE_PID
kill $HUB_PID
```

### 3. File Watch Test

```go
func TestFileWatchEvents(t *testing.T) {
    // Test create, update, delete events
    // Verify each triggers appropriate action
    // Check atomic writes work correctly
}
```

## Success Criteria

1. ✅ Instance files created AFTER server is listening
2. ✅ Files use OS-specific config directories
3. ✅ Atomic file writes (temp → rename)
4. ✅ File watcher detects new instances < 100ms
5. ✅ instances/list returns discovered instances
6. ✅ No blocking file operations
7. ✅ Graceful handling of invalid files
8. ✅ Cleanup on instance shutdown

## Edge Cases

### 1. Directory Doesn't Exist
- Create directory structure on first use
- Handle permission errors gracefully

### 2. Invalid JSON Files
- Skip and log error
- Don't crash the watcher

### 3. Stale Instance Files
- Will be handled by connection manager in step 3
- For now, include all files in list

### 4. Race Conditions
- Instance writes file before port is ready
- Solution: Write file AFTER net.Listen()

### 5. File System Full
- Log error and continue without discovery
- Instance runs standalone

## Security Considerations

1. **File Permissions**
   - Instance files: 0644 (readable by user)
   - Directories: 0755
   - No sensitive data in files

2. **Path Validation**
   - Sanitize instance IDs (no path traversal)
   - Use filepath.Clean on all paths

3. **Resource Limits**
   - Limit number of watched files
   - Bounded channels for events

## Next Steps

With discovery working:
1. Step 3: Implement connection management
2. Step 4: Add tool proxying
3. Step 5: Health monitoring

## Code Checklist

- [ ] Update instance server to register after listening
- [ ] Create instance watcher with fsnotify
- [ ] Implement atomic file writes
- [ ] Add discovery to hub server
- [ ] Update instances/list to return real data
- [ ] Add comprehensive tests
- [ ] Handle all edge cases
- [ ] Document security considerations