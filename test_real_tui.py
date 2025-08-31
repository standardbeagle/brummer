#!/usr/bin/env python3
"""Test real TUI interaction"""

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

def test_real_tui():
    """Test with real interaction"""
    
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
        
        # Set terminal size
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
            
            output = []
            
            def read_and_respond(duration=1.0):
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
                                
                                # Print what we see
                                sys.stdout.write(decoded)
                                sys.stdout.flush()
                                
                                # Respond to terminal queries
                                if ']10;?' in decoded:  # Foreground color query
                                    # Respond with a color (black)
                                    os.write(master, b'\x1b]10;rgb:0000/0000/0000\x1b\\')
                                if '[6n' in decoded:  # Cursor position query
                                    # Respond with cursor at 1,1
                                    os.write(master, b'\x1b[1;1R')
                                
                        except OSError:
                            break
                return ''.join(output)
            
            print("Starting TUI with responses...")
            out1 = read_and_respond(1.0)
            
            print("\n\n=== After initial startup ===")
            out2 = read_and_respond(2.0)
            
            print("\n\n=== Total output length: %d bytes ===" % len(''.join(output)))
            
            # Send quit
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
    test_real_tui()