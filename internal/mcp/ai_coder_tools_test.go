package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/standardbeagle/brummer/internal/aicoder"
)

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
	m.coders[coder.ID] = coder
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
	if coder, exists := m.coders[id]; exists {
		coder.Status = aicoder.StatusRunning
	}
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
func setupMCPTestServer(t *testing.T) (*Server, *MockAICoderManager) {
	server := NewServer()
	mockManager := NewMockAICoderManager()
	
	// Register AI coder tools
	RegisterAICoderTools(server, mockManager)
	
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
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 1)
	assert.Contains(t, result.Content[0].Text, "test-coder-123")
	assert.Contains(t, result.Content[0].Text, "Test Coder")
	assert.Contains(t, result.Content[0].Text, "creating")

	mockManager.AssertExpectations(t)
}

func TestAICoderTools_Create_InvalidProvider(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Setup mock expectations
	mockManager.On("CreateCoder", mock.Anything, mock.Anything).
		Return((*aicoder.AICoderProcess)(nil), assert.AnError)

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
	require.NotNil(t, result)
	assert.True(t, result.IsError)

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
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 1)
	
	content := result.Content[0].Text
	assert.Contains(t, content, "coder-1")
	assert.Contains(t, content, "coder-2")
	assert.Contains(t, content, "running")
	assert.Contains(t, content, "completed")

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
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 1)
	assert.Contains(t, result.Content[0].Text, "No AI coders")

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
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 1)
	
	content := result.Content[0].Text
	assert.Contains(t, content, "test-coder-123")
	assert.Contains(t, content, "running")
	assert.Contains(t, content, "75%")

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
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	mockManager.AssertExpectations(t)
}

func TestAICoderTools_Control_Start(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Setup mock expectations
	mockManager.On("StartCoder", "test-coder-123").Return(nil)

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
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "started")

	mockManager.AssertExpectations(t)
}

func TestAICoderTools_Control_InvalidAction(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Prepare request
	params := map[string]interface{}{
		"coder_id": "test-coder-123",
		"action":   "invalid-action",
	}
	jsonParams, err := json.Marshal(params)
	require.NoError(t, err)

	// Execute
	result, err := server.CallTool(context.Background(), "ai_coder_control", json.RawMessage(jsonParams))

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "invalid action")
}

func TestAICoderTools_Control_AllActions(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Test all control actions
	actions := []string{"start", "stop", "pause", "resume"}

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
			require.NotNil(t, result)
			assert.False(t, result.IsError, "Action %s should succeed", action)
		})
	}

	mockManager.AssertExpectations(t)
}

func TestAICoderTools_Update_Success(t *testing.T) {
	server, mockManager := setupMCPTestServer(t)

	// Setup mock expectations
	mockManager.On("UpdateCoderTask", "test-coder-123", "new task description").Return(nil)

	// Prepare request
	params := map[string]interface{}{
		"coder_id": "test-coder-123",
		"task":     "new task description",
	}
	jsonParams, err := json.Marshal(params)
	require.NoError(t, err)

	// Execute
	result, err := server.CallTool(context.Background(), "ai_coder_update", json.RawMessage(jsonParams))

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "updated")

	mockManager.AssertExpectations(t)
}

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

// Table-driven test for parameter validation
func TestAICoderTools_ParameterValidation(t *testing.T) {
	server, _ := setupMCPTestServer(t)

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
			jsonParams, err := json.Marshal(tt.params)
			require.NoError(t, err)

			result, err := server.CallTool(context.Background(), tt.toolName, json.RawMessage(jsonParams))

			if tt.shouldFail {
				// Should either return an error or a result marked as error
				if err == nil {
					require.NotNil(t, result)
					assert.True(t, result.IsError, "Expected validation error for %s with params %v", tt.toolName, tt.params)
				}
			} else {
				// Parameter validation should pass (actual functionality may still fail due to mocks)
				require.NoError(t, err)
			}
		})
	}
}