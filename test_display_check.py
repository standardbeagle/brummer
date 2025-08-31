#!/usr/bin/env python3
"""Check if scripts are being displayed as process items"""

import pty
import os
import sys
import time
import select
import tempfile
import json
import re

def test_display_check():
    """Check what's actually displayed in the process view"""
    
    print("="*60)
    print("TEST: Process View Display Check")
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
            read_output(2.0)
            
            # Get full output
            full_output = ''.join(output)
            
            # Clean ANSI codes for easier analysis
            ansi_escape = re.compile(r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')
            clean_output = ansi_escape.sub('', full_output)
            
            print("\n" + "="*60)
            print("ANALYSIS")
            print("="*60)
            
            # Check for key elements
            if "⏹️  Stopped Processes" in clean_output or "Stopped Processes" in clean_output:
                print("✅ Found 'Stopped Processes' header")
            
            if "Total: 3" in clean_output:
                print("✅ Found 'Total: 3' - scripts are being counted")
            
            # Look for script names
            for script in ["hello", "dev", "test"]:
                if script in clean_output:
                    print(f"✅ Found script '{script}' in output")
                    # Find the line containing the script
                    for line in clean_output.split('\n'):
                        if script in line:
                            print(f"   Line: {repr(line[:80])}")
                            break
                else:
                    print(f"❌ Script '{script}' not found")
            
            # Look for selection indicator
            if "▶" in clean_output:
                print("✅ Selection indicator found")
                # Find what's selected
                for line in clean_output.split('\n'):
                    if "▶" in line:
                        print(f"   Selected line: {repr(line[:80])}")
                        break
            else:
                print("❌ No selection indicator")
            
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
    test_display_check()