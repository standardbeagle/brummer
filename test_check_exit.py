#!/usr/bin/env python3
"""Check if child exits"""

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
import signal

def test_check_exit():
    """Test and check if child exits"""
    
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
        
        ttyname = os.ttyname(slave)
        print(f"Created PTY: {ttyname}")
        
        pid = os.fork()
        
        if pid == 0:  # Child
            try:
                os.close(master)
                os.setsid()
                
                fd = os.open(ttyname, os.O_RDWR)
                fcntl.ioctl(fd, termios.TIOCSCTTY, 0)
                
                os.dup2(fd, 0)
                os.dup2(fd, 1) 
                os.dup2(fd, 2)
                
                if fd > 2:
                    os.close(fd)
                os.close(slave)
                
                os.chdir(tmpdir)
                
                # Run brummer directly
                os.execve('/home/beagle/work/brummer/brum', ['brum'], env)
                
                sys.exit(1)
            except Exception as e:
                sys.stderr.write(f"[CHILD] Exception: {e}\n")
                sys.exit(1)
            
        else:  # Parent
            os.close(slave)
            
            print(f"Started child process: {pid}")
            
            # Monitor for 5 seconds
            for i in range(50):
                # Check if child is still alive
                pid_result, status = os.waitpid(pid, os.WNOHANG)
                if pid_result != 0:
                    print(f"\nChild exited after {i*0.1:.1f} seconds")
                    if os.WIFSIGNALED(status):
                        print(f"Child was killed by signal: {os.WTERMSIG(status)}")
                    elif os.WIFEXITED(status):
                        print(f"Child exited with code: {os.WEXITSTATUS(status)}")
                    break
                
                # Read any output
                ready, _, _ = select.select([master], [], [], 0)
                if ready:
                    try:
                        data = os.read(master, 4096)
                        if data:
                            sys.stdout.write(data.decode('utf-8', errors='replace'))
                            sys.stdout.flush()
                    except OSError:
                        pass
                
                time.sleep(0.1)
            else:
                print(f"\nChild still running after 5 seconds")
                
                # Try to get stack trace
                try:
                    os.kill(pid, signal.SIGQUIT)  # SIGQUIT causes Go to dump stack
                    time.sleep(0.5)
                    
                    # Read any output
                    try:
                        data = os.read(master, 8192)
                        if data:
                            print("\nStack trace:")
                            print(data.decode('utf-8', errors='replace'))
                    except:
                        pass
                except:
                    pass
                
                # Kill it
                try:
                    os.kill(pid, signal.SIGTERM)
                except:
                    pass
            
            os.close(master)
            
            # Check strace log if it exists
            if os.path.exists('/tmp/brum_strace.log'):
                print("\nLast 50 lines of strace:")
                os.system('tail -50 /tmp/brum_strace.log')

if __name__ == '__main__':
    test_check_exit()