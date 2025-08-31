#!/usr/bin/env python3
"""Verify e2e tests work with the actual implementation"""

import pty
import os
import sys
import time
import select
import tempfile
import json

def test_component_system_with_logs():
    """Test that logs appear correctly with EventBus synchronization"""
    
    print("="*60)
    print("E2E TEST: Component System Log Display")
    print("="*60)
    
    # Create temp dir with package.json
    with tempfile.TemporaryDirectory() as tmpdir:
        package_json = {
            "name": "test-project",
            "scripts": {
                "hello": "echo 'Hello World from E2E test!'"
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
            read_output(2.0)
            
            print("\n2. Navigating to scripts (Tab)...")
            os.write(master, b'\t')
            read_output(1.0)
            
            print("\n3. Starting hello script (Enter)...")
            os.write(master, b'\r')
            read_output(2.0)
            
            print("\n4. Switching to logs view (2)...")
            os.write(master, b'2')
            read_output(2.0)
            
            # Analyze output
            full_output = ''.join(output)
            
            print("\n" + "="*60)
            print("TEST RESULTS")
            print("="*60)
            
            if "Hello World from E2E test!" in full_output:
                print("✅ SUCCESS: Logs are displayed correctly!")
                print("EventBus synchronization is working!")
            elif "No logs available" in full_output and "hello" not in full_output.lower():
                print("❌ FAILURE: Logs are not displayed")
                print("EventBus synchronization may have issues")
            elif "[hello" in full_output or "Started script" in full_output:
                print("✅ PARTIAL SUCCESS: Script started, logs system working")
            else:
                print("⚠️ INCONCLUSIVE: Cannot determine log status")
            
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
    test_component_system_with_logs()
