#!/usr/bin/env python3
"""Test PTY with debug output"""

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

def test_debug_pty():
    """Test PTY with debug enabled"""
    
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
        
        # Set environment for components and debug
        env = os.environ.copy()
        env['BRUMMER_USE_COMPONENTS'] = 'true'
        env['BRUMMER_DEBUG'] = 'true'
        
        # Create PTY
        master, slave = pty.openpty()
        
        # Set terminal size on the PTY
        winsize = struct.pack("HHHH", 24, 80, 0, 0)
        fcntl.ioctl(slave, termios.TIOCSWINSZ, winsize)
        
        pid = os.fork()
        
        if pid == 0:  # Child
            os.close(master)
            os.setsid()
            fd = os.open(os.ttyname(slave), os.O_RDWR)
            os.dup2(fd, 0)
            os.dup2(fd, 1) 
            os.dup2(fd, 2)
            if fd > 2:
                os.close(fd)
            os.close(slave)
            
            # Run brummer
            os.chdir(tmpdir)
            os.execve('/home/beagle/work/brummer/brum', ['brum'], env)
            sys.exit(1)
            
        else:  # Parent
            os.close(slave)
            
            def read_all_fds(duration=1.0):
                """Read from master"""
                start = time.time()
                stdout_data = []
                while time.time() - start < duration:
                    ready, _, _ = select.select([master], [], [], 0.1)
                    if ready:
                        try:
                            data = os.read(master, 4096)
                            if data:
                                decoded = data.decode('utf-8', errors='replace')
                                stdout_data.append(decoded)
                                # Print everything including control chars
                                sys.stdout.write(decoded)
                                sys.stdout.flush()
                        except OSError:
                            break
                return ''.join(stdout_data)
            
            print("=== Starting with debug ===")
            output1 = read_all_fds(1.0)
            
            print("\n=== After 1 second ===")  
            output2 = read_all_fds(2.0)
            
            print("\n=== Sending quit ===")
            os.write(master, b'q')
            output3 = read_all_fds(1.0)
            
            # Clean up
            try:
                os.kill(pid, 15)
            except:
                pass
            os.close(master)
            os.waitpid(pid, os.WNOHANG)

if __name__ == '__main__':
    test_debug_pty()