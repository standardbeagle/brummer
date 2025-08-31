#!/usr/bin/env python3
"""Very basic test of component system"""

import subprocess
import tempfile
import json
import os
import time

def test_basic():
    """Test basic component loading"""
    
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
        print(f"Package.json content:")
        with open(os.path.join(tmpdir, "package.json")) as f:
            print(f.read())
        
        # Run with components - timeout quickly  
        env = os.environ.copy()
        env['BRUMMER_USE_COMPONENTS'] = 'true'
        
        print("\nRunning with components enabled...")
        proc = subprocess.Popen(
            ['/home/beagle/work/brummer/brum'],
            cwd=tmpdir,
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        # Wait a bit then kill
        time.sleep(3)
        proc.terminate()
        proc.wait()
        
        stdout, stderr = proc.communicate()
        
        print("STDOUT:", stdout if stdout else "(empty)")
        print("STDERR:", stderr if stderr else "(empty)")

if __name__ == '__main__':
    test_basic()