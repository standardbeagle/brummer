#!/usr/bin/env python3
"""Test error display in component system using pty."""

import os
import pty
import sys
import time
import select
import termios
import tty
import threading

def read_output(master_fd, duration=5):
    """Read output from the pty for a given duration."""
    output = []
    start_time = time.time()
    
    while time.time() - start_time < duration:
        # Check if there's data to read
        if master_fd in select.select([master_fd], [], [], 0.1)[0]:
            try:
                data = os.read(master_fd, 1024).decode('utf-8', errors='replace')
                output.append(data)
                # Print to see what's happening
                sys.stdout.write(data)
                sys.stdout.flush()
            except OSError:
                break
    
    return ''.join(output)

def test_component_errors():
    """Test error display with component system."""
    print("Testing error display with component system enabled...")
    print("-" * 60)
    
    # Set environment for components
    env = os.environ.copy()
    env['BRUMMER_USE_COMPONENTS'] = 'true'
    
    # Copy test package.json
    os.system('cp test-errors-package.json package.json')
    
    # Create a pty
    master_fd, slave_fd = pty.openpty()
    
    # Start brummer with errors script
    pid = os.fork()
    if pid == 0:
        # Child process
        os.close(master_fd)
        
        # Set slave as stdin/stdout/stderr
        os.dup2(slave_fd, 0)
        os.dup2(slave_fd, 1)
        os.dup2(slave_fd, 2)
        
        # Make it a controlling terminal
        os.setsid()
        os.dup2(slave_fd, 0)
        
        # Close the original slave_fd
        if slave_fd > 2:
            os.close(slave_fd)
        
        # Execute brummer with errors script
        os.execve('./brum', ['./brum', 'errors'], env)
        sys.exit(1)  # Should not reach here
    
    # Parent process
    os.close(slave_fd)
    
    # Give it time to start
    time.sleep(2)
    
    print("\n\n=== Initial view (Processes) ===")
    read_output(master_fd, 2)
    
    # Switch to Tab 3 (Errors)
    print("\n\n=== Switching to Tab 3 (Errors) ===")
    os.write(master_fd, b'3')
    time.sleep(1)
    
    # Read the error view
    output = read_output(master_fd, 3)
    
    # Check if we see error indicators
    if "❌" in output or "Error" in output or "error" in output:
        print("\n\n✅ SUCCESS: Errors are being displayed in the error tab!")
    else:
        print("\n\n⚠️  WARNING: No errors visible in the error tab")
        print("Output received:")
        print(output)
    
    # Send quit command
    print("\n\n=== Quitting ===")
    os.write(master_fd, b'q')
    time.sleep(0.5)
    
    # Clean up
    os.close(master_fd)
    os.waitpid(pid, 0)
    
    print("\n" + "=" * 60)
    print("Test complete!")

if __name__ == "__main__":
    test_component_errors()