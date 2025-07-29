package aicoder

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock AI Provider for testing
type MockAIProvider struct {
	mock.Mock
}

func (m *MockAIProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockAIProvider) GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error) {
	args := m.Called(ctx, prompt, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GenerateResult), args.Error(1)
}

func (m *MockAIProvider) StreamGenerate(ctx context.Context, prompt string, options GenerateOptions) (<-chan GenerateUpdate, error) {
	args := m.Called(ctx, prompt, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan GenerateUpdate), args.Error(1)
}

func (m *MockAIProvider) ValidateConfig(config ProviderConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockAIProvider) GetCapabilities() ProviderCapabilities {
	args := m.Called()
	return args.Get(0).(ProviderCapabilities)
}

// Mock Event Bus
type MockEventBus struct {
	mock.Mock
	events []interface{}
}

func (m *MockEventBus) Emit(eventType string, data interface{}) {
	m.Called(eventType, data)
	m.events = append(m.events, data)
}

func (m *MockEventBus) Subscribe(eventType string, handler func(interface{})) {
	m.Called(eventType, handler)
}

// Test Setup
func setupTestManager(t *testing.T) (*AICoderManager, *MockEventBus, string) {
	tmpDir := t.TempDir()

	eventBus := &MockEventBus{}
	config := &TestConfig{
		WorkspaceBaseDir: tmpDir,
		MaxConcurrent:    3,
		DefaultProvider:  "mock",
		TimeoutMinutes:   10,
	}

	manager, err := NewAICoderManagerWithoutMockProvider(config, eventBus)
	require.NoError(t, err)

	return manager, eventBus, tmpDir
}

// Helper to register mock provider
func registerMockProvider(t *testing.T, manager *AICoderManager, name string) *MockAIProvider {
	mockProvider := &MockAIProvider{}
	mockProvider.On("Name").Return(name)
	mockProvider.On("GetCapabilities").Return(ProviderCapabilities{
		SupportsStreaming: true,
		MaxContextTokens:  100000,
		MaxOutputTokens:   4096,
		SupportedModels:   []string{"mock-model"},
	})
	// Add GenerateCode expectation for the provider validation call
	mockProvider.On("GenerateCode", mock.Anything, mock.Anything, mock.Anything).
		Return(&GenerateResult{
			Code:    "// Mock generated code",
			Summary: "Mock generation successful",
		}, nil)
	err := manager.RegisterProvider(name, mockProvider)
	require.NoError(t, err)
	return mockProvider
}

// Unit Tests
func TestAICoderManager_CreateCoder_Success(t *testing.T) {
	manager, eventBus, _ := setupTestManager(t)
	registerMockProvider(t, manager, "mock")

	req := CreateCoderRequest{
		Provider: "mock",
		Task:     "implement user authentication",
	}

	// Set up mock expectations
	eventBus.On("Emit", "ai_coder_created", mock.Anything)

	// Execute
	coder, err := manager.CreateCoder(context.Background(), req)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, coder)
	assert.Equal(t, StatusCreating, coder.Status)
	assert.Equal(t, "mock", coder.Provider)
	assert.Equal(t, req.Task, coder.Task)
	assert.DirExists(t, coder.WorkspaceDir)

	// Verify event was emitted
	eventBus.AssertCalled(t, "Emit", "ai_coder_created", mock.Anything)
}

func TestAICoderManager_CreateCoder_InvalidProvider(t *testing.T) {
	manager, _, _ := setupTestManager(t)

	req := CreateCoderRequest{
		Provider: "nonexistent",
		Task:     "some task",
	}

	// Execute
	coder, err := manager.CreateCoder(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, coder)
	assert.Contains(t, err.Error(), "provider nonexistent not found")
}

func TestAICoderManager_CreateCoder_WorkspaceError(t *testing.T) {
	manager, _, tmpDir := setupTestManager(t)

	// Make workspace directory read-only to cause error
	err := os.Chmod(tmpDir, 0444)
	require.NoError(t, err)
	defer os.Chmod(tmpDir, 0755)

	req := CreateCoderRequest{
		Provider: "mock",
		Task:     "some task",
	}

	// Execute
	coder, err := manager.CreateCoder(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, coder)
}

