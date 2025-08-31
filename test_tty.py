#!/usr/bin/env python3
"""
Test script to run brummer with a proper TTY for testing component system
"""

import pty
import os
import sys
import time
import select

def run_with_tty():
    # Set environment variable for component system
    os.environ['BRUMMER_USE_COMPONENTS'] = 'true'
    
    # Create a pseudo-terminal
    master, slave = pty.openpty()
    
    # Fork a child process
    pid = os.fork()
    
    if pid == 0:  # Child process
        # Close master side
        os.close(master)
        
        # Make slave the controlling terminal
        os.setsid()
        os.dup2(slave, 0)  # stdin
        os.dup2(slave, 1)  # stdout
        os.dup2(slave, 2)  # stderr
        os.close(slave)
        
        # Run brummer with dev script
        os.execvp('./brum', ['./brum', 'dev'])
        sys.exit(1)  # Should never reach here
        
    else:  # Parent process
        # Close slave side
        os.close(slave)
        
        print("Starting brummer with component system enabled...")
        print("Press Ctrl+C to exit\n")
        
        # Read output for 10 seconds
        start_time = time.time()
        timeout = 10  # seconds
        
        try:
            while time.time() - start_time < timeout:
                # Check if data is available
                ready, _, _ = select.select([master], [], [], 0.1)
                if ready:
                    try:
                        data = os.read(master, 1024)
                        if data:
                            sys.stdout.write(data.decode('utf-8', errors='replace'))
                            sys.stdout.flush()
                    except OSError:
                        break
                        
        except KeyboardInterrupt:
            print("\nStopping...")
        
        # Send quit signal to child
        try:
            os.write(master, b'q')
            time.sleep(0.5)
        except:
            pass
        
        # Clean up
        os.close(master)
        os.waitpid(pid, 0)
        
        print("\n\nTest complete!")

if __name__ == '__main__':
    run_with_tty()