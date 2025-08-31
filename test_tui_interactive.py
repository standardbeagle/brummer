#!/usr/bin/env python3
"""
Interactive TUI test script with PTY support for testing component system
Allows sending keystrokes and capturing output
"""

import pty
import os
import sys
import time
import select
import termios
import tty
import threading

class TUITester:
    def __init__(self, use_components=True):
        self.master = None
        self.slave = None
        self.pid = None
        self.use_components = use_components
        self.output_buffer = []
        self.running = True
        
    def start(self):
        """Start brummer in a PTY"""
        # Set environment variable for component system
        if self.use_components:
            os.environ['BRUMMER_USE_COMPONENTS'] = 'true'
        else:
            os.environ.pop('BRUMMER_USE_COMPONENTS', None)
            
        # Create a pseudo-terminal
        self.master, self.slave = pty.openpty()
        
        # Fork a child process
        self.pid = os.fork()
        
        if self.pid == 0:  # Child process
            # Close master side
            os.close(self.master)
            
            # Make slave the controlling terminal
            os.setsid()
            os.dup2(self.slave, 0)  # stdin
            os.dup2(self.slave, 1)  # stdout
            os.dup2(self.slave, 2)  # stderr
            os.close(self.slave)
            
            # Run brummer
            os.execvp('./brum', ['./brum'])
            sys.exit(1)  # Should never reach here
        else:
            # Close slave side in parent
            os.close(self.slave)
            
            # Start output reader thread
            self.reader_thread = threading.Thread(target=self._read_output)
            self.reader_thread.daemon = True
            self.reader_thread.start()
            
    def _read_output(self):
        """Read output from the PTY"""
        while self.running:
            try:
                ready, _, _ = select.select([self.master], [], [], 0.1)
                if ready:
                    data = os.read(self.master, 4096)
                    if data:
                        decoded = data.decode('utf-8', errors='replace')
                        self.output_buffer.append(decoded)
                        sys.stdout.write(decoded)
                        sys.stdout.flush()
            except (OSError, ValueError):
                break
                
    def send_key(self, key):
        """Send a keystroke to the TUI"""
        try:
            os.write(self.master, key.encode())
            time.sleep(0.1)  # Small delay for UI to update
        except OSError:
            pass
            
    def send_text(self, text):
        """Send text to the TUI"""
        try:
            os.write(self.master, text.encode())
            time.sleep(0.1)
        except OSError:
            pass
            
    def wait(self, seconds):
        """Wait for specified seconds"""
        time.sleep(seconds)
        
    def get_output(self):
        """Get accumulated output"""
        return ''.join(self.output_buffer)
        
    def clear_output(self):
        """Clear output buffer"""
        self.output_buffer = []
        
    def stop(self):
        """Stop the TUI"""
        self.running = False
        try:
            # Send quit command
            self.send_key('q')
            time.sleep(0.5)
        except:
            pass
            
        # Clean up
        try:
            os.close(self.master)
        except:
            pass
            
        if self.pid:
            try:
                os.waitpid(self.pid, 0)
            except:
                pass

def test_component_system():
    """Test the component system with logs"""
    print("="*60)
    print("Testing COMPONENT SYSTEM")
    print("="*60)
    
    tester = TUITester(use_components=True)
    tester.start()
    
    print("\n1. Waiting for TUI to initialize...")
    tester.wait(2)
    
    print("\n2. Starting 'hello' script (Tab to Scripts, Enter to start)...")
    tester.send_key('\t')  # Tab to scripts
    tester.wait(0.5)
    tester.send_key('\r')  # Enter to start script
    tester.wait(2)
    
    print("\n3. Switching to Logs view (press 2)...")
    tester.send_key('2')
    tester.wait(2)
    
    print("\n4. Checking if logs are displayed...")
    output = tester.get_output()
    
    # Check for logs
    if "No logs available" in output:
        print("❌ FAILED: Logs are not displayed (shows 'No logs available')")
    elif "Hello World" in output:
        print("✅ SUCCESS: Logs are displayed correctly!")
    else:
        print("⚠️  WARNING: Cannot determine if logs are displayed")
        
    print("\n5. Stopping TUI...")
    tester.stop()
    tester.wait(1)
    
    return "Hello World" in output

def test_monolithic_system():
    """Test the monolithic system (without components)"""
    print("\n" + "="*60)
    print("Testing MONOLITHIC SYSTEM (for comparison)")
    print("="*60)
    
    tester = TUITester(use_components=False)
    tester.start()
    
    print("\n1. Waiting for TUI to initialize...")
    tester.wait(2)
    
    print("\n2. Starting 'hello' script...")
    tester.send_key('\t')  # Tab to scripts
    tester.wait(0.5)
    tester.send_key('\r')  # Enter to start script
    tester.wait(2)
    
    print("\n3. Switching to Logs view...")
    tester.send_key('2')
    tester.wait(2)
    
    print("\n4. Checking if logs are displayed...")
    output = tester.get_output()
    
    if "Hello World" in output:
        print("✅ Monolithic system shows logs correctly")
    else:
        print("❌ Monolithic system also has issues")
        
    print("\n5. Stopping TUI...")
    tester.stop()
    tester.wait(1)
    
    return "Hello World" in output

if __name__ == '__main__':
    # Test component system
    component_works = test_component_system()
    
    # Test monolithic system for comparison
    monolithic_works = test_monolithic_system()
    
    # Summary
    print("\n" + "="*60)
    print("TEST SUMMARY")
    print("="*60)
    print(f"Component System:  {'✅ WORKING' if component_works else '❌ NOT WORKING'}")
    print(f"Monolithic System: {'✅ WORKING' if monolithic_works else '❌ NOT WORKING'}")
    
    if not component_works and monolithic_works:
        print("\n⚠️  Component system needs fixing - logs not displaying")
    elif component_works:
        print("\n🎉 Component system is working correctly!")
    else:
        print("\n⚠️  Both systems have issues")