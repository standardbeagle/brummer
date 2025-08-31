#!/usr/bin/env python3
"""Test script that continuously generates errors with varying patterns."""

import sys
import time
import random
import threading
import signal

running = True

def signal_handler(sig, frame):
    """Handle Ctrl+C gracefully."""
    global running
    running = False
    print("\n\nStopping error generation...", file=sys.stderr)
    sys.exit(0)

signal.signal(signal.SIGINT, signal_handler)

def generate_random_error():
    """Generate a random error message."""
    error_templates = [
        "Error: Connection to database lost",
        "Error: Failed to parse JSON response", 
        "Error: Unauthorized access attempt",
        "Error: File not found: {file}",
        "Error: Memory allocation failed",
        "Error: Network timeout after {timeout}ms",
        "Error: Invalid API key provided",
        "Error: Rate limit exceeded",
        "Error: Disk space insufficient",
        "Error: Process terminated unexpectedly",
        "TypeError: Cannot read property '{prop}' of undefined",
        "ReferenceError: {var} is not defined",
        "SyntaxError: Unexpected token {token}",
        "RangeError: Invalid array length",
        "Error: ENOENT: no such file or directory",
        "Error: EACCES: permission denied",
        "Error: EADDRINUSE: address already in use",
        "Error: ETIMEDOUT: connection timed out",
    ]
    
    template = random.choice(error_templates)
    
    # Replace placeholders with random values
    error = template.format(
        file=f"/path/to/file{random.randint(1, 100)}.txt",
        timeout=random.randint(1000, 30000),
        prop=random.choice(["data", "value", "result", "response", "user"]),
        var=random.choice(["userData", "config", "apiKey", "connection", "result"]),
        token=random.choice(["{", "}", "[", "]", ":", ","])
    )
    
    return error

def generate_burst_errors():
    """Generate a burst of errors."""
    burst_size = random.randint(3, 8)
    for i in range(burst_size):
        error = generate_random_error()
        timestamp = time.strftime("%H:%M:%S")
        print(f"[{timestamp}] {error}", file=sys.stderr)
        time.sleep(0.05)  # Rapid succession

def generate_warning():
    """Generate a warning message."""
    warnings = [
        "Warning: Deprecated function called",
        "Warning: Low memory available",
        "Warning: High CPU usage detected",
        "Warning: Cache miss rate above threshold",
        "Warning: SSL certificate expires in 7 days",
        "Warning: Slow query detected (>1000ms)",
    ]
    warning = random.choice(warnings)
    timestamp = time.strftime("%H:%M:%S")
    print(f"[{timestamp}] {warning}", file=sys.stderr)

def generate_normal_output():
    """Generate normal output to intersperse with errors."""
    messages = [
        "Processing request...",
        "Connected to server",
        "Data synchronized",
        "Cache updated",
        "Health check passed",
        "Request completed successfully",
        "Session initialized",
    ]
    message = random.choice(messages)
    timestamp = time.strftime("%H:%M:%S")
    print(f"[{timestamp}] INFO: {message}")

def main():
    """Main function to continuously generate errors."""
    print("Starting continuous error generation...")
    print("Press Ctrl+C to stop")
    print("")
    print("This script simulates a real application with:")
    print("- Random errors at varying intervals")
    print("- Burst errors (multiple errors in quick succession)")
    print("- Mixed stdout and stderr output")
    print("- Warnings and info messages")
    print("")
    time.sleep(2)
    
    error_count = 0
    warning_count = 0
    
    while running:
        # Decide what to generate
        choice = random.random()
        
        if choice < 0.4:  # 40% chance of error
            if random.random() < 0.2:  # 20% chance of burst
                print(f"\n[BURST] Multiple errors incoming...", file=sys.stderr)
                generate_burst_errors()
                error_count += random.randint(3, 8)
            else:
                error = generate_random_error()
                timestamp = time.strftime("%H:%M:%S")
                
                # Sometimes include stack trace
                if random.random() < 0.3:
                    print(f"[{timestamp}] {error}", file=sys.stderr)
                    print("    at processRequest (server.js:142:15)", file=sys.stderr)
                    print("    at handleConnection (server.js:78:22)", file=sys.stderr)
                else:
                    print(f"[{timestamp}] {error}", file=sys.stderr)
                
                error_count += 1
                
        elif choice < 0.6:  # 20% chance of warning
            generate_warning()
            warning_count += 1
            
        else:  # 40% chance of normal output
            generate_normal_output()
        
        # Print stats occasionally
        if random.random() < 0.05:  # 5% chance
            print(f"\n=== Stats: {error_count} errors, {warning_count} warnings generated ===\n")
        
        # Variable delay between messages
        delay = random.uniform(0.5, 3.0)
        time.sleep(delay)

if __name__ == "__main__":
    main()