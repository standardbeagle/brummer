#!/usr/bin/env python3
"""Debug PTY test"""

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

def test_pty_debug():
    """Test with detailed debugging"""
    
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
        env['TERM'] = 'xterm-256color'  # Set a proper terminal type
        
        # Create PTY
        master, slave = pty.openpty()
        
        # Set terminal size
        winsize = struct.pack("HHHH", 24, 80, 0, 0)
        fcntl.ioctl(slave, termios.TIOCSWINSZ, winsize)
        
        # Get ttyname for debug
        ttyname = os.ttyname(slave)
        print(f"Created PTY: {ttyname}")
        
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
                
                # Verify we have a TTY
                if os.isatty(0):
                    sys.stderr.write("[CHILD] stdin is a TTY\n")
                else:
                    sys.stderr.write("[CHILD] stdin is NOT a TTY\n")
                
                # Run brummer
                os.chdir(tmpdir)
                sys.stderr.write(f"[CHILD] About to exec brummer in {tmpdir}\n")
                sys.stderr.flush()
                
                os.execve('/home/beagle/work/brummer/brum', ['brum'], env)
                
                # Should never reach here
                sys.stderr.write("[CHILD] execve failed!\n")
                sys.exit(1)
            except Exception as e:
                sys.stderr.write(f"[CHILD] Exception: {e}\n")
                sys.exit(1)
            
        else:  # Parent
            os.close(slave)
            
            print(f"Started child process: {pid}")
            
            output = []
            start = time.time()
            
            while time.time() - start < 3.0:
                # Check if child is still alive
                pid_result, status = os.waitpid(pid, os.WNOHANG)
                if pid_result != 0:
                    print(f"Child exited with status: {status}")
                    if os.WIFSIGNALED(status):
                        print(f"Child was killed by signal: {os.WTERMSIG(status)}")
                    elif os.WIFEXITED(status):
                        print(f"Child exited with code: {os.WEXITSTATUS(status)}")
                    break
                
                # Read any output
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
                        print(f"Read error: {e}")
                        break
            
            print(f"\nTotal output: {len(''.join(output))} bytes")
            
            # Clean up
            try:
                os.kill(pid, 15)
            except:
                pass
            os.close(master)
            
            # Check debug log
            if os.path.exists('/tmp/brummer_debug.log'):
                print("\nDEBUG LOG:")
                with open('/tmp/brummer_debug.log', 'r') as f:
                    print(f.read())

if __name__ == '__main__':
    test_pty_debug()