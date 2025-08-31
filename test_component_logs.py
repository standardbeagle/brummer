#!/usr/bin/env python3
"""Test if component system displays logs correctly after race condition fix"""

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

def test_component_logs():
    """Test if logs display correctly with component system after the fix"""
    
    print("="*60)
    print("TESTING COMPONENT SYSTEM WITH RACE CONDITION FIX")
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
        
        # Run a simple echo command that should succeed
        os.execvp('./brum', ['./brum', 'echo', 'Hello from fixed brummer'])
        sys.exit(1)
        
    else:  # Parent
        os.close(slave)
        
        print("\n1. Starting './brum echo \"Hello from fixed brummer\"' with component system...")
        time.sleep(1)  # Wait for startup
        output = capture_output(master, 2)
        
        # Check if component system started
        if "Component system enabled" in output:
            print("✅ Component system is enabled")
        else:
            print("⚠️ Component system message not seen")
            
        print("\n2. Pressing '2' to view logs...")
        os.write(master, b'2')
        time.sleep(1)
        output = capture_output(master, 2)
        
        # Analyze logs view
        if "No logs available" in output:
            print("❌ BUG STILL EXISTS: Logs view shows 'No logs available'")
        elif "Hello from fixed brummer" in output:
            print("✅ SUCCESS: Logs are displayed correctly!")
        elif "Started command" in output:
            print("✅ PARTIAL SUCCESS: Startup logs shown")
        elif len(output.strip()) < 50:
            print("❌ BUG STILL EXISTS: Logs view is empty")
        else:
            print("⚠️ Cannot determine log status")
            
        print("\n3. Pressing '1' to view processes...")
        os.write(master, b'1')
        time.sleep(1)
        output = capture_output(master, 2)
        
        if "echo" in output.lower() or "custom" in output.lower():
            print("✅ Process is shown in process list")
        elif "No processes running" in output:
            print("⚠️ Process already finished (expected for echo)")
        else:
            print("❌ Unexpected process view state")
            
        # Kill the process
        try:
            os.kill(pid, 15)  # SIGTERM
            time.sleep(0.5)
        except:
            pass
            
        os.close(master)
        os.waitpid(pid, os.WNOHANG)
        
        print("\n" + "="*60)
        print("TEST COMPLETE")
        print("="*60)

if __name__ == '__main__':
    test_component_logs()
