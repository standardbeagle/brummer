#!/usr/bin/env python3
"""Test without PTY to see error"""

import subprocess
import tempfile
import json
import os
import time

def test_no_pty():
    """Test without PTY to see error"""
    
    # Create temp dir with package.json
    with tempfile.TemporaryDirectory() as tmpdir:
        package_json = {
            "name": "test-project",
            "scripts": {
                "hello": "echo 'Hello World!'"
            }
        }
        
        with open(os.path.join(tmpdir, "package.json"), "w") as f:
            json.dump(package_json, f, indent=2)
        
        # Set environment for components
        env = os.environ.copy()
        env['BRUMMER_USE_COMPONENTS'] = 'true'
        
        # Run directly without PTY
        print("Running brummer without PTY...")
        proc = subprocess.Popen(
            ['/home/beagle/work/brummer/brum'],
            cwd=tmpdir,
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        # Wait a bit
        time.sleep(2)
        
        # Try to terminate
        proc.terminate()
        
        # Get output
        stdout, stderr = proc.communicate(timeout=2)
        
        print("STDOUT:")
        print(stdout)
        print("\nSTDERR:")
        print(stderr)
        
        # Check debug log
        if os.path.exists('/tmp/brummer_debug.log'):
            print("\nDEBUG LOG:")
            with open('/tmp/brummer_debug.log', 'r') as f:
                print(f.read())

if __name__ == '__main__':
    test_no_pty()