#!/usr/bin/env python3
"""Test PTY setup to ensure TTY is available"""

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

def test_pty_tty():
    """Test that PTY provides proper TTY"""
    
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
        
        # Create PTY
        master, slave = pty.openpty()
        
        # Set terminal size on the PTY
        winsize = struct.pack("HHHH", 24, 80, 0, 0)
        fcntl.ioctl(slave, termios.TIOCSWINSZ, winsize)
        
        print(f"PTY created: master={master}, slave={slave}")
        print(f"TTY name: {os.ttyname(slave)}")
        
        pid = os.fork()
        
        if pid == 0:  # Child
            os.close(master)
            
            # Make the slave the controlling terminal
            os.setsid()
            
            # Open slave as controlling terminal
            fd = os.open(os.ttyname(slave), os.O_RDWR)
            
            # Redirect stdio
            os.dup2(fd, 0)
            os.dup2(fd, 1) 
            os.dup2(fd, 2)
            
            if fd > 2:
                os.close(fd)
            os.close(slave)
            
            # Verify we have a TTY
            if os.isatty(0):
                print("Child: stdin is a TTY", file=sys.stderr)
            else:
                print("Child: stdin is NOT a TTY", file=sys.stderr)
            
            # Change to test directory and run brummer
            os.chdir(tmpdir)
            os.execve('/home/beagle/work/brummer/brum', ['brum'], env)
            sys.exit(1)
            
        else:  # Parent
            os.close(slave)
            
            def read_output(duration=1.0):
                """Read output for duration"""
                start = time.time()
                output = []
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
                        except OSError as e:
                            print(f"Read error: {e}", file=sys.stderr)
                            break
                return ''.join(output)
            
            print("\nStarting TUI with PTY...")
            output1 = read_output(1.0)
            
            print("\nWaiting for initialization...")
            output2 = read_output(2.0)
            
            print("\nSending quit command...")
            os.write(master, b'q')
            output3 = read_output(1.0)
            
            # Clean up
            try:
                os.kill(pid, 15)
            except:
                pass
            os.close(master)
            os.waitpid(pid, os.WNOHANG)
            
            # Check if we got actual TUI output
            full_output = output1 + output2 + output3
            if len(full_output) > 100:
                print(f"\nSUCCESS: Got {len(full_output)} bytes of output")
            else:
                print(f"\nFAILURE: Only got {len(full_output)} bytes")

if __name__ == '__main__':
    test_pty_tty()