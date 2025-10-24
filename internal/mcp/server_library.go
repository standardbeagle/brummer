package mcp

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/standardbeagle/brummer/internal/repl"
	"github.com/standardbeagle/brummer/pkg/events"
)

// LibraryParams holds parameters for library operations
type LibraryParams struct {
	Operation   string            `json:"operation"`
	Name        string            `json:"name"`
	Code        string            `json:"code"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Tags        []string          `json:"tags"`
	Examples    []string          `json:"examples"`
	Parameters  map[string]string `json:"parameters"`
	ReturnType  string            `json:"returnType"`
	Author      string            `json:"author"`
	Version     string            `json:"version"`
	Query       string            `json:"query"`
}

// Library Manager Initialization

// getLibraryManager returns the library manager, initializing it if necessary
func (s *MCPServer) getLibraryManager() *repl.LibraryManager {
	s.libraryManagerOnce.Do(func() {
		s.libraryManager = repl.NewLibraryManager()

		// Install builtin scripts in the background to avoid blocking
		// Use atomic operations to track installation status
		if atomic.CompareAndSwapInt32(&s.builtinScriptsInstalled, 0, 1) {
			go func() {
				err := repl.InstallBuiltinScripts()
				if err != nil {
					s.builtinScriptsInstallError = err
					// Log via event bus instead of stdout
					s.eventBus.Publish(events.Event{
						Type: events.SystemMessage,
						Data: map[string]interface{}{
							"level":   "warning",
							"message": fmt.Sprintf("Failed to install builtin scripts: %v", err),
						},
					})
				}
				// Mark installation complete (success or failure)
				atomic.StoreInt32(&s.builtinScriptsInstalled, 2)
			}()
		}
	})
	return s.libraryManager
}

// isBuiltinScriptsInstallationComplete returns true if builtin script installation is finished
func (s *MCPServer) isBuiltinScriptsInstallationComplete() bool {
	return atomic.LoadInt32(&s.builtinScriptsInstalled) == 2
}

// getBuiltinScriptsInstallationError returns the last installation error, if any
func (s *MCPServer) getBuiltinScriptsInstallationError() error {
	if atomic.LoadInt32(&s.builtinScriptsInstalled) == 2 {
		return s.builtinScriptsInstallError
	}
	return nil
}

// REPL Response Management

// registerREPLResponse registers a channel to receive REPL responses
func (s *MCPServer) registerREPLResponse(responseID string) chan interface{} {
	s.replMu.Lock()
	defer s.replMu.Unlock()

	responseChan := make(chan interface{}, 1)
	s.replResponseChans[responseID] = responseChan
	return responseChan
}

// unregisterREPLResponse removes the response channel for the given ID
func (s *MCPServer) unregisterREPLResponse(responseID string) {
	s.replMu.Lock()
	defer s.replMu.Unlock()

	if ch, exists := s.replResponseChans[responseID]; exists {
		close(ch)
		delete(s.replResponseChans, responseID)
	}
}

// handleREPLResponse processes an incoming REPL response from the browser
func (s *MCPServer) handleREPLResponse(responseID string, response interface{}) {
	s.replMu.RLock()
	responseChan, exists := s.replResponseChans[responseID]
	s.replMu.RUnlock()

	if exists && responseChan != nil {
		select {
		case responseChan <- response:
			// Response sent successfully
		default:
			// Channel is full or closed, ignore
		}
	}
}

// Library Operation Handlers

// handleLibraryList handles the list operation for repl_library
func (s *MCPServer) handleLibraryList() (interface{}, error) {
	libraryManager := s.getLibraryManager()
	scripts, err := libraryManager.ListScripts()
	if err != nil {
		return nil, fmt.Errorf("failed to list scripts: %w", err)
	}

	info, _ := libraryManager.GetLibraryInfo()
	return map[string]interface{}{
		"scripts":  scripts,
		"count":    len(scripts),
		"loadedAt": info.LoadedAt.Format(time.RFC3339),
		"usage": map[string]string{
			"inject":  "Use operation 'inject' to load library into browser",
			"browser": "Access via window.brummerLibrary or window.lib",
			"help":    "In browser: lib.help() for available functions",
		},
	}, nil
}

// handleLibraryAdd handles the add operation for repl_library
func (s *MCPServer) handleLibraryAdd(params LibraryParams) (interface{}, error) {
	if params.Name == "" {
		return nil, newParameterError("name", "add")
	}
	if params.Code == "" {
		return nil, newParameterError("code", "add")
	}
	if params.Description == "" {
		return nil, newParameterError("description", "add")
	}

	libraryManager := s.getLibraryManager()
	metadata := repl.ScriptMetadata{
		Description: params.Description,
		Category:    params.Category,
		Tags:        params.Tags,
		Examples:    params.Examples,
		Parameters:  params.Parameters,
		ReturnType:  params.ReturnType,
		Author:      params.Author,
		Version:     params.Version,
	}

	if err := libraryManager.AddScript(params.Name, params.Code, metadata); err != nil {
		return nil, fmt.Errorf("failed to add script '%s': %w", params.Name, err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Script '%s' added successfully", params.Name),
		"name":    params.Name,
	}, nil
}

// handleLibraryRemove handles the remove operation for repl_library
func (s *MCPServer) handleLibraryRemove(params LibraryParams) (interface{}, error) {
	if params.Name == "" {
		return nil, newParameterError("name", "remove")
	}

	libraryManager := s.getLibraryManager()
	if err := libraryManager.RemoveScript(params.Name); err != nil {
		return nil, fmt.Errorf("failed to remove script '%s': %w", params.Name, err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Script '%s' removed successfully", params.Name),
		"name":    params.Name,
	}, nil
}

// handleLibraryUpdate handles the update operation for repl_library
func (s *MCPServer) handleLibraryUpdate(params LibraryParams) (interface{}, error) {
	if params.Name == "" {
		return nil, newParameterError("name", "update")
	}
	if params.Code == "" {
		return nil, newParameterError("code", "update")
	}
	if params.Description == "" {
		return nil, newParameterError("description", "update")
	}

	libraryManager := s.getLibraryManager()
	metadata := repl.ScriptMetadata{
		Description: params.Description,
		Category:    params.Category,
		Tags:        params.Tags,
		Examples:    params.Examples,
		Parameters:  params.Parameters,
		ReturnType:  params.ReturnType,
		Author:      params.Author,
		Version:     params.Version,
	}

	if err := libraryManager.UpdateScript(params.Name, params.Code, metadata); err != nil {
		return nil, fmt.Errorf("failed to update script '%s': %w", params.Name, err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Script '%s' updated successfully", params.Name),
		"name":    params.Name,
	}, nil
}

// handleLibrarySearch handles the search operation for repl_library
func (s *MCPServer) handleLibrarySearch(params LibraryParams) (interface{}, error) {
	if params.Query == "" {
		return nil, newParameterError("query", "search")
	}

	libraryManager := s.getLibraryManager()
	results, err := libraryManager.SearchScripts(params.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to search scripts with query '%s': %w", params.Query, err)
	}

	return map[string]interface{}{
		"results": results,
		"count":   len(results),
		"query":   params.Query,
	}, nil
}

// handleLibraryCategories handles the categories operation for repl_library
func (s *MCPServer) handleLibraryCategories() (interface{}, error) {
	libraryManager := s.getLibraryManager()
	categories, err := libraryManager.GetCategories()
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	return map[string]interface{}{
		"categories": categories,
		"count":      len(categories),
	}, nil
}

// handleLibraryGet handles the get operation for repl_library
func (s *MCPServer) handleLibraryGet(params LibraryParams) (interface{}, error) {
	if params.Name == "" {
		return nil, newParameterError("name", "get")
	}

	libraryManager := s.getLibraryManager()
	script, err := libraryManager.GetScript(params.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get script '%s': %w", params.Name, err)
	}

	return map[string]interface{}{
		"name":     script.Name,
		"filename": script.Filename,
		"metadata": script.Metadata,
		"code":     script.Code,
		"filePath": script.FilePath,
		"modTime":  script.ModTime.Format(time.RFC3339),
	}, nil
}

// handleLibraryInject handles the inject operation for repl_library
func (s *MCPServer) handleLibraryInject() (interface{}, error) {
	// Check proxy server availability first
	if s.proxyServer == nil {
		return nil, newProxyNotAvailableError("injection")
	}

	libraryManager := s.getLibraryManager()

	// Generate library injection code
	injectionCode, err := libraryManager.GenerateLibraryInjectionCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate library code: %w", err)
	}

	// Execute the injection code in the browser
	responseID := fmt.Sprintf("library-inject-%d", time.Now().UnixNano())
	responseChan := s.registerREPLResponse(responseID)
	defer s.unregisterREPLResponse(responseID)

	// Send injection command via WebSocket
	s.proxyServer.BroadcastToWebSockets("command", map[string]interface{}{
		"action":     "repl",
		"code":       injectionCode,
		"responseId": responseID,
	})

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		info, _ := libraryManager.GetLibraryInfo()
		return map[string]interface{}{
			"success":      true,
			"message":      "Library injected successfully into browser",
			"scriptsCount": info.Count,
			"injectedAt":   time.Now().Format(time.RFC3339),
			"usage": map[string]string{
				"access":    "window.brummerLibrary or window.lib",
				"help":      "lib.help() - show all functions",
				"functions": "lib.help('functionName') - get function details",
				"list":      "lib.list() - list all functions",
			},
			"response": response,
		}, nil
	case <-time.After(repl.REPLResponseTimeout):
		return nil, fmt.Errorf("timeout waiting for injection response - make sure a browser window is open with the proxy URL")
	}
}
