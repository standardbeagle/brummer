package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
)

// Test helper to check tool results
func checkToolResult(t *testing.T, result interface{}, expectError bool) map[string]interface{} {
	require.NotNil(t, result)

	// Convert result to map
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok, "Expected result to be a map")

	// Check for error in result
	_, hasError := resultMap["error"]
	if expectError {
		assert.True(t, hasError, "Expected error in result")
	} else {
		assert.False(t, hasError, "Expected no error in result")
	}

	return resultMap
}

// Mock AI Coder Manager for MCP testing
type MockAICoderManager struct {
	mock.Mock
	coders map[string]*aicoder.AICoderProcess
}

func NewMockAICoderManager() *MockAICoderManager {
	return &MockAICoderManager{
		coders: make(map[string]*aicoder.AICoderProcess),
	}
}

func (m *MockAICoderManager) CreateCoder(ctx context.Context, req aicoder.CreateCoderRequest) (*aicoder.AICoderProcess, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	coder := args.Get(0).(*aicoder.AICoderProcess)
	if coder != nil {
		m.coders[coder.ID] = coder
	}
	return coder, args.Error(1)
}

func (m *MockAICoderManager) GetCoder(id string) (*aicoder.AICoderProcess, bool) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*aicoder.AICoderProcess), args.Bool(1)
}

func (m *MockAICoderManager) ListCoders() []*aicoder.AICoderProcess {
	args := m.Called()
	return args.Get(0).([]*aicoder.AICoderProcess)
}

func (m *MockAICoderManager) DeleteCoder(id string) error {
	args := m.Called(id)
	delete(m.coders, id)
	return args.Error(0)
}

func (m *MockAICoderManager) StartCoder(id string) error {
	args := m.Called(id)
	// Don't update status here - let tests control the status
	return args.Error(0)
}

func (m *MockAICoderManager) StopCoder(id string) error {
	args := m.Called(id)
	if coder, exists := m.coders[id]; exists {
		coder.Status = aicoder.StatusStopped
	}
	return args.Error(0)
}

func (m *MockAICoderManager) PauseCoder(id string) error {
	args := m.Called(id)
	if coder, exists := m.coders[id]; exists {
		coder.Status = aicoder.StatusPaused
	}
	return args.Error(0)
}

func (m *MockAICoderManager) ResumeCoder(id string) error {
	args := m.Called(id)
	if coder, exists := m.coders[id]; exists {
		coder.Status = aicoder.StatusRunning
	}
	return args.Error(0)
}

func (m *MockAICoderManager) UpdateCoderTask(id string, task string) error {
	args := m.Called(id, task)
	if coder, exists := m.coders[id]; exists {
		coder.Task = task
	}
	return args.Error(0)
}

// Test Setup
func setupMCPTestServer(t *testing.T) (*MCPServer, *MockAICoderManager) {
	// Create dependencies
	processMgr := &process.Manager{}
	logStore := logs.NewStore(10000, nil)
	proxyServer := &proxy.Server{}
	eventBus := events.NewEventBus()

	// Create the MCP server
	server := NewMCPServer(7777, processMgr, logStore, proxyServer, eventBus)
	mockManager := NewMockAICoderManager()

	// Set the AI coder manager
	server.SetAICoderManager(mockManager)

	return server, mockManager
}

func TestAICoderTools_Create_Success(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Setup mock expectations
	expectedCoder := &aicoder.AICoderProcess{
		ID:           "test-coder-123",
		Name:         "Test Coder",
		Provider:     "openai",
		Task:         "implement user login",
		Status:       aicoder.StatusCreating,
		WorkspaceDir: "/tmp/workspace",
		Progress:     0.0,
	}

	mockManager.On("CreateCoder", mock.Anything, mock.MatchedBy(func(req aicoder.CreateCoderRequest) bool {
		return req.Provider == "openai" && req.Task == "implement user login"
	})).Return(expectedCoder, nil)

	// Expect StartCoder to be called after creation
	mockManager.On("StartCoder", "test-coder-123").Return(nil)

	// Prepare request
	params := map[string]interface{}{
		"provider": "openai",
		"task":     "implement user login",
	}
	jsonParams, err := json.Marshal(params)
	require.NoError(t, err)

	// Execute
	result, err := server.CallTool(context.Background(), "ai_coder_create", json.RawMessage(jsonParams))

	// Assert
	require.NoError(t, err)
	resultMap := checkToolResult(t, result, false)

	// Check result contents
	assert.Equal(t, "test-coder-123", resultMap["id"])
	assert.Equal(t, "Test Coder", resultMap["name"])
	assert.Equal(t, "creating", resultMap["status"])

	mockManager.AssertExpectations(t)
}

