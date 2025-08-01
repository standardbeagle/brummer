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

func TestAICoderIntegration_FullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup
	manager, eventBus, tmpDir := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, tmpDir)

	// Set up event bus mock expectations
	eventBus.On("Emit", mock.Anything, mock.Anything)

	// Test data
	req := CreateCoderRequest{
		Provider: "mock",
		Task:     "Create a simple REST API with authentication",
		WorkspaceFiles: []string{
			"main.go",
			"auth.go",
			"handlers.go",
		},
	}

	// Step 1: Create AI coder
	coder, err := manager.CreateCoder(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, StatusCreating, coder.Status)

	// Verify workspace created
	assert.DirExists(t, coder.WorkspaceDir)

	// Step 2: Start AI coder
	err = manager.StartCoder(coder.ID)
	require.NoError(t, err)

	// Wait for status update
	time.Sleep(100 * time.Millisecond)

	updated, exists := manager.GetCoder(coder.ID)
	require.True(t, exists)
	assert.Equal(t, StatusRunning, updated.Status)

	// Step 3: Simulate some progress
	err = updated.UpdateProgress(0.5, "Generated authentication module")
	require.NoError(t, err)

	// Step 4: Create some workspace files
	err = updated.WriteFile("main.go", []byte("package main\n\nfunc main() {\n\t// TODO: implement\n}"))
	require.NoError(t, err)

	err = updated.WriteFile("auth.go", []byte("package main\n\n// Authentication module"))
	require.NoError(t, err)

	// Verify files exist
	files, err := updated.ListWorkspaceFiles()
	require.NoError(t, err)
	assert.Contains(t, files, "main.go")
	assert.Contains(t, files, "auth.go")

	// Step 5: Complete the task
	err = updated.UpdateProgress(1.0, "Task completed successfully")
	require.NoError(t, err)

	updated.SetStatus(StatusCompleted)

	// Step 6: Verify final state
	final, exists := manager.GetCoder(coder.ID)
	require.True(t, exists)
	assert.Equal(t, StatusCompleted, final.Status)
	assert.Equal(t, 1.0, final.Progress)

	// Verify events were emitted
	assert.Greater(t, len(eventBus.events), 0)
}

func TestAICoderIntegration_WorkspaceOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup
	manager, eventBus, tmpDir := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, tmpDir)

	// Set up event bus mock expectations
	eventBus.On("Emit", mock.Anything, mock.Anything)

	// Create coder
	coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
		Provider: "mock",
		Task:     "test workspace operations",
	})
	require.NoError(t, err)

	// Test file operations
	testCases := []struct {
		filename string
		content  string
	}{
		{"hello.go", "package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}"},
		{"README.md", "# Test Project\n\nThis is a test project."},
		{"config.json", "{\"name\": \"test\", \"version\": \"1.0.0\"}"},
	}

	// Write files
	for _, tc := range testCases {
		err := coder.WriteFile(tc.filename, []byte(tc.content))
		require.NoError(t, err, "Failed to write %s", tc.filename)
	}

	// List files
	files, err := coder.ListWorkspaceFiles()
	require.NoError(t, err)

	for _, tc := range testCases {
		assert.Contains(t, files, tc.filename)
	}

	// Read files back
	for _, tc := range testCases {
		content, err := coder.ReadFile(tc.filename)
		require.NoError(t, err, "Failed to read %s", tc.filename)
		assert.Equal(t, tc.content, string(content))
	}

	// Test path validation (security)
	err = coder.WriteFile("../outside.txt", []byte("should fail"))
	assert.Error(t, err, "Should prevent writing outside workspace")

	err = coder.WriteFile("/etc/passwd", []byte("should fail"))
	assert.Error(t, err, "Should prevent writing to system files")
}

func TestAICoderIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup
	manager, eventBus, tmpDir := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, tmpDir)

	// Set up event bus mock expectations
	eventBus.On("Emit", mock.Anything, mock.Anything)

	// Test provider error handling
	mockProvider := &MockAIProvider{}
	mockProvider.On("Name").Return("failing-provider")
	mockProvider.On("GetCapabilities").Return(ProviderCapabilities{
		SupportsStreaming: false,
		MaxContextTokens:  1000,
		MaxOutputTokens:   100,
		SupportedModels:   []string{"failing-model"},
	})
	mockProvider.On("GenerateCode", mock.Anything, mock.Anything, mock.Anything).
		Return((*GenerateResult)(nil), assert.AnError)

	regErr := manager.RegisterProvider("failing-provider", mockProvider)
	require.NoError(t, regErr)

	// Create coder with failing provider
	coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
		Provider: "failing-provider",
		Task:     "test error handling",
	})
	require.NoError(t, err)

	// Start coder - should handle provider failure gracefully
	err = manager.StartCoder(coder.ID)
	assert.Error(t, err)

	// Verify coder status reflects failure
	updated, exists := manager.GetCoder(coder.ID)
	require.True(t, exists)
	assert.Equal(t, StatusFailed, updated.Status)
}

