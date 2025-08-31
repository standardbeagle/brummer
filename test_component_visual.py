#!/usr/bin/env python3
"""Test to see what scripts are being displayed"""

import pty
import os
import sys
import time
import select
import tempfile
import json

def test_component_visual():
    """See what's actually displayed in the component system"""
    
    print("="*60)
    print("TEST: Component System Visual Check")
    print("="*60)
    
    # Create temp dir with package.json
    with tempfile.TemporaryDirectory() as tmpdir:
        package_json = {
            "name": "test-project",
            "scripts": {
                "hello": "echo 'Hello World from script test!'",
                "dev": "echo 'Dev server running'",
                "test": "echo 'Running tests...'"
            }
        }
        
        with open(os.path.join(tmpdir, "package.json"), "w") as f:
            json.dump(package_json, f, indent=2)
        
        # Set environment for components
        env = os.environ.copy()
        env['BRUMMER_USE_COMPONENTS'] = 'true'
        
        # Create PTY
        master, slave = pty.openpty()
        
        pid = os.fork()
        
        if pid == 0:  # Child
            os.close(master)
            os.setsid()
            os.dup2(slave, 0)
            os.dup2(slave, 1) 
            os.dup2(slave, 2)
            os.close(slave)
            
            # Change to test directory and run brummer
            os.chdir(tmpdir)
            os.execve('/home/beagle/work/brummer/brum', ['brum'], env)
            sys.exit(1)
            
        else:  # Parent
            os.close(slave)
            
            output = []
            
            def read_output(duration=1.0):
                """Read output for duration"""
                start = time.time()
                while time.time() - start < duration:
                    ready, _, _ = select.select([master], [], [], 0.1)
                    if ready:
                        try:
                            data = os.read(master, 4096)
                            if data:
                                decoded = data.decode('utf-8', errors='replace')
                                output.append(decoded)
                                sys.stdout.write(decoded)
                                sys.stdout.flush()
                        except OSError:
                            break
            
            print("\n1. Starting TUI and waiting...")
            read_output(3.0)
            
            # Clear screen sequences
            print("\n" + "="*60)
            print("SCREEN SNAPSHOT")
            print("="*60)
            
            # Extract visible content
            full_output = ''.join(output)
            
            # Look for the process items
            if "hello" in full_output:
                print("✅ Found 'hello' script in display")
            if "dev" in full_output:
                print("✅ Found 'dev' script in display")
            if "test" in full_output:
                print("✅ Found 'test' script in display")
                
            # Check if selection indicator is visible
            if "▶" in full_output:
                print("✅ Selection indicator (▶) is visible")
            else:
                print("❌ No selection indicator visible")
                
            # Look for the specific lines in the last output
            lines = full_output.split('\n')
            for i, line in enumerate(lines[-30:]):  # Check last 30 lines
                if "hello" in line or "dev" in line or "test" in line:
                    print(f"Script line {i}: {repr(line[:80])}")
            
            # Kill the process
            os.write(master, b'q')
            time.sleep(0.5)
            try:
                os.kill(pid, 15)
            except:
                pass
            os.close(master)
            os.waitpid(pid, os.WNOHANG)

if __name__ == '__main__':
    test_component_visual()