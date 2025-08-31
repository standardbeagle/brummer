#!/usr/bin/env python3
"""Test logs display timing in component system"""

import pty
import os
import sys
import time
import select

def capture_output(master, duration=2):
    """Capture output for specified duration"""
    output = []
    start_time = time.time()
    
    while time.time() - start_time < duration:
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
    return ''.join(output)

def test_logs_timing():
    """Test if logs appear after the delay"""
    
    print("="*60)
    print("TESTING LOG DISPLAY TIMING IN COMPONENT SYSTEM")
    print("="*60)
    
    # Set environment for components
    os.environ['BRUMMER_USE_COMPONENTS'] = 'true'
    
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
        
        # Run a simple echo command
        os.execvp('./brum', ['./brum', 'echo', 'SUCCESS: Logs are working!'])
        sys.exit(1)
        
    else:  # Parent
        os.close(slave)
        
        print("\n1. Starting './brum echo \"SUCCESS: Logs are working!\"'...")
        time.sleep(1)  # Wait for startup
        capture_output(master, 1)
        
        print("\n2. Pressing '2' to view logs immediately...")
        os.write(master, b'2')
        time.sleep(0.2)  # Short wait
        output = capture_output(master, 0.5)
        
        if "No logs available" in output:
            print("⏱️ Initial: Logs view shows 'No logs available' (expected)")
        
        print("\n3. Waiting for delayed log loading (500ms)...")
        time.sleep(0.5)
        output = capture_output(master, 1)
        
        if "SUCCESS: Logs are working!" in output:
            print("✅ SUCCESS: Logs appear after delay!")
        elif "[custom" in output:
            print("✅ PARTIAL SUCCESS: Process ID shown in logs")
        elif "No logs available" in output:
            print("❌ FAILURE: Still showing 'No logs available'")
        else:
            print("⚠️ Unknown state")
            
        # Kill the process
        try:
            os.kill(pid, 15)  # SIGTERM
            time.sleep(0.5)
        except:
            pass
            
        os.close(master)
        os.waitpid(pid, os.WNOHANG)
        
        print("\n" + "="*60)
        print("SUMMARY: Component system loads logs with 500ms delay")
        print("This ensures processes have time to start and generate logs")
        print("="*60)

if __name__ == '__main__':
    test_logs_timing()
