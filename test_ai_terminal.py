#!/usr/bin/env python3
"""
Test script for AI terminal tab in component mode.
Tests that /ai terminal command properly connects PTY to the frontend.
"""
import pexpect
import sys
import time
import os

def test_ai_terminal():
    """Test the AI terminal tab with component mode enabled."""

    # Enable component mode
    env = os.environ.copy()
    env['BRUMMER_USE_COMPONENTS'] = 'true'

    # Start brummer with component mode
    print("Starting brummer with BRUMMER_USE_COMPONENTS=true...")
    child = pexpect.spawn('./brum', env=env, timeout=10, encoding='utf-8')
    child.logfile = sys.stdout

    try:
        # Wait for initial UI to load
        print("Waiting for UI to load...")
        time.sleep(2)

        # Press '/' to open command palette
        print("\nOpening command palette...")
        child.send('/')
        time.sleep(0.5)

        # Type '/ai terminal' command
        print("Typing '/ai terminal' command...")
        child.send('ai terminal')
        time.sleep(0.5)

        # Press Enter to execute
        print("Executing command...")
        child.send('\r')
        time.sleep(2)

        # The AI tab should now be active and show a terminal
        # Type a simple command to test PTY connection
        print("\nTesting PTY connection by typing 'echo HELLO'...")
        child.send('echo HELLO\r')
        time.sleep(1)

        # Check if we can see the echo output
        print("\nChecking for 'HELLO' in output...")
        index = child.expect(['HELLO', pexpect.TIMEOUT], timeout=3)

        if index == 0:
            print("✅ SUCCESS: PTY is connected! Echo output received.")

            # Test another command
            print("\nTesting with 'pwd' command...")
            child.send('pwd\r')
            time.sleep(1)

            # Try to see current directory in output
            index = child.expect(['/.*', pexpect.TIMEOUT], timeout=2)
            if index == 0:
                print("✅ SUCCESS: pwd command executed and displayed!")
            else:
                print("⚠️  WARNING: pwd output not detected (but PTY is working)")

            # Exit cleanly
            print("\nExiting...")
            child.send('\x03')  # Ctrl+C
            time.sleep(0.5)
            child.send('q')
            time.sleep(0.5)

            return True
        else:
            print("❌ FAILURE: PTY is NOT connected. Echo output not received.")
            print("\nThis means the terminal isn't connected to the frontend.")

            # Try to exit
            child.send('\x03')
            time.sleep(0.5)
            child.send('q')

            return False

    except pexpect.exceptions.EOF:
        print("❌ Process ended unexpectedly")
        return False
    except pexpect.exceptions.TIMEOUT:
        print("❌ Timeout waiting for response")
        return False
    finally:
        # Clean up
        try:
            child.terminate(force=True)
        except:
            pass

if __name__ == '__main__':
    print("=" * 70)
    print("AI Terminal Tab Connection Test")
    print("=" * 70)
    print()

    # Build first
    print("Building brummer...")
    ret = os.system('make build > /dev/null 2>&1')
    if ret != 0:
        print("❌ Build failed")
        sys.exit(1)

    print("Build successful!\n")

    # Run test
    success = test_ai_terminal()

    print()
    print("=" * 70)
    if success:
        print("✅ TEST PASSED: AI terminal tab is working correctly!")
        sys.exit(0)
    else:
        print("❌ TEST FAILED: AI terminal tab is NOT connected to frontend")
        sys.exit(1)