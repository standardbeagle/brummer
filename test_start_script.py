#!/usr/bin/env python3
"""Test starting a script with Enter key"""

import pty
import os
import sys
import time
import select
import tempfile
import json
import fcntl
import termios
import struct

def test_start_script():
    """Test starting script with Enter"""
    
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
        
        # Set environment for components
        env = os.environ.copy()
        env['BRUMMER_USE_COMPONENTS'] = 'true'
        env['TERM'] = 'xterm-256color'
        
        # Create PTY
        master, slave = pty.openpty()
        
        # Set terminal size
        winsize = struct.pack("HHHH", 24, 80, 0, 0)
        fcntl.ioctl(slave, termios.TIOCSWINSZ, winsize)
        
        # Get ttyname for debug
        ttyname = os.ttyname(slave)
        
        pid = os.fork()
        
        if pid == 0:  # Child
            try:
                os.close(master)
                os.setsid()
                
                # This is the key part - properly set up TTY
                fd = os.open(ttyname, os.O_RDWR)
                
                # Make this the controlling terminal
                fcntl.ioctl(fd, termios.TIOCSCTTY, 0)
                
                os.dup2(fd, 0)
                os.dup2(fd, 1) 
                os.dup2(fd, 2)
                
                if fd > 2:
                    os.close(fd)
                os.close(slave)
                
                # Run brummer
                os.chdir(tmpdir)
                os.execve('/home/beagle/work/brummer/brum', ['brum'], env)
                
                # Should never reach here
                sys.exit(1)
            except Exception as e:
                sys.stderr.write(f"[CHILD] Exception: {e}\n")
                sys.exit(1)
            
        else:  # Parent
            os.close(slave)
            
            output = []
            
            def read_output(duration=1.0):
                """Read output and respond to queries"""
                start = time.time()
                while time.time() - start < duration:
                    ready, _, _ = select.select([master], [], [], 0.1)
                    if ready:
                        try:
                            data = os.read(master, 4096)
                            if data:
                                decoded = data.decode('utf-8', errors='replace')
                                output.append(decoded)
                                
                                # Respond to terminal queries
                                if ']10;?' in decoded:  # Foreground color query
                                    os.write(master, b'\x1b]10;rgb:0000/0000/0000\x1b\\')
                                if ']11;?' in decoded:  # Background color query
                                    os.write(master, b'\x1b]11;rgb:ffff/ffff/ffff\x1b\\')
                                if '[6n' in decoded:  # Cursor position query
                                    os.write(master, b'\x1b[1;1R')
                                
                        except OSError:
                            break
                return ''.join(output)
            
            print("Starting TUI...")
            # Wait for initial render
            initial = read_output(1.0)
            
            # Check if script is shown
            if 'hello' in initial and '(ready to run)' in initial:
                print("✓ Script 'hello' is shown as ready to run")
            else:
                print("✗ Script not shown properly")
                print(initial)
            
            print("\nPressing Enter to start script...")
            # Send Enter key
            os.write(master, b'\r')
            
            # Wait for response
            response = read_output(2.0)
            
            # Check if script started or view switched to logs
            if 'Hello World!' in response or 'started' in response.lower() or '2.Log' in response:
                print("✓ Script appears to have started or switched to logs view")
                print("\nOutput after Enter:")
                print(response[-1000:])  # Last 1000 chars
            else:
                print("✗ Script may not have started")
                print("\nOutput after Enter:")
                print(response[-1000:])
            
            # Send 'q' to quit
            os.write(master, b'q')
            time.sleep(0.5)
            
            # Clean up
            try:
                os.kill(pid, 15)
            except:
                pass
            os.close(master)
            os.waitpid(pid, os.WNOHANG)

if __name__ == '__main__':
    test_start_script()