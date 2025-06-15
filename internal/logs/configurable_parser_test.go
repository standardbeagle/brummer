package logs

import (
	"testing"
	"time"
)

func TestConfigurableErrorParser(t *testing.T) {
	parser, err := NewDefaultConfigurableErrorParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Test JavaScript error detection
	t.Run("JavaScript Error", func(t *testing.T) {
		content := "TypeError: Cannot read property 'length' of undefined"
		result := parser.ProcessLine("test-1", "dev", content, time.Now())

		if result == nil {
			t.Fatal("Expected error to be detected")
		}

		if result.Type != "JavaScriptError" {
			t.Errorf("Expected type JavaScriptError, got %s", result.Type)
		}

		if result.Language != "javascript" {
			t.Errorf("Expected language javascript, got %s", result.Language)
		}
	})

	// Test TypeScript error detection
	t.Run("TypeScript Error", func(t *testing.T) {
		content := "TS2345: Argument of type 'string' is not assignable to parameter of type 'number'"
		result := parser.ProcessLine("test-2", "dev", content, time.Now())

		if result == nil {
			t.Fatal("Expected error to be detected")
		}

		if result.Type != "TypeScriptError" {
			t.Errorf("Expected type TypeScriptError, got %s", result.Type)
		}
	})

	// Test multi-line error handling
	t.Run("Multi-line Error", func(t *testing.T) {
		// Start error
		line1 := "UnhandledPromiseRejectionWarning: Error: Connection failed"
		result1 := parser.ProcessLine("test-3", "dev", line1, time.Now())
		if result1 != nil {
			t.Error("Expected first line to not complete error yet")
		}

		// Stack trace line
		line2 := "    at Database.connect (/app/database.js:42:15)"
		result2 := parser.ProcessLine("test-3", "dev", line2, time.Now())
		if result2 != nil {
			t.Error("Expected stack trace line to not complete error yet")
		}

		// Context line
		line3 := "    at async main (/app/index.js:10:5)"
		result3 := parser.ProcessLine("test-3", "dev", line3, time.Now())
		if result3 != nil {
			t.Error("Expected context line to not complete error yet")
		}

		// End with non-error line
		line4 := "Server starting on port 3000"
		result4 := parser.ProcessLine("test-3", "dev", line4, time.Now())

		if result4 == nil {
			t.Fatal("Expected error to be completed")
		}

		if len(result4.Stack) == 0 {
			t.Error("Expected stack trace to be captured")
		}

		if len(result4.Raw) != 3 {
			t.Errorf("Expected 3 raw lines, got %d", len(result4.Raw))
		}
	})

	// Test React error detection
	t.Run("React Error", func(t *testing.T) {
		content := "Warning: Each child in a list should have a unique \"key\" prop"
		result := parser.ProcessLine("test-4", "dev", content, time.Now())

		if result == nil {
			t.Fatal("Expected error to be detected")
		}

		if result.Type != "ReactError" {
			t.Errorf("Expected type ReactError, got %s", result.Type)
		}

		if result.Severity != "warning" {
			t.Errorf("Expected severity warning, got %s", result.Severity)
		}
	})

	// Test log prefix stripping
	t.Run("Log Prefix Stripping", func(t *testing.T) {
		content := "[12:34:56] [dev]: TypeError: undefined is not a function"
		result := parser.ProcessLine("test-5", "dev", content, time.Now())

		if result == nil {
			t.Fatal("Expected error to be detected")
		}

		// Message should not contain timestamp or process prefix
		if result.Message == content {
			t.Error("Expected log prefixes to be stripped from message")
		}
	})

	// Test language detection
	t.Run("Language Detection", func(t *testing.T) {
		tests := []struct {
			content  string
			expected string
		}{
			{"Error at /app/server.js:42:15", "javascript"},
			{"panic: runtime error: index out of range", "go"},
			{"Traceback (most recent call last):", "python"},
			{"Exception in thread \"main\" java.lang.NullPointerException", "java"},
			{"error[E0277]: the trait bound", "rust"},
		}

		for _, test := range tests {
			result := parser.ProcessLine("test-lang", "dev", test.content, time.Now())
			if result != nil && result.Language != test.expected {
				t.Errorf("Content '%s': expected language %s, got %s", 
					test.content, test.expected, result.Language)
			}
		}
	})
}

func TestConfigurableErrorParserConfiguration(t *testing.T) {
	parser, err := NewDefaultConfigurableErrorParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config := parser.GetConfig()

	// Test that configuration was loaded
	if config.Settings.MaxContextLines == 0 {
		t.Error("Expected MaxContextLines to be set")
	}

	if len(config.ErrorPatterns) == 0 {
		t.Error("Expected error patterns to be loaded")
	}

	if len(config.LanguageDetection) == 0 {
		t.Error("Expected language detection config to be loaded")
	}

	// Test specific configurations
	if jsConfig, exists := config.LanguageDetection["javascript"]; exists {
		if len(jsConfig.FileExtensions) == 0 {
			t.Error("Expected JavaScript file extensions to be configured")
		}
	} else {
		t.Error("Expected JavaScript language detection to be configured")
	}
}

func TestCustomErrorProcessing(t *testing.T) {
	parser, err := NewDefaultConfigurableErrorParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Test MongoDB error with hostname extraction
	t.Run("MongoDB Error", func(t *testing.T) {
		content := "MongoError: getaddrinfo ENOTFOUND test-host"
		line2 := "  hostname: 'test-host'"
		
		result1 := parser.ProcessLine("test-mongo", "dev", content, time.Now())
		if result1 != nil {
			t.Error("Expected MongoDB error to be multi-line")
		}
		
		parser.ProcessLine("test-mongo", "dev", line2, time.Now())
		
		// End with non-error line
		line3 := "Attempting reconnection..."
		result3 := parser.ProcessLine("test-mongo", "dev", line3, time.Now())
		
		if result3 == nil {
			t.Fatal("Expected MongoDB error to be completed")
		}
		
		if result3.Type != "MongoError" {
			t.Errorf("Expected type MongoError, got %s", result3.Type)
		}
	})

	// Test network error detection
	t.Run("Network Error", func(t *testing.T) {
		content := "ECONNREFUSED: Connection refused to localhost:5432"
		result := parser.ProcessLine("test-net", "dev", content, time.Now())

		if result == nil {
			t.Fatal("Expected network error to be detected")
		}

		if result.Type != "NetworkError" {
			t.Errorf("Expected type NetworkError, got %s", result.Type)
		}
	})
}

func TestErrorLimits(t *testing.T) {
	parser, err := NewDefaultConfigurableErrorParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Generate many errors to test memory limits
	maxErrors := parser.config.Limits.MaxErrorsInMemory
	
	// Create more errors than the limit
	for i := 0; i < maxErrors+10; i++ {
		content := "Error: Test error " + string(rune(i))
		parser.ProcessLine("test-limit", "dev", content, time.Now())
	}

	errors := parser.GetErrors()
	
	if len(errors) > maxErrors {
		t.Errorf("Expected at most %d errors in memory, got %d", maxErrors, len(errors))
	}
}

func BenchmarkConfigurableErrorParser(b *testing.B) {
	parser, err := NewDefaultConfigurableErrorParser()
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}

	testContent := "TypeError: Cannot read property 'length' of undefined"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.ProcessLine("bench", "dev", testContent, time.Now())
	}
}