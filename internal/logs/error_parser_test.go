package logs

import (
	"testing"
	"time"
)

func TestErrorParser_ReactTypeScriptErrors(t *testing.T) {
	parser := NewErrorParser()
	timestamp := time.Now()

	tests := []struct {
		name        string
		logLine     string
		expectError bool
		errorType   string
		message     string
		language    string
	}{
		{
			name:        "React TypeScript Error",
			logLine:     "TS2345: Argument of type 'string' is not assignable to parameter of type 'number | (() => number)'.",
			expectError: true,
			errorType:   "TS2345",
			message:     "Argument of type 'string' is not assignable to parameter of type 'number | (() => number)'.",
			language:    "javascript",
		},
		{
			name:        "React Build Error",
			logLine:     "Failed to compile.",
			expectError: true,
			errorType:   "CompilationError",
			message:     "Failed to compile.",
			language:    "javascript",
		},
		{
			name:        "React JSX Key Error",
			logLine:     "ERROR: Missing \"key\" prop for element in iterator  react/jsx-key",
			expectError: true,
			errorType:   "ReactError",
			message:     "Missing \"key\" prop for element in iterator  react/jsx-key",
			language:    "javascript",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorCtx := parser.ProcessLine("test-process", "react-build", tt.logLine, timestamp)

			if tt.expectError && errorCtx == nil {
				t.Errorf("Expected error but got none for: %s", tt.logLine)
				return
			}

			if !tt.expectError && errorCtx != nil {
				t.Errorf("Did not expect error but got one for: %s", tt.logLine)
				return
			}

			if errorCtx != nil {
				if errorCtx.Type != tt.errorType {
					t.Errorf("Expected error type %s, got %s", tt.errorType, errorCtx.Type)
				}
				if errorCtx.Language != tt.language {
					t.Errorf("Expected language %s, got %s", tt.language, errorCtx.Language)
				}
			}
		})
	}
}

func TestErrorParser_VueTypeScriptErrors(t *testing.T) {
	parser := NewErrorParser()
	timestamp := time.Now()

	tests := []struct {
		name        string
		logLine     string
		expectError bool
		errorType   string
		language    string
	}{
		{
			name:        "Vue TypeScript Property Error",
			logLine:     "error TS2339: Property 'nonExistentProperty' does not exist on type 'CreateComponentPublicInstanceWithMixins'",
			expectError: true,
			errorType:   "TS2339",
			language:    "javascript",
		},
		{
			name:        "Vue Null Reference Error",
			logLine:     "error TS18047: '__VLS_ctx.user' is possibly 'null'.",
			expectError: true,
			errorType:   "TS18047",
			language:    "javascript",
		},
		{
			name:        "Vue Build Type Check Error",
			logLine:     "ERROR: \"type-check\" exited with 2.",
			expectError: true,
			errorType:   "Error",
			language:    "javascript",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorCtx := parser.ProcessLine("vue-process", "vue-build", tt.logLine, timestamp)

			if tt.expectError && errorCtx == nil {
				t.Errorf("Expected error but got none for: %s", tt.logLine)
			}
		})
	}
}

func TestErrorParser_NextJSErrors(t *testing.T) {
	parser := NewErrorParser()
	timestamp := time.Now()

	tests := []struct {
		name        string
		logLine     string
		expectError bool
		errorType   string
	}{
		{
			name:        "Next.js ESLint Error",
			logLine:     "18:28  Error: Unexpected any. Specify a different type.  @typescript-eslint/no-explicit-any",
			expectError: true,
			errorType:   "Error",
		},
		{
			name:        "Next.js Missing Key Error",
			logLine:     "165:9  Error: Missing \"key\" prop for element in iterator  react/jsx-key",
			expectError: true,
			errorType:   "Error",
		},
		{
			name:        "Next.js Compilation Success",
			logLine:     "✓ Compiled successfully in 12.0s",
			expectError: false,
			errorType:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorCtx := parser.ProcessLine("next-process", "next-build", tt.logLine, timestamp)

			if tt.expectError && errorCtx == nil {
				t.Errorf("Expected error but got none for: %s", tt.logLine)
			}

			if !tt.expectError && errorCtx != nil {
				t.Errorf("Did not expect error but got one for: %s", tt.logLine)
			}
		})
	}
}

