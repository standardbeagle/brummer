#!/usr/bin/env python3
"""Test URL and proxy display in component system"""

import pexpect
import time
import sys
import os

def test_url_detection():
    """Test URL detection and proxy display"""
    
    # Set component system flag
    os.environ['BRUMMER_USE_COMPONENTS'] = 'true'
    
    print("Starting brummer with component system...")
    
    # Use the test package.json
    child = pexpect.spawn('./brum', ['-d', '.'], encoding='utf-8', timeout=10)
    child.logfile = sys.stdout
    
    try:
        # Wait for TUI to start
        time.sleep(2)
        
        # Start the dev script (which logs URLs)
        print("\nStarting dev script...")
        child.send('dev\r')
        time.sleep(3)
        
        # Switch to URLs view (tab 4)
        print("\nSwitching to URLs view...")
        child.send('4')
        time.sleep(2)
        
        # Take a screenshot by waiting
        print("\nURLs view should now be visible...")
        time.sleep(3)
        
        # Switch to logs view to see output
        print("\nSwitching to logs view...")
        child.send('2')
        time.sleep(2)
        
        # Back to URLs view
        print("\nBack to URLs view...")
        child.send('4')
        time.sleep(3)
        
        # Quit
        print("\nQuitting...")
        child.send('q')
        child.send('y')
        
    except pexpect.TIMEOUT:
        print("\nTimeout occurred")
    except Exception as e:
        print(f"\nError: {e}")
    finally:
        try:
            child.terminate()
        except:
            pass

if __name__ == "__main__":
    # Change to directory with test package.json
    if os.path.exists('test-proxy-package.json'):
        os.system('cp test-proxy-package.json package.json')
    
    test_url_detection()