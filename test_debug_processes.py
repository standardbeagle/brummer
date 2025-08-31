#!/usr/bin/env python3
"""Debug process loading to see what's happening"""

import pty
import os
import sys
import time
import select
import tempfile
import json

def test_process_debug():
    """Debug process loading"""
    
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
        env['BRUMMER_DEBUG'] = 'true'  # Enable debug output
        
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
                                sys.stdout.write(decoded)
                                sys.stdout.flush()
                        except OSError:
                            break
            
            print("Starting with debug output...\n")
            read_output(0.5)
            
            print("\n\nWaiting for tick...\n")
            read_output(2.5)
            
            print("\n\nLet's see if we can start a script (pressing Enter)...\n")
            os.write(master, b'\r')
            read_output(1.0)
            
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
    test_process_debug()