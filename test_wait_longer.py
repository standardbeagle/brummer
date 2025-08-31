#!/usr/bin/env python3
"""Test component system with longer wait to see if scripts appear after tick"""

import pty
import os
import sys
import time
import select
import tempfile
import json

def test_with_wait():
    """Test that scripts appear after the first tick"""
    
    print("="*60)
    print("TEST: Component System with Wait")
    print("="*60)
    
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
            
            print("\n1. Starting TUI...")
            read_output(1.0)
            
            print("\n2. Waiting 2 seconds for first tick...")
            read_output(2.0)
            
            print("\n3. Checking if scripts are now visible...")
            
            # Check output
            full_output = ''.join(output)
            
            if "hello" in full_output:
                print("✅ SUCCESS: Scripts are displayed after tick!")
                
                # Now try to start a script
                print("\n4. Pressing Enter to start first script...")
                os.write(master, b'\r')
                read_output(2.0)
                
                print("\n5. Switching to logs view...")
                os.write(master, b'2')
                read_output(2.0)
                
                full_output = ''.join(output)
                if "Hello World" in full_output:
                    print("✅ SUCCESS: Script started and logs displayed!")
                else:
                    print("❌ Script may not have started properly")
            else:
                print("❌ FAILURE: Scripts not appearing even after tick")
                print("Looking for 'Stopped Processes' header...")
                if "Stopped Processes" in full_output or "⏹️" in full_output:
                    print("Header found but scripts not listed")
                else:
                    print("No stopped processes section at all")
            
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
    test_with_wait()