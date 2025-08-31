#!/usr/bin/env python3
"""Test error display with component system enabled."""

import subprocess
import time
import sys
import os

def test_error_display():
    """Test that errors appear in the error tab."""
    print("Testing error display with component system enabled...")
    
    # Set environment to use components
    env = os.environ.copy()
    env['BRUMMER_USE_COMPONENTS'] = 'true'
    
    # Start brummer with the error test script
    print("Starting brummer with component system enabled...")
    proc = subprocess.Popen(
        ['./brum', 'errors'],
        env=env,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    
    # Wait for startup and initial errors
    time.sleep(3)
    
    # Switch to errors view (Tab 3)
    print("Switching to errors view (Tab 3)...")
    proc.stdin.write('3')
    proc.stdin.flush()
    time.sleep(1)
    
    # Take screenshot (if needed)
    print("Errors should now be visible in Tab 3")
    
    # Let it run for a bit more
    time.sleep(5)
    
    # Exit
    print("Exiting...")
    proc.stdin.write('q')
    proc.stdin.flush()
    
    # Wait for process to complete
    proc.wait(timeout=2)
    
    print("Test complete!")

if __name__ == "__main__":
    test_error_display()