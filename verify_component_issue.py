#!/usr/bin/env python3
"""
Reproduce the exact issue the user reported:
1. Start with BRUMMER_USE_COMPONENTS=true ./brum server
2. Press '2' for logs - should show logs but shows empty
3. Press '1' for processes - check what's running
"""

import pty
import os
import sys
import time
import select
import signal

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
    """Test if component system displays logs when running 'server' command"""
    
    print("="*60)
    print("REPRODUCING USER'S ISSUE")
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
        
        # Run exactly what user ran
        os.execvp('./brum', ['./brum', 'server'])
        sys.exit(1)
        
    else:  # Parent
        os.close(slave)
        
        print("\n1. Starting './brum server' with component system...")
        output = capture_output(master, 3)
        
        # Check if component system started
        if "Component system enabled" in output:
            print("✅ Component system is enabled")
        else:
            print("⚠️ Component system message not seen")
            
        # Check what command is actually running
        if "benchling" in output.lower() or "Usage:" in output:
            print("❌ Wrong command executed! Running benchling-mcp server instead of detecting project type")
        elif "rails server" in output:
            print("✅ Correctly detected Rails project")
        elif "Failed to start" in output:
            print("⚠️ Command failed to start (expected if no Rails)")
        else:
            print("⚠️ Unknown behavior")
            
        print("\n2. Pressing '2' to view logs...")
        os.write(master, b'2')
        time.sleep(2)
        output = capture_output(master, 2)
        
        # Analyze logs view
        if "No logs available" in output:
            print("❌ CONFIRMED BUG: Logs view shows 'No logs available'")
        elif "server" in output.lower() or "Started" in output:
            print("✅ Logs are displayed")
        elif len(output.strip()) < 50:  # Very little output
            print("❌ CONFIRMED BUG: Logs view is empty or nearly empty")
        else:
            print("⚠️ Cannot determine log status")
            
        print("\n3. Pressing '1' to view processes...")
        os.write(master, b'1')
        time.sleep(1)
        output = capture_output(master, 2)
        
        if "server" in output.lower():
            print("✅ Process 'server' is shown in process list")
            # Check what type of server
            if "rails" in output.lower():
                print("  - Running Rails server")
            elif "benchling" in output.lower():
                print("  - Running benchling-mcp server (WRONG!)")
            else:
                print("  - Running unknown server command")
        else:
            print("⚠️ No server process found")
            
        print("\n4. Pressing 'r' to refresh...")
        os.write(master, b'r')
        time.sleep(1)
        output = capture_output(master, 2)
        
        print("\n5. Going back to logs view...")
        os.write(master, b'2')
        time.sleep(1)
        output = capture_output(master, 2)
        
        if "No logs available" in output or len(output.strip()) < 50:
            print("❌ STILL NO LOGS after refresh")
        else:
            print("⚠️ Checking output...")
            
        # Kill the process
        try:
            os.kill(pid, signal.SIGTERM)
            time.sleep(0.5)
            os.kill(pid, signal.SIGKILL)
        except:
            pass
            
        os.close(master)
        os.waitpid(pid, os.WNOHANG)
        
        print("\n" + "="*60)
        print("TEST COMPLETE - Component system has log display issues")
        print("="*60)

# Now test monolithic for comparison
def test_monolithic_logs():
    """Test monolithic system for comparison"""
    
    print("\n" + "="*60)
    print("TESTING MONOLITHIC SYSTEM FOR COMPARISON")
    print("="*60)
    
    # Remove component env var
    os.environ.pop('BRUMMER_USE_COMPONENTS', None)
    
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
        
        os.execvp('./brum', ['./brum', 'server'])
        sys.exit(1)
        
    else:  # Parent
        os.close(slave)
        
        print("\n1. Starting './brum server' with monolithic system...")
        output = capture_output(master, 3)
        
        print("\n2. Pressing '2' to view logs...")
        os.write(master, b'2')
        time.sleep(2)
        output = capture_output(master, 2)
        
        if "server" in output.lower() or "Started" in output:
            print("✅ Monolithic system shows logs correctly")
        else:
            print("⚠️ Monolithic system also has issues")
            
        # Kill the process
        try:
            os.kill(pid, signal.SIGTERM)
            time.sleep(0.5)
            os.kill(pid, signal.SIGKILL)
        except:
            pass
            
        os.close(master)
        os.waitpid(pid, os.WNOHANG)

if __name__ == '__main__':
    test_component_logs()
    test_monolithic_logs()