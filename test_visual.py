#!/usr/bin/env python3
"""Visual test to see what's actually displayed"""

import pty
import os
import sys
import time
import select
import tempfile
import json
import re

def test_visual():
    """Check what's displayed after waiting"""
    
    # Create temp dir with package.json
    with tempfile.TemporaryDirectory() as tmpdir:
        package_json = {
            "name": "test-project",
            "scripts": {
                "hello": "echo 'Hello World!'",
                "dev": "echo 'Dev server'",
                "test": "echo 'Testing...'"
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
            
            print("Starting TUI...")
            read_output(0.5)
            
            print("\nWaiting 2 seconds for tick...")
            read_output(2.0)
            
            print("\nFinal state:\n" + "="*60)
            read_output(0.5)
            
            # Clean ANSI codes for analysis
            full_output = ''.join(output)
            ansi_escape = re.compile(r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')
            clean_output = ansi_escape.sub('', full_output)
            
            # Look for key items
            print("\n" + "="*60)
            print("Analysis:")
            if "hello" in clean_output.lower():
                print("✅ Found 'hello'")
            if "dev" in clean_output.lower():
                print("✅ Found 'dev'")
            if "test" in clean_output.lower():
                print("✅ Found 'test'")
            if "Stopped Processes" in clean_output:
                print("✅ Found 'Stopped Processes' header")
            if "No processes running" in clean_output:
                print("❌ Shows 'No processes running' (empty state)")
            
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
    test_visual()