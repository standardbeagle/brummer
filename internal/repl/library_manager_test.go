package repl

import (
	"strings"
	"testing"
	"time"
)

func TestLibraryManager(t *testing.T) {
	// Set up test environment
	tempDir := t.TempDir()
	oldGetScriptsDir := getScriptsDirectory
	getScriptsDirectory = func() (string, error) {
		return tempDir, nil
	}
	defer func() { getScriptsDirectory = oldGetScriptsDir }()

	// Create library manager
	lm := NewLibraryManager()

	// Test that cache valid seconds is set correctly
	if lm.cacheValidSecs != DefaultCacheValidSecs {
		t.Errorf("cacheValidSecs = %d, want %d", lm.cacheValidSecs, DefaultCacheValidSecs)
	}

	t.Run("LoadScripts empty directory", func(t *testing.T) {
		scripts, err := lm.LoadScripts()
		if err != nil {
			t.Fatalf("LoadScripts() error = %v", err)
		}
		if len(scripts) != 0 {
			t.Errorf("expected 0 scripts, got %d", len(scripts))
		}
	})

	t.Run("AddScript", func(t *testing.T) {
		metadata := ScriptMetadata{
			Description: "Test script for library manager",
			Category:    "test",
			Tags:        []string{"test", "library"},
			Author:      "Test",
			Version:     "1.0.0",
		}

		err := lm.AddScript("libtest", "function libtest() { return true; }", metadata)
		if err != nil {
			t.Fatalf("AddScript() error = %v", err)
		}

		// Verify script was added
		scripts, err := lm.LoadScripts()
		if err != nil {
			t.Fatalf("LoadScripts() error = %v", err)
		}
		if len(scripts) != 1 {
			t.Errorf("expected 1 script, got %d", len(scripts))
		}
	})

	t.Run("GetScript", func(t *testing.T) {
		script, err := lm.GetScript("libtest")
		if err != nil {
			t.Fatalf("GetScript() error = %v", err)
		}
		if script.Name != "libtest" {
			t.Errorf("got script name %q, want %q", script.Name, "libtest")
		}
		if script.Metadata.Description != "Test script for library manager" {
			t.Errorf("got wrong description: %q", script.Metadata.Description)
		}
	})

	t.Run("SearchScripts", func(t *testing.T) {
		// Add another script
		metadata := ScriptMetadata{
			Description: "Utility helper function",
			Category:    "util",
			Tags:        []string{"helper", "utility"},
		}
		lm.AddScript("helper", "function helper() {}", metadata)

		// Search by description
		results, err := lm.SearchScripts("library")
		if err != nil {
			t.Fatalf("SearchScripts() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 result for 'library', got %d", len(results))
		}

		// Search by tag
		results, err = lm.SearchScripts("utility")
		if err != nil {
			t.Fatalf("SearchScripts() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 result for 'utility', got %d", len(results))
		}

		// Search by category
		results, err = lm.SearchScripts("test")
		if err != nil {
			t.Fatalf("SearchScripts() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 result for 'test', got %d", len(results))
		}
	})

	t.Run("GetCategories", func(t *testing.T) {
		categories, err := lm.GetCategories()
		if err != nil {
			t.Fatalf("GetCategories() error = %v", err)
		}

		expectedCategories := map[string]bool{"test": true, "util": true}
		if len(categories) != len(expectedCategories) {
			t.Errorf("expected %d categories, got %d", len(expectedCategories), len(categories))
		}

		for _, cat := range categories {
			if !expectedCategories[cat] {
				t.Errorf("unexpected category: %q", cat)
			}
		}
	})

	t.Run("UpdateScript", func(t *testing.T) {
		updatedMetadata := ScriptMetadata{
			Description: "Updated description",
			Category:    "test",
			Version:     "2.0.0",
		}

		err := lm.UpdateScript("libtest", "function libtest() { return false; }", updatedMetadata)
		if err != nil {
			t.Fatalf("UpdateScript() error = %v", err)
		}

		// Verify update
		script, err := lm.GetScript("libtest")
		if err != nil {
			t.Fatalf("GetScript() error = %v", err)
		}
		if script.Metadata.Description != "Updated description" {
			t.Errorf("description not updated: %q", script.Metadata.Description)
		}
		if script.Metadata.Version != "2.0.0" {
			t.Errorf("version not updated: %q", script.Metadata.Version)
		}
		if !strings.Contains(script.Code, "return false") {
			t.Errorf("code not updated")
		}
	})

	t.Run("RemoveScript", func(t *testing.T) {
		err := lm.RemoveScript("helper")
		if err != nil {
			t.Fatalf("RemoveScript() error = %v", err)
		}

		// Verify removal
		_, err = lm.GetScript("helper")
		if err == nil {
			t.Errorf("GetScript() should error after removal")
		}

		scripts, _ := lm.LoadScripts()
		if len(scripts) != 1 {
			t.Errorf("expected 1 script after removal, got %d", len(scripts))
		}
	})

	t.Run("Cache behavior", func(t *testing.T) {
		// Load scripts (should hit cache)
		start := time.Now()
		scripts1, _ := lm.LoadScripts()
		elapsed1 := time.Since(start)

		// Second load should be faster due to cache
		start = time.Now()
		scripts2, _ := lm.LoadScripts()
		elapsed2 := time.Since(start)

		if len(scripts1) != len(scripts2) {
			t.Errorf("cached results differ in length")
		}

		// Cache should be significantly faster (at least 10x)
		// This is a rough heuristic, may need adjustment
		if elapsed2 > elapsed1/10 {
			t.Logf("Warning: cache might not be working efficiently. First load: %v, Second load: %v", elapsed1, elapsed2)
		}
	})

	t.Run("Error cases", func(t *testing.T) {
		// Try to add script with existing name
		err := lm.AddScript("libtest", "code", ScriptMetadata{Description: "test"})
		if err == nil {
			t.Errorf("AddScript() should error for duplicate name")
		}

		// Try to update non-existent script
		err = lm.UpdateScript("nonexistent", "code", ScriptMetadata{Description: "test"})
		if err == nil {
			t.Errorf("UpdateScript() should error for non-existent script")
		}

		// Try to remove non-existent script
		err = lm.RemoveScript("nonexistent")
		if err == nil {
			t.Errorf("RemoveScript() should error for non-existent script")
		}

		// Try invalid operations
		err = lm.AddScript("", "code", ScriptMetadata{Description: "test"})
		if err == nil {
			t.Errorf("AddScript() should error for empty name")
		}

		err = lm.AddScript("test", "", ScriptMetadata{Description: "test"})
		if err == nil {
			t.Errorf("AddScript() should error for empty code")
		}

		err = lm.AddScript("test", "code", ScriptMetadata{})
		if err == nil {
			t.Errorf("AddScript() should error for empty description")
		}
	})

	t.Run("GetLibraryInfo", func(t *testing.T) {
		info, err := lm.GetLibraryInfo()
		if err != nil {
			t.Fatalf("GetLibraryInfo() error = %v", err)
		}

		if info.Count != len(info.Scripts) {
			t.Errorf("info.Count = %d, but len(Scripts) = %d", info.Count, len(info.Scripts))
		}

		if info.LoadedAt.IsZero() {
			t.Errorf("LoadedAt should not be zero")
		}
	})

	t.Run("GenerateLibraryInjectionCode", func(t *testing.T) {
		code, err := lm.GenerateLibraryInjectionCode()
		if err != nil {
			t.Fatalf("GenerateLibraryInjectionCode() error = %v", err)
		}

		// Should contain the library setup
		if !strings.Contains(code, "window.brummerLibrary") {
			t.Errorf("generated code missing window.brummerLibrary")
		}

		// Should contain the remaining script
		if !strings.Contains(code, "libtest") {
			t.Errorf("generated code missing libtest function")
		}
	})
}