func TestAICoderManager_GetCoder_Exists(t *testing.T) {
	manager, eventBus, _ := setupTestManager(t)
	registerMockProvider(t, manager, "mock")

	// Create a coder first
	eventBus.On("Emit", mock.Anything, mock.Anything)
	coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
		Provider: "mock",
		Task:     "test task",
	})
	require.NoError(t, err)

	// Execute
	retrieved, exists := manager.GetCoder(coder.ID)

	// Assert
	assert.True(t, exists)
	assert.Equal(t, coder.ID, retrieved.ID)
}

func TestAICoderManager_GetCoder_NotExists(t *testing.T) {
	manager, _, _ := setupTestManager(t)

	// Execute
	coder, exists := manager.GetCoder("nonexistent-id")

	// Assert
	assert.False(t, exists)
	assert.Nil(t, coder)
}

func TestAICoderManager_ListCoders_Empty(t *testing.T) {
	manager, _, _ := setupTestManager(t)

	// Execute
	coders := manager.ListCoders()

	// Assert
	assert.Empty(t, coders)
}

func TestAICoderManager_ListCoders_Multiple(t *testing.T) {
	manager, eventBus, _ := setupTestManager(t)
	registerMockProvider(t, manager, "mock")
	eventBus.On("Emit", mock.Anything, mock.Anything)

	// Create multiple coders
	req1 := CreateCoderRequest{Provider: "mock", Task: "task 1"}
	req2 := CreateCoderRequest{Provider: "mock", Task: "task 2"}

	coder1, err := manager.CreateCoder(context.Background(), req1)
	require.NoError(t, err)

	coder2, err := manager.CreateCoder(context.Background(), req2)
	require.NoError(t, err)

	// Execute
	coders := manager.ListCoders()

	// Assert
	assert.Len(t, coders, 2)
	coderIDs := []string{coders[0].ID, coders[1].ID}
	assert.Contains(t, coderIDs, coder1.ID)
	assert.Contains(t, coderIDs, coder2.ID)
}

func TestAICoderManager_DeleteCoder_Success(t *testing.T) {
	manager, eventBus, _ := setupTestManager(t)
	registerMockProvider(t, manager, "mock")
	eventBus.On("Emit", mock.Anything, mock.Anything)

	// Create a coder first
	coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
		Provider: "mock",
		Task:     "test task",
	})
	require.NoError(t, err)

	// Execute
	err = manager.DeleteCoder(coder.ID)

	// Assert
	assert.NoError(t, err)

	// Verify coder is gone
	_, exists := manager.GetCoder(coder.ID)
	assert.False(t, exists)

	// Verify workspace is cleaned up
	assert.NoDirExists(t, coder.WorkspaceDir)
}

