#!/usr/bin/env python3
"""
Simple test for /ai command - just check if the command is recognized
"""

import pty
import os
import sys
import time
import select
import tempfile
import json

def test_ai_simple():
    """Simple test of /ai command"""
    
    # Create temp dir with package.json
    with tempfile.TemporaryDirectory() as tmpdir:
        package_json = {
            "name": "test-ai-simple",
            "scripts": {
                "hello": "echo 'Hello!'"
            }
        }
        
        with open(os.path.join(tmpdir, "package.json"), "w") as f:
            json.dump(package_json, f, indent=2)
        
        # Set environment
        env = os.environ.copy()
        env['BRUMMER_USE_COMPONENTS'] = 'true'
        
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
            
            # Run brummer
            os.chdir(tmpdir)
            os.execve('/home/beagle/work/brummer/brum', ['brum'], env)
            sys.exit(1)
            
        else:  # Parent
            os.close(slave)
            
            def read_output(duration=1.0):
                start = time.time()
                output = []
                while time.time() - start < duration:
                    ready, _, _ = select.select([master], [], [], 0.1)
                    if ready:
                        try:
                            data = os.read(master, 4096)
                            if data:
                                decoded = data.decode('utf-8', errors='replace')
                                output.append(decoded)
                                print(decoded, end='')
                        except OSError:
                            break
                return ''.join(output)
            
            print("=== Starting Brummer ===")
            startup_output = read_output(2.0)
            
            print("\n=== Sending slash to open command palette ===")
            os.write(master, b'/')
            time.sleep(0.5)
            slash_output = read_output(1.0)
            
            print("\n=== Typing 'ai test-claude' ===")
            os.write(master, b'ai test-claude')
            time.sleep(0.5)
            command_output = read_output(1.0)
            
            print("\n=== Pressing Enter ===")
            os.write(master, b'\r')
            time.sleep(2.0)
            enter_output = read_output(2.0)
            
            print("\n=== Switching to AI Coder tab (6) ===")
            os.write(master, b'6')
            time.sleep(1.0)
            tab_output = read_output(1.0)

            print("\n=== Switching to Logs tab (2) to check for AI PTY logs ===")
            os.write(master, b'2')
            time.sleep(1.0)
            logs_output = read_output(2.0)

            print("\n=== Back to AI Coder tab (6) ===")
            os.write(master, b'6')
            time.sleep(1.0)
            final_tab_output = read_output(1.0)

            print("\n=== Sending Enter to terminal ===")
            os.write(master, b'\r')
            time.sleep(1.0)
            enter_output = read_output(1.0)

            print("\n=== Sending 'echo hello' to terminal ===")
            os.write(master, b'echo hello\r')
            time.sleep(1.0)
            echo_output = read_output(1.0)

            print("\n=== Final output check ===")
            final_output = read_output(1.0)
            
            # Combine all output
            all_output = startup_output + slash_output + command_output + enter_output + tab_output + logs_output + final_tab_output + enter_output + echo_output + final_output
            
            print(f"\n=== ANALYSIS ===")
            print(f"Total output length: {len(all_output)}")
            print(f"Contains 'AI': {'AI' in all_output}")
            print(f"Contains 'coder': {'coder' in all_output}")
            print(f"Contains 'terminal': {'terminal' in all_output}")
            print(f"Contains 'Unknown command': {'Unknown command' in all_output}")
            print(f"Contains 'Available commands': {'Available commands' in all_output}")
            print(f"Contains 'Starting AI': {'Starting AI' in all_output}")
            print(f"Contains 'AIPTYStartMsg': {'AIPTYStartMsg' in all_output}")
            print(f"Contains 'ai-pty': {'ai-pty' in all_output}")
            print(f"Contains 'PTY started': {'PTY started' in all_output}")
            print(f"Contains 'Error': {'Error' in all_output}")
            
            # Clean up
            os.write(master, b'q')
            time.sleep(0.5)
            try:
                os.kill(pid, 15)
            except:
                pass
            os.close(master)
            os.waitpid(pid, os.WNOHANG)

if __name__ == '__main__':
    test_ai_simple()
