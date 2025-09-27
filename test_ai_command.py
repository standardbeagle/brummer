#!/usr/bin/env python3
"""
Test script for /ai command functionality
Tests the current state of the AI terminal feature
"""

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

class AICommandTester:
    def __init__(self):
        self.master = None
        self.slave = None
        self.pid = None
        self.output_buffer = []
        
    def start_brummer(self, tmpdir):
        """Start brummer in a PTY"""
        # Set environment for components
        env = os.environ.copy()
        env['BRUMMER_USE_COMPONENTS'] = 'true'
        
        # Create PTY
        self.master, self.slave = pty.openpty()
        
        # Set terminal size
        winsize = struct.pack("HHHH", 24, 80, 0, 0)
        fcntl.ioctl(self.slave, termios.TIOCSWINSZ, winsize)
        
        self.pid = os.fork()
        
        if self.pid == 0:  # Child
            os.close(self.master)
            os.setsid()
            fd = os.open(os.ttyname(self.slave), os.O_RDWR)
            os.dup2(fd, 0)
            os.dup2(fd, 1) 
            os.dup2(fd, 2)
            if fd > 2:
                os.close(fd)
            os.close(self.slave)
            
            # Run brummer
            os.chdir(tmpdir)
            os.execve('/home/beagle/work/brummer/brum', ['brum'], env)
            sys.exit(1)
            
        else:  # Parent
            os.close(self.slave)
            
    def read_output(self, duration=1.0, verbose=False):
        """Read output for specified duration"""
        start = time.time()
        output = []
        while time.time() - start < duration:
            ready, _, _ = select.select([self.master], [], [], 0.1)
            if ready:
                try:
                    data = os.read(self.master, 4096)
                    if data:
                        decoded = data.decode('utf-8', errors='replace')
                        output.append(decoded)
                        self.output_buffer.append(decoded)
                        # Print for debugging if verbose
                        if verbose:
                            sys.stdout.write(decoded)
                            sys.stdout.flush()
                except OSError:
                    break
        return ''.join(output)
    
    def send_command(self, command):
        """Send a command to brummer"""
        try:
            # Send slash to open command palette
            os.write(self.master, b'/')
            time.sleep(0.2)
            
            # Send the command
            os.write(self.master, command.encode())
            time.sleep(0.2)
            
            # Press Enter
            os.write(self.master, b'\r')
            time.sleep(0.5)
            
        except OSError as e:
            print(f"Error sending command: {e}")
    
    def send_key(self, key):
        """Send a single key"""
        try:
            if isinstance(key, str):
                os.write(self.master, key.encode())
            else:
                os.write(self.master, key)
            time.sleep(0.1)
        except OSError:
            pass
    
    def cleanup(self):
        """Clean up resources"""
        try:
            # Send quit
            os.write(self.master, b'q')
            time.sleep(0.5)
        except:
            pass
            
        try:
            os.kill(self.pid, 15)
        except:
            pass
        try:
            os.close(self.master)
        except:
            pass
        try:
            os.waitpid(self.pid, os.WNOHANG)
        except:
            pass

def test_ai_command():
    """Test the /ai command functionality"""
    print("="*60)
    print("TESTING /AI COMMAND FUNCTIONALITY")
    print("="*60)
    
    # Create temp dir with package.json
    with tempfile.TemporaryDirectory() as tmpdir:
        package_json = {
            "name": "test-ai-project",
            "scripts": {
                "hello": "echo 'Hello from AI test!'"
            }
        }
        
        with open(os.path.join(tmpdir, "package.json"), "w") as f:
            json.dump(package_json, f, indent=2)
        
        tester = AICommandTester()
        
        try:
            print("\n1. Starting Brummer...")
            tester.start_brummer(tmpdir)
            
            print("\n2. Waiting for initialization...")
            initial_output = tester.read_output(2.0, verbose=True)

            print("\n3. Testing /ai terminal command...")
            tester.send_command("ai terminal")
            ai_output = tester.read_output(3.0, verbose=True)

            print("\n4. Switching to AI Coder tab (key 6)...")
            tester.send_key('6')
            time.sleep(0.5)
            tab_output = tester.read_output(1.0, verbose=True)

            print("\n4b. Checking current tab content...")
            current_tab_output = tester.read_output(1.0, verbose=True)

            print("\n5. Testing /ai test-claude command...")
            tester.send_command("ai test-claude")
            test_claude_output = tester.read_output(3.0, verbose=True)
            
            print("\n6. Analyzing results...")
            all_output = ''.join(tester.output_buffer)
            
            # Check for various indicators of success
            success_indicators = []
            error_indicators = []

            print(f"\n--- DEBUG: Full output length: {len(all_output)} chars ---")
            print(f"--- DEBUG: Output contains 'AI': {'AI' in all_output} ---")
            print(f"--- DEBUG: Output contains 'coder': {'coder' in all_output} ---")

            if "AI coder feature is not enabled" in all_output:
                error_indicators.append("AI coder feature not enabled")
            elif "AI coder feature is not initialized" in all_output:
                error_indicators.append("AI coder feature not initialized")

            if "Starting AI coder session creation" in all_output:
                success_indicators.append("AI session creation started")

            if "Started terminal AI coder session" in all_output:
                success_indicators.append("Terminal AI session started")

            if "Started test-claude AI coder session" in all_output:
                success_indicators.append("Test Claude AI session started")

            if "PTY started" in all_output:
                success_indicators.append("PTY session created")

            if "Error creating AI coder" in all_output:
                error_indicators.append("Error creating AI coder")

            if "Error creating PTY session" in all_output:
                error_indicators.append("Error creating PTY session")

            # Check for command palette indicators
            if "Unknown command" in all_output:
                error_indicators.append("Unknown command error")

            if "Available commands" in all_output:
                success_indicators.append("Command palette is working")
            
            # Print results
            print("\n" + "="*60)
            print("RESULTS")
            print("="*60)
            
            if success_indicators:
                print("✅ SUCCESS INDICATORS:")
                for indicator in success_indicators:
                    print(f"   - {indicator}")
            
            if error_indicators:
                print("❌ ERROR INDICATORS:")
                for indicator in error_indicators:
                    print(f"   - {indicator}")
            
            if not success_indicators and not error_indicators:
                print("⚠️  NO CLEAR INDICATORS - Check output above")
            
            # Overall assessment
            if success_indicators and not error_indicators:
                print("\n🎉 /AI COMMAND APPEARS TO BE WORKING!")
                return True
            elif error_indicators:
                print("\n❌ /AI COMMAND HAS ISSUES")
                return False
            else:
                print("\n⚠️  /AI COMMAND STATUS UNCLEAR")
                return False
                
        finally:
            print("\n7. Cleaning up...")
            tester.cleanup()

if __name__ == '__main__':
    success = test_ai_command()
    sys.exit(0 if success else 1)