func TestValidateScript(t *testing.T) {
	lm := NewLibraryManager()

	tests := []struct {
		name       string
		scriptName string
		code       string
		metadata   ScriptMetadata
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid script",
			scriptName: "valid",
			code:       "function valid() { return true; }",
			metadata:   ScriptMetadata{Description: "Valid script"},
			wantErr:    false,
		},
		{
			name:       "empty name",
			scriptName: "",
			code:       "function() {}",
			metadata:   ScriptMetadata{Description: "test"},
			wantErr:    true,
			errMsg:     "name is required",
		},
		{
			name:       "empty code",
			scriptName: "test",
			code:       "",
			metadata:   ScriptMetadata{Description: "test"},
			wantErr:    true,
			errMsg:     "code is required",
		},
		{
			name:       "empty description",
			scriptName: "test",
			code:       "function() {}",
			metadata:   ScriptMetadata{},
			wantErr:    true,
			errMsg:     "description is required",
		},
		{
			name:       "unbalanced braces",
			scriptName: "test",
			code:       "function test() { if (true) { return 1; }",
			metadata:   ScriptMetadata{Description: "test"},
			wantErr:    true,
			errMsg:     "unbalanced braces",
		},
		{
			name:       "unbalanced parentheses",
			scriptName: "test",
			code:       "function test( { return 1; }",
			metadata:   ScriptMetadata{Description: "test"},
			wantErr:    true,
			errMsg:     "unbalanced parentheses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := lm.ValidateScript(tt.scriptName, tt.code, tt.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScript() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateScript() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}