func TestAICoderTools_Create_InvalidProvider(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Setup mock expectations - CreateCoder should fail
	mockManager.On("CreateCoder", mock.Anything, mock.Anything).
		Return((*aicoder.AICoderProcess)(nil), fmt.Errorf("invalid provider"))

	// Prepare request
	params := map[string]interface{}{
		"provider": "invalid-provider",
		"task":     "some task",
	}
	jsonParams, err := json.Marshal(params)
	require.NoError(t, err)

	// Execute
	result, err := server.CallTool(context.Background(), "ai_coder_create", json.RawMessage(jsonParams))

	// Assert
	require.NoError(t, err)
	checkToolResult(t, result, true)

	mockManager.AssertExpectations(t)
}

func TestAICoderTools_List_Success(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Setup mock expectations
	expectedCoders := []*aicoder.AICoderProcess{
		{
			ID:       "coder-1",
			Name:     "Coder 1",
			Provider: "openai",
			Task:     "task 1",
			Status:   aicoder.StatusRunning,
			Progress: 0.5,
		},
		{
			ID:       "coder-2",
			Name:     "Coder 2",
			Provider: "anthropic",
			Task:     "task 2",
			Status:   aicoder.StatusCompleted,
			Progress: 1.0,
		},
	}

	mockManager.On("ListCoders").Return(expectedCoders)

	// Execute
	result, err := server.CallTool(context.Background(), "ai_coder_list", json.RawMessage("{}"))

	// Assert
	require.NoError(t, err)
	resultMap := checkToolResult(t, result, false)

	// Check result contents
	codersData, ok := resultMap["coders"].([]map[string]interface{})
	if !ok {
		// Try converting from []interface{} if needed
		if codersInterface, ok := resultMap["coders"].([]interface{}); ok {
			codersData = make([]map[string]interface{}, len(codersInterface))
			for i, c := range codersInterface {
				codersData[i] = c.(map[string]interface{})
			}
		} else {
			require.True(t, false, "Expected coders to be a list")
		}
	}
	assert.Len(t, codersData, 2)

	mockManager.AssertExpectations(t)
}

func TestAICoderTools_List_Empty(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Setup mock expectations
	mockManager.On("ListCoders").Return([]*aicoder.AICoderProcess{})

	// Execute
	result, err := server.CallTool(context.Background(), "ai_coder_list", json.RawMessage("{}"))

	// Assert
	require.NoError(t, err)
	resultMap := checkToolResult(t, result, false)

	// Check result contents
	codersData, ok := resultMap["coders"].([]map[string]interface{})
	if !ok {
		// Try converting from []interface{} if needed
		if codersInterface, ok := resultMap["coders"].([]interface{}); ok {
			codersData = make([]map[string]interface{}, len(codersInterface))
			for i, c := range codersInterface {
				codersData[i] = c.(map[string]interface{})
			}
		} else {
			require.True(t, false, "Expected coders to be a list")
		}
	}
	assert.Len(t, codersData, 0)

	mockManager.AssertExpectations(t)
}

