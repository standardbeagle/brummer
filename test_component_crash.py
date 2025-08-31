#!/usr/bin/env python3
"""
Simple test to check if component system starts without crashing
"""

import pty
import os
import sys
import time
import select
import signal

def test_startup(use_components=True):
    """Test if brummer starts without crashing"""
    
    # Set environment variable for component system
    if use_components:
        os.environ['BRUMMER_USE_COMPONENTS'] = 'true'
        print("Testing WITH component system...")
    else:
        os.environ.pop('BRUMMER_USE_COMPONENTS', None)
        print("Testing WITHOUT component system...")
    
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
        
        # Run brummer
        os.execvp('./brum', ['./brum'])
        sys.exit(1)  # Should never reach here
        
    else:  # Parent process
        # Close slave side
        os.close(slave)
        
        # Collect output for 3 seconds
        output = []
        start_time = time.time()
        timeout = 3
        
        while time.time() - start_time < timeout:
            ready, _, _ = select.select([master], [], [], 0.1)
            if ready:
                try:
                    data = os.read(master, 4096)
                    if data:
                        decoded = data.decode('utf-8', errors='replace')
                        output.append(decoded)
                        # Check for panic
                        if "panic:" in decoded or "Caught panic:" in decoded:
                            print("❌ CRASHED with panic!")
                            print(decoded)
                            break
                except OSError:
                    break
        
        # Send quit signal
        try:
            os.write(master, b'q')
            time.sleep(0.2)
        except:
            pass
        
        # Kill child process
        try:
            os.kill(pid, signal.SIGTERM)
            time.sleep(0.2)
            os.kill(pid, signal.SIGKILL)
        except:
            pass
            
        # Clean up
        os.close(master)
        try:
            os.waitpid(pid, os.WNOHANG)
        except:
            pass
        
        # Check output
        full_output = ''.join(output)
        
        if "panic:" in full_output or "Caught panic:" in full_output:
            return False, "Crashed with panic"
        elif "Component system enabled" in full_output:
            return True, "Started with components"
        elif len(full_output) > 100:  # Some reasonable output
            return True, "Started successfully"
        else:
            return False, "No output or immediate exit"

# Test component system
print("="*50)
comp_success, comp_msg = test_startup(use_components=True)
print(f"Component system: {'✅ SUCCESS' if comp_success else '❌ FAILED'} - {comp_msg}")

print("\n" + "="*50)

# Test monolithic system  
mono_success, mono_msg = test_startup(use_components=False)
print(f"Monolithic system: {'✅ SUCCESS' if mono_success else '❌ FAILED'} - {mono_msg}")

print("\n" + "="*50)
print("SUMMARY:")
if comp_success and mono_success:
    print("✅ Both systems start successfully")
elif not comp_success and mono_success:
    print("❌ Component system has issues, monolithic works")
elif comp_success and not mono_success:
    print("⚠️  Monolithic system has issues, components work")
else:
    print("❌ Both systems have issues")