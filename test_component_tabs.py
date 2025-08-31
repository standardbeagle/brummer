#!/usr/bin/env python3
"""Test that all tabs are visible in component view."""

import os
import time
import sys
import pty
import select
import termios
import tty

def test_component_tabs():
    """Test that all 7 tabs are available in component view."""
    
    # Set component mode environment variable
    env = os.environ.copy()
    env['BRUMMER_USE_COMPONENTS'] = 'true'
    
    # Create a pseudo-terminal
    master, slave = pty.openpty()
    
    # Set terminal size
    import fcntl
    import struct
    size = struct.pack('HHHH', 30, 120, 0, 0)
    fcntl.ioctl(slave, termios.TIOCSWINSZ, size)
    
    # Start brummer with component mode
    pid = os.fork()
    if pid == 0:
        # Child process
        os.setsid()
        os.dup2(slave, 0)
        os.dup2(slave, 1)
        os.dup2(slave, 2)
        os.close(master)
        os.close(slave)
        os.execve('./brum', ['./brum', '--no-mcp'], env)
        sys.exit(1)  # Should not reach here
    
    # Parent process
    os.close(slave)
    
    # Set non-blocking
    import fcntl
    flags = fcntl.fcntl(master, fcntl.F_GETFL)
    fcntl.fcntl(master, fcntl.F_SETFL, flags | os.O_NONBLOCK)
    
    output = ""
    tabs_found = set()
    start_time = time.time()
    
    try:
        # Wait for initial output
        time.sleep(2)
        
        # Read initial screen
        while True:
            ready, _, _ = select.select([master], [], [], 0.1)
            if ready:
                try:
                    data = os.read(master, 4096)
                    output += data.decode('utf-8', errors='ignore')
                except OSError:
                    break
            else:
                break
        
        print("Initial screen:")
        print("-" * 40)
        # Only print last 20 lines for clarity
        lines = output.split('\n')
        for line in lines[-20:]:
            print(line[:100])  # Truncate long lines
        print("-" * 40)
        
        # Check for tabs in output
        tab_indicators = [
            ('1', 'Processes', '🏃'),
            ('2', 'Logs', '📄'),
            ('3', 'Errors', '❌'),
            ('4', 'URLs', '🔗'),
            ('5', 'Web', '🌐'),
            ('6', 'AI', '🤖'),
            ('7', 'Settings', '⚙️')
        ]
        
        for key, name, icon in tab_indicators:
            # Check if tab is visible
            if icon in output or name in output:
                tabs_found.add(name)
                print(f"✓ Found tab {key}: {name} {icon}")
        
        # Try pressing number keys to switch tabs
        for key in ['2', '3', '4', '5', '6', '7', '1']:
            print(f"\nSwitching to tab {key}...")
            os.write(master, key.encode())
            time.sleep(0.5)
            
            # Read response
            output_chunk = ""
            while True:
                ready, _, _ = select.select([master], [], [], 0.1)
                if ready:
                    try:
                        data = os.read(master, 4096)
                        output_chunk += data.decode('utf-8', errors='ignore')
                    except OSError:
                        break
                else:
                    break
            
            # Check what view we're in
            if '📄' in output_chunk and 'Logs' in output_chunk:
                print("  → Switched to Logs view")
                tabs_found.add('Logs')
            elif '❌' in output_chunk and 'Errors' in output_chunk:
                print("  → Switched to Errors view")
                tabs_found.add('Errors')
            elif '🔗' in output_chunk and 'URLs' in output_chunk:
                print("  → Switched to URLs view")
                tabs_found.add('URLs')
            elif '🌐' in output_chunk and 'Web' in output_chunk:
                print("  → Switched to Web view")
                tabs_found.add('Web')
            elif '🤖' in output_chunk and 'AI' in output_chunk:
                print("  → Switched to AI view")
                tabs_found.add('AI')
            elif '⚙️' in output_chunk and 'Settings' in output_chunk:
                print("  → Switched to Settings view")
                tabs_found.add('Settings')
            elif '🏃' in output_chunk and 'Processes' in output_chunk:
                print("  → Switched to Processes view")
                tabs_found.add('Processes')
        
        # Send quit command
        os.write(master, b'q')
        time.sleep(0.5)
        
    finally:
        # Cleanup
        try:
            os.kill(pid, 9)
            os.waitpid(pid, 0)
        except:
            pass
        os.close(master)
    
    # Report results
    print("\n" + "=" * 50)
    print("COMPONENT TABS TEST RESULTS")
    print("=" * 50)
    
    expected_tabs = {'Processes', 'Logs', 'Errors', 'URLs', 'Web', 'AI', 'Settings'}
    missing_tabs = expected_tabs - tabs_found
    
    if tabs_found:
        print(f"✅ Found {len(tabs_found)}/{len(expected_tabs)} tabs:")
        for tab in sorted(tabs_found):
            print(f"   • {tab}")
    
    if missing_tabs:
        print(f"\n❌ Missing {len(missing_tabs)} tabs:")
        for tab in sorted(missing_tabs):
            print(f"   • {tab}")
        return False
    else:
        print(f"\n🎉 All {len(expected_tabs)} tabs are working!")
        return True

if __name__ == '__main__':
    success = test_component_tabs()
    sys.exit(0 if success else 1)