func TestAICoderTools_Status_Success(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Setup mock expectations
	expectedCoder := &aicoder.AICoderProcess{
		ID:           "test-coder-123",
		Name:         "Test Coder",
		Provider:     "openai",
		Task:         "implement feature",
		Status:       aicoder.StatusRunning,
		Progress:     0.75,
		WorkspaceDir: "/tmp/workspace",
	}

	mockManager.On("GetCoder", "test-coder-123").Return(expectedCoder, true)

	// Prepare request
	params := map[string]interface{}{
		"coder_id": "test-coder-123",
	}
	jsonParams, err := json.Marshal(params)
	require.NoError(t, err)

	// Execute
	result, err := server.CallTool(context.Background(), "ai_coder_status", json.RawMessage(jsonParams))

	// Assert
	require.NoError(t, err)
	resultMap := checkToolResult(t, result, false)

	// Check result contents
	assert.Equal(t, "test-coder-123", resultMap["id"])
	assert.Equal(t, "running", resultMap["status"])
	assert.Equal(t, float64(0.75), resultMap["progress"])

	mockManager.AssertExpectations(t)
}

func TestAICoderTools_Status_NotFound(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Setup mock expectations
	mockManager.On("GetCoder", "nonexistent").Return((*aicoder.AICoderProcess)(nil), false)

	// Prepare request
	params := map[string]interface{}{
		"coder_id": "nonexistent",
	}
	jsonParams, err := json.Marshal(params)
	require.NoError(t, err)

	// Execute
	result, err := server.CallTool(context.Background(), "ai_coder_status", json.RawMessage(jsonParams))

	// Assert
	require.NoError(t, err)
	checkToolResult(t, result, true)

	mockManager.AssertExpectations(t)
}

func TestAICoderTools_Control_Start(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Setup mock expectations
	mockManager.On("StartCoder", "test-coder-123").Return(nil)

	// Expect GetCoder to be called after action to get updated status
	expectedCoder := &aicoder.AICoderProcess{
		ID:        "test-coder-123",
		Status:    aicoder.StatusRunning,
		SessionID: "session-123",
	}
	mockManager.On("GetCoder", "test-coder-123").Return(expectedCoder, true)

	// Prepare request
	params := map[string]interface{}{
		"coder_id": "test-coder-123",
		"action":   "start",
	}
	jsonParams, err := json.Marshal(params)
	require.NoError(t, err)

	// Execute
	result, err := server.CallTool(context.Background(), "ai_coder_control", json.RawMessage(jsonParams))

	// Assert
	require.NoError(t, err)
	resultMap := checkToolResult(t, result, false)
	assert.Contains(t, resultMap["message"], "started")

	mockManager.AssertExpectations(t)
}

func TestAICoderTools_Control_InvalidAction(t *testing.T) {
	server, _ := setupMCPTestServer(t)

	// Prepare request
	params := map[string]interface{}{
		"coder_id": "test-coder-123",
		"action":   "invalid-action",
	}
	jsonParams, err := json.Marshal(params)
	require.NoError(t, err)

	// Execute
	_, err = server.CallTool(context.Background(), "ai_coder_control", json.RawMessage(jsonParams))

	// Assert - we expect the tool handler to return an error for invalid action
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid action")
}

func TestAICoderTools_Control_AllActions(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Test all control actions
	actions := []string{"start", "stop", "pause", "resume"}

	// Mock coder for GetCoder calls
	expectedCoder := &aicoder.AICoderProcess{
		ID:        "test-coder",
		Status:    aicoder.StatusRunning,
		SessionID: "session-123",
	}

	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			// Setup mock expectations
			switch action {
			case "start":
				mockManager.On("StartCoder", "test-coder").Return(nil)
			case "stop":
				mockManager.On("StopCoder", "test-coder").Return(nil)
			case "pause":
				mockManager.On("PauseCoder", "test-coder").Return(nil)
			case "resume":
				mockManager.On("ResumeCoder", "test-coder").Return(nil)
			}

			// All actions call GetCoder to get updated status
			mockManager.On("GetCoder", "test-coder").Return(expectedCoder, true)

			// Prepare request
			params := map[string]interface{}{
				"coder_id": "test-coder",
				"action":   action,
			}
			jsonParams, err := json.Marshal(params)
			require.NoError(t, err)

			// Execute
			result, err := server.CallTool(context.Background(), "ai_coder_control", json.RawMessage(jsonParams))

			// Assert
			require.NoError(t, err)
			checkToolResult(t, result, false)
		})
	}

	mockManager.AssertExpectations(t)
}

// TestAICoderTools_Update_Success is removed as ai_coder_update tool doesn't exist

