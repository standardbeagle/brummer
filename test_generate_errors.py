#!/usr/bin/env python3
"""Test script to generate various types of errors for TUI testing."""

import sys
import time
import random
import json

def generate_javascript_error():
    """Generate a JavaScript-style error."""
    print("Error: Cannot read property 'undefined' of null", file=sys.stderr)
    print("    at Object.<anonymous> (/app/src/index.js:42:15)", file=sys.stderr)
    print("    at Module._compile (internal/modules/cjs/loader.js:1063:30)", file=sys.stderr)
    time.sleep(0.1)

def generate_python_traceback():
    """Generate a Python-style traceback."""
    print("Traceback (most recent call last):", file=sys.stderr)
    print('  File "app.py", line 156, in <module>', file=sys.stderr)
    print('    result = process_data(input_data)', file=sys.stderr)
    print('  File "app.py", line 89, in process_data', file=sys.stderr)
    print('    return data["key"] / 0', file=sys.stderr)
    print("ZeroDivisionError: division by zero", file=sys.stderr)
    time.sleep(0.1)

def generate_compilation_error():
    """Generate a compilation error."""
    print("main.go:45:2: undefined: someFunction", file=sys.stderr)
    print("main.go:67:15: cannot use x (type int) as type string in argument to fmt.Printf", file=sys.stderr)
    print("main.go:89:1: syntax error: unexpected }", file=sys.stderr)
    time.sleep(0.1)

def generate_network_error():
    """Generate network-related errors."""
    print("Error: connect ECONNREFUSED 127.0.0.1:3000", file=sys.stderr)
    print("Error: getaddrinfo ENOTFOUND api.example.com", file=sys.stderr)
    print("Error: Request timeout after 30000ms", file=sys.stderr)
    time.sleep(0.1)

def generate_permission_error():
    """Generate permission errors."""
    print("Error: EACCES: permission denied, open '/etc/passwd'", file=sys.stderr)
    print("Error: EPERM: operation not permitted, unlink '/usr/bin/node'", file=sys.stderr)
    time.sleep(0.1)

def generate_memory_error():
    """Generate memory-related errors."""
    print("FATAL ERROR: Ineffective mark-compacts near heap limit Allocation failed - JavaScript heap out of memory", file=sys.stderr)
    print("<--- Last few GCs --->", file=sys.stderr)
    print("[1234:0x7f8b9c0] 15234 ms: Mark-sweep 2045.3 (2051.5) -> 2044.2 (2051.5) MB", file=sys.stderr)
    time.sleep(0.1)

def generate_database_error():
    """Generate database errors."""
    print("Error: ER_ACCESS_DENIED_ERROR: Access denied for user 'root'@'localhost'", file=sys.stderr)
    print("Error: SQLITE_CONSTRAINT: UNIQUE constraint failed: users.email", file=sys.stderr)
    print("Error: Connection lost: The server closed the connection.", file=sys.stderr)
    time.sleep(0.1)

def generate_validation_error():
    """Generate validation errors."""
    print("ValidationError: User validation failed:", file=sys.stderr)
    print("  - email: Please provide a valid email address", file=sys.stderr)
    print("  - password: Password must be at least 8 characters", file=sys.stderr)
    print("  - age: Must be a positive number", file=sys.stderr)
    time.sleep(0.1)

def generate_json_error():
    """Generate JSON parsing errors."""
    print("SyntaxError: Unexpected token } in JSON at position 123", file=sys.stderr)
    print("SyntaxError: Unexpected end of JSON input", file=sys.stderr)
    print('SyntaxError: Unexpected string in JSON at position 45', file=sys.stderr)
    time.sleep(0.1)

def generate_assertion_error():
    """Generate assertion failures."""
    print("AssertionError: expected 200 to equal 404", file=sys.stderr)
    print("    at Context.<anonymous> (test/api.spec.js:25:18)", file=sys.stderr)
    print("AssertionError: expected true to be false", file=sys.stderr)
    time.sleep(0.1)

def generate_mixed_output():
    """Generate mixed stdout and stderr output."""
    print("INFO: Starting application...")
    print("ERROR: Failed to load configuration", file=sys.stderr)
    print("INFO: Attempting fallback configuration")
    print("WARNING: Using default settings", file=sys.stderr)
    print("ERROR: Critical component failed to initialize", file=sys.stderr)
    time.sleep(0.1)

def generate_continuous_errors():
    """Generate continuous stream of errors."""
    error_types = [
        "Error: Connection refused",
        "Error: Timeout exceeded",
        "Error: Invalid input",
        "Error: Resource not found",
        "Error: Access denied",
        "Error: Operation failed",
    ]
    
    for i in range(10):
        error = random.choice(error_types)
        print(f"[{time.strftime('%H:%M:%S')}] {error} (attempt {i+1}/10)", file=sys.stderr)
        time.sleep(0.2)

def generate_panic_error():
    """Generate a Go-style panic."""
    print("panic: runtime error: index out of range [5] with length 3", file=sys.stderr)
    print("", file=sys.stderr)
    print("goroutine 1 [running]:", file=sys.stderr)
    print("main.processItems(0xc0000b4000, 0x3, 0x3)", file=sys.stderr)
    print("        /app/main.go:45 +0x1a5", file=sys.stderr)
    print("main.main()", file=sys.stderr)
    print("        /app/main.go:20 +0x85", file=sys.stderr)
    time.sleep(0.1)

def generate_stack_overflow():
    """Generate a stack overflow error."""
    print("RangeError: Maximum call stack size exceeded", file=sys.stderr)
    for i in range(5):
        print(f"    at recursiveFunction (/app/recursive.js:{10+i}:5)", file=sys.stderr)
    print("    ... 9995 more lines", file=sys.stderr)
    time.sleep(0.1)

def main():
    """Main function to generate various errors."""
    print("Starting error generation test script...")
    print("This script will generate various types of errors to test the TUI error display.")
    print("")
    
    # Run through all error types
    error_generators = [
        ("JavaScript Error", generate_javascript_error),
        ("Python Traceback", generate_python_traceback),
        ("Compilation Error", generate_compilation_error),
        ("Network Error", generate_network_error),
        ("Permission Error", generate_permission_error),
        ("Memory Error", generate_memory_error),
        ("Database Error", generate_database_error),
        ("Validation Error", generate_validation_error),
        ("JSON Error", generate_json_error),
        ("Assertion Error", generate_assertion_error),
        ("Mixed Output", generate_mixed_output),
        ("Go Panic", generate_panic_error),
        ("Stack Overflow", generate_stack_overflow),
        ("Continuous Errors", generate_continuous_errors),
    ]
    
    for name, generator in error_generators:
        print(f"\n=== Generating {name} ===")
        generator()
        time.sleep(0.5)  # Pause between error types
    
    print("\n=== Error generation complete ===")
    print("Check the Errors view (Tab 3) to see all generated errors")
    
    # Exit with error code to indicate errors were generated
    sys.exit(1)

if __name__ == "__main__":
    main()