func TestErrorParser_ExpressTypeScriptErrors(t *testing.T) {
	parser := NewErrorParser()
	timestamp := time.Now()

	tests := []struct {
		name        string
		logLine     string
		expectError bool
		errorType   string
	}{
		{
			name:        "Express TypeScript Overload Error",
			logLine:     "src/server.ts(71,31): error TS2769: No overload matches this call.",
			expectError: true,
			errorType:   "TS2769",
		},
		{
			name:        "Express Type Assignment Error",
			logLine:     "Argument of type '(req: Request, res: Response) => express.Response<any, Record<string, any>> | undefined' is not assignable to parameter",
			expectError: true,
			errorType:   "Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorCtx := parser.ProcessLine("express-process", "express-build", tt.logLine, timestamp)

			if tt.expectError && errorCtx == nil {
				t.Errorf("Expected error but got none for: %s", tt.logLine)
			}
		})
	}
}

func TestErrorParser_JavaScriptRuntimeErrors(t *testing.T) {
	parser := NewErrorParser()
	timestamp := time.Now()

	tests := []struct {
		name        string
		logLines    []string
		expectError bool
		errorType   string
		language    string
	}{
		{
			name: "TypeError with Stack Trace",
			logLines: []string{
				"TypeError: Cannot read properties of null (reading 'someProperty')",
				"    at Object.<anonymous> (/path/to/file.js:10:5)",
				"    at Module._compile (internal/modules/cjs/loader.js:999:30)",
				"    at Object.Module._extensions..js (internal/modules/cjs/loader.js:1027:10)",
			},
			expectError: true,
			errorType:   "TypeError",
			language:    "javascript",
		},
		{
			name: "ReferenceError",
			logLines: []string{
				"ReferenceError: undefinedVariable is not defined",
				"    at /path/to/file.js:15:1",
			},
			expectError: true,
			errorType:   "ReferenceError",
			language:    "javascript",
		},
		{
			name: "Network Error",
			logLines: []string{
				"FetchError: request to https://invalid-domain.nonexistent/ failed, reason: getaddrinfo ENOTFOUND invalid-domain.nonexistent",
			},
			expectError: true,
			errorType:   "FetchError",
			language:    "javascript",
		},
		{
			name: "Promise Rejection",
			logLines: []string{
				"UnhandledPromiseRejectionWarning: Error: Promise rejection test",
				"    at /path/to/file.js:25:17",
			},
			expectError: true,
			errorType:   "UnhandledPromiseRejectionWarning",
			language:    "javascript",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errorCtx *ErrorContext

			for i, line := range tt.logLines {
				errorCtx = parser.ProcessLine("js-process", "node", line, timestamp.Add(time.Duration(i)*time.Millisecond))
			}

			if tt.expectError && errorCtx == nil {
				// Check if there are any errors stored
				errors := parser.GetErrors()
				if len(errors) == 0 {
					t.Errorf("Expected error but got none for: %v", tt.logLines)
					return
				}
				errorCtx = &errors[len(errors)-1]
			}

			if !tt.expectError && errorCtx != nil {
				t.Errorf("Did not expect error but got one for: %v", tt.logLines)
				return
			}

			if errorCtx != nil {
				if errorCtx.Type != tt.errorType {
					t.Errorf("Expected error type %s, got %s", tt.errorType, errorCtx.Type)
				}
				if errorCtx.Language != tt.language {
					t.Errorf("Expected language %s, got %s", tt.language, errorCtx.Language)
				}
				if len(errorCtx.Stack) == 0 && len(tt.logLines) > 1 {
					t.Errorf("Expected stack trace but got none")
				}
			}
		})
	}
}