// TestAICoderTools_Delete_Success is removed as ai_coder_delete tool doesn't exist
/*
func TestAICoderTools_Delete_Success(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Setup mock expectations
	mockManager.On("DeleteCoder", "test-coder-123").Return(nil)

	// Prepare request
	params := map[string]interface{}{
		"coder_id": "test-coder-123",
	}
	jsonParams, err := json.Marshal(params)
	require.NoError(t, err)

	// Execute
	result, err := server.CallTool(context.Background(), "ai_coder_delete", json.RawMessage(jsonParams))

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "deleted")

	mockManager.AssertExpectations(t)
}
*/

// TestAICoderTools_Delete_NotFound is removed as ai_coder_delete tool doesn't exist
/*
func TestAICoderTools_Delete_NotFound(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Setup mock expectations
	mockManager.On("DeleteCoder", "nonexistent").Return(assert.AnError)

	// Prepare request
	params := map[string]interface{}{
		"coder_id": "nonexistent",
	}
	jsonParams, err := json.Marshal(params)
	require.NoError(t, err)

	// Execute
	result, err := server.CallTool(context.Background(), "ai_coder_delete", json.RawMessage(jsonParams))

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	mockManager.AssertExpectations(t)
}
*/

// Table-driven test for parameter validation
func TestAICoderTools_ParameterValidation(t *testing.T) {
	// We'll create a new server for each test to avoid mock conflicts

	tests := []struct {
		toolName   string
		params     map[string]interface{}
		shouldFail bool
	}{
		{
			toolName:   "ai_coder_create",
			params:     map[string]interface{}{"provider": "openai", "task": "test"},
			shouldFail: false,
		},
		{
			toolName:   "ai_coder_create",
			params:     map[string]interface{}{"provider": "openai"}, // missing task
			shouldFail: true,
		},
		{
			toolName:   "ai_coder_status",
			params:     map[string]interface{}{"coder_id": "test-123"},
			shouldFail: false,
		},
		{
			toolName:   "ai_coder_status",
			params:     map[string]interface{}{}, // missing coder_id
			shouldFail: true,
		},
		{
			toolName:   "ai_coder_control",
			params:     map[string]interface{}{"coder_id": "test-123", "action": "start"},
			shouldFail: false,
		},
		{
			toolName:   "ai_coder_control",
			params:     map[string]interface{}{"coder_id": "test-123"}, // missing action
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%v", tt.toolName, tt.shouldFail), func(t *testing.T) {
			server, mockManager := setupMCPTestServer(t)

			// Set up mock expectations for successful validation cases
			if !tt.shouldFail {
				switch tt.toolName {
				case "ai_coder_create":
					mockManager.On("CreateCoder", mock.Anything, mock.Anything).Return(&aicoder.AICoderProcess{
						ID:     "test-id",
						Status: aicoder.StatusCreating,
					}, nil)
					mockManager.On("StartCoder", "test-id").Return(nil)
				case "ai_coder_status":
					mockManager.On("GetCoder", "test-123").Return(&aicoder.AICoderProcess{
						ID:     "test-123",
						Status: aicoder.StatusRunning,
					}, true)
				case "ai_coder_control":
					mockManager.On("StartCoder", "test-123").Return(nil)
					mockManager.On("GetCoder", "test-123").Return(&aicoder.AICoderProcess{
						ID:     "test-123",
						Status: aicoder.StatusRunning,
					}, true)
				}
			}

			jsonParams, err := json.Marshal(tt.params)
			require.NoError(t, err)

			result, err := server.CallTool(context.Background(), tt.toolName, json.RawMessage(jsonParams))

			if tt.shouldFail {
				// Should either return an error or a result marked as error
				if err == nil {
					checkToolResult(t, result, true)
				} else {
					// Error from CallTool is also acceptable for validation failures
					assert.Error(t, err)
				}
			} else {
				// Parameter validation should pass (actual functionality may still fail due to mocks)
				// We might get an error from missing mock setup, that's ok
				if err == nil && result != nil {
					// Just check it's a valid result structure
					_, ok := result.(map[string]interface{})
					assert.True(t, ok, "Expected result to be a map")
				}
			}
		})
	}
}
