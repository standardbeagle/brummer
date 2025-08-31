#!/usr/bin/env python3
"""Test logs directly without TUI interaction"""

import subprocess
import time
import os

def test_logs():
    """Test if logs are being captured"""
    
    print("="*60)
    print("TESTING DIRECT LOG CAPTURE")
    print("="*60)
    
    # Set environment for components
    env = os.environ.copy()
    env['BRUMMER_USE_COMPONENTS'] = 'true'
    
    # Start process
    proc = subprocess.Popen(
        ['./brum', '--no-tui', 'echo', 'HELLO FROM TEST'],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        env=env,
        text=True
    )
    
    # Wait and collect output
    time.sleep(2)
    proc.terminate()
    output, _ = proc.communicate()
    
    print("Output from brummer:")
    print(output)
    
    if "HELLO FROM TEST" in output:
        print("\n✅ SUCCESS: Command output captured")
    elif "Started command" in output:
        print("\n✅ PARTIAL: Command started")
    else:
        print("\n❌ FAILURE: No indication of command execution")

if __name__ == '__main__':
    test_logs()
