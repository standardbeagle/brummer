#!/usr/bin/env python3
"""Test script to verify proxy information display in URLs view"""

import subprocess
import time
import os
import sys
import signal
import requests
from http.server import HTTPServer, SimpleHTTPRequestHandler
import threading

def run_simple_server(port):
    """Run a simple HTTP server on the specified port"""
    class Handler(SimpleHTTPRequestHandler):
        def log_message(self, format, *args):
            # Log to stdout so it appears in brummer logs
            print(f"[SERVER {port}] {format % args}", flush=True)
    
    server = HTTPServer(('localhost', port), Handler)
    server_thread = threading.Thread(target=server.serve_forever)
    server_thread.daemon = True
    server_thread.start()
    return server

def test_proxy_display():
    """Test that proxy information is displayed in URLs view"""
    
    # Set environment variable to use components
    os.environ['BRUMMER_USE_COMPONENTS'] = 'true'
    
    print("Starting test HTTP server on port 8080...")
    server = run_simple_server(8080)
    time.sleep(1)
    
    # Create a test package.json with a script that hits the server
    package_json = {
        "name": "test-proxy",
        "scripts": {
            "test-server": "while true; do echo 'Server running at http://localhost:8080'; curl -s http://localhost:8080 > /dev/null; sleep 2; done"
        }
    }
    
    import json
    with open('/tmp/test-proxy-package.json', 'w') as f:
        json.dump(package_json, f)
    
    print("\nStarting brummer with component system...")
    # Start brummer in the test directory
    os.chdir('/tmp')
    os.system('cp /tmp/test-proxy-package.json package.json')
    
    # Start brummer with proxy in reverse mode
    print("Starting brummer with proxy server...")
    cmd = [
        '/home/beagle/work/brummer/brum',
        '--proxy-mode', 'reverse',
        'test-server'
    ]
    
    print(f"Running: {' '.join(cmd)}")
    print("\nTest Instructions:")
    print("1. The test server script should start automatically")
    print("2. Press '4' to switch to URLs view")
    print("3. Look for http://localhost:8080 with proxy information")
    print("4. The proxy info should show the proxy URL and port")
    print("5. Press 'q' to quit when done\n")
    
    # Run brummer
    subprocess.run(cmd)
    
    print("\nTest complete!")

if __name__ == "__main__":
    try:
        test_proxy_display()
    except KeyboardInterrupt:
        print("\nTest interrupted")
        sys.exit(0)