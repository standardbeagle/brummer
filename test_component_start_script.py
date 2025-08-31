#!/usr/bin/env python3
"""Test that the component system can start scripts properly"""

import pty
import os
import sys
import time
import select
import tempfile
import json

def test_component_start_script():
    """Test that scripts can be started in component system"""
    
    print("="*60)
    print("TEST: Component System Script Starting")
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
            
            print("\n1. Starting TUI...")
            read_output(2.0)
            
            print("\n2. Checking initial view...")
            # The process view should be visible by default
            
            print("\n3. Pressing Enter to start hello script...")
            # The first script should be selected by default
            os.write(master, b'\r')  # Press Enter to start selected script
            read_output(3.0)
            
            print("\n4. Switching to logs view (2)...")
            os.write(master, b'2')
            read_output(2.0)
            
            # Analyze output
            full_output = ''.join(output)
            
            print("\n" + "="*60)
            print("TEST RESULTS")
            print("="*60)
            
            # Look for various indicators
            if "Hello World from script test!" in full_output:
                print("✅ SUCCESS: Script output found in logs!")
                result = 0
            elif "hello" in full_output.lower() and "running" in full_output.lower():
                print("✅ SUCCESS: Script appears to be running!")
                result = 0
            elif "No logs available" in full_output:
                print("❌ FAILURE: Logs view shows 'No logs available'")
                print("Script may not have started properly")
                result = 1
            else:
                print("⚠️ INCONCLUSIVE: Cannot determine if script started")
                print(f"Output contains {len(full_output)} characters")
                # Save output for debugging
                with open('/tmp/component_test_output.txt', 'w') as f:
                    f.write(full_output)
                print("Full output saved to /tmp/component_test_output.txt")
                result = 1
            
            # Kill the process
            os.write(master, b'q')
            time.sleep(0.5)
            try:
                os.kill(pid, 15)
            except:
                pass
            os.close(master)
            os.waitpid(pid, os.WNOHANG)
            
            return result

if __name__ == '__main__':
    result = test_component_start_script()
    sys.exit(result)