func TestAICoderIntegration_ConcurrentWorkspaceAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup
	manager, eventBus, tmpDir := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, tmpDir)

	eventBus.On("Emit", mock.Anything, mock.Anything)

	// Create a coder
	coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
		Provider: "mock",
		Task:     "concurrent workspace test",
	})
	require.NoError(t, err)

	const numWorkers = 5
	const filesPerWorker = 10

	done := make(chan error, numWorkers)

	// Start multiple workers writing files concurrently
	for worker := 0; worker < numWorkers; worker++ {
		go func(workerID int) {
			var err error
			defer func() { done <- err }()

			for i := 0; i < filesPerWorker; i++ {
				filename := fmt.Sprintf("worker_%d_file_%d.txt", workerID, i)
				content := fmt.Sprintf("Content from worker %d, file %d", workerID, i)

				writeErr := coder.WriteFile(filename, []byte(content))
				if writeErr != nil {
					err = writeErr
					return
				}

				// Verify we can read it back
				readContent, readErr := coder.ReadFile(filename)
				if readErr != nil {
					err = readErr
					return
				}

				if string(readContent) != content {
					err = fmt.Errorf("content mismatch for %s", filename)
					return
				}
			}
		}(worker)
	}

	// Wait for all workers to complete
	for i := 0; i < numWorkers; i++ {
		select {
		case err := <-done:
			require.NoError(t, err, "Worker %d failed", i)
		case <-time.After(10 * time.Second):
			t.Fatal("Concurrent workspace test timed out")
		}
	}

	// Verify all files were created
	files, err := coder.ListWorkspaceFiles()
	require.NoError(t, err)
	// Account for the metadata file created by WorkspaceManager
	expectedFiles := numWorkers*filesPerWorker + 1 // +1 for .aicoder/metadata.json
	assert.Len(t, files, expectedFiles)
}

func TestAICoderIntegration_WorkspaceCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup
	manager, eventBus, tmpDir := setupIntegrationTest(t)
	defer cleanupIntegrationTest(t, tmpDir)

	// Set up event bus mock expectations
	eventBus.On("Emit", mock.Anything, mock.Anything)

	// Create multiple coders
	var coders []*AICoderProcess
	var workspaceDirs []string

	for i := 0; i < 3; i++ {
		coder, err := manager.CreateCoder(context.Background(), CreateCoderRequest{
			Provider: "mock",
			Task:     fmt.Sprintf("workspace cleanup test %d", i),
		})
		require.NoError(t, err)

		coders = append(coders, coder)
		workspaceDirs = append(workspaceDirs, coder.WorkspaceDir)

		// Create some files in the workspace
		err = coder.WriteFile("test.txt", []byte("test content"))
		require.NoError(t, err)
	}

	// Verify all workspaces exist
	for _, dir := range workspaceDirs {
		assert.DirExists(t, dir)
	}

	// Delete coders
	for _, coder := range coders {
		err := manager.DeleteCoder(coder.ID)
		require.NoError(t, err)
	}

	// Verify all workspaces are cleaned up
	for _, dir := range workspaceDirs {
		assert.NoDirExists(t, dir)
	}
}

// Helper functions
func setupIntegrationTest(t *testing.T) (*AICoderManager, *MockEventBus, string) {
	tmpDir := t.TempDir()

	eventBus := &MockEventBus{}
	config := &TestConfig{
		WorkspaceBaseDir: tmpDir,
		MaxConcurrent:    5,
		DefaultProvider:  "mock",
		TimeoutMinutes:   10,
	}

	manager, err := NewAICoderManagerWithoutMockProvider(config, eventBus)
	require.NoError(t, err)

	// Register working mock provider
	mockProvider := &MockAIProvider{}
	mockProvider.On("Name").Return("mock")
	mockProvider.On("GetCapabilities").Return(ProviderCapabilities{
		SupportsStreaming: true,
		MaxContextTokens:  100000,
		MaxOutputTokens:   4096,
		SupportedModels:   []string{"mock-model"},
	})
	mockProvider.On("GenerateCode", mock.Anything, mock.Anything, mock.Anything).
		Return(&GenerateResult{
			Code:    "// Generated code",
			Summary: "Code generated successfully",
		}, nil)

	err = manager.RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	return manager, eventBus, tmpDir
}

func cleanupIntegrationTest(t *testing.T, tmpDir string) {
	err := os.RemoveAll(tmpDir)
	if err != nil {
		t.Logf("Warning: failed to cleanup test directory: %v", err)
	}
}
