#!/usr/bin/env python3
"""Debug rendering to see exact output"""

import pty
import os
import sys
import time
import select
import tempfile
import json
import re

def test_render_debug():
    """Debug what's being rendered"""
    
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
                                # Don't print to stdout to avoid clutter
                        except OSError:
                            break
            
            print("Starting TUI...")
            read_output(0.5)
            
            print("Waiting for tick...")
            read_output(2.5)
            
            # Get full output
            full_output = ''.join(output)
            
            # Show raw output with all characters
            print("\nRaw output (last 1000 chars):")
            print("=" * 60)
            print(repr(full_output[-1000:]))
            
            print("\n" + "=" * 60)
            print("Looking for key patterns:")
            
            # Clean ANSI codes
            ansi_escape = re.compile(r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')
            clean_output = ansi_escape.sub('', full_output)
            
            # Split by lines for better analysis
            lines = clean_output.split('\n')
            print(f"Total lines: {len(lines)}")
            
            # Look for key patterns
            for i, line in enumerate(lines):
                if 'hello' in line.lower():
                    print(f"Line {i}: Found 'hello': {repr(line)}")
                if 'dev' in line.lower():
                    print(f"Line {i}: Found 'dev': {repr(line)}")
                if 'test' in line.lower() and 'test-project' not in line.lower():
                    print(f"Line {i}: Found 'test': {repr(line)}")
                if 'Stopped' in line:
                    print(f"Line {i}: Found 'Stopped': {repr(line)}")
                if 'Total:' in line:
                    print(f"Line {i}: Found stats: {repr(line)}")
                if 'ready to run' in line:
                    print(f"Line {i}: Found ready: {repr(line)}")
            
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
    test_render_debug()