func TestErrorParser_BuildAndCompilationErrors(t *testing.T) {
	parser := NewErrorParser()
	timestamp := time.Now()

	tests := []struct {
		name        string
		logLines    []string
		expectError bool
		errorType   string
	}{
		{
			name: "Webpack Build Error",
			logLines: []string{
				"ERROR in ./src/App.tsx",
				"Module not found: Error: Can't resolve 'non-existent-module' in '/path/to/src'",
			},
			expectError: true,
			errorType:   "Error",
		},
		{
			name: "ESLint Error",
			logLines: []string{
				"  1:1  error  'React' must be in scope when using JSX  react/react-in-jsx-scope",
			},
			expectError: true,
			errorType:   "Error",
		},
		{
			name: "Syntax Error",
			logLines: []string{
				"SyntaxError: Unexpected token '}' in JSON at position 15",
			},
			expectError: true,
			errorType:   "SyntaxError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errorCtx *ErrorContext

			for i, line := range tt.logLines {
				errorCtx = parser.ProcessLine("build-process", "webpack", line, timestamp.Add(time.Duration(i)*time.Millisecond))
			}

			if tt.expectError && errorCtx == nil {
				errors := parser.GetErrors()
				if len(errors) == 0 {
					t.Errorf("Expected error but got none for: %v", tt.logLines)
					return
				}
			}
		})
	}
}

func TestErrorParser_MongoDBErrors(t *testing.T) {
	parser := NewErrorParser()
	timestamp := time.Now()

	logLines := []string{
		"MongoError: getaddrinfo ENOTFOUND cluster0.mongodb.net",
		"    hostname: 'cluster0.mongodb.net'",
		"    code: 'ENOTFOUND'",
	}

	var errorCtx *ErrorContext
	for i, line := range logLines {
		errorCtx = parser.ProcessLine("mongo-process", "node", line, timestamp.Add(time.Duration(i)*time.Millisecond))
	}

	if errorCtx == nil {
		errors := parser.GetErrors()
		if len(errors) == 0 {
			t.Errorf("Expected MongoDB error but got none")
			return
		}
		errorCtx = &errors[len(errors)-1]
	}

	if errorCtx.Type != "MongoError" {
		t.Errorf("Expected MongoError type, got %s", errorCtx.Type)
	}

	if !contains(errorCtx.Message, "hostname") {
		t.Errorf("Expected hostname in message, got: %s", errorCtx.Message)
	}
}

func TestErrorParser_MultilineErrorParsing(t *testing.T) {
	parser := NewErrorParser()
	timestamp := time.Now()

	// Test complex multi-line error with context
	logLines := []string{
		"Error: Database connection failed",
		"    at connectToDatabase (/app/src/database.js:25:15)",
		"    at Server.start (/app/src/server.js:10:5)",
		"    at Object.<anonymous> (/app/src/index.js:5:1)",
		"  Config: {",
		"    host: 'localhost',",
		"    port: 5432,",
		"    database: 'myapp'",
		"  }",
		"  Retry attempts: 3",
	}

	var errorCtx *ErrorContext
	for i, line := range logLines {
		result := parser.ProcessLine("db-process", "node", line, timestamp.Add(time.Duration(i)*time.Millisecond))
		if result != nil {
			errorCtx = result
		}
	}

	if errorCtx == nil {
		errors := parser.GetErrors()
		if len(errors) == 0 {
			t.Errorf("Expected multiline error but got none")
			return
		}
		errorCtx = &errors[len(errors)-1]
	}

	if len(errorCtx.Stack) < 3 {
		t.Errorf("Expected at least 3 stack trace lines, got %d", len(errorCtx.Stack))
	}

	if len(errorCtx.Context) < 5 {
		t.Errorf("Expected at least 5 context lines, got %d", len(errorCtx.Context))
	}

	if len(errorCtx.Raw) != len(logLines) {
		t.Errorf("Expected %d raw lines, got %d", len(logLines), len(errorCtx.Raw))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsAt(s, substr))))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
