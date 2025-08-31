#!/usr/bin/env python3
"""Test TUI startup with stderr output"""

import subprocess
import tempfile
import json
import os
import time
import sys

def test_tui_startup():
    """Test TUI startup with stderr capture"""
    
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
        
        print(f"Created test dir: {tmpdir}")
        
        # Run with components and capture stderr 
        env = os.environ.copy()
        env['BRUMMER_USE_COMPONENTS'] = 'true'
        
        # Run in a subprocess with timeout
        print("\nStarting brummer with component system...")
        proc = subprocess.Popen(
            ['/home/beagle/work/brummer/brum'],
            cwd=tmpdir,
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            stdin=subprocess.PIPE
        )
        
        # Wait a bit for startup
        time.sleep(2)
        
        # Send 'q' to quit
        proc.stdin.write(b'q')
        proc.stdin.flush()
        
        # Wait for exit
        try:
            stdout, stderr = proc.communicate(timeout=2)
        except subprocess.TimeoutExpired:
            proc.terminate()
            stdout, stderr = proc.communicate()
        
        print("\n=== STDOUT ===")
        if stdout:
            print(stdout.decode('utf-8', errors='replace'))
        else:
            print("(empty)")
        
        print("\n=== STDERR ===")
        if stderr:
            print(stderr.decode('utf-8', errors='replace'))
        else:
            print("(empty)")
        
        print("\n=== EXIT CODE ===")
        print(proc.returncode)

if __name__ == '__main__':
    test_tui_startup()