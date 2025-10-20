package fixtures

import (
	"fmt"
	"time"
)

// Test error fixtures for error grouping functionality

// StackTraceError returns a JavaScript stack trace error
func StackTraceError() string {
	return `Error: Cannot find module './config'
    at Function.Module._resolveFilename (node:internal/modules/cjs/loader:1075:15)
    at Function.Module._load (node:internal/modules/cjs/loader:920:27)
    at Module.require (node:internal/modules/cjs/helpers:123:19)
    at require (node:internal/modules/cjs/helpers:179:18)
    at Object.<anonymous> (/app/index.js:5:15)
    at Module._compile (node:internal/modules/cjs/loader:1101:14)
    at Object.Module._extensions..js (node:internal/modules/cjs/loader:1153:10)
    at Module.load (node:internal/modules/cjs/loader:981:32)
    at Function.Module._load (node:internal/modules/cjs/loader:822:12)
    at Function.executeUserEntryPoint [as runMain] (node:internal/modules/run_main:81:12)`
}

// CompilationError returns a TypeScript compilation error
func CompilationError() string {
	return `src/app.ts:15:7 - error TS2304: Cannot find name 'foo'.

15   const foo = bar();
         ~~~

Found 1 error.

error Command failed with exit code 1.`
}

// TestFailureError returns a Jest test failure
func TestFailureError() string {
	return `FAIL src/app.test.ts
  App › should render
    expect(received).toBe(expected)

    Expected: "Hello"
    Received: "World"

      at Object.<anonymous> (src/app.test.ts:10:20)

Test Suites: 1 failed, 1 total
Tests:       1 failed, 1 total`
}

// GenericMultiLineError returns a generic multi-line error
func GenericMultiLineError() string {
	return `Something went wrong processing the request
  Details: Invalid input parameters
  Status: 400 Bad Request
  Timestamp: ` + time.Now().Format(time.RFC3339) + `
  Please check your input and try again`
}

// NestedError returns an error with nested error information
func NestedError() string {
	return `Error: Failed to connect to database
Caused by: Error: Connection timeout
  at Database.connect (/app/db.js:45:10)
  at App.start (/app/index.js:23:5)
Original error: Error: ECONNREFUSED 127.0.0.1:5432
  at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1187:16)`
}

// VeryLongStackTrace returns a stack trace with 100+ lines for stress testing
func VeryLongStackTrace() string {
	var error string = "Error: Very deep call stack\n"
	for i := 1; i <= 100; i++ {
		error += fmt.Sprintf("    at level%d (file%d.js:%d:15)\n", i, i, i*10)
	}
	return error
}

// SpecialCharactersError returns an error with unicode and special characters
func SpecialCharactersError() string {
	return `Error: Invalid character encoding detected 🚨
  Input: café résumé naïve façade
  Unicode: U+00E9 U+00E9 U+00EF U+00E9
  Status: ❌ Failed to process
  Fix: Ensure UTF-8 encoding ✓`
}

// ANSIError returns an error with ANSI color codes
func ANSIError() string {
	return "\033[31mError:\033[0m \033[1mSomething went wrong\033[0m\n" +
		"  \033[33mWarning:\033[0m Check your configuration\n" +
		"  \033[32mInfo:\033[0m See documentation for details"
}

// EmptyError returns an empty error line
func EmptyError() string {
	return ""
}

// SingleLineError returns a single line error (should not be grouped)
func SingleLineError() string {
	return "Error: Simple one-line error message"
}

// MultipleErrors returns multiple separate errors for batch testing
func MultipleErrors() []string {
	return []string{
		StackTraceError(),
		CompilationError(),
		TestFailureError(),
		GenericMultiLineError(),
		SingleLineError(),
	}
}

// ErrorWithTimestamp returns an error with timestamp in different formats
func ErrorWithTimestamp() string {
	return fmt.Sprintf(`[%s] ERROR: Something failed
  Process: %d
  Module: test-module
  Details: Check logs for more information`,
		time.Now().Format("2006-01-02 15:04:05"),
		12345,
	)
}