func TestAICoderManager_DeleteCoder_NotExists(t *testing.T) {
	manager, _, _ := setupTestManager(t)

	// Execute
	err := manager.DeleteCoder("nonexistent-id")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// Concurrent access tests
func TestAICoderManager_ConcurrentAccess(t *testing.T) {
	manager, eventBus, _ := setupTestManager(t)
	registerMockProvider(t, manager, "mock")
	eventBus.On("Emit", mock.Anything, mock.Anything)

	const numGoroutines = 10
	const numOperations = 50

	// Start multiple goroutines performing operations
	done := make(chan error, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			var err error
			defer func() { done <- err }()

			for j := 0; j < numOperations; j++ {
				// Create coder
				coder, createErr := manager.CreateCoder(context.Background(), CreateCoderRequest{
					Provider: "mock",
					Task:     fmt.Sprintf("task-%d-%d", id, j),
				})
				if createErr != nil {
					err = createErr
					return
				}

				// List coders
				coders := manager.ListCoders()
				assert.NotEmpty(t, coders)

				// Get coder
				retrieved, exists := manager.GetCoder(coder.ID)
				if !exists || retrieved == nil {
					err = fmt.Errorf("failed to retrieve coder %s", coder.ID)
					return
				}

				// Delete coder
				deleteErr := manager.DeleteCoder(coder.ID)
				if deleteErr != nil {
					err = deleteErr
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-done:
			require.NoError(t, err, "Goroutine %d failed", i)
		case <-time.After(30 * time.Second):
			t.Fatal("Test timed out")
		}
	}
}

// Table-driven tests for status transitions
func TestAICoderManager_StatusTransitions(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  AICoderStatus
		operation      string
		expectedStatus AICoderStatus
		expectedError  bool
	}{
		{
			name:           "Start from Creating",
			initialStatus:  StatusCreating,
			operation:      "start",
			expectedStatus: StatusRunning,
			expectedError:  false,
		},
		{
			name:           "Pause from Running",
			initialStatus:  StatusRunning,
			operation:      "pause",
			expectedStatus: StatusPaused,
			expectedError:  false,
		},
		{
			name:           "Resume from Paused",
			initialStatus:  StatusPaused,
			operation:      "resume",
			expectedStatus: StatusRunning,
			expectedError:  false,
		},
		{
			name:           "Stop from Running",
			initialStatus:  StatusRunning,
			operation:      "stop",
			expectedStatus: StatusStopped,
			expectedError:  false,
		},
		{
			name:           "Invalid transition",
			initialStatus:  StatusCompleted,
			operation:      "start",
			expectedStatus: StatusCompleted,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, eventBus, _ := setupTestManager(t)
			registerMockProvider(t, manager, "mock")
			eventBus.On("Emit", mock.Anything, mock.Anything)

			// Create coder and set initial status
			coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
				Provider: "mock",
				Task:     "test task",
			})
			require.NoError(t, err)

			// Setup initial state properly
			if tt.initialStatus == StatusRunning || tt.initialStatus == StatusPaused {
				// Start the coder first so it's managed by the process manager
				err = manager.StartCoder(coder.ID)
				require.NoError(t, err)
				// Then set the desired initial status after starting
				if tt.initialStatus == StatusPaused {
					coder.SetStatus(StatusPaused)
				}
			} else {
				// Just set the status for non-running states
				coder.Status = tt.initialStatus
			}

			// Execute operation
			var opErr error
			switch tt.operation {
			case "start":
				opErr = manager.StartCoder(coder.ID)
			case "pause":
				opErr = manager.PauseCoder(coder.ID)
			case "resume":
				opErr = manager.ResumeCoder(coder.ID)
			case "stop":
				opErr = manager.StopCoder(coder.ID)
			}

			// Assert
			if tt.expectedError {
				assert.Error(t, opErr)
			} else {
				assert.NoError(t, opErr)
			}

			// Check final status
			retrieved, exists := manager.GetCoder(coder.ID)
			require.True(t, exists)
			assert.Equal(t, tt.expectedStatus, retrieved.Status)
		})
	}
}

func TestAICoderManager_MaxConcurrentLimit(t *testing.T) {
	manager, eventBus, _ := setupTestManager(t)
	registerMockProvider(t, manager, "mock")
	eventBus.On("Emit", mock.Anything, mock.Anything)

	// Create maximum number of coders (3)
	var coders []*AICoderProcess
	for i := 0; i < 3; i++ {
		coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
			Provider: "mock",
			Task:     fmt.Sprintf("task %d", i),
		})
		require.NoError(t, err)
		coders = append(coders, coder)

		// Start each coder
		err = manager.StartCoder(coder.ID)
		require.NoError(t, err)
	}

	// Try to create one more - should fail due to limit
	coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
		Provider: "mock",
		Task:     "excess task",
	})
	assert.Error(t, err)
	assert.Nil(t, coder)
	assert.Contains(t, err.Error(), "maximum concurrent")

	// Stop one coder
	err = manager.StopCoder(coders[0].ID)
	require.NoError(t, err)

	// Now should be able to create another
	coder, err = manager.CreateCoder(context.Background(), CreateCoderRequest{
		Provider: "mock",
		Task:     "new task",
	})
	assert.NoError(t, err)
	assert.NotNil(t, coder)
}

// TestConfig implementation for testing
type TestConfig struct {
	WorkspaceBaseDir string
	MaxConcurrent    int
	DefaultProvider  string
	TimeoutMinutes   int
}

func (c *TestConfig) GetAICoderConfig() AICoderConfig {
	return AICoderConfig{
		WorkspaceBaseDir: c.WorkspaceBaseDir,
		MaxConcurrent:    c.MaxConcurrent,
		DefaultProvider:  c.DefaultProvider,
		TimeoutMinutes:   c.TimeoutMinutes,